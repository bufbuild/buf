// Copyright 2020-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package buflsp implements a language server for Protobuf.
//
// The main entry-point of this package is the Serve() function, which creates a new LSP server.
package buflsp

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/bufbuild/buf/private/buf/bufformat"
	"github.com/bufbuild/protocompile/ast"
	"go.lsp.dev/protocol"
)

const (
	serverName = "buf-lsp"
)

const (
	semanticTypeType = iota
	semanticTypeStruct
	semanticTypeVariable
	semanticTypeEnum
	semanticTypeEnumMember
	semanticTypeInterface
	semanticTypeMethod
	semanticTypeDecorator
)

var (
	// These slices must match the order of the indices in the above const block.
	semanticTypeLegend = []string{
		"type", "struct", "variable", "enum",
		"enumMember", "interface", "method", "decorator",
	}
	semanticModifierLegend = []string{}
)

// server is an implementation of protocol.Server.
//
// This is a separate type from buflsp.lsp so that the dozens of handler methods for this
// type are kept separate from the rest of the logic.
//
// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification.
type server struct {
	// This automatically implements all of protocol.Server for us. By default,
	// every method returns an error.
	nyi

	// We embed the LSP pointer as well, since it only has private members.
	*lsp
}

// newServer creates a protocol.Server implementation out of an lsp.
func newServer(lsp *lsp) protocol.Server {
	return &server{lsp: lsp}
}

// Methods for server are grouped according to the groups in the LSP protocol specification.

// -- Lifecycle Methods

// Initialize is the first message the LSP receives from the client. This is where all
// initialization of the server wrt to the project is is invoked on must occur.
func (s *server) Initialize(
	ctx context.Context,
	params *protocol.InitializeParams,
) (*protocol.InitializeResult, error) {
	if err := s.init(ctx, params); err != nil {
		return nil, err
	}

	info := &protocol.ServerInfo{Name: serverName}
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		info.Version = buildInfo.Main.Version
	}

	// The LSP protocol library doesn't actually provide SemanticTokensOptions
	// correctly.
	type SemanticTokensLegend struct {
		TokenTypes     []string `json:"tokenTypes"`
		TokenModifiers []string `json:"tokenModifiers"`
	}
	type SemanticTokensOptions struct {
		protocol.WorkDoneProgressOptions

		Legend SemanticTokensLegend `json:"legend"`
		Full   bool                 `json:"full"`
	}

	return &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			// These are all the things we advertise to the client we can do.
			// For now, incomplete features are explicitly disabled here as TODOs.
			TextDocumentSync: &protocol.TextDocumentSyncOptions{
				OpenClose: true,
				// Request that whole files be sent to us. Protobuf IDL files don't
				// usually get especially huge, so this simplifies our logic without
				// necessarily making the LSP slow.
				Change: protocol.TextDocumentSyncKindFull,
				Save: &protocol.SaveOptions{
					IncludeText: false,
				},
			},
			DefinitionProvider: &protocol.DefinitionOptions{
				WorkDoneProgressOptions: protocol.WorkDoneProgressOptions{WorkDoneProgress: true},
			},
			DocumentFormattingProvider: true,
			HoverProvider:              true,
			SemanticTokensProvider: &SemanticTokensOptions{
				WorkDoneProgressOptions: protocol.WorkDoneProgressOptions{WorkDoneProgress: true},
				Legend: SemanticTokensLegend{
					TokenTypes:     semanticTypeLegend,
					TokenModifiers: semanticModifierLegend,
				},
				Full: true,
			},
		},
		ServerInfo: info,
	}, nil
}

// Initialized is sent by the client after it receives the Initialize response and has
// initialized itself. This is only a notification.
func (s *server) Initialized(
	ctx context.Context,
	params *protocol.InitializedParams,
) error {
	workspaceCapabilities := s.initParams.Load().Capabilities.Workspace
	if workspaceCapabilities == nil {
		return nil
	}
	didChangeConfiguration := workspaceCapabilities.DidChangeConfiguration
	if didChangeConfiguration != nil && didChangeConfiguration.DynamicRegistration {
		// The error is logged for us by the client wrapper.
		_ = s.client.RegisterCapability(ctx, &protocol.RegistrationParams{
			Registrations: []protocol.Registration{
				{Method: protocol.MethodWorkspaceDidChangeConfiguration},
			},
		})
	}

	return nil
}

func (s *server) SetTrace(
	ctx context.Context,
	params *protocol.SetTraceParams,
) error {
	s.lsp.traceValue.Store(&params.Value)
	return nil
}

// Shutdown is sent by the client when it wants the server to shut down and exit.
// The client will wait until Shutdown returns, and then call Exit.
func (s *server) Shutdown(ctx context.Context) error {
	return nil
}

// Exit is a notification that the client has seen shutdown complete, and that the
// server should now exit.
func (s *server) Exit(ctx context.Context) error {
	// TODO: return an error if Shutdown() has not been called yet.

	// Close the connection. This will let the server shut down gracefully once this
	// notification is replied to.
	return s.lsp.conn.Close()
}

// DidChangeConfiguration is sent whenever the client changes its config settings.
func (s *server) DidChangeConfiguration(
	ctx context.Context,
	params *protocol.DidChangeConfigurationParams,
) error {
	// We need to refresh every open file's settings, and refresh the file
	// itself.
	s.fileManager.uriToFile.Range(func(_ protocol.URI, file *file) bool {
		if file.IsOpenInEditor() {
			file.RefreshSettings(ctx)
			file.Refresh(ctx)
		}
		return true
	})

	return nil
}

// -- File synchronization methods.

// DidOpen is called whenever the client opens a document. This is our signal to parse
// the file.
func (s *server) DidOpen(
	ctx context.Context,
	params *protocol.DidOpenTextDocumentParams,
) error {
	file := s.fileManager.Open(ctx, params.TextDocument.URI)
	file.RefreshSettings(ctx)
	file.Update(ctx, params.TextDocument.Version, params.TextDocument.Text)
	file.Refresh(ctx)
	return nil
}

// DidChange is called whenever the client opens a document. This is our signal to parse
// the file.
func (s *server) DidChange(
	ctx context.Context,
	params *protocol.DidChangeTextDocumentParams,
) error {
	file := s.fileManager.Get(params.TextDocument.URI)
	if file == nil {
		// Update for a file we don't know about? Seems bad!
		return fmt.Errorf("received update for file that was not open: %q", params.TextDocument.URI)
	}

	file.Update(ctx, params.TextDocument.Version, params.ContentChanges[0].Text)
	file.Refresh(ctx)
	return nil
}

// DidSave is called whenever the client saves a document.
func (s *server) DidSave(
	ctx context.Context,
	params *protocol.DidSaveTextDocumentParams,
) error {
	// We use this as an opportunity to do a refresh; some lints, such as
	// breaking-against-last-saved, rely on this.
	file := s.fileManager.Get(params.TextDocument.URI)
	if file == nil {
		// Update for a file we don't know about? Seems bad!
		return fmt.Errorf("received update for file that was not open: %q", params.TextDocument.URI)
	}
	file.Refresh(ctx)
	return nil
}

// Formatting is called whenever the user explicitly requests formatting.
func (s *server) Formatting(
	ctx context.Context,
	params *protocol.DocumentFormattingParams,
) ([]protocol.TextEdit, error) {
	file := s.fileManager.Get(params.TextDocument.URI)
	if file == nil {
		// Format for a file we don't know about? Seems bad!
		return nil, fmt.Errorf("received update for file that was not open: %q", params.TextDocument.URI)
	}

	// We check the diagnostics on the file, if there are any build errors, we do not want
	// to format an invalid AST, so we skip formatting and return an error for logging.
	errorCount := 0
	for _, diagnostic := range file.diagnostics {
		if diagnostic.Severity == protocol.DiagnosticSeverityError {
			errorCount += 1
		}
	}
	if errorCount > 0 {
		return nil, fmt.Errorf("cannot format file %q, %v error(s) found", file.uri.Filename(), errorCount)
	}

	// Currently we have no way to honor any of the parameters.
	_ = params
	if file.fileNode == nil {
		return nil, nil
	}

	var out strings.Builder
	if err := bufformat.FormatFileNode(&out, file.fileNode); err != nil {
		return nil, err
	}

	newText := out.String()
	// Avoid formatting the file if text has not changed.
	if newText == file.text {
		return nil, nil
	}

	// XXX: The current compiler does not expose a span for the full file. Instead of
	// potentially undershooting the correct span (which can cause comments at the
	// start and end of the file to be duplicated), we instead manually count up the
	// number of lines in the file. This is comparatively cheap, compared to sending the
	// entire file over a domain socket.
	var lastLine, lastLineStart int
	for i := range len(file.text) {
		// NOTE: we are iterating over bytes, not runes.
		if file.text[i] == '\n' {
			lastLine++
			lastLineStart = i + 1 // Skip the \n.
		}
	}
	lastChar := len(file.text[lastLineStart:]) - 1 // Bytes, not runes!
	return []protocol.TextEdit{
		{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      0,
					Character: 0,
				},
				End: protocol.Position{
					Line:      uint32(lastLine),
					Character: uint32(lastChar),
				},
			},
			NewText: newText,
		},
	}, nil
}

// DidClose is called whenever the client closes a document.
func (s *server) DidClose(
	ctx context.Context,
	params *protocol.DidCloseTextDocumentParams,
) error {
	s.fileManager.Close(ctx, params.TextDocument.URI)
	return nil
}

// -- Language functionality methods.

// Hover is the entry point for hover inlays.
func (s *server) Hover(
	ctx context.Context,
	params *protocol.HoverParams,
) (*protocol.Hover, error) {
	file := s.fileManager.Get(params.TextDocument.URI)
	if file == nil {
		return nil, nil
	}

	symbol := file.SymbolAt(ctx, params.Position)
	if symbol == nil {
		return nil, nil
	}

	docs := symbol.FormatDocs(ctx)
	if docs == "" {
		return nil, nil
	}

	// Escape < and > occurrences in the docs.
	replacer := strings.NewReplacer("<", "&lt;", ">", "&gt;")
	docs = replacer.Replace(docs)

	range_ := symbol.Range() // Need to spill this here because Hover.Range is a pointer.
	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: docs,
		},
		Range: &range_,
	}, nil
}

// Definition is the entry point for go-to-definition.
func (s *server) Definition(
	ctx context.Context,
	params *protocol.DefinitionParams,
) ([]protocol.Location, error) {
	file := s.fileManager.Get(params.TextDocument.URI)
	if file == nil {
		return nil, nil
	}

	progress := newProgressFromClient(s.lsp, &params.WorkDoneProgressParams)
	progress.Begin(ctx, "Searching")
	defer progress.Done(ctx)

	symbol := file.SymbolAt(ctx, params.Position)
	if symbol == nil {
		return nil, nil
	}

	if imp, ok := symbol.kind.(*import_); ok {
		// This is an import, we just want to jump to the file.
		return []protocol.Location{{URI: imp.file.uri}}, nil
	}

	def, _ := symbol.Definition(ctx)
	if def != nil {
		return []protocol.Location{{
			URI:   def.file.uri,
			Range: def.Range(),
		}}, nil
	}

	return nil, nil
}

// SemanticTokensFull is called to render semantic token information on the client.
func (s *server) SemanticTokensFull(
	ctx context.Context,
	params *protocol.SemanticTokensParams,
) (*protocol.SemanticTokens, error) {
	file := s.fileManager.Get(params.TextDocument.URI)
	if file == nil {
		return nil, nil
	}

	progress := newProgressFromClient(s.lsp, &params.WorkDoneProgressParams)
	progress.Begin(ctx, "Processing Tokens")
	defer progress.Done(ctx)

	// In the case where there are no symbols for the file, we return nil for SemanticTokensFull.
	// This is based on the specification for the method textDocument/semanticTokens/full,
	// the expected response is the union type `SemanticTokens | null`.
	// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_semanticTokens
	if len(file.symbols) == 0 {
		return nil, nil
	}

	var (
		encoded           []uint32
		prevLine, prevCol uint32
	)
	for i, symbol := range file.symbols {
		progress.Report(ctx, fmt.Sprintf("%d/%d", i+1, len(file.symbols)), float64(i)/float64(len(file.symbols)))

		var semanticType uint32

		if symbol.isOption {
			semanticType = semanticTypeDecorator
		} else if def, defNode := symbol.Definition(ctx); def != nil {
			switch defNode.(type) {
			case *ast.FileNode:
				continue
			case *ast.MessageNode, *ast.GroupNode:
				semanticType = semanticTypeStruct
			case *ast.FieldNode, *ast.MapFieldNode, *ast.OneofNode:
				semanticType = semanticTypeVariable
			case *ast.EnumNode:
				semanticType = semanticTypeEnum
			case *ast.EnumValueNode:
				semanticType = semanticTypeEnumMember
			case *ast.ServiceNode:
				semanticType = semanticTypeInterface
			case *ast.RPCNode:
				semanticType = semanticTypeMethod
			}
		} else if _, ok := symbol.kind.(*builtin); ok {
			semanticType = semanticTypeType
		} else {
			continue
		}

		// This fairly painful encoding is described in detail here:
		// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_semanticTokens
		start, end := symbol.info.Start(), symbol.info.End()
		for i := start.Line; i <= end.Line; i++ {
			newLine := uint32(i - 1)
			var newCol uint32
			if i == start.Line {
				newCol = uint32(start.Col - 1)
				if prevLine == newLine {
					newCol -= prevCol
				}
			}

			symbolLen := uint32(end.Col - 1)
			if i == start.Line {
				symbolLen -= uint32(start.Col - 1)
			}

			encoded = append(encoded, newLine-prevLine, newCol, symbolLen, semanticType, 0)
			prevLine = newLine
			if i == start.Line {
				prevCol = uint32(start.Col - 1)
			} else {
				prevCol = 0
			}
		}
	}

	return &protocol.SemanticTokens{Data: encoded}, nil
}

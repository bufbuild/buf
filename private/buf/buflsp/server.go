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

package buflsp

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"unicode/utf16"

	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/buf/bufformat"
	"github.com/bufbuild/protocompile/experimental/ir"
	"github.com/bufbuild/protocompile/parser"
	"github.com/bufbuild/protocompile/reporter"
	"go.lsp.dev/protocol"
)

const (
	serverName = "buf-lsp"

	maxSymbolResults = 1000
)

// The subset of SemanticTokenTypes that we support.
// Must match the order of [semanticTypeLegend].
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.18/specification/#textDocument_semanticTokens
const (
	semanticTypeType = iota
	semanticTypeStruct
	semanticTypeVariable
	semanticTypeEnum
	semanticTypeEnumMember
	semanticTypeInterface
	semanticTypeMethod
	semanticTypeDecorator
	semanticTypeNamespace
)

// The subset of SemanticTokenModifiers that we support.
// Must match the order of [semanticModifierLegend].
// Semantic modifiers are encoded as a bitset, hence the shifted iota.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.18/specification/#textDocument_semanticTokens
const (
	semanticModifierDeprecated = 1 << iota
	semanticModifierDefaultLibrary
)

var (
	// These slices must match the order of the indices in the above const blocks.
	semanticTypeLegend = []string{
		"type",
		"struct",
		"variable",
		"enum",
		"enumMember",
		"interface",
		"method",
		"decorator",
		"namespace",
	}
	semanticModifierLegend = []string{
		"deprecated",
		"defaultLibrary", // maps to builtin values
	}
)

// server is an implementation of [protocol.Server].
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
			CompletionProvider: &protocol.CompletionOptions{
				ResolveProvider:   true,
				TriggerCharacters: []string{".", "\"", "/"},
			},
			DefinitionProvider:         &protocol.DefinitionOptions{},
			TypeDefinitionProvider:     &protocol.TypeDefinitionOptions{},
			DocumentFormattingProvider: true,
			HoverProvider:              true,
			ReferencesProvider:         &protocol.ReferenceOptions{},
			SemanticTokensProvider: &SemanticTokensOptions{
				Legend: SemanticTokensLegend{
					TokenTypes:     semanticTypeLegend,
					TokenModifiers: semanticModifierLegend,
				},
				Full: true,
			},
			WorkspaceSymbolProvider: true,
			DocumentSymbolProvider:  true,
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
	// No initialization required.
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
	s.lsp.shutdown = true
	return nil
}

// Exit is a notification that the client has seen shutdown complete, and that the
// server should now exit.
func (s *server) Exit(ctx context.Context) error {
	if !s.lsp.shutdown {
		return errors.New("shutdown was not called or not yet completed")
	}

	// Close the connection. This will let the server shut down gracefully once this
	// notification is replied to.
	return s.lsp.conn.Close()
}

// -- File synchronization methods.

// DidOpen is called whenever the client opens a document. This is our signal to parse
// the file.
func (s *server) DidOpen(
	ctx context.Context,
	params *protocol.DidOpenTextDocumentParams,
) error {
	file := s.fileManager.Track(params.TextDocument.URI)
	file.RefreshWorkspace(ctx)
	file.Update(ctx, params.TextDocument.Version, params.TextDocument.Text)
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
	file.RefreshWorkspace(ctx)
	return nil
}

// Formatting is called whenever the user explicitly requests formatting.
//
// NOTE: this still uses the current compiler since formatting is not yet implemented with
// the new compiler. This will be ported over once that is ready. For now, we parse the file
// on-demand for formatting.
func (s *server) Formatting(
	ctx context.Context,
	params *protocol.DocumentFormattingParams,
) ([]protocol.TextEdit, error) {
	file := s.fileManager.Get(params.TextDocument.URI)
	if file == nil {
		// Format for a file we don't know about? Seems bad!
		return nil, fmt.Errorf("received update for file that was not open: %q", params.TextDocument.URI)
	}
	var errorsWithPos []reporter.ErrorWithPos
	var warningErrorsWithPos []reporter.ErrorWithPos
	handler := reporter.NewHandler(reporter.NewReporter(
		func(errorWithPos reporter.ErrorWithPos) error {
			errorsWithPos = append(errorsWithPos, errorWithPos)
			return nil
		},
		func(errorWithPos reporter.ErrorWithPos) {
			warningErrorsWithPos = append(warningErrorsWithPos, errorWithPos)
		},
	))
	parsed, err := parser.Parse(file.uri.Filename(), strings.NewReader(file.file.Text()), handler)
	if err == nil {
		_, _ = parser.ResultFromAST(parsed, true, handler)
	}
	if len(errorsWithPos) > 0 {
		return nil, fmt.Errorf("cannot format file %q, %v error(s) found", file.uri.Filename(), len(errorsWithPos))
	}
	// Currently we have no way to honor any of the parameters.
	_ = params
	if parsed == nil {
		return nil, nil
	}
	var out strings.Builder
	if err := bufformat.FormatFileNode(&out, parsed); err != nil {
		return nil, err
	}
	newText := out.String()
	if newText == file.file.Text() {
		return nil, nil
	}

	// Calculate the end location for the file range.
	endLine := strings.Count(file.file.Text(), "\n")
	endCharacter := 0
	for _, char := range file.file.Text()[strings.LastIndexByte(file.file.Text(), '\n')+1:] {
		endCharacter += utf16.RuneLen(char)
	}
	return []protocol.TextEdit{
		{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      0,
					Character: 0,
				},
				End: protocol.Position{
					Line:      uint32(endLine),
					Character: uint32(endCharacter),
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
	if file := s.fileManager.Get(params.TextDocument.URI); file != nil {
		file.Close(ctx)
	}
	return nil
}

// -- Language functionality methods.

// Hover is the entry point for hover inlays.
func (s *server) Hover(
	ctx context.Context,
	params *protocol.HoverParams,
) (*protocol.Hover, error) {
	symbol := s.getSymbol(ctx, params.TextDocument.URI, params.Position)
	if symbol == nil {
		return nil, nil
	}

	docs := symbol.FormatDocs()
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
	symbol := s.getSymbol(ctx, params.TextDocument.URI, params.Position)
	if symbol == nil {
		return nil, nil
	}
	return []protocol.Location{
		symbol.Definition(),
	}, nil
}

// TypeDefinition is the entry point for go-to type-definition.
func (s *server) TypeDefinition(
	ctx context.Context,
	params *protocol.TypeDefinitionParams,
) ([]protocol.Location, error) {
	symbol := s.getSymbol(ctx, params.TextDocument.URI, params.Position)
	if symbol == nil {
		return nil, nil
	}
	return []protocol.Location{
		symbol.TypeDefinition(),
	}, nil
}

// References is the entry point for get-all-references.
func (s *server) References(
	ctx context.Context,
	params *protocol.ReferenceParams,
) ([]protocol.Location, error) {
	symbol := s.getSymbol(ctx, params.TextDocument.URI, params.Position)
	if symbol == nil {
		return nil, nil
	}
	// We deduplicate the references here in the case where a file's symbols have not yet
	// been refreshed, but a new file with references to symbols in said file is opened. This
	// can cause duplicate references to be appended and not all clients deduplicate the
	// returned references.
	//
	// We also do not want to refresh all symbols in the workspace when a single file is
	// interacted with, since that could be detrimental to performance.
	return xslices.Deduplicate(symbol.References(params.Context.IncludeDeclaration)), nil
}

// Completion is the entry point for code completion.
func (s *server) Completion(
	ctx context.Context,
	params *protocol.CompletionParams,
) (*protocol.CompletionList, error) {
	file := s.fileManager.Get(params.TextDocument.URI)
	if file == nil {
		return nil, nil
	}
	items := getCompletionItems(ctx, file, params.Position)
	if len(items) == 0 {
		return nil, nil
	}
	return &protocol.CompletionList{Items: items}, nil
}

// CompletionResolve is the entry point for resolving additional details for a completion item.
func (s *server) CompletionResolve(
	ctx context.Context,
	params *protocol.CompletionItem,
) (*protocol.CompletionItem, error) {
	return resolveCompletionItem(ctx, params)
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
	// In the case where there are no symbols for the file, we return nil for SemanticTokensFull.
	// This is based on the specification for the method textDocument/semanticTokens/full,
	// the expected response is the union type `SemanticTokens | null`.
	// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_semanticTokens
	if len(file.symbols) == 0 {
		return nil, nil
	}
	// This fairly painful encoding is described in detail here:
	// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_semanticTokens
	var (
		encoded           []uint32
		prevLine, prevCol uint32
	)
	for _, symbol := range file.symbols {
		var semanticType uint32
		var semanticModifier uint32
		_, ok := symbol.kind.(*option)
		if ok {
			semanticType = semanticTypeDecorator
		} else {
			switch symbol.ir.Kind() {
			case ir.SymbolKindPackage:
				semanticType = semanticTypeNamespace
			case ir.SymbolKindMessage:
				semanticType = semanticTypeStruct
			case ir.SymbolKindEnum:
				semanticType = semanticTypeEnum
			case ir.SymbolKindField:
				// For predeclared types, we set semanticType to semanticTypeType
				if symbol.IsBuiltIn() {
					semanticType = semanticTypeType
					semanticModifier += semanticModifierDefaultLibrary
				} else {
					semanticType = semanticTypeVariable
				}
			case ir.SymbolKindEnumValue:
				semanticType = semanticTypeEnumMember
			case ir.SymbolKindExtension:
				semanticType = semanticTypeVariable
			case ir.SymbolKindService:
				semanticType = semanticTypeInterface
			case ir.SymbolKindMethod:
				semanticType = semanticTypeMethod
			default:
				continue
			}
		}
		if _, ok := symbol.ir.Deprecated().AsBool(); ok {
			semanticModifier += semanticModifierDeprecated
		}

		startLocation := symbol.span.Location(symbol.span.Start, positionalEncoding)
		endLocation := symbol.span.Location(symbol.span.End, positionalEncoding)

		for i := startLocation.Line; i <= endLocation.Line; i++ {
			newLine := uint32(i - 1)
			var newCol uint32
			if i == startLocation.Line {
				newCol = uint32(startLocation.Column - 1)
				if prevLine == newLine {
					newCol -= prevCol
				}
			}
			symbolLen := uint32(endLocation.Column - 1)
			if i == startLocation.Line {
				symbolLen -= uint32(startLocation.Column - 1)
			}
			encoded = append(encoded, newLine-prevLine, newCol, symbolLen, semanticType, semanticModifier)
			prevLine = newLine
			if i == startLocation.Line {
				prevCol = uint32(startLocation.Column - 1)
			} else {
				prevCol = 0
			}
		}
	}
	if len(encoded) == 0 {
		return nil, nil
	}
	return &protocol.SemanticTokens{Data: encoded}, nil
}

// Symbols is the entry point for workspace-wide symbol search.
func (s *server) Symbols(
	ctx context.Context,
	params *protocol.WorkspaceSymbolParams,
) ([]protocol.SymbolInformation, error) {
	query := strings.ToLower(params.Query)
	var results []protocol.SymbolInformation
	for _, file := range s.fileManager.uriToFile.Range {
		for symbol := range file.GetSymbols(query) {
			results = append(results, symbol)
			if len(results) > maxSymbolResults {
				break
			}
		}
	}
	return results, nil
}

// DocumentSymbol is the entry point for document symbol search.
func (s *server) DocumentSymbol(ctx context.Context, params *protocol.DocumentSymbolParams) (
	result []any, // []protocol.SymbolInformation
	err error,
) {
	file := s.fileManager.Get(params.TextDocument.URI)
	if file == nil {
		return nil, nil
	}
	anyResults := []any{}
	for symbol := range file.GetSymbols("") {
		anyResults = append(anyResults, symbol)
		if len(anyResults) > maxSymbolResults {
			break
		}
	}
	return anyResults, nil
}

// getSymbol is a helper function that gets the *[symbol] for the given [protocol.URI] and
// [protocol.Position].
func (s *server) getSymbol(
	ctx context.Context,
	uri protocol.URI,
	position protocol.Position,
) *symbol {
	file := s.fileManager.Get(uri)
	if file == nil {
		return nil
	}
	return file.SymbolAt(ctx, position)
}

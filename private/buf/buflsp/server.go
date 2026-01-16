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
	"regexp"
	"runtime/debug"
	"strings"
	"unicode/utf16"

	celpv "buf.build/go/protovalidate/cel"
	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/buf/bufformat"
	"github.com/bufbuild/protocompile/parser"
	"github.com/bufbuild/protocompile/reporter"
	"github.com/google/cel-go/cel"
	"go.lsp.dev/protocol"
	"mvdan.cc/xurls/v2"
)

const (
	serverName = "buf-lsp"

	maxSymbolResults = 1000
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

	// httpsURLRegex is used to find https:// URLs in comments for document links.
	httpsURLRegex *regexp.Regexp
	// celEnv is the CEL environment used for parsing protovalidate expressions.
	celEnv *cel.Env
}

// newServer creates a protocol.Server implementation out of an lsp.
func newServer(lsp *lsp) (protocol.Server, error) {
	httpsURLRegex, err := xurls.StrictMatchingScheme("https://")
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTPS URL regex: %w", err)
	}
	celEnv, err := cel.NewEnv(
		cel.Lib(celpv.NewLibrary()),
		cel.EnableMacroCallTracking(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}
	return &server{
		lsp:           lsp,
		httpsURLRegex: httpsURLRegex,
		celEnv:        celEnv,
	}, nil
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
			CodeActionProvider: &protocol.CodeActionOptions{
				CodeActionKinds: []protocol.CodeActionKind{
					protocol.SourceOrganizeImports,
					CodeActionKindSourceDeprecate,
				},
			},
			CompletionProvider: &protocol.CompletionOptions{
				ResolveProvider:   true,
				TriggerCharacters: []string{".", "\"", "/"},
			},
			DefinitionProvider:         &protocol.DefinitionOptions{},
			TypeDefinitionProvider:     &protocol.TypeDefinitionOptions{},
			DocumentFormattingProvider: true,
			DocumentHighlightProvider:  true,
			HoverProvider:              true,
			ReferencesProvider:         &protocol.ReferenceOptions{},
			RenameProvider: &protocol.RenameOptions{
				PrepareProvider: true,
			},
			SemanticTokensProvider: &SemanticTokensOptions{
				Legend: SemanticTokensLegend{
					TokenTypes:     semanticTypeLegend,
					TokenModifiers: semanticModifierLegend,
				},
				Full: true,
			},
			WorkspaceSymbolProvider: true,
			DocumentSymbolProvider:  true,
			FoldingRangeProvider:    true,
			DocumentLinkProvider:    &protocol.DocumentLinkOptions{},
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
	return semanticTokensFull(s.fileManager.Get(params.TextDocument.URI), s.celEnv)
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

// DocumentHighlight is the entry point for highlighting occurrences of a symbol within a file.
//
// Supported symbol types for highlighting are [referenceable] and [reference] (messages, enums,
// extensions, and their references). All highlights use Text kind.
// Services, RPC methods, enum values, and field names are not highlighted.
func (s *server) DocumentHighlight(ctx context.Context, params *protocol.DocumentHighlightParams) ([]protocol.DocumentHighlight, error) {
	symbol := s.getSymbol(ctx, params.TextDocument.URI, params.Position)
	if symbol == nil {
		return nil, nil
	}
	return symbol.DocumentHighlights(), nil
}

// CodeAction is called when the client requests code actions for a given range.
func (s *server) CodeAction(ctx context.Context, params *protocol.CodeActionParams) ([]protocol.CodeAction, error) {
	file := s.fileManager.Get(params.TextDocument.URI)
	if file == nil {
		return nil, nil
	}
	codeActionSet := xslices.ToStructMap(params.Context.Only)

	var actions []protocol.CodeAction
	if _, ok := codeActionSet[protocol.SourceOrganizeImports]; len(codeActionSet) == 0 || ok {
		if organizeImportsAction := s.getOrganizeImportsCodeAction(ctx, file); organizeImportsAction != nil {
			actions = append(actions, *organizeImportsAction)
		}
	}
	if _, ok := codeActionSet[CodeActionKindSourceDeprecate]; len(codeActionSet) == 0 || ok {
		if deprecateAction := s.getDeprecateCodeAction(ctx, file, params); deprecateAction != nil {
			actions = append(actions, *deprecateAction)
		}
	}
	return actions, nil
}

// PrepareRename is the entry point for checking workspace wide renaming of a symbol.
//
// If a symbol can be renamed, PrepareRename will return the range for the rename. Returning
// an empty range indicates that the requested position cannot be renamed and the client
// will handle providing feedback to the user.
//
// Supported symbol types for renaming are [referenceable], [static], and [reference].
func (s *server) PrepareRename(ctx context.Context, params *protocol.PrepareRenameParams) (*protocol.Range, error) {
	symbol := s.getSymbol(ctx, params.TextDocument.URI, params.Position)
	if symbol == nil {
		return nil, nil
	}
	switch symbol.kind.(type) {
	case *referenceable, *static, *reference:
		// Don't allow renaming symbols defined in non-local files (e.g., WKTs from ~/.cache)
		// Check the definition if available, otherwise check the symbol's own file
		defFile := symbol.file
		if symbol.def != nil && symbol.def.file != nil {
			defFile = symbol.def.file
		}
		if defFile != nil && !defFile.IsLocal() {
			return nil, fmt.Errorf("cannot rename a symbol in a non-local file")
		}
		rnge := reportSpanToProtocolRange(symbol.span)
		return &rnge, nil
	}
	return nil, nil
}

// Rename is the entry point for workspace wide renaming of a symbol.
func (s *server) Rename(
	ctx context.Context,
	params *protocol.RenameParams,
) (*protocol.WorkspaceEdit, error) {
	symbol := s.getSymbol(ctx, params.TextDocument.URI, params.Position)
	if symbol == nil {
		return nil, nil
	}
	// Don't allow renaming symbols defined in non-local files (e.g., WKTs from ~/.cache)
	// Check the definition if available, otherwise check the symbol's own file
	defFile := symbol.file
	if symbol.def != nil && symbol.def.file != nil {
		defFile = symbol.def.file
	}
	if defFile != nil && !defFile.IsLocal() {
		return nil, fmt.Errorf("cannot rename a symbol in a non-local file")
	}
	return symbol.Rename(params.NewName)
}

// FoldingRanges is the entry point for folding ranges, which allows collapsing/expanding blocks of code.
func (s *server) FoldingRanges(
	ctx context.Context,
	params *protocol.FoldingRangeParams,
) ([]protocol.FoldingRange, error) {
	file := s.fileManager.Get(params.TextDocument.URI)
	if file == nil {
		return nil, nil
	}
	return s.foldingRange(file), nil
}

// DocumentLink is the entry point for document links, which makes imports and URLs clickable.
func (s *server) DocumentLink(
	ctx context.Context,
	params *protocol.DocumentLinkParams,
) ([]protocol.DocumentLink, error) {
	file := s.fileManager.Get(params.TextDocument.URI)
	if file == nil {
		return nil, nil
	}
	return s.documentLink(file), nil
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

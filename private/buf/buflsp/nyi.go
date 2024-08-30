// Copyright 2020-2024 Buf Technologies, Inc.
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

// This file provides an implementation of protocol.Server where every function returns an error.

package buflsp

import (
	"context"
	"fmt"
	"runtime"

	"go.lsp.dev/protocol"
)

// validate the protocol.Server implementation.
var _ protocol.Server = nyiServer{}

// nyiServer implements protocol. Server, but every function returns an error.
type nyiServer struct{}

// nyi returns a "not yet implemented" error containing the name of the function that called it.
func makeNYI() error {
	caller := "<unknown function>"
	if pc, _, _, ok := runtime.Caller(1); ok {
		if fn := runtime.FuncForPC(pc); fn != nil {
			caller = fn.Name()
		}
	}

	return fmt.Errorf("not yet implemented, sorry: %s", caller)
}

// NOTE: The functions below were generated using code completion. Do not edit!

func (nyiServer) CodeAction(ctx context.Context, params *protocol.CodeActionParams) (result []protocol.CodeAction, err error) {
	return nil, makeNYI()
}
func (nyiServer) CodeLens(ctx context.Context, params *protocol.CodeLensParams) (result []protocol.CodeLens, err error) {
	return nil, makeNYI()
}
func (nyiServer) CodeLensRefresh(ctx context.Context) (err error) {
	return makeNYI()
}
func (nyiServer) CodeLensResolve(ctx context.Context, params *protocol.CodeLens) (result *protocol.CodeLens, err error) {
	return nil, makeNYI()
}
func (nyiServer) ColorPresentation(ctx context.Context, params *protocol.ColorPresentationParams) (result []protocol.ColorPresentation, err error) {
	return nil, makeNYI()
}
func (nyiServer) Completion(ctx context.Context, params *protocol.CompletionParams) (result *protocol.CompletionList, err error) {
	return nil, makeNYI()
}
func (nyiServer) CompletionResolve(ctx context.Context, params *protocol.CompletionItem) (result *protocol.CompletionItem, err error) {
	return nil, makeNYI()
}
func (nyiServer) Declaration(ctx context.Context, params *protocol.DeclarationParams) (result []protocol.Location, err error) {
	return nil, makeNYI()
}
func (nyiServer) Definition(ctx context.Context, params *protocol.DefinitionParams) (result []protocol.Location, err error) {
	return nil, makeNYI()
}
func (nyiServer) DidChange(ctx context.Context, params *protocol.DidChangeTextDocumentParams) (err error) {
	return makeNYI()
}
func (nyiServer) DidChangeConfiguration(ctx context.Context, params *protocol.DidChangeConfigurationParams) (err error) {
	return makeNYI()
}
func (nyiServer) DidChangeWatchedFiles(ctx context.Context, params *protocol.DidChangeWatchedFilesParams) (err error) {
	return makeNYI()
}
func (nyiServer) DidChangeWorkspaceFolders(ctx context.Context, params *protocol.DidChangeWorkspaceFoldersParams) (err error) {
	return makeNYI()
}
func (nyiServer) DidClose(ctx context.Context, params *protocol.DidCloseTextDocumentParams) (err error) {
	return makeNYI()
}
func (nyiServer) DidCreateFiles(ctx context.Context, params *protocol.CreateFilesParams) (err error) {
	return makeNYI()
}
func (nyiServer) DidDeleteFiles(ctx context.Context, params *protocol.DeleteFilesParams) (err error) {
	return makeNYI()
}
func (nyiServer) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) (err error) {
	return makeNYI()
}
func (nyiServer) DidRenameFiles(ctx context.Context, params *protocol.RenameFilesParams) (err error) {
	return makeNYI()
}
func (nyiServer) DidSave(ctx context.Context, params *protocol.DidSaveTextDocumentParams) (err error) {
	return makeNYI()
}
func (nyiServer) DocumentColor(ctx context.Context, params *protocol.DocumentColorParams) (result []protocol.ColorInformation, err error) {
	return nil, makeNYI()
}
func (nyiServer) DocumentHighlight(ctx context.Context, params *protocol.DocumentHighlightParams) (result []protocol.DocumentHighlight, err error) {
	return nil, makeNYI()
}
func (nyiServer) DocumentLink(ctx context.Context, params *protocol.DocumentLinkParams) (result []protocol.DocumentLink, err error) {
	return nil, makeNYI()
}
func (nyiServer) DocumentLinkResolve(ctx context.Context, params *protocol.DocumentLink) (result *protocol.DocumentLink, err error) {
	return nil, makeNYI()
}
func (nyiServer) DocumentSymbol(ctx context.Context, params *protocol.DocumentSymbolParams) (result []interface{}, err error) {
	return nil, makeNYI()
}
func (nyiServer) ExecuteCommand(ctx context.Context, params *protocol.ExecuteCommandParams) (result interface{}, err error) {
	return nil, makeNYI()
}
func (nyiServer) Exit(ctx context.Context) (err error) {
	return makeNYI()
}
func (nyiServer) FoldingRanges(ctx context.Context, params *protocol.FoldingRangeParams) (result []protocol.FoldingRange, err error) {
	return nil, makeNYI()
}
func (nyiServer) Formatting(ctx context.Context, params *protocol.DocumentFormattingParams) (result []protocol.TextEdit, err error) {
	return nil, makeNYI()
}
func (nyiServer) Hover(ctx context.Context, params *protocol.HoverParams) (result *protocol.Hover, err error) {
	return nil, makeNYI()
}
func (nyiServer) Implementation(ctx context.Context, params *protocol.ImplementationParams) (result []protocol.Location, err error) {
	return nil, makeNYI()
}
func (nyiServer) IncomingCalls(ctx context.Context, params *protocol.CallHierarchyIncomingCallsParams) (result []protocol.CallHierarchyIncomingCall, err error) {
	return nil, makeNYI()
}
func (nyiServer) Initialize(ctx context.Context, params *protocol.InitializeParams) (result *protocol.InitializeResult, err error) {
	return nil, makeNYI()
}
func (nyiServer) Initialized(ctx context.Context, params *protocol.InitializedParams) (err error) {
	return makeNYI()
}
func (nyiServer) LinkedEditingRange(ctx context.Context, params *protocol.LinkedEditingRangeParams) (result *protocol.LinkedEditingRanges, err error) {
	return nil, makeNYI()
}
func (nyiServer) LogTrace(ctx context.Context, params *protocol.LogTraceParams) (err error) {
	return makeNYI()
}
func (nyiServer) Moniker(ctx context.Context, params *protocol.MonikerParams) (result []protocol.Moniker, err error) {
	return nil, makeNYI()
}
func (nyiServer) OnTypeFormatting(ctx context.Context, params *protocol.DocumentOnTypeFormattingParams) (result []protocol.TextEdit, err error) {
	return nil, makeNYI()
}
func (nyiServer) OutgoingCalls(ctx context.Context, params *protocol.CallHierarchyOutgoingCallsParams) (result []protocol.CallHierarchyOutgoingCall, err error) {
	return nil, makeNYI()
}
func (nyiServer) PrepareCallHierarchy(ctx context.Context, params *protocol.CallHierarchyPrepareParams) (result []protocol.CallHierarchyItem, err error) {
	return nil, makeNYI()
}
func (nyiServer) PrepareRename(ctx context.Context, params *protocol.PrepareRenameParams) (result *protocol.Range, err error) {
	return nil, makeNYI()
}
func (nyiServer) RangeFormatting(ctx context.Context, params *protocol.DocumentRangeFormattingParams) (result []protocol.TextEdit, err error) {
	return nil, makeNYI()
}
func (nyiServer) References(ctx context.Context, params *protocol.ReferenceParams) (result []protocol.Location, err error) {
	return nil, makeNYI()
}
func (nyiServer) Rename(ctx context.Context, params *protocol.RenameParams) (result *protocol.WorkspaceEdit, err error) {
	return nil, makeNYI()
}
func (nyiServer) Request(ctx context.Context, method string, params interface{}) (result interface{}, err error) {
	return nil, makeNYI()
}
func (nyiServer) SemanticTokensFull(ctx context.Context, params *protocol.SemanticTokensParams) (result *protocol.SemanticTokens, err error) {
	return nil, makeNYI()
}
func (nyiServer) SemanticTokensFullDelta(ctx context.Context, params *protocol.SemanticTokensDeltaParams) (result interface{}, err error) {
	return nil, makeNYI()
}
func (nyiServer) SemanticTokensRange(ctx context.Context, params *protocol.SemanticTokensRangeParams) (result *protocol.SemanticTokens, err error) {
	return nil, makeNYI()
}
func (nyiServer) SemanticTokensRefresh(ctx context.Context) (err error) {
	return makeNYI()
}
func (nyiServer) SetTrace(ctx context.Context, params *protocol.SetTraceParams) (err error) {
	return makeNYI()
}
func (nyiServer) ShowDocument(ctx context.Context, params *protocol.ShowDocumentParams) (result *protocol.ShowDocumentResult, err error) {
	return nil, makeNYI()
}
func (nyiServer) Shutdown(ctx context.Context) (err error) {
	return makeNYI()
}
func (nyiServer) SignatureHelp(ctx context.Context, params *protocol.SignatureHelpParams) (result *protocol.SignatureHelp, err error) {
	return nil, makeNYI()
}
func (nyiServer) Symbols(ctx context.Context, params *protocol.WorkspaceSymbolParams) (result []protocol.SymbolInformation, err error) {
	return nil, makeNYI()
}
func (nyiServer) TypeDefinition(ctx context.Context, params *protocol.TypeDefinitionParams) (result []protocol.Location, err error) {
	return nil, makeNYI()
}
func (nyiServer) WillCreateFiles(ctx context.Context, params *protocol.CreateFilesParams) (result *protocol.WorkspaceEdit, err error) {
	return nil, makeNYI()
}
func (nyiServer) WillDeleteFiles(ctx context.Context, params *protocol.DeleteFilesParams) (result *protocol.WorkspaceEdit, err error) {
	return nil, makeNYI()
}
func (nyiServer) WillRenameFiles(ctx context.Context, params *protocol.RenameFilesParams) (result *protocol.WorkspaceEdit, err error) {
	return nil, makeNYI()
}
func (nyiServer) WillSave(ctx context.Context, params *protocol.WillSaveTextDocumentParams) (err error) {
	return makeNYI()
}
func (nyiServer) WillSaveWaitUntil(ctx context.Context, params *protocol.WillSaveTextDocumentParams) (result []protocol.TextEdit, err error) {
	return nil, makeNYI()
}
func (nyiServer) WorkDoneProgressCancel(ctx context.Context, params *protocol.WorkDoneProgressCancelParams) (err error) {
	return makeNYI()
}

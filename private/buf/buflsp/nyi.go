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
var _ protocol.Server = nyi{}

// nyi implements protocol. Server, but every function returns an error.
type nyi struct{}

// NOTE: The functions below were generated using code completion. Do not edit!

func (nyi) CodeAction(ctx context.Context, params *protocol.CodeActionParams) (result []protocol.CodeAction, err error) {
	return nil, newNYIError()
}
func (nyi) CodeLens(ctx context.Context, params *protocol.CodeLensParams) (result []protocol.CodeLens, err error) {
	return nil, newNYIError()
}
func (nyi) CodeLensRefresh(ctx context.Context) (err error) {
	return newNYIError()
}
func (nyi) CodeLensResolve(ctx context.Context, params *protocol.CodeLens) (result *protocol.CodeLens, err error) {
	return nil, newNYIError()
}
func (nyi) ColorPresentation(ctx context.Context, params *protocol.ColorPresentationParams) (result []protocol.ColorPresentation, err error) {
	return nil, newNYIError()
}
func (nyi) Completion(ctx context.Context, params *protocol.CompletionParams) (result *protocol.CompletionList, err error) {
	return nil, newNYIError()
}
func (nyi) CompletionResolve(ctx context.Context, params *protocol.CompletionItem) (result *protocol.CompletionItem, err error) {
	return nil, newNYIError()
}
func (nyi) Declaration(ctx context.Context, params *protocol.DeclarationParams) (result []protocol.Location, err error) {
	return nil, newNYIError()
}
func (nyi) Definition(ctx context.Context, params *protocol.DefinitionParams) (result []protocol.Location, err error) {
	return nil, newNYIError()
}
func (nyi) DidChange(ctx context.Context, params *protocol.DidChangeTextDocumentParams) (err error) {
	return newNYIError()
}
func (nyi) DidChangeConfiguration(ctx context.Context, params *protocol.DidChangeConfigurationParams) (err error) {
	return newNYIError()
}
func (nyi) DidChangeWatchedFiles(ctx context.Context, params *protocol.DidChangeWatchedFilesParams) (err error) {
	return newNYIError()
}
func (nyi) DidChangeWorkspaceFolders(ctx context.Context, params *protocol.DidChangeWorkspaceFoldersParams) (err error) {
	return newNYIError()
}
func (nyi) DidClose(ctx context.Context, params *protocol.DidCloseTextDocumentParams) (err error) {
	return newNYIError()
}
func (nyi) DidCreateFiles(ctx context.Context, params *protocol.CreateFilesParams) (err error) {
	return newNYIError()
}
func (nyi) DidDeleteFiles(ctx context.Context, params *protocol.DeleteFilesParams) (err error) {
	return newNYIError()
}
func (nyi) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) (err error) {
	return newNYIError()
}
func (nyi) DidRenameFiles(ctx context.Context, params *protocol.RenameFilesParams) (err error) {
	return newNYIError()
}
func (nyi) DidSave(ctx context.Context, params *protocol.DidSaveTextDocumentParams) (err error) {
	return newNYIError()
}
func (nyi) DocumentColor(ctx context.Context, params *protocol.DocumentColorParams) (result []protocol.ColorInformation, err error) {
	return nil, newNYIError()
}
func (nyi) DocumentHighlight(ctx context.Context, params *protocol.DocumentHighlightParams) (result []protocol.DocumentHighlight, err error) {
	return nil, newNYIError()
}
func (nyi) DocumentLink(ctx context.Context, params *protocol.DocumentLinkParams) (result []protocol.DocumentLink, err error) {
	return nil, newNYIError()
}
func (nyi) DocumentLinkResolve(ctx context.Context, params *protocol.DocumentLink) (result *protocol.DocumentLink, err error) {
	return nil, newNYIError()
}
func (nyi) DocumentSymbol(ctx context.Context, params *protocol.DocumentSymbolParams) (result []interface{}, err error) {
	return nil, newNYIError()
}
func (nyi) ExecuteCommand(ctx context.Context, params *protocol.ExecuteCommandParams) (result interface{}, err error) {
	return nil, newNYIError()
}
func (nyi) Exit(ctx context.Context) (err error) {
	return newNYIError()
}
func (nyi) FoldingRanges(ctx context.Context, params *protocol.FoldingRangeParams) (result []protocol.FoldingRange, err error) {
	return nil, newNYIError()
}
func (nyi) Formatting(ctx context.Context, params *protocol.DocumentFormattingParams) (result []protocol.TextEdit, err error) {
	return nil, newNYIError()
}
func (nyi) Hover(ctx context.Context, params *protocol.HoverParams) (result *protocol.Hover, err error) {
	return nil, newNYIError()
}
func (nyi) Implementation(ctx context.Context, params *protocol.ImplementationParams) (result []protocol.Location, err error) {
	return nil, newNYIError()
}
func (nyi) IncomingCalls(ctx context.Context, params *protocol.CallHierarchyIncomingCallsParams) (result []protocol.CallHierarchyIncomingCall, err error) {
	return nil, newNYIError()
}
func (nyi) Initialize(ctx context.Context, params *protocol.InitializeParams) (result *protocol.InitializeResult, err error) {
	return nil, newNYIError()
}
func (nyi) Initialized(ctx context.Context, params *protocol.InitializedParams) (err error) {
	return newNYIError()
}
func (nyi) LinkedEditingRange(ctx context.Context, params *protocol.LinkedEditingRangeParams) (result *protocol.LinkedEditingRanges, err error) {
	return nil, newNYIError()
}
func (nyi) LogTrace(ctx context.Context, params *protocol.LogTraceParams) (err error) {
	return newNYIError()
}
func (nyi) Moniker(ctx context.Context, params *protocol.MonikerParams) (result []protocol.Moniker, err error) {
	return nil, newNYIError()
}
func (nyi) OnTypeFormatting(ctx context.Context, params *protocol.DocumentOnTypeFormattingParams) (result []protocol.TextEdit, err error) {
	return nil, newNYIError()
}
func (nyi) OutgoingCalls(ctx context.Context, params *protocol.CallHierarchyOutgoingCallsParams) (result []protocol.CallHierarchyOutgoingCall, err error) {
	return nil, newNYIError()
}
func (nyi) PrepareCallHierarchy(ctx context.Context, params *protocol.CallHierarchyPrepareParams) (result []protocol.CallHierarchyItem, err error) {
	return nil, newNYIError()
}
func (nyi) PrepareRename(ctx context.Context, params *protocol.PrepareRenameParams) (result *protocol.Range, err error) {
	return nil, newNYIError()
}
func (nyi) RangeFormatting(ctx context.Context, params *protocol.DocumentRangeFormattingParams) (result []protocol.TextEdit, err error) {
	return nil, newNYIError()
}
func (nyi) References(ctx context.Context, params *protocol.ReferenceParams) (result []protocol.Location, err error) {
	return nil, newNYIError()
}
func (nyi) Rename(ctx context.Context, params *protocol.RenameParams) (result *protocol.WorkspaceEdit, err error) {
	return nil, newNYIError()
}
func (nyi) Request(ctx context.Context, method string, params interface{}) (result interface{}, err error) {
	return nil, newNYIError()
}
func (nyi) SemanticTokensFull(ctx context.Context, params *protocol.SemanticTokensParams) (result *protocol.SemanticTokens, err error) {
	return nil, newNYIError()
}
func (nyi) SemanticTokensFullDelta(ctx context.Context, params *protocol.SemanticTokensDeltaParams) (result interface{}, err error) {
	return nil, newNYIError()
}
func (nyi) SemanticTokensRange(ctx context.Context, params *protocol.SemanticTokensRangeParams) (result *protocol.SemanticTokens, err error) {
	return nil, newNYIError()
}
func (nyi) SemanticTokensRefresh(ctx context.Context) (err error) {
	return newNYIError()
}
func (nyi) SetTrace(ctx context.Context, params *protocol.SetTraceParams) (err error) {
	return newNYIError()
}
func (nyi) ShowDocument(ctx context.Context, params *protocol.ShowDocumentParams) (result *protocol.ShowDocumentResult, err error) {
	return nil, newNYIError()
}
func (nyi) Shutdown(ctx context.Context) (err error) {
	return newNYIError()
}
func (nyi) SignatureHelp(ctx context.Context, params *protocol.SignatureHelpParams) (result *protocol.SignatureHelp, err error) {
	return nil, newNYIError()
}
func (nyi) Symbols(ctx context.Context, params *protocol.WorkspaceSymbolParams) (result []protocol.SymbolInformation, err error) {
	return nil, newNYIError()
}
func (nyi) TypeDefinition(ctx context.Context, params *protocol.TypeDefinitionParams) (result []protocol.Location, err error) {
	return nil, newNYIError()
}
func (nyi) WillCreateFiles(ctx context.Context, params *protocol.CreateFilesParams) (result *protocol.WorkspaceEdit, err error) {
	return nil, newNYIError()
}
func (nyi) WillDeleteFiles(ctx context.Context, params *protocol.DeleteFilesParams) (result *protocol.WorkspaceEdit, err error) {
	return nil, newNYIError()
}
func (nyi) WillRenameFiles(ctx context.Context, params *protocol.RenameFilesParams) (result *protocol.WorkspaceEdit, err error) {
	return nil, newNYIError()
}
func (nyi) WillSave(ctx context.Context, params *protocol.WillSaveTextDocumentParams) (err error) {
	return newNYIError()
}
func (nyi) WillSaveWaitUntil(ctx context.Context, params *protocol.WillSaveTextDocumentParams) (result []protocol.TextEdit, err error) {
	return nil, newNYIError()
}
func (nyi) WorkDoneProgressCancel(ctx context.Context, params *protocol.WorkDoneProgressCancelParams) (err error) {
	return newNYIError()
}

// nyi returns a "not yet implemented" error containing the name of the function that called it.
func newNYIError() error {
	caller := "<unknown function>"
	if pc, _, _, ok := runtime.Caller(1); ok {
		if fn := runtime.FuncForPC(pc); fn != nil {
			caller = fn.Name()
		}
	}

	return fmt.Errorf("not yet implemented, sorry: %s", caller)
}

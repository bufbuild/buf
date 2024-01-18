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

package buflsp

import (
	"context"
	"errors"

	"go.lsp.dev/protocol"
)

type nopServer struct{}

func (nopServer) Initialize(ctx context.Context, params *protocol.InitializeParams) (*protocol.InitializeResult, error) {
	return nil, errors.New("not implemented: Initialize")
}

func (nopServer) Initialized(ctx context.Context, params *protocol.InitializedParams) error {
	return nil
}

func (nopServer) Shutdown(ctx context.Context) error {
	return errors.New("not implemented: Shutdown")
}

func (nopServer) Exit(ctx context.Context) error {
	return errors.New("not implemented: Exit")
}

func (nopServer) WorkDoneProgressCancel(ctx context.Context, params *protocol.WorkDoneProgressCancelParams) error {
	return errors.New("not implemented: WorkDoneProgressCancel")
}

func (nopServer) LogTrace(ctx context.Context, params *protocol.LogTraceParams) error {
	return errors.New("not implemented: LogTrace")
}

func (nopServer) SetTrace(ctx context.Context, params *protocol.SetTraceParams) error {
	return nil
}

func (nopServer) CodeAction(ctx context.Context, params *protocol.CodeActionParams) ([]protocol.CodeAction, error) {
	return nil, errors.New("not implemented: CodeAction")
}

func (nopServer) CodeLens(ctx context.Context, params *protocol.CodeLensParams) ([]protocol.CodeLens, error) {
	return nil, errors.New("not implemented: CodeLens")
}

func (nopServer) CodeLensResolve(ctx context.Context, params *protocol.CodeLens) (*protocol.CodeLens, error) {
	return nil, errors.New("not implemented: CodeLensResolve")
}

func (nopServer) ColorPresentation(ctx context.Context, params *protocol.ColorPresentationParams) ([]protocol.ColorPresentation, error) {
	return nil, errors.New("not implemented: ColorPresentation")
}

func (nopServer) Completion(ctx context.Context, params *protocol.CompletionParams) (*protocol.CompletionList, error) {
	return nil, errors.New("not implemented: Completion")
}

func (nopServer) CompletionResolve(ctx context.Context, params *protocol.CompletionItem) (*protocol.CompletionItem, error) {
	return nil, errors.New("not implemented: CompletionResolve")
}

func (nopServer) Declaration(ctx context.Context, params *protocol.DeclarationParams) ([]protocol.Location /* Declaration | DeclarationLink[] | null */, error) {
	return nil, errors.New("not implemented: Declaration")
}

func (nopServer) Definition(ctx context.Context, params *protocol.DefinitionParams) ([]protocol.Location /* Definition | DefinitionLink[] | null */, error) {
	return nil, errors.New("not implemented: Definition")
}

func (nopServer) DidChange(ctx context.Context, params *protocol.DidChangeTextDocumentParams) error {
	return errors.New("not implemented: DidChange")
}

func (nopServer) DidChangeConfiguration(ctx context.Context, params *protocol.DidChangeConfigurationParams) error {
	return errors.New("not implemented: DidChangeConfiguration")
}

func (nopServer) DidChangeWatchedFiles(ctx context.Context, params *protocol.DidChangeWatchedFilesParams) error {
	return errors.New("not implemented: DidChangeWatchedFiles")
}

func (nopServer) DidChangeWorkspaceFolders(ctx context.Context, params *protocol.DidChangeWorkspaceFoldersParams) error {
	return errors.New("not implemented: DidChangeWorkspaceFolders")
}

func (nopServer) DidClose(ctx context.Context, params *protocol.DidCloseTextDocumentParams) error {
	return errors.New("not implemented: DidClose")
}

func (nopServer) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) error {
	return errors.New("not implemented: DidOpen")
}

func (nopServer) DidSave(ctx context.Context, params *protocol.DidSaveTextDocumentParams) error {
	return errors.New("not implemented: DidSave")
}

func (nopServer) DocumentColor(ctx context.Context, params *protocol.DocumentColorParams) ([]protocol.ColorInformation, error) {
	return nil, errors.New("not implemented: DocumentColor")
}

func (nopServer) DocumentHighlight(ctx context.Context, params *protocol.DocumentHighlightParams) ([]protocol.DocumentHighlight, error) {
	return nil, errors.New("not implemented: DocumentHighlight")
}

func (nopServer) DocumentLink(ctx context.Context, params *protocol.DocumentLinkParams) ([]protocol.DocumentLink, error) {
	return nil, errors.New("not implemented: DocumentLink")
}

func (nopServer) DocumentLinkResolve(ctx context.Context, params *protocol.DocumentLink) (*protocol.DocumentLink, error) {
	return nil, errors.New("not implemented: DocumentLinkResolve")
}

func (nopServer) DocumentSymbol(ctx context.Context, params *protocol.DocumentSymbolParams) ([]interface{} /* []SymbolInformation | []DocumentSymbol */, error) {
	return nil, errors.New("not implemented: DocumentSymbol")
}

func (nopServer) ExecuteCommand(ctx context.Context, params *protocol.ExecuteCommandParams) (interface{}, error) {
	return nil, errors.New("not implemented: ExecuteCommand")
}

func (nopServer) FoldingRanges(ctx context.Context, params *protocol.FoldingRangeParams) ([]protocol.FoldingRange, error) {
	return nil, errors.New("not implemented: FoldingRanges")
}

func (nopServer) Formatting(ctx context.Context, params *protocol.DocumentFormattingParams) ([]protocol.TextEdit, error) {
	return nil, errors.New("not implemented: Formatting")
}

func (nopServer) Hover(ctx context.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	return nil, errors.New("not implemented: Hover")
}

func (nopServer) Implementation(ctx context.Context, params *protocol.ImplementationParams) ([]protocol.Location, error) {
	return nil, errors.New("not implemented: Implementation")
}

func (nopServer) OnTypeFormatting(ctx context.Context, params *protocol.DocumentOnTypeFormattingParams) ([]protocol.TextEdit, error) {
	return nil, errors.New("not implemented: OnTypeFormatting")
}

func (nopServer) PrepareRename(ctx context.Context, params *protocol.PrepareRenameParams) (*protocol.Range, error) {
	return nil, errors.New("not implemented: PrepareRename")
}

func (nopServer) RangeFormatting(ctx context.Context, params *protocol.DocumentRangeFormattingParams) ([]protocol.TextEdit, error) {
	return nil, errors.New("not implemented: RangeFormatting")
}

func (nopServer) References(ctx context.Context, params *protocol.ReferenceParams) ([]protocol.Location, error) {
	return nil, errors.New("not implemented: References")
}

func (nopServer) Rename(ctx context.Context, params *protocol.RenameParams) (*protocol.WorkspaceEdit, error) {
	return nil, errors.New("not implemented: Rename")
}

func (nopServer) SignatureHelp(ctx context.Context, params *protocol.SignatureHelpParams) (*protocol.SignatureHelp, error) {
	return nil, errors.New("not implemented: SignatureHelp")
}

func (nopServer) Symbols(ctx context.Context, params *protocol.WorkspaceSymbolParams) ([]protocol.SymbolInformation, error) {
	return nil, errors.New("not implemented: Symbols")
}

func (nopServer) TypeDefinition(ctx context.Context, params *protocol.TypeDefinitionParams) ([]protocol.Location, error) {
	return nil, errors.New("not implemented: TypeDefinition")
}

func (nopServer) WillSave(ctx context.Context, params *protocol.WillSaveTextDocumentParams) error {
	return errors.New("not implemented: WillSave")
}

func (nopServer) WillSaveWaitUntil(ctx context.Context, params *protocol.WillSaveTextDocumentParams) ([]protocol.TextEdit, error) {
	return nil, errors.New("not implemented: WillSaveWaitUntil")
}

func (nopServer) ShowDocument(ctx context.Context, params *protocol.ShowDocumentParams) (*protocol.ShowDocumentResult, error) {
	return nil, errors.New("not implemented: ShowDocument")
}

func (nopServer) WillCreateFiles(ctx context.Context, params *protocol.CreateFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, errors.New("not implemented: WillCreateFiles")
}

func (nopServer) DidCreateFiles(ctx context.Context, params *protocol.CreateFilesParams) error {
	return errors.New("not implemented: DidCreateFiles")
}

func (nopServer) WillRenameFiles(ctx context.Context, params *protocol.RenameFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, errors.New("not implemented: WillRenameFiles")
}

func (nopServer) DidRenameFiles(ctx context.Context, params *protocol.RenameFilesParams) error {
	return errors.New("not implemented: DidRenameFiles")
}

func (nopServer) WillDeleteFiles(ctx context.Context, params *protocol.DeleteFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, errors.New("not implemented: WillDeleteFiles")
}

func (nopServer) DidDeleteFiles(ctx context.Context, params *protocol.DeleteFilesParams) error {
	return errors.New("not implemented: DidDeleteFiles")
}

func (nopServer) CodeLensRefresh(ctx context.Context) error {
	return errors.New("not implemented: CodeLensRefresh")
}

func (nopServer) PrepareCallHierarchy(ctx context.Context, params *protocol.CallHierarchyPrepareParams) ([]protocol.CallHierarchyItem, error) {
	return nil, errors.New("not implemented: PrepareCallHierarchy")
}

func (nopServer) IncomingCalls(ctx context.Context, params *protocol.CallHierarchyIncomingCallsParams) ([]protocol.CallHierarchyIncomingCall, error) {
	return nil, errors.New("not implemented: IncomingCalls")
}

func (nopServer) OutgoingCalls(ctx context.Context, params *protocol.CallHierarchyOutgoingCallsParams) ([]protocol.CallHierarchyOutgoingCall, error) {
	return nil, errors.New("not implemented: OutgoingCalls")
}

func (nopServer) SemanticTokensFull(ctx context.Context, params *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	return nil, errors.New("not implemented: SemanticTokensFull")
}

func (nopServer) SemanticTokensFullDelta(ctx context.Context, params *protocol.SemanticTokensDeltaParams) (interface{} /* SemanticTokens | SemanticTokensDelta */, error) {
	return nil, errors.New("not implemented: SemanticTokensFullDelta")
}

func (nopServer) SemanticTokensRange(ctx context.Context, params *protocol.SemanticTokensRangeParams) (*protocol.SemanticTokens, error) {
	return nil, errors.New("not implemented: SemanticTokensRange")
}

func (nopServer) SemanticTokensRefresh(ctx context.Context) error {
	return errors.New("not implemented: SemanticTokensRefresh")
}

func (nopServer) LinkedEditingRange(ctx context.Context, params *protocol.LinkedEditingRangeParams) (*protocol.LinkedEditingRanges, error) {
	return nil, errors.New("not implemented: LinkedEditingRange")
}

func (nopServer) Moniker(ctx context.Context, params *protocol.MonikerParams) ([]protocol.Moniker, error) {
	return nil, errors.New("not implemented: Moniker")
}

func (nopServer) Request(ctx context.Context, method string, params interface{}) (interface{}, error) {
	return nil, errors.New("not implemented: Request")
}

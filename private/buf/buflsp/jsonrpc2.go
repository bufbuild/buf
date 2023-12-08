// Copyright 2020-2023 Buf Technologies, Inc.
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

type noopServer struct {
}

func (ns *noopServer) Initialize(ctx context.Context, params *protocol.InitializeParams) (*protocol.InitializeResult, error) {
	return nil, errors.New("not implemented: Initialize")
}

func (ns *noopServer) Initialized(ctx context.Context, params *protocol.InitializedParams) error {
	return nil
}
func (ns *noopServer) Shutdown(ctx context.Context) error {
	return errors.New("not implemented: Shutdown")
}
func (ns *noopServer) Exit(ctx context.Context) error {
	return errors.New("not implemented: Exit")
}
func (ns *noopServer) WorkDoneProgressCancel(ctx context.Context, params *protocol.WorkDoneProgressCancelParams) error {
	return errors.New("not implemented: WorkDoneProgressCancel")
}
func (ns *noopServer) LogTrace(ctx context.Context, params *protocol.LogTraceParams) error {
	return errors.New("not implemented: LogTrace")
}
func (ns *noopServer) SetTrace(ctx context.Context, params *protocol.SetTraceParams) error {
	return nil
}
func (ns *noopServer) CodeAction(ctx context.Context, params *protocol.CodeActionParams) ([]protocol.CodeAction, error) {
	return nil, errors.New("not implemented: CodeAction")
}

func (ns *noopServer) CodeLens(ctx context.Context, params *protocol.CodeLensParams) ([]protocol.CodeLens, error) {
	return nil, errors.New("not implemented: CodeLens")
}

func (ns *noopServer) CodeLensResolve(ctx context.Context, params *protocol.CodeLens) (*protocol.CodeLens, error) {
	return nil, errors.New("not implemented: CodeLensResolve")
}

func (ns *noopServer) ColorPresentation(ctx context.Context, params *protocol.ColorPresentationParams) ([]protocol.ColorPresentation, error) {
	return nil, errors.New("not implemented: ColorPresentation")
}

func (ns *noopServer) Completion(ctx context.Context, params *protocol.CompletionParams) (*protocol.CompletionList, error) {
	return nil, errors.New("not implemented: Completion")
}

func (ns *noopServer) CompletionResolve(ctx context.Context, params *protocol.CompletionItem) (*protocol.CompletionItem, error) {
	return nil, errors.New("not implemented: CompletionResolve")
}

func (ns *noopServer) Declaration(ctx context.Context, params *protocol.DeclarationParams) ([]protocol.Location /* Declaration | DeclarationLink[] | null */, error) {
	return nil, errors.New("not implemented: Declaration")
}

func (ns *noopServer) Definition(ctx context.Context, params *protocol.DefinitionParams) ([]protocol.Location /* Definition | DefinitionLink[] | null */, error) {
	return nil, errors.New("not implemented: Definition")
}

func (ns *noopServer) DidChange(ctx context.Context, params *protocol.DidChangeTextDocumentParams) error {
	return errors.New("not implemented: DidChange")
}
func (ns *noopServer) DidChangeConfiguration(ctx context.Context, params *protocol.DidChangeConfigurationParams) error {
	return errors.New("not implemented: DidChangeConfiguration")
}
func (ns *noopServer) DidChangeWatchedFiles(ctx context.Context, params *protocol.DidChangeWatchedFilesParams) error {
	return errors.New("not implemented: DidChangeWatchedFiles")
}
func (ns *noopServer) DidChangeWorkspaceFolders(ctx context.Context, params *protocol.DidChangeWorkspaceFoldersParams) error {
	return errors.New("not implemented: DidChangeWorkspaceFolders")
}
func (ns *noopServer) DidClose(ctx context.Context, params *protocol.DidCloseTextDocumentParams) error {
	return errors.New("not implemented: DidClose")
}
func (ns *noopServer) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) error {
	return errors.New("not implemented: DidOpen")
}
func (ns *noopServer) DidSave(ctx context.Context, params *protocol.DidSaveTextDocumentParams) error {
	return errors.New("not implemented: DidSave")
}
func (ns *noopServer) DocumentColor(ctx context.Context, params *protocol.DocumentColorParams) ([]protocol.ColorInformation, error) {
	return nil, errors.New("not implemented: DocumentColor")
}

func (ns *noopServer) DocumentHighlight(ctx context.Context, params *protocol.DocumentHighlightParams) ([]protocol.DocumentHighlight, error) {
	return nil, errors.New("not implemented: DocumentHighlight")
}

func (ns *noopServer) DocumentLink(ctx context.Context, params *protocol.DocumentLinkParams) ([]protocol.DocumentLink, error) {
	return nil, errors.New("not implemented: DocumentLink")
}

func (ns *noopServer) DocumentLinkResolve(ctx context.Context, params *protocol.DocumentLink) (*protocol.DocumentLink, error) {
	return nil, errors.New("not implemented: DocumentLinkResolve")
}

func (ns *noopServer) DocumentSymbol(ctx context.Context, params *protocol.DocumentSymbolParams) ([]interface{} /* []SymbolInformation | []DocumentSymbol */, error) {
	return nil, errors.New("not implemented: DocumentSymbol")
}

func (ns *noopServer) ExecuteCommand(ctx context.Context, params *protocol.ExecuteCommandParams) (interface{}, error) {
	return nil, errors.New("not implemented: ExecuteCommand")
}

func (ns *noopServer) FoldingRanges(ctx context.Context, params *protocol.FoldingRangeParams) ([]protocol.FoldingRange, error) {
	return nil, errors.New("not implemented: FoldingRanges")
}

func (ns *noopServer) Formatting(ctx context.Context, params *protocol.DocumentFormattingParams) ([]protocol.TextEdit, error) {
	return nil, errors.New("not implemented: Formatting")
}

func (ns *noopServer) Hover(ctx context.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	return nil, errors.New("not implemented: Hover")
}

func (ns *noopServer) Implementation(ctx context.Context, params *protocol.ImplementationParams) ([]protocol.Location, error) {
	return nil, errors.New("not implemented: Implementation")
}

func (ns *noopServer) OnTypeFormatting(ctx context.Context, params *protocol.DocumentOnTypeFormattingParams) ([]protocol.TextEdit, error) {
	return nil, errors.New("not implemented: OnTypeFormatting")
}

func (ns *noopServer) PrepareRename(ctx context.Context, params *protocol.PrepareRenameParams) (*protocol.Range, error) {
	return nil, errors.New("not implemented: PrepareRename")
}

func (ns *noopServer) RangeFormatting(ctx context.Context, params *protocol.DocumentRangeFormattingParams) ([]protocol.TextEdit, error) {
	return nil, errors.New("not implemented: RangeFormatting")
}

func (ns *noopServer) References(ctx context.Context, params *protocol.ReferenceParams) ([]protocol.Location, error) {
	return nil, errors.New("not implemented: References")
}

func (ns *noopServer) Rename(ctx context.Context, params *protocol.RenameParams) (*protocol.WorkspaceEdit, error) {
	return nil, errors.New("not implemented: Rename")
}

func (ns *noopServer) SignatureHelp(ctx context.Context, params *protocol.SignatureHelpParams) (*protocol.SignatureHelp, error) {
	return nil, errors.New("not implemented: SignatureHelp")
}

func (ns *noopServer) Symbols(ctx context.Context, params *protocol.WorkspaceSymbolParams) ([]protocol.SymbolInformation, error) {
	return nil, errors.New("not implemented: Symbols")
}

func (ns *noopServer) TypeDefinition(ctx context.Context, params *protocol.TypeDefinitionParams) ([]protocol.Location, error) {
	return nil, errors.New("not implemented: TypeDefinition")
}

func (ns *noopServer) WillSave(ctx context.Context, params *protocol.WillSaveTextDocumentParams) error {
	return errors.New("not implemented: WillSave")
}
func (ns *noopServer) WillSaveWaitUntil(ctx context.Context, params *protocol.WillSaveTextDocumentParams) ([]protocol.TextEdit, error) {
	return nil, errors.New("not implemented: WillSaveWaitUntil")
}

func (ns *noopServer) ShowDocument(ctx context.Context, params *protocol.ShowDocumentParams) (*protocol.ShowDocumentResult, error) {
	return nil, errors.New("not implemented: ShowDocument")
}

func (ns *noopServer) WillCreateFiles(ctx context.Context, params *protocol.CreateFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, errors.New("not implemented: WillCreateFiles")
}

func (ns *noopServer) DidCreateFiles(ctx context.Context, params *protocol.CreateFilesParams) error {
	return errors.New("not implemented: DidCreateFiles")
}
func (ns *noopServer) WillRenameFiles(ctx context.Context, params *protocol.RenameFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, errors.New("not implemented: WillRenameFiles")
}

func (ns *noopServer) DidRenameFiles(ctx context.Context, params *protocol.RenameFilesParams) error {
	return errors.New("not implemented: DidRenameFiles")
}
func (ns *noopServer) WillDeleteFiles(ctx context.Context, params *protocol.DeleteFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, errors.New("not implemented: WillDeleteFiles")
}

func (ns *noopServer) DidDeleteFiles(ctx context.Context, params *protocol.DeleteFilesParams) error {
	return errors.New("not implemented: DidDeleteFiles")
}
func (ns *noopServer) CodeLensRefresh(ctx context.Context) error {
	return errors.New("not implemented: CodeLensRefresh")
}
func (ns *noopServer) PrepareCallHierarchy(ctx context.Context, params *protocol.CallHierarchyPrepareParams) ([]protocol.CallHierarchyItem, error) {
	return nil, errors.New("not implemented: PrepareCallHierarchy")
}

func (ns *noopServer) IncomingCalls(ctx context.Context, params *protocol.CallHierarchyIncomingCallsParams) ([]protocol.CallHierarchyIncomingCall, error) {
	return nil, errors.New("not implemented: IncomingCalls")
}

func (ns *noopServer) OutgoingCalls(ctx context.Context, params *protocol.CallHierarchyOutgoingCallsParams) ([]protocol.CallHierarchyOutgoingCall, error) {
	return nil, errors.New("not implemented: OutgoingCalls")
}

func (ns *noopServer) SemanticTokensFull(ctx context.Context, params *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	return nil, errors.New("not implemented: SemanticTokensFull")
}

func (ns *noopServer) SemanticTokensFullDelta(ctx context.Context, params *protocol.SemanticTokensDeltaParams) (interface{} /* SemanticTokens | SemanticTokensDelta */, error) {
	return nil, errors.New("not implemented: SemanticTokensFullDelta")
}

func (ns *noopServer) SemanticTokensRange(ctx context.Context, params *protocol.SemanticTokensRangeParams) (*protocol.SemanticTokens, error) {
	return nil, errors.New("not implemented: SemanticTokensRange")
}

func (ns *noopServer) SemanticTokensRefresh(ctx context.Context) error {
	return errors.New("not implemented: SemanticTokensRefresh")
}
func (ns *noopServer) LinkedEditingRange(ctx context.Context, params *protocol.LinkedEditingRangeParams) (*protocol.LinkedEditingRanges, error) {
	return nil, errors.New("not implemented: LinkedEditingRange")
}

func (ns *noopServer) Moniker(ctx context.Context, params *protocol.MonikerParams) ([]protocol.Moniker, error) {
	return nil, errors.New("not implemented: Moniker")
}

func (ns *noopServer) Request(ctx context.Context, method string, params interface{}) (interface{}, error) {
	return nil, errors.New("not implemented: Request")
}

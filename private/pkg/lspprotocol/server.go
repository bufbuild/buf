// Copyright 2020-2026 Buf Technologies, Inc.
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

package lspprotocol

import (
	"context"
	"encoding/json"

	"github.com/bufbuild/buf/private/pkg/jsonrpc2"
)

// Server represents a Language Server Protocol server.
type Server interface {
	CodeAction(ctx context.Context, params *CodeActionParams) (result []CodeAction, err error)
	CodeLens(ctx context.Context, params *CodeLensParams) (result []CodeLens, err error)
	CodeLensRefresh(ctx context.Context) (err error)
	CodeLensResolve(ctx context.Context, params *CodeLens) (result *CodeLens, err error)
	ColorPresentation(ctx context.Context, params *ColorPresentationParams) (result []ColorPresentation, err error)
	Completion(ctx context.Context, params *CompletionParams) (result *CompletionList, err error)
	CompletionResolve(ctx context.Context, params *CompletionItem) (result *CompletionItem, err error)
	Declaration(ctx context.Context, params *DeclarationParams) (result []Location, err error)
	Definition(ctx context.Context, params *DefinitionParams) (result []Location, err error)
	DidChange(ctx context.Context, params *DidChangeTextDocumentParams) (err error)
	DidChangeConfiguration(ctx context.Context, params *DidChangeConfigurationParams) (err error)
	DidChangeWatchedFiles(ctx context.Context, params *DidChangeWatchedFilesParams) (err error)
	DidChangeWorkspaceFolders(ctx context.Context, params *DidChangeWorkspaceFoldersParams) (err error)
	DidClose(ctx context.Context, params *DidCloseTextDocumentParams) (err error)
	DidCreateFiles(ctx context.Context, params *CreateFilesParams) (err error)
	DidDeleteFiles(ctx context.Context, params *DeleteFilesParams) (err error)
	DidOpen(ctx context.Context, params *DidOpenTextDocumentParams) (err error)
	DidRenameFiles(ctx context.Context, params *RenameFilesParams) (err error)
	DidSave(ctx context.Context, params *DidSaveTextDocumentParams) (err error)
	DocumentColor(ctx context.Context, params *DocumentColorParams) (result []ColorInformation, err error)
	DocumentHighlight(ctx context.Context, params *DocumentHighlightParams) (result []DocumentHighlight, err error)
	DocumentLink(ctx context.Context, params *DocumentLinkParams) (result []DocumentLink, err error)
	DocumentLinkResolve(ctx context.Context, params *DocumentLink) (result *DocumentLink, err error)
	DocumentSymbol(ctx context.Context, params *DocumentSymbolParams) (result []any, err error)
	ExecuteCommand(ctx context.Context, params *ExecuteCommandParams) (result any, err error)
	Exit(ctx context.Context) (err error)
	FoldingRanges(ctx context.Context, params *FoldingRangeParams) (result []FoldingRange, err error)
	Formatting(ctx context.Context, params *DocumentFormattingParams) (result []TextEdit, err error)
	Hover(ctx context.Context, params *HoverParams) (result *Hover, err error)
	Implementation(ctx context.Context, params *ImplementationParams) (result []Location, err error)
	IncomingCalls(ctx context.Context, params *CallHierarchyIncomingCallsParams) (result []CallHierarchyIncomingCall, err error)
	Initialize(ctx context.Context, params *InitializeParams) (result *InitializeResult, err error)
	Initialized(ctx context.Context, params *InitializedParams) (err error)
	LinkedEditingRange(ctx context.Context, params *LinkedEditingRangeParams) (result *LinkedEditingRanges, err error)
	LogTrace(ctx context.Context, params *LogTraceParams) (err error)
	Moniker(ctx context.Context, params *MonikerParams) (result []Moniker, err error)
	OnTypeFormatting(ctx context.Context, params *DocumentOnTypeFormattingParams) (result []TextEdit, err error)
	OutgoingCalls(ctx context.Context, params *CallHierarchyOutgoingCallsParams) (result []CallHierarchyOutgoingCall, err error)
	PrepareCallHierarchy(ctx context.Context, params *CallHierarchyPrepareParams) (result []CallHierarchyItem, err error)
	PrepareRename(ctx context.Context, params *PrepareRenameParams) (result *Range, err error)
	RangeFormatting(ctx context.Context, params *DocumentRangeFormattingParams) (result []TextEdit, err error)
	References(ctx context.Context, params *ReferenceParams) (result []Location, err error)
	Rename(ctx context.Context, params *RenameParams) (result *WorkspaceEdit, err error)
	Request(ctx context.Context, method string, params any) (result any, err error)
	SemanticTokensFull(ctx context.Context, params *SemanticTokensParams) (result *SemanticTokens, err error)
	SemanticTokensFullDelta(ctx context.Context, params *SemanticTokensDeltaParams) (result any, err error)
	SemanticTokensRange(ctx context.Context, params *SemanticTokensRangeParams) (result *SemanticTokens, err error)
	SemanticTokensRefresh(ctx context.Context) (err error)
	SetTrace(ctx context.Context, params *SetTraceParams) (err error)
	ShowDocument(ctx context.Context, params *ShowDocumentParams) (result *ShowDocumentResult, err error)
	Shutdown(ctx context.Context) (err error)
	SignatureHelp(ctx context.Context, params *SignatureHelpParams) (result *SignatureHelp, err error)
	Symbols(ctx context.Context, params *WorkspaceSymbolParams) (result []SymbolInformation, err error)
	TypeDefinition(ctx context.Context, params *TypeDefinitionParams) (result []Location, err error)
	WillCreateFiles(ctx context.Context, params *CreateFilesParams) (result *WorkspaceEdit, err error)
	WillDeleteFiles(ctx context.Context, params *DeleteFilesParams) (result *WorkspaceEdit, err error)
	WillRenameFiles(ctx context.Context, params *RenameFilesParams) (result *WorkspaceEdit, err error)
	WillSave(ctx context.Context, params *WillSaveTextDocumentParams) (err error)
	WillSaveWaitUntil(ctx context.Context, params *WillSaveTextDocumentParams) (result []TextEdit, err error)
	WorkDoneProgressCancel(ctx context.Context, params *WorkDoneProgressCancelParams) (err error)
}

// list of server method names.
const (
	// MethodCancelRequest method name of "$/cancelRequest".
	MethodCancelRequest = "$/cancelRequest"

	// MethodInitialize method name of "initialize".
	MethodInitialize = "initialize"

	// MethodInitialized method name of "initialized".
	MethodInitialized = "initialized"

	// MethodShutdown method name of "shutdown".
	MethodShutdown = "shutdown"

	// MethodExit method name of "exit".
	MethodExit = "exit"

	// MethodWorkDoneProgressCancel method name of "window/workDoneProgress/cancel".
	MethodWorkDoneProgressCancel = "window/workDoneProgress/cancel"

	// MethodLogTrace method name of "$/logTrace".
	MethodLogTrace = "$/logTrace"

	// MethodSetTrace method name of "$/setTrace".
	MethodSetTrace = "$/setTrace"

	// MethodTextDocumentCodeAction method name of "textDocument/codeAction".
	MethodTextDocumentCodeAction = "textDocument/codeAction"

	// MethodTextDocumentCodeLens method name of "textDocument/codeLens".
	MethodTextDocumentCodeLens = "textDocument/codeLens"

	// MethodCodeLensResolve method name of "codeLens/resolve".
	MethodCodeLensResolve = "codeLens/resolve"

	// MethodTextDocumentColorPresentation method name of "textDocument/colorPresentation".
	MethodTextDocumentColorPresentation = "textDocument/colorPresentation"

	// MethodTextDocumentCompletion method name of "textDocument/completion".
	MethodTextDocumentCompletion = "textDocument/completion"

	// MethodCompletionItemResolve method name of "completionItem/resolve".
	MethodCompletionItemResolve = "completionItem/resolve"

	// MethodTextDocumentDeclaration method name of "textDocument/declaration".
	MethodTextDocumentDeclaration = "textDocument/declaration"

	// MethodTextDocumentDefinition method name of "textDocument/definition".
	MethodTextDocumentDefinition = "textDocument/definition"

	// MethodTextDocumentDidChange method name of "textDocument/didChange".
	MethodTextDocumentDidChange = "textDocument/didChange"

	// MethodWorkspaceDidChangeConfiguration method name of "workspace/didChangeConfiguration".
	MethodWorkspaceDidChangeConfiguration = "workspace/didChangeConfiguration"

	// MethodWorkspaceDidChangeWatchedFiles method name of "workspace/didChangeWatchedFiles".
	MethodWorkspaceDidChangeWatchedFiles = "workspace/didChangeWatchedFiles"

	// MethodWorkspaceDidChangeWorkspaceFolders method name of "workspace/didChangeWorkspaceFolders".
	MethodWorkspaceDidChangeWorkspaceFolders = "workspace/didChangeWorkspaceFolders"

	// MethodTextDocumentDidClose method name of "textDocument/didClose".
	MethodTextDocumentDidClose = "textDocument/didClose"

	// MethodWorkspaceDidCreateFiles method name of "workspace/didCreateFiles".
	MethodWorkspaceDidCreateFiles = "workspace/didCreateFiles"

	// MethodWorkspaceDidDeleteFiles method name of "workspace/didDeleteFiles".
	MethodWorkspaceDidDeleteFiles = "workspace/didDeleteFiles"

	// MethodTextDocumentDidOpen method name of "textDocument/didOpen".
	MethodTextDocumentDidOpen = "textDocument/didOpen"

	// MethodWorkspaceDidRenameFiles method name of "workspace/didRenameFiles".
	MethodWorkspaceDidRenameFiles = "workspace/didRenameFiles"

	// MethodTextDocumentDidSave method name of "textDocument/didSave".
	MethodTextDocumentDidSave = "textDocument/didSave"

	// MethodTextDocumentDocumentColor method name of "textDocument/documentColor".
	MethodTextDocumentDocumentColor = "textDocument/documentColor"

	// MethodTextDocumentDocumentHighlight method name of "textDocument/documentHighlight".
	MethodTextDocumentDocumentHighlight = "textDocument/documentHighlight"

	// MethodTextDocumentDocumentLink method name of "textDocument/documentLink".
	MethodTextDocumentDocumentLink = "textDocument/documentLink"

	// MethodDocumentLinkResolve method name of "documentLink/resolve".
	MethodDocumentLinkResolve = "documentLink/resolve"

	// MethodTextDocumentDocumentSymbol method name of "textDocument/documentSymbol".
	MethodTextDocumentDocumentSymbol = "textDocument/documentSymbol"

	// MethodWorkspaceExecuteCommand method name of "workspace/executeCommand".
	MethodWorkspaceExecuteCommand = "workspace/executeCommand"

	// MethodTextDocumentFoldingRange method name of "textDocument/foldingRange".
	MethodTextDocumentFoldingRange = "textDocument/foldingRange"

	// MethodTextDocumentFormatting method name of "textDocument/formatting".
	MethodTextDocumentFormatting = "textDocument/formatting"

	// MethodTextDocumentHover method name of "textDocument/hover".
	MethodTextDocumentHover = "textDocument/hover"

	// MethodTextDocumentImplementation method name of "textDocument/implementation".
	MethodTextDocumentImplementation = "textDocument/implementation"

	// MethodCallHierarchyIncomingCalls method name of "callHierarchy/incomingCalls".
	MethodCallHierarchyIncomingCalls = "callHierarchy/incomingCalls"

	// MethodTextDocumentLinkedEditingRange method name of "textDocument/linkedEditingRange".
	MethodTextDocumentLinkedEditingRange = "textDocument/linkedEditingRange"

	// MethodTextDocumentMoniker method name of "textDocument/moniker".
	MethodTextDocumentMoniker = "textDocument/moniker"

	// MethodTextDocumentOnTypeFormatting method name of "textDocument/onTypeFormatting".
	MethodTextDocumentOnTypeFormatting = "textDocument/onTypeFormatting"

	// MethodCallHierarchyOutgoingCalls method name of "callHierarchy/outgoingCalls".
	MethodCallHierarchyOutgoingCalls = "callHierarchy/outgoingCalls"

	// MethodTextDocumentPrepareCallHierarchy method name of "textDocument/prepareCallHierarchy".
	MethodTextDocumentPrepareCallHierarchy = "textDocument/prepareCallHierarchy"

	// MethodTextDocumentPrepareRename method name of "textDocument/prepareRename".
	MethodTextDocumentPrepareRename = "textDocument/prepareRename"

	// MethodTextDocumentRangeFormatting method name of "textDocument/rangeFormatting".
	MethodTextDocumentRangeFormatting = "textDocument/rangeFormatting"

	// MethodTextDocumentReferences method name of "textDocument/references".
	MethodTextDocumentReferences = "textDocument/references"

	// MethodTextDocumentRename method name of "textDocument/rename".
	MethodTextDocumentRename = "textDocument/rename"

	// MethodTextDocumentSemanticTokensFull method name of "textDocument/semanticTokens/full".
	MethodTextDocumentSemanticTokensFull = "textDocument/semanticTokens/full"

	// MethodTextDocumentSemanticTokensFullDelta method name of "textDocument/semanticTokens/full/delta".
	MethodTextDocumentSemanticTokensFullDelta = "textDocument/semanticTokens/full/delta"

	// MethodTextDocumentSemanticTokensRange method name of "textDocument/semanticTokens/range".
	MethodTextDocumentSemanticTokensRange = "textDocument/semanticTokens/range"

	// MethodWorkspaceSemanticTokensRefresh method name of "workspace/semanticTokens/refresh".
	MethodWorkspaceSemanticTokensRefresh = "workspace/semanticTokens/refresh"

	// MethodTextDocumentSignatureHelp method name of "textDocument/signatureHelp".
	MethodTextDocumentSignatureHelp = "textDocument/signatureHelp"

	// MethodTextDocumentTypeDefinition method name of "textDocument/typeDefinition".
	MethodTextDocumentTypeDefinition = "textDocument/typeDefinition"

	// MethodWorkspaceWillCreateFiles method name of "workspace/willCreateFiles".
	MethodWorkspaceWillCreateFiles = "workspace/willCreateFiles"

	// MethodWorkspaceWillDeleteFiles method name of "workspace/willDeleteFiles".
	MethodWorkspaceWillDeleteFiles = "workspace/willDeleteFiles"

	// MethodWorkspaceWillRenameFiles method name of "workspace/willRenameFiles".
	MethodWorkspaceWillRenameFiles = "workspace/willRenameFiles"

	// MethodTextDocumentWillSave method name of "textDocument/willSave".
	MethodTextDocumentWillSave = "textDocument/willSave"

	// MethodTextDocumentWillSaveWaitUntil method name of "textDocument/willSaveWaitUntil".
	MethodTextDocumentWillSaveWaitUntil = "textDocument/willSaveWaitUntil"

	// MethodWorkspaceDiagnostic method name of "workspace/diagnostic".
	MethodWorkspaceDiagnostic = "workspace/diagnostic"

	// MethodWorkspaceDiagnosticRefresh method name of "workspace/diagnosticRefresh".
	MethodWorkspaceDiagnosticRefresh = "workspace/diagnosticRefresh"

	// MethodTextDocumentInlayHint method name of "textDocument/inlayHint".
	MethodTextDocumentInlayHint = "textDocument/inlayHint"

	// MethodInlayHintResolve method name of "inlayHint/resolve".
	MethodInlayHintResolve = "inlayHint/resolve"

	// MethodWorkspaceInlayHintRefresh method name of "workspace/inlayHint/refresh".
	MethodWorkspaceInlayHintRefresh = "workspace/inlayHint/refresh"

	// MethodWorkspaceSymbol method name of "workspace/symbol".
	MethodWorkspaceSymbol = "workspace/symbol"

	// MethodCodeLensRefresh method name of "workspace/codeLens/refresh".
	MethodCodeLensRefresh = "workspace/codeLens/refresh"

	// MethodShowDocument method name of "window/showDocument".
	MethodShowDocument = "window/showDocument"
)

// TraceValue aliases for buflsp package use.
const (
	TraceOff     = Off
	TraceMessage = Messages
	TraceVerbose = Verbose
)

// ErrRequestCancelled is returned when a request is cancelled.
var ErrRequestCancelled = &jsonrpc2.Error{Code: -32800, Message: "request cancelled"}

// ServerDispatch dispatches a request to the appropriate Server method based
// on req.Method. Returns (result, error); the caller is responsible for
// sending the response. Returns (nil, nil) for unrecognised methods.
func ServerDispatch(ctx context.Context, server Server, req *jsonrpc2.Request) (any, error) {
	switch req.Method {
	case MethodInitialize: // request
		var params InitializeParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.Initialize(ctx, &params)
		return resp, err

	case MethodInitialized: // notification
		var params InitializedParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		err := server.Initialized(ctx, &params)
		return nil, err

	case MethodShutdown: // request
		if req.Params != nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: "expected no params"}
		}
		err := server.Shutdown(ctx)
		return nil, err

	case MethodExit: // notification
		if req.Params != nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: "expected no params"}
		}
		err := server.Exit(ctx)
		return nil, err

	case MethodWorkDoneProgressCancel: // notification
		var params WorkDoneProgressCancelParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		err := server.WorkDoneProgressCancel(ctx, &params)
		return nil, err

	case MethodLogTrace: // notification
		var params LogTraceParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		err := server.LogTrace(ctx, &params)
		return nil, err

	case MethodSetTrace: // notification
		var params SetTraceParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		err := server.SetTrace(ctx, &params)
		return nil, err

	case MethodTextDocumentCodeAction: // request
		var params CodeActionParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.CodeAction(ctx, &params)
		return resp, err

	case MethodTextDocumentCodeLens: // request
		var params CodeLensParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.CodeLens(ctx, &params)
		return resp, err

	case MethodCodeLensResolve: // request
		var params CodeLens
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.CodeLensResolve(ctx, &params)
		return resp, err

	case MethodTextDocumentColorPresentation: // request
		var params ColorPresentationParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.ColorPresentation(ctx, &params)
		return resp, err

	case MethodTextDocumentCompletion: // request
		var params CompletionParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.Completion(ctx, &params)
		return resp, err

	case MethodCompletionItemResolve: // request
		var params CompletionItem
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.CompletionResolve(ctx, &params)
		return resp, err

	case MethodTextDocumentDeclaration: // request
		var params DeclarationParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.Declaration(ctx, &params)
		return resp, err

	case MethodTextDocumentDefinition: // request
		var params DefinitionParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.Definition(ctx, &params)
		return resp, err

	case MethodTextDocumentDidChange: // notification
		var params DidChangeTextDocumentParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		err := server.DidChange(ctx, &params)
		return nil, err

	case MethodWorkspaceDidChangeConfiguration: // notification
		var params DidChangeConfigurationParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		err := server.DidChangeConfiguration(ctx, &params)
		return nil, err

	case MethodWorkspaceDidChangeWatchedFiles: // notification
		var params DidChangeWatchedFilesParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		err := server.DidChangeWatchedFiles(ctx, &params)
		return nil, err

	case MethodWorkspaceDidChangeWorkspaceFolders: // notification
		var params DidChangeWorkspaceFoldersParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		err := server.DidChangeWorkspaceFolders(ctx, &params)
		return nil, err

	case MethodTextDocumentDidClose: // notification
		var params DidCloseTextDocumentParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		err := server.DidClose(ctx, &params)
		return nil, err

	case MethodTextDocumentDidOpen: // notification
		var params DidOpenTextDocumentParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		err := server.DidOpen(ctx, &params)
		return nil, err

	case MethodTextDocumentDidSave: // notification
		var params DidSaveTextDocumentParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		err := server.DidSave(ctx, &params)
		return nil, err

	case MethodTextDocumentDocumentColor: // request
		var params DocumentColorParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.DocumentColor(ctx, &params)
		return resp, err

	case MethodTextDocumentDocumentHighlight: // request
		var params DocumentHighlightParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.DocumentHighlight(ctx, &params)
		return resp, err

	case MethodTextDocumentDocumentLink: // request
		var params DocumentLinkParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.DocumentLink(ctx, &params)
		return resp, err

	case MethodDocumentLinkResolve: // request
		var params DocumentLink
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.DocumentLinkResolve(ctx, &params)
		return resp, err

	case MethodTextDocumentDocumentSymbol: // request
		var params DocumentSymbolParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.DocumentSymbol(ctx, &params)
		return resp, err

	case MethodWorkspaceExecuteCommand: // request
		var params ExecuteCommandParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.ExecuteCommand(ctx, &params)
		return resp, err

	case MethodTextDocumentFoldingRange: // request
		var params FoldingRangeParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.FoldingRanges(ctx, &params)
		return resp, err

	case MethodTextDocumentFormatting: // request
		var params DocumentFormattingParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.Formatting(ctx, &params)
		return resp, err

	case MethodTextDocumentHover: // request
		var params HoverParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.Hover(ctx, &params)
		return resp, err

	case MethodTextDocumentImplementation: // request
		var params ImplementationParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.Implementation(ctx, &params)
		return resp, err

	case MethodTextDocumentOnTypeFormatting: // request
		var params DocumentOnTypeFormattingParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.OnTypeFormatting(ctx, &params)
		return resp, err

	case MethodTextDocumentPrepareRename: // request
		var params PrepareRenameParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.PrepareRename(ctx, &params)
		return resp, err

	case MethodTextDocumentRangeFormatting: // request
		var params DocumentRangeFormattingParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.RangeFormatting(ctx, &params)
		return resp, err

	case MethodTextDocumentReferences: // request
		var params ReferenceParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.References(ctx, &params)
		return resp, err

	case MethodTextDocumentRename: // request
		var params RenameParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.Rename(ctx, &params)
		return resp, err

	case MethodTextDocumentSignatureHelp: // request
		var params SignatureHelpParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.SignatureHelp(ctx, &params)
		return resp, err

	case MethodWorkspaceSymbol: // request
		var params WorkspaceSymbolParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.Symbols(ctx, &params)
		return resp, err

	case MethodTextDocumentTypeDefinition: // request
		var params TypeDefinitionParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.TypeDefinition(ctx, &params)
		return resp, err

	case MethodTextDocumentWillSave: // notification
		var params WillSaveTextDocumentParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		err := server.WillSave(ctx, &params)
		return nil, err

	case MethodTextDocumentWillSaveWaitUntil: // request
		var params WillSaveTextDocumentParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.WillSaveWaitUntil(ctx, &params)
		return resp, err

	case MethodShowDocument: // request
		var params ShowDocumentParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.ShowDocument(ctx, &params)
		return resp, err

	case MethodWorkspaceWillCreateFiles: // request
		var params CreateFilesParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.WillCreateFiles(ctx, &params)
		return resp, err

	case MethodWorkspaceDidCreateFiles: // notification
		var params CreateFilesParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		err := server.DidCreateFiles(ctx, &params)
		return nil, err

	case MethodWorkspaceWillRenameFiles: // request
		var params RenameFilesParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.WillRenameFiles(ctx, &params)
		return resp, err

	case MethodWorkspaceDidRenameFiles: // notification
		var params RenameFilesParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		err := server.DidRenameFiles(ctx, &params)
		return nil, err

	case MethodWorkspaceWillDeleteFiles: // request
		var params DeleteFilesParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.WillDeleteFiles(ctx, &params)
		return resp, err

	case MethodWorkspaceDidDeleteFiles: // notification
		var params DeleteFilesParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		err := server.DidDeleteFiles(ctx, &params)
		return nil, err

	case MethodCodeLensRefresh: // request
		if req.Params != nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: "expected no params"}
		}
		err := server.CodeLensRefresh(ctx)
		return nil, err

	case MethodTextDocumentPrepareCallHierarchy: // request
		var params CallHierarchyPrepareParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.PrepareCallHierarchy(ctx, &params)
		return resp, err

	case MethodCallHierarchyIncomingCalls: // request
		var params CallHierarchyIncomingCallsParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.IncomingCalls(ctx, &params)
		return resp, err

	case MethodCallHierarchyOutgoingCalls: // request
		var params CallHierarchyOutgoingCallsParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.OutgoingCalls(ctx, &params)
		return resp, err

	case MethodTextDocumentSemanticTokensFull: // request
		var params SemanticTokensParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.SemanticTokensFull(ctx, &params)
		return resp, err

	case MethodTextDocumentSemanticTokensFullDelta: // request
		var params SemanticTokensDeltaParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.SemanticTokensFullDelta(ctx, &params)
		return resp, err

	case MethodTextDocumentSemanticTokensRange: // request
		var params SemanticTokensRangeParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.SemanticTokensRange(ctx, &params)
		return resp, err

	case MethodWorkspaceSemanticTokensRefresh: // request
		if req.Params != nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: "expected no params"}
		}
		err := server.SemanticTokensRefresh(ctx)
		return nil, err

	case MethodTextDocumentLinkedEditingRange: // request
		var params LinkedEditingRangeParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.LinkedEditingRange(ctx, &params)
		return resp, err

	case MethodTextDocumentMoniker: // request
		var params MonikerParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			}
		}
		resp, err := server.Moniker(ctx, &params)
		return resp, err

	default:
		return nil, nil
	}
}

// ServerHandler returns a Handler that dispatches LSP requests to server.
func ServerHandler(server Server) jsonrpc2.Handler {
	return jsonrpc2.HandlerFunc(func(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (any, error) {
		return ServerDispatch(ctx, server, req)
	})
}

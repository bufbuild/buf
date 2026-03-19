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

	"github.com/bufbuild/buf/private/pkg/jsonrpc2"
)

// Client represents a Language Server Protocol client.
type Client interface {
	Progress(ctx context.Context, params *ProgressParams) (err error)
	WorkDoneProgressCreate(ctx context.Context, params *WorkDoneProgressCreateParams) (err error)
	LogMessage(ctx context.Context, params *LogMessageParams) (err error)
	PublishDiagnostics(ctx context.Context, params *PublishDiagnosticsParams) (err error)
	ShowMessage(ctx context.Context, params *ShowMessageParams) (err error)
	ShowMessageRequest(ctx context.Context, params *ShowMessageRequestParams) (result *MessageActionItem, err error)
	Telemetry(ctx context.Context, params any) (err error)
	RegisterCapability(ctx context.Context, params *RegistrationParams) (err error)
	UnregisterCapability(ctx context.Context, params *UnregistrationParams) (err error)
	ApplyEdit(ctx context.Context, params *ApplyWorkspaceEditParams) (result bool, err error)
	Configuration(ctx context.Context, params *ConfigurationParams) (result []any, err error)
	WorkspaceFolders(ctx context.Context) (result []WorkspaceFolder, err error)
}

// list of client method names.
const (
	// MethodProgress method name of "$/progress".
	MethodProgress = "$/progress"

	// MethodWorkDoneProgressCreate method name of "window/workDoneProgress/create".
	MethodWorkDoneProgressCreate = "window/workDoneProgress/create"

	// MethodWindowLogMessage method name of "window/logMessage".
	MethodWindowLogMessage = "window/logMessage"

	// MethodWindowShowMessage method name of "window/showMessage".
	MethodWindowShowMessage = "window/showMessage"

	// MethodWindowShowMessageRequest method name of "window/showMessageRequest".
	MethodWindowShowMessageRequest = "window/showMessageRequest"

	// MethodTelemetryEvent method name of "telemetry/event".
	MethodTelemetryEvent = "telemetry/event"

	// MethodClientRegisterCapability method name of "client/registerCapability".
	MethodClientRegisterCapability = "client/registerCapability"

	// MethodClientUnregisterCapability method name of "client/unregisterCapability".
	MethodClientUnregisterCapability = "client/unregisterCapability"

	// MethodTextDocumentPublishDiagnostics method name of "textDocument/publishDiagnostics".
	MethodTextDocumentPublishDiagnostics = "textDocument/publishDiagnostics"

	// MethodWorkspaceApplyEdit method name of "workspace/applyEdit".
	MethodWorkspaceApplyEdit = "workspace/applyEdit"

	// MethodWorkspaceConfiguration method name of "workspace/configuration".
	MethodWorkspaceConfiguration = "workspace/configuration"

	// MethodWorkspaceWorkspaceFolders method name of "workspace/workspaceFolders".
	MethodWorkspaceWorkspaceFolders = "workspace/workspaceFolders"
)

// ClientDispatcher returns a Client that dispatches LSP notifications and
// requests over the given caller. The caller is typically a *jsonrpc2.Conn or
// a logging wrapper that implements jsonrpc2.Caller.
func ClientDispatcher(caller jsonrpc2.Caller) Client {
	return &client{conn: caller}
}

// client implements the Client interface by sending messages over a Caller.
type client struct {
	conn jsonrpc2.Caller
}

var _ Client = (*client)(nil)

// Progress is the base protocol support to report progress in a generic fashion.
//
// @since 3.16.0.
func (c *client) Progress(ctx context.Context, params *ProgressParams) error {
	return c.conn.Notify(ctx, MethodProgress, params)
}

// WorkDoneProgressCreate sends the request from the server to the client to ask the client to create a work done progress.
//
// @since 3.16.0.
func (c *client) WorkDoneProgressCreate(ctx context.Context, params *WorkDoneProgressCreateParams) error {
	return c.conn.Call(ctx, MethodWorkDoneProgressCreate, params, nil)
}

// LogMessage sends the notification from the server to the client to ask the client to log a particular message.
func (c *client) LogMessage(ctx context.Context, params *LogMessageParams) error {
	return c.conn.Notify(ctx, MethodWindowLogMessage, params)
}

// PublishDiagnostics sends the notification from the server to the client to signal results of validation runs.
func (c *client) PublishDiagnostics(ctx context.Context, params *PublishDiagnosticsParams) error {
	return c.conn.Notify(ctx, MethodTextDocumentPublishDiagnostics, params)
}

// ShowMessage sends the notification from a server to a client to ask the client to display a particular message in the user interface.
func (c *client) ShowMessage(ctx context.Context, params *ShowMessageParams) error {
	return c.conn.Notify(ctx, MethodWindowShowMessage, params)
}

// ShowMessageRequest sends the request from a server to a client to ask the client to display a particular message in the user interface.
//
// In addition to the show message notification the request allows to pass actions and to wait for an answer from the client.
func (c *client) ShowMessageRequest(ctx context.Context, params *ShowMessageRequestParams) (*MessageActionItem, error) {
	var result *MessageActionItem
	if err := c.conn.Call(ctx, MethodWindowShowMessageRequest, params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Telemetry sends the notification from the server to the client to ask the client to log a telemetry event.
func (c *client) Telemetry(ctx context.Context, params any) error {
	return c.conn.Notify(ctx, MethodTelemetryEvent, params)
}

// RegisterCapability sends the request from the server to the client to register for a new capability on the client side.
func (c *client) RegisterCapability(ctx context.Context, params *RegistrationParams) error {
	return c.conn.Call(ctx, MethodClientRegisterCapability, params, nil)
}

// UnregisterCapability sends the request from the server to the client to unregister a previously registered capability.
func (c *client) UnregisterCapability(ctx context.Context, params *UnregistrationParams) error {
	return c.conn.Call(ctx, MethodClientUnregisterCapability, params, nil)
}

// ApplyEdit sends the request from the server to the client to modify resource on the client side.
func (c *client) ApplyEdit(ctx context.Context, params *ApplyWorkspaceEditParams) (bool, error) {
	var result bool
	if err := c.conn.Call(ctx, MethodWorkspaceApplyEdit, params, &result); err != nil {
		return false, err
	}
	return result, nil
}

// Configuration sends the request from the server to the client to fetch configuration settings from the client.
func (c *client) Configuration(ctx context.Context, params *ConfigurationParams) ([]any, error) {
	var result []any
	if err := c.conn.Call(ctx, MethodWorkspaceConfiguration, params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// WorkspaceFolders sends the request from the server to the client to fetch the current open list of workspace folders.
//
// @since 3.6.0.
func (c *client) WorkspaceFolders(ctx context.Context) ([]WorkspaceFolder, error) {
	var result []WorkspaceFolder
	if err := c.conn.Call(ctx, MethodWorkspaceWorkspaceFolders, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

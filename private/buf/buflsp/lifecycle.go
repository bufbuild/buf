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

// This file defines all of the lifecycle message handlers for buflsp.server.

package buflsp

import (
	"context"
	"runtime/debug"

	"go.lsp.dev/protocol"
)

var serverInfo = makeServerInfo()

func makeServerInfo() protocol.ServerInfo {
	info := protocol.ServerInfo{Name: "buf-lsp"}
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		info.Version = buildInfo.Main.Version
	}
	return info
}

// Initialize is the first message the LSP receives from the client. This is where all
// initialization of the server wrt to the project is is invoked on must occur.
func (server *server) Initialize(
	ctx context.Context,
	params *protocol.InitializeParams,
) (*protocol.InitializeResult, error) {
	if err := server.init(ctx, params); err != nil {
		return nil, err
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
			},
			DefinitionProvider: true,
			HoverProvider:      true,
		},
		ServerInfo: &serverInfo,
	}, nil
}

// Initialized is sent by the client after it receives the Initialize response and has
// initialized itself. This is only a notification.
func (server *server) Initialized(
	ctx context.Context,
	params *protocol.InitializedParams,
) error {
	return nil
}

func (server *server) SetTrace(
	ctx context.Context,
	params *protocol.SetTraceParams,
) error {
	server.traceValue.Store(&params.Value)
	return nil
}

// Shutdown is sent by the client when it wants the server to shut down and exit.
// The client will wait until Shutdown returns, and then call Exit.
func (server *server) Shutdown(ctx context.Context) error {
	return nil
}

// Exit is a notification that the client has seen shutdown complete, and that the
// server should now exit.
func (server *server) Exit(ctx context.Context) error {
	// TODO: return an error if Shutdown() has not been called yet.

	// Close the connection. This will let the server shut down gracefully once this
	// notification is replied to.
	return server.conn.Close()
}

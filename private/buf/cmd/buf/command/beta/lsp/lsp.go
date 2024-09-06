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

// Package lsp defines the entry-point for the Buf LSP within the CLI.
//
// The actual implementation of the LSP lives under private/buf/buflsp
package lsp

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/buflsp"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/ioext"
	"github.com/spf13/pflag"
	"go.lsp.dev/jsonrpc2"
)

// NewCommand constructs the CLI command for executing the LSP.
func NewCommand(name string, builder appext.Builder) *appcmd.Command {
	var lsp lsp
	return &appcmd.Command{
		Use:       name,
		Short:     "Start the language server.",
		Args:      appcmd.NoArgs,
		Run:       builder.NewRunFunc(lsp.Listen),
		BindFlags: lsp.Bind,
	}
}

// lsp represents configuration state for the LSP command.
//
// In other words, this type translates user input into configration for the server itself. This
// type defines functions for starting the server according to that configuration. This is done
// so that the actual server implementation is completely agnostic to the transport.
type lsp struct {
	// A file path to a UNIX socket to use for IPC. If empty, stdio is used instead.
	PipePath string
}

// Bind sets up the CLI flags that the LSP needs.
func (lsp *lsp) Bind(flagSet *pflag.FlagSet) {
	// NOTE: --pipe is chosen because that's what the vscode LSP client expects.
	flagSet.StringVar(&lsp.PipePath, "pipe", "", "path to a UNIX socket to listen on; uses stdio if not specified")
}

// Listen starts the LSP server and listens on the configured
func (lsp *lsp) Listen(ctx context.Context, container appext.Container) error {
	bufcli.WarnBetaCommand(ctx, container)

	transport, err := lsp.dial(container)
	if err != nil {
		return err
	}

	conn, err := buflsp.Serve(ctx, container, jsonrpc2.NewStream(transport))
	if err != nil {
		return err
	}
	<-conn.Done()
	return conn.Err()
}

// dial opens a connection to the LSP client.
func (lsp *lsp) dial(container appext.Container) (io.ReadWriteCloser, error) {
	switch {
	case lsp.PipePath != "":
		conn, err := net.Dial("unix", lsp.PipePath)
		if err != nil {
			return nil, fmt.Errorf("could not open IPC socket %q: %w", lsp.PipePath, err)
		}
		return conn, nil

	// TODO: Add other transport implementations, such as TCP, here!

	default:
		// Fall back to stdio by default.
		return ioext.CompositeReadWriteCloser(
			container.Stdin(),
			container.Stdout(),
			ioext.NopCloser,
		), nil
	}
}

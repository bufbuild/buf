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

// Package lspserve defines the entry-point for the Buf LSP within the CLI.
//
// The actual implementation of the LSP lives under private/buf/buflsp
package lspserve

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"buf.build/go/standard/xio"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/buflsp"
	"github.com/bufbuild/protocompile/experimental/incremental"
	"github.com/spf13/pflag"
	"go.lsp.dev/jsonrpc2"
)

const (
	// pipe is chosen because that's what the vscode LSP client expects.
	pipeFlagName = "pipe"
)

// NewCommand constructs the CLI command for executing the LSP.
func NewCommand(
	name string,
	builder appext.Builder,
	deprecated string,
	hidden bool,
	beta bool,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:        name,
		Short:      "Start the language server",
		Args:       appcmd.NoArgs,
		Deprecated: deprecated,
		Hidden:     hidden,
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				if beta {
					bufcli.WarnBetaCommand(ctx, container)
				}
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	// A file path to a UNIX socket to use for IPC. If empty, stdio is used instead.
	PipePath string
}

// Bind sets up the CLI flags that the LSP needs.
func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.PipePath,
		pipeFlagName,
		"",
		"path to a UNIX socket to listen on; uses stdio if not specified",
	)
}

func newFlags() *flags {
	return &flags{}
}

// run starts the LSP server and listens on the configured.
func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) (retErr error) {
	transport, err := dial(container, flags)
	if err != nil {
		return err
	}

	wktStore, err := bufcli.NewWKTStore(container)
	if err != nil {
		return err
	}
	wktBucket, err := wktStore.GetBucket(ctx)
	if err != nil {
		return err
	}

	controller, err := bufcli.NewController(container)
	if err != nil {
		return err
	}

	wasmRuntime, err := bufcli.NewWasmRuntime(ctx, container)
	if err != nil {
		return err
	}
	defer func() {
		retErr = errors.Join(retErr, wasmRuntime.Close(ctx))
	}()

	conn, err := buflsp.Serve(
		ctx,
		wktBucket,
		container,
		controller,
		wasmRuntime,
		jsonrpc2.NewStream(transport),
		incremental.New(),
	)
	if err != nil {
		return err
	}
	<-conn.Done()
	return conn.Err()
}

// dial opens a connection to the LSP client.
func dial(container appext.Container, flags *flags) (io.ReadWriteCloser, error) {
	switch {
	case flags.PipePath != "":
		conn, err := net.Dial("unix", flags.PipePath)
		if err != nil {
			return nil, fmt.Errorf("could not open IPC socket %q: %w", flags.PipePath, err)
		}
		return conn, nil

	// TODO: Add other transport implementations, such as TCP, here!

	default:
		// Fall back to stdio by default.
		return xio.CompositeReadWriteCloser(
			container.Stdin(),
			container.Stdout(),
			xio.NopCloser,
		), nil
	}
}

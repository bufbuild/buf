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
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/ioext"
	"github.com/bufbuild/buf/private/pkg/wasm"
	"github.com/spf13/pflag"
	"go.lsp.dev/jsonrpc2"
	"go.uber.org/multierr"
)

const (
	// pipe is chosen because that's what the vscode LSP client expects.
	pipeFlagName = "pipe"
)

// NewCommand constructs the CLI command for executing the LSP.
func NewCommand(name string, builder appext.Builder) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name,
		Short: "Start the language server",
		Args:  appcmd.NoArgs,
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
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
	bufcli.WarnBetaCommand(ctx, container)

	transport, err := dial(container, flags)
	if err != nil {
		return err
	}

	controller, err := bufcli.NewController(container)
	if err != nil {
		return err
	}

	wasmRuntimeCacheDir, err := bufcli.CreateWasmRuntimeCacheDir(container)
	if err != nil {
		return err
	}
	wasmRuntime, err := wasm.NewRuntime(ctx, wasm.WithLocalCacheDir(wasmRuntimeCacheDir))
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, wasmRuntime.Close(ctx))
	}()
	checkClient, err := bufcheck.NewClient(
		container.Logger(),
		bufcheck.NewRunnerProvider(command.NewRunner(), wasmRuntime),
		bufcheck.ClientWithStderr(container.Stderr()),
	)
	if err != nil {
		return err
	}

	conn, err := buflsp.Serve(ctx, container, controller, checkClient, jsonrpc2.NewStream(transport))
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
		return ioext.CompositeReadWriteCloser(
			container.Stdin(),
			container.Stdout(),
			ioext.NopCloser,
		), nil
	}
}

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

package lsp

import (
	"context"
	"fmt"
	"net"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/ioext"
	"github.com/spf13/pflag"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name,
		Short: "Start the language server.",
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
	Port uint32
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.Uint32Var(&f.Port, "port", 0, "port to listen on")
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	bufcli.WarnBetaCommand(ctx, container)
	jsonrpc2Stream, err := newJSONRPC2Stream(container, flags)
	if err != nil {
		return err
	}
	jsonrpc2Conn := jsonrpc2.NewConn(jsonrpc2Stream)
	server, err := bufcli.NewLSPServer(ctx, container, jsonrpc2Conn)
	if err != nil {
		return err
	}
	jsonrpc2Conn.Go(ctx, protocol.ServerHandler(server, nil))
	<-ctx.Done()
	return jsonrpc2Conn.Err()
}

func newJSONRPC2Stream(container appext.Container, flags *flags) (jsonrpc2.Stream, error) {
	if flags.Port > 0 {
		conn, err := net.Dial("tcp", fmt.Sprintf(":%d", flags.Port))
		if err != nil {
			return nil, err
		}
		return jsonrpc2.NewStream(conn), nil
	}
	return jsonrpc2.NewStream(
		ioext.CompositeReadWriteCloser(
			container.Stdin(),
			container.Stdout(),
			ioext.NopCloser,
		),
	), nil
}

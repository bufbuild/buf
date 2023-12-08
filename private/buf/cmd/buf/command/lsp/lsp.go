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

package lsp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/buflsp"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

const (
	disableSymlinksFlagName = "disable-symlinks"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name,
		Short: "Start the language server.",
		Args:  cobra.MaximumNArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	DisableSymlinks bool
	Port            uint32
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindDisableSymlinks(flagSet, &f.DisableSymlinks, disableSymlinksFlagName)
	flagSet.Uint32Var(&f.Port, "port", 0, "port to listen on")
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	var jstream jsonrpc2.Stream
	if flags.Port > 0 {
		conn, err := net.Dial("tcp", fmt.Sprintf(":%d", flags.Port))
		if err != nil {
			return err
		}
		jstream = jsonrpc2.NewStream(conn)
	} else {
		jstream = jsonrpc2.NewStream(&readWriteCloser{
			reader: container.Stdin(),
			writer: container.Stdout(),
		})
	}
	jconn := jsonrpc2.NewConn(jstream)
	controller, err := bufcli.NewController(container)
	if err != nil {
		return err
	}
	server, err := buflsp.NewBufLsp(
		ctx,
		jconn,
		container.Logger(),
		container,
		controller,
	)
	if err != nil {
		return err
	}
	jconn.Go(ctx, protocol.ServerHandler(
		server,
		nil,
	))
	<-ctx.Done()
	return nil
}

type readWriteCloser struct {
	reader io.Reader
	writer io.Writer
}

func (r *readWriteCloser) Read(b []byte) (int, error) {
	return r.reader.Read(b)
}

func (r *readWriteCloser) Write(b []byte) (int, error) {
	return r.writer.Write(b)
}

func (r *readWriteCloser) Close() error {
	var errs []error
	if closer, ok := r.writer.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if closer, ok := r.reader.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

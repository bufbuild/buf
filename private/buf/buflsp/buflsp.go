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

// Package buflsp implements a language server for Protobuf.
//
// The main entry-point of this package is the New() function, which creates a new LSP server.
package buflsp

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/gen/data/datawkt"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.uber.org/zap"
)

// ErrNotInit is returned by LSP server methods that are called without first
// initializing the server.
var ErrNotInit = fmt.Errorf("the first call to the server must be the %q method", protocol.MethodInitialize)

// New constructs a new LSP server, ready to begin listening.
func New(
	ctx context.Context,
	container appext.Container,
	conn jsonrpc2.Conn,
) (protocol.Server, error) {
	controller, err := bufcli.NewController(container)
	if err != nil {
		return nil, err
	}

	server := &server{
		conn:   conn,
		client: protocol.ClientDispatcher(conn, container.Logger()),

		logger:     container.Logger(),
		tracer:     tracing.NewTracer(container.Tracer()),
		controller: controller,
	}
	server.files = newFiles(ctx, server)
	off := protocol.TraceOff
	server.traceValue.Store(&off)

	return server, nil
}

// server is an LSP server.
//
// This type contains all the necessary book-keeping for keeping the server running.
// Its handler methods are not defined in buflsp.go; they are defined in other files, grouped
// according to the groupings in https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification
type server struct {
	nyiServer

	conn   jsonrpc2.Conn
	client protocol.Client

	logger     *zap.Logger
	tracer     tracing.Tracer
	controller bufctl.Controller
	files      *files

	wktModuleSet bufmodule.ModuleSet

	// These are atomics, because they are read often add written to
	// almost never, but potentially concurrently. Having them side-by-side
	// is fine; they are almost never written to so false sharing is not a
	// concern.
	initParams atomic.Pointer[protocol.InitializeParams]
	traceValue atomic.Pointer[protocol.TraceValue]
}

// checkInit is a helper that checks if initialization has occurred and, if not,
// returns an appropriate error.
func (server *server) checkInit() error {
	if server.initParams.Load() != nil {
		return nil
	}
	return ErrNotInit
}

// init performs *actual* initialization of the server. This is called by Initialize().
//
// It may only be called once for a given server.
func (server *server) init(ctx context.Context, params *protocol.InitializeParams) error {
	if server.initParams.Load() != nil {
		return fmt.Errorf("called the %q method more than once", protocol.MethodInitialize)
	}
	server.initParams.Store(params)

	// TODO: set up logging. We need to forward everything from server.logger through to
	// the client, if tracing is turned on. The right way to do this is with an extra
	// goroutine and some channels.

	// Load the WKTs asap. They're always needed and don't need to hit the filesystem.
	if err := server.loadWTKs(ctx); err != nil {
		return err
	}

	return nil
}

// loadWKTs loads a ModuleSet for the well-known types.
func (server *server) loadWTKs(ctx context.Context) (err error) {
	builder := bufmodule.NewModuleSetBuilder(
		ctx,
		server.tracer,
		bufmodule.NopModuleDataProvider,
		bufmodule.NopCommitProvider,
	)
	// DO NOT MERGE: is isTarget necessary?
	builder.AddLocalModule(datawkt.ReadBucket, "." /*isTarget=*/, true)
	server.wktModuleSet, err = builder.Build()
	return
}

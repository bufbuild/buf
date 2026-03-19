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

// Package buflsp implements a language server for Protobuf.
//
// The main entry-point of this package is the Serve() function, which creates a new LSP server.
package buflsp

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime/debug"
	"sync"
	"sync/atomic"

	"buf.build/go/app/appext"
	"buf.build/go/standard/xlog/xslog"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/pkg/jsonrpc2"
	"github.com/bufbuild/buf/private/pkg/lspprotocol"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/wasm"
	"github.com/bufbuild/protocompile/experimental/incremental"
	"github.com/bufbuild/protocompile/experimental/ir"
	"github.com/bufbuild/protocompile/experimental/source"
)

// Serve spawns a new LSP server, listening on the given stream.
//
// Returns the connection for managing the server.
func Serve(
	ctx context.Context,
	bufVersion string,
	wktBucket storage.ReadBucket,
	container appext.Container,
	controller bufctl.Controller,
	wasmRuntime wasm.Runtime,
	rwc io.ReadWriteCloser,
	queryExecutor *incremental.Executor,
) (*jsonrpc2.Conn, error) {
	// Prefer build info version if available
	if buildInfo, ok := debug.ReadBuildInfo(); ok && buildInfo.Main.Version != "" {
		bufVersion = buildInfo.Main.Version
	}

	logger := container.Logger()
	logger = logger.With(slog.String("buf_version", bufVersion))
	logger.Info("starting LSP server")

	// connCtx is a context scoped to the connection's lifetime. It is cancelled
	// when the connection is done (or when ctx is cancelled), so that background
	// goroutines (e.g. RunChecks) do not outlive the connection.
	connCtx, connCancel := context.WithCancel(context.Background())

	lspState := &lsp{
		container:     container,
		logger:        logger,
		bufVersion:    bufVersion,
		controller:    controller,
		wasmRuntime:   wasmRuntime,
		wktBucket:     wktBucket,
		queryExecutor: queryExecutor,
		opener:        source.NewMap(nil),
		irSession:     new(ir.Session),
		connCtx:       connCtx,
		connCancel:    connCancel,
	}
	lspState.fileManager = newFileManager(lspState)
	lspState.workspaceManager = newWorkspaceManager(lspState)
	off := lspprotocol.TraceOff
	lspState.traceValue.Store(&off)

	handler, err := lspState.newHandler()
	if err != nil {
		connCancel()
		return nil, err
	}

	conn := jsonrpc2.NewConn(ctx, rwc, jsonrpc2.GoHandler(handler))
	// Store the conn atomically so Exit() can call Close() safely even though
	// NewConn already started the read loop.
	lspState.conn.Store(conn)
	lspState.client = lspprotocol.ClientDispatcher(&connWrapper{conn: conn, logger: logger})

	go func() {
		select {
		case <-ctx.Done():
		case <-conn.Done():
		}
		connCancel()
	}()

	return conn, nil
}

// *** PRIVATE ***

// lsp contains all of the LSP server's state. (I.e., it is the "god class" the protocol requires
// that we implement).
//
// This type does not implement lspprotocol.Server; see server.go for that.
// This type contains all the necessary book-keeping for keeping the server running.
// Its handler methods are not defined in buflsp.go; they are defined in other files, grouped
// according to the groupings in the LSP specification.
type lsp struct {
	// conn is stored atomically because NewConn starts the read loop immediately,
	// creating a tiny window before we assign conn. The LSP protocol guarantees
	// initialize is the first request, so this is safe in practice, but we use
	// atomic storage to satisfy the race detector.
	conn   atomic.Pointer[jsonrpc2.Conn]
	client lspprotocol.Client

	container  appext.Container
	connCtx    context.Context    // cancelled when the connection is done
	connCancel context.CancelFunc // cancels connCtx

	logger           *slog.Logger
	bufVersion       string // buf version, set at server creation
	controller       bufctl.Controller
	wasmRuntime      wasm.Runtime
	fileManager      *fileManager
	workspaceManager *workspaceManager
	queryExecutor    *incremental.Executor
	opener           source.Map
	irSession        *ir.Session
	wktBucket        storage.ReadBucket
	shutdown         bool

	lock sync.Mutex

	// These are atomics, because they are read often and written to
	// almost never, but potentially concurrently. Having them side-by-side
	// is fine; they are almost never written to so false sharing is not a
	// concern.
	initParams atomic.Pointer[lspprotocol.InitializeParams]
	traceValue atomic.Pointer[lspprotocol.TraceValue]
}

// init performs *actual* initialization of the server. This is called by Initialize().
//
// It may only be called once for a given server.
func (l *lsp) init(_ context.Context, params *lspprotocol.InitializeParams) error {
	if l.initParams.Load() != nil {
		return fmt.Errorf("called the %q method more than once", lspprotocol.MethodInitialize)
	}
	l.initParams.Store(params)

	// TODO: set up logging. We need to forward everything from server.logger through to
	// the client, if tracing is turned on. The right way to do this is with an extra
	// goroutine and some channels.

	return nil
}

// newHandler constructs an RPC handler that wraps the default one from jsonrpc2. This allows us
// to inject debug logging, tracing, and timeouts to requests.
func (l *lsp) newHandler() (jsonrpc2.Handler, error) {
	server, err := newServer(l)
	if err != nil {
		return nil, err
	}
	// CancelHandler intercepts $/cancelRequest notifications from the client and
	// cancels the context of the matching in-flight request. GoHandler (applied in
	// Serve) ensures each request runs in its own goroutine so the cancellable
	// context is the one running inside the spawned goroutine.
	return jsonrpc2.CancelHandler(jsonrpc2.HandlerFunc(func(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (any, error) {
		l.logger.Debug(
			"handling request",
			slog.String("method", req.Method),
		)
		defer xslog.DebugProfile(
			l.logger,
			slog.String("method", req.Method),
		)()

		var result any
		var err error
		if req.Method != lspprotocol.MethodInitialize && l.initParams.Load() == nil {
			// Verify that the server has been initialized if this isn't the initialization
			// request.
			err = fmt.Errorf("the first call to the server must be the %q method", lspprotocol.MethodInitialize)
		} else {
			l.lock.Lock()
			result, err = lspprotocol.ServerDispatch(ctx, server, req)
			l.lock.Unlock()
		}

		if err != nil {
			l.logger.Warn(
				"responding with error",
				slog.String("method", req.Method),
				xslog.ErrorAttr(err),
			)
		} else {
			l.logger.Debug(
				"responding",
				slog.String("method", req.Method),
			)
		}
		return result, err
	})), nil
}

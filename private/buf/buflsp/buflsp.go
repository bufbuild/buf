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
// The main entry-point of this package is the Serve() function, which creates a new LSP server.
package buflsp

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/slogext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.uber.org/zap"
)

// Serve spawns a new LSP server, listening on the given stream.
//
// Returns a context for managing the server.
func Serve(
	ctx context.Context,
	container appext.Container,
	controller bufctl.Controller,
	checkClient bufcheck.Client,
	stream jsonrpc2.Stream,
) (jsonrpc2.Conn, error) {
	// The LSP protocol deals with absolute filesystem paths. This requires us to
	// bypass the bucket API completely, so we create a bucket pointing at the filesystem
	// root.
	bucketProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	bucket, err := bucketProvider.NewReadWriteBucket(
		"/", // TODO: This is not correct for Windows.
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return nil, err
	}

	conn := jsonrpc2.NewConn(stream)
	lsp := &lsp{
		conn: conn,
		client: protocol.ClientDispatcher(
			&connWrapper{Conn: conn, logger: container.Logger()},
			zap.NewNop(), // The logging from protocol itself isn't very good, we've replaced it with connAdapter here.
		),
		logger:      container.Logger(),
		controller:  controller,
		checkClient: checkClient,
		rootBucket:  bucket,
	}
	lsp.fileManager = newFileManager(lsp)
	off := protocol.TraceOff
	lsp.traceValue.Store(&off)

	conn.Go(ctx, lsp.newHandler())
	return conn, nil
}

// *** PRIVATE ***

// lsp contains all of the LSP server's state. (I.e., it is the "god class" the protocol requires
// that we implement).
//
// This type does not implement protocol.Server; see server.go for that.
// This type contains all the necessary book-keeping for keeping the server running.
// Its handler methods are not defined in buflsp.go; they are defined in other files, grouped
// according to the groupings in
type lsp struct {
	conn   jsonrpc2.Conn
	client protocol.Client

	logger      *slog.Logger
	controller  bufctl.Controller
	checkClient bufcheck.Client
	rootBucket  storage.ReadBucket
	fileManager *fileManager

	// These are atomics, because they are read often and written to
	// almost never, but potentially concurrently. Having them side-by-side
	// is fine; they are almost never written to so false sharing is not a
	// concern.
	initParams atomic.Pointer[protocol.InitializeParams]
	traceValue atomic.Pointer[protocol.TraceValue]
}

// init performs *actual* initialization of the server. This is called by Initialize().
//
// It may only be called once for a given server.
func (l *lsp) init(params *protocol.InitializeParams) error {
	if l.initParams.Load() != nil {
		return fmt.Errorf("called the %q method more than once", protocol.MethodInitialize)
	}
	l.initParams.Store(params)

	// TODO: set up logging. We need to forward everything from server.logger through to
	// the client, if tracing is turned on. The right way to do this is with an extra
	// goroutine and some channels.

	return nil
}

// findImportable finds all files that can potentially be imported by the proto file at
// uri. This returns a map from potential Protobuf import path to the URI of the file it would import.
//
// Note that this performs no validation on these files, because those files might be open in the
// editor and might contain invalid syntax at the moment. We only want to get their paths and nothing
// more.
func (l *lsp) findImportable(
	ctx context.Context,
	uri protocol.URI,
) (map[string]bufimage.ImageFileInfo, error) {
	fileInfos, err := l.controller.GetImportableImageFileInfos(ctx, uri.Filename())
	if err != nil {
		return nil, err
	}

	imports := make(map[string]bufimage.ImageFileInfo)
	for _, fileInfo := range fileInfos {
		imports[fileInfo.Path()] = fileInfo
	}

	l.logger.DebugContext(ctx, fmt.Sprintf("found imports for %q: %#v", uri, imports))

	return imports, nil
}

// newHandler constructs an RPC handler that wraps the default one from jsonrpc2. This allows us
// to inject debug logging, tracing, and timeouts to requests.
func (l *lsp) newHandler() jsonrpc2.Handler {
	actual := protocol.ServerHandler(newServer(l), nil)
	return func(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) (retErr error) {
		defer slogext.DebugProfile(l.logger, slog.String("method", req.Method()), slog.Any("params", req.Params()))()

		ctx = withRequestID(ctx)

		replier := l.wrapReplier(reply, req)

		// Verify that the server has been initialized if this isn't the initialization
		// request.
		if req.Method() != protocol.MethodInitialize && l.initParams.Load() == nil {
			return replier(ctx, nil, fmt.Errorf("the first call to the server must be the %q method", protocol.MethodInitialize))
		}

		return actual(ctx, replier, req)
	}
}

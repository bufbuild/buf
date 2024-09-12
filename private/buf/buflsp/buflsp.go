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
	"sync/atomic"
	"time"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

const (
	// TODO: make configurable
	handlerTimeout = 3 * time.Second
)

// Serve spawns a new LSP server, listening on the given stream.
//
// Returns a context for managing the server.
func Serve(
	ctx context.Context,
	container appext.Container,
	stream jsonrpc2.Stream,
) (jsonrpc2.Conn, error) {
	controller, err := bufcli.NewController(container)
	if err != nil {
		return nil, err
	}

	moduleDataProvider, err := bufcli.NewModuleDataProvider(container)
	if err != nil {
		return nil, err
	}

	// The LSP protocol deals with absolute filesystem paths. This requires us to
	// bypass the bucket API completely, so we create a bucket pointing at the filesystem
	// root.
	bucketProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	bucket, err := bucketProvider.NewReadWriteBucket(
		"/", // This is not correct for Windows.
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return nil, err
	}

	container.CacheDirPath()

	conn := jsonrpc2.NewConn(stream)
	lsp := &lsp{
		conn: conn,
		client: protocol.ClientDispatcher(
			&connAdapter{Conn: conn, logger: container.Logger()},
			zap.NewNop(), // The logging from protocol itself isn't very good, we've replaced it with connAdapter here.
		),
		logger:             container.Logger(),
		tracer:             tracing.NewTracer(container.Tracer()),
		controller:         controller,
		rootBucket:         bucket,
		moduleDataProvider: moduleDataProvider,
	}
	lsp.fileManager = newFiles(lsp)
	off := protocol.TraceOff
	lsp.traceValue.Store(&off)

	conn.Go(ctx, lsp.newHandler())
	return conn, nil
}

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

	logger             *zap.Logger
	tracer             tracing.Tracer
	controller         bufctl.Controller
	rootBucket         storage.ReadBucket
	moduleDataProvider bufmodule.ModuleDataProvider
	fileManager        *fileManager

	// These are atomics, because they are read often add written to
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

	l.logger.Sugar().Debugf("found imports for %q: %#v", uri, imports)

	return imports, nil
}

// newHandler constructs an RPC handler that wraps the default one from jsonrpc2. This allows us
// to inject debug logging, tracing, and timeouts to requests.
func (l *lsp) newHandler() jsonrpc2.Handler {
	actual := protocol.ServerHandler(newServer(l), nil)
	return func(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) (retErr error) {
		ctx, span := l.tracer.Start(
			ctx,
			tracing.WithErr(&retErr),
			tracing.WithAttributes(attribute.String("method", req.Method())),
		)
		defer span.End()

		l.logger.Debug(
			"processing request",
			zap.String("method", req.Method()),
			zap.ByteString("params", req.Params()),
		)

		// Each request is handled in a separate goroutine, and has a fixed timeout.
		// This is to enforce responsiveness on the LSP side: if something is going to take
		// a long time, it should be offloaded.
		ctx, done := context.WithTimeout(ctx, handlerTimeout)
		ctx = withRequestID(ctx)

		go func() {
			defer done()
			replier := l.adaptReplier(reply, req)

			// Verify that the server has been initialized if this isn't the initialization
			// request.
			if req.Method() != protocol.MethodInitialize && l.initParams.Load() == nil {
				retErr = replier(ctx, nil, fmt.Errorf("the first call to the server must be the %q method",
					protocol.MethodInitialize))
				return
			}

			retErr = actual(ctx, replier, req)
		}()

		<-ctx.Done()
		if ctx.Err() == context.DeadlineExceeded {
			// Don't return this error; that will kill the whole server!
			l.logger.Sugar().Errorf("timed out while handling %s; this is likely a bug", req.Method())
		}
		return retErr
	}
}

// adaptReplier wraps a jsonrpc2.Replier, allowing us to inject logging and tracing and so on.
func (l *lsp) adaptReplier(reply jsonrpc2.Replier, req jsonrpc2.Request) jsonrpc2.Replier {
	return func(ctx context.Context, result any, err error) error {
		if err != nil {
			l.logger.Warn(
				"responding with error",
				zap.String("method", req.Method()),
				zap.Error(err),
			)
		} else {
			l.logger.Debug(
				"responding",
				zap.String("method", req.Method()),
				zap.Reflect("params", result),
			)
		}

		return reply(ctx, result, err)
	}
}

// connAdapter wraps a connection and logs calls and notifications.
//
// By default, the ClientDispatcher does not log the bodies of requests and responses, making
// for much lower-quality debugging.
type connAdapter struct {
	jsonrpc2.Conn

	logger *zap.Logger
}

func (c *connAdapter) Call(
	ctx context.Context, method string, params, result any) (id jsonrpc2.ID, err error) {
	c.logger.Debug(
		"call",
		zap.String("method", method),
		zap.Reflect("params", params),
	)

	id, err = c.Conn.Call(ctx, method, params, result)
	if err != nil {
		c.logger.Warn(
			"call returned error",
			zap.String("method", method),
			zap.Error(err),
		)
	} else {
		c.logger.Warn(
			"call returned",
			zap.String("method", method),
			zap.Reflect("result", result),
		)
	}

	return
}

func (c *connAdapter) Notify(
	ctx context.Context, method string, params any) error {
	c.logger.Debug(
		"notify",
		zap.String("method", method),
		zap.Reflect("params", params),
	)

	err := c.Conn.Notify(ctx, method, params)
	if err != nil {
		c.logger.Warn(
			"notify returned error",
			zap.String("method", method),
			zap.Error(err),
		)
	} else {
		c.logger.Warn(
			"notify returned",
			zap.String("method", method),
		)
	}

	return err
}

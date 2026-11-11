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

package buflsp

import (
	"context"
	"log/slog"
	"slices"

	"buf.build/go/standard/xlog/xslog"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

// lspCodec is the [jsonrpc2.Codec] for LSP payloads.
//
// It uses [protocol.Marshal] and [protocol.Unmarshal], which understand the
// protocol package's union types. This matches the codec that
// [protocol.NewServer] installs, which is not exported; we can't use
// [protocol.NewServer] because we wrap the handler it would construct.
type lspCodec struct{}

var _ jsonrpc2.Codec = lspCodec{}

func (lspCodec) Marshal(value any) ([]byte, error) {
	switch message := value.(type) {
	case jsonrpc2.RawMessage:
		if message == nil {
			return []byte("null"), nil
		}
		return message, nil
	case *jsonrpc2.RawMessage:
		if message == nil || *message == nil {
			return []byte("null"), nil
		}
		return *message, nil
	}
	return protocol.Marshal(value)
}

func (lspCodec) Unmarshal(data []byte, value any) error {
	if message, ok := value.(*jsonrpc2.RawMessage); ok {
		*message = slices.Clone(jsonrpc2.RawMessage(data))
		return nil
	}
	return protocol.Unmarshal(data, value)
}

// orderedAsyncHandler wraps handler so that messages are handled one at a time,
// in the order they are received, without blocking the connection's read loop.
//
// LSP requires that messages that change state, like textDocument/didChange,
// are handled before the messages that follow them. [jsonrpc2.AsyncHandler]
// handles messages concurrently and in no particular order. Because the read
// loop is not blocked, notifications like $/cancelRequest that are handled
// before reaching this handler are still processed immediately.
//
// The returned handler must only be called from the connection's read loop.
func orderedAsyncHandler(handler jsonrpc2.Handler) jsonrpc2.Handler {
	previousDone := make(chan struct{})
	close(previousDone)
	return func(ctx context.Context, request *jsonrpc2.Request) (any, error) {
		// This runs on the read loop, so messages enter the queue in the order
		// they are received.
		waitForPrevious := previousDone
		done := make(chan struct{})
		previousDone = done
		defer close(done)

		jsonrpc2.Async(ctx)
		<-waitForPrevious
		return handler(ctx, request)
	}
}

// logResponse logs the outcome of handling a request, allowing us to inject
// logging and tracing and so on.
func (l *lsp) logResponse(method string, err error) {
	if err != nil {
		l.logger.Warn(
			"responding with error",
			slog.String("method", method),
			xslog.ErrorAttr(err),
		)
	} else {
		l.logger.Debug(
			"responding",
			slog.String("method", method),
		)
	}
}

// connWrapper wraps a connection and logs calls and notifications.
//
// By default, the ClientDispatcher does not log the bodies of requests and responses, making
// for much lower-quality debugging.
type connWrapper struct {
	jsonrpc2.Conn

	logger *slog.Logger
}

func (c *connWrapper) Call(
	ctx context.Context, method string, params, result any) (id jsonrpc2.ID, err error) {
	c.logger.Debug(
		"call",
		slog.String("method", method),
	)

	id, err = c.Conn.Call(ctx, method, params, result)
	if err != nil {
		c.logger.Warn(
			"call returned error",
			slog.String("method", method),
			xslog.ErrorAttr(err),
		)
	} else {
		c.logger.Debug(
			"call returned",
			slog.String("method", method),
		)
	}

	return
}

func (c *connWrapper) Notify(
	ctx context.Context, method string, params any) error {
	c.logger.Debug(
		"notify",
		slog.String("method", method),
	)

	err := c.Conn.Notify(ctx, method, params)
	if err != nil {
		c.logger.Warn(
			"notify returned error",
			slog.String("method", method),
			xslog.ErrorAttr(err),
		)
	} else {
		c.logger.Debug(
			"notify returned",
			slog.String("method", method),
		)
	}

	return err
}

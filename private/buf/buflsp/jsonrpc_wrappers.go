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

	"go.lsp.dev/jsonrpc2"
	"go.uber.org/zap"
)

// wrapReplier wraps a jsonrpc2.Replier, allowing us to inject logging and tracing and so on.
func (l *lsp) wrapReplier(reply jsonrpc2.Replier, req jsonrpc2.Request) jsonrpc2.Replier {
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

// connWrapper wraps a connection and logs calls and notifications.
//
// By default, the ClientDispatcher does not log the bodies of requests and responses, making
// for much lower-quality debugging.
type connWrapper struct {
	jsonrpc2.Conn

	logger *zap.Logger
}

func (c *connWrapper) Call(
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

func (c *connWrapper) Notify(
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

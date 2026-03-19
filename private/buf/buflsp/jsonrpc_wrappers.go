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

	"buf.build/go/standard/xlog/xslog"
	"github.com/bufbuild/buf/private/pkg/jsonrpc2"
)

// connWrapper wraps a *jsonrpc2.Conn, adding logging around Call and Notify.
// It implements the jsonrpc2.Caller interface used by lspprotocol.ClientDispatcher.
type connWrapper struct {
	conn   *jsonrpc2.Conn
	logger *slog.Logger
}

func (c *connWrapper) Call(
	ctx context.Context, method string, params, result any) error {
	c.logger.Debug(
		"call",
		slog.String("method", method),
	)

	err := c.conn.Call(ctx, method, params, result)
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

	return err
}

func (c *connWrapper) Notify(
	ctx context.Context, method string, params any) error {
	c.logger.Debug(
		"notify",
		slog.String("method", method),
	)

	err := c.conn.Notify(ctx, method, params)
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

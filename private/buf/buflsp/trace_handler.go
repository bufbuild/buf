// Copyright 2020-2025 Buf Technologies, Inc.
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
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

// traceHandler is a [slog.Handler] that forwards log messages to the LSP client
// via $/logTrace when tracing is enabled.
type traceHandler struct {
	base       slog.Handler
	conn       jsonrpc2.Conn
	traceValue *atomic.Pointer[protocol.TraceValue]
}

// newTraceHandler creates a new trace handler that wraps the base handler.
func newTraceHandler(base slog.Handler, conn jsonrpc2.Conn, traceValue *atomic.Pointer[protocol.TraceValue]) *traceHandler {
	return &traceHandler{
		base:       base,
		conn:       conn,
		traceValue: traceValue,
	}
}

// Enabled implements [slog.Handler].
func (h *traceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.base.Enabled(ctx, level)
}

// Handle implements [slog.Handler].
func (h *traceHandler) Handle(ctx context.Context, r slog.Record) error {
	// TODO: do we even want to have a base here?
	if err := h.base.Handle(ctx, r); err != nil {
		return fmt.Errorf("handling base record: %w", err)
	}

	// Check if tracing is enabled
	trace := *h.traceValue.Load()
	if trace == protocol.TraceOff {
		return nil
	}

	// Filter based on trace level and log level
	// TraceMessage: forward Info and above
	// TraceVerbose: forward Debug and above
	if trace == protocol.TraceMessage && r.Level < slog.LevelInfo {
		return nil
	}

	// Build the log message
	var buf strings.Builder
	buf.WriteString(r.Message)

	// Add attributes
	if r.NumAttrs() > 0 {
		buf.WriteString(" ")
		first := true
		r.Attrs(func(a slog.Attr) bool {
			if !first {
				buf.WriteString(" ")
			}
			first = false
			buf.WriteString(a.Key)
			buf.WriteString("=")
			buf.WriteString(a.Value.String())
			return true
		})
	}

	params := &protocol.LogTraceParams{
		Message: buf.String(),
	}
	if trace == protocol.TraceVerbose {
		// TODO: I don't think the type of this field is correct upstream.
		// Ref: https://github.com/go-language-server/protocol/issues/57
		params.Verbose = protocol.TraceVerbose
	}

	return h.conn.Notify(ctx, protocol.MethodLogTrace, params)
}

// WithAttrs implements [slog.Handler].
func (h *traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &traceHandler{
		base:       h.base.WithAttrs(attrs),
		conn:       h.conn,
		traceValue: h.traceValue,
	}
}

// WithGroup implements [slog.Handler].
func (h *traceHandler) WithGroup(name string) slog.Handler {
	return &traceHandler{
		base:       h.base.WithGroup(name),
		conn:       h.conn,
		traceValue: h.traceValue,
	}
}

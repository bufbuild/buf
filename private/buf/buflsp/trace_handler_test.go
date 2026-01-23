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

//go:build go1.25

package buflsp

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

// mockConn is a mock implementation of jsonrpc2.Conn for testing.
type mockConn struct {
	jsonrpc2.Conn
	messages []protocol.LogTraceParams
	mu       sync.Mutex
}

func (m *mockConn) Notify(_ context.Context, method string, params any) error {
	if method == protocol.MethodLogTrace {
		m.mu.Lock()
		defer m.mu.Unlock()
		if p, ok := params.(*protocol.LogTraceParams); ok {
			m.messages = append(m.messages, *p)
		}
	}
	return nil
}

func (m *mockConn) getMessages() []protocol.LogTraceParams {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]protocol.LogTraceParams{}, m.messages...)
}

func TestTraceHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		traceValue     protocol.TraceValue
		logActions     func(*slog.Logger)
		expectedCount  int
		expectedMsgs   []string
		checkVerbose   bool
		verboseExpect  protocol.TraceValue
	}{
		{
			name:       "trace off sends no messages",
			traceValue: protocol.TraceOff,
			logActions: func(l *slog.Logger) {
				l.Info("info message")
				l.Debug("debug message")
			},
			expectedCount: 0,
		},
		{
			name:       "trace messages forwards info and above",
			traceValue: protocol.TraceMessage,
			logActions: func(l *slog.Logger) {
				l.Info("info message")
				l.Debug("debug message") // Should be filtered out
			},
			expectedCount: 1,
			expectedMsgs:  []string{"info message"},
		},
		{
			name:       "trace verbose forwards debug and above",
			traceValue: protocol.TraceVerbose,
			logActions: func(l *slog.Logger) {
				l.Info("info message")
				l.Debug("debug message")
			},
			expectedCount: 2,
			expectedMsgs:  []string{"info message", "debug message"},
			checkVerbose:  true,
			verboseExpect: protocol.TraceVerbose,
		},
		{
			name:       "all log levels in verbose mode",
			traceValue: protocol.TraceVerbose,
			logActions: func(l *slog.Logger) {
				l.Error("error message")
				l.Warn("warn message")
				l.Info("info message")
				l.Debug("debug message")
			},
			expectedCount: 4,
			expectedMsgs:  []string{"error message", "warn message", "info message", "debug message"},
		},
		{
			name:       "attributes are included",
			traceValue: protocol.TraceVerbose,
			logActions: func(l *slog.Logger) {
				l.Info("test message", "key", "value", "number", 42)
			},
			expectedCount: 1,
			expectedMsgs:  []string{"test message key=value number=42"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				conn := &mockConn{}
				traceValue := atomic.Pointer[protocol.TraceValue]{}
				traceValue.Store(&tt.traceValue)

				// Use a handler that accepts all log levels
				baseHandler := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
				handler := newTraceHandler(baseHandler, conn, &traceValue)
				logger := slog.New(handler)

				// Execute log actions
				tt.logActions(logger)

				// Wait for async goroutines to complete
				synctest.Wait()

				// Check message count
				messages := conn.getMessages()
				require.Len(t, messages, tt.expectedCount)

				// Check expected messages (if any)
				if len(tt.expectedMsgs) > 0 {
					// Build a set of actual messages
					actualMsgs := make(map[string]bool)
					for _, msg := range messages {
						actualMsgs[msg.Message] = true
					}

					// Verify all expected messages are present
					for _, expected := range tt.expectedMsgs {
						assert.True(t, actualMsgs[expected], "expected message %q not found", expected)
					}
				}

				// Check verbose field if requested
				if tt.checkVerbose {
					for _, msg := range messages {
						assert.Equal(t, tt.verboseExpect, msg.Verbose, "unexpected Verbose field value")
					}
				}
			})
		})
	}
}

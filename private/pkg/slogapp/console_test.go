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

package slogapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsWriterTTY(t *testing.T) {
	t.Parallel()

	// A bytes.Buffer has no Fd method, so it's not a TTY.
	var buf bytes.Buffer
	assert.False(t, isWriterTTY(&buf))

	// An os.Pipe is never a TTY even though it has an Fd method.
	_, w, err := os.Pipe()
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })
	assert.False(t, isWriterTTY(w))
}

func TestConsoleLogOutput(t *testing.T) {
	t.Parallel()

	testConsolLogOutput(t, func(logger *slog.Logger) {
		logger.Info("hello", slog.String("a", "b"))
		logger.Info("hello world")
	}, []map[string]any{{
		slog.LevelKey:   colorize("INFO", getColor(slog.LevelInfo)),
		slog.MessageKey: "hello",
		"a":             "b",
	}, {
		slog.LevelKey:   colorize("INFO", getColor(slog.LevelInfo)),
		slog.MessageKey: "hello world",
	}}, withConsoleColor(true))

	testConsolLogOutput(t, func(logger *slog.Logger) {
		logger.Info("info", slog.String("a", "b"))
		logger.Error("error")
	}, []map[string]any{{
		slog.LevelKey:   "INFO",
		slog.MessageKey: "info",
		"a":             "b",
	}, {
		slog.LevelKey:   "ERROR",
		slog.MessageKey: "error",
	}})

	testConsolLogOutput(t, func(logger *slog.Logger) {
		logger = logger.With(slog.String("a", "b"))
		logger = logger.WithGroup("g")
		logger.Error("error message", slog.String("c", "d"))
		logger.Info("info message")
		logger.Debug("debuf message", slog.String("c", "d"))
	}, []map[string]any{{
		slog.LevelKey:   colorize("ERROR", getColor(slog.LevelError)),
		slog.MessageKey: "error message",
		"a":             "b",
		"g": map[string]any{
			"c": "d",
		},
	}, {
		slog.LevelKey:   colorize("INFO", getColor(slog.LevelInfo)),
		slog.MessageKey: "info message",
		"a":             "b",
	}}, withConsoleColor(true))

	testConsolLogOutput(t, func(logger *slog.Logger) {
		logger.Info("key spaces", slog.String("a key", "with spaces"))
	}, []map[string]any{{
		slog.LevelKey:   colorize("INFO", getColor(slog.LevelInfo)),
		slog.MessageKey: "key spaces",
		"a key":         "with spaces",
	}}, withConsoleColor(true))
}

func testConsolLogOutput(t *testing.T, run func(logger *slog.Logger), expects []map[string]any, options ...consoleHandlerOption) {
	t.Helper()
	var buf bytes.Buffer
	consoleHandler := newConsoleHandler(&buf, slog.LevelInfo, options...)
	logger := slog.New(consoleHandler)
	run(logger)

	var outputs []map[string]any
	for _, line := range bytes.Split(buf.Bytes(), []byte{'\n'}) {
		if len(line) == 0 {
			continue
		}
		lineAttrs, err := testParseLogLine(line)
		if !assert.NoError(t, err) {
			continue
		}
		outputs = append(outputs, lineAttrs)
	}
	t.Log(buf.String())
	require.Equal(t, len(expects), len(outputs))
	for i := range len(outputs) {
		output, expect := outputs[i], expects[i]
		assert.Equal(t, expect, output)
	}
}

// testParseLogLine passes the output of a single log line.
func testParseLogLine(lineBytes []byte) (map[string]any, error) {
	top := map[string]any{}
	line := string(bytes.TrimSpace(lineBytes))
	index, line, _ := strings.Cut(line, consoleSeparator)
	top[slog.LevelKey] = index
	if len(line) == 0 {
		return top, nil
	}
	message, line := line, ""
	// Find the JSON attributes by looking for the first space followed by a '{'.
	// This may fail for complex messages but fine for testing.
	if jsonIndex := strings.Index(message, consoleSeparator+"{"); jsonIndex >= 0 {
		message, line = message[:jsonIndex], message[jsonIndex+1:]
	}
	top[slog.MessageKey] = message
	if len(line) > 0 {
		// Capture the JSON attributes.
		var attrs map[string]any
		if err := json.Unmarshal([]byte(line), &attrs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON attrs: %w", err)
		}
		// Merge the JSON attributes into the top-level map.
		for key, value := range attrs {
			if _, ok := top[key]; ok {
				return nil, fmt.Errorf("duplicate key %q in JSON attributes", key)
			}
			top[key] = value
		}
	}
	return top, nil
}

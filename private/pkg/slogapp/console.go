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
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/mattn/go-colorable"
	"golang.org/x/term"
)

const (
	// color codes for ANSI escape sequences.
	colorBlack color = iota + 30
	colorRed
	colorGreen
	colorYellow
	colorBlue
	colorMagenta
	colorCyan
	colorWhite

	// consoleSeparator is the separator used in console output.
	consoleSeparator = "\t"
)

// color represents an ANSI color code.
type color uint8

type consoleHandlerOption func(*consoleHandlerOptions)

// withConsoleColor explicitly enables or disables color output, bypassing TTY detection.
// Without this option, color is enabled only if the writer is a TTY.
//
// If the environment variable NO_COLOR is set, colors will be disabled regardless of this setting.
func withConsoleColor(enable bool) consoleHandlerOption {
	return func(options *consoleHandlerOptions) {
		options.enableColor = enable
		options.colorExplicit = true
	}
}

type consoleHandlerOptions struct {
	enableColor   bool
	colorExplicit bool
}

func newConsoleHandlerOptions() *consoleHandlerOptions {
	return &consoleHandlerOptions{}
}

// consoleHandler is a custom slog.Handler that formats log messages for the console.
type consoleHandler struct {
	enableColor bool
	out         io.Writer
	lock        *sync.Mutex   // Lock protects access to the buffer.
	buffer      *bytes.Buffer // Buffer output for the delegate's writer.
	delegate    slog.Handler  // Delegate writes to buffer.
}

// newConsoleHandler creates a new consoleHandler with the specified output writer and log level.
//
// It pretty prints the level (optionally with color) and message with JSON encoded attributes.
// It wraps the output writer with colorable if it's os.Stdout or os.Stderr to support color output on Windows.
// It logs attributes formatted using the slog.JSONHandler as a delegate.
// It uses a mutex to synchronize access to the output. Not suitable for high-throughput logging.
func newConsoleHandler(out io.Writer, logLevel slog.Level, options ...consoleHandlerOption) *consoleHandler {
	consoleHandlerOptions := newConsoleHandlerOptions()
	for _, option := range options {
		option(consoleHandlerOptions)
	}
	// If color was not explicitly set, auto-detect based on whether the writer is a TTY.
	enableColor := consoleHandlerOptions.enableColor
	if !consoleHandlerOptions.colorExplicit {
		enableColor = isWriterTTY(out)
	}
	// NO_COLOR always wins.
	if os.Getenv("NO_COLOR") != "" {
		enableColor = false
	}
	// Wrap the output writer with colorable if it's os.Stdout or os.Stderr
	// to support color output on Windows.
	if enableColor && (out == os.Stderr || out == os.Stdout) {
		file, _ := out.(*os.File)
		out = colorable.NewColorable(file)
	}
	// A delegate handler is used to format the log attributes.
	// It uses a buffer to accumulate the log attributes before writing them to the output.
	// The buffer is protected by the lock.
	var (
		lock   sync.Mutex
		buffer bytes.Buffer
	)
	delegateHandler := slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
		Level:       logLevel,
		ReplaceAttr: consoleReplaceAttr,
	})
	return &consoleHandler{
		enableColor: enableColor,
		out:         out,
		lock:        &lock,
		buffer:      &buffer,
		delegate:    delegateHandler,
	}
}

// Enabled implements the slog.Handler interface.
func (c *consoleHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return c.delegate.Enabled(ctx, level)
}

// Handle implements the slog.Handler interface.
func (c *consoleHandler) Handle(ctx context.Context, r slog.Record) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.buffer.Reset()
	if !r.Time.IsZero() {
		c.buffer.WriteString(r.Time.Format(time.RFC3339))
		c.buffer.WriteString(consoleSeparator)
	}
	if c.enableColor {
		c.buffer.WriteString(colorize(r.Level.String(), getColor(r.Level)))
	} else {
		c.buffer.WriteString(r.Level.String())
	}
	c.buffer.WriteString(consoleSeparator)
	c.buffer.WriteString(r.Message)
	bufN := c.buffer.Len()
	c.buffer.WriteString(consoleSeparator)
	// Delegate must always be called, as it may have attributes to write.
	if err := c.delegate.Handle(ctx, r); err != nil {
		return err
	}
	if c.buffer.Len() == bufN+len(consoleSeparator+"{}\n") {
		// No attributes to write, trim the buffer to remove the empty JSON object.
		c.buffer.Truncate(bufN)
		c.buffer.WriteByte('\n')
	}
	_, err := c.buffer.WriteTo(c.out)
	return err
}

// WithAttrs implements the slog.Handler interface.
func (c *consoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return c.cloneWithDelegate(c.delegate.WithAttrs(attrs))
}

// WithGroup implements the slog.Handler interface.
func (c *consoleHandler) WithGroup(name string) slog.Handler {
	return c.cloneWithDelegate(c.delegate.WithGroup(name))
}

// cloneWithDelegate creates a new consoleHandler with a new delegate handler.
func (c *consoleHandler) cloneWithDelegate(delegate slog.Handler) *consoleHandler {
	return &consoleHandler{
		enableColor: c.enableColor,
		delegate:    delegate,
		out:         c.out,
		lock:        c.lock,
		buffer:      c.buffer,
	}
}

// getColor returns the color code for the specified log level.
func getColor(level slog.Level) color {
	switch {
	case level >= slog.LevelError:
		return colorRed
	case level >= slog.LevelWarn:
		return colorYellow
	case level >= slog.LevelInfo:
		return colorBlue
	case level >= slog.LevelDebug:
		return colorMagenta
	default:
		return 0
	}
}

// colorize formats the string with the specified color.
func colorize(s string, color color) string {
	if color == 0 {
		return s
	}
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", color, s)
}

// isWriterTTY checks if the given writer is connected to a TTY.
// It tries to extract a file descriptor from the writer and check if it's a terminal.
func isWriterTTY(w io.Writer) bool {
	type fdGetter interface {
		Fd() uintptr
	}
	if fdOut, ok := w.(fdGetter); ok {
		return term.IsTerminal(int(fdOut.Fd()))
	}
	// Can't determine, default to false (no TTY)
	return false
}

// consoleReplaceAttr is a custom ReplaceAttr function for consoleHandler.
// It silences the time, level, and message attributes to avoid duplication.
func consoleReplaceAttr(groups []string, a slog.Attr) slog.Attr {
	switch a.Key {
	case slog.TimeKey, slog.LevelKey, slog.MessageKey:
		return slog.Attr{}
	default:
		return defaultReplaceAttr(groups, a)
	}
}

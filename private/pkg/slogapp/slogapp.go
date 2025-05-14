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

// Package slogapp builds slog.Loggers.
package slogapp

import (
	"fmt"
	"io"
	"log/slog"

	"buf.build/go/app/appext"
)

// LoggerProvider is an appext.LoggerProvider for use with appext.BuilderWithLoggerProvider.
func LoggerProvider(container appext.NameContainer, logLevel appext.LogLevel, logFormat appext.LogFormat) (*slog.Logger, error) {
	return NewLogger(container.Stderr(), logLevel, logFormat)
}

// NewLogger returns a new Logger for the given level and format.
func NewLogger(writer io.Writer, logLevel appext.LogLevel, logFormat appext.LogFormat) (*slog.Logger, error) {
	handler, err := getHandler(writer, logLevel, logFormat)
	if err != nil {
		return nil, err
	}
	return slog.New(handler), nil
}

func getLevel(logLevel appext.LogLevel) (slog.Level, error) {
	switch logLevel {
	case appext.LogLevelDebug:
		return slog.LevelDebug, nil
	case appext.LogLevelInfo:
		return slog.LevelInfo, nil
	case appext.LogLevelWarn:
		return slog.LevelWarn, nil
	case appext.LogLevelError:
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid level %v", logLevel)
	}
}

func getHandler(writer io.Writer, logLevel appext.LogLevel, logFormat appext.LogFormat) (slog.Handler, error) {
	level, err := getLevel(logLevel)
	if err != nil {
		return nil, err
	}
	switch logFormat {
	case appext.LogFormatJSON:
		return slog.NewJSONHandler(writer, &slog.HandlerOptions{
			Level:       level,
			ReplaceAttr: defaultReplaceAttr,
		}), nil
	case appext.LogFormatText:
		// Use a custom console handler that formats log messages in a human-readable format.
		return newConsoleHandler(writer, level), nil
	case appext.LogFormatColor:
		// Use a custom console handler that formats log messages in a human-readable format, with colors.
		return newConsoleHandler(writer, level, withConsoleColor(true)), nil
	default:
		return nil, fmt.Errorf("invalid logFormat: %v", logFormat)
	}
}

// defaultReplaceAttr provides a default ReplaceAttr func for [slog.HandlerOptions].
func defaultReplaceAttr(groups []string, a slog.Attr) slog.Attr {
	// Make all Duration type values more useful by converting to their default String
	// representation, instead of using an integer number of nanoseconds.
	if a.Value.Kind() == slog.KindDuration {
		a.Value = slog.StringValue(a.Value.Duration().String())
	}
	return a
}

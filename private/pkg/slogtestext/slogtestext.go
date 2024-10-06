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

// Package slogtestext provides Loggers for testing.
package slogtestext

import (
	"log/slog"
	"os"
	"testing"

	"github.com/bufbuild/buf/private/pkg/slogbuild"
	"github.com/stretchr/testify/require"
)

// NewLogger returns a new Logger for testing.
func NewLogger(t testing.TB, options ...LoggerOption) *slog.Logger {
	loggerOptions := newLoggerOptions()
	for _, option := range options {
		option(loggerOptions)
	}
	// It's weird that we are going from slog.Level to string, and then within
	// slogbuild we are going from string to slog.Level, but there's no real reason
	// to clean this up at this exact point until we understand our slog call
	// patterns a bit more.
	logger, err := slogbuild.NewLoggerForFlagValues(os.Stderr, loggerOptions.level.String(), "text")
	require.NoError(t, err)
	return logger
}

// LoggerOption is an option for a new testing Logger.
type LoggerOption func(*loggerOptions)

// WithLevel specifies the Level to use for the Logger.
//
// The default is slog.LevelDebug.
func WithLevel(level slog.Level) LoggerOption {
	return func(loggerOptions *loggerOptions) {
		loggerOptions.level = level
	}
}

// *** PRIVATE ***

type loggerOptions struct {
	level slog.Level
}

func newLoggerOptions() *loggerOptions {
	return &loggerOptions{
		level: slog.LevelDebug,
	}
}

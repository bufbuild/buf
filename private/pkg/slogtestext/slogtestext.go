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

// Package slogtestext provides Loggers for testing.
package slogtestext

import (
	"log/slog"
	"testing"

	"buf.build/go/app/appext"
	"github.com/bufbuild/buf/private/pkg/slogapp"
	"github.com/stretchr/testify/require"
)

// NewLogger returns a new Logger for testing.
func NewLogger(t testing.TB, options ...LoggerOption) *slog.Logger {
	loggerOptions := newLoggerOptions()
	for _, option := range options {
		option(loggerOptions)
	}
	logger, err := slogapp.NewLogger(t.Output(), loggerOptions.logLevel, appext.LogFormatText)
	require.NoError(t, err)
	return logger
}

// LoggerOption is an option for a new testing Logger.
type LoggerOption func(*loggerOptions)

// WithLogLevel specifies the LogLevel to use for the Logger.
//
// The default is appext.LogLevelDebug.
func WithLogLevel(logLevel appext.LogLevel) LoggerOption {
	return func(loggerOptions *loggerOptions) {
		loggerOptions.logLevel = logLevel
	}
}

// *** PRIVATE ***

type loggerOptions struct {
	logLevel appext.LogLevel
}

func newLoggerOptions() *loggerOptions {
	return &loggerOptions{
		logLevel: appext.LogLevelDebug,
	}
}

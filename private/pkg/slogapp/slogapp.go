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

// Package slogapp builds slog.Loggers.
package slogapp

import (
	"io"
	"log/slog"

	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/zapapp"
	"go.uber.org/zap/exp/zapslog"
)

// LoggerProvider is an appext.LoggerProvider for use with appext.BuilderWithLoggerProvider.
func LoggerProvider(container appext.NameContainer, logLevel appext.LogLevel, logFormat appext.LogFormat) (*slog.Logger, error) {
	return NewLogger(container.Stderr(), logLevel, logFormat)
}

// NewLogger returns a new Logger for the given level and format.
func NewLogger(writer io.Writer, logLevel appext.LogLevel, logFormat appext.LogFormat) (*slog.Logger, error) {
	core, err := zapapp.NewCore(writer, logLevel, logFormat)
	if err != nil {
		return nil, err
	}
	return slog.New(zapslog.NewHandler(core)), nil
}

// Copyright 2020 Buf Technologies, Inc.
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

// Package applog contains utilities to work with logging.
package applog

import (
	"fmt"
	"io"
	"strings"

	"github.com/bufbuild/buf/internal/pkg/app"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Container is a container.
type Container interface {
	app.Container

	Logger() *zap.Logger
}

// NewContainer returns a new Container.
func NewContainer(appContainer app.Container, logger *zap.Logger) Container {
	return newContainer(appContainer, logger)
}

// NewLogger returns a new Logger.
//
// The level can be [debug,info,warn,error]. The default is info.
// The format can be [text,color,json]. The default is color.
func NewLogger(writer io.Writer, level string, format string) (*zap.Logger, error) {
	zapLevel, err := getZapLevel(level)
	if err != nil {
		return nil, err
	}
	zapEncoder, err := getZapEncoder(format)
	if err != nil {
		return nil, err
	}
	return zap.New(
		zapcore.NewCore(
			zapEncoder,
			zapcore.Lock(zapcore.AddSync(writer)),
			zap.NewAtomicLevelAt(zapLevel),
		),
	), nil
}

func getZapLevel(level string) (zapcore.Level, error) {
	level = strings.TrimSpace(strings.ToLower(level))
	switch level {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info", "":
		return zapcore.InfoLevel, nil
	case "warn":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	default:
		return 0, fmt.Errorf("unknown log level [debug,info,warn,error]: %q", level)
	}
}

func getZapEncoder(format string) (zapcore.Encoder, error) {
	format = strings.TrimSpace(strings.ToLower(format))
	switch format {
	case "text":
		return zapcore.NewConsoleEncoder(textEncoderConfig), nil
	case "color", "":
		return zapcore.NewConsoleEncoder(colortextEncoderConfig), nil
	case "json":
		return zapcore.NewJSONEncoder(jsonEncoderConfig), nil
	default:
		return nil, fmt.Errorf("unknown log format [text,color,json]: %q", format)
	}
}

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

// Package zaputil implements utilities for zap.
package zaputil

import (
	"fmt"
	"io"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger returns a new Logger.
func NewLogger(
	writer io.Writer,
	level zapcore.Level,
	encoder zapcore.Encoder,
) *zap.Logger {
	return zap.New(
		zapcore.NewCore(
			encoder,
			zapcore.Lock(zapcore.AddSync(writer)),
			zap.NewAtomicLevelAt(level),
		),
	)
}

// NewLoggerForFlagValues returns a new Logger for the given level and format strings.
//
// The level can be [debug,info,warn,error]. The default is info.
// The format can be [text,color,json]. The default is color.
func NewLoggerForFlagValues(writer io.Writer, levelString string, format string) (*zap.Logger, error) {
	level, err := getZapLevel(levelString)
	if err != nil {
		return nil, err
	}
	encoder, err := getZapEncoder(format)
	if err != nil {
		return nil, err
	}
	return NewLogger(writer, level, encoder), nil
}

// NewTextEncoder returns a new text Encoder.
func NewTextEncoder() zapcore.Encoder {
	return zapcore.NewConsoleEncoder(textEncoderConfig)
}

// NewColortextEncoder returns a new colortext Encoder.
func NewColortextEncoder() zapcore.Encoder {
	return zapcore.NewConsoleEncoder(colortextEncoderConfig)
}

// NewJSONEncoder returns a new JSON encoder.
func NewJSONEncoder() zapcore.Encoder {
	return zapcore.NewJSONEncoder(jsonEncoderConfig)
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
		return NewTextEncoder(), nil
	case "color", "":
		return NewColortextEncoder(), nil
	case "json":
		return NewJSONEncoder(), nil
	default:
		return nil, fmt.Errorf("unknown log format [text,color,json]: %q", format)
	}
}

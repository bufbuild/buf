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

// Package zapapp builds zap.Loggers.
package zapapp

import (
	"fmt"
	"io"

	"github.com/bufbuild/buf/private/pkg/app/appext"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewCoreForFlagValues returns a new Core for the given level and format strings.
func NewCore(writer io.Writer, logLevel appext.LogLevel, logFormat appext.LogFormat) (zapcore.Core, error) {
	level, err := getLevel(logLevel)
	if err != nil {
		return nil, err
	}
	encoder, err := getEncoder(logFormat)
	if err != nil {
		return nil, err
	}
	return zapcore.NewCore(
		encoder,
		zapcore.Lock(zapcore.AddSync(writer)),
		zap.NewAtomicLevelAt(level),
	), nil
}

func getLevel(logLevel appext.LogLevel) (zapcore.Level, error) {
	switch logLevel {
	case appext.LogLevelDebug:
		return zapcore.DebugLevel, nil
	case appext.LogLevelInfo:
		return zapcore.InfoLevel, nil
	case appext.LogLevelWarn:
		return zapcore.WarnLevel, nil
	case appext.LogLevelError:
		return zapcore.ErrorLevel, nil
	default:
		return 0, fmt.Errorf("unknown appext.LogLevel: %v", logLevel)
	}
}

func getEncoder(logFormat appext.LogFormat) (zapcore.Encoder, error) {
	switch logFormat {
	case appext.LogFormatText:
		return newTextEncoder(), nil
	case appext.LogFormatColor:
		return newColortextEncoder(), nil
	case appext.LogFormatJSON:
		return newJSONEncoder(), nil
	default:
		return nil, fmt.Errorf("unknown appext.LogFormat: %v", logFormat)
	}
}

func newTextEncoder() zapcore.Encoder {
	return zapcore.NewConsoleEncoder(textEncoderConfig)
}

func newColortextEncoder() zapcore.Encoder {
	return zapcore.NewConsoleEncoder(colortextEncoderConfig)
}

func newJSONEncoder() zapcore.Encoder {
	return zapcore.NewJSONEncoder(jsonEncoderConfig)
}

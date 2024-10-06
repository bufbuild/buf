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

// Package zapbuild builds zap Loggers.
package zapbuild

import (
	"fmt"
	"io"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewCoreForFlagValues returns a new Core for the given level and format strings.
//
// The level can be [debug,info,warn,error]. The default is info.
// The format can be [text,color,json]. The default is color.
func NewCoreForFlagValues(writer io.Writer, levelString string, formatString string) (zapcore.Core, error) {
	level, err := getLevel(levelString)
	if err != nil {
		return nil, err
	}
	encoder, err := getEncoder(formatString)
	if err != nil {
		return nil, err
	}
	return zapcore.NewCore(
		encoder,
		zapcore.Lock(zapcore.AddSync(writer)),
		zap.NewAtomicLevelAt(level),
	), nil
}

func getLevel(levelString string) (zapcore.Level, error) {
	levelString = strings.TrimSpace(strings.ToLower(levelString))
	switch levelString {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info", "":
		return zapcore.InfoLevel, nil
	case "warn":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	default:
		return 0, fmt.Errorf("unknown log level [debug,info,warn,error]: %q", levelString)
	}
}

func getEncoder(formatString string) (zapcore.Encoder, error) {
	formatString = strings.TrimSpace(strings.ToLower(formatString))
	switch formatString {
	case "text":
		return newTextEncoder(), nil
	case "color", "":
		return newColortextEncoder(), nil
	case "json":
		return newJSONEncoder(), nil
	default:
		return nil, fmt.Errorf("unknown log format [text,color,json]: %q", formatString)
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

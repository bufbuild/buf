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

// Package slogbuild builds slog.Loggers based on flags.
package slogbuild

import (
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/lmittmann/tint"
)

// NewLoggerForFlagValues returns a new Logger for the given level and format strings.
//
// The level can be [debug,info,warn,error]. The default is info.
// The format can be [text,color,json]. The default is color.
func NewLoggerForFlagValues(writer io.Writer, levelString string, formatString string) (*slog.Logger, error) {
	level, err := getLevel(levelString)
	if err != nil {
		return nil, err
	}
	handler, err := getHandler(writer, level, formatString)
	if err != nil {
		return nil, err
	}
	return slog.New(handler), nil
}

func getLevel(levelString string) (slog.Level, error) {
	levelString = strings.TrimSpace(strings.ToLower(levelString))
	switch levelString {
	case "debug":
		return slog.LevelDebug, nil
	case "info", "":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unknown log level [debug,info,warn,error]: %q", levelString)
	}
}

func getHandler(writer io.Writer, level slog.Level, formatString string) (slog.Handler, error) {
	formatString = strings.TrimSpace(strings.ToLower(formatString))
	switch formatString {
	case "text":
		return slog.NewTextHandler(writer, &slog.HandlerOptions{Level: level}), nil
	case "color", "":
		return tint.NewHandler(writer, &tint.Options{Level: level}), nil
	case "json":
		return slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: level}), nil
	default:
		return nil, fmt.Errorf("unknown log format [text,color,json]: %q", formatString)
	}
}

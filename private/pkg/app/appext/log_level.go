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

package appext

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

const (
	// LogLevelDebug is the debug log level.
	LogLevelDebug LogLevel = iota + 1
	// LogLevelInfo is the infolog level.
	LogLevelInfo
	// LogLevelWarn is the warn log level.
	LogLevelWarn
	// LogLevelError is the error log level.
	LogLevelError
)

// LogLevel is a level to print logs in.
type LogLevel int

// String implements fmt.Stringer
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "debug"
	case LogLevelInfo:
		return "info"
	case LogLevelWarn:
		return "warn"
	case LogLevelError:
		return "error"
	default:
		return strconv.Itoa(int(l))
	}
}

// SlogLevel returns the corresponding slog.Level.
//
// If l is known, this return the corresponding value.
// If l < LogLevelDebug, this returns slog.LevelDebug.
// If l > LogLevelError, this returns slog.LevelError.
// Otherwise, this returns slog.LevelInfo.
func (l LogLevel) SlogLevel() slog.Level {
	switch l {
	case LogLevelDebug:
		return slog.LevelDebug
	case LogLevelInfo:
		return slog.LevelInfo
	case LogLevelWarn:
		return slog.LevelWarn
	case LogLevelError:
		return slog.LevelError
	}
	if l < LogLevelDebug {
		return slog.LevelDebug
	}
	if l > LogLevelError {
		return slog.LevelError
	}
	return slog.LevelInfo
}

// ParseLogLevel parses the log level for the string.
//
// If logLevelString is empty, this returns LogLevelInfo.
func ParseLogLevel(logLevelString string) (LogLevel, error) {
	logLevelString = strings.TrimSpace(strings.ToLower(logLevelString))
	switch logLevelString {
	case "debug":
		return LogLevelDebug, nil
	case "info", "":
		return LogLevelInfo, nil
	case "warn":
		return LogLevelWarn, nil
	case "error":
		return LogLevelError, nil
	default:
		return 0, fmt.Errorf("unknown log level [debug,info,warn,error]: %q", logLevelString)
	}
}

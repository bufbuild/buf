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

// Package slogext implements extended functionality for slog.
package slogext

import (
	"context"
	"log/slog"
	"runtime"
	"time"
)

var (
	// NopLogger is a no-op Logger.
	NopLogger = slog.New(NopHandler)
	// NopHandler is no-op Handler.
	NopHandler slog.Handler = nopHandler{}
)

// ErrorAttr returns a slog.Attr for the error.
//
// If err is nil, this returns slog.Attr{}.
func ErrorAttr(err error) slog.Attr {
	if err == nil {
		return slog.Attr{}
	}
	return slog.Any("error", err)
}

// DebugProfile will result in the function's elapsed time being printed as a debug log line.
func DebugProfile(logger *slog.Logger, extraFields ...any) func() {
	message := getRuntimeFrame(2).Function
	start := time.Now()
	return func() {
		logger.Debug(
			message,
			append(
				[]any{
					slog.Duration("duration", time.Since(start)),
				},
				extraFields...,
			)...,
		)
	}
}

// *** PRIVATE ***

type nopHandler struct{}

func (nopHandler) Enabled(context.Context, slog.Level) bool {
	return false
}

func (nopHandler) Handle(context.Context, slog.Record) error {
	return nil
}

func (n nopHandler) WithAttrs([]slog.Attr) slog.Handler {
	return n
}

func (n nopHandler) WithGroup(string) slog.Handler {
	return n
}

func getRuntimeFrame(skipFrames int) runtime.Frame {
	targetFrameIndex := skipFrames + 2
	programCounters := make([]uintptr, targetFrameIndex+2)
	n := runtime.Callers(0, programCounters)
	var frame runtime.Frame
	if n > 0 {
		frames := runtime.CallersFrames(programCounters[:n])
		for more, frameIndex := true, 0; more && frameIndex <= targetFrameIndex; frameIndex++ {
			var frameCandidate runtime.Frame
			frameCandidate, more = frames.Next()
			if frameIndex == targetFrameIndex {
				frame = frameCandidate
			}
		}
	}
	return frame
}

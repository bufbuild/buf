// Copyright 2020 Buf Technologies Inc.
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

// Package utillog implements log utilities.
package utillog

import (
	"time"

	"go.uber.org/zap"
)

// Defer returns a function to defer that logs at the debug level.
//
// defer utillog.Defer(logger, "foo")()
func Defer(logger *zap.Logger, name string, fields ...zap.Field) func() {
	start := time.Now()
	return func() {
		fields = append(fields, zap.Duration("duration", time.Since(start)))
		logger.Debug(name, fields...)
	}
}

// DeferWithError returns a function to defer that logs at the debug level.
//
// defer utillog.DeferWithError(logger, "foo", &retErr)()
func DeferWithError(logger *zap.Logger, name string, retErrPtr *error, fields ...zap.Field) func() {
	start := time.Now()
	return func() {
		fields = append(fields, zap.Duration("duration", time.Since(start)))
		if retErrPtr != nil && *retErrPtr != nil {
			fields = append(fields, zap.Error(*retErrPtr))
		}
		logger.Debug(name, fields...)
	}
}

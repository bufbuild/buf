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

// Package instrument implements instrumentation.
package instrument

import (
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Timer logs a duration to a logger.
type Timer interface {
	End(...zap.Field)
}

// Start returns a new Timer.
func Start(logger *zap.Logger, message string, fields ...zap.Field) Timer {
	if checkedEntry := logger.Check(zap.DebugLevel, message); checkedEntry != nil {
		return newTimer(checkedEntry, fields...)
	}
	return nopTimer{}
}

type timer struct {
	checkedEntry *zapcore.CheckedEntry
	fields       []zap.Field
	start        time.Time
}

func newTimer(checkedEntry *zapcore.CheckedEntry, fields ...zap.Field) *timer {
	return &timer{
		checkedEntry: checkedEntry,
		fields:       fields,
		start:        time.Now(),
	}
}

func (t *timer) End(extraFields ...zap.Field) {
	t.checkedEntry.Write(
		append(
			t.fields,
			append(
				extraFields,
				zap.Duration("duration", time.Since(t.start)),
			)...,
		)...,
	)
}

type nopTimer struct{}

func (nopTimer) End(...zap.Field) {}

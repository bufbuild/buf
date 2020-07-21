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

package observabilityzap

import (
	"context"
	"time"

	"go.opencensus.io/trace"
	"go.uber.org/zap"
)

type zapTracer struct {
	logger *zap.Logger
}

func newExporter(logger *zap.Logger) *zapTracer {
	return &zapTracer{
		logger: logger,
	}
}

func (t *zapTracer) Run(ctx context.Context) error {
	trace.ApplyConfig(trace.Config{
		DefaultSampler: trace.AlwaysSample(),
	})
	trace.RegisterExporter(t)
	return nil
}

// ExportSpan implements the opencensus trace.Exporter interface.
func (t *zapTracer) ExportSpan(sd *trace.SpanData) {
	if sd == nil || !sd.IsSampled() {
		return
	}
	checkedEntry := t.logger.Check(zap.DebugLevel, sd.Message)
	if checkedEntry == nil {
		return
	}
	fields := []zap.Field{
		zap.String("name", sd.Name),
		zap.String("message", sd.Message),
		zap.Duration("duration", time.Since(sd.StartTime)),
	}
	for key, att := range sd.Attributes {
		fields = append(fields, zap.Any(key, att))
	}
	checkedEntry.Write(fields...)
}

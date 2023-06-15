// Copyright 2020-2023 Buf Technologies, Inc.
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

	"go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
)

var _ trace.SpanExporter = &zapExporter{}

type zapExporter struct {
	logger *zap.Logger
}

func newZapExporter(logger *zap.Logger) *zapExporter {
	return &zapExporter{
		logger: logger,
	}
}

func (z *zapExporter) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	for _, span := range spans {
		if !span.SpanContext().IsSampled() {
			continue
		}
		if checkedEntry := z.logger.Check(zap.DebugLevel, span.Name()); checkedEntry != nil {
			fields := []zap.Field{
				zap.Duration("duration", span.EndTime().Sub(span.StartTime())),
			}
			for _, attribute := range span.Attributes() {
				fields = append(fields, zap.Any(string(attribute.Key), attribute.Value.AsInterface()))
			}
			checkedEntry.Write(fields...)
		}
	}
	return nil
}

func (z *zapExporter) Shutdown(ctx context.Context) error {
	return nil
}

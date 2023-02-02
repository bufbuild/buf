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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
)

// Start creates a Zap logging exporter for Opentelemetry traces and returns
// the exporter. The exporter implements io.Closer for clean-up.
func Start(logger *zap.Logger) *tracerProviderCloser {
	exporter := newZapExporter(logger)
	tracerProviderOptions := []trace.TracerProviderOption{
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithSpanProcessor(trace.NewBatchSpanProcessor(exporter)),
	}
	tracerProvider := newTracerProviderCloser(trace.NewTracerProvider(tracerProviderOptions...))
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return tracerProvider
}

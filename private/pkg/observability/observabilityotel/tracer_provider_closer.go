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

package observabilityotel

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

type tracerProviderCloser struct {
	tracerProvider TracerProvider
}

func newTracerProviderCloser(tracerProvider TracerProvider) *tracerProviderCloser {
	return &tracerProviderCloser{
		tracerProvider: tracerProvider,
	}
}

func (t *tracerProviderCloser) Close() error {
	// Note: the application layer above does not pass down a context required for
	// otelsdktrace.TracerProvider.Shutdown
	return t.tracerProvider.Shutdown(context.Background())
}

func (t *tracerProviderCloser) Tracer(instrumentationName string, opts ...trace.TracerOption) trace.Tracer {
	return t.tracerProvider.Tracer(instrumentationName, opts...)
}

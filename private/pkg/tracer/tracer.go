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

package tracer

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Start creates a span.
func Start(ctx context.Context, tracerName string, spanName string) (context.Context, trace.Span) {
	return otel.GetTracerProvider().Tracer(tracerName).Start(ctx, spanName)
}

// Start creates a span, recording the error from retErrAddr on End.
func StartRetErr(ctx context.Context, tracerName string, spanName string, retErrAddr *error) (context.Context, trace.Span) {
	ctx, span := otel.GetTracerProvider().Tracer(tracerName).Start(ctx, spanName)
	return ctx, newRetErrSpan(span, retErrAddr)
}

// Do runs f with a span.
func Do(ctx context.Context, tracerName string, spanName string, f func(context.Context) error) (retErr error) {
	ctx, span := StartRetErr(ctx, tracerName, spanName, &retErr)
	defer span.End()
	return f(ctx)
}

type retErrSpan struct {
	trace.Span
	retErrAddr *error
}

func newRetErrSpan(span trace.Span, retErrAddr *error) *retErrSpan {
	return &retErrSpan{
		Span:       span,
		retErrAddr: retErrAddr,
	}
}

func (s *retErrSpan) End(options ...trace.SpanEndOption) {
	s.Span.End(options...)
	if s.retErrAddr != nil {
		if retErr := *s.retErrAddr; retErr != nil {
			s.Span.RecordError(retErr)
			s.Span.SetStatus(codes.Error, retErr.Error())
		}
	}
}

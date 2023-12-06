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
	"runtime"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Start creates a span.
func Start(ctx context.Context, tracerName string, options ...StartOption) (context.Context, trace.Span) {
	startOptions := newStartOptions()
	for _, option := range options {
		option(startOptions)
	}
	spanName := startOptions.spanName
	if spanName == "" {
		spanName = getRuntimeFrame(2).Function
	}
	var spanStartOptions []trace.SpanStartOption
	if len(startOptions.attributes) > 0 {
		spanStartOptions = append(
			spanStartOptions,
			trace.WithAttributes(startOptions.attributes...),
		)
	}
	ctx, span := otel.GetTracerProvider().Tracer(tracerName).Start(ctx, spanName, spanStartOptions...)
	return ctx, newWrappedSpan(span, startOptions.errAddr)
}

// Do runs f with a span.
func Do(
	ctx context.Context,
	tracerName string,
	f func(context.Context) error,
	options ...StartOption,
) error {
	ctx, span := Start(ctx, tracerName, options...)
	defer span.End()
	return f(ctx)
}

// StartOption is an option for Start or Do.
type StartOption func(*startOptions)

// WithTracerName sets the given tracer name.
//
// The default is to use filename.Base(os.Args[0])
func WithTracerName(tracerName string) StartOption {
	return func(startOptions *startOptions) {
		startOptions.tracerName = tracerName
	}
}

// WithSpanName sets the span name.
//
// The default is to use the calling function name.
func WithSpanName(spanName string) StartOption {
	return func(startOptions *startOptions) {
		startOptions.spanName = spanName
	}
}

// WithErr will result in the given error being recorded on span.End()
// if the error is not nil, and the status being set to error.
func WithErr(errAddr *error) StartOption {
	return func(startOptions *startOptions) {
		startOptions.errAddr = errAddr
	}
}

// WithAttributes adds the given attributes.
func WithAttributes(attributes ...attribute.KeyValue) StartOption {
	return func(startOptions *startOptions) {
		startOptions.attributes = append(startOptions.attributes, attributes...)
	}
}

// *** PRIVATE ***

type wrappedSpan struct {
	trace.Span
	errAddr *error
}

func newWrappedSpan(span trace.Span, errAddr *error) *wrappedSpan {
	return &wrappedSpan{
		Span:    span,
		errAddr: errAddr,
	}
}

func (s *wrappedSpan) End(options ...trace.SpanEndOption) {
	s.Span.End(options...)
	if s.errAddr != nil {
		if retErr := *s.errAddr; retErr != nil {
			s.Span.RecordError(retErr)
			s.Span.SetStatus(codes.Error, retErr.Error())
		}
	}
}

type startOptions struct {
	tracerName string
	spanName   string
	errAddr    *error
	attributes []attribute.KeyValue
}

func newStartOptions() *startOptions {
	return &startOptions{}
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

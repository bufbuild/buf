// Copyright 2020-2021 Buf Technologies, Inc.
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

package observability

import (
	"io"

	"github.com/bufbuild/buf/internal/pkg/ioutilextended"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

// TraceExportCloser describes the interface used to export OpenCensus traces
// and cleaning of resources.
type TraceExportCloser interface {
	trace.Exporter
	io.Closer
}

// TraceViewExportCloser implements both OpenCensus view and trace exporting.
type TraceViewExportCloser interface {
	view.Exporter
	TraceExportCloser
}

// Start initializes tracing.
//
// Tracing is a global function due to how go.opencensus.io is written.
// The returned io.Closer needs to be called at the completion of the program.
func Start(options ...StartOption) io.Closer {
	startOptions := newStartOptions()
	for _, option := range options {
		option(startOptions)
	}
	if len(startOptions.traceExportClosers) == 0 && len(startOptions.traceViewExportClosers) == 0 {
		return ioutilextended.NopCloser
	}
	trace.ApplyConfig(
		trace.Config{
			DefaultSampler: trace.AlwaysSample(),
		},
	)
	var closers []io.Closer
	for _, traceExportCloser := range startOptions.traceExportClosers {
		traceExportCloser := traceExportCloser
		trace.RegisterExporter(traceExportCloser)
		closers = append(closers, traceExportCloser)
	}
	for _, traceViewExportCloser := range startOptions.traceViewExportClosers {
		traceViewExportCloser := traceViewExportCloser
		trace.RegisterExporter(traceViewExportCloser)
		view.RegisterExporter(traceViewExportCloser)
		closers = append(closers, traceViewExportCloser)
	}
	return ioutilextended.ChainCloser(closers...)
}

// StartOption is an option for start.
type StartOption func(*startOptions)

// StartWithTraceExportCloser returns a new StartOption that adds the given TraceExportCloser.
func StartWithTraceExportCloser(traceExportCloser TraceExportCloser) StartOption {
	return func(startOptions *startOptions) {
		startOptions.traceExportClosers = append(startOptions.traceExportClosers, traceExportCloser)
	}
}

// StartWithTraceViewExportCloser returns a new StartOption that adds the given TraceViewExportCloser.
func StartWithTraceViewExportCloser(traceViewExportCloser TraceViewExportCloser) StartOption {
	return func(startOptions *startOptions) {
		startOptions.traceViewExportClosers = append(startOptions.traceViewExportClosers, traceViewExportCloser)
	}
}

type startOptions struct {
	traceExportClosers     []TraceExportCloser
	traceViewExportClosers []TraceViewExportCloser
}

func newStartOptions() *startOptions {
	return &startOptions{}
}

// Copyright 2020-2022 Buf Technologies, Inc.
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

package httpserver

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

const (
	// DefaultShutdownTimeout is the default shutdown timeout.
	DefaultShutdownTimeout = 10 * time.Second
	// DefaultReadHeaderTimeout is the default read header timeout.
	DefaultReadHeaderTimeout = 30 * time.Second
	// DefaultIdleTimeout is the amount of time an HTTP/2 connection can be idle.
	DefaultIdleTimeout = 3 * time.Minute
)

// Mapper initializes a Router.
type Mapper interface {
	Map(chi.Router) error
}

// MapperFunc is a function for a Mapper
type MapperFunc func(chi.Router) error

// Map implements Mapper.
func (m MapperFunc) Map(router chi.Router) error {
	return m(router)
}

// NewHTTPHandlerMapper returns a new Mapper for the http.Handler.
func NewHTTPHandlerMapper(
	handler http.Handler,
	options ...HTTPHandlerMapperOption,
) Mapper {
	return newHTTPHandlerMapper(handler, options...)
}

// HTTPHandlerMapperOption is an option for a new HTTPHandlerMapper.
type HTTPHandlerMapperOption func(*httpHandlerMapper)

// PrefixedHTTPHandler is an http.Handler with a path prefix.
//
// A router should route all requests with the path prefix to this handler.
type PrefixedHTTPHandler interface {
	http.Handler
	PathPrefix() string
}

// Runner is a runner.
type Runner interface {
	// Run runs the router.
	//
	// Blocking.
	// The runner is cancelled when the input context is cancelled.
	// The listener is closed upon return.
	//
	// Response write errors are logged. Response write errors can be ignored.
	//
	// Can be called multiple times, resulting in different runs.
	Run(ctx context.Context, listener net.Listener, mappers ...Mapper) error
}

// NewRunner returns a new Runner.
func NewRunner(logger *zap.Logger, options ...RunnerOption) Runner {
	return newRunner(logger, options...)
}

// RunnerOption is an option for a new Runner.
type RunnerOption func(*runner)

// RunnerWithShutdownTimeout returns a new RunnerOption that uses the given shutdown timeout.
//
// The default is to use DefaultShutdownTimeout.
// If shutdownTimeout is 0, no graceful shutdown will be performed.
func RunnerWithShutdownTimeout(shutdownTimeout time.Duration) RunnerOption {
	return func(runner *runner) {
		runner.shutdownTimeout = shutdownTimeout
	}
}

// RunnerWithReadHeaderTimeout returns a new RunnerOption that uses the given read header timeout.
//
// The default is to use DefaultReadHeaderTimeout.
// If readHeaderTimeout is 0, no read header timeout will be used.
func RunnerWithReadHeaderTimeout(readHeaderTimeout time.Duration) RunnerOption {
	return func(runner *runner) {
		runner.readHeaderTimeout = readHeaderTimeout
	}
}

// RunnerWithTLSConfig returns a new RunnerOption that uses the given tls.Config.
//
// The default is to use no TLS.
func RunnerWithTLSConfig(tlsConfig *tls.Config) RunnerOption {
	return func(runner *runner) {
		runner.tlsConfig = tlsConfig
	}
}

// RunnerWithObservability returns a new RunnerOption that turns on
// OpenCensus tracing and metrics.
//
// The default is to not turn on observability.
func RunnerWithObservability(middleware func(http.Handler) http.Handler) RunnerOption {
	return func(runner *runner) {
		runner.observability = middleware
	}
}

// RunnerWithHealth returns a new RunnerOption that turns a health check endpoint on at /health.
//
// The default is to not turn on health.
func RunnerWithHealth() RunnerOption {
	return func(runner *runner) {
		runner.health = true
	}
}

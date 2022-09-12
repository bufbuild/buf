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

// HTTPHandlerMapperWithPrefix returns a new HTTPHandlerMapperOption that uses the prefix.
func HTTPHandlerMapperWithPrefix(prefix string) HTTPHandlerMapperOption {
	return func(httpHandlerMapper *httpHandlerMapper) {
		httpHandlerMapper.prefix = prefix
	}
}

// MapperWithMiddlewares rewrites the Map function of the Mapper to call the provided middlewares
// inside a chi Group before mapping
func MapperWithMiddlewares(mapper Mapper, middlewares chi.Middlewares) Mapper {
	return MapperFunc(func(parent chi.Router) error {
		var err error
		parent.Group(func(r chi.Router) {
			r.Use(middlewares...)
			err = mapper.Map(r)
		})
		return err
	})
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

// RunnerWithMiddlewares returns a new RunnerOption that use middlewares when the Runner Run.
func RunnerWithMiddlewares(middlewares ...func(http.Handler) http.Handler) RunnerOption {
	return func(runner *runner) {
		runner.middlewares = append(runner.middlewares, middlewares...)
	}
}

// RunnerWithWalkFunc returns a new RunnerOption that runs chi.Walk to walk the router
// after all middlewares and routes have been mounted, but before the server is started.
func RunnerWithWalkFunc(walkFunc chi.WalkFunc) RunnerOption {
	return func(runner *runner) {
		runner.walkFunc = walkFunc
	}
}

// RunnerWithMaxBodySize returns a new RunnerOption that sets the max size of
// incoming request body.
//
// The default is to not limit body size.
func RunnerWithMaxBodySize(maxBodySize int64) RunnerOption {
	return func(runner *runner) {
		runner.maxBodySize = maxBodySize
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

// RunnerWithSilentEndpoints returns a new RunnerOption that disables logging from the
// provided endpoints.
//
// The default is to not silence any endpoints.
func RunnerWithSilentEndpoints(silentEndpoints ...string) RunnerOption {
	return func(runner *runner) {
		for _, endpoint := range silentEndpoints {
			runner.silentEndpoints[endpoint] = struct{}{}
		}
	}
}

// RunOption is an option for a new Run.
type RunOption = RunnerOption

// RunWithShutdownTimeout returns a new RunOption that uses the given shutdown timeout.
//
// The default is to use DefaultShutdownTimeout.
// If shutdownTimeout is 0, no graceful shutdown will be performed.
func RunWithShutdownTimeout(shutdownTimeout time.Duration) RunOption {
	return func(runner *runner) {
		runner.shutdownTimeout = shutdownTimeout
	}
}

// RunWithReadHeaderTimeout returns a new RunOption that uses the given read header timeout.
//
// The default is to use DefaultReadHeaderTimeout.
// If readHeaderTimeout is 0, no read header timeout will be used.
func RunWithReadHeaderTimeout(readHeaderTimeout time.Duration) RunOption {
	return func(runner *runner) {
		runner.readHeaderTimeout = readHeaderTimeout
	}
}

// RunWithTLSConfig returns a new RunOption that uses the given tls.Config.
//
// The default is to use no TLS.
func RunWithTLSConfig(tlsConfig *tls.Config) RunOption {
	return func(runner *runner) {
		runner.tlsConfig = tlsConfig
	}
}

// Run will start a HTTP server listening on the provided listener and
// serving the provided handler. This call is blocking and the run
// is cancelled when the input context is cancelled, the listener is
// closed upon return.
//
// The Run can be configured further by passing a variety of options.
func Run(
	ctx context.Context,
	logger *zap.Logger,
	listener net.Listener,
	handler http.Handler,
	options ...RunOption,
) (retErr error) {
	return newRunner(logger, options...).serve(ctx, listener, handler)
}

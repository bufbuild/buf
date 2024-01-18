// Copyright 2020-2024 Buf Technologies, Inc.
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
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"
)

const (
	// DefaultShutdownTimeout is the default shutdown timeout.
	DefaultShutdownTimeout = 10 * time.Second
	// DefaultReadHeaderTimeout is the default read header timeout.
	DefaultReadHeaderTimeout = 30 * time.Second
	// DefaultIdleTimeout is the amount of time an HTTP/2 connection can be idle.
	DefaultIdleTimeout = 3 * time.Minute
)

type runner struct {
	shutdownTimeout   time.Duration
	readHeaderTimeout time.Duration
	tlsConfig         *tls.Config
	walkFunc          chi.WalkFunc
	disableH2C        bool
}

// RunOption is an option for a new Run.
type RunOption func(*runner)

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

// RunWithWalkFunc returns a new RunOption that runs chi.Walk to walk the
// handler after all middlewares and routes have been mounted, but before the
// server is started.
// The walkFunc will only be called if the handler passed to Run is a
// chi.Routes.
func RunWithWalkFunc(walkFunc chi.WalkFunc) RunOption {
	return func(runner *runner) {
		runner.walkFunc = walkFunc
	}
}

// RunWithoutH2C disables use of H2C (used when RunWithTLSConfig is not called).
func RunWithoutH2C() RunOption {
	return func(runner *runner) {
		runner.disableH2C = true
	}
}

// Run will start a HTTP server listening on the provided listener and
// serving the provided handler. This call is blocking and the run
// is cancelled when the input context is cancelled, the listener is
// closed upon return.
//
// The Run function can be configured further by passing a variety of options.
func Run(
	ctx context.Context,
	logger *zap.Logger,
	listener net.Listener,
	handler http.Handler,
	options ...RunOption,
) error {
	s := &runner{
		shutdownTimeout:   DefaultShutdownTimeout,
		readHeaderTimeout: DefaultReadHeaderTimeout,
	}
	for _, option := range options {
		option(s)
	}
	stdLogger, err := zap.NewStdLogAt(logger.Named("httpserver"), zap.ErrorLevel)
	if err != nil {
		return err
	}
	httpServer := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: s.readHeaderTimeout,
		ErrorLog:          stdLogger,
		TLSConfig:         s.tlsConfig,
	}
	if s.tlsConfig == nil && !s.disableH2C {
		httpServer.Handler = h2c.NewHandler(handler, &http2.Server{
			IdleTimeout: DefaultIdleTimeout,
		})
	}
	if s.walkFunc != nil {
		routes, ok := handler.(chi.Routes)
		if ok {
			if err := chi.Walk(routes, s.walkFunc); err != nil {
				return err
			}
		}
	}

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return httpServe(httpServer, listener)
	})
	eg.Go(func() error {
		<-ctx.Done()
		start := time.Now()
		logger.Info("shutdown_starting", zap.Duration("shutdown_timeout", s.shutdownTimeout))
		defer func() {
			logger.Info("shutdown_finished", zap.Duration("duration", time.Since(start)))
		}()
		if s.shutdownTimeout != 0 {
			ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
			defer cancel()
			return httpServer.Shutdown(ctx)
		}
		return httpServer.Close()
	})

	logger.Info(
		"starting",
		zap.String("address", listener.Addr().String()),
		zap.Duration("shutdown_timeout", s.shutdownTimeout),
		zap.Bool("tls", s.tlsConfig != nil),
	)
	if err := eg.Wait(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

func httpServe(httpServer *http.Server, listener net.Listener) error {
	if httpServer.TLSConfig != nil {
		return httpServer.ServeTLS(listener, "", "")
	}
	return httpServer.Serve(listener)
}

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
	"strings"
	"time"

	"github.com/bufbuild/buf/private/pkg/rpc"
	"github.com/bufbuild/buf/private/pkg/rpc/rpcheader"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"
)

type runner struct {
	logger            *zap.Logger
	shutdownTimeout   time.Duration
	readHeaderTimeout time.Duration
	tlsConfig         *tls.Config
	middlewares       []func(http.Handler) http.Handler
	health            bool
	maxBodySize       int64
	walkFunc          chi.WalkFunc
}

func newRunner(logger *zap.Logger, options ...RunnerOption) *runner {
	runner := &runner{
		logger:            logger.Named("httpserver"),
		shutdownTimeout:   DefaultShutdownTimeout,
		readHeaderTimeout: DefaultReadHeaderTimeout,
	}
	for _, option := range options {
		option(runner)
	}
	return runner
}

// This should be the last interceptor installed.

func (s *runner) Run(
	ctx context.Context,
	listener net.Listener,
	mappers ...Mapper,
) (retErr error) {
	start := time.Now()
	defer func() {
		if retErr != nil {
			s.logger.Error("finished", zap.Duration("duration", time.Since(start)), zap.Error(retErr))
		} else {
			s.logger.Info("finished", zap.Duration("duration", time.Since(start)))
		}
	}()

	mux := chi.NewMux()
	mux.Use(middleware.Recoverer)
	mux.Use(middleware.StripSlashes)
	mux.Use(newZapMiddleware(s.logger))
	if s.maxBodySize > 0 {
		mux.Use(func(next http.Handler) http.Handler {
			return http.MaxBytesHandler(next, s.maxBodySize)
		})
	}
	mux.Use(NewServerInterceptor())
	mux.Use(s.middlewares...)
	for _, mapper := range mappers {
		if err := mapper.Map(mux); err != nil {
			return err
		}
	}
	if s.health {
		mux.Get(
			"/health",
			func(responseWriter http.ResponseWriter, _ *http.Request) {
				responseWriter.WriteHeader(http.StatusOK)
			},
		)
	}
	if s.walkFunc != nil {
		if err := chi.Walk(mux, s.walkFunc); err != nil {
			return err
		}
	}

	stdLogger, err := zap.NewStdLogAt(s.logger, zap.ErrorLevel)
	if err != nil {
		return err
	}
	httpServer := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: s.readHeaderTimeout,
		ErrorLog:          stdLogger,
		TLSConfig:         s.tlsConfig,
	}
	if s.tlsConfig == nil {
		httpServer.Handler = h2c.NewHandler(mux, &http2.Server{
			IdleTimeout: DefaultIdleTimeout,
		})
	}

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return httpServe(httpServer, listener)
	})
	eg.Go(func() error {
		<-ctx.Done()
		start := time.Now()
		s.logger.Info("shutdown_starting", zap.Duration("shutdown_timeout", s.shutdownTimeout))
		defer s.logger.Info("shutdown_finished", zap.Duration("duration", time.Since(start)))
		if s.shutdownTimeout != 0 {
			ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
			defer cancel()
			return httpServer.Shutdown(ctx)
		}
		return httpServer.Close()
	})

	s.logger.Info(
		"starting",
		zap.String("address", listener.Addr().String()),
		zap.Duration("shutdown_timeout", s.shutdownTimeout),
		zap.Bool("tls", s.tlsConfig != nil),
		zap.Int("middlewares", len(s.middlewares)),
		zap.Int64("max_body_size", s.maxBodySize),
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

func newServerInterceptor() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if len(request.Header) > 0 {
				request = request.WithContext(
					rpc.WithIncomingHeaders(
						request.Context(),
						fromHTTPHeader(
							request.Header,
						),
					),
				)
			}
			next.ServeHTTP(writer, request)
		})
	}
}

func fromHTTPHeader(httpHeader http.Header) map[string]string {
	headers := make(map[string]string)
	for key, values := range httpHeader {
		key = strings.ToLower(key)
		// prefix so that we strip out other headers
		// rpc clients and servers should only be aware of headers set with the rpc package
		if strings.HasPrefix(key, rpcheader.KeyPrefix) {
			if key := strings.TrimPrefix(key, rpcheader.KeyPrefix); key != "" {
				headers[key] = values[0]
			}
		}
	}
	return headers
}

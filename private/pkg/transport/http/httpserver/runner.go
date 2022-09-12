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
	silentEndpoints   map[string]struct{}
}

func newRunner(logger *zap.Logger, options ...RunnerOption) *runner {
	runner := &runner{
		logger:            logger.Named("httpserver"),
		shutdownTimeout:   DefaultShutdownTimeout,
		readHeaderTimeout: DefaultReadHeaderTimeout,
		silentEndpoints:   make(map[string]struct{}),
	}
	for _, option := range options {
		option(runner)
	}
	return runner
}

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
	mux.Use(newZapMiddleware(s.logger, s.silentEndpoints))
	if s.maxBodySize > 0 {
		mux.Use(func(next http.Handler) http.Handler {
			return http.MaxBytesHandler(next, s.maxBodySize)
		})
	}
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
	return s.serve(ctx, listener, mux)
}

// serve is intended to only be called within the httpserver package by
// the Run function. Once callers have migrated away from NewRunner().Run()
// it would be ideal to move this logic into the Run function.
func (s *runner) serve(
	ctx context.Context,
	listener net.Listener,
	mux http.Handler,
) error {
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

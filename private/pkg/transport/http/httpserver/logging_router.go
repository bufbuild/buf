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
	"net/http"
	"path"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type loggingRouter struct {
	logger   *zap.Logger
	level    zapcore.Level
	delegate chi.Router
	prefix   string
}

func newLoggingRouter(
	logger *zap.Logger,
	level zapcore.Level,
	delegate chi.Router,
	prefix string,
) *loggingRouter {
	if delegateLoggingRouter, ok := delegate.(*loggingRouter); ok {
		delegate = delegateLoggingRouter.getDelegate()
	}
	return &loggingRouter{
		logger:   logger,
		level:    level,
		delegate: delegate,
		prefix:   prefix,
	}
}

func (r *loggingRouter) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	r.delegate.ServeHTTP(responseWriter, request)
}

func (r *loggingRouter) Routes() []chi.Route {
	return r.delegate.Routes()
}

func (r *loggingRouter) Middlewares() chi.Middlewares {
	return r.delegate.Middlewares()
}

func (r *loggingRouter) Match(rctx *chi.Context, method, path string) bool {
	return r.delegate.Match(rctx, method, path)
}

func (r *loggingRouter) Use(middlewares ...func(http.Handler) http.Handler) {
	r.delegate.Use(middlewares...)
}

func (r *loggingRouter) With(middlewares ...func(http.Handler) http.Handler) chi.Router {
	return newLoggingRouter(
		r.logger,
		r.level,
		r.delegate.With(middlewares...),
		r.prefix,
	)
}

func (r *loggingRouter) Group(fn func(r chi.Router)) chi.Router {
	return newLoggingRouter(
		r.logger,
		r.level,
		r.delegate.Group(
			func(router chi.Router) {
				fn(
					newLoggingRouter(
						r.logger,
						r.level,
						router,
						r.prefix,
					),
				)
			},
		),
		r.prefix,
	)
}

func (r *loggingRouter) Route(pattern string, fn func(r chi.Router)) chi.Router {
	return newLoggingRouter(
		r.logger,
		r.level,
		r.delegate.Route(
			pattern,
			func(router chi.Router) {
				fn(
					newLoggingRouter(
						r.logger,
						r.level,
						router,
						path.Join(r.prefix, pattern),
					),
				)
			},
		),
		path.Join(r.prefix, pattern),
	)
}

func (r *loggingRouter) Mount(pattern string, h http.Handler) {
	r.delegate.Mount(pattern, h)
	r.logRouteAdded("mount", pattern)
}

func (r *loggingRouter) Handle(pattern string, h http.Handler) {
	r.delegate.Handle(pattern, h)
	r.logRouteAdded("handle", pattern)
}

func (r *loggingRouter) HandleFunc(pattern string, h http.HandlerFunc) {
	r.delegate.HandleFunc(pattern, h)
	r.logRouteAdded("handle_func", pattern)
}

func (r *loggingRouter) Method(method, pattern string, h http.Handler) {
	r.delegate.Method(method, pattern, h)
	r.logRouteAdded("method_"+method, pattern)
}

func (r *loggingRouter) MethodFunc(method, pattern string, h http.HandlerFunc) {
	r.delegate.MethodFunc(method, pattern, h)
	r.logRouteAdded("method_func_"+method, pattern)
}

func (r *loggingRouter) Connect(pattern string, h http.HandlerFunc) {
	r.delegate.Connect(pattern, h)
	r.logRouteAdded("connect", pattern)
}

func (r *loggingRouter) Delete(pattern string, h http.HandlerFunc) {
	r.delegate.Delete(pattern, h)
	r.logRouteAdded("delete", pattern)
}

func (r *loggingRouter) Get(pattern string, h http.HandlerFunc) {
	r.delegate.Get(pattern, h)
	r.logRouteAdded("get", pattern)
}

func (r *loggingRouter) Head(pattern string, h http.HandlerFunc) {
	r.delegate.Head(pattern, h)
	r.logRouteAdded("head", pattern)
}

func (r *loggingRouter) Options(pattern string, h http.HandlerFunc) {
	r.delegate.Options(pattern, h)
	r.logRouteAdded("options", pattern)
}

func (r *loggingRouter) Patch(pattern string, h http.HandlerFunc) {
	r.delegate.Patch(pattern, h)
	r.logRouteAdded("patch", pattern)
}

func (r *loggingRouter) Post(pattern string, h http.HandlerFunc) {
	r.delegate.Post(pattern, h)
	r.logRouteAdded("post", pattern)
}

func (r *loggingRouter) Put(pattern string, h http.HandlerFunc) {
	r.delegate.Put(pattern, h)
	r.logRouteAdded("put", pattern)
}

func (r *loggingRouter) Trace(pattern string, h http.HandlerFunc) {
	r.delegate.Trace(pattern, h)
	r.logRouteAdded("trace", pattern)
}

func (r *loggingRouter) NotFound(h http.HandlerFunc) {
	r.delegate.NotFound(h)
	r.logRouteAdded("__not_found__", "__global__")
}

func (r *loggingRouter) MethodNotAllowed(h http.HandlerFunc) {
	r.delegate.MethodNotAllowed(h)
	r.logRouteAdded("__method_not_allowed__", "__global__")
}

func (r *loggingRouter) getDelegate() chi.Router {
	return r.delegate
}

func (r *loggingRouter) logRouteAdded(f string, route string) {
	if checkedEntry := r.logger.Check(r.level, "route_added"); checkedEntry != nil {
		checkedEntry.Write(
			zap.String("route", path.Join(r.prefix, route)),
			zap.String("function", f),
		)
	}
}

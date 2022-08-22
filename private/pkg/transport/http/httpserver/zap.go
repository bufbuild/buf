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
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func newZapMiddleware(logger *zap.Logger, silentEndpoints []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			logEntry := newLogEntry(logger, request)
			wrapResponseWriter := middleware.NewWrapResponseWriter(newCheckedResponseWriter(logger, writer, request), request.ProtoMajor)
			if !contains(silentEndpoints, request.URL.Path) {
				defer logRequest(logEntry, wrapResponseWriter, time.Now())
			}
			next.ServeHTTP(wrapResponseWriter, middleware.WithLogEntry(request, logEntry))
		})
	}
}

func contains(l []string, s string) bool {
	for _, v := range l {
		if v == s {
			return true
		}
	}
	return false
}

func logRequest(
	logEntry middleware.LogEntry,
	wrapResponseWriter middleware.WrapResponseWriter,
	start time.Time,
) {
	logEntry.Write(
		wrapResponseWriter.Status(),
		wrapResponseWriter.BytesWritten(),
		nil,
		time.Since(start),
		nil,
	)
}

type logEntry struct {
	logger  *zap.Logger
	request *http.Request
}

func newLogEntry(logger *zap.Logger, request *http.Request) *logEntry {
	return &logEntry{
		logger:  logger,
		request: request,
	}
}

func (l *logEntry) Write(status int, size int, _ http.Header, duration time.Duration, _ interface{}) {
	fields := append(
		getRequestFields(l.request),
		zap.Int("status", status),
		zap.Int("size", size),
		zap.Duration("duration", duration),
	)
	l.logger.Info("request", fields...)
}

func (l *logEntry) Panic(value interface{}, stack []byte) {
	l.logger.Error(
		"request_panic",
		append(
			getRequestFields(l.request),
			zap.Any("value", value),
			zap.String("stack", string(stack)),
		)...,
	)
}

func getRequestFields(request *http.Request) []zap.Field {
	return []zap.Field{
		zap.String("proto", request.Proto),
		zap.String("method", request.Method),
		zap.String("host", request.Host),
		zap.String("path", request.RequestURI),
	}
}

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
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// includes fields described in https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry#httprequest
type httpRequestLog struct {
	requestMethod string
	requestUrl    string
	status        int
	responseSize  string
	userAgent     string
	remoteIp      string
	serverIp      string
	latency       string
	protocol      string
}

func (h *httpRequestLog) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("requestMethod", h.requestMethod)
	enc.AddString("requestUrl", h.requestUrl)
	enc.AddInt("status", h.status)
	enc.AddString("responseSize", h.responseSize)
	enc.AddString("userAgent", h.userAgent)
	enc.AddString("remoteIP", h.remoteIp)
	enc.AddString("latency", h.latency)
	enc.AddString("protocol", h.protocol)

	return nil
}

func newZapMiddleware(logger *zap.Logger, silentEndpoints map[string]struct{}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			logEntry := newLogEntry(logger, request)
			wrapResponseWriter := middleware.NewWrapResponseWriter(newCheckedResponseWriter(logger, writer, request), request.ProtoMajor)
			if _, ok := silentEndpoints[request.URL.Path]; !ok {
				defer logRequest(logEntry, wrapResponseWriter, time.Now())
			}
			next.ServeHTTP(wrapResponseWriter, middleware.WithLogEntry(request, logEntry))
		})
	}
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
	httpField := zap.Object(
		"httpRequest",
		newHttpRequestLog(l.request, status, size, duration),
	)
	fields := append(
		getTopLevelRequestFields(l.request),
		httpField,
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

// this gets request fields that are relevant but not covered by http request log
func getTopLevelRequestFields(request *http.Request) []zap.Field {
	return []zap.Field{
		zap.String("host", request.Host),
		zap.String("path", request.RequestURI),
	}
}

func newHttpRequestLog(r *http.Request, status int, responseSize int, duration time.Duration) *httpRequestLog {
	return &httpRequestLog{
		requestMethod: r.Method,
		requestUrl:    getFullUrl(r),
		status:        status,
		responseSize:  fmt.Sprintf("%v", responseSize),
		userAgent:     r.UserAgent(),
		remoteIp:      r.RemoteAddr,
		latency:       fmt.Sprintf("%fs", duration.Seconds()),
		protocol:      r.Proto,
	}
}

func getFullUrl(r *http.Request) string {
	if r.URL.IsAbs() {
		return r.URL.String()
	}
	return fmt.Sprintf("%v%v", r.Host, r.URL)
}

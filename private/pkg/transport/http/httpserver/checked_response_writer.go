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

	"go.uber.org/zap"
)

type checkedResponseWriter struct {
	logger         *zap.Logger
	responseWriter http.ResponseWriter
	request        *http.Request
}

func newCheckedResponseWriter(
	logger *zap.Logger,
	responseWriter http.ResponseWriter,
	request *http.Request,
) *checkedResponseWriter {
	return &checkedResponseWriter{
		logger:         logger,
		responseWriter: responseWriter,
		request:        request,
	}
}

func (c *checkedResponseWriter) Header() http.Header {
	return c.responseWriter.Header()
}

func (c *checkedResponseWriter) Write(data []byte) (int, error) {
	n, err := c.responseWriter.Write(data)
	if err != nil {
		c.logger.Error(
			"write_error",
			append(
				getRequestFields(c.request),
				zap.Error(err),
			)...,
		)
	}
	if n != len(data) {
		c.logger.Error(
			"write_incomplete",
			append(
				getRequestFields(c.request),
				zap.Int("expected_length", len(data)),
				zap.Int("actual_length", n),
			)...,
		)
	}
	return n, err
}

func (c *checkedResponseWriter) WriteHeader(statusCode int) {
	c.responseWriter.WriteHeader(statusCode)
}

func (c *checkedResponseWriter) Flush() {
	if f, ok := c.responseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

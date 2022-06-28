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
	"net/http"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufconnect"
)

const (
	keyPrefix = "rpc-"
)

type incomingHeadersContextKey struct{}

// WithIncomingHeaders adds the given headers to the context, which are simple key/value with no differentiation between unset and nil.
//
// If headers is empty or nil, this is a no-op.
// If a key or value is empty, this is a no-op for that key.
// If a key was already set, this will overwrite the value for the key.
func withIncomingHeaders(ctx context.Context, headers map[string]string) context.Context {
	if updatedHeaders := withHeaders(ctx.Value(incomingHeadersContextKey{}), headers); len(updatedHeaders) != 0 {
		return context.WithValue(ctx, incomingHeadersContextKey{}, updatedHeaders)
	}
	return ctx
}

func normalizeHeaderKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}

// Returns nil if there is no modification.
func withHeaders(contextValue interface{}, headers map[string]string) map[string]string {
	var m map[string]string
	for key, value := range headers {
		if value != "" {
			if normalizedKey := normalizeHeaderKey(key); normalizedKey != "" {
				if m == nil {
					m = newHeaderMap(contextValue, len(headers))
				}
				m[normalizedKey] = value
			}
		}
	}
	return m
}

func newHeaderMap(contextValue interface{}, additionalLen int) map[string]string {
	if contextValue == nil {
		return make(map[string]string, additionalLen)
	}
	existing, ok := contextValue.(map[string]string)
	if !ok {
		return make(map[string]string, additionalLen)
	}
	m := make(map[string]string, len(existing)+additionalLen)
	for key, value := range existing {
		m[key] = value
	}
	return m
}

func fromHTTPHeader(httpHeader http.Header) map[string]string {
	headers := make(map[string]string)
	for key, values := range httpHeader {
		// Convert to lowercase and then trim the keyPrefix if it exists
		key = strings.TrimPrefix(strings.ToLower(key), keyPrefix)
		// If this header is one of the auth token header or CLI version, add it to the returned map
		if key == bufconnect.AuthenticationHeader || key == bufconnect.CliVersionHeaderName {
			headers[key] = values[0]
		}
	}
	return headers
}

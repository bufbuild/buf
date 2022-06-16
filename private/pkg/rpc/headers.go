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

package rpc

import (
	"context"
	"strings"
)

const (
	// AuthenticationHeader is the standard OAuth header used for authenticating
	// a user. Ignore the misnomer.
	AuthenticationHeader = "Authorization"
	// AuthenticationTokenPrefix is the standard OAuth token prefix.
	// We use it for familiarity.
	AuthenticationTokenPrefix = "Bearer "
	// KeyPrefix is the prefix of headers set with rpc packages.
	KeyPrefix = "rpc-"
)

type outgoingHeadersContextKey struct{}
type incomingHeadersContextKey struct{}

// GetIncomingContextHeader gets the given header key from an incoming context.
//
// Headers are simple key/value with no differentiation between unset and nil and are case-insensitive.
//
// If the key is unset, this returns the empty string.
func GetIncomingContextHeader(ctx context.Context, key string) string {
	if contextValue := ctx.Value(incomingHeadersContextKey{}); contextValue != nil {
		value, ok := contextValue.(map[string]string)[normalizeHeaderKey(key)]
		if !ok {
			// Unreachable
			return ""
		}
		return value
	}
	return ""
}

// GetOutgoingContextHeader gets the given header key from an outgoing context.
//
// Headers are simple key/value with no differentiation between unset and nil and are case-insensitive.
//
// If the key is unset, this returns the empty string.
func GetOutgoingContextHeader(ctx context.Context, key string) string {
	if contextValue := ctx.Value(outgoingHeadersContextKey{}); contextValue != nil {
		value, ok := contextValue.(map[string]string)[normalizeHeaderKey(key)]
		if !ok {
			// Unreachable
			return ""
		}
		return value
	}
	return ""
}

// GetIncomingContextHeaders gets the headers from an incoming context
//
// Headers are simple key/value with no differentiation between unset and nil and are case-insensitive.
//
// If there are no headers, returns an empty map.
func GetIncomingContextHeaders(ctx context.Context) map[string]string {
	return newHeaderMap(ctx.Value(incomingHeadersContextKey{}), 0)
}

// GetOutgoingContextHeaders gets the headers from an outgoing context
//
// Headers are simple key/value with no differentiation between unset and nil and are case-insensitive.
//
// If there are no headers, returns an empty map.
func GetOutgoingContextHeaders(ctx context.Context) map[string]string {
	return newHeaderMap(ctx.Value(outgoingHeadersContextKey{}), 0)
}

// WithIncomingContextHeader adds the given header to an incoming context.
//
// Headers are simple key/value with no differentiation between unset and nil and are case-insensitive.
//
// If the key or value is empty, this is a no-op.
// If the key was already set, this will overwrite the value for the key.
func WithIncomingContextHeader(ctx context.Context, key string, value string) context.Context {
	if updatedHeaders := withHeader(ctx.Value(incomingHeadersContextKey{}), key, value); len(updatedHeaders) != 0 {
		return context.WithValue(ctx, incomingHeadersContextKey{}, updatedHeaders)
	}
	return ctx
}

// WithOutgoingContextHeader adds the given header to the outgoing context.
//
// Headers are simple key/value with no differentiation between unset and nil and are case-insensitive.
//
// If the key or value is empty, this is a no-op.
// If the key was already set, this will overwrite the value for the key.
func WithOutgoingContextHeader(ctx context.Context, key string, value string) context.Context {
	if updatedHeaders := withHeader(ctx.Value(outgoingHeadersContextKey{}), key, value); len(updatedHeaders) != 0 {
		return context.WithValue(ctx, outgoingHeadersContextKey{}, updatedHeaders)
	}
	return ctx
}

// WithIncomingContextHeaders adds the given headers to the incoming context.
//
// Headers are simple key/value with no differentiation between unset and nil and are case-insensitive.
//
// If headers is empty or nil, this is a no-op.
// If a key or value is empty, this is a no-op for that key.
// If a key was already set, this will overwrite the value for the key.
func WithIncomingContextHeaders(ctx context.Context, headers map[string]string) context.Context {
	if updatedHeaders := withHeaders(ctx.Value(incomingHeadersContextKey{}), headers); len(updatedHeaders) != 0 {
		return context.WithValue(ctx, incomingHeadersContextKey{}, updatedHeaders)
	}
	return ctx
}

// WithOutgoingContextHeaders adds the given headers to the outgoing context.
//
// Headers are simple key/value with no differentiation between unset and nil and are case-insensitive.
//
// If headers is empty or nil, this is a no-op.
// If a key or value is empty, this is a no-op for that key.
// If a key was already set, this will overwrite the value for the key.
func WithOutgoingContextHeaders(ctx context.Context, headers map[string]string) context.Context {
	if updatedHeaders := withHeaders(ctx.Value(outgoingHeadersContextKey{}), headers); len(updatedHeaders) != 0 {
		return context.WithValue(ctx, outgoingHeadersContextKey{}, updatedHeaders)
	}
	return ctx
}

// StripHeaderPrefixes strips the KeyPrefix from any of the header keys.
func StripHeaderPrefixes(headers map[string]string) map[string]string {
	out := make(map[string]string)
	for key, value := range headers {
		key = strings.ToLower(key)
		if !strings.HasPrefix(key, KeyPrefix) {
			out[key] = value
		}
		trimmedKey := strings.TrimPrefix(key, KeyPrefix)
		if trimmedKey == "" {
			out[key] = value
		}
		out[trimmedKey] = value
	}
	return out
}

func normalizeHeaderKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}

// Returns nil if there is no modification.
func withHeader(contextValue interface{}, key string, value string) map[string]string {
	if value != "" {
		if normalizedKey := normalizeHeaderKey(key); normalizedKey != "" {
			m := newHeaderMap(contextValue, 1)
			m[normalizedKey] = value
			return m
		}
	}
	return nil
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

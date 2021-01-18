// Copyright 2020-2021 Buf Technologies, Inc.
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

type outgoingHeadersContextKey struct{}
type incomingHeadersContextKey struct{}

// GetIncomingHeader gets the given header key.
//
// Headers are simple key/value with no differentiation between unset and nil.
// This is as opposed to i.e. grpc that does key/slice value with differentiation between unset and nil.
// Headers are case-insensitive.
//
// If the key is unset, this returns the empty string.
func GetIncomingHeader(ctx context.Context, key string) string {
	if contextValue := ctx.Value(incomingHeadersContextKey{}); contextValue != nil {
		return contextValue.(map[string]string)[normalizeHeaderKey(key)]
	}
	return ""
}

// GetOutgoingHeader gets the given header key.
//
// Headers are simple key/value with no differentiation between unset and nil.
// This is as opposed to i.e. grpc that does key/slice value with differentiation between unset and nil.
// Headers are case-insensitive.
//
// If the key is unset, this returns the empty string.
func GetOutgoingHeader(ctx context.Context, key string) string {
	if contextValue := ctx.Value(outgoingHeadersContextKey{}); contextValue != nil {
		return contextValue.(map[string]string)[normalizeHeaderKey(key)]
	}
	return ""
}

// GetIncomingHeaders gets the headers..
//
// Headers are simple key/value with no differentiation between unset and nil.
// This is as opposed to i.e. grpc that does key/slice value with differentiation between unset and nil.
// Headers are case-insensitive.
//
// If there are no headers, returns nil
func GetIncomingHeaders(ctx context.Context) map[string]string {
	if contextValue := ctx.Value(incomingHeadersContextKey{}); contextValue != nil {
		headers := contextValue.(map[string]string)
		headersCopy := make(map[string]string, len(headers))
		for key, value := range headers {
			headersCopy[key] = value
		}
		return headersCopy
	}
	return nil
}

// GetOutgoingHeaders gets the headers..
//
// Headers are simple key/value with no differentiation between unset and nil.
// This is as opposed to i.e. grpc that does key/slice value with differentiation between unset and nil.
// Headers are case-insensitive.
//
// If there are no headers, returns nil
func GetOutgoingHeaders(ctx context.Context) map[string]string {
	if contextValue := ctx.Value(outgoingHeadersContextKey{}); contextValue != nil {
		headers := contextValue.(map[string]string)
		headersCopy := make(map[string]string, len(headers))
		for key, value := range headers {
			headersCopy[key] = value
		}
		return headersCopy
	}
	return nil
}

// WithIncomingHeader adds the given header to the context.
//
// Headers are simple key/value with no differentiation between unset and nil.
// This is as opposed to i.e. grpc that does key/slice value with differentiation between unset and nil.
// Headers are case-insensitive.
//
// If the key or value is empty, this is a no-op.
// If the key was already set, this will overwrite the value for the key.
//
// This should generally
func WithIncomingHeader(ctx context.Context, key string, value string) context.Context {
	return WithIncomingHeaders(ctx, map[string]string{key: value})
}

// WithOutgoingHeader adds the given header to the context.
//
// Headers are simple key/value with no differentiation between unset and nil.
// This is as opposed to i.e. grpc that does key/slice value with differentiation between unset and nil.
// Headers are case-insensitive.
//
// If the key or value is empty, this is a no-op.
// If the key was already set, this will overwrite the value for the key.
func WithOutgoingHeader(ctx context.Context, key string, value string) context.Context {
	return WithOutgoingHeaders(ctx, map[string]string{key: value})
}

// WithOutgoingHeaders adds the given headers to the context.
//
// Headers are simple key/value with no differentiation between unset and nil.
// This is as opposed to i.e. grpc that does key/slice value with differentiation between unset and nil.
// Headers are case-insensitive.
//
// If headers is empty or nil, this is a no-op.
// If a key or value is empty, this is a no-op for that key.
// If a key was already set, this will overwrite the value for the key.
func WithOutgoingHeaders(ctx context.Context, headers map[string]string) context.Context {
	if updatedHeaders := updateHeaders(ctx.Value(outgoingHeadersContextKey{}), headers); len(updatedHeaders) != 0 {
		return context.WithValue(ctx, outgoingHeadersContextKey{}, updatedHeaders)
	}
	return ctx
}

// WithIncomingHeaders adds the given headers to the context.
//
// Headers are simple key/value with no differentiation between unset and nil.
// This is as opposed to i.e. grpc that does key/slice value with differentiation between unset and nil.
// Headers are case-insensitive.
//
// If headers is empty or nil, this is a no-op.
// If a key or value is empty, this is a no-op for that key.
// If a key was already set, this will overwrite the value for the key.
func WithIncomingHeaders(ctx context.Context, headers map[string]string) context.Context {
	if updatedHeaders := updateHeaders(ctx.Value(incomingHeadersContextKey{}), headers); len(updatedHeaders) != 0 {
		return context.WithValue(ctx, incomingHeadersContextKey{}, updatedHeaders)
	}
	return ctx
}

func normalizeHeaderKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}

func updateHeaders(contextValue interface{}, headers map[string]string) map[string]string {
	m := make(map[string]string)
	// Explicitly copy existing contextValue to avoid mutating parent context.
	if contextValue != nil {
		existing := contextValue.(map[string]string)
		for key, value := range existing {
			m[key] = value
		}
	}
	for key, value := range headers {
		if value != "" {
			if normalizedKey := normalizeHeaderKey(key); normalizedKey != "" {
				m[normalizedKey] = value
			}
		}
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

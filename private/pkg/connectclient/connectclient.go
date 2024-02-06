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

package connectclient

import (
	"connectrpc.com/connect"
)

// TODO FUTURE: eliminate config, just make the below MakeOptions, add httpClient as a parameter to Make

// Config holds configuration for creating Connect RPC clients.
type Config struct {
	httpClient              connect.HTTPClient
	addressMapper           func(string) string
	interceptors            []connect.Interceptor
	authInterceptorProvider func(string) connect.UnaryInterceptorFunc
}

// NewConfig creates a new client configuration with the given HTTP client
// and options.
func NewConfig(httpClient connect.HTTPClient, options ...ConfigOption) *Config {
	cfg := &Config{
		httpClient: httpClient,
	}
	for _, opt := range options {
		opt(cfg)
	}
	return cfg
}

// ConfigOption is an option for customizing a new Config.
type ConfigOption func(*Config)

// WithAddressMapper maps the address with the given function.
func WithAddressMapper(addressMapper func(string) string) ConfigOption {
	return func(cfg *Config) {
		cfg.addressMapper = addressMapper
	}
}

// WithInterceptors adds the slice of interceptors to all clients returned from this provider.
func WithInterceptors(interceptors []connect.Interceptor) ConfigOption {
	return func(cfg *Config) {
		cfg.interceptors = interceptors
	}
}

// WithAuthInterceptorProvider configures a provider that, when invoked, returns an interceptor that can be added
// to a client for setting the auth token
func WithAuthInterceptorProvider(authInterceptorProvider func(string) connect.UnaryInterceptorFunc) ConfigOption {
	return func(cfg *Config) {
		cfg.authInterceptorProvider = authInterceptorProvider
	}
}

// StubFactory is the type of a generated factory function, for creating Connect client stubs.
type StubFactory[T any] func(connect.HTTPClient, string, ...connect.ClientOption) T

// Make uses the given generated factory function to create a new connect client.
func Make[T any](cfg *Config, address string, factory StubFactory[T]) T {
	interceptors := append([]connect.Interceptor{}, cfg.interceptors...)
	if cfg.authInterceptorProvider != nil {
		interceptor := cfg.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if cfg.addressMapper != nil {
		address = cfg.addressMapper(address)
	}
	return factory(cfg.httpClient, address, connect.WithInterceptors(interceptors...))
}

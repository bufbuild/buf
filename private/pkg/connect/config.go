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

package connect

import (
	connect_go "github.com/bufbuild/connect-go"
)

type ClientConfig struct {
	httpClient              connect_go.HTTPClient
	addressMapper           func(string) string
	interceptors            []connect_go.Interceptor
	authInterceptorProvider func(string) connect_go.UnaryInterceptorFunc
}

func NewClientConfig(httpClient connect_go.HTTPClient, options ...ClientConfigOption) *ClientConfig {
	cfg  := ClientConfig{
		httpClient: httpClient,
	}
	for _, opt := range options {
		opt(&cfg)
	}
	return &cfg
}

// ClientConfigOption is an option for a new ClentConfig.
type ClientConfigOption func(*ClientConfig)

// WithAddressMapper maps the address with the given function.
func WithAddressMapper(addressMapper func(string) string) ClientConfigOption {
	return func(cfg *ClientConfig) {
		cfg.addressMapper = addressMapper
	}
}

// WithInterceptors adds the slice of interceptors to all clients returned from this provider.
func WithInterceptors(interceptors []connect_go.Interceptor) ClientConfigOption {
	return func(cfg *ClientConfig) {
		cfg.interceptors = interceptors
	}
}

// WithAuthInterceptorProvider configures a provider that, when invoked, returns an interceptor that can be added
// to a client for setting the auth token
func WithAuthInterceptorProvider(authInterceptorProvider func(string) connect_go.UnaryInterceptorFunc) ClientConfigOption {
	return func(cfg *ClientConfig) {
		cfg.authInterceptorProvider = authInterceptorProvider
	}
}

// ClientFactory is the type of a generated factory function, for creating Connect client stubs.
type ClientFactory[T any] func(connect_go.HTTPClient, string, ...connect_go.ClientOption) T

// MakeClient uses the given generated factory function to create a new connect client.
func MakeClient[T any](cfg *ClientConfig, address string, factory ClientFactory[T]) T {
	interceptors := append([]connect_go.Interceptor{}, cfg.interceptors...)
	if cfg.authInterceptorProvider != nil {
		interceptor := cfg.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if cfg.addressMapper != nil {
		address = cfg.addressMapper(address)
	}
	return factory(cfg.httpClient, address, connect_go.WithInterceptors(interceptors...))
}

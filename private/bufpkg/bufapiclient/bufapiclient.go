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

// Package bufapiclient provides client-side Connect constructs.
package bufapiclient

import (
	"net/http"

	"github.com/bufbuild/buf/private/gen/proto/apiclient/buf/alpha/registry/v1alpha1/registryv1alpha1apiclient"
	"github.com/bufbuild/buf/private/gen/proto/apiclientconnect/buf/alpha/registry/v1alpha1/registryv1alpha1apiclientconnect"
	"github.com/bufbuild/connect-go"
	"go.uber.org/zap"
)

// NewConnectClientProvider creates a new Provider using Connect as its underlying transport.
func NewConnectClientProvider(
	logger *zap.Logger,
	client *http.Client,
	options ...RegistryProviderOption,
) (registryv1alpha1apiclient.Provider, error) {
	registryProviderOptions := &registryProviderOptions{}
	for _, option := range options {
		option(registryProviderOptions)
	}
	providerOptions := []registryv1alpha1apiclientconnect.ProviderOption{
		registryv1alpha1apiclientconnect.WithAddressMapper(registryProviderOptions.addressMapper),
		registryv1alpha1apiclientconnect.WithInterceptors(registryProviderOptions.interceptors),
		registryv1alpha1apiclientconnect.WithAuthInterceptorProvider(registryProviderOptions.authInterceptorProvider),
	}
	return registryv1alpha1apiclientconnect.NewProvider(
		logger,
		client,
		providerOptions...,
	), nil
}

// RegistryProviderOption is an option for a new registry Provider.
type RegistryProviderOption func(*registryProviderOptions)

// interceptorProviderFunc is a type that represents a function that returns an interceptor using a given address
type interceptorProviderFunc func(string) connect.UnaryInterceptorFunc

type registryProviderOptions struct {
	addressMapper           func(string) string
	interceptors            []connect.Interceptor
	authInterceptorProvider interceptorProviderFunc
}

// RegistryProviderWithAddressMapper returns a new RegistryProviderOption that maps
// addresses with the given function.
func RegistryProviderWithAddressMapper(addressMapper func(string) string) RegistryProviderOption {
	return func(options *registryProviderOptions) {
		options.addressMapper = addressMapper
	}
}

// RegistryProviderWithInterceptors adds the provided interceptors to all clients returned from the provider
func RegistryProviderWithInterceptors(interceptors ...connect.Interceptor) RegistryProviderOption {
	return func(options *registryProviderOptions) {
		options.interceptors = interceptors
	}
}

// RegistryProviderWithAuthInterceptorProvider adds the given interceptor provider, which when invoked with an address
// will return an interceptor that looks up an auth token for the provided address
func RegistryProviderWithAuthInterceptorProvider(authInterceptorProvider interceptorProviderFunc) RegistryProviderOption {
	return func(options *registryProviderOptions) {
		options.authInterceptorProvider = authInterceptorProvider
	}
}

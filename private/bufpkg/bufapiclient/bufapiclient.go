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
	"context"
	"net/http"

	"github.com/bufbuild/buf/private/gen/proto/apiclient/buf/alpha/registry/v1alpha1/registryv1alpha1apiclient"
	"github.com/bufbuild/buf/private/gen/proto/apiclientconnect/buf/alpha/registry/v1alpha1/registryv1alpha1apiclientconnect"
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
		registryv1alpha1apiclientconnect.WithContextModifierProvider(registryProviderOptions.contextModifierProvider),
	}
	if registryProviderOptions.withGRPC {
		providerOptions = append(providerOptions, registryv1alpha1apiclientconnect.WithGRPC())
	}
	return registryv1alpha1apiclientconnect.NewProvider(
		logger,
		client,
		providerOptions...,
	), nil
}

// RegistryProviderOption is an option for a new registry Provider.
type RegistryProviderOption func(*registryProviderOptions)

type registryProviderOptions struct {
	addressMapper           func(string) string
	contextModifierProvider func(string) (func(context.Context) context.Context, error)
	withGRPC                bool
}

// RegistryProviderWithAddressMapper returns a new RegistryProviderOption that maps
// addresses with the given function.
func RegistryProviderWithAddressMapper(addressMapper func(string) string) RegistryProviderOption {
	return func(options *registryProviderOptions) {
		options.addressMapper = addressMapper
	}
}

// RegistryProviderWithContextModifierProvider returns a new RegistryProviderOption that
// creates a context modifier for a given address. This is used to modify the context
// before every RPC invocation.
func RegistryProviderWithContextModifierProvider(contextModifierProvider func(address string) (func(context.Context) context.Context, error)) RegistryProviderOption {
	return func(options *registryProviderOptions) {
		options.contextModifierProvider = contextModifierProvider
	}
}

// RegistryProviderWithGRPC returns a new RegistryProviderOption that allows for configuration of the underlying transport
// of a provider.  Using this option sets the transport to gRPC while omitting it, defaults to using Connect.
func RegistryProviderWithGRPC() RegistryProviderOption {
	return func(options *registryProviderOptions) {
		options.withGRPC = true
	}
}

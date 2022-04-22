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

// Code generated by protoc-gen-go-apiclientgrpc. DO NOT EDIT.

package registryv1alpha2apiclientgrpc

import (
	context "context"
	registryv1alpha2api "github.com/bufbuild/buf/private/gen/proto/api/buf/alpha/registry/v1alpha2/registryv1alpha2api"
	registryv1alpha2apiclient "github.com/bufbuild/buf/private/gen/proto/apiclient/buf/alpha/registry/v1alpha2/registryv1alpha2apiclient"
	v1alpha2 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha2"
	grpcclient "github.com/bufbuild/buf/private/pkg/transport/grpc/grpcclient"
	zap "go.uber.org/zap"
)

// NewProvider returns a new Provider.
func NewProvider(
	logger *zap.Logger,
	clientConnProvider grpcclient.ClientConnProvider,
	options ...ProviderOption,
) registryv1alpha2apiclient.Provider {
	provider := &provider{
		logger:             logger,
		clientConnProvider: clientConnProvider,
	}
	for _, option := range options {
		option(provider)
	}
	return provider
}

type provider struct {
	logger                  *zap.Logger
	clientConnProvider      grpcclient.ClientConnProvider
	addressMapper           func(string) string
	contextModifierProvider func(string) (func(context.Context) context.Context, error)
}

// ProviderOption is an option for a new Provider.
type ProviderOption func(*provider)

// WithAddressMapper maps the address with the given function.
func WithAddressMapper(addressMapper func(string) string) ProviderOption {
	return func(provider *provider) {
		provider.addressMapper = addressMapper
	}
}

// WithContextModifierProvider provides a function that  modifies the context before every RPC invocation.
// Applied before the address mapper.
func WithContextModifierProvider(contextModifierProvider func(address string) (func(context.Context) context.Context, error)) ProviderOption {
	return func(provider *provider) {
		provider.contextModifierProvider = contextModifierProvider
	}
}

func (p *provider) NewRemoteModuleService(ctx context.Context, address string) (registryv1alpha2api.RemoteModuleService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(address)
		if err != nil {
			return nil, err
		}
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	clientConn, err := p.clientConnProvider.NewClientConn(ctx, address)
	if err != nil {
		return nil, err
	}
	return &remoteModuleService{
		logger:          p.logger,
		client:          v1alpha2.NewRemoteModuleServiceClient(clientConn),
		contextModifier: contextModifier,
	}, nil
}

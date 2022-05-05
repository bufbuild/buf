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

// Package bufapiclient provides client-side gRPC constructs.
package bufapiclient

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"

	"github.com/bufbuild/buf/private/gen/proto/apiclient/buf/alpha/registry/v1alpha1/registryv1alpha1apiclient"
	"github.com/bufbuild/buf/private/gen/proto/apiclientgrpc/buf/alpha/registry/v1alpha1/registryv1alpha1apiclientgrpc"
	"github.com/bufbuild/buf/private/pkg/transport/grpc/grpcclient"
	"github.com/bufbuild/buf/private/pkg/transport/http/httpclient"
	"github.com/bufbuild/buf/private/pkg/transport/http2client"
	"go.uber.org/zap"
)

func newGRPCClientProvider(
	ctx context.Context,
	logger *zap.Logger,
	tlsConfig *tls.Config,
	options []RegistryProviderOption,
) (registryv1alpha1apiclient.Provider, error) {
	registryProviderOptions := &registryProviderOptions{}
	for _, option := range options {
		option(registryProviderOptions)
	}
	clientConnProvider, err := NewGRPCClientConnProvider(ctx, logger, tlsConfig)
	if err != nil {
		return nil, err
	}
	return registryv1alpha1apiclientgrpc.NewProvider(
		logger,
		clientConnProvider,
		registryv1alpha1apiclientgrpc.WithAddressMapper(registryProviderOptions.addressMapper),
		registryv1alpha1apiclientgrpc.WithContextModifierProvider(registryProviderOptions.contextModifierProvider),
	), nil
}

// NewRegistryProvider creates a new registryv1alpha1apiclient.Provider gRPC.
//
// If tlsConfig is nil, no TLS is used.
func NewRegistryProvider(
	ctx context.Context,
	logger *zap.Logger,
	tlsConfig *tls.Config,
	options ...RegistryProviderOption,
) (registryv1alpha1apiclient.Provider, error) {
	if os.Getenv("USE_CONNECT") == "1" {
		// Initialize a connect client
		_ = http2client.NewClient()
		// this isust here for an example
		// authnService = registryv1alpha1connectclient.NewAuthnServiceClient(
		// 	h2cClient,
		// 	"https://"+buftransport.PrependAPISubdomain(remote),
		// 	connect.WithGRPC(),
		// )
		//  Replace the grpcclient provider with this  new connect provider when the plugin is done
		// return registryv1alphaapiclientconnect.NewProvider(h2cClient, remote, options), nil
		return newGRPCClientProvider(ctx, logger, tlsConfig, options)

	} else {
		return newGRPCClientProvider(ctx, logger, tlsConfig, options)
	}
}

// RegistryProviderOption is an option for a new registry Provider.
type RegistryProviderOption func(*registryProviderOptions)

type registryProviderOptions struct {
	addressMapper           func(string) string
	contextModifierProvider func(string) (func(context.Context) context.Context, error)
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

// NewGRPCClientConnProvider returns a new gRPC ClientConnProvider.
//
// TODO: move this to another location.
func NewGRPCClientConnProvider(
	ctx context.Context,
	logger *zap.Logger,
	tlsConfig *tls.Config,
) (grpcclient.ClientConnProvider, error) {
	return grpcclient.NewClientConnProvider(
		ctx,
		logger,
		grpcclient.ClientConnProviderWithTLSConfig(
			tlsConfig,
		),
		grpcclient.ClientConnProviderWithObservability(),
		grpcclient.ClientConnProviderWithGZIPCompression(),
	)
}

// NewHTTPClient returns a new HTTP Client.
//
// TODO: move this to another location.
func NewHTTPClient(
	tlsConfig *tls.Config,
) httpclient.Client {
	return httpclient.NewClient(
		httpclient.ClientWithTLSConfig(
			tlsConfig,
		),
		httpclient.ClientWithObservability(),
	)
}

// NewHTTP2Client returns a new HTTP/2 Client.
func NewHTTP2Client() *http.Client {
	return http2client.NewClient(
		http2client.WithObservability(),
	)
}

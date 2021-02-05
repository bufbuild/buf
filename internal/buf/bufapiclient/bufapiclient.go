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

// Package bufapiclient switches between grpc and twirp on the client-side.
package bufapiclient

import (
	"context"
	"crypto/tls"

	"github.com/bufbuild/buf/internal/gen/proto/apiclient/buf/alpha/registry/v1alpha1/registryv1alpha1apiclient"
	"github.com/bufbuild/buf/internal/gen/proto/apiclientgrpc/buf/alpha/registry/v1alpha1/registryv1alpha1apiclientgrpc"
	"github.com/bufbuild/buf/internal/gen/proto/apiclienttwirp/buf/alpha/registry/v1alpha1/registryv1alpha1apiclienttwirp"
	"github.com/bufbuild/buf/internal/pkg/transport/grpc/grpcclient"
	"github.com/bufbuild/buf/internal/pkg/transport/http/httpclient"
	"go.uber.org/zap"
)

// NewRegistryProvider creates a new registryv1alpha1apiclient.Provider for either grpc or twirp.
//
// If tlsConfig is nil, no TLS is used.
func NewRegistryProvider(
	ctx context.Context,
	logger *zap.Logger,
	tlsConfig *tls.Config,
	useGRPC bool,
) (registryv1alpha1apiclient.Provider, error) {
	if useGRPC {
		clientConnProvider, err := NewGRPCClientConnProvider(ctx, logger, tlsConfig)
		if err != nil {
			return nil, err
		}
		return registryv1alpha1apiclientgrpc.NewProvider(logger, clientConnProvider), nil
	}
	httpClient, err := NewHTTPClient(tlsConfig)
	if err != nil {
		return nil, err
	}
	return registryv1alpha1apiclienttwirp.NewProvider(logger, httpClient), nil
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
	)
}

// NewHTTPClient returns a new HTTP Client.
//
// TODO: move this to another location.
func NewHTTPClient(
	tlsConfig *tls.Config,
) (httpclient.Client, error) {
	return httpclient.NewClient(
		httpclient.ClientWithTLSConfig(
			tlsConfig,
		),
		httpclient.ClientWithObservability(),
	)
}

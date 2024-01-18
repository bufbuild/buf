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

// Package bufapimodule provides bufmodule types based on bufapi types.
package bufapimodule

import (
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"go.uber.org/zap"
)

type DownloadServiceClientFactory func(address string) registryv1alpha1connect.DownloadServiceClient
type RepositoryCommitServiceClientFactory func(address string) registryv1alpha1connect.RepositoryCommitServiceClient
type RepositoryServiceClientFactory func(address string) registryv1alpha1connect.RepositoryServiceClient

func NewDownloadServiceClientFactory(clientConfig *connectclient.Config) DownloadServiceClientFactory {
	return func(address string) registryv1alpha1connect.DownloadServiceClient {
		return connectclient.Make(clientConfig, address, registryv1alpha1connect.NewDownloadServiceClient)
	}
}

func NewRepositoryCommitServiceClientFactory(clientConfig *connectclient.Config) RepositoryCommitServiceClientFactory {
	return func(address string) registryv1alpha1connect.RepositoryCommitServiceClient {
		return connectclient.Make(clientConfig, address, registryv1alpha1connect.NewRepositoryCommitServiceClient)
	}
}

func NewRepositoryServiceClientFactory(clientConfig *connectclient.Config) RepositoryServiceClientFactory {
	return func(address string) registryv1alpha1connect.RepositoryServiceClient {
		return connectclient.Make(clientConfig, address, registryv1alpha1connect.NewRepositoryServiceClient)
	}
}

// NewModuleReader returns a new ModuleReader backed by the download service.
func NewModuleReader(
	logger *zap.Logger,
	downloadClientFactory DownloadServiceClientFactory,
	opts ...ModuleReaderOption,
) bufmodule.ModuleReader {
	return newModuleReader(
		logger,
		downloadClientFactory,
		opts...,
	)
}

// ModuleReaderOption allows configuration of a module reader.
type ModuleReaderOption func(reader *moduleReader)

// ModuleReaderWithDeprecationWarning makes the module reader print a warning
// when reading a deprecated module.
func ModuleReaderWithDeprecationWarning(
	repositoryClientFactory RepositoryServiceClientFactory,
) ModuleReaderOption {
	return func(reader *moduleReader) {
		reader.repositoryClientFactory = repositoryClientFactory
	}
}

// NewModuleResolver returns a new ModuleResolver backed by the resolve service.
func NewModuleResolver(
	logger *zap.Logger,
	repositoryCommitClientFactory RepositoryCommitServiceClientFactory,
) bufmodule.ModuleResolver {
	return newModuleResolver(logger, repositoryCommitClientFactory)
}

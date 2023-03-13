// Copyright 2020-2023 Buf Technologies, Inc.
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

package bufmodulecache

import (
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/filelock"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/verbose"
	"go.uber.org/zap"
)

// ModuleReaderOption is an option for creating a ModuleReader.
type ModuleReaderOption func(*moduleReaderOptions)

type RepositoryServiceClientFactory func(address string) registryv1alpha1connect.RepositoryServiceClient

func NewRepositoryServiceClientFactory(clientConfig *connectclient.Config) RepositoryServiceClientFactory {
	return func(address string) registryv1alpha1connect.RepositoryServiceClient {
		return connectclient.Make(clientConfig, address, registryv1alpha1connect.NewRepositoryServiceClient)
	}
}

// NewModuleReader returns a new ModuleReader that uses cache as a caching layer, and
// delegate as the source of truth.
func NewModuleReader(
	logger *zap.Logger,
	verbosePrinter verbose.Printer,
	fileLocker filelock.Locker,
	dataReadWriteBucket storage.ReadWriteBucket,
	sumReadWriteBucket storage.ReadWriteBucket,
	delegate bufmodule.ModuleReader,
	repositoryClientFactory RepositoryServiceClientFactory,
	options ...ModuleReaderOption,
) bufmodule.ModuleReader {
	return newModuleReader(
		logger,
		verbosePrinter,
		fileLocker,
		dataReadWriteBucket,
		sumReadWriteBucket,
		delegate,
		repositoryClientFactory,
		options...,
	)
}

// ModuleReaderWithExternalPaths is used to preserve the external paths
// to the files resolved from the module cache.
func ModuleReaderWithExternalPaths() ModuleReaderOption {
	return func(moduleReaderOptions *moduleReaderOptions) {
		moduleReaderOptions.allowCacheExternalPaths = true
	}
}

// NewCASModuleReader creates a new module reader using content addressable storage.
// This doesn't require file locking and enables support for tamper proofing.
func NewCASModuleReader(
	logger *zap.Logger,
	verbosePrinter verbose.Printer,
	bucket storage.ReadWriteBucket,
	delegate bufmodule.ModuleReader,
	repositoryClientFactory RepositoryServiceClientFactory,
) bufmodule.ModuleReader {
	return newCASModuleReader(
		bucket,
		delegate,
		repositoryClientFactory,
		logger,
		verbosePrinter,
	)
}

type moduleReaderOptions struct {
	allowCacheExternalPaths bool
}

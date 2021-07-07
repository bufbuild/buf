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

package bufcli

import (
	"context"
	"os"
	"sync"

	"github.com/bufbuild/buf/internal/buf/bufapimodule"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmodulecache"
	"github.com/bufbuild/buf/internal/gen/proto/apiclient/buf/alpha/registry/v1alpha1/registryv1alpha1apiclient"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
	"github.com/bufbuild/buf/internal/pkg/filelock"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
)

type registryModuleResolverReaderProvider struct {
	registryProvider registryv1alpha1apiclient.Provider
	setupErr         error
	setup            sync.Once
}

func newRegistryModuleResolverReaderProvider() *registryModuleResolverReaderProvider {
	return &registryModuleResolverReaderProvider{}
}

// GetModuleResolver returns a new ModuleResolver.
func (m *registryModuleResolverReaderProvider) GetModuleResolver(ctx context.Context, container appflag.Container) (bufmodule.ModuleResolver, error) {
	m.setup.Do(func() {
		m.registryProvider, m.setupErr = NewRegistryProvider(ctx, container)
	})
	if m.setupErr != nil {
		return nil, m.setupErr
	}
	return bufapimodule.NewModuleResolver(
		container.Logger(),
		m.registryProvider,
	), nil
}

// GetModuleReader returns a new ModuleReader.
func (m *registryModuleResolverReaderProvider) GetModuleReader(ctx context.Context, container appflag.Container) (bufmodule.ModuleReader, error) {
	m.setup.Do(func() {
		m.registryProvider, m.setupErr = NewRegistryProvider(ctx, container)
	})
	if m.setupErr != nil {
		return nil, m.setupErr
	}
	cacheModuleDataDirPath := normalpath.Join(container.CacheDirPath(), v1CacheModuleDataRelDirPath)
	cacheModuleLockDirPath := normalpath.Join(container.CacheDirPath(), v1CacheModuleLockRelDirPath)
	cacheModuleSumDirPath := normalpath.Join(container.CacheDirPath(), v1CacheModuleSumRelDirPath)
	if err := createCacheDirs(
		cacheModuleDataDirPath,
		cacheModuleLockDirPath,
		cacheModuleSumDirPath,
	); err != nil {
		return nil, err
	}
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	// do NOT want to enable symlinks for our cache
	dataReadWriteBucket, err := storageosProvider.NewReadWriteBucket(cacheModuleDataDirPath)
	if err != nil {
		return nil, err
	}
	// do NOT want to enable symlinks for our cache
	sumReadWriteBucket, err := storageosProvider.NewReadWriteBucket(cacheModuleSumDirPath)
	if err != nil {
		return nil, err
	}
	fileLocker, err := filelock.NewLocker(cacheModuleLockDirPath)
	if err != nil {
		return nil, err
	}
	moduleReader := bufmodulecache.NewModuleReader(
		container.Logger(),
		dataReadWriteBucket,
		sumReadWriteBucket,
		bufapimodule.NewModuleReader(
			m.registryProvider,
		),
		bufmodulecache.WithMessageWriter(
			container.Stderr(),
		),
		bufmodulecache.WithFileLocker(fileLocker),
	)
	return moduleReader, nil
}

func createCacheDirs(dirPaths ...string) error {
	for _, dirPath := range dirPaths {
		if err := os.MkdirAll(normalpath.Unnormalize(dirPath), 0755); err != nil {
			return err
		}
	}
	return nil
}

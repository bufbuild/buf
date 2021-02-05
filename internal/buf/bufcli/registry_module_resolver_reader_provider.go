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

const modDir = "mod"

var lockDir = normalpath.Join("lock", "mod")

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
	modCacheDirPath := normalpath.Join(container.CacheDirPath(), modDir)
	if err := os.MkdirAll(normalpath.Unnormalize(modCacheDirPath), 0755); err != nil {
		return nil, err
	}
	lockCacheDirPath := normalpath.Join(container.CacheDirPath(), lockDir)
	if err := os.MkdirAll(normalpath.Unnormalize(lockCacheDirPath), 0755); err != nil {
		return nil, err
	}
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	// do NOT want to enable symlinks for our cache
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(modCacheDirPath)
	if err != nil {
		return nil, err
	}
	fileLocker, err := filelock.NewLocker(lockCacheDirPath)
	if err != nil {
		return nil, err
	}
	moduleReader := bufmodulecache.NewModuleReader(
		container.Logger(),
		readWriteBucket,
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

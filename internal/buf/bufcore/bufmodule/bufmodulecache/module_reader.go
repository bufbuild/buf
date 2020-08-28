// Copyright 2020 Buf Technologies, Inc.
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
	"context"
	"fmt"
	"sync"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/pkg/storage"
)

type moduleReader struct {
	cache    bufmodule.ModuleReadWriter
	delegate bufmodule.ModuleReader

	count     int
	cacheHits int
	lock      sync.RWMutex
}

func newModuleReader(
	cache bufmodule.ModuleReadWriter,
	delegate bufmodule.ModuleReader,
) *moduleReader {
	return &moduleReader{
		cache:    cache,
		delegate: delegate,
	}
}

func (m *moduleReader) GetModule(
	ctx context.Context,
	moduleName bufmodule.ModuleName,
) (bufmodule.Module, error) {
	module, err := m.cache.GetModule(ctx, moduleName)
	if err != nil {
		if storage.IsNotExist(err) {
			module, err := m.getModuleUncached(ctx, moduleName)
			if err != nil {
				return nil, err
			}
			m.lock.Lock()
			m.count++
			m.lock.Unlock()
			return module, nil
		}
		return nil, err
	}
	m.lock.Lock()
	m.count++
	m.cacheHits++
	m.lock.Unlock()
	return module, nil
}

func (m *moduleReader) getModuleUncached(
	ctx context.Context,
	moduleName bufmodule.ModuleName,
) (bufmodule.Module, error) {
	module, err := m.delegate.GetModule(ctx, moduleName)
	if err != nil {
		return nil, err
	}
	unresolvedModuleName, err := bufmodule.UnresolvedModuleName(moduleName)
	if err != nil {
		return nil, err
	}
	cacheModuleName, err := m.cache.PutModule(
		ctx,
		unresolvedModuleName,
		module,
	)
	if err != nil {
		return nil, err
	}
	if !bufmodule.ModuleNameEqual(moduleName, cacheModuleName) {
		return nil, fmt.Errorf("mismatched cache module name: %q %q", moduleName.String(), cacheModuleName.String())
	}
	return module, nil
}

func (m *moduleReader) getCount() int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.count
}

func (m *moduleReader) getCacheHits() int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.cacheHits
}

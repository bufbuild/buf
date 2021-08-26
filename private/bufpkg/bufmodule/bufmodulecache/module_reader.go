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

package bufmodulecache

import (
	"context"
	"io"
	"sync"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/filelock"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type moduleReader struct {
	logger        *zap.Logger
	cache         *moduleCacher
	delegate      bufmodule.ModuleReader
	messageWriter io.Writer
	fileLocker    filelock.Locker

	count     int
	cacheHits int
	lock      sync.RWMutex
}

func newModuleReader(
	logger *zap.Logger,
	dataReadWriteBucket storage.ReadWriteBucket,
	sumReadWriteBucket storage.ReadWriteBucket,
	delegate bufmodule.ModuleReader,
	options ...ModuleReaderOption,
) *moduleReader {
	moduleReader := &moduleReader{
		logger:     logger,
		delegate:   delegate,
		fileLocker: filelock.NewNopLocker(),
	}
	for _, option := range options {
		option(moduleReader)
	}
	moduleReader.cache = newModuleCacher(
		dataReadWriteBucket,
		sumReadWriteBucket,
		moduleReader.fileLocker,
	)
	return moduleReader
}

func (m *moduleReader) GetModule(
	ctx context.Context,
	modulePin bufmodule.ModulePin,
) (module bufmodule.Module, retErr error) {
	module, err := m.cache.GetModule(ctx, modulePin)
	if err != nil {
		if storage.IsNotExist(err) {
			m.logger.Debug(
				"cache_miss",
				zap.String("module_pin", modulePin.String()),
			)
			// We need to use a separate file lock in this case because it would otherwise contend with
			// the file lock used to read and write from the cache (i.e. deadlock).
			unlocker, err := m.fileLocker.Lock(ctx, newCacheKey(modulePin)+".download")
			if err != nil {
				return nil, err
			}
			defer func() {
				retErr = multierr.Append(retErr, unlocker.Unlock())
			}()
			// Now that we have acquired the write lock, we're guaranteed to be the only running process in this branch.
			// Another process might have already made it here, so first check if the module was already fetched and
			// stored in the cache before continuing.
			module, err := m.cache.GetModule(ctx, modulePin)
			if err != nil {
				if !storage.IsNotExist(err) {
					return nil, err
				}
			}
			if err == nil {
				// Another process fetched the module before us, so we can return early.
				return module, nil
			}
			if m.messageWriter != nil {
				if _, err := m.messageWriter.Write([]byte("buf: downloading " + modulePin.String() + "\n")); err != nil {
					return nil, err
				}
			}
			module, err = m.delegate.GetModule(ctx, modulePin)
			if err != nil {
				return nil, err
			}
			if err := m.cache.PutModule(
				ctx,
				modulePin,
				module,
			); err != nil {
				return nil, err
			}
			m.lock.Lock()
			m.count++
			m.lock.Unlock()
			return module, nil
		}
		return nil, err
	}
	m.logger.Debug(
		"cache_hit",
		zap.String("module_pin", modulePin.String()),
	)
	m.lock.Lock()
	m.count++
	m.cacheHits++
	m.lock.Unlock()
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

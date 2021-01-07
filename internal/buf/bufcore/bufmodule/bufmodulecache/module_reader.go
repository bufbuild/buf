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

	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/pkg/filelock"
	"github.com/bufbuild/buf/internal/pkg/storage"
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
	readWriteBucket storage.ReadWriteBucket,
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
	moduleReader.cache = newModuleCacher(readWriteBucket, moduleReader.fileLocker)
	return moduleReader
}

func (m *moduleReader) GetModule(
	ctx context.Context,
	modulePin bufmodule.ModulePin,
) (bufmodule.Module, error) {
	module, err := m.cache.GetModule(ctx, modulePin)
	if err != nil {
		if storage.IsNotExist(err) {
			m.logger.Debug(
				"cache_miss",
				zap.String("module_pin", modulePin.String()),
			)
			if m.messageWriter != nil {
				if _, err := m.messageWriter.Write([]byte("buf: downloading " + modulePin.String() + "\n")); err != nil {
					return nil, err
				}
			}
			module, err := m.delegate.GetModule(ctx, modulePin)
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

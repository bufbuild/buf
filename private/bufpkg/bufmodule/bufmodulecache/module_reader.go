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
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/filelock"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/verbose"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type moduleReader struct {
	logger                  *zap.Logger
	verbosePrinter          verbose.Printer
	fileLocker              filelock.Locker
	cache                   *moduleCacher
	delegate                bufmodule.ModuleReader
	repositoryClientFactory RepositoryServiceClientFactory

	stats *cacheStats
}

func newModuleReader(
	logger *zap.Logger,
	verbosePrinter verbose.Printer,
	fileLocker filelock.Locker,
	dataReadWriteBucket storage.ReadWriteBucket,
	sumReadWriteBucket storage.ReadWriteBucket,
	delegate bufmodule.ModuleReader,
	repositoryClientFactory RepositoryServiceClientFactory,
	options ...ModuleReaderOption,
) *moduleReader {
	moduleReaderOptions := &moduleReaderOptions{}
	for _, option := range options {
		option(moduleReaderOptions)
	}
	return &moduleReader{
		logger:         logger,
		verbosePrinter: verbosePrinter,
		fileLocker:     fileLocker,
		cache: newModuleCacher(
			logger,
			dataReadWriteBucket,
			sumReadWriteBucket,
			moduleReaderOptions.allowCacheExternalPaths,
		),
		delegate:                delegate,
		repositoryClientFactory: repositoryClientFactory,
		stats:                   &cacheStats{},
	}
}

func (m *moduleReader) GetModule(
	ctx context.Context,
	modulePin bufmoduleref.ModulePin,
) (_ bufmodule.Module, retErr error) {
	cacheKey := newCacheKey(modulePin)

	// First, do a GetModule with a read lock to see if we have a valid module.
	readUnlocker, err := m.fileLocker.RLock(ctx, cacheKey)
	if err != nil {
		return nil, err
	}
	module, err := m.cache.GetModule(ctx, modulePin)
	err = multierr.Append(err, readUnlocker.Unlock())
	if err == nil {
		m.logger.Debug(
			"cache_hit",
			zap.String("module_pin", modulePin.String()),
		)
		m.stats.MarkHit()
		return module, nil
	}
	if !storage.IsNotExist(err) {
		return nil, err
	}

	// We now had a IsNotExist error, so we do a write lock and check again (double locking).
	// If we still have an error, we do a GetModule from the delegate, and put the result.
	//
	// Note that IsNotExist will happen if there was a checksum mismatch as well, in which case
	// we want to overwrite whatever is actually in the cache and self-correct the issue
	unlocker, err := m.fileLocker.Lock(ctx, cacheKey)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, unlocker.Unlock())
	}()
	module, err = m.cache.GetModule(ctx, modulePin)
	if err == nil {
		m.logger.Debug(
			"cache_hit",
			zap.String("module_pin", modulePin.String()),
		)
		m.stats.MarkHit()
		return module, nil
	}
	if !storage.IsNotExist(err) {
		return nil, err
	}
	m.stats.MarkMiss()

	// We now had a IsNotExist error within a write lock, so go to the delegate and then put.
	m.logger.Debug(
		"cache_miss",
		zap.String("module_pin", modulePin.String()),
	)
	m.verbosePrinter.Printf("downloading " + modulePin.String())
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
	if err := warnIfDeprecated(ctx, m.repositoryClientFactory, modulePin, m.logger); err != nil {
		return nil, err
	}
	return module, nil
}

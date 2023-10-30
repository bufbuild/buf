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
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/verbose"
	"go.uber.org/zap"
)

type casModuleReader struct {
	// required parameters
	delegate                bufmodule.ModuleReader
	repositoryClientFactory RepositoryServiceClientFactory
	logger                  *zap.Logger
	verbosePrinter          verbose.Printer
	// initialized in newCASModuleReader
	cache *casModuleCacher
	stats *cacheStats
}

var _ bufmodule.ModuleReader = (*casModuleReader)(nil)

func newCASModuleReader(
	bucket storage.ReadWriteBucket,
	delegate bufmodule.ModuleReader,
	repositoryClientFactory RepositoryServiceClientFactory,
	logger *zap.Logger,
	verbosePrinter verbose.Printer,
) *casModuleReader {
	return &casModuleReader{
		delegate:                delegate,
		repositoryClientFactory: repositoryClientFactory,
		logger:                  logger,
		verbosePrinter:          verbosePrinter,
		cache: &casModuleCacher{
			logger: logger,
			bucket: bucket,
		},
		stats: &cacheStats{},
	}
}

func (c *casModuleReader) GetModule(
	ctx context.Context,
	modulePin bufmoduleref.ModulePin,
) (bufmodule.Module, error) {
	var modulePinDigest bufcas.Digest
	if digest := modulePin.Digest(); digest != "" {
		var err error
		modulePinDigest, err = bufcas.ParseDigest(digest)
		// Fail fast if the buf.lock file contains a malformed digest
		if err != nil {
			return nil, fmt.Errorf("malformed module digest %q: %w", digest, err)
		}
	}
	cachedModule, err := c.cache.GetModule(ctx, modulePin)
	if err == nil {
		c.stats.MarkHit()
		return cachedModule, nil
	}
	c.logger.Debug("module cache miss", zap.Error(err))
	c.stats.MarkMiss()
	remoteModule, err := c.delegate.GetModule(ctx, modulePin)
	if err != nil {
		return nil, err
	}
	// FileSet should always be set.
	if remoteModule.FileSet() == nil {
		return nil, errors.New("required FileSet not set on Module")
	}
	if modulePinDigest != nil {
		manifestBlob, err := bufcas.ManifestToBlob(remoteModule.FileSet().Manifest())
		if err != nil {
			return nil, err
		}
		manifestDigest := manifestBlob.Digest()
		if !bufcas.DigestEqual(modulePinDigest, manifestDigest) {
			// buf.lock module digest and BSR module don't match - fail without overwriting cache
			return nil, fmt.Errorf("module digest mismatch - expected: %q, found: %q", modulePinDigest, manifestDigest)
		}
	}
	if err := c.cache.PutModule(ctx, modulePin, remoteModule); err != nil {
		return nil, err
	}
	if err := warnIfDeprecated(ctx, c.repositoryClientFactory, modulePin, c.logger); err != nil {
		return nil, err
	}
	return remoteModule, nil
}

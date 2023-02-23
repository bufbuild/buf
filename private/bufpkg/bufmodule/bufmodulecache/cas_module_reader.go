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
	"fmt"

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
	module, err := c.cache.GetModule(ctx, modulePin)
	if err == nil {
		c.stats.MarkHit()
		return module, nil
	}
	c.logger.Debug("module cache miss", zap.Error(err))
	c.stats.MarkMiss()
	module, err = c.delegate.GetModule(ctx, modulePin)
	if err != nil {
		return nil, err
	}
	// Manifest and BlobSet should always be set if tamper proofing is enabled.
	// If not, the BSR doesn't support tamper proofing while the CLI feature is enabled.
	if module.Manifest() == nil || module.BlobSet() == nil {
		return nil, fmt.Errorf("required manifest/blobSet not set on module")
	}
	if modulePinDigest := modulePin.Digest(); modulePinDigest != "" {
		manifestBlob, err := module.Manifest().Blob()
		if err != nil {
			return nil, err
		}
		manifestDigest := manifestBlob.Digest().String()
		if manifestDigest != modulePinDigest {
			// buf.lock module digest and BSR module don't match - fail without overwriting cache
			return nil, fmt.Errorf("module digest mismatch - expected: %q, found: %q", modulePinDigest, manifestDigest)
		}
	}
	if err := c.cache.PutModule(ctx, modulePin, module); err != nil {
		return nil, err
	}
	if err := warnIfDeprecated(ctx, c.repositoryClientFactory, modulePin, c.logger); err != nil {
		return nil, err
	}
	return module, nil
}

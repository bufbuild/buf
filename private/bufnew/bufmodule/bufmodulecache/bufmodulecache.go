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
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"sync/atomic"

	"github.com/bufbuild/buf/private/bufnew/bufconfig"
	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
)

// NewModuleDataProvider returns a new ModuleDataProvider that caches the results of the delegate.
//
// The given Bucket is used as a cache. This package can choose to use the bucket however it wishes.
func NewModuleDataProvider(
	delegate bufmodule.ModuleDataProvider,
	moduleCacheBucket storage.ReadWriteBucket,
) bufmodule.ModuleDataProvider {
	return newModuleDataProvider(delegate, moduleCacheBucket)
}

/// *** PRIVATE ***

type moduleDataProvider struct {
	delegate          bufmodule.ModuleDataProvider
	moduleCacheBucket storage.ReadWriteBucket

	moduleKeysRetrieved atomic.Int64
	moduleKeysHit       atomic.Int64
}

func newModuleDataProvider(
	delegate bufmodule.ModuleDataProvider,
	moduleCacheBucket storage.ReadWriteBucket,
) *moduleDataProvider {
	return &moduleDataProvider{
		delegate:          delegate,
		moduleCacheBucket: moduleCacheBucket,
	}
}

func (p *moduleDataProvider) GetModuleDatasForModuleKeys(
	ctx context.Context,
	moduleKeys ...bufmodule.ModuleKey,
) ([]bufmodule.ModuleData, error) {
	moduleDatas := make([]bufmodule.ModuleData, len(moduleKeys))

	// The indexes within moduleKeys of the ModuleKeys that did not have a cached ModuleData.
	// We will then fetch these specific ModuleKeys in one shot from the delegate.
	var missedModuleKeysIndexes []int
	for i, moduleKey := range moduleKeys {
		cachedModuleData, err := p.getCachedModuleDataForModuleKey(ctx, moduleKey)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				// If the error is not a ErrNotExist error, we want to return, as this means some
				// other error other than the ModuleData was not cached.
				return nil, err
			}
			missedModuleKeysIndexes = append(missedModuleKeysIndexes, i)
			continue
		}
		// We put the cached ModuleData at the specific location it is expected to be returned,
		// given that the returned ModuleData order must match the input ModuleKey order.
		moduleDatas[i] = cachedModuleData
	}

	if len(missedModuleKeysIndexes) > 0 {
		missedModuleDatas, err := p.delegate.GetModuleDatasForModuleKeys(
			ctx,
			// Map the indexes of to the actual ModuleKeys.
			slicesext.Map(
				missedModuleKeysIndexes,
				func(i int) bufmodule.ModuleKey { return moduleKeys[i] },
			)...,
		)
		if err != nil {
			// Automatically returns an error with fs.ErrNotExist if a ModuleKey is not found.
			return nil, err
		}
		// Just a sanity check.
		if len(missedModuleDatas) != len(missedModuleKeysIndexes) {
			return nil, fmt.Errorf("expected %d ModuleDatas, got %d", len(missedModuleKeysIndexes), len(missedModuleDatas))
		}
		for i, missedModuleKeysIndex := range missedModuleKeysIndexes {
			// i is the index within missedModuleDatas, while missedModuleKeysIndex is the index
			// within missedModuleKeysIndexes, and consequently moduleKeys.
			missedModuleData := missedModuleDatas[i]
			if err := p.putMissedModuleData(ctx, missedModuleData); err != nil {
				return nil, err
			}
			// Put in the specific location we expect the ModuleData to be returned.
			moduleDatas[missedModuleKeysIndex] = missedModuleData
		}
	}

	p.moduleKeysRetrieved.Add(int64(len(moduleDatas)))
	p.moduleKeysHit.Add(int64(len(moduleDatas) - len(missedModuleKeysIndexes)))

	return moduleDatas, nil
}

func (p *moduleDataProvider) getCachedModuleDataForModuleKey(
	ctx context.Context,
	moduleKey bufmodule.ModuleKey,
) (bufmodule.ModuleData, error) {
	moduleFullName := moduleKey.ModuleFullName()
	digest, err := moduleKey.Digest()
	if err != nil {
		return nil, err
	}
	modulePrefix := getModulePrefix(moduleFullName, digest)
	// We rely on the buf.lock file being the last file to be written in putMissedModuleData.
	// If the buf.lock does not exist, we act as if there is no value in the cache, which will
	// result in bad data being overwritten.
	bufLockFile, err := bufconfig.GetBufLockFileForPrefix(ctx, p.moduleCacheBucket, modulePrefix)
	if err != nil {
		return nil, err
	}
	// It is OK that this ReadBucket contains the buf.lock; the buf.lock will be ignored. See
	// comments on ModuleData.Bucket().
	moduleDataBucket := storage.MapReadBucket(p.moduleCacheBucket, storage.MapOnPrefix(modulePrefix))
	return bufmodule.NewModuleData(
		moduleKey,
		func() (storage.ReadBucket, error) {
			return moduleDataBucket, nil
		},
		func() ([]bufmodule.ModuleKey, error) {
			return bufLockFile.DepModuleKeys(), nil
		},
		// This will do tamper-proofing.
		bufmodule.ModuleDataWithActualDigest(digest),
	)
}

func (p *moduleDataProvider) putMissedModuleData(
	ctx context.Context,
	moduleData bufmodule.ModuleData,
) error {
	moduleKey := moduleData.ModuleKey()
	moduleFullName := moduleKey.ModuleFullName()
	digest, err := moduleKey.Digest()
	if err != nil {
		return err
	}
	modulePrefix := getModulePrefix(moduleFullName, digest)
	depModuleKeys, err := moduleData.DeclaredDepModuleKeys()
	if err != nil {
		return err
	}
	bufLockFile, err := bufconfig.NewBufLockFile(bufconfig.FileVersionV2, depModuleKeys)
	if err != nil {
		return err
	}
	moduleDataBucket, err := moduleData.Bucket()
	if err != nil {
		return err
	}
	if _, err := storage.Copy(
		ctx,
		moduleDataBucket,
		storage.MapWriteBucket(p.moduleCacheBucket, storage.MapOnPrefix(modulePrefix)),
		storage.CopyWithAtomic(),
	); err != nil {
		return err
	}
	// Put the buf.lock last, so that we only have a buf.lock if the cache is finished writing.
	// We can use the existence of the buf.lock file to say whether or not the cache contains a
	// given ModuleKey, otherwise we overwrite any contents in the cache.
	return bufconfig.PutBufLockFileForPrefix(ctx, p.moduleCacheBucket, modulePrefix, bufLockFile)
}

func (p *moduleDataProvider) getModuleKeysRetrieved() int {
	return int(p.moduleKeysRetrieved.Load())
}

func (p *moduleDataProvider) getModuleKeysHit() int {
	return int(p.moduleKeysHit.Load())
}

// Returns the module's path within the cache. This is "registry/owner/name/${DIGEST_TYPE}/${DIGEST}",
// e.g. the module "buf.build/acme/weather" with digest "shake256:12345" will return
// "buf.build/acme/weather/shake256/12345".
func getModulePrefix(moduleFullName bufmodule.ModuleFullName, digest bufcas.Digest) string {
	return normalpath.Join(
		moduleFullName.Registry(),
		moduleFullName.Owner(),
		moduleFullName.Name(),
		digest.Type().String(),
		hex.EncodeToString(digest.Value()),
	)
}

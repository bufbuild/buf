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

package bufmodulestore

import (
	"context"
	"encoding/hex"
	"errors"
	"io/fs"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
)

// ModuleDataStore reads and writes ModulesDatas.
type ModuleDataStore interface {
	bufmodule.ModuleDataProvider
	ModuleDataWriter
}

// NewModuleDataStore returns a new ModuleDataStore for the given bucket.
//
// It is assumed that the ModuleDataStore has complete control of the bucket.
//
// This is typically used to interact with a cache directory.
func NewModuleDataStore(bucket storage.ReadWriteBucket) ModuleDataStore {
	return newModuleDataStore(bucket)
}

// ModuleDataStoreOption is an option for a new ModuleDataStore.
type ModuleDataStoreOption func(*moduleDataStore)

// ModuleDataStoreWithZip returns a new ModuleDataStoreOption that reads and stores
// zip files instead of storing individual files in the bucket.
func ModuleDataStoreWithZip() ModuleDataStoreOption {
	return func(moduleDataStore *moduleDataStore) {
		moduleDataStore.zip = true
	}
}

/// *** PRIVATE ***

type moduleDataStore struct {
	bucket storage.ReadWriteBucket

	zip bool
}

func newModuleDataStore(
	bucket storage.ReadWriteBucket,
) *moduleDataStore {
	return &moduleDataStore{
		bucket: bucket,
	}
}

func (p *moduleDataStore) GetOptionalModuleDatasForModuleKeys(
	ctx context.Context,
	moduleKeys ...bufmodule.ModuleKey,
) ([]bufmodule.OptionalModuleData, error) {
	optionalModuleDatas := make([]bufmodule.OptionalModuleData, len(moduleKeys))
	for i, moduleKey := range moduleKeys {
		moduleData, err := p.getModuleDataForModuleKey(ctx, moduleKey)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, err
			}
		}
		optionalModuleDatas[i] = bufmodule.NewOptionalModuleData(moduleData)
	}
	return optionalModuleDatas, nil
}

func (p *moduleDataStore) PutModuleDatas(
	ctx context.Context,
	moduleDatas ...bufmodule.ModuleData,
) error {
	for _, moduleData := range moduleDatas {
		if err := p.putModuleData(ctx, moduleData); err != nil {
			return err
		}
	}
	return nil
}

func (p *moduleDataStore) getModuleDataForModuleKey(
	ctx context.Context,
	moduleKey bufmodule.ModuleKey,
) (bufmodule.ModuleData, error) {
	moduleFullName := moduleKey.ModuleFullName()
	digest, err := moduleKey.Digest()
	if err != nil {
		return nil, err
	}
	moduleStorePrefix := getModuleStorePrefix(moduleFullName, digest)
	// It is OK that this ReadBucket contains the buf.lock; the buf.lock will be ignored. See
	// comments on ModuleData.Bucket().
	moduleDataBucket := storage.MapReadBucket(p.bucket, storage.MapOnPrefix(moduleStorePrefix))
	// We rely on the buf.lock file being the last file to be written in putMissedModuleData.
	// If the buf.lock does not exist, we act as if there is no value in the store, which will
	// result in bad data being overwritten.
	bufLockFile, err := bufconfig.GetBufLockFileForPrefix(ctx, moduleDataBucket, ".")
	if err != nil {
		return nil, err
	}
	return bufmodule.NewModuleData(
		moduleKey,
		func() (storage.ReadBucket, error) {
			return moduleDataBucket, nil
		},
		func() ([]bufmodule.ModuleKey, error) {
			return bufLockFile.DepModuleKeys(), nil
		},
		// This will do tamper-proofing.
		// TODO: No it won't.
		bufmodule.ModuleDataWithActualDigest(digest),
	)
}

func (p *moduleDataStore) putModuleData(
	ctx context.Context,
	moduleData bufmodule.ModuleData,
) error {
	moduleKey := moduleData.ModuleKey()
	moduleFullName := moduleKey.ModuleFullName()
	digest, err := moduleKey.Digest()
	if err != nil {
		return err
	}
	moduleStorePrefix := getModuleStorePrefix(moduleFullName, digest)
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
	putBucket := storage.MapWriteBucket(p.bucket, storage.MapOnPrefix(moduleStorePrefix))
	if _, err := storage.Copy(
		ctx,
		moduleDataBucket,
		putBucket,
		storage.CopyWithAtomic(),
	); err != nil {
		return err
	}
	// Put the buf.lock last, so that we only have a buf.lock if the cache is finished writing.
	// We can use the existence of the buf.lock file to say whether or not the cache contains a
	// given ModuleKey, otherwise we overwrite any contents in the cache.
	return bufconfig.PutBufLockFileForPrefix(ctx, putBucket, ".", bufLockFile)
}

// Returns the module's path within the store. This is "registry/owner/name/${DIGEST_TYPE}/${DIGEST}",
// e.g. the module "buf.build/acme/weather" with digest "shake256:12345" will return
// "buf.build/acme/weather/shake256/12345".
func getModuleStorePrefix(moduleFullName bufmodule.ModuleFullName, digest bufcas.Digest) string {
	return normalpath.Join(
		moduleFullName.Registry(),
		moduleFullName.Owner(),
		moduleFullName.Name(),
		digest.Type().String(),
		hex.EncodeToString(digest.Value()),
	)
}

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
	"github.com/bufbuild/buf/private/pkg/storage/storagearchive"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"go.uber.org/multierr"
	"go.uber.org/zap"
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
//
// TODO: make self-correcting. Just delete and return not found if there is an error on read,
// or at least make this optional.
func NewModuleDataStore(
	logger *zap.Logger,
	bucket storage.ReadWriteBucket,
	options ...ModuleDataStoreOption,
) ModuleDataStore {
	return newModuleDataStore(logger, bucket, options...)
}

// ModuleDataStoreOption is an option for a new ModuleDataStore.
type ModuleDataStoreOption func(*moduleDataStore)

// ModuleDataStoreWithTar returns a new ModuleDataStoreOption that reads and stores
// tar files instead of storing individual files in a directory in the bucket.
//
// The default is to store individual files in a directory.
func ModuleDataStoreWithTar() ModuleDataStoreOption {
	return func(moduleDataStore *moduleDataStore) {
		moduleDataStore.tar = true
	}
}

/// *** PRIVATE ***

type moduleDataStore struct {
	logger *zap.Logger
	bucket storage.ReadWriteBucket

	tar bool
}

func newModuleDataStore(
	logger *zap.Logger,
	bucket storage.ReadWriteBucket,
	options ...ModuleDataStoreOption,
) *moduleDataStore {
	moduleDataStore := &moduleDataStore{
		logger: logger,
		bucket: bucket,
	}
	for _, option := range options {
		option(moduleDataStore)
	}
	return moduleDataStore
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
	var bucket storage.ReadBucket
	if p.tar {
		bucket, err = p.getReadBucketForTar(ctx, moduleFullName, digest)
		if err != nil {
			return nil, err
		}
	} else {
		bucket = p.getReadBucketForDir(moduleFullName, digest)
	}
	// We rely on the buf.lock file being the last file to be written in the store.
	// If the buf.lock does not exist, we act as if there is no value in the store, which will
	// result in bad data being overwritten.
	bufLockFile, err := bufconfig.GetBufLockFileForPrefix(ctx, bucket, ".")
	p.logDebugModuleFullNameAndDigest(
		moduleFullName,
		digest,
		"module store get buf.lock",
		zap.Bool("found", err == nil),
		zap.Error(err),
	)
	if err != nil {
		return nil, err
	}
	return bufmodule.NewModuleData(
		moduleKey,
		func() (storage.ReadBucket, error) {
			// It is OK that this ReadBucket contains the buf.lock; the buf.lock will be ignored. See
			// comments on ModuleData.Bucket().
			return bucket, nil
		},
		func() ([]bufmodule.ModuleKey, error) {
			return bufLockFile.DepModuleKeys(), nil
		},
		// This will do tamper-proofing.
		// TODO: No it won't.
		bufmodule.ModuleDataWithActualDigest(digest),
	)
}

func (p *moduleDataStore) getReadBucketForDir(
	moduleFullName bufmodule.ModuleFullName,
	digest bufcas.Digest,
) storage.ReadBucket {
	dirPath := getModuleStoreDirPath(moduleFullName, digest)
	p.logDebugModuleFullNameAndDigest(
		moduleFullName,
		digest,
		"module store get dir read bucket",
		zap.String("dirPath", dirPath),
	)
	return storage.MapReadBucket(p.bucket, storage.MapOnPrefix(dirPath))
}

func (p *moduleDataStore) getReadBucketForTar(
	ctx context.Context,
	moduleFullName bufmodule.ModuleFullName,
	digest bufcas.Digest,
) (_ storage.ReadBucket, retErr error) {
	tarPath := getModuleStoreTarPath(moduleFullName, digest)
	defer func() {
		p.logDebugModuleFullNameAndDigest(
			moduleFullName,
			digest,
			"module store get tar read bucket",
			zap.String("tarPath", tarPath),
			zap.Bool("found", retErr == nil),
			zap.Error(retErr),
		)
	}()
	readObjectCloser, err := p.bucket.Get(ctx, tarPath)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readObjectCloser.Close())
	}()
	readWriteBucket := storagemem.NewReadWriteBucket()
	if err := storagearchive.Untar(
		ctx,
		readObjectCloser,
		readWriteBucket,
		nil,
		0,
	); err != nil {
		return nil, err
	}
	return readWriteBucket, nil
}

func (p *moduleDataStore) putModuleData(
	ctx context.Context,
	moduleData bufmodule.ModuleData,
) (retErr error) {
	moduleKey := moduleData.ModuleKey()
	moduleFullName := moduleKey.ModuleFullName()
	digest, err := moduleKey.Digest()
	if err != nil {
		return err
	}
	var bucket storage.WriteBucket
	if p.tar {
		var callback func(ctx context.Context) error
		bucket, callback = p.getWriteBucketAndCallbackForTar(moduleFullName, digest)
		defer func() {
			if retErr == nil {
				// Only call the callback if we have had no error.
				retErr = multierr.Append(retErr, callback(ctx))
			}
		}()
	} else {
		bucket = p.getWriteBucketForDir(moduleFullName, digest)
	}
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
		bucket,
		storage.CopyWithAtomic(),
	); err != nil {
		return err
	}
	// Put the buf.lock last, so that we only have a buf.lock if the cache is finished writing.
	// We can use the existence of the buf.lock file to say whether or not the cache contains a
	// given ModuleKey, otherwise we overwrite any contents in the cache.
	return bufconfig.PutBufLockFileForPrefix(ctx, bucket, ".", bufLockFile)
}

func (p *moduleDataStore) getWriteBucketForDir(
	moduleFullName bufmodule.ModuleFullName,
	digest bufcas.Digest,
) storage.WriteBucket {
	dirPath := getModuleStoreDirPath(moduleFullName, digest)
	p.logDebugModuleFullNameAndDigest(
		moduleFullName,
		digest,
		"module store put dir write bucket",
		zap.String("dirPath", dirPath),
	)
	return storage.MapWriteBucket(p.bucket, storage.MapOnPrefix(dirPath))
}

func (p *moduleDataStore) getWriteBucketAndCallbackForTar(
	moduleFullName bufmodule.ModuleFullName,
	digest bufcas.Digest,
) (storage.WriteBucket, func(context.Context) error) {
	readWriteBucket := storagemem.NewReadWriteBucket()
	return readWriteBucket, func(ctx context.Context) (retErr error) {
		tarPath := getModuleStoreTarPath(moduleFullName, digest)
		defer func() {
			p.logDebugModuleFullNameAndDigest(
				moduleFullName,
				digest,
				"module store put tar to write bucket",
				zap.String("tarPath", tarPath),
				zap.Bool("found", retErr == nil),
				zap.Error(retErr),
			)
		}()
		writeObjectCloser, err := p.bucket.Put(
			ctx,
			tarPath,
			// Not needed since single file, but doing for now.
			storage.PutWithAtomic(),
		)
		if err != nil {
			return err
		}
		defer func() {
			retErr = multierr.Append(retErr, writeObjectCloser.Close())
		}()
		return storagearchive.Tar(
			ctx,
			readWriteBucket,
			writeObjectCloser,
		)
	}
}

func (p *moduleDataStore) logDebugModuleFullNameAndDigest(
	moduleFullName bufmodule.ModuleFullName,
	digest bufcas.Digest,
	message string,
	fields ...zap.Field,
) {
	if checkedEntry := p.logger.Check(zap.DebugLevel, message); checkedEntry != nil {
		checkedEntry.Write(
			append(
				[]zap.Field{
					zap.String("moduleFullName", moduleFullName.String()),
					zap.String("digest", digest.String()),
				},
				fields...,
			)...,
		)
	}
}

// Returns the module's path within the store if storing individual files.
//
// This is "registry/owner/name/${DIGEST_TYPE}/${DIGEST}",
// e.g. the module "buf.build/acme/weather" with digest "shake256:12345" will return
// "buf.build/acme/weather/shake256/12345".
func getModuleStoreDirPath(moduleFullName bufmodule.ModuleFullName, digest bufcas.Digest) string {
	return normalpath.Join(
		moduleFullName.Registry(),
		moduleFullName.Owner(),
		moduleFullName.Name(),
		digest.Type().String(),
		hex.EncodeToString(digest.Value()),
	)
}

// Returns the module's path within the store if storing tar files.
//
// This is "registry/owner/name/${DIGEST_TYPE}/${DIGEST}.tar",
// e.g. the module "buf.build/acme/weather" with digest "shake256:12345" will return
// "buf.build/acme/weather/shake256/12345.tar".
func getModuleStoreTarPath(moduleFullName bufmodule.ModuleFullName, digest bufcas.Digest) string {
	return normalpath.Join(
		moduleFullName.Registry(),
		moduleFullName.Owner(),
		moduleFullName.Name(),
		digest.Type().String(),
		hex.EncodeToString(digest.Value())+".tar",
	)
}
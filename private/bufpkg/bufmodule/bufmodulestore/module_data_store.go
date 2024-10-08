// Copyright 2020-2024 Buf Technologies, Inc.
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
	"errors"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/filelock"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/slogext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagearchive"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"go.uber.org/multierr"
)

var (
	externalModuleDataVersion      = "v1"
	externalModuleDataFileName     = "module.yaml"
	externalModuleDataFilesDir     = "files"
	externalModuleDataV1BufYAMLDir = "v1_buf_yaml"
	externalModuleDataV1BufLockDir = "v1_buf_lock"
	externalModuleDataLockFileExt  = ".lock"
)

// ModuleDatasResult is a result for a get of ModuleDatas.
type ModuleDatasResult interface {
	// FoundModuleDatas is the ModuleDatas that were found.
	//
	// Ordered by the order of input ModuleKeys.
	FoundModuleDatas() []bufmodule.ModuleData
	// NotFoundModuleKeys is the input ModuleKeys that were not found.
	//
	// Ordered by the order of input ModuleKeys.
	NotFoundModuleKeys() []bufmodule.ModuleKey

	isModuleDatasResult()
}

// ModuleStore reads and writes ModulesDatas.
type ModuleDataStore interface {
	// GetModuleDatasForModuleKey gets the ModuleDatas from the store for the ModuleKeys.
	//
	// Returns the found ModuleDatas, and the input ModuleKeys that were not found, each
	// ordered by the order of the input ModuleKeys.
	GetModuleDatasForModuleKeys(context.Context, []bufmodule.ModuleKey) (
		foundModuleDatas []bufmodule.ModuleData,
		notFoundModuleKeys []bufmodule.ModuleKey,
		err error,
	)

	// Put puts the ModuleDatas to the store.
	PutModuleDatas(ctx context.Context, moduleDatas []bufmodule.ModuleData) error
}

// NewModuleDataStore returns a new ModuleDataStore for the given bucket.
//
// It is assumed that the ModuleDataStore has complete control of the bucket.
//
// This is typically used to interact with a cache directory.
func NewModuleDataStore(
	logger *slog.Logger,
	bucket storage.ReadWriteBucket,
	locker filelock.Locker,
	options ...ModuleDataStoreOption,
) ModuleDataStore {
	return newModuleDataStore(logger, bucket, locker, options...)
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
	logger *slog.Logger
	bucket storage.ReadWriteBucket
	locker filelock.Locker

	tar bool
}

func newModuleDataStore(
	logger *slog.Logger,
	bucket storage.ReadWriteBucket,
	locker filelock.Locker,
	options ...ModuleDataStoreOption,
) *moduleDataStore {
	moduleDataStore := &moduleDataStore{
		logger: logger,
		bucket: bucket,
		locker: locker,
	}
	for _, option := range options {
		option(moduleDataStore)
	}
	return moduleDataStore
}

func (p *moduleDataStore) GetModuleDatasForModuleKeys(
	ctx context.Context,
	moduleKeys []bufmodule.ModuleKey,
) ([]bufmodule.ModuleData, []bufmodule.ModuleKey, error) {
	var foundModuleDatas []bufmodule.ModuleData
	var notFoundModuleKeys []bufmodule.ModuleKey
	for _, moduleKey := range moduleKeys {
		moduleData, err := p.getModuleDataForModuleKey(ctx, moduleKey)
		if err != nil {
			// Any error returned from getModuleDataForModuleKey means that no module data is read
			// from the cache, and is treated as a cache miss so we can fetch new module data and
			// repopulate the cache.
			notFoundModuleKeys = append(notFoundModuleKeys, moduleKey)
		} else {
			foundModuleDatas = append(foundModuleDatas, moduleData)
		}
	}
	return foundModuleDatas, notFoundModuleKeys, nil
}

func (p *moduleDataStore) PutModuleDatas(
	ctx context.Context,
	moduleDatas []bufmodule.ModuleData,
) error {
	for _, moduleData := range moduleDatas {
		if err := p.putModuleData(ctx, moduleData); err != nil {
			return err
		}
	}
	return nil
}

// getModuleDataForModuleKey reads the module data for the module key from the cache.
//
// If moduleDataStore is configured to store the module data as tarballs, then we read a
// single tar for all module data stored under the module key.
//
// If moduleDataStore is configured to store module data as individual files, then it
// takes the following steps to read the module data:
//
//  1. Acquire a shared lock on the module data lock file for the module key.
//  2. Attempt to read the module.yaml file for the module key.
//  3. If no valid module.yaml is present, then return an error and no data is read from
//     the cache. The module.yaml is always written to the cache last, so we consider valid
//     module data to be present if module.yaml is present.
//  4. If a valid module.yaml is found, then attempt to read the module data files from the
//     cache.
//  5. If an error occurs while reading the files and/or an invalid config file is found,
//     then no module data is read from the cache.
//  6. Once all files have been read from the cache, release the shared lock on the module
//     data lock file.
//
// It is important to note that when we read from the cache, we use the presence and contents
// of module.yaml to determine if module data exists in the cache. If there is manual intervention
// that corrupts the contents of the cache, but leaves module.yaml in-tact, then we read
// the files as valid module data at this layer, and it fails tamper-proofing digest checks
// when the module data is accessed.
func (p *moduleDataStore) getModuleDataForModuleKey(
	ctx context.Context,
	moduleKey bufmodule.ModuleKey,
) (retValue bufmodule.ModuleData, retErr error) {
	var moduleCacheBucket storage.ReadBucket
	var err error
	if p.tar {
		moduleCacheBucket, err = p.getReadBucketForTar(ctx, moduleKey)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				// If there is an error fetching the tar bucket that is not because the path does
				// not exist, we assume this is corrupted and delete the tar.
				tarPath, err := getModuleDataStoreTarPath(moduleKey)
				if err != nil {
					return nil, err
				}
				if err := p.bucket.Delete(ctx, tarPath); err != nil {
					return nil, err
				}
				// Return a path error indicating the module data was not found
				return nil, &fs.PathError{Op: "read", Path: tarPath, Err: fs.ErrNotExist}
			}
			return nil, err
		}
	} else {
		dirPath, err := getModuleDataStoreDirPath(moduleKey)
		if err != nil {
			return nil, err
		}
		p.logDebugModuleKey(
			ctx,
			moduleKey,
			"module data store dir read write bucket",
			slog.String("dirPath", dirPath),
		)
		moduleCacheBucket = storage.MapReadWriteBucket(p.bucket, storage.MapOnPrefix(dirPath))
		moduleDataStoreDirLockPath, err := getModuleDataStoreDirLockPath(moduleKey)
		if err != nil {
			return nil, err
		}
		// Acquire a shared lock for module data lock file for reading module data from the cache.
		unlocker, err := p.locker.RLock(ctx, moduleDataStoreDirLockPath)
		if err != nil {
			return nil, err
		}
		defer func() {
			// Release lock on the module data lock file.
			if err := unlocker.Unlock(); err != nil {
				retErr = multierr.Append(retErr, err)
			}
		}()
	}
	// Attempt to read module.yaml from cache. The module.yaml file is always written last,
	// so if a valid module.yaml file is present, then we proceed to read the rest of the
	// the module data.
	data, err := storage.ReadPath(ctx, moduleCacheBucket, externalModuleDataFileName)
	p.logDebugModuleKey(
		ctx,
		moduleKey,
		fmt.Sprintf("module data store get %s", externalModuleDataFileName),
		slog.Bool("found", err == nil),
		slogext.ErrorAttr(err),
	)
	if err != nil {
		return nil, err
	}
	var externalModuleData externalModuleData
	if err := encoding.UnmarshalYAMLNonStrict(data, &externalModuleData); err != nil {
		return nil, err
	}
	if !externalModuleData.isValid() {
		return nil, fmt.Errorf("invalid %s from cache for %s: %+v", externalModuleDataFileName, moduleKey.String(), externalModuleData)
	}
	// A valid module.yaml was found, we proceed to reading module data.

	// We don't want to do this lazily (or anything else in this function) as we want to
	// make sure everything we have is valid before returning so we can auto-correct
	// the cache if necessary.
	declaredDepModuleKeys, err := slicesext.MapError(
		externalModuleData.Deps,
		getDeclaredDepModuleKeyForExternalModuleDataDep,
	)
	if err != nil {
		return nil, err
	}
	var v1BufYAMLObjectData bufmodule.ObjectData
	if externalModuleData.V1BufYAMLFile != "" {
		// We do not want to use bufconfig.GetBufYAMLFileForPrefix as this validates the
		// buf.yaml, and potentially calls out to i.e. resolve digests. We just want to raw data.
		v1BufYAMLFileData, err := storage.ReadPath(ctx, moduleCacheBucket, externalModuleData.V1BufYAMLFile)
		if err != nil {
			return nil, err
		}
		v1BufYAMLObjectData, err = bufmodule.NewObjectData(
			normalpath.Base(externalModuleData.V1BufYAMLFile),
			v1BufYAMLFileData,
		)
		if err != nil {
			return nil, err
		}
	}
	var v1BufLockObjectData bufmodule.ObjectData
	if externalModuleData.V1BufLockFile != "" {
		// We do not want to use bufconfig.GetBufLockFileForPrefix as this validates the
		// buf.lock, and potentially calls out to i.e. resolve digests. We just want to raw data.
		v1BufLockFileData, err := storage.ReadPath(ctx, moduleCacheBucket, externalModuleData.V1BufLockFile)
		if err != nil {
			return nil, err
		}
		v1BufLockObjectData, err = bufmodule.NewObjectData(
			normalpath.Base(externalModuleData.V1BufLockFile),
			v1BufLockFileData,
		)
		if err != nil {
			return nil, err
		}
	}
	// We rely on the module.yaml file being the last file to be written in the store.
	// If module.yaml does not exist, we act as if there is no value in the store, which will
	// result in bad data being overwritten.
	return bufmodule.NewModuleData(
		ctx,
		moduleKey,
		func() (storage.ReadBucket, error) {
			return storage.StripReadBucketExternalPaths(
				storage.MapReadBucket(
					moduleCacheBucket,
					storage.MapOnPrefix(externalModuleData.FilesDir),
				),
			), nil
		},
		func() ([]bufmodule.ModuleKey, error) {
			return declaredDepModuleKeys, nil
		},
		func() (bufmodule.ObjectData, error) {
			return v1BufYAMLObjectData, nil
		},
		func() (bufmodule.ObjectData, error) {
			return v1BufLockObjectData, nil
		},
	), nil
}

// putModuleData puts the module data into the module cache.
//
// If moduleDataStore is configured to store the module data as tarballs, then a single tar
// for all module data is stored under the module key.
//
// If moduleDataStore is configured to store individual files, then it takes the following steps:
//
//  1. Acquire a shared lock on the module lock file for the module key.
//  2. Attempt to read the module.yaml file to ensure that there is no valid module already
//     stored in the cache. The module.yaml file is always written last in the cache, so if
//     it is present, then valid module data is present, and no new module data is written.
//  3. If no module.yaml is present, then we release the shared lock and acquire an exclusive
//     lock on the module lock file for writing module data.
//  4. Once the exclusive lock is acquired, we do another check to ensure that there is no
//     module.yaml present. This is because the shared lock is not upgraded to an exclusive
//     lock, we released the shared lock before acquiring the exclusive lock, so to ensure
//     there are absolutely no race conditions, we do another check.
//  5. Once we determine there is no module.yaml present, we proceed to writing the module
//     data to the cache.
//  6. We write the module.yaml after we've written all other module data files.
func (p *moduleDataStore) putModuleData(
	ctx context.Context,
	moduleData bufmodule.ModuleData,
) (retErr error) {
	moduleKey := moduleData.ModuleKey()
	var moduleCacheBucket storage.ReadWriteBucket
	var err error
	if p.tar {
		var callback func(ctx context.Context) error
		moduleCacheBucket, callback = p.getWriteBucketAndCallbackForTar(moduleKey)
		defer func() {
			if retErr == nil {
				// Only call the callback if we have had no error.
				retErr = multierr.Append(retErr, callback(ctx))
			}
		}()
	} else {
		dirPath, err := getModuleDataStoreDirPath(moduleKey)
		if err != nil {
			return err
		}
		p.logDebugModuleKey(
			ctx,
			moduleKey,
			"module data store dir read write bucket",
			slog.String("dirPath", dirPath),
		)
		moduleCacheBucket = storage.MapReadWriteBucket(p.bucket, storage.MapOnPrefix(dirPath))
		moduleDataStoreDirLockPath, err := getModuleDataStoreDirLockPath(moduleKey)
		if err != nil {
			return err
		}
		// Acquire shared lock to check for a valid module.yaml before writing to the module cache.
		readUnlocker, err := p.locker.RLock(ctx, moduleDataStoreDirLockPath)
		if err != nil {
			return err
		}
		defer func() {
			if readUnlocker != nil {
				if err := readUnlocker.Unlock(); err != nil {
					retErr = multierr.Append(retErr, err)
				}
			}
		}()
		data, err := storage.ReadPath(ctx, moduleCacheBucket, externalModuleDataFileName)
		p.logDebugModuleKey(
			ctx,
			moduleKey,
			fmt.Sprintf("module data store put read check %s", externalModuleDataFileName),
			slog.Bool("found", err == nil),
			slogext.ErrorAttr(err),
		)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if err == nil {
			var externalModuleData externalModuleData
			if err := encoding.UnmarshalYAMLNonStrict(data, &externalModuleData); err != nil {
				return err
			}
			// If a valid module.yaml is present, since module.yaml is always written last, we
			// assume that there is valid module data, and we do not attempt to write new data here.
			if externalModuleData.isValid() {
				return nil
			}
		}
		// Otherwise, release shared lock and set readUnlocker to nil in order to acquire an
		// exclusive lock for writing module data to the cache. filelock does not allow us to
		// upgrade the shared lock to an exclusive lock, so we need to release the shared lock
		// before acquiring an exclusive lock.
		if readUnlocker != nil {
			err := readUnlocker.Unlock()
			readUnlocker = nil // unset the readUnlocker since we are upgrading the lock
			if err != nil {
				return err
			}
		}
		// Acquire exclusive lock on module lock file for writing module data to the cache.
		unlocker, err := p.locker.Lock(ctx, moduleDataStoreDirLockPath)
		if err != nil {
			return err
		}
		defer func() {
			if err := unlocker.Unlock(); err != nil {
				retErr = multierr.Append(retErr, err)
			}
		}()
		// Before we start writing module data to the cache, we first check to see if module.yaml
		// is present again after acquiring the exclusive lock.
		// This is because the shared lock was released before acquiring the exclusive lock,
		// and we need to make sure no valid module data was written in the interim.
		data, err = storage.ReadPath(ctx, moduleCacheBucket, externalModuleDataFileName)
		p.logDebugModuleKey(
			ctx,
			moduleKey,
			fmt.Sprintf("module data store put check %s", externalModuleDataFileName),
			slog.Bool("found", err == nil),
			slogext.ErrorAttr(err),
		)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if err == nil {
			var externalModuleData externalModuleData
			if err := encoding.UnmarshalYAMLNonStrict(data, &externalModuleData); err != nil {
				return err
			}
			// If a valid module.yaml is present, then we do not overwrite with new data.
			if externalModuleData.isValid() {
				return nil
			}
		}
	}
	// Proceed to writing module data.
	depModuleKeys, err := moduleData.DeclaredDepModuleKeys()
	if err != nil {
		return err
	}
	externalModuleData := externalModuleData{
		Version: externalModuleDataVersion,
		Deps:    make([]externalModuleDataDep, len(depModuleKeys)),
	}

	for i, depModuleKey := range depModuleKeys {
		digest, err := depModuleKey.Digest()
		if err != nil {
			return err
		}
		externalModuleData.Deps[i] = externalModuleDataDep{
			Name:   depModuleKey.ModuleFullName().String(),
			Commit: uuidutil.ToDashless(depModuleKey.CommitID()),
			Digest: digest.String(),
		}
	}

	filesBucket, err := moduleData.Bucket()
	if err != nil {
		return err
	}
	if _, err := storage.Copy(
		ctx,
		filesBucket,
		storage.MapWriteBucket(moduleCacheBucket, storage.MapOnPrefix(externalModuleDataFilesDir)),
	); err != nil {
		return err
	}
	externalModuleData.FilesDir = externalModuleDataFilesDir

	v1BufYAMLObjectData, err := moduleData.V1Beta1OrV1BufYAMLObjectData()
	if err != nil {
		return err
	}
	if v1BufYAMLObjectData != nil {
		v1BufYAMLFilePath := normalpath.Join(externalModuleDataV1BufYAMLDir, v1BufYAMLObjectData.Name())
		if err := storage.PutPath(ctx, moduleCacheBucket, v1BufYAMLFilePath, v1BufYAMLObjectData.Data()); err != nil {
			return err
		}
		externalModuleData.V1BufYAMLFile = v1BufYAMLFilePath
	}

	v1BufLockObjectData, err := moduleData.V1Beta1OrV1BufLockObjectData()
	if err != nil {
		return err
	}
	if v1BufLockObjectData != nil {
		v1BufLockFilePath := normalpath.Join(externalModuleDataV1BufLockDir, v1BufLockObjectData.Name())
		if err := storage.PutPath(ctx, moduleCacheBucket, v1BufLockFilePath, v1BufLockObjectData.Data()); err != nil {
			return err
		}
		externalModuleData.V1BufLockFile = v1BufLockFilePath
	}

	data, err := encoding.MarshalYAML(externalModuleData)
	if err != nil {
		return err
	}
	// Put the module.yaml last, so that we only have a module.yaml if the cache is finished writing.
	// We can use the existence of the module.yaml file to say whether or not the cache contains a
	// given ModuleKey, otherwise we overwrite any contents in the cache.
	return storage.PutPath(
		ctx,
		moduleCacheBucket,
		externalModuleDataFileName,
		data,
		storage.PutWithAtomic(),
	)
}

// May return fs.ErrNotExist error if tar not found.
func (p *moduleDataStore) getReadBucketForTar(
	ctx context.Context,
	moduleKey bufmodule.ModuleKey,
) (_ storage.ReadBucket, retErr error) {
	tarPath, err := getModuleDataStoreTarPath(moduleKey)
	if err != nil {
		return nil, err
	}
	defer func() {
		p.logDebugModuleKey(
			ctx,
			moduleKey,
			"module data store get tar read bucket",
			slog.String("tarPath", tarPath),
			slog.Bool("found", retErr == nil),
			slog.Any("error", retErr),
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
	); err != nil {
		return nil, err
	}
	return readWriteBucket, nil
}

func (p *moduleDataStore) getWriteBucketAndCallbackForTar(
	moduleKey bufmodule.ModuleKey,
) (storage.ReadWriteBucket, func(context.Context) error) {
	readWriteBucket := storagemem.NewReadWriteBucket()
	return readWriteBucket, func(ctx context.Context) (retErr error) {
		tarPath, err := getModuleDataStoreTarPath(moduleKey)
		if err != nil {
			return err
		}
		defer func() {
			p.logDebugModuleKey(
				ctx,
				moduleKey,
				"module data store put tar to write bucket",
				slog.String("tarPath", tarPath),
				slog.Bool("found", retErr == nil),
				slog.Any("error", retErr),
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

func (p *moduleDataStore) logDebugModuleKey(ctx context.Context, moduleKey bufmodule.ModuleKey, message string, fields ...any) {
	logDebugModuleKey(ctx, p.logger, moduleKey, message, fields...)
}

// Returns the module's path within the store if storing individual files.
//
// This is "digestType/registry/owner/name/dashlessCommitID",
// e.g. the module "buf.build/acme/weather" with commit "12345-abcde" and digest
// type "b5" will return "b5/buf.build/acme/weather/12345abcde".
func getModuleDataStoreDirPath(moduleKey bufmodule.ModuleKey) (string, error) {
	digest, err := moduleKey.Digest()
	if err != nil {
		return "", err
	}
	return normalpath.Join(
		digest.Type().String(),
		moduleKey.ModuleFullName().Registry(),
		moduleKey.ModuleFullName().Owner(),
		moduleKey.ModuleFullName().Name(),
		uuidutil.ToDashless(moduleKey.CommitID()),
	), nil
}

// Returns the module's path within the store if storing tar files.
//
// This is "registry/owner/name/dashlessCommitID.tar",
// e.g. the module "buf.build/acme/weather" with commit "12345-abcde" and digest
// type "b5" will return "b5/buf.build/acme/weather/12345abcde.tar".
func getModuleDataStoreTarPath(moduleKey bufmodule.ModuleKey) (string, error) {
	digest, err := moduleKey.Digest()
	if err != nil {
		return "", err
	}
	return normalpath.Join(
		digest.Type().String(),
		moduleKey.ModuleFullName().Registry(),
		moduleKey.ModuleFullName().Owner(),
		moduleKey.ModuleFullName().Name(),
		uuidutil.ToDashless(moduleKey.CommitID())+".tar",
	), nil
}

func getDeclaredDepModuleKeyForExternalModuleDataDep(dep externalModuleDataDep) (bufmodule.ModuleKey, error) {
	if dep.Name == "" {
		return nil, errors.New("no module name specified")
	}
	moduleFullName, err := bufmodule.ParseModuleFullName(dep.Name)
	if err != nil {
		return nil, fmt.Errorf("invalid module name: %w", err)
	}
	if dep.Commit == "" {
		return nil, fmt.Errorf("no commit specified for module %s", moduleFullName.String())
	}
	if dep.Digest == "" {
		return nil, fmt.Errorf("no digest specified for module %s", moduleFullName.String())
	}
	digest, err := bufmodule.ParseDigest(dep.Digest)
	if err != nil {
		return nil, err
	}
	commitID, err := uuidutil.FromDashless(dep.Commit)
	if err != nil {
		return nil, err
	}
	return bufmodule.NewModuleKey(
		moduleFullName,
		commitID,
		func() (bufmodule.Digest, error) {
			return digest, nil
		},
	)
}

func getModuleDataStoreDirLockPath(moduleKey bufmodule.ModuleKey) (string, error) {
	moduleDataStoreDirPath, err := getModuleDataStoreDirPath(moduleKey)
	if err != nil {
		return "", err
	}
	return moduleDataStoreDirPath + externalModuleDataLockFileExt, nil
}

// externalModuleData is the store representation of a ModuleData.
//
// We could use a protobuf Message for this.
//
// Note that we do not want to use bufconfig.BufLockFile. This would hard-link the API
// and persistence layers, and a bufconfig.BufLockFile does not have all the information that
// a bufmodule.ModuleData has.
type externalModuleData struct {
	Version       string                  `json:"version,omitempty" yaml:"version,omitempty"`
	FilesDir      string                  `json:"files_dir,omitempty" yaml:"files_dir,omitempty"`
	Deps          []externalModuleDataDep `json:"deps,omitempty" yaml:"deps,omitempty"`
	V1BufYAMLFile string                  `json:"v1_buf_yaml_file,omitempty" yaml:"v1_buf_yaml_file,omitempty"`
	V1BufLockFile string                  `json:"v1_buf_lock_file,omitempty" yaml:"v1_buf_lock_file,omitempty"`
}

// isValid returns true if all the information we currently expect to be on
// an externalModuleData is present, and the version matches.
//
// If we add to externalModuleData over time or change the version, old values will be
// incomplete, and we will auto-evict them from the store.
func (e externalModuleData) isValid() bool {
	for _, dep := range e.Deps {
		if !dep.isValid() {
			return false
		}
	}
	return e.Version == externalModuleDataVersion &&
		len(e.FilesDir) > 0
}

// externalModuleDataDep represents a dependency.
type externalModuleDataDep struct {
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	// Dashless
	Commit string `json:"commit,omitempty" yaml:"commit,omitempty"`
	Digest string `json:"digest,omitempty" yaml:"digest,omitempty"`
}

func (e externalModuleDataDep) isValid() bool {
	return len(e.Name) > 0 &&
		len(e.Commit) > 0 &&
		len(e.Digest) > 0
}

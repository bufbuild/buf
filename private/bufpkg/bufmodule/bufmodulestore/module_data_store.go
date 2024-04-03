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

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagearchive"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

var (
	externalModuleDataVersion      = "v1"
	externalModuleDataFileName     = "module.yaml"
	externalModuleDataFilesDir     = "files"
	externalModuleDataV1BufYAMLDir = "v1_buf_yaml"
	externalModuleDataV1BufLockDir = "v1_buf_lock"
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

func (p *moduleDataStore) GetModuleDatasForModuleKeys(
	ctx context.Context,
	moduleKeys []bufmodule.ModuleKey,
) ([]bufmodule.ModuleData, []bufmodule.ModuleKey, error) {
	var foundModuleDatas []bufmodule.ModuleData
	var notFoundModuleKeys []bufmodule.ModuleKey
	for _, moduleKey := range moduleKeys {
		moduleData, err := p.getModuleDataForModuleKey(ctx, moduleKey)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, nil, err
			}
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
				return nil, p.deleteInvalidModuleData(ctx, moduleKey, err)
			}
			return nil, err
		}
	} else {
		moduleCacheBucket, err = p.getReadWriteBucketForDir(moduleKey)
		// Not checking for fs.ErrNotExist. Function only returns error on actual system error.
		if err != nil {
			return nil, err
		}
	}
	defer func() {
		if retErr != nil {
			retErr = p.deleteInvalidModuleData(ctx, moduleKey, retErr)
		}
	}()
	data, err := storage.ReadPath(ctx, moduleCacheBucket, externalModuleDataFileName)
	p.logDebugModuleKey(
		moduleKey,
		fmt.Sprintf("module data store get %s", externalModuleDataFileName),
		zap.Bool("found", err == nil),
		zap.Error(err),
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

func (p *moduleDataStore) deleteInvalidModuleData(
	ctx context.Context,
	moduleKey bufmodule.ModuleKey,
	invalidErr error,
) (retErr error) {
	p.logDebugModuleKey(
		moduleKey,
		"module data store invalid module data",
		zap.Error(invalidErr),
	)
	defer func() {
		if retErr != nil {
			// Do not return error, just log. We always returns a file not found error.
			p.logDebugModuleKey(
				moduleKey,
				"module data store could not delete module data",
				zap.Error(retErr),
			)
		}
		// This will act as if the file is not found.
		retErr = &fs.PathError{Op: "read", Path: moduleKey.String(), Err: fs.ErrNotExist}
	}()

	if p.tar {
		tarPath, err := getModuleDataStoreTarPath(moduleKey)
		if err != nil {
			return err
		}
		return p.bucket.Delete(ctx, tarPath)
	}
	dirPath, err := getModuleDataStoreDirPath(moduleKey)
	if err != nil {
		return err
	}
	return p.bucket.DeleteAll(ctx, dirPath)
}

func (p *moduleDataStore) putModuleData(
	ctx context.Context,
	moduleData bufmodule.ModuleData,
) (retErr error) {
	moduleKey := moduleData.ModuleKey()
	var moduleCacheBucket storage.WriteBucket
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
		moduleCacheBucket, err = p.getReadWriteBucketForDir(moduleKey)
		if err != nil {
			return err
		}
	}
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
		storage.CopyWithAtomic(),
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
	return storage.PutPath(ctx, moduleCacheBucket, externalModuleDataFileName, data)
}

// Only returns error on actual system error.
func (p *moduleDataStore) getReadWriteBucketForDir(
	moduleKey bufmodule.ModuleKey,
) (storage.ReadWriteBucket, error) {
	dirPath, err := getModuleDataStoreDirPath(moduleKey)
	if err != nil {
		return nil, err
	}
	p.logDebugModuleKey(
		moduleKey,
		"module data store dir read write bucket",
		zap.String("dirPath", dirPath),
	)
	return storage.MapReadWriteBucket(p.bucket, storage.MapOnPrefix(dirPath)), nil
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
			moduleKey,
			"module data store get tar read bucket",
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
	); err != nil {
		return nil, err
	}
	return readWriteBucket, nil
}

func (p *moduleDataStore) getWriteBucketAndCallbackForTar(
	moduleKey bufmodule.ModuleKey,
) (storage.WriteBucket, func(context.Context) error) {
	readWriteBucket := storagemem.NewReadWriteBucket()
	return readWriteBucket, func(ctx context.Context) (retErr error) {
		tarPath, err := getModuleDataStoreTarPath(moduleKey)
		if err != nil {
			return err
		}
		defer func() {
			p.logDebugModuleKey(
				moduleKey,
				"module data store put tar to write bucket",
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

func (p *moduleDataStore) logDebugModuleKey(moduleKey bufmodule.ModuleKey, message string, fields ...zap.Field) {
	logDebugModuleKey(p.logger, moduleKey, message, fields...)
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

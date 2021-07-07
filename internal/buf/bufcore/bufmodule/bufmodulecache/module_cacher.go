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
	"io/ioutil"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/buf/buflock"
	"github.com/bufbuild/buf/internal/pkg/filelock"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/multierr"
)

type moduleCacher struct {
	dataReadWriteBucket storage.ReadWriteBucket
	sumReadWriteBucket  storage.ReadWriteBucket
	fileLocker          filelock.Locker
}

func newModuleCacher(
	dataReadWriteBucket storage.ReadWriteBucket,
	sumReadWriteBucket storage.ReadWriteBucket,
	fileLocker filelock.Locker,
) *moduleCacher {
	return &moduleCacher{
		dataReadWriteBucket: dataReadWriteBucket,
		sumReadWriteBucket:  sumReadWriteBucket,
		fileLocker:          fileLocker,
	}
}

func (m *moduleCacher) GetModule(
	ctx context.Context,
	modulePin bufmodule.ModulePin,
) (bufmodule.Module, error) {
	module, storedDigest, err := m.getModuleAndStoredDigest(ctx, modulePin)
	if err != nil {
		return nil, err
	}
	// This can happen if we couldn't find the sum file, which means
	// we are in an invalid state
	if storedDigest == "" {
		if err := m.deleteInvalidModule(ctx, modulePin); err != nil {
			return nil, err
		}
		return nil, storage.NewErrNotExist(newCacheKey(modulePin))
	}
	digest, err := bufmodule.ModuleDigestB2(ctx, module)
	if err != nil {
		return nil, err
	}
	if digest != storedDigest {
		if err := m.deleteInvalidModule(ctx, modulePin); err != nil {
			return nil, err
		}
		return nil, storage.NewErrNotExist(newCacheKey(modulePin))
	}
	return module, nil
}

func (m *moduleCacher) PutModule(
	ctx context.Context,
	modulePin bufmodule.ModulePin,
	module bufmodule.Module,
) (retErr error) {
	modulePath := newCacheKey(modulePin)
	digest, err := bufmodule.ModuleDigestB2(ctx, module)
	if err != nil {
		return err
	}

	unlocker, err := m.fileLocker.Lock(ctx, modulePath)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, unlocker.Unlock())
	}()

	dataReadWriteBucket := storage.MapReadWriteBucket(
		m.dataReadWriteBucket,
		storage.MapOnPrefix(modulePath),
	)
	exists, err := storage.Exists(ctx, dataReadWriteBucket, buflock.ExternalConfigFilePath)
	if err != nil {
		return err
	}
	if exists {
		// If the module already exists in the cache, we want to make sure we delete it
		// before putting new data
		if err := dataReadWriteBucket.DeleteAll(ctx, ""); err != nil {
			return err
		}
	}
	if err := bufmodule.ModuleToBucket(ctx, module, dataReadWriteBucket); err != nil {
		return err
	}
	// This will overwrite if necessary
	if err := storage.PutPath(ctx, m.sumReadWriteBucket, modulePath, []byte(digest)); err != nil {
		return multierr.Append(
			err,
			// Try to clean up after ourselves.
			dataReadWriteBucket.DeleteAll(ctx, ""),
		)
	}
	return nil
}

// Putting these two into one function because these are the two we need a read lock for.
// If  no digest is returned, we should assume we should delete the module from Data.
func (m *moduleCacher) getModuleAndStoredDigest(
	ctx context.Context,
	modulePin bufmodule.ModulePin,
) (_ bufmodule.Module, _ string, retErr error) {
	modulePath := newCacheKey(modulePin)

	unlocker, err := m.fileLocker.RLock(ctx, modulePath)
	if err != nil {
		return nil, "", err
	}
	defer func() {
		retErr = multierr.Append(retErr, unlocker.Unlock())
	}()

	dataReadWriteBucket := storage.MapReadWriteBucket(
		m.dataReadWriteBucket,
		storage.MapOnPrefix(modulePath),
	)
	exists, err := storage.Exists(ctx, dataReadWriteBucket, buflock.ExternalConfigFilePath)
	if err != nil {
		return nil, "", err
	}
	if !exists {
		return nil, "", storage.NewErrNotExist(modulePath)
	}
	module, err := bufmodule.NewModuleForBucket(
		ctx,
		dataReadWriteBucket,
		bufmodule.ModuleWithModuleIdentityAndCommit(modulePin, modulePin.Commit()),
	)
	if err != nil {
		return nil, "", err
	}

	digestReadObjectCloser, err := m.sumReadWriteBucket.Get(ctx, modulePath)
	if err != nil {
		if storage.IsNotExist(err) {
			// This signals that we do not have a digest, which should signal to the
			// calling function to delete what we found from data.
			return nil, "", nil
		}
		return nil, "", err
	}
	defer func() {
		retErr = multierr.Append(retErr, digestReadObjectCloser.Close())
	}()
	digestData, err := ioutil.ReadAll(digestReadObjectCloser)
	if err != nil {
		return nil, "", err
	}

	return module, string(digestData), nil
}

func (m *moduleCacher) deleteInvalidModule(
	ctx context.Context,
	modulePin bufmodule.ModulePin,
) (retErr error) {
	modulePath := newCacheKey(modulePin)

	unlocker, err := m.fileLocker.Lock(ctx, modulePath)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, unlocker.Unlock())
	}()
	dataReadWriteBucket := storage.MapReadWriteBucket(
		m.dataReadWriteBucket,
		storage.MapOnPrefix(modulePath),
	)
	var deleteErr error
	// Ignore if this doesn't exist, we're just cleaning up
	if err := dataReadWriteBucket.DeleteAll(ctx, ""); err != nil && !storage.IsNotExist(err) {
		deleteErr = multierr.Append(deleteErr, err)
	}
	// Ignore if this doesn't exist, we're just cleaning up
	if err := m.sumReadWriteBucket.Delete(ctx, modulePath); err != nil && !storage.IsNotExist(err) {
		deleteErr = multierr.Append(deleteErr, err)
	}
	return deleteErr
}

// newCacheKey returns the key associated with the given module pin.
// The cache key is of the form: remote/owner/repository/commit.
func newCacheKey(modulePin bufmodule.ModulePin) string {
	return normalpath.Join(modulePin.Remote(), modulePin.Owner(), modulePin.Repository(), modulePin.Commit())
}

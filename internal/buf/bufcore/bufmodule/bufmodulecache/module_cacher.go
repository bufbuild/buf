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

	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/pkg/filelock"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/multierr"
)

type moduleCacher struct {
	readWriteBucket storage.ReadWriteBucket
	fileLocker      filelock.Locker
}

func newModuleCacher(
	readWriteBucket storage.ReadWriteBucket,
	fileLocker filelock.Locker,
) *moduleCacher {
	return &moduleCacher{
		readWriteBucket: readWriteBucket,
		fileLocker:      fileLocker,
	}
}

func (m *moduleCacher) GetModule(
	ctx context.Context,
	modulePin bufmodule.ModulePin,
) (_ bufmodule.Module, retErr error) {
	modulePath := newCacheKey(modulePin)
	unlocker, err := m.fileLocker.RLock(ctx, modulePath)
	if err != nil {
		return nil, err
	}
	// We cannot defer the unlock here because the conditional write lock would never succeed.
	// Instead, we manually call unlocker.Unlock in each case below.
	readWriteBucket := storage.MapReadWriteBucket(m.readWriteBucket, storage.MapOnPrefix(modulePath))
	exists, err := storage.Exists(ctx, readWriteBucket, bufmodule.LockFilePath)
	if err != nil {
		return nil, multierr.Append(err, unlocker.Unlock())
	}
	if !exists {
		return nil, multierr.Append(storage.NewErrNotExist(modulePath), unlocker.Unlock())
	}
	module, err := bufmodule.NewModuleForBucket(ctx, readWriteBucket)
	if err != nil {
		return nil, multierr.Append(err, unlocker.Unlock())
	}
	if err := unlocker.Unlock(); err != nil {
		return nil, err
	}
	if err := bufmodule.ValidateModuleMatchesDigest(ctx, module, modulePin); err != nil {
		// Delete module if it's invalid.
		unlocker, lockErr := m.fileLocker.Lock(ctx, modulePath)
		if lockErr != nil {
			err = multierr.Append(err, lockErr)
		}
		defer func() {
			retErr = multierr.Append(retErr, unlocker.Unlock())
		}()
		return nil, multierr.Append(err, readWriteBucket.DeleteAll(ctx, ""))
	}
	return module, nil
}

func (m *moduleCacher) PutModule(
	ctx context.Context,
	modulePin bufmodule.ModulePin,
	module bufmodule.Module,
) (retErr error) {
	if err := bufmodule.ValidateModuleMatchesDigest(ctx, module, modulePin); err != nil {
		return err
	}
	modulePath := newCacheKey(modulePin)
	unlocker, err := m.fileLocker.Lock(ctx, modulePath)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, unlocker.Unlock())
	}()
	readWriteBucket := storage.MapReadWriteBucket(m.readWriteBucket, storage.MapOnPrefix(modulePath))
	return bufmodule.ModuleToBucket(ctx, module, readWriteBucket)
}

// newCacheKey returns the key associated with the given module pin.
// The cache key is of the form: owner/repository/commit.
func newCacheKey(modulePin bufmodule.ModulePin) string {
	return normalpath.Join(modulePin.Owner(), modulePin.Repository(), modulePin.Commit())
}

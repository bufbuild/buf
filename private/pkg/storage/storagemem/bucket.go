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

package storagemem

import (
	"context"
	"io/fs"
	"sort"
	"sync"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem/internal"
	"github.com/bufbuild/buf/private/pkg/storage/storageutil"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

type bucket struct {
	pathToImmutableObject map[string]*internal.ImmutableObject
	lock                  sync.RWMutex
}

func newBucket(pathToImmutableObject map[string]*internal.ImmutableObject) *bucket {
	if pathToImmutableObject == nil {
		pathToImmutableObject = make(map[string]*internal.ImmutableObject)
	}
	return &bucket{
		pathToImmutableObject: pathToImmutableObject,
	}
}

func (b *bucket) Get(ctx context.Context, path string) (storage.ReadObjectCloser, error) {
	immutableObject, err := b.readLockAndGetImmutableObject(ctx, path)
	if err != nil {
		return nil, err
	}
	return newReadObjectCloser(immutableObject), nil
}

func (b *bucket) Stat(ctx context.Context, path string) (storage.ObjectInfo, error) {
	return b.readLockAndGetImmutableObject(ctx, path)
}

func (b *bucket) Walk(ctx context.Context, prefix string, f func(storage.ObjectInfo) error) error {
	prefix, err := storageutil.ValidatePrefix(prefix)
	if err != nil {
		return err
	}
	walkChecker := storageutil.NewWalkChecker()
	b.lock.RLock()
	defer b.lock.RUnlock()
	// To ensure same iteration order.
	// We could create this in-place during puts with an insertion sort if this
	// gets to be time prohibitive.
	paths := make([]string, 0, len(b.pathToImmutableObject))
	for path := range b.pathToImmutableObject {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		immutableObject, ok := b.pathToImmutableObject[path]
		if !ok {
			// this is a system error
			return syserror.Newf("path %q not in pathToObject", path)
		}
		if err := walkChecker.Check(ctx); err != nil {
			return err
		}
		if !normalpath.EqualsOrContainsPath(prefix, path, normalpath.Relative) {
			continue
		}
		if err := f(immutableObject); err != nil {
			return err
		}
	}
	return nil
}

func (b *bucket) Put(ctx context.Context, path string, _ ...storage.PutOption) (storage.WriteObjectCloser, error) {
	// No need to lock as we do no modifications until close
	path, err := storageutil.ValidatePath(path)
	if err != nil {
		return nil, err
	}
	// storagemem writes are already atomic - don't need special handling for PutWithAtomic.
	return newWriteObjectCloser(b, path), nil
}

func (b *bucket) Delete(ctx context.Context, path string) error {
	path, err := storageutil.ValidatePath(path)
	if err != nil {
		return err
	}
	b.lock.Lock()
	defer b.lock.Unlock()
	if _, ok := b.pathToImmutableObject[path]; !ok {
		return &fs.PathError{Op: "stat", Path: path, Err: fs.ErrNotExist}
	}
	// Note that if there is an existing reader for an object of the same path,
	// that reader will continue to read the original file, but we accept this
	// as no less consistent than os mechanics.
	delete(b.pathToImmutableObject, path)
	return nil
}

func (b *bucket) DeleteAll(ctx context.Context, prefix string) error {
	prefix, err := storageutil.ValidatePrefix(prefix)
	if err != nil {
		return err
	}
	b.lock.Lock()
	defer b.lock.Unlock()
	for path := range b.pathToImmutableObject {
		if normalpath.EqualsOrContainsPath(prefix, path, normalpath.Relative) {
			// Note that if there is an existing reader for an object of the same path,
			// that reader will continue to read the original file, but we accept this
			// as no less consistent than os mechanics.
			delete(b.pathToImmutableObject, path)
		}
	}
	return nil
}

func (*bucket) SetExternalAndLocalPathsSupported() bool {
	return true
}

func (b *bucket) ToReadBucket() (storage.ReadBucket, error) {
	return b, nil
}

func (b *bucket) readLockAndGetImmutableObject(ctx context.Context, path string) (*internal.ImmutableObject, error) {
	path, err := storageutil.ValidatePath(path)
	if err != nil {
		return nil, err
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	immutableObject, ok := b.pathToImmutableObject[path]
	if !ok {
		// it would be nice if this was external path for every bucket
		// the issue is here: we don't know the external path for memory buckets
		// because we store external paths individually, so if we do not have
		// an object, we do not have an external path
		return nil, &fs.PathError{Op: "read", Path: path, Err: fs.ErrNotExist}
	}
	return immutableObject, nil
}

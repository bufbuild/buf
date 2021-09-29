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

package storagemem

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem/internal"
	"github.com/bufbuild/buf/private/pkg/storage/storageutil"
)

var errDuplicatePath = errors.New("duplicate path")

type readBucket struct {
	pathToImmutableObject map[string]*internal.ImmutableObject
	paths                 []string
}

func newReadBucketForPathToData(pathToData map[string][]byte) (*readBucket, error) {
	pathToImmutableObject := make(map[string]*internal.ImmutableObject, len(pathToData))
	for path, data := range pathToData {
		pathToImmutableObject[path] = internal.NewImmutableObject(path, "", data)
	}
	return newReadBucket(pathToImmutableObject)
}

func newReadBucket(
	pathToImmutableObject map[string]*internal.ImmutableObject,
) (*readBucket, error) {
	paths := make([]string, 0, len(pathToImmutableObject))
	for path := range pathToImmutableObject {
		path, err := storageutil.ValidatePath(path)
		if err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return &readBucket{
		pathToImmutableObject: pathToImmutableObject,
		paths:                 paths,
	}, nil
}

func (b *readBucket) Get(ctx context.Context, path string) (storage.ReadObjectCloser, error) {
	immutableObject, err := b.getImmutableObject(ctx, path)
	if err != nil {
		return nil, err
	}
	return newReadObjectCloser(immutableObject), nil
}

func (b *readBucket) Stat(ctx context.Context, path string) (storage.ObjectInfo, error) {
	return b.getImmutableObject(ctx, path)
}

func (b *readBucket) Walk(ctx context.Context, prefix string, f func(storage.ObjectInfo) error) error {
	prefix, err := storageutil.ValidatePrefix(prefix)
	if err != nil {
		return err
	}
	walkChecker := storageutil.NewWalkChecker()
	for _, path := range b.paths {
		immutableObject, ok := b.pathToImmutableObject[path]
		if !ok {
			// this is a system error
			return fmt.Errorf("path %q not in pathToObject", path)
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

func (b *readBucket) getImmutableObject(ctx context.Context, path string) (*internal.ImmutableObject, error) {
	path, err := storageutil.ValidatePath(path)
	if err != nil {
		return nil, err
	}
	immutableObject, ok := b.pathToImmutableObject[path]
	if !ok {
		// it would be nice if this was external path for every bucket
		// the issue is here: we don't know the external path for memory buckets
		// because we store external paths individually, so if we do not have
		// an object, we do not have an external path
		return nil, storage.NewErrNotExist(path)
	}
	return immutableObject, nil
}

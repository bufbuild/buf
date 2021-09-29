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
	"sync"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageutil"
)

type readBucketBuilder struct {
	pathToImmutableObject map[string]*immutableObject
	lock                  sync.Mutex
}

func newReadBucketBuilder() *readBucketBuilder {
	return &readBucketBuilder{
		pathToImmutableObject: make(map[string]*immutableObject),
	}
}

func (b *readBucketBuilder) Put(ctx context.Context, path string) (storage.WriteObjectCloser, error) {
	path, err := normalpath.NormalizeAndValidate(path)
	if err != nil {
		return nil, err
	}
	if path == "." {
		return nil, errors.New("cannot put root")
	}
	return newWriteObjectCloser(b, path), nil
}

func (b *readBucketBuilder) Delete(ctx context.Context, path string) error {
	path, err := normalpath.NormalizeAndValidate(path)
	if err != nil {
		return err
	}
	b.lock.Lock()
	defer b.lock.Unlock()
	if _, ok := b.pathToImmutableObject[path]; !ok {
		return storage.NewErrNotExist(path)
	}
	delete(b.pathToImmutableObject, path)
	return nil
}

func (b *readBucketBuilder) DeleteAll(ctx context.Context, prefix string) error {
	prefix, err := storageutil.ValidatePrefix(prefix)
	if err != nil {
		return err
	}
	b.lock.Lock()
	defer b.lock.Unlock()
	for path := range b.pathToImmutableObject {
		if normalpath.EqualsOrContainsPath(prefix, path, normalpath.Relative) {
			delete(b.pathToImmutableObject, path)
		}
	}
	return nil
}

func (*readBucketBuilder) SetExternalPathSupported() bool {
	return true
}

func (b *readBucketBuilder) ToReadBucket() (storage.ReadBucket, error) {
	return newReadBucket(b.pathToImmutableObject)
}

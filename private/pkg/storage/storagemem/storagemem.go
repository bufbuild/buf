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

// Package storagemem implements an in-memory storage Bucket.
package storagemem

import (
	"context"
	"errors"

	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem/internal"
	"github.com/bufbuild/buf/private/pkg/storage/storageutil"
)

var errDuplicatePath = errors.New("duplicate path")

// NewReadWriteBucket returns a new in-memory ReadWriteBucket.
func NewReadWriteBucket() storage.ReadWriteBucket {
	return newBucket(nil)
}

// NewReadBucket returns a new ReadBucket.
func NewReadBucket(pathToData map[string][]byte) (storage.ReadBucket, error) {
	pathToImmutableObject := make(map[string]*internal.ImmutableObject, len(pathToData))
	for path, data := range pathToData {
		path, err := storageutil.ValidatePath(path)
		if err != nil {
			return nil, err
		}
		// This could happen if two paths normalize to the same path.
		if _, ok := pathToImmutableObject[path]; ok {
			return nil, errDuplicatePath
		}
		pathToImmutableObject[path] = internal.NewImmutableObject(path, "", "", data)
	}
	return newBucket(pathToImmutableObject), nil
}

// CopyReadBucket will copy the input ReadBucket into an in-memory ReadBucket, if the input
// ReadBucket is not already an in-memory ReadBucket.
func CopyReadBucket(ctx context.Context, inputBucket storage.ReadBucket) (storage.ReadBucket, error) {
	if _, ok := inputBucket.(*bucket); ok {
		return inputBucket, nil
	}
	inMemoryBucket := newBucket(nil)
	if _, err := storage.Copy(ctx, inputBucket, inMemoryBucket, storage.CopyWithExternalAndLocalPaths()); err != nil {
		return nil, err
	}
	return inMemoryBucket, nil
}

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

// Package storagemem implements an in-memory storage Bucket.
package storagemem

import (
	"errors"

	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem/internal"
	"github.com/bufbuild/buf/private/pkg/storage/storageutil"
)

var errDuplicatePath = errors.New("duplicate path")

type options struct {
	pathToData map[string][]byte
}

// Option is provided by NewReadWriteBucketWithOptions options.
type Option interface {
	apply(*options)
}

type pathData map[string][]byte

func (pd pathData) apply(opts *options) {
	opts.pathToData = pd
}

// WithFiles adds files by path to their content into the bucket.
func WithFiles(pathToData map[string][]byte) Option {
	return (pathData)(pathToData)
}

// NewReadWriteBucket returns a new in-memory ReadWriteBucket.
// Deprecated: Use NewReadWriteBucketWithOptions without any options.
func NewReadWriteBucket() storage.ReadWriteBucket {
	return newBucket(nil)
}

// NewReadWriteBucketWithOptions returns a new in-memory ReadWriteBucket.
// Errors are returned with invalid options.
func NewReadWriteBucketWithOptions(opts ...Option) (storage.ReadWriteBucket, error) {
	opt := options{}
	for _, o := range opts {
		o.apply(&opt)
	}

	pathToImmutableObject := make(map[string]*internal.ImmutableObject, len(opt.pathToData))
	for path, data := range opt.pathToData {
		path, err := storageutil.ValidatePath(path)
		if err != nil {
			return nil, err
		}
		// This could happen if two paths normalize to the same path.
		if _, ok := pathToImmutableObject[path]; ok {
			return nil, errDuplicatePath
		}
		pathToImmutableObject[path] = internal.NewImmutableObject(path, "", data)
	}
	return newBucket(pathToImmutableObject), nil
}

// NewReadBucket returns a new ReadBucket.
func NewReadBucket(pathToData map[string][]byte) (storage.ReadBucket, error) {
	return NewReadWriteBucketWithOptions(WithFiles(pathToData))
}

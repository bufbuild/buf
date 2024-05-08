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

package storage

import (
	"context"

	"github.com/bufbuild/buf/private/pkg/storage/storageutil"
)

// StripReadBucketExternalPaths strips the differentiated ExternalPaths from objects
// returned from the ReadBucket, instead replacing them with the Paths.
//
// This is used in situations where the ExternalPath is actually i.e. in a cache, and
// you don't want to expose this information to callers.
func StripReadBucketExternalPaths(readBucket ReadBucket) ReadBucket {
	return newStripReadBucket(readBucket)
}

type stripReadBucket struct {
	delegate ReadBucket
}

func newStripReadBucket(delegate ReadBucket) *stripReadBucket {
	return &stripReadBucket{
		delegate: delegate,
	}
}

func (r *stripReadBucket) Get(ctx context.Context, path string) (ReadObjectCloser, error) {
	readObjectCloser, err := r.delegate.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	return stripReadObjectCloserExternalPath(readObjectCloser), nil
}

func (r *stripReadBucket) Stat(ctx context.Context, path string) (ObjectInfo, error) {
	objectInfo, err := r.delegate.Stat(ctx, path)
	if err != nil {
		return nil, err
	}
	return stripObjectInfoExternalPath(objectInfo), nil
}

func (r *stripReadBucket) Walk(ctx context.Context, prefix string, f func(ObjectInfo) error) error {
	return r.delegate.Walk(
		ctx,
		prefix,
		func(objectInfo ObjectInfo) error {
			return f(stripObjectInfoExternalPath(objectInfo))
		},
	)
}

func stripObjectInfoExternalPath(objectInfo ObjectInfo) ObjectInfo {
	path := objectInfo.Path()
	if path == objectInfo.ExternalPath() {
		return objectInfo
	}
	return storageutil.NewObjectInfo(path, path, objectInfo.LocalPath())
}

func stripReadObjectCloserExternalPath(readObjectCloser ReadObjectCloser) ReadObjectCloser {
	if readObjectCloser.Path() == readObjectCloser.ExternalPath() {
		return readObjectCloser
	}
	return compositeReadObjectCloser{stripObjectInfoExternalPath(readObjectCloser), readObjectCloser}
}

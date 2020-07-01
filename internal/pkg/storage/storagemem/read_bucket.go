// Copyright 2020 Buf Technologies, Inc.
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
	"bytes"
	"context"
	"errors"

	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/internal"
)

var errDuplicatePath = errors.New("duplicate path")

type readBucket struct {
	pathToObject map[string]*object
}

func newReadBucket(
	pathToData map[string][]byte,
	options ...ReadBucketOption,
) (*readBucket, error) {
	readBucketOptions := &readBucketOptions{}
	for _, option := range options {
		option(readBucketOptions)
	}
	pathToObject := make(map[string]*object, len(pathToData))
	for path, data := range pathToData {
		path, err := internal.ValidatePath(path)
		if err != nil {
			return nil, err
		}
		if _, ok := pathToObject[path]; ok {
			return nil, normalpath.NewError(path, errDuplicatePath)
		}
		externalPath := path
		if readBucketOptions.externalPathResolver != nil {
			externalPath, err = readBucketOptions.externalPathResolver(path)
			if err != nil {
				return nil, err
			}
		} else if len(readBucketOptions.pathToExternalPath) > 0 {
			if mapExternalPath := readBucketOptions.pathToExternalPath[path]; mapExternalPath != "" {
				externalPath = mapExternalPath
			}
		}
		pathToObject[path] = newObject(
			path,
			externalPath,
			data,
		)
	}
	return &readBucket{
		pathToObject: pathToObject,
	}, nil
}

func (b *readBucket) Get(ctx context.Context, path string) (storage.ReadObjectCloser, error) {
	object, err := b.getObject(ctx, path)
	if err != nil {
		return nil, err
	}
	return newReadObjectCloser(path, object), nil
}

func (b *readBucket) Stat(ctx context.Context, path string) (storage.ObjectInfo, error) {
	return b.getObject(ctx, path)
}

func (b *readBucket) Walk(ctx context.Context, prefix string, f func(storage.ObjectInfo) error) error {
	prefix, err := internal.ValidatePrefix(prefix)
	if err != nil {
		return err
	}
	walkChecker := internal.NewWalkChecker()
	for path, object := range b.pathToObject {
		if err := walkChecker.Check(ctx); err != nil {
			return err
		}
		if !normalpath.EqualsOrContainsPath(prefix, path) {
			continue
		}
		if err := f(object); err != nil {
			return err
		}
	}
	return nil
}

func (b *readBucket) getObject(ctx context.Context, path string) (*object, error) {
	path, err := internal.ValidatePath(path)
	if err != nil {
		return nil, err
	}
	object, ok := b.pathToObject[path]
	if !ok {
		// it would be nice if this was external path for every bucket
		// the issue is here: we don't know the external path for memory buckets
		// because we store external paths individually, so if we do not have
		// an object, we do not have an external path
		return nil, storage.NewErrNotExist(path)
	}
	return object, nil
}

type readObjectCloser struct {
	internal.ObjectInfo

	reader *bytes.Reader
	closed bool
}

func newReadObjectCloser(path string, object *object) *readObjectCloser {
	return &readObjectCloser{
		ObjectInfo: object.ObjectInfo,
		reader:     bytes.NewReader(object.data),
	}
}

func (r *readObjectCloser) Read(p []byte) (int, error) {
	if r.closed {
		return 0, storage.ErrClosed
	}
	return r.reader.Read(p)
}

func (r *readObjectCloser) Close() error {
	if r.closed {
		return storage.ErrClosed
	}
	r.closed = true
	return nil
}

type object struct {
	internal.ObjectInfo

	data []byte
}

func newObject(
	path string,
	externalPath string,
	data []byte,
) *object {
	return &object{
		ObjectInfo: internal.NewObjectInfo(
			uint32(len(data)),
			path,
			externalPath,
		),
		data: data,
	}
}

type readBucketOptions struct {
	externalPathResolver func(string) (string, error)
	pathToExternalPath   map[string]string
}

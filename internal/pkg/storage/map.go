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

package storage

import (
	"context"
	"errors"
	"io"

	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/storage/internal"
)

// Map maps the bucket.
//
// If the Mappers are empty, the original ReadBucket is returned.
// If there is more than one Mapper, the Mappers are called in order
// for UnmapFullPath, with the order reversed for MapPath and MapPrefix.
//
// That is, order these assuming you are starting with a full path and
// working to a path.
func Map(readBucket ReadBucket, mappers ...Mapper) ReadBucket {
	switch len(mappers) {
	case 0:
		return readBucket
	case 1:
		return newMapReadBucket(readBucket, mappers[0])
	default:
		return newMapReadBucket(readBucket, MapChain(mappers...))
	}
}

type mapReadBucket struct {
	delegate ReadBucket
	mapper   Mapper
}

func newMapReadBucket(
	delegate ReadBucket,
	mapper Mapper,
) *mapReadBucket {
	return &mapReadBucket{
		delegate: delegate,
		mapper:   mapper,
	}
}

func (m *mapReadBucket) Get(ctx context.Context, path string) (ReadObjectCloser, error) {
	fullPath, err := m.getFullPath(path)
	if err != nil {
		return nil, err
	}
	readObjectCloser, err := m.delegate.Get(ctx, fullPath)
	// TODO: if this is a path error, we should replace the path
	if err != nil {
		return nil, err
	}
	return replaceReadObjectCloserPath(readObjectCloser, path), nil
}

func (m *mapReadBucket) Stat(ctx context.Context, path string) (ObjectInfo, error) {
	fullPath, err := m.getFullPath(path)
	if err != nil {
		return nil, err
	}
	objectInfo, err := m.delegate.Stat(ctx, fullPath)
	// TODO: if this is a path error, we should replace the path
	if err != nil {
		return nil, err
	}
	return replaceObjectInfoPath(objectInfo, path), nil

}

func (m *mapReadBucket) Walk(ctx context.Context, prefix string, f func(ObjectInfo) error) error {
	prefix, err := normalpath.NormalizeAndValidate(prefix)
	if err != nil {
		return err
	}
	fullPrefix, matches := m.mapper.MapPrefix(prefix)
	if !matches {
		return nil
	}
	return m.delegate.Walk(
		ctx,
		fullPrefix,
		func(objectInfo ObjectInfo) error {
			path, matches, err := m.mapper.UnmapFullPath(objectInfo.Path())
			if err != nil {
				return err
			}
			if !matches {
				return nil
			}
			return f(replaceObjectInfoPath(objectInfo, path))
		},
	)
}

func (m *mapReadBucket) getFullPath(path string) (string, error) {
	path, err := normalpath.NormalizeAndValidate(path)
	if err != nil {
		return "", err
	}
	if path == "." {
		return "", errors.New("cannot get root")
	}
	fullPath, matches := m.mapper.MapPath(path)
	if !matches {
		return "", NewErrNotExist(path)
	}
	return fullPath, nil
}

func replaceObjectInfoPath(objectInfo ObjectInfo, path string) internal.ObjectInfo {
	return internal.NewObjectInfo(
		objectInfo.Size(),
		path,
		objectInfo.ExternalPath(),
	)
}

func replaceReadObjectCloserPath(readObjectCloser ReadObjectCloser, path string) ReadObjectCloser {
	return compositeReadObjectCloser{replaceObjectInfoPath(readObjectCloser, path), readObjectCloser}
}

type compositeReadObjectCloser struct {
	internal.ObjectInfo
	io.ReadCloser
}

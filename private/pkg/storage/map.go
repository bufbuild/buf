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
	"errors"
	"fmt"
	"io"
	"io/fs"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage/storageutil"
)

// MapReadBucket maps the ReadBucket.
//
// If the Mappers are empty, the original ReadBucket is returned.
// If there is more than one Mapper, the Mappers are called in order
// for UnmapFullPath, with the order reversed for MapPath and MapPrefix.
//
// That is, order these assuming you are starting with a full path and
// working to a path.
func MapReadBucket(readBucket ReadBucket, mappers ...Mapper) ReadBucket {
	if len(mappers) == 0 {
		return readBucket
	}
	return newMapReadBucketCloser(readBucket, nil, MapChain(mappers...))
}

// MapReadBucketCloser maps the ReadBucketCloser.
//
// If the Mappers are empty, the original ReadBucketCloser is returned.
// If there is more than one Mapper, the Mappers are called in order
// for UnmapFullPath, with the order reversed for MapPath and MapPrefix.
//
// That is, order these assuming you are starting with a full path and
// working to a path.
func MapReadBucketCloser(readBucketCloser ReadBucketCloser, mappers ...Mapper) ReadBucketCloser {
	if len(mappers) == 0 {
		return readBucketCloser
	}
	return newMapReadBucketCloser(readBucketCloser, readBucketCloser.Close, MapChain(mappers...))
}

// MapWriteBucket maps the WriteBucket.
//
// If the Mappers are empty, the original WriteBucket is returned.
// If there is more than one Mapper, the Mappers are called in order
// for UnmapFullPath, with the order reversed for MapPath and MapPrefix.
//
// That is, order these assuming you are starting with a full path and
// working to a path.
//
// If a path that does not match is called for Put, an error is returned.
func MapWriteBucket(writeBucket WriteBucket, mappers ...Mapper) WriteBucket {
	if len(mappers) == 0 {
		return writeBucket
	}
	return newMapWriteBucketCloser(writeBucket, nil, MapChain(mappers...))
}

// MapWriteBucketCloser maps the WriteBucketCloser.
//
// If the Mappers are empty, the original WriteBucketCloser is returned.
// If there is more than one Mapper, the Mappers are called in order
// for UnmapFullPath, with the order reversed for MapPath and MapPrefix.
//
// That is, order these assuming you are starting with a full path and
// working to a path.
//
// If a path that does not match is called for Put, an error is returned.
func MapWriteBucketCloser(writeBucketCloser WriteBucketCloser, mappers ...Mapper) WriteBucketCloser {
	if len(mappers) == 0 {
		return writeBucketCloser
	}
	return newMapWriteBucketCloser(writeBucketCloser, writeBucketCloser.Close, MapChain(mappers...))
}

// MapReadWriteBucket maps the ReadWriteBucket.
//
// If the Mappers are empty, the original ReadWriteBucket is returned.
// If there is more than one Mapper, the Mappers are called in order
// for UnmapFullPath, with the order reversed for MapPath and MapPrefix.
//
// That is, order these assuming you are starting with a full path and
// working to a path.
func MapReadWriteBucket(readWriteBucket ReadWriteBucket, mappers ...Mapper) ReadWriteBucket {
	if len(mappers) == 0 {
		return readWriteBucket
	}
	mapper := MapChain(mappers...)
	return compositeReadWriteBucketCloser{
		newMapReadBucketCloser(readWriteBucket, nil, mapper),
		newMapWriteBucketCloser(readWriteBucket, nil, mapper),
		nil,
	}
}

// MapReadWriteBucketCloser maps the ReadWriteBucketCloser.
//
// If the Mappers are empty, the original ReadWriteBucketCloser is returned.
// If there is more than one Mapper, the Mappers are called in order
// for UnmapFullPath, with the order reversed for MapPath and MapPrefix.
//
// That is, order these assuming you are starting with a full path and
// working to a path.
func MapReadWriteBucketCloser(readWriteBucketCloser ReadWriteBucketCloser, mappers ...Mapper) ReadWriteBucketCloser {
	if len(mappers) == 0 {
		return readWriteBucketCloser
	}
	mapper := MapChain(mappers...)
	return compositeReadWriteBucketCloser{
		newMapReadBucketCloser(readWriteBucketCloser, nil, mapper),
		newMapWriteBucketCloser(readWriteBucketCloser, nil, mapper),
		readWriteBucketCloser.Close,
	}
}

type mapReadBucketCloser struct {
	delegate  ReadBucket
	closeFunc func() error
	mapper    Mapper
}

func newMapReadBucketCloser(
	delegate ReadBucket,
	closeFunc func() error,
	mapper Mapper,
) *mapReadBucketCloser {
	return &mapReadBucketCloser{
		delegate:  delegate,
		closeFunc: closeFunc,
		mapper:    mapper,
	}
}

func (r *mapReadBucketCloser) Get(ctx context.Context, path string) (ReadObjectCloser, error) {
	fullPath, err := r.getFullPath("read", path)
	if err != nil {
		return nil, err
	}
	readObjectCloser, err := r.delegate.Get(ctx, fullPath)
	// TODO: if this is a path error, we should replace the path
	if err != nil {
		return nil, err
	}
	return replaceReadObjectCloserPath(readObjectCloser, path), nil
}

func (r *mapReadBucketCloser) Stat(ctx context.Context, path string) (ObjectInfo, error) {
	fullPath, err := r.getFullPath("stat", path)
	if err != nil {
		return nil, err
	}
	objectInfo, err := r.delegate.Stat(ctx, fullPath)
	// TODO: if this is a path error, we should replace the path
	if err != nil {
		return nil, err
	}
	return replaceObjectInfoPath(objectInfo, path), nil
}

func (r *mapReadBucketCloser) Walk(ctx context.Context, prefix string, f func(ObjectInfo) error) error {
	prefix, err := normalpath.NormalizeAndValidate(prefix)
	if err != nil {
		return err
	}
	fullPrefix, matches := r.mapper.MapPrefix(prefix)
	if !matches {
		return nil
	}
	return r.delegate.Walk(
		ctx,
		fullPrefix,
		func(objectInfo ObjectInfo) error {
			path, matches, err := r.mapper.UnmapFullPath(objectInfo.Path())
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

func (r *mapReadBucketCloser) Close() error {
	if r.closeFunc != nil {
		return r.closeFunc()
	}
	return nil
}

func (r *mapReadBucketCloser) getFullPath(op string, path string) (string, error) {
	path, err := normalpath.NormalizeAndValidate(path)
	if err != nil {
		return "", err
	}
	if path == "." {
		return "", errors.New("cannot get root")
	}
	fullPath, matches := r.mapper.MapPath(path)
	if !matches {
		return "", &fs.PathError{Op: op, Path: path, Err: fs.ErrNotExist}
	}
	return fullPath, nil
}

type mapWriteBucketCloser struct {
	delegate  WriteBucket
	closeFunc func() error
	mapper    Mapper
}

func newMapWriteBucketCloser(
	delegate WriteBucket,
	closeFunc func() error,
	mapper Mapper,
) *mapWriteBucketCloser {
	return &mapWriteBucketCloser{
		delegate: delegate,
		mapper:   mapper,
	}
}

func (w *mapWriteBucketCloser) Put(ctx context.Context, path string, opts ...PutOption) (WriteObjectCloser, error) {
	fullPath, err := w.getFullPath(path)
	if err != nil {
		return nil, err
	}
	writeObjectCloser, err := w.delegate.Put(ctx, fullPath, opts...)
	// TODO: if this is a path error, we should replace the path
	if err != nil {
		return nil, err
	}
	return replaceWriteObjectCloserExternalAndLocalPathsNotSupported(writeObjectCloser), nil
}

func (w *mapWriteBucketCloser) Delete(ctx context.Context, path string) error {
	fullPath, err := w.getFullPath(path)
	if err != nil {
		return err
	}
	return w.delegate.Delete(ctx, fullPath)
}

func (w *mapWriteBucketCloser) DeleteAll(ctx context.Context, prefix string) error {
	prefix, err := normalpath.NormalizeAndValidate(prefix)
	if err != nil {
		return err
	}
	fullPrefix, matches := w.mapper.MapPrefix(prefix)
	if !matches {
		return nil
	}
	return w.delegate.DeleteAll(ctx, fullPrefix)
}

func (*mapWriteBucketCloser) SetExternalAndLocalPathsSupported() bool {
	return false
}

func (w *mapWriteBucketCloser) Close() error {
	if w.closeFunc != nil {
		return w.closeFunc()
	}
	return nil
}

func (w *mapWriteBucketCloser) getFullPath(path string) (string, error) {
	path, err := normalpath.NormalizeAndValidate(path)
	if err != nil {
		return "", err
	}
	if path == "." {
		return "", errors.New("cannot get root")
	}
	fullPath, matches := w.mapper.MapPath(path)
	if !matches {
		return "", fmt.Errorf("path does not match: %s", path)
	}
	return fullPath, nil
}

func replaceObjectInfoPath(objectInfo ObjectInfo, path string) ObjectInfo {
	if objectInfo.Path() == path {
		return objectInfo
	}
	return storageutil.NewObjectInfo(
		path,
		objectInfo.ExternalPath(),
		objectInfo.LocalPath(),
	)
}

func replaceReadObjectCloserPath(readObjectCloser ReadObjectCloser, path string) ReadObjectCloser {
	if readObjectCloser.Path() == path {
		return readObjectCloser
	}
	return compositeReadObjectCloser{replaceObjectInfoPath(readObjectCloser, path), readObjectCloser}
}

func replaceWriteObjectCloserExternalAndLocalPathsNotSupported(writeObjectCloser WriteObjectCloser) WriteObjectCloser {
	return writeObjectCloserExternalAndLocalPathsNotSuppoted{writeObjectCloser}
}

type writeObjectCloserExternalAndLocalPathsNotSuppoted struct {
	io.WriteCloser
}

func (writeObjectCloserExternalAndLocalPathsNotSuppoted) SetExternalPath(string) error {
	return ErrSetExternalPathUnsupported
}

func (writeObjectCloserExternalAndLocalPathsNotSuppoted) SetLocalPath(string) error {
	return ErrSetLocalPathUnsupported
}

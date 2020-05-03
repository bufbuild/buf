// Copyright 2020 Buf Technologies Inc.
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

package storageos

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"

	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/internal"
	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
)

type bucket struct {
	rootPath string
	closed   bool
}

func newBucket(rootPath string) (*bucket, error) {
	rootPath = storagepath.Unnormalize(rootPath)
	fileInfo, err := os.Stat(rootPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, storage.NewErrNotExist(rootPath)
		}
		return nil, err
	}
	if !fileInfo.IsDir() {
		return nil, newErrNotDir(rootPath)
	}
	// allow anything with OS buckets including absolute paths
	// and jumping context
	rootPath = storagepath.Normalize(rootPath)
	return &bucket{
		rootPath: rootPath,
	}, nil
}

func (b *bucket) Get(ctx context.Context, path string) (storage.ReadObject, error) {
	path, err := storagepath.NormalizeAndValidate(path)
	if err != nil {
		return nil, err
	}
	if path == "." {
		return nil, errors.New("cannot get root")
	}
	actualPath := storagepath.Unnormalize(storagepath.Join(b.rootPath, path))
	if b.closed {
		return nil, storage.ErrClosed
	}
	// this is potentially introducing two calls to a file
	// instead of one, ie we do both Stat and Open as opposed
	// to just Open
	// we do this to make sure we are only reading regular files
	fileInfo, err := os.Stat(actualPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, storage.NewErrNotExist(path)
		}
		return nil, err
	}
	if !fileInfo.Mode().IsRegular() {
		// making this a user error as any access means this was generally requested
		// by the user, since we only call the function for Walk on regular files
		return nil, fmt.Errorf("%q is not a regular file", path)
	}
	file, err := os.Open(actualPath)
	if err != nil {
		return nil, err
	}
	if fileInfo.Size() > int64(math.MaxUint32) {
		return nil, fmt.Errorf("file too large: %d", fileInfo.Size())
	}
	return newReadObject(file, uint32(fileInfo.Size())), nil
}

func (b *bucket) Stat(ctx context.Context, path string) (storage.ObjectInfo, error) {
	path, err := storagepath.NormalizeAndValidate(path)
	if err != nil {
		return nil, err
	}
	if path == "." {
		return nil, errors.New("cannot check root")
	}
	actualPath := storagepath.Unnormalize(storagepath.Join(b.rootPath, path))
	if b.closed {
		return nil, storage.ErrClosed
	}
	fileInfo, err := os.Stat(actualPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, storage.NewErrNotExist(path)
		}
		return nil, err
	}
	if !fileInfo.Mode().IsRegular() {
		// should this be an error, or just return false?
		// probably an error as we should not be trying to access
		// files that are not regular, however in walk we just
		// filter non-regular files
		// making this a user error as any access means this was generally requested
		// by the user, since we only call the function for Walk on regular files
		return nil, fmt.Errorf("%q is not a regular file", path)
	}
	return internal.NewObjectInfo(uint32(fileInfo.Size())), nil
}

func (b *bucket) Walk(ctx context.Context, prefix string, f func(string) error) error {
	prefix, err := storagepath.NormalizeAndValidate(prefix)
	if err != nil {
		return err
	}
	prefix = storagepath.Unnormalize(storagepath.Join(b.rootPath, prefix))
	if b.closed {
		return storage.ErrClosed
	}
	fileCount := 0
	// Walk does not follow symlinks
	return filepath.Walk(
		prefix,
		func(path string, fileInfo os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			fileCount++
			select {
			case <-ctx.Done():
				err := ctx.Err()
				if err == context.DeadlineExceeded {
					return fmt.Errorf("timed out after walking %d files: %v", fileCount, err)
				}
				return err
			default:
			}
			if fileInfo.Mode().IsRegular() {
				rel, err := storagepath.Rel(b.rootPath, storagepath.Normalize(path))
				if err != nil {
					return err
				}
				// just in case
				rel, err = storagepath.NormalizeAndValidate(rel)
				if err != nil {
					return err
				}
				if err := f(rel); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

func (b *bucket) Put(ctx context.Context, path string, size uint32) (storage.WriteObject, error) {
	path, err := storagepath.NormalizeAndValidate(path)
	if err != nil {
		return nil, err
	}
	if path == "." {
		return nil, errors.New("cannot put root")
	}
	path = storagepath.Unnormalize(storagepath.Join(b.rootPath, path))
	if b.closed {
		return nil, storage.ErrClosed
	}
	dir := storagepath.Unnormalize(storagepath.Dir(storagepath.Normalize(path)))
	fileInfo, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else if !fileInfo.IsDir() {
		return nil, newErrNotDir(dir)
	}
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return newWriteObject(file, size), nil
}

func (b *bucket) Info() storage.BucketInfo {
	return internal.NewBucketInfo(false)
}

func (b *bucket) Close() error {
	if b.closed {
		return storage.ErrClosed
	}
	b.closed = true
	return nil
}

type readObject struct {
	file *os.File
	size uint32
}

func newReadObject(file *os.File, size uint32) *readObject {
	return &readObject{
		file: file,
		size: size,
	}
}

func (r *readObject) Read(p []byte) (int, error) {
	return r.file.Read(p)
}

func (r *readObject) Close() error {
	return r.file.Close()
}

func (r *readObject) Info() storage.ObjectInfo {
	return internal.NewObjectInfo(r.size)
}

type writeObject struct {
	file    *os.File
	size    uint32
	written int
}

func newWriteObject(file *os.File, size uint32) *writeObject {
	return &writeObject{
		file: file,
		size: size,
	}
}

func (w *writeObject) Write(p []byte) (int, error) {
	if uint32(w.written+len(p)) > w.size {
		return 0, io.EOF
	}
	n, err := w.file.Write(p)
	w.written += n
	return n, err
}

func (w *writeObject) Close() error {
	err := w.file.Close()
	if uint32(w.written) != w.size {
		return storage.ErrIncompleteWrite
	}
	return err
}

func (w *writeObject) Info() storage.ObjectInfo {
	return internal.NewObjectInfo(w.size)
}

// newErrNotDir returns a new Error for a path not being a directory.
func newErrNotDir(path string) *storagepath.Error {
	return storagepath.NewError(path, errNotDir)
}

// Copyright 2020-2025 Buf Technologies, Inc.
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
	"io/fs"
	"os"
	"path/filepath"
	"sync/atomic"

	"buf.build/go/standard/xpath/xfilepath"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageutil"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

// errNotDir is the error returned if a path is not a directory.
var errNotDir = errors.New("not a directory")

type bucket struct {
	rootPath         string
	absoluteRootPath string
	symlinks         bool
}

func newBucket(rootPath string, symlinks bool) (*bucket, error) {
	rootPath = normalpath.Unnormalize(rootPath)
	if err := validateDirPathExists(rootPath, symlinks); err != nil {
		return nil, err
	}
	absoluteRootPath, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, err
	}
	// do not validate - allow anything with OS buckets including
	// absolute paths and jumping context
	rootPath = normalpath.Normalize(rootPath)
	return &bucket{
		rootPath:         rootPath,
		absoluteRootPath: absoluteRootPath,
		symlinks:         symlinks,
	}, nil
}

func (b *bucket) Get(ctx context.Context, path string) (storage.ReadObjectCloser, error) {
	externalPath, err := b.getExternalPath(path)
	if err != nil {
		return nil, err
	}
	if err := b.validateExternalPath(path, externalPath); err != nil {
		return nil, err
	}
	resolvedPath := externalPath
	if b.symlinks {
		resolvedPath, err = filepath.EvalSymlinks(externalPath)
		if err != nil {
			return nil, err
		}
	}
	file, err := os.Open(resolvedPath)
	if err != nil {
		return nil, err
	}
	// we could use fileInfo.Name() however we might as well use the externalPath
	return newReadObjectCloser(
		path,
		externalPath,
		file,
	), nil
}

func (b *bucket) Stat(ctx context.Context, path string) (storage.ObjectInfo, error) {
	externalPath, err := b.getExternalPath(path)
	if err != nil {
		return nil, err
	}
	if err := b.validateExternalPath(path, externalPath); err != nil {
		return nil, err
	}
	// we could use fileInfo.Name() however we might as well use the externalPath
	return storageutil.NewObjectInfo(
		path,
		externalPath,
		// In storageos, the external path is also the local path.
		externalPath,
	), nil
}

func (b *bucket) Walk(
	ctx context.Context,
	prefix string,
	f func(storage.ObjectInfo) error,
) error {
	externalPrefix, err := b.getExternalPrefix(prefix)
	if err != nil {
		return err
	}
	walkChecker := storageutil.NewWalkChecker()
	var walkOptions []xfilepath.WalkOption
	if b.symlinks {
		walkOptions = append(walkOptions, xfilepath.WalkWithSymlinks())
	}
	if err := xfilepath.Walk(
		externalPrefix,
		func(externalPath string, fileInfo os.FileInfo, err error) error {
			if err != nil {
				// this can happen if a symlink is broken
				// in this case, we just want to continue the walk
				if b.symlinks && os.IsNotExist(err) {
					return nil
				}
				return err
			}
			if err := walkChecker.Check(ctx); err != nil {
				return err
			}
			absoluteExternalPath, err := filepath.Abs(externalPath)
			if err != nil {
				return err
			}
			if fileInfo.Mode().IsRegular() {
				path, err := normalpath.Rel(b.absoluteRootPath, absoluteExternalPath)
				if err != nil {
					return err
				}
				// just in case
				path, err = normalpath.NormalizeAndValidate(path)
				if err != nil {
					return err
				}
				if err := f(
					storageutil.NewObjectInfo(
						path,
						externalPath,
						// In storageos, the external path is also the local path.
						externalPath,
					),
				); err != nil {
					return err
				}
			}
			return nil
		},
		walkOptions...,
	); err != nil {
		if os.IsNotExist(err) {
			// Should be a no-op according to the spec.
			return nil
		}
		return err
	}
	return nil
}

func (b *bucket) Put(ctx context.Context, path string, options ...storage.PutOption) (storage.WriteObjectCloser, error) {
	externalPath, err := b.getExternalPath(path)
	if err != nil {
		return nil, err
	}
	externalDir := filepath.Dir(externalPath)
	var fileInfo os.FileInfo
	if b.symlinks {
		fileInfo, err = os.Stat(externalDir)
	} else {
		fileInfo, err = os.Lstat(externalDir)
	}
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(externalDir, 0755); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else if !fileInfo.IsDir() {
		return nil, newErrNotDir(externalDir)
	}
	var file *os.File
	var finalPath string
	if storage.NewPutOptions(options).Atomic() {
		file, err = os.CreateTemp(externalDir, ".tmp"+filepath.Base(externalPath)+"*")
		finalPath = externalPath
	} else {
		file, err = os.Create(externalPath)
	}
	if err != nil {
		return nil, err
	}
	return newWriteObjectCloser(
		file,
		finalPath,
	), nil
}

func (b *bucket) Delete(ctx context.Context, path string) error {
	externalPath, err := b.getExternalPath(path)
	if err != nil {
		return err
	}
	// Note: this deletes the file at the path, but it may
	// leave orphan parent directories around that were
	// created by the MkdirAll in Put.
	if err := os.Remove(externalPath); err != nil {
		if os.IsNotExist(err) {
			return &fs.PathError{Op: "stat", Path: path, Err: fs.ErrNotExist}
		}
		return err
	}
	return nil
}

func (b *bucket) DeleteAll(ctx context.Context, prefix string) error {
	externalPrefix, err := b.getExternalPrefix(prefix)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(externalPrefix); err != nil {
		// this is a no-nop per the documentation
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return nil
}

func (*bucket) SetExternalAndLocalPathsSupported() bool {
	return false
}

func (b *bucket) getExternalPath(path string) (string, error) {
	path, err := storageutil.ValidatePath(path)
	if err != nil {
		return "", err
	}
	realClean, err := xfilepath.RealClean(normalpath.Join(b.rootPath, path))
	if err != nil {
		return "", err
	}
	return normalpath.Unnormalize(realClean), nil
}

func (b *bucket) validateExternalPath(path string, externalPath string) error {
	// this is potentially introducing two calls to a file
	// instead of one, ie we do both Stat and Open as opposed to just Open
	// we do this to make sure we are only reading regular files
	var fileInfo os.FileInfo
	var err error
	if b.symlinks {
		fileInfo, err = os.Stat(externalPath)
	} else {
		fileInfo, err = os.Lstat(externalPath)
	}
	if err != nil {
		if os.IsNotExist(err) {
			return &fs.PathError{Op: "stat", Path: path, Err: fs.ErrNotExist}
		}
		// The path might have a regular file in one of its
		// elements (e.g. 'foo/bar/baz.proto' where 'bar' is a
		// regular file).
		//
		// In this case, the standard library will return an
		// os.PathError, but there isn't an exported error value
		// to check against (i.e. os.Is*). But we can still discover
		// whether or not this is the case by checking if any of the
		// path components represents a regular file (e.g. 'foo/bar').
		//
		// It's important that we detect these cases so that
		// multi buckets don't unnecessarily fail when one of
		// its delegates actually defines the path.
		lastParentPath := externalPath
		parentPath := filepath.Dir(externalPath)
		for ; parentPath != lastParentPath; lastParentPath, parentPath = parentPath, filepath.Dir(parentPath) {
			parentFileInfo, err := os.Stat(parentPath)
			if err != nil {
				continue
			}
			if parentFileInfo.Mode().IsRegular() {
				// This error primarily serves as a sentinel error,
				// but we preserve the original path argument so that
				// the error still makes sense to the user.
				return &fs.PathError{Op: "stat", Path: path, Err: fs.ErrNotExist}
			}
		}
		return err
	}
	if !fileInfo.Mode().IsRegular() {
		// making this a user error as any access means this was generally requested
		// by the user, since we only call the function for Walk on regular files
		return &fs.PathError{Op: "stat", Path: path, Err: fs.ErrNotExist}
	}
	return nil
}

func (b *bucket) getExternalPrefix(prefix string) (string, error) {
	prefix, err := storageutil.ValidatePrefix(prefix)
	if err != nil {
		return "", err
	}
	realClean, err := xfilepath.RealClean(normalpath.Join(b.rootPath, prefix))
	if err != nil {
		return "", err
	}
	return normalpath.Unnormalize(realClean), nil
}

type readObjectCloser struct {
	// we use ObjectInfo for Path, ExternalPath, etc to make sure this is static
	// we put ObjectInfos in maps in other places so we do not want this to change
	// this could be a problem if the underlying file is concurrently moved or resized however
	storageutil.ObjectInfo

	file *os.File
}

func newReadObjectCloser(
	path string,
	externalPath string,
	file *os.File,
) *readObjectCloser {
	return &readObjectCloser{
		ObjectInfo: storageutil.NewObjectInfo(
			path,
			externalPath,
			// In storageos, the external path is the local path
			externalPath,
		),
		file: file,
	}
}

func (r *readObjectCloser) Read(p []byte) (int, error) {
	n, err := r.file.Read(p)
	return n, toStorageError(err)
}

func (r *readObjectCloser) Close() error {
	return toStorageError(r.file.Close())
}

type writeObjectCloser struct {
	file *os.File
	// path is set during atomic writes to the final path where the file should be created.
	// If set, the file is a temp file that needs to be renamed to this path if Write/Close are successful.
	path string
	// writeErr contains the first non-nil error caught by a call to Write.
	// This is returned in Close for atomic writes to prevent writing an incomplete file.
	writeErr onceError
}

func newWriteObjectCloser(
	file *os.File,
	path string,
) *writeObjectCloser {
	return &writeObjectCloser{
		file: file,
		path: path,
	}
}

func (w *writeObjectCloser) Write(p []byte) (int, error) {
	n, err := w.file.Write(p)
	if err != nil {
		w.writeErr.Store(err)
	}
	return n, toStorageError(err)
}

func (w *writeObjectCloser) SetExternalPath(string) error {
	return storage.ErrSetExternalPathUnsupported
}

func (w *writeObjectCloser) SetLocalPath(string) error {
	return storage.ErrSetLocalPathUnsupported
}

func (w *writeObjectCloser) Close() error {
	err := toStorageError(w.file.Close())
	// This is an atomic write operation - we need to rename to the final path
	if w.path != "" {
		atomicWriteErr := errors.Join(w.writeErr.Load(), err)
		// Failed during Write or Close - remove temporary file without rename
		if atomicWriteErr != nil {
			return toStorageError(errors.Join(atomicWriteErr, os.Remove(w.file.Name())))
		}
		if err := os.Rename(w.file.Name(), w.path); err != nil {
			return toStorageError(errors.Join(err, os.Remove(w.file.Name())))
		}
	}
	return err
}

// onceError is an object that will only store an error once.
type onceError struct {
	err atomic.Value
}

// Store stores the err.
func (e *onceError) Store(err error) {
	e.err.CompareAndSwap(nil, err)
}

// Load loads the stored error.
func (e *onceError) Load() error {
	atomicValue := e.err.Load()
	if atomicValue == nil {
		return nil
	}
	err, ok := atomicValue.(error)
	if !ok {
		return syserror.Newf("expected error but got %T", atomicValue)
	}
	return err
}

// newErrNotDir returns a new Error for a path not being a directory.
func newErrNotDir(path string) *normalpath.Error {
	return normalpath.NewError(path, errNotDir)
}

func toStorageError(err error) error {
	if errors.Is(err, os.ErrClosed) {
		return storage.ErrClosed
	}
	return err
}

// validateDirPathExists returns a non-nil error if the given dirPath
// is not a valid directory path.
func validateDirPathExists(dirPath string, symlinks bool) error {
	var fileInfo os.FileInfo
	var err error
	if symlinks {
		fileInfo, err = os.Stat(dirPath)
	} else {
		fileInfo, err = os.Lstat(dirPath)
	}
	if err != nil {
		return err
	}
	if !fileInfo.IsDir() {
		return newErrNotDir(dirPath)
	}
	return nil
}

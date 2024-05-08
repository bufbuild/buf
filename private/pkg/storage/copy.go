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
	"io"
	"sync/atomic"

	"github.com/bufbuild/buf/private/pkg/thread"
	"go.uber.org/multierr"
)

// Copy copies the bucket at from to the bucket at to.
//
// Copies done concurrently.
// Returns the number of files copied.
func Copy(
	ctx context.Context,
	from ReadBucket,
	to WriteBucket,
	options ...CopyOption,
) (int, error) {
	copyOptions := newCopyOptions()
	for _, option := range options {
		option(copyOptions)
	}
	return copyPaths(
		ctx,
		from,
		to,
		copyOptions.externalAndLocalPaths,
		copyOptions.atomic,
	)
}

// CopyReadObject copies the contents of the ReadObject into the WriteBucket at the path.
func CopyReadObject(
	ctx context.Context,
	writeBucket WriteBucket,
	readObject ReadObject,
	options ...CopyOption,
) (retErr error) {
	copyOptions := newCopyOptions()
	for _, option := range options {
		option(copyOptions)
	}
	return copyReadObject(
		ctx,
		readObject,
		writeBucket,
		readObject.Path(),
		copyOptions.externalAndLocalPaths,
		copyOptions.atomic,
	)
}

// CopyReader copies the contents of the Reader into the WriteBucket at the path.
func CopyReader(
	ctx context.Context,
	writeBucket WriteBucket,
	reader io.Reader,
	path string,
) (retErr error) {
	writeObjectCloser, err := writeBucket.Put(ctx, path)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, writeObjectCloser.Close())
	}()
	_, err = io.Copy(writeObjectCloser, reader)
	return err
}

// CopyPath copies the fromPath from the ReadBucket to the toPath on the WriteBucket.
func CopyPath(
	ctx context.Context,
	from ReadBucket,
	fromPath string,
	to WriteBucket,
	toPath string,
	options ...CopyOption,
) error {
	copyOptions := newCopyOptions()
	for _, option := range options {
		option(copyOptions)
	}
	return copyPath(
		ctx,
		from,
		fromPath,
		to,
		toPath,
		copyOptions.externalAndLocalPaths,
		copyOptions.atomic,
	)
}

// CopyOption is an option for Copy.
type CopyOption func(*copyOptions)

// CopyWithExternalAndLocalPaths returns a new CopyOption that says to copy external and local paths.
//
// The to WriteBucket must support setting external and local paths.
func CopyWithExternalAndLocalPaths() CopyOption {
	return func(copyOptions *copyOptions) {
		copyOptions.externalAndLocalPaths = true
	}
}

// CopyWithAtomic returns a new CopyOption that says to set PutWithAtomic when copying each file.
//
// See the documentation on PutWithAtomic for more details.
func CopyWithAtomic() CopyOption {
	return func(copyOptions *copyOptions) {
		copyOptions.atomic = true
	}
}

func copyPaths(
	ctx context.Context,
	from ReadBucket,
	to WriteBucket,
	copyExternalAndLocalPaths bool,
	atomicOpt bool,
) (int, error) {
	paths, err := AllPaths(ctx, from, "")
	if err != nil {
		return 0, err
	}
	var count atomic.Int64
	jobs := make([]func(context.Context) error, len(paths))
	for i, path := range paths {
		path := path
		jobs[i] = func(ctx context.Context) error {
			if err := copyPath(ctx, from, path, to, path, copyExternalAndLocalPaths, atomicOpt); err != nil {
				return err
			}
			count.Add(1)
			return nil
		}
	}
	err = thread.Parallelize(ctx, jobs)
	return int(count.Load()), err
}

// copyPath copies the path from the bucket at from to the bucket at to using the given paths.
//
// Paths will be normalized within this function.
func copyPath(
	ctx context.Context,
	from ReadBucket,
	fromPath string,
	to WriteBucket,
	toPath string,
	copyExternalAndLocalPaths bool,
	atomic bool,
) (retErr error) {
	readObjectCloser, err := from.Get(ctx, fromPath)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(err, readObjectCloser.Close())
	}()
	return copyReadObject(ctx, readObjectCloser, to, toPath, copyExternalAndLocalPaths, atomic)
}

func copyReadObject(
	ctx context.Context,
	readObject ReadObject,
	to WriteBucket,
	toPath string,
	copyExternalAndLocalPaths bool,
	atomic bool,
) (retErr error) {
	var putOptions []PutOption
	if atomic {
		putOptions = append(putOptions, PutWithAtomic())
	}
	writeObjectCloser, err := to.Put(ctx, toPath, putOptions...)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, writeObjectCloser.Close())
	}()
	if copyExternalAndLocalPaths {
		if err := writeObjectCloser.SetExternalPath(readObject.ExternalPath()); err != nil {
			return err
		}
		if err := writeObjectCloser.SetLocalPath(readObject.LocalPath()); err != nil {
			return err
		}
	}
	_, err = io.Copy(writeObjectCloser, readObject)
	return err
}

type copyOptions struct {
	externalAndLocalPaths bool
	atomic                bool
}

func newCopyOptions() *copyOptions {
	return &copyOptions{}
}

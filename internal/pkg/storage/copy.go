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
	"io"
	"sync"

	"github.com/bufbuild/buf/internal/pkg/thread"
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
	copyOptions := &copyOptions{}
	for _, option := range options {
		option(copyOptions)
	}
	return copyPaths(
		ctx,
		from,
		to,
		walkBucketFunc(from, ""),
		copyOptions.externalPaths,
		false,
	)
}

// CopyOption is an option for Copy.
type CopyOption func(*copyOptions)

// CopyWithExternalPaths returns a new CopyOption that says to copy external paths.
//
// The to WriteBucket must support setting external paths.
func CopyWithExternalPaths() CopyOption {
	return func(copyOptions *copyOptions) {
		copyOptions.externalPaths = true
	}
}

func copyPaths(
	ctx context.Context,
	from ReadBucket,
	to WriteBucket,
	walk func(context.Context, func(string) error) error,
	copyExternalPaths bool,
	allowNotExist bool,
) (int, error) {
	semaphoreC := make(chan struct{}, thread.Parallelism())
	var retErr error
	var count int
	var wg sync.WaitGroup
	var lock sync.Mutex
	if walkErr := walk(
		ctx,
		func(path string) error {
			newPath := path
			wg.Add(1)
			semaphoreC <- struct{}{}
			go func() {
				err := copyPath(ctx, from, to, path, newPath, copyExternalPaths)
				lock.Lock()
				if err != nil {
					if !allowNotExist || !IsNotExist(err) {
						retErr = multierr.Append(retErr, err)
					}
				} else {
					count++
				}
				lock.Unlock()
				<-semaphoreC
				wg.Done()
			}()
			return nil
		},
	); walkErr != nil {
		return count, walkErr
	}
	wg.Wait()
	return count, retErr
}

// copyPath copies the path from the bucket at from to the bucket at to using the given paths.
//
// Paths will be normalized within this function.
func copyPath(
	ctx context.Context,
	from ReadBucket,
	to WriteBucket,
	fromPath string,
	toPath string,
	copyExternalPaths bool,
) error {
	readObjectCloser, err := from.Get(ctx, fromPath)
	if err != nil {
		return err
	}
	writeObjectCloser, err := to.Put(ctx, toPath, readObjectCloser.Size())
	if err != nil {
		return multierr.Append(err, readObjectCloser.Close())
	}
	if copyExternalPaths {
		// do this before copying so that writeObjectCloser.Close() will fail due to incomplete write
		if err := writeObjectCloser.SetExternalPath(readObjectCloser.ExternalPath()); err != nil {
			return multierr.Append(err, multierr.Append(writeObjectCloser.Close(), readObjectCloser.Close()))
		}
	}
	_, err = io.Copy(writeObjectCloser, readObjectCloser)
	return multierr.Append(err, multierr.Append(writeObjectCloser.Close(), readObjectCloser.Close()))
}

func walkBucketFunc(bucket ReadBucket, prefix string) func(context.Context, func(string) error) error {
	return func(ctx context.Context, f func(string) error) error {
		return bucket.Walk(ctx, prefix, func(objectInfo ObjectInfo) error {
			return f(objectInfo.Path())
		})
	}
}

type copyOptions struct {
	externalPaths bool
}

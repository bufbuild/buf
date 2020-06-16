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
	"fmt"
	"io"
	"sync"

	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/thread"
	"go.uber.org/multierr"
)

// transformer can be nil
func copyPaths(
	ctx context.Context,
	from ReadBucket,
	to ReadWriteBucket,
	walk func(context.Context, func(string) error) error,
	options []normalpath.TransformerOption,
	allowNotExist bool,
) (int, error) {
	transformer := normalpath.NewTransformer(options...)
	semaphoreC := make(chan struct{}, thread.Parallelism())
	var retErr error
	var count int
	var wg sync.WaitGroup
	var lock sync.Mutex
	if walkErr := walk(
		ctx,
		func(path string) error {
			newPath := path
			if transformer != nil {
				var ok bool
				newPath, ok = transformer.Transform(path)
				if !ok {
					return nil
				}
			}
			wg.Add(1)
			semaphoreC <- struct{}{}
			go func() {
				err := copyPath(ctx, from, to, path, newPath)
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

// returns ErrNotExist if fromPath does not exist
func copyPath(
	ctx context.Context,
	from ReadBucket,
	to ReadWriteBucket,
	fromPath string,
	toPath string,
) error {
	readObject, err := from.Get(ctx, fromPath)
	if err != nil {
		return err
	}
	writeObject, err := to.Put(ctx, toPath, readObject.Size())
	if err != nil {
		return multierr.Append(err, readObject.Close())
	}
	_, err = io.Copy(writeObject, readObject)
	return multierr.Append(err, multierr.Append(writeObject.Close(), readObject.Close()))
}

func walkBucketFunc(bucket ReadBucket, prefix string) func(context.Context, func(string) error) error {
	return func(ctx context.Context, f func(string) error) error {
		return bucket.Walk(ctx, prefix, f)
	}
}

func walkPathsFunc(paths []string) func(context.Context, func(string) error) error {
	return func(ctx context.Context, f func(string) error) error {
		fileCount := 0
		for _, path := range paths {
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
			if err := f(path); err != nil {
				return err
			}
		}
		return nil
	}
}

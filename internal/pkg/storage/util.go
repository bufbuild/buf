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
	"io/ioutil"

	"go.uber.org/multierr"
)

// ReadPath is analogous to ioutil.ReadFile.
//
// Returns an error that fufills IsNotExist if the path does not exist.
func ReadPath(ctx context.Context, readBucket ReadBucket, path string) (_ []byte, retErr error) {
	readObject, err := readBucket.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := readObject.Close(); err != nil && retErr == nil {
			retErr = err
		}
	}()
	return ioutil.ReadAll(readObject)
}

// WalkReadObjects walks the bucket and calls get on each, closing the resulting ReadObjectCloser
// when done.
func WalkReadObjects(
	ctx context.Context,
	readBucket ReadBucket,
	prefix string,
	f func(ReadObject) error,
) error {
	return readBucket.Walk(
		ctx,
		prefix,
		func(objectInfo ObjectInfo) error {
			readObjectCloser, err := readBucket.Get(ctx, objectInfo.Path())
			if err != nil {
				return err
			}
			return multierr.Append(f(readObjectCloser), readObjectCloser.Close())
		},
	)
}

// Exists returns true if the path exists, false otherwise.
//
// Returns error on system error.
func Exists(ctx context.Context, readBucket ReadBucket, path string) (bool, error) {
	_, err := readBucket.Stat(ctx, path)
	if err != nil {
		if IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

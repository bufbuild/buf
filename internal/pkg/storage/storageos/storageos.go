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

// Package storageos implements an os-backed storage Bucket.
package storageos

import (
	"errors"

	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
)

// errNotDir is the error returned if a path does not dir.
var errNotDir = errors.New("not a directory")

// IsNotDir returns true for a Error that is for a root path not being a directory.
//
// This is only returned when creating a Bucket, or when putting a file into a directory
// path - paths within buckets are all regular files.
func IsNotDir(err error) bool {
	return storagepath.ErrorEquals(err, errNotDir)
}

// NewReadWriteBucketCloser returns a new OS bucket.
//
// Only regular files are handled, that is Exists should only be called
// for regular files, Get and Put only work for regular files, Put
// automatically calls Mkdir, and Walk only calls f on regular files.
//
// Not thread-safe.
func NewReadWriteBucketCloser(rootPath string) (storage.ReadWriteBucketCloser, error) {
	return newBucket(rootPath)
}

// NewReadBucketCloser returns a new read-only OS bucket.
//
// It is better to use this if you want to make sure your callers are not writing
// to the filesystem.
//
// Only regular files are handled, that is Exists should only be called
// for regular files, Get and Put only work for regular files, Put
// automatically calls Mkdir, and Walk only calls f on regular files.
//
// Not thread-safe.
func NewReadBucketCloser(rootPath string) (storage.ReadBucketCloser, error) {
	return newBucket(rootPath)
}

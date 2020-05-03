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

// Package storagemem implements an in-memory storage Bucket.
package storagemem

import (
	"github.com/bufbuild/buf/internal/pkg/storage"
)

// NewReadWriteBucketCloser returns a new in-memory bucket.
func NewReadWriteBucketCloser() storage.ReadWriteBucketCloser {
	return newBucket()
}

// NewImmutableReadBucket returns a new immutable read-only in-memory bucket.
//
// The data in the map will be directly used, and not copied. It should not be
// modified after passing the map to this function.
func NewImmutableReadBucket(pathToData map[string][]byte) (storage.ReadBucket, error) {
	return newImmutableBucket(pathToData)
}

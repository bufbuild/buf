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

// Package storagemem implements an in-memory storage Bucket.
package storagemem

import (
	"context"

	"github.com/bufbuild/buf/internal/pkg/storage"
)

// ReadBucketBuilder builds ReadBuckets.
type ReadBucketBuilder interface {
	storage.WriteBucket
	// ToReadBucket returns a ReadBucket for the current data in the WriteBucket.
	//
	// No further calls can be made to the ReadBucketBuilder after this call.
	// This is functionally equivalent to a Close in other contexts.
	ToReadBucket(options ...ReadBucketOption) (storage.ReadBucket, error)
}

// NewReadBucketBuilder returns a new in-memory ReadBucketBuilder.
func NewReadBucketBuilder() ReadBucketBuilder {
	return newReadBucketBuilder()
}

// NewReadBucket returns a new ReadBucket.
func NewReadBucket(pathToData map[string][]byte, options ...ReadBucketOption) (storage.ReadBucket, error) {
	return newReadBucket(pathToData, options...)
}

// CopyReadBucket copies the contents of the read bucket into a new, memory-backed read bucket.
func CopyReadBucket(ctx context.Context, readBucket storage.ReadBucket) (storage.ReadBucket, error) {
	readBucketBuilder := NewReadBucketBuilder()
	if _, err := storage.Copy(ctx, readBucket, readBucketBuilder); err != nil {
		return nil, err
	}
	return readBucketBuilder.ToReadBucket()
}

// ReadBucketOption is an option for a new ReadBucket.
type ReadBucketOption func(*readBucketOptions)

// WithExternalPathResolver uses the given resolver to resolve paths to external paths.
//
// The default is to use the path as the external path.
// This ExternalPathResolver takes precedence over any explicitly set external paths.
func WithExternalPathResolver(externalPathResolver func(string) (string, error)) ReadBucketOption {
	return func(readBucketOptions *readBucketOptions) {
		readBucketOptions.externalPathResolver = externalPathResolver
	}
}

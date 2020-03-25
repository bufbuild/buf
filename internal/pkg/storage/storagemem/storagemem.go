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

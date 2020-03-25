// Package storagemem implements an in-memory storage Bucket.
package storagemem

import (
	"github.com/bufbuild/buf/internal/pkg/storage"
)

// NewReadWriteBucketCloser returns a new in-memory bucket.
func NewReadWriteBucketCloser() storage.ReadWriteBucketCloser {
	return newBucket()
}

// Package storagemem implements an in-memory storage Bucket.
package storagemem

import (
	"github.com/bufbuild/buf/internal/pkg/bytepool"
	"github.com/bufbuild/buf/internal/pkg/storage"
)

// BucketType is the bucket type.
const BucketType = "mem"

// NewBucket returns a new in-memory bucket.
func NewBucket(segList *bytepool.SegList) storage.Bucket {
	return newBucket(segList)
}

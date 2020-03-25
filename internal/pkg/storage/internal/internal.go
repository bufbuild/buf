package internal

import "github.com/bufbuild/buf/internal/pkg/storage"

// NewObjectInfo returns a new ObjectInfo.
func NewObjectInfo(size uint32) storage.ObjectInfo {
	return objectInfo{size: size}
}

type objectInfo struct {
	size uint32
}

func (o objectInfo) Size() uint32 {
	return o.size
}

// NewBucketInfo returns a new BucketInfo.
func NewBucketInfo(inMemory bool) storage.BucketInfo {
	return bucketInfo{inMemory: inMemory}
}

type bucketInfo struct {
	inMemory bool
}

func (b bucketInfo) InMemory() bool {
	return b.inMemory
}

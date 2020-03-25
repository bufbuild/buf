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

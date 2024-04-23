// Copyright 2020-2024 Buf Technologies, Inc.
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
	"io"
)

// ReadBucket is a simple read-only bucket.
//
// All paths are regular files - Buckets do not handle directories.
// All paths must be relative.
// All paths are cleaned and ToSlash'ed by each function.
// Paths must not jump the bucket context, that is after clean, they
// cannot contain "..".
type ReadBucket interface {
	// Get gets the path.
	//
	// The behavior of concurrently Getting and Putting an object is undefined.
	// The returned ReadObjectCloser is not thread-safe.
	//
	// Returns ErrNotExist if the path does not exist, other error
	// if there is a system error.
	Get(ctx context.Context, path string) (ReadObjectCloser, error)
	// Stat gets info in the object.
	//
	// Returns ErrNotExist if the path does not exist, other error
	// if there is a system error.
	Stat(ctx context.Context, path string) (ObjectInfo, error)
	// Walk walks the bucket with the prefix, calling f on each path.
	// If the prefix doesn't exist, this is a no-op.
	//
	// Note that foo/barbaz will not be called for foo/bar, but will
	// be called for foo/bar/baz.
	//
	// Note that a prefix can also be equal to a path in the bucket, in which
	// case Walk will walk this single file. That is, if file a/b/c.txt is in
	// the bucket, Walk(ctx, "a/b/c.txt", ...) will result in a single iteration,
	// calling f on the file "a/b/c.txt".
	//
	// All paths given to f are normalized and validated.
	// If f returns error, Walk will stop short and return this error.
	// Returns other error on system error.
	Walk(ctx context.Context, prefix string, f func(ObjectInfo) error) error
}

// WriteBucket is a write-only bucket.
type WriteBucket interface {
	// Put returns a WriteObjectCloser to write to the path.
	//
	// The path is truncated on close.
	// The behavior of concurrently Getting and Putting an object is undefined.
	// The returned WriteObjectCloser is not thread-safe.
	//
	// Note that an object may appear via Get and Stat calls before the WriteObjectCloser
	// has closed. To guarantee that an object will only appear once the WriteObjectCloser
	// is closed, pass PutWithAtomic.
	//
	// Returns error on system error.
	Put(ctx context.Context, path string, options ...PutOption) (WriteObjectCloser, error)
	// Delete deletes the object at the path.
	//
	// Returns ErrNotExist if the path does not exist, other error
	// if there is a system error.
	Delete(ctx context.Context, path string) error
	// DeleteAll deletes all objects with the prefix.
	// If the prefix doesn't exist, this is a no-op.
	//
	// Note that the prefix is used as a filepath prefix, and
	// NOT a string prefix. For example, the prefix "foo/bar"
	// will delete "foo/bar/baz", but NOT "foo/barbaz".
	DeleteAll(ctx context.Context, prefix string) error
	// SetExternalAndLocalPathsSupported returns true if SetExternalPath and SetLocalPath are supported.
	//
	// For example, in-memory buckets may choose to return true so that object sources
	// are preserved, but filesystem buckets may choose to return false as they have
	// their own external paths.
	SetExternalAndLocalPathsSupported() bool
}

// ReadWriteBucket is a simple read/write bucket.
type ReadWriteBucket interface {
	ReadBucket
	WriteBucket
}

// ReadBucketCloser is a read-only bucket that must be closed.
type ReadBucketCloser interface {
	io.Closer
	ReadBucket
}

// NopReadBucketCloser returns a ReadBucketCloser for the ReadBucket.
func NopReadBucketCloser(readBucket ReadBucket) ReadBucketCloser {
	return nopReadBucketCloser{readBucket}
}

// WriteBucketCloser is a write-only bucket that must be closed.
type WriteBucketCloser interface {
	io.Closer
	WriteBucket
}

// NopWriteBucketCloser returns a WriteBucketCloser for the WriteBucket.
func NopWriteBucketCloser(writeBucket WriteBucket) WriteBucketCloser {
	return nopWriteBucketCloser{writeBucket}
}

// ReadWriteBucketCloser is a read/write bucket that must be closed.
type ReadWriteBucketCloser interface {
	io.Closer
	ReadWriteBucket
}

// NopReadWriteBucketCloser returns a ReadWriteBucketCloser for the ReadWriteBucket.
func NopReadWriteBucketCloser(readWriteBucket ReadWriteBucket) ReadWriteBucketCloser {
	return nopReadWriteBucketCloser{readWriteBucket}
}

// ObjectInfo contains object info.
//
// An ObjectInfo will always be the same for a given path within a given Bucket,
// that is an ObjectInfo is cacheable for a given Bucket.
type ObjectInfo interface {
	// Path is the path of the object.
	//
	// This will always correspond to a path within the Bucket. For sub-buckets, this is the sub-path, but the
	// external path will include the sub-bucket path.
	//
	// This path will always be normalized, validated, and non-empty.
	Path() string
	// ExternalPath is the path that identifies the object externally.
	//
	// This path is not necessarily a file path, and should only be used to
	// uniquely identify this file as compared to other assets, to for display
	// to users.
	//
	// The path will be unnormalized, if it is a file path.
	// The path will never be empty. If a given implementation has no external path, this falls back to path.
	//
	// Example:
	//   Directory: /foo/bar
	//   Path: baz/bat.proto
	//   ExternalPath: /foo/bar/baz/bat.proto
	//
	// Example:
	//   Directory: .
	//   Path: baz/bat.proto
	//   ExternalPath: baz/bat.proto
	//
	// Example:
	//   S3 Bucket: https://s3.amazonaws.com/foo
	//   Path: baz/bat.proto
	//   ExternalPath: s3://foo/baz/bat.proto
	ExternalPath() string

	// LocalPath is the path on disk of the object, if the object originated from a local disk.
	//
	// This will be unnormalized if present.
	//
	// Will not be present if the path did not originate from disk. For example, objects that originated
	// from archives, git repositories, or object stores will not have this present.
	LocalPath() string
}

// ReadObject is an object read from a bucket.
type ReadObject interface {
	ObjectInfo
	io.Reader
}

// ReadObjectCloser is a ReadObject with a closer.
//
// It must be closed when done.
type ReadObjectCloser interface {
	ReadObject
	io.Closer
}

// WriteObject object written to a bucket.
type WriteObject interface {
	io.Writer

	// SetExternalPath attempts to explicitly set the external path for the new object.
	//
	// If SetExternalAndLocalPathsSupported returns false, this returns error.
	SetExternalPath(externalPath string) error
	// SetLocalPath attempts to explicitly set the local path for the new object.
	//
	// If SetExternalAndLocalPathsSupported  returns false, this returns  error.
	SetLocalPath(localPath string) error
}

// WriteObjectCloser is a WriteObject with a closer.
//
// It must be closed when done.
type WriteObjectCloser interface {
	WriteObject
	io.Closer
}

// PutOptions are the possible options that can be passed to a Put operation.
type PutOptions interface {
	// Atomic ensures that the Put fully writes the file before making it
	// available to readers. This happens by default for some implementations,
	// while others may need to perform a sequence of operations to ensure
	// atomic writes.
	//
	// The Put operation is complete and the path will be readable once the
	// returned WriteObjectCloser is written and closed (without an error).
	// Any errors will cause the Put to be skipped (no path will be created).
	Atomic() bool
	// SuggestedDisableChunking says to suggest disable chunking entirely.
	//
	// If SuggestedChunkSize() > 0, this will always be false.
	//
	// This is a suggestion, implementations may choose to ignore this option.
	SuggestedDisableChunking() bool
	// SuggestedChunkSize sets the given size in bytes as a suggested chunk
	// size to use by the Bucket implementation for this Put call.
	// Some implementations of Put allow multi-part upload, and allow customizing the
	// chunk size of each part upload, or even disabling multi-part upload.
	//
	// This is a suggestion, implementations may choose to ignore this option.
	SuggestedChunkSize() int

	isPutOptions()
}

// NewPutOptions returns a new PutOptions.
//
// This is used by Bucket implementations.
func NewPutOptions(options []PutOption) PutOptions {
	putOptions := newPutOptions()
	for _, option := range options {
		option(putOptions)
	}
	return putOptions
}

// PutOption is an option passed when putting an object in a bucket.
type PutOption func(*putOptions)

// PutWithSuggestedChunkSize sets the given size in bytes as a suggested chunk
// size to use by the Bucket implementation for this Put call.
// Some implementations of Put allow multi-part upload, and allow customizing the
// chunk size of each part upload, or even disabling multi-part upload.
//
// Setting a suggestedChunkSize of 0 says to suggest disable chunking
// Negative values will be ignored.
//
// This is a suggestion, implementations may choose to ignore this option.
func PutWithSuggestedChunkSize(suggestedChunkSize int) PutOption {
	return func(putOptions *putOptions) {
		if suggestedChunkSize < 0 {
			return
		}
		putOptions.suggestedChunkSize = &suggestedChunkSize
	}
}

// PutWithAtomic ensures that the Put fully writes the file before making it
// available to readers. This happens by default for some implementations,
// while others may need to perform a sequence of operations to ensure
// atomic writes.
//
// The Put operation is complete and the path will be readable once the
// returned WriteObjectCloser is written and closed (without an error).
// Any errors will cause the Put to be skipped (no path will be created).
func PutWithAtomic() PutOption {
	return func(putOptions *putOptions) {
		putOptions.atomic = true
	}
}

type nopReadBucketCloser struct {
	ReadBucket
}

func (nopReadBucketCloser) Close() error {
	return nil
}

type nopWriteBucketCloser struct {
	WriteBucket
}

func (nopWriteBucketCloser) Close() error {
	return nil
}

type nopReadWriteBucketCloser struct {
	ReadWriteBucket
}

func (nopReadWriteBucketCloser) Close() error {
	return nil
}

type putOptions struct {
	atomic             bool
	suggestedChunkSize *int
}

func newPutOptions() *putOptions {
	return &putOptions{}
}

func (p *putOptions) Atomic() bool {
	return p.atomic
}

func (p *putOptions) SuggestedDisableChunking() bool {
	return p.suggestedChunkSize != nil && *p.suggestedChunkSize == 0
}

func (p *putOptions) SuggestedChunkSize() int {
	if p.suggestedChunkSize == nil {
		return 0
	}
	return *p.suggestedChunkSize
}

func (*putOptions) isPutOptions() {}

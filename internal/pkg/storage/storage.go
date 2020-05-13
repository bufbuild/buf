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

// Package storage implements a simple storage abstraction.
//
// This is meant to abstract filesystem calls, as well as be a wrapper
// for in-memory or remote storage. It also provides a smaller attack
// vector as implementations can do verifications as to what is accessed
// and what is not.
package storage

import (
	"context"
	"errors"
	"io"

	"github.com/bufbuild/buf/internal/pkg/normalpath"
)

var (
	// ErrIncompleteWrite is the error returned if a write is not complete
	ErrIncompleteWrite = errors.New("incomplete write")
	// ErrClosed is the error returned if a bucket or object is already closed.
	ErrClosed = errors.New("already closed")

	// errNotExist is the error returned if a path does not exist.
	errNotExist = errors.New("does not exist")
)

// NewErrNotExist returns a new error for a path not existing.
func NewErrNotExist(path string) error {
	return normalpath.NewError(path, errNotExist)
}

// IsNotExist returns true for a error that is for a path not existing.
func IsNotExist(err error) bool {
	return normalpath.ErrorEquals(err, errNotExist)
}

// ObjectInfo contains object info.
type ObjectInfo interface {
	// Size is the size of the object.
	//
	// For writes, the write size must sum up to this size when closed, otherwise ErrIncompleteWrite is returned.
	// For writes, any write over this size will return io.EOF.
	Size() uint32
}

// ReadObject is a read-only object.
//
// It must be closed when done.
type ReadObject interface {
	io.ReadCloser

	Info() ObjectInfo
}

// WriteObject is a write-only object.
//
// It must be closed when done.
type WriteObject interface {
	io.WriteCloser

	Info() ObjectInfo
}

// BucketInfo contains bucket info.
type BucketInfo interface {
	// InMemory returns true if the bucket is in memory.
	InMemory() bool
}

// ReadBucket is a simple read-only bucket.
//
// All paths regular files - Buckets do not handle directories.
// All paths must be relative.
// All paths are cleaned and ToSlash'ed by each function.
// Paths must not jump the bucket context, that is after clean, they
// cannot contain "..".
type ReadBucket interface {
	// Get gets the path.
	//
	// Returns ErrNotExist if the path does not exist, other error
	// if there is a system error.
	Get(ctx context.Context, path string) (ReadObject, error)
	// Stat gets info in the object.
	//
	// Returns ErrNotExist if the path does not exist, other error
	// if there is a system error.
	Stat(ctx context.Context, path string) (ObjectInfo, error)
	// Walk walks the bucket with the prefix, calling f on each path.
	//
	// Note that foo/barbaz will not be called for foo/bar, but will
	// be called for foo/bar/baz.
	//
	// All paths given to f are normalized and validated.
	// If f returns error, Walk will stop short and return this error.
	// Returns other error on system error.
	Walk(ctx context.Context, prefix string, f func(string) error) error

	Info() BucketInfo
}

// ReadBucketCloser is a read-only bucket that must be closed.
type ReadBucketCloser interface {
	io.Closer
	ReadBucket
}

// ReadWriteBucket is a simple read/write bucket.
type ReadWriteBucket interface {
	ReadBucket

	// Put returns a WriteCloser to write to the path.
	//
	// The path is truncated beforehand.
	//
	// Returns error on system error.
	Put(ctx context.Context, path string, size uint32) (WriteObject, error)
}

// ReadWriteBucketCloser is a read/write bucket that must be closed.
type ReadWriteBucketCloser interface {
	io.Closer
	ReadWriteBucket
}

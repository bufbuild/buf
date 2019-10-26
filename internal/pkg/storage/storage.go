// Package storage implements a simple storage abstraction.
//
// This is meant to abstract filesystem calls, as well as be a wrapper
// for in-memory or remote storage. It also provides a smaller attack
// vector as implementations can do verifications as to what is accessed
// and what is not.
package storage

import (
	"context"
	"io"

	"github.com/bufbuild/buf/internal/pkg/errs"
)

var (
	// ErrIncompleteWrite is the error returned if a write is not complete
	ErrIncompleteWrite = errs.NewInternal("incomplete write")
	// ErrClosed is the error returned if a bucket or object is already closed.
	ErrClosed = errs.NewInternal("already closed")

	// errNotExist is the error returned if a path does not exist.
	errNotExist = errs.NewInternal("does not exist")
)

// PathError is a path error.
type PathError struct {
	Path string
	Err  error
}

// Error implements error.
func (p *PathError) Error() string {
	errString := ""
	if p.Err != nil {
		errString = p.Err.Error()
	}
	if errString == "" {
		errString = "error"
	}
	return p.Path + ": " + errString
}

// NewErrNotExist returns a new PathError for a path not existing.
func NewErrNotExist(path string) *PathError {
	return &PathError{
		Path: path,
		Err:  errNotExist,
	}
}

// IsNotExist returns true for a PathError that is for a path not existing.
func IsNotExist(err error) bool {
	if err == nil {
		return false
	}
	pathError, ok := err.(*PathError)
	if !ok {
		return false
	}
	return pathError.Err == errNotExist
}

// ReadObject is a read-only object.
//
// It must be closed when done.
type ReadObject interface {
	io.ReadCloser

	// Size is the size of the object.
	Size() uint32
}

// WriteObject is a write-only object.
//
// It must be closed when done.
type WriteObject interface {
	io.WriteCloser

	// Size is the size of the object.
	//
	// When closed, the writes must sum up to this size, otherwise ErrIncompleteWrite is returned.
	// Any writes over the size will return io.EOF.
	Size() uint32
}

// ReadBucket is a simple read-only bucket.
//
// All paths regular files - Buckets do not handle directories.
// All paths must be relative.
// All paths are cleaned and ToSlash'ed by each function.
// Paths must not jump the bucket context, that is after clean, they
// cannot contain "..".
type ReadBucket interface {
	io.Closer

	// Type returns the type of bucket.
	Type() string
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
}

// Bucket is a simple read/write bucket.
type Bucket interface {
	ReadBucket

	// Put returns a WriteCloser to write to the path.
	//
	// The path is truncated beforehand.
	//
	// Returns error on system error.
	Put(ctx context.Context, path string, size uint32) (WriteObject, error)
}

// ObjectInfo is info on an object.
type ObjectInfo struct {
	Size uint32
}

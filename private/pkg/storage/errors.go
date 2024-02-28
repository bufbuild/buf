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
	"errors"
	"fmt"
	"io/fs"
	"strings"
)

var (
	// ErrClosed is the error returned if a bucket or object is already closed.
	ErrClosed = errors.New("already closed")
	// ErrSetExternalPathUnsupported is the error returned if a bucket does not support SetExternalPath.
	ErrSetExternalPathUnsupported = errors.New("setting the external path is unsupported for this bucket")
	// ErrSetLocalPathUnsupported is the error returned if a bucket does not support SetLocalPath.
	ErrSetLocalPathUnsupported = errors.New("setting the local path is unsupported for this bucket")
)

// NewErrNotExist returns a new error for a path not existing.
//
// Deprecated: use &fs.PathError{Op: "Operation", Path: path, Err: fs.ErrNotExist} instead.
func NewErrNotExist(path string) error {
	// fs.PathErrors should also have Op, but we don't have an obvious one here.
	return &fs.PathError{Path: path, Err: fs.ErrNotExist}
}

// IsNotExist returns true for a error that is for a path not existing.
//
// Deprecated: Use errors.Is(err, fs.ErrNotExist) instead.
func IsNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}

// NewErrExistsMultipleLocations returns a new error if a path exists in multiple locations.
func NewErrExistsMultipleLocations(path string, externalPaths ...string) error {
	return &errorExistsMultipleLocations{
		Path:          path,
		ExternalPaths: externalPaths,
	}
}

// IsExistsMultipleLocations returns true if the error is for a path existing in multiple locations.
func IsExistsMultipleLocations(err error) bool {
	if err == nil {
		return false
	}
	asErr := &errorExistsMultipleLocations{}
	return errors.As(err, &asErr)
}

// IsWriteLimitReached returns true if the error is of writes exceeding the limit of the bucket.
func IsWriteLimitReached(err error) bool {
	if err == nil {
		return false
	}
	asErr := &errWriteLimitReached{}
	return errors.As(err, &asErr)
}

// errorExistsMultipleLocations is the error returned if a path exists in multiple locations.
type errorExistsMultipleLocations struct {
	Path          string
	ExternalPaths []string
}

// Error implements error.
func (e *errorExistsMultipleLocations) Error() string {
	return e.Path + " exists in multiple locations: " + strings.Join(e.ExternalPaths, " ")
}

// errWriteLimitReached is the error returned if the write limit is reached.
//
// See [LimitWriteBucket].
type errWriteLimitReached struct {
	Limit       int64
	ExceedingBy int64
}

// Error implements error.
func (e *errWriteLimitReached) Error() string {
	return fmt.Sprintf("write limit reached: limit: %d, exceeding by: %d", e.Limit, e.ExceedingBy)
}

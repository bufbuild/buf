// Copyright 2020-2022 Buf Technologies, Inc.
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
	"strings"

	"github.com/bufbuild/buf/private/pkg/normalpath"
)

var (
	// ErrClosed is the error returned if a bucket or object is already closed.
	ErrClosed = errors.New("already closed")
	// ErrSetExternalPathUnsupported is the error returned if a bucket does not support SetExternalPath.
	ErrSetExternalPathUnsupported = errors.New("setting the external path is unsupported for this bucket")

	// errNotExist is the error returned if a path does not exist.
	errNotExist = errors.New("does not exist")
	// errInvalidPath is the error returned if a path is invalid.
	errInvalidPath = errors.New("invalid path")
)

// NewErrNotExist returns a new error for a path not existing.
func NewErrNotExist(path string) error {
	return normalpath.NewError(path, errNotExist)
}

// IsNotExist returns true for a error that is for a path not existing.
func IsNotExist(err error) bool {
	return errors.Is(err, errNotExist)
}

// NewErrInvalidPath returns a new error for an invalid path.
func NewErrInvalidPath(path string) error {
	return normalpath.NewError(path, errInvalidPath)
}

// NewErrInvalidPathf returns a new error for an invalid path.
func NewErrInvalidPathf(path string, template string, args ...interface{}) error {
	return normalpath.NewError(path, fmt.Errorf("%w: %s", errInvalidPath, fmt.Sprintf(template, args...)))
}

// IsInvalidPath returns true for a error that is for an invalid path.
func IsInvalidPath(err error) bool {
	return errors.Is(err, errInvalidPath)
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

// errorExistsMultipleLocations is the error returned if a path exists in multiple locations.
type errorExistsMultipleLocations struct {
	Path          string
	ExternalPaths []string
}

// Error implements error.
func (e *errorExistsMultipleLocations) Error() string {
	return e.Path + " exists in multiple locations: " + strings.Join(e.ExternalPaths, " ")
}

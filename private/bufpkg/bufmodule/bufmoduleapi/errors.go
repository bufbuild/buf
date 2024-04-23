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

package bufmoduleapi

import (
	"errors"
	"io/fs"

	"connectrpc.com/connect"
)

// notFoundError represents when a resource was not found.
//
// The underlying error will always be fs.ErrNotExist, to fulfill the bufmodule Provider contracts.
type notFoundError struct {
	message string
}

// Error implements error.
func (n *notFoundError) Error() string {
	if n == nil {
		return ""
	}
	if n.message == "" {
		return "not found"
	}
	return n.message
}

// Unwrap implements errors.Unwrap for Error.
//
// It always returns fs.ErrNotExist if n is not nil.
func (n *notFoundError) Unwrap() error {
	if n == nil {
		return nil
	}
	return fs.ErrNotExist
}

// maybeNewNotFoundError will convert the error into a NotFoundError if it is a connect Error with code NotFound.
//
// It is assumed that the underlying connect Error contains a formatted error message including that the resource was not found.
func maybeNewNotFoundError(err error) error {
	if err == nil {
		return nil
	}
	var connectError *connect.Error
	if !errors.As(err, &connectError) {
		return err
	}
	if connectError.Code() != connect.CodeNotFound {
		return err
	}
	return &notFoundError{
		message: connectError.Message(),
	}
}

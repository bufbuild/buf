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

// Package syserror handles "system errors".
//
// A system error is an error that should never actually happen, and should be
// propogated to the user as a bug in the codebase. System errors are generally
// defensive-programming assertions that we want to check for in our codebase.
//
// If a system error occurs, you may want to send a specialized help that says
// how to file a bug report.
package syserror

import (
	"errors"
	"fmt"
)

// Error is a system error.
type Error struct {
	Underlying error
}

// Error implements error.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Underlying == nil {
		return ""
	}
	return "system error: " + e.Underlying.Error()
}

// Unwrap implements errors.Unwrap for Error.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Underlying
}

// New is a convenience function that returns a new system error by calling errors.new.
func New(text string) *Error {
	return &Error{
		Underlying: errors.New(text),
	}
}

// Newf is a convenience function that returns a new system error by calling fmt.Errorf.
func Newf(format string, args ...any) *Error {
	return &Error{
		Underlying: fmt.Errorf(format, args...),
	}
}

// Wrap returns a new system error for err.
//
// If err is already a system error, this returns err.
func Wrap(err error) error {
	if Is(err) {
		return err
	}
	return &Error{
		Underlying: err,
	}
}

// Is is a convenience function that returns true if err is a system error.
func Is(err error) bool {
	_, ok := As(err)
	return ok
}

// As is a convenience function that returns err as an Error and true if err is an Error,
// and err and false otherwise.
func As(err error) (*Error, bool) {
	target := &Error{}
	ok := errors.As(err, &target)
	return target, ok
}

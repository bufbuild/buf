// Copyright 2020-2023 Buf Technologies, Inc.
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

// Wrap returns a new system error for the underlying error.
func Wrap(err error) error {
	if Is(err) {
		return err
	}
	return &sysError{
		err: err,
	}
}

// New is a convenience function that returns a new system error by calling errors.new.
func New(text string) error {
	return &sysError{
		err: errors.New(text),
	}
}

// Newf is a convenience function that returns a new system error by calling fmt.Errorf.
func Newf(format string, args ...any) error {
	return &sysError{
		err: fmt.Errorf(format, args...),
	}
}

// Is returns true if the error is a system error.
func Is(err error) bool {
	if err == nil {
		return false
	}
	target := &sysError{}
	return errors.As(err, &target)

}

type sysError struct {
	err error
}

func (e *sysError) Error() string {
	if e == nil {
		return ""
	}
	if e.err == nil {
		return ""
	}
	return "system error: " + e.err.Error()
}

func (e *sysError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

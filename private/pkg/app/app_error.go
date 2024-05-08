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

package app

import (
	"errors"
	"fmt"
)

type appError struct {
	exitCode int
	err      error
}

func newAppError(exitCode int, err error) *appError {
	if exitCode == 0 {
		err = fmt.Errorf(
			"got invalid exit code %d when constructing appError (original error was %w)",
			exitCode,
			err,
		)
		exitCode = 1
	}
	if err == nil {
		err = errors.New("got nil error when constructing appError")
	}
	return &appError{
		exitCode: exitCode,
		err:      err,
	}
}

func (e *appError) Error() string {
	if e == nil {
		return ""
	}
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *appError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func printError(container StderrContainer, err error) {
	if errString := err.Error(); errString != "" {
		_, _ = fmt.Fprintln(container.Stderr(), errString)
	}
}

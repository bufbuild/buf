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

package bufcli

import (
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
)

var (
	// ErrNotATTY is returned when an input io.Reader is not a TTY where it is expected.
	ErrNotATTY = errors.New("reader was not a TTY as expected")

	// ErrNoConfigFile is used when the user tries to execute a command without a configuration file.
	ErrNoConfigFile = errors.New(`no buf.yaml file discovered in the specified directory`)
)

// NewTooManyEmptyAnswersError is used when the user does not answer a prompt in
// the given number of attempts.
func NewTooManyEmptyAnswersError(attempts int) error {
	return fmt.Errorf("did not receive an answer in %d attempts", attempts)
}

// NewOrganizationNameAlreadyExistsError informs the user that an organization with
// that name already exists.
func NewOrganizationNameAlreadyExistsError(name string) error {
	return fmt.Errorf("an organization named %q already exists", name)
}

// NewRepositoryNameAlreadyExistsError informs the user that a repository
// with that name already exists.
func NewRepositoryNameAlreadyExistsError(name string) error {
	return fmt.Errorf("a repository named %q already exists", name)
}

// NewLabelNameAlreadyExistsError informs the user that a label
// with that name already exists.
func NewLabelNameAlreadyExistsError(name string) error {
	return fmt.Errorf("a label named %q already exists", name)
}

// NewOrganizationNotFoundError informs the user that an organization with
// that name does not exist.
func NewOrganizationNotFoundError(name string) error {
	return fmt.Errorf(`an organization named %q does not exist, use "buf beta registry organization create" to create one`, name)
}

// NewOrganizationOrUserNotFoundError informs the user that an organization or user with
// that name does not exist.
func NewOrganizationOrUserNotFoundError(name string) error {
	return fmt.Errorf(`an organization or user named %q does not exist`, name)
}

// NewRepositoryNotFoundError informs the user that a repository with
// that name does not exist.
func NewRepositoryNotFoundError(name string) error {
	return fmt.Errorf(`a repository named %q does not exist, use "buf beta registry repository create" to create one`, name)
}

// NewModuleRefNotFoundError informs the user that a ModuleRef does not exist.
func NewModuleRefNotFoundError(moduleRef bufmodule.ModuleRef) error {
	return fmt.Errorf("%q does not exist", moduleRef)
}

// NewLabelNotFoundError informs the user that a ModuleRef does not exist as a label.
func NewLabelNotFoundError(moduleRef bufmodule.ModuleRef) error {
	return fmt.Errorf("label %q does not exist", moduleRef)
}

// NewTokenNotFoundError informs the user that a token with
// that identifier does not exist.
func NewTokenNotFoundError(tokenID string) error {
	return fmt.Errorf("a token with ID %q does not exist", tokenID)
}

// NewInvalidRemoteError informs the user that the given remote is invalid.
func NewInvalidRemoteError(err error, remote string, moduleFullName string) error {
	var connectErr *connect.Error
	ok := errors.As(err, &connectErr)
	if ok {
		err = connectErr.Unwrap()
	}
	return fmt.Errorf("%w. Are you sure %q (derived from module name %q) is a Buf Schema Registry?", err, remote, moduleFullName)
}

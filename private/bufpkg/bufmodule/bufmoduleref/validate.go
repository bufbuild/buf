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

package bufmoduleref

import (
	"errors"
	"fmt"
	"strings"

	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/netext"
)

// ValidateProtoModuleReference verifies the given module reference is well-formed.
// It performs client-side validation only, and is limited to fields
// we do not think will change in the future.
func ValidateProtoModuleReference(protoModuleReference *modulev1alpha1.ModuleReference) error {
	if protoModuleReference == nil {
		return errors.New("module reference is required")
	}
	if err := validateRemote(protoModuleReference.Remote); err != nil {
		return err
	}
	if err := validateOwner(protoModuleReference.Owner, "owner"); err != nil {
		return err
	}
	if err := validateRepository(protoModuleReference.Repository); err != nil {
		return err
	}
	return validateReference(protoModuleReference.Reference)
}

// ValidateProtoModulePin verifies the given module pin is well-formed.
// It performs client-side validation only, and is limited to fields
// we do not think will change in the future.
func ValidateProtoModulePin(protoModulePin *modulev1alpha1.ModulePin) error {
	if protoModulePin == nil {
		return errors.New("module pin is required")
	}
	if err := validateRemote(protoModulePin.Remote); err != nil {
		return err
	}
	if err := validateOwner(protoModulePin.Owner, "owner"); err != nil {
		return err
	}
	if err := validateRepository(protoModulePin.Repository); err != nil {
		return err
	}
	if err := validateCommit(protoModulePin.Commit); err != nil {
		return err
	}
	return nil
}

// ValidateModuleFilePath validates that the module file path is not empty.
// It performs client-side validation only, and is limited to properties
// we do not think will change in the future.
func ValidateModuleFilePath(path string) error {
	if path == "" {
		return errors.New("empty path")
	}
	return nil
}

// ValidateRemoteNotEmpty validates that the given remote address is not an empty string
// It performs client-side validation only, and is limited to fields
// we do not think will change in the future.
func ValidateRemoteNotEmpty(remote string) error {
	if remote == "" {
		return appcmd.NewInvalidArgumentError("you must specify a remote module")
	}
	return nil
}

// ValidateRemoteHasNoPaths validates that the given remote address contains no paths/subdirectories after the root
// It performs client-side validation only, and is limited to fields
// we do not think will change in the future.
func ValidateRemoteHasNoPaths(remote string) error {
	_, path, ok := strings.Cut(remote, "/")
	if ok && path != "" {
		return appcmd.NewInvalidArgumentError(fmt.Sprintf(`invalid remote address, must not contain any paths. Try removing "/%s" from the address.`, path))
	}
	return nil
}

func validateModuleOwner(moduleOwner ModuleOwner) error {
	if moduleOwner == nil {
		return errors.New("module owner is required")
	}
	if err := validateRemote(moduleOwner.Remote()); err != nil {
		return err
	}
	if err := validateOwner(moduleOwner.Owner(), "owner"); err != nil {
		return err
	}
	return nil
}

func validateModuleIdentity(moduleIdentity ModuleIdentity) error {
	if moduleIdentity == nil {
		return errors.New("module identity is required")
	}
	if err := validateRemote(moduleIdentity.Remote()); err != nil {
		return err
	}
	if err := validateOwner(moduleIdentity.Owner(), "owner"); err != nil {
		return err
	}
	if err := validateRepository(moduleIdentity.Repository()); err != nil {
		return err
	}
	return nil
}

func validateRemote(remote string) error {
	if _, err := netext.ValidateHostname(remote); err != nil {
		return fmt.Errorf("invalid remote %q: %w", remote, err)
	}
	return nil
}

func validateOwner(owner string, ownerType string) error {
	if owner == "" {
		return fmt.Errorf("%s name is required", ownerType)
	}
	return nil
}

func validateRepository(repository string) error {
	if repository == "" {
		return errors.New("repository name is required")
	}
	return nil
}

func validateReference(reference string) error {
	if reference == "" {
		return errors.New("repository reference is required")
	}
	return nil
}

func validateCommit(commit string) error {
	if commit == "" {
		return errors.New("empty commit")
	}
	return nil
}

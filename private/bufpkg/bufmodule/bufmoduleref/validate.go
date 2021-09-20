// Copyright 2020-2021 Buf Technologies, Inc.
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
	"github.com/bufbuild/buf/private/pkg/netextended"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	if err := ValidateOwner(protoModuleReference.Owner, "owner"); err != nil {
		return err
	}
	if err := ValidateRepository(protoModuleReference.Repository); err != nil {
		return err
	}
	return ValidateReference(protoModuleReference.Reference)
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
	if err := ValidateOwner(protoModulePin.Owner, "owner"); err != nil {
		return err
	}
	if err := ValidateRepository(protoModulePin.Repository); err != nil {
		return err
	}
	if err := ValidateBranch(protoModulePin.Branch); err != nil {
		return err
	}
	if err := ValidateCommit(protoModulePin.Commit); err != nil {
		return err
	}
	if err := validateDigest(protoModulePin.Digest); err != nil {
		return err
	}
	if err := validateCreateTime(protoModulePin.CreateTime); err != nil {
		return err
	}
	return nil
}

// ValidateUser verifies the given user name is well-formed.
// It performs client-side validation only, and is limited to properties
// we do not think will change in the future.
func ValidateUser(user string) error {
	return ValidateOwner(user, "user")
}

// ValidateOrganization verifies the given organization name is well-formed.
// It performs client-side validation only, and is limited to properties
// we do not think will change in the future.
func ValidateOrganization(organization string) error {
	return ValidateOwner(organization, "organization")
}

// ValidateOwner verifies the given owner name is well-formed.
// It performs client-side validation only, and is limited to properties
// we do not think will change in the future.
func ValidateOwner(owner string, ownerType string) error {
	if owner == "" {
		return fmt.Errorf("%s name is required", ownerType)
	}
	return nil
}

// ValidateRepository verifies the given repository name is well-formed.
// It performs client-side validation only, and is limited to properties
// we do not think will change in the future.
func ValidateRepository(repository string) error {
	if repository == "" {
		return errors.New("repository name is required")
	}
	return nil
}

// ValidateReference validates that the given ModuleReference reference is well-formed.
// It performs client-side validation only, and is limited to properties
// we do not think will change in the future.
func ValidateReference(reference string) error {
	if reference == "" {
		return errors.New("repository reference is required")
	}
	return nil
}

// ValidateCommit verifies the given commit is well-formed.
// It performs client-side validation only, and is limited to properties
// we do not think will change in the future.
func ValidateCommit(commit string) error {
	if commit == "" {
		return errors.New("empty commit")
	}
	return nil
}

// ValidateBranch verifies the given repository branch is well-formed.
// It performs client-side validation only, and is limited to properties
// we do not think will change in the future.
func ValidateBranch(branch string) error {
	if branch != MainBranch {
		return fmt.Errorf("branch is not %s", MainBranch)
	}
	//if branch == "" {
	//	return errors.New("repository branch is required")
	//}
	return nil
}

// ValidateTag verifies the given tag is well-formed.
// It performs client-side validation only, and is limited to properties
// we do not think will change in the future.
func ValidateTag(tag string) error {
	if tag == "" {
		return errors.New("repository tag is required")
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

// validateDigest verifies the given digest's prefix,
// decodes its base64 representation and checks the
// length of the encoded bytes.
// It performs client-side validation only, and is limited to properties
// we do not think will change in the future.
func validateDigest(digest string) error {
	if digest == "" {
		return errors.New("empty digest")
	}
	split := strings.SplitN(digest, "-", 2)
	if len(split) != 2 {
		return fmt.Errorf("invalid digest: %s", digest)
	}
	digestPrefix := split[0]
	digestValue := split[1]
	if digestPrefix == "" {
		return fmt.Errorf("empty digest prefix for %s", digest)
	}
	if digestValue == "" {
		return fmt.Errorf("empty digest value for %s", digest)
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
	if err := ValidateOwner(moduleOwner.Owner(), "owner"); err != nil {
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
	if err := ValidateOwner(moduleIdentity.Owner(), "owner"); err != nil {
		return err
	}
	if err := ValidateRepository(moduleIdentity.Repository()); err != nil {
		return err
	}
	return nil
}

func validateRemote(remote string) error {
	if _, err := netextended.ValidateHostname(remote); err != nil {
		return fmt.Errorf("invalid remote %q: %w", remote, err)
	}
	return nil
}

func validateCreateTime(createTime *timestamppb.Timestamp) error {
	if createTime == nil {
		return errors.New("create_time is required")
	}
	if createTime.Seconds == 0 && createTime.Nanos == 0 {
		return errors.New("create_time must not be 0")
	}
	return createTime.CheckValid()
}

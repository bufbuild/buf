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

package bufmodule

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	modulev1alpha1 "github.com/bufbuild/buf/internal/gen/proto/go/buf/alpha/module/v1alpha1"
	"github.com/bufbuild/buf/internal/pkg/netextended"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
	"github.com/bufbuild/buf/internal/pkg/uuidutil"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	ownerMinLength      = 4
	ownerMaxLength      = 32
	repositoryMinLength = 2
	repositoryMaxLength = 32
	branchMinLength     = 2
	branchMaxLength     = 64
	// 32MB
	maxModuleTotalContentLength = 32 << 20
	protoFileMaxCount           = 16384
)

// ValidateProtoModule verifies the given module is well-formed.
func ValidateProtoModule(protoModule *modulev1alpha1.Module) error {
	if protoModule == nil {
		return errors.New("module is required")
	}
	if len(protoModule.Files) == 0 {
		return errors.New("module has no files")
	}
	if len(protoModule.Files) > protoFileMaxCount {
		return fmt.Errorf("module can contain at most %d files", protoFileMaxCount)
	}
	totalContentLength := 0
	filePathMap := make(map[string]struct{}, len(protoModule.Files))
	for _, protoModuleFile := range protoModule.Files {
		if err := validateModuleFilePath(protoModuleFile.Path); err != nil {
			return err
		}
		if _, ok := filePathMap[protoModuleFile.Path]; ok {
			return fmt.Errorf("duplicate module file path: %s", protoModuleFile.Path)
		}
		filePathMap[protoModuleFile.Path] = struct{}{}
		totalContentLength += len(protoModuleFile.Content)
	}
	if totalContentLength > maxModuleTotalContentLength {
		return fmt.Errorf("total module content length is %d when max is %d", totalContentLength, maxModuleTotalContentLength)
	}
	for _, dependency := range protoModule.Dependencies {
		if err := ValidateProtoModulePin(dependency); err != nil {
			return fmt.Errorf("module had invalid dependency: %v", err)
		}
	}
	return nil
}

// ValidateProtoModuleReference verifies the given module reference is well-formed.
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
	branch := protoModuleReference.GetBranch()
	commit := protoModuleReference.GetCommit()
	switch {
	case branch == "" && commit == "":
		return fmt.Errorf("module reference must have either a branch or commit")
	case branch != "" && commit == "":
		if err := ValidateBranch(branch); err != nil {
			return err
		}
	case branch == "" && commit != "":
		if err := ValidateCommit(commit); err != nil {
			return err
		}
	default:
		// should never happen due to oneof
		return fmt.Errorf("module reference cannot have both a branch and commit")
	}
	return nil
}

// ValidateProtoModulePin verifies the given module pin is well-formed.
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
	if err := ValidateDigest(protoModulePin.Digest); err != nil {
		return err
	}
	if err := validateCreateTime(protoModulePin.CreateTime); err != nil {
		return err
	}
	return nil
}

// ValidateUser verifies the given user name is well-formed.
func ValidateUser(user string) error {
	return ValidateOwner(user, "user")
}

// ValidateOrganization verifies the given organization name is well-formed.
func ValidateOrganization(organization string) error {
	return ValidateOwner(organization, "organization")
}

// ValidateOwner verifies the given owner name is well-formed.
func ValidateOwner(owner string, ownerType string) error {
	if owner == "" {
		return fmt.Errorf("%s name is required", ownerType)
	}
	if len(owner) < ownerMinLength || len(owner) > ownerMaxLength {
		return fmt.Errorf("%s name %q must be between at least %d and at most %d characters", ownerType, owner, ownerMinLength, ownerMaxLength)
	}
	for _, char := range owner {
		if !stringutil.IsLowerAlphanumeric(char) && char != '-' {
			return fmt.Errorf("%s name %q must only contain lowercase letters, digits, or hyphens (-)", ownerType, owner)
		}
	}
	return nil
}

// ValidateRepository verifies the given repository name is well-formed.
func ValidateRepository(repository string) error {
	if repository == "" {
		return errors.New("repository name is required")
	}
	if len(repository) < repositoryMinLength || len(repository) > repositoryMaxLength {
		return fmt.Errorf("repository name must be at least %d and at most %d characters", repositoryMinLength, repositoryMaxLength)
	}
	for _, char := range repository {
		if !stringutil.IsLowerAlphanumeric(char) && char != '-' {
			return fmt.Errorf("repository name %q must only contain lowercase letters, digits, or hyphens (-)", repository)
		}
	}
	return nil
}

// ValidateBranch verifies the given repository branch is well-formed.
func ValidateBranch(branch string) error {
	if branch == "" {
		return errors.New("repository branch is required")
	}
	if len(branch) < branchMinLength || len(branch) > branchMaxLength {
		return fmt.Errorf("repository branch %q must be at least %d and at most %d characters", branch, branchMinLength, branchMaxLength)
	}
	for _, char := range branch {
		if !stringutil.IsLowerAlphanumeric(char) && char != '-' && char != '.' {
			return fmt.Errorf("repository branch %q must only contain lowercase letters, digits, periods (.), or hyphens (-)", branch)
		}
	}
	if err := uuidutil.ValidateDashless(branch); err == nil {
		return fmt.Errorf("repository branch %q must not be parseable as a valid commit", branch)
	}
	return nil
}

// ValidateCommit verifies the given commit is well-formed.
func ValidateCommit(commit string) error {
	if commit == "" {
		return errors.New("empty commit")
	}
	if err := uuidutil.ValidateDashless(commit); err != nil {
		return fmt.Errorf("commit is invalid: %v", err)
	}
	return nil
}

// ValidateDigest verifies the given digest's prefix,
// decodes its base64 representation and checks the
// length of the encoded bytes.
func ValidateDigest(digest string) error {
	if digest == "" {
		return errors.New("empty digest")
	}
	split := strings.SplitN(digest, "-", 2)
	if len(split) != 2 {
		return fmt.Errorf("invalid digest: %s", digest)
	}
	digestPrefix := split[0]
	digestValue := split[1]
	if digestPrefix != b1DigestPrefix {
		return fmt.Errorf("unknown digest prefix: %s", digestPrefix)
	}
	decoded, err := base64.URLEncoding.DecodeString(digestValue)
	if err != nil {
		return fmt.Errorf("failed to decode digest %s: %v", digestValue, err)
	}
	if len(decoded) != 32 {
		return fmt.Errorf("invalid sha256 hash, expected 32 bytes: %s", digestValue)
	}
	return nil
}

// ValidateModuleMatchesDigest validates that the Module matches the digest.
//
// This is just a convenience function.
func ValidateModuleMatchesDigest(ctx context.Context, module Module, modulePin ModulePin) error {
	digest, err := ModuleDigest(ctx, module)
	if err != nil {
		return err
	}
	if digest != modulePin.Digest() {
		return fmt.Errorf("mismatched module digest for %q: expected: %q got: %q", modulePin.IdentityString(), modulePin.Digest(), digest)
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

func validateModuleFilePath(path string) error {
	normalizedPath, err := normalpath.NormalizeAndValidate(path)
	if err != nil {
		return err
	}
	if path != normalizedPath {
		return fmt.Errorf("module file had non-normalized path: %s", path)
	}
	return validateModuleFilePathWithoutNormalization(path)
}

func validateModuleFilePathWithoutNormalization(path string) error {
	if path == "" {
		return errors.New("empty path")
	}
	if normalpath.Ext(path) != ".proto" {
		return fmt.Errorf("path %s did not have extension .proto", path)
	}
	return nil
}

// Copyright 2020 Buf Technologies, Inc.
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
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	modulev1 "github.com/bufbuild/buf/internal/gen/proto/go/buf/module/v1"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
)

const (
	// 32MB
	maxModuleTotalContentLength    = 32 << 20
	ownerNameMinLength             = 3
	ownerNameMaxLength             = 64
	protoFileMaxCount              = 16384
	remoteMinLength                = 1
	remoteMaxLength                = 256
	repositoryNameMinLength        = 2
	repositoryNameMaxLength        = 64
	repositoryVersionNameMinLength = 2
	repositoryVersionNameMaxLength = 32
)

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

// ValidateProtoModule verifies the given module is well-formed.
func ValidateProtoModule(protoModule *modulev1.Module) error {
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
	return nil
}

// ValidateProtoModuleName verifies the given module name is well-formed.
func ValidateProtoModuleName(protoModuleName *modulev1.ModuleName) error {
	if protoModuleName == nil {
		return errors.New("module name is required")
	}
	if err := validateRemote(protoModuleName.Remote); err != nil {
		return err
	}
	if err := ValidateOwnerName(protoModuleName.Owner, "owner"); err != nil {
		return err
	}
	if err := ValidateRepositoryName(protoModuleName.Repository); err != nil {
		return err
	}
	if err := ValidateRepositoryVersionName(protoModuleName.Version); err != nil {
		return err
	}
	if protoModuleName.Digest != "" {
		if err := ValidateDigest(protoModuleName.Digest); err != nil {
			return err
		}
	}
	return nil
}

// ValidateOwnerName is used to validate owner names, i.e. usernames and organization names.
func ValidateOwnerName(ownerName string, ownerType string) error {
	if ownerName == "" {
		return fmt.Errorf("%s name is required", ownerType)
	}
	if len(ownerName) < ownerNameMinLength || len(ownerName) > ownerNameMaxLength {
		return fmt.Errorf("%s name %q must be between at least %d and at most %d characters", ownerType, ownerName, ownerNameMinLength, ownerNameMaxLength)
	}
	for _, char := range ownerName {
		if !isLowerAlphanumeric(char) && char != '-' {
			return fmt.Errorf("%s name %q must only contain lowercase letters, digits, or hyphens (-)", ownerType, ownerName)
		}
	}
	return nil
}

// ValidateRepositoryName verifies the given repository name is well-formed.
func ValidateRepositoryName(repositoryName string) error {
	if repositoryName == "" {
		return errors.New("repository name is required")
	}
	if len(repositoryName) < repositoryNameMinLength || len(repositoryName) > repositoryNameMaxLength {
		return fmt.Errorf("repository name must be at least %d and at most %d characters", repositoryNameMinLength, repositoryNameMaxLength)
	}
	for _, char := range repositoryName {
		if !isLowerAlphanumeric(char) && char != '-' {
			return fmt.Errorf("repository name %q must only contain lowercase letters, digits, or hyphens (-)", repositoryName)
		}
	}
	return nil
}

// ValidateRepositoryVersionName verifies the given repository version name is well-formed.
func ValidateRepositoryVersionName(versionName string) error {
	if versionName == "" {
		return errors.New("repository version name is required")
	}
	if len(versionName) < repositoryVersionNameMinLength || len(versionName) > repositoryVersionNameMaxLength {
		return fmt.Errorf("repository version name %q must be at least %d and at most %d characters", versionName, repositoryVersionNameMinLength, repositoryVersionNameMaxLength)
	}
	for _, char := range versionName {
		if !isLowerAlphanumeric(char) && char != '-' && char != '.' {
			return fmt.Errorf("repository version name %q must only contain lowercase letters, digits, periods (.), or hyphens (-)", versionName)
		}
	}
	return nil
}

func validateRemote(remote string) error {
	if remote == "" {
		return errors.New("remote is required")
	}
	if len(remote) < remoteMinLength || len(remote) > remoteMaxLength {
		return fmt.Errorf("remote %q must be at least %d and at most %d characters", remote, remoteMinLength, remoteMaxLength)
	}
	return nil
}

func validateModuleFilePaths(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	pathMap := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		if err := validateModuleFilePath(path); err != nil {
			return err
		}
		if _, ok := pathMap[path]; ok {
			return fmt.Errorf("duplicate module file path: %s", path)
		}
		pathMap[path] = struct{}{}
	}
	return nil
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

// islowerAlphanumeric returns true for [0-9a-z].
func isLowerAlphanumeric(r rune) bool {
	return ('0' <= r && r <= '9') || ('a' <= r && r <= 'z')
}

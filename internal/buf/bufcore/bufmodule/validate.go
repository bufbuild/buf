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

// 32MB
const maxModuleTotalContentLength = 32 << 20

func validateProtoModule(protoModule *modulev1.Module) error {
	if protoModule == nil {
		return errors.New("nil Module")
	}
	if len(protoModule.Files) == 0 {
		return errors.New("Module had no files")
	}
	if err := protoModule.Validate(); err != nil {
		return err
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

func validateProtoModuleName(protoModuleName *modulev1.ModuleName) error {
	if protoModuleName == nil {
		return errors.New("nil ModuleName")
	}
	if err := protoModuleName.Validate(); err != nil {
		return err
	}
	if protoModuleName.Digest != "" {
		if err := validateDigest(protoModuleName.Digest); err != nil {
			return err
		}
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

// validateDigest verifies the given digest's prefix,
// decodes its base64 representation and checks the
// length of the encoded bytes.
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

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

package bufmodule

import (
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
)

const (
	// 32MB
	maxModuleTotalContentLength = 32 << 20
	protoFileMaxCount           = 16384
)

// ValidateProtoModule verifies the given module is well-formed.
// It performs client-side validation only, and is limited to fields
// we do not think will change in the future.
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
		if err := bufmoduleref.ValidateModuleFilePath(protoModuleFile.Path); err != nil {
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
		if err := bufmoduleref.ValidateProtoModulePin(dependency); err != nil {
			return fmt.Errorf("module had invalid dependency: %v", err)
		}
	}
	return nil
}

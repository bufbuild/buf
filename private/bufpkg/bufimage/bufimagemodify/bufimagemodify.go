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

package bufimagemodify

import (
	"fmt"
	"strconv"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/gen/data/datawkt"
	"google.golang.org/protobuf/types/descriptorpb"
)

// TODO: remove code dealing with the old config (maps)

// TODO: rename this one
func isWellKnownType(imageFile bufimage.ImageFile) bool {
	return datawkt.Exists(imageFile.Path())
}

// int32SliceIsEqual returns true if x and y contain the same elements.
func int32SliceIsEqual(x []int32, y []int32) bool {
	if len(x) != len(y) {
		return false
	}
	for i, elem := range x {
		if elem != y[i] {
			return false
		}
	}
	return true
}

func stringOverridesToBoolOverrides(stringOverrides map[string]string) (map[string]bool, error) {
	validatedOverrides := make(map[string]bool, len(stringOverrides))
	for fileImportPath, overrideString := range stringOverrides {
		overrideBool, err := strconv.ParseBool(overrideString)
		if err != nil {
			return nil, fmt.Errorf("non-boolean override %s set for file %s", overrideString, fileImportPath)
		}
		validatedOverrides[fileImportPath] = overrideBool
	}
	return validatedOverrides, nil
}

func stringOverridesToOptimizeModeOverrides(stringOverrides map[string]string) (map[string]descriptorpb.FileOptions_OptimizeMode, error) {
	validatedOverrides := make(map[string]descriptorpb.FileOptions_OptimizeMode, len(stringOverrides))
	for fileImportPath, stringOverride := range stringOverrides {
		optimizeMode, ok := descriptorpb.FileOptions_OptimizeMode_value[stringOverride]
		if !ok {
			return nil, fmt.Errorf("invalid optimize mode %s set for file %s", stringOverride, fileImportPath)
		}
		validatedOverrides[fileImportPath] = descriptorpb.FileOptions_OptimizeMode(optimizeMode)
	}
	return validatedOverrides, nil
}

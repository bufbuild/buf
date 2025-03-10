// Copyright 2020-2025 Buf Technologies, Inc.
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

package protosourcepath

import (
	"slices"

	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	enumValueNameTypeTag   = int32(1)
	enumValueNumberTypeTag = int32(2)
	enumValueOptionTypeTag = int32(3)
)

var (
	terminalEnumValueTokens = []int32{
		enumValueNameTypeTag,
		enumValueNumberTypeTag,
	}
)

// enumValues is the state when an element representing enum values in the source path was
// parsed.
func enumValues(
	_ int32,
	fullSourcePath protoreflect.SourcePath,
	index int,
	excludeChildAssociatedPaths bool,
) (state, []protoreflect.SourcePath, error) {
	associatedPaths := []protoreflect.SourcePath{
		currentPath(fullSourcePath, index),
	}
	if !excludeChildAssociatedPaths {
		associatedPaths = append(
			associatedPaths,
			childAssociatedPath(fullSourcePath, index, enumValueNameTypeTag),
			childAssociatedPath(fullSourcePath, index, enumValueNumberTypeTag),
		)
	}
	if len(fullSourcePath) == index+1 {
		// This does not extend beyond the enum value declaration, return associated paths and
		// terminate here.
		return nil, associatedPaths, nil
	}
	return enumValue, associatedPaths, nil
}

// enumValue is the state when an element representing a specific child path of an enum was
// parsed.
func enumValue(token int32, sourcePath protoreflect.SourcePath, i int, _ bool) (state, []protoreflect.SourcePath, error) {
	if slices.Contains(terminalEnumValueTokens, token) {
		// Encountered a terminal enum value path, terminate here.
		return nil, nil, nil
	}
	switch token {
	case enumValueOptionTypeTag:
		// For options, we add the full path and then return the options state to validate
		// the path.
		return options, []protoreflect.SourcePath{slices.Clone(sourcePath)}, nil
	}
	return nil, nil, newInvalidSourcePathError(sourcePath, "invalid enum value path")
}

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

func enumValues(_ int32, sourcePath protoreflect.SourcePath, i int) (state, []protoreflect.SourcePath, error) {
	// TODO(doria): should we handle the index?
	associatedPaths := []protoreflect.SourcePath{
		currentPath(sourcePath, i),
		childAssociatedPath(sourcePath, i, enumValueNameTypeTag),
		childAssociatedPath(sourcePath, i, enumValueNumberTypeTag),
	}
	if len(sourcePath) == i+1 {
		// This does not extend beyond the enum value declaration, return the name and number
		// as associated paths and terminate here:
		return nil, associatedPaths, nil
	}
	// Otherwise, continue to the enum value structure
	return enumValue, associatedPaths, nil
}

func enumValue(token int32, sourcePath protoreflect.SourcePath, i int) (state, []protoreflect.SourcePath, error) {
	if slices.Contains(terminalEnumValueTokens, token) {
		// Encountered a terminal enum value token, validate and terminate here
		if len(sourcePath) != i+1 {
			return nil, nil, newInvalidSourcePathError(sourcePath, "invalid enum value path")
		}
		return nil, nil, nil
	}
	switch token {
	case enumValueOptionTypeTag:
		if len(sourcePath) < i+2 {
			return nil, nil, newInvalidSourcePathError(sourcePath, "cannot have enum value option declaration without option number")
		}
		return options, nil, nil
	}
	// TODO(doria): implement non-terminal enum value tokens
	return nil, nil, newInvalidSourcePathError(sourcePath, "invalid or unimplemented source path")
}

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
	enumNameTypeTag          = int32(1)
	enumValuesTypeTag        = int32(2)
	enumOptionTypeTag        = int32(3)
	enumReservedRangeTypeTag = int32(4)
	enumReservedNameTypeTag  = int32(5)
)

// enums is the state when an element representing enums in the source path was parsed.
func enums(
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
			childAssociatedPath(fullSourcePath, index, enumNameTypeTag),
		)
	}
	if len(fullSourcePath) == index+1 {
		// This path does not extend beyond the enum declaration, return associated paths and
		// terminate here.
		return nil, associatedPaths, nil
	}
	return enum, associatedPaths, nil
}

// enum is the state when an element representing a specific child path of an enum was parsed.
func enum(token int32, fullSourcePath protoreflect.SourcePath, index int, _ bool) (state, []protoreflect.SourcePath, error) {
	switch token {
	case enumNameTypeTag:
		// The enum name has already been added, can terminate here immediately.
		return nil, nil, nil
	case enumValuesTypeTag:
		// We check to make sure that the length of the source path contains at least the current
		// token and an index. This is because all source paths for enum values are expected
		// to have indices.
		if len(fullSourcePath) < index+2 {
			return nil, nil, newInvalidSourcePathError(fullSourcePath, "cannot have enum value declaration without index")
		}
		return enumValues, nil, nil
	case enumOptionTypeTag:
		// For options, we add the full path and then return the options state to validate
		// the path.
		return options, []protoreflect.SourcePath{slices.Clone(fullSourcePath)}, nil
	case enumReservedRangeTypeTag:
		// For reserved ranges, we add the full path and then return the reserved ranges state to
		// validate the path.
		return reservedRanges, []protoreflect.SourcePath{currentPath(fullSourcePath, index)}, nil
	case enumReservedNameTypeTag:
		// For reserved names, we add the full path and then return the reserved names state to
		// validate the path.
		return reservedNames, []protoreflect.SourcePath{currentPath(fullSourcePath, index)}, nil
	}
	return nil, nil, newInvalidSourcePathError(fullSourcePath, "invalid enum path")
}

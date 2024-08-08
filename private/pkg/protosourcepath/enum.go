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
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	enumNameTypeTag          = int32(1)
	enumValuesTypeTag        = int32(2)
	enumOptionTypeTag        = int32(3)
	enumReservedRangeTypeTag = int32(4)
	enumReservedNameTypeTag  = int32(5)
)

func enums(_ int32, sourcePath protoreflect.SourcePath, i int) (state, []protoreflect.SourcePath, error) {
	// TODO(doria): should we handle the index?
	// Add enum declaration and enum name to associated paths
	associatedPaths := []protoreflect.SourcePath{
		currentPath(sourcePath, i),
		childAssociatedPath(sourcePath, i, enumNameTypeTag),
	}
	if len(sourcePath) == i+1 {
		// This does not extend beyond the declaration, return associated paths and terminate here.
		return nil, associatedPaths, nil
	}
	// Otherwise, move on to enum structure
	return enum, associatedPaths, nil
}

func enum(token int32, sourcePath protoreflect.SourcePath, i int) (state, []protoreflect.SourcePath, error) {
	switch token {
	case enumNameTypeTag:
		// This is the enum name, which is already added, can termiante here immediately.
		return nil, nil, nil
	case enumValuesTypeTag:
		return enumValues, nil, nil
	case enumOptionTypeTag:
		if len(sourcePath) < i+2 {
			return nil, nil, newInvalidSourcePathError(sourcePath, "cannot have enum option declaration without option number")
		}
		return options, nil, nil
	case enumReservedRangeTypeTag:
		return reservedRanges, []protoreflect.SourcePath{currentPath(sourcePath, i)}, nil
	case enumReservedNameTypeTag:
		return reservedNames, []protoreflect.SourcePath{currentPath(sourcePath, i)}, nil
	}
	return nil, nil, newInvalidSourcePathError(sourcePath, "invalid source path")
}

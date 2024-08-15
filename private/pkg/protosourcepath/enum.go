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
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	enumNameTypeTag          = int32(1)
	enumValuesTypeTag        = int32(2)
	enumOptionTypeTag        = int32(3)
	enumReservedRangeTypeTag = int32(4)
	enumReservedNameTypeTag  = int32(5)
)

func enums(
	_ int32,
	sourcePath protoreflect.SourcePath,
	i int,
	excludeChildAssociatedPaths bool,
) (state, []protoreflect.SourcePath, error) {
	associatedPaths := []protoreflect.SourcePath{
		currentPath(sourcePath, i),
	}
	if !excludeChildAssociatedPaths {
		associatedPaths = append(
			associatedPaths,
			childAssociatedPath(sourcePath, i, enumNameTypeTag),
		)
	}
	if len(sourcePath) == i+1 {
		// This path does not extend beyond the enum declaration, return associated paths and
		// terminate here.
		return nil, associatedPaths, nil
	}
	return enum, associatedPaths, nil
}

func enum(token int32, sourcePath protoreflect.SourcePath, i int, _ bool) (state, []protoreflect.SourcePath, error) {
	switch token {
	case enumNameTypeTag:
		// The enum name has already been added, can terminate here immediately.
		return nil, nil, nil
	case enumValuesTypeTag:
		if len(sourcePath) < i+2 {
			return nil, nil, newInvalidSourcePathError(sourcePath, "cannot have enum value declaration without index")
		}
		return enumValues, nil, nil
	case enumOptionTypeTag:
		// Return the entire path and then handle the option
		return options, []protoreflect.SourcePath{slicesext.Copy(sourcePath)}, nil
	case enumReservedRangeTypeTag:
		return reservedRanges, []protoreflect.SourcePath{currentPath(sourcePath, i)}, nil
	case enumReservedNameTypeTag:
		return reservedNames, []protoreflect.SourcePath{currentPath(sourcePath, i)}, nil
	}
	return nil, nil, newInvalidSourcePathError(sourcePath, "invalid enum path")
}

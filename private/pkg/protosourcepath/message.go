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
	messageNameTypeTag                 = int32(1)
	mesasgeFieldsTypeTag               = int32(2)
	nestedMessagesTypeTag              = int32(3)
	nestedEnumsTypeTag                 = int32(4)
	messageOneOfsTypeTag               = int32(8)
	messageOneOfNameTypeTag            = int32(1)
	messageOneOfOptionTypeTag          = int32(2)
	messageOptionTypeTag               = int32(7)
	messageExtensionsTypeTag           = int32(6)
	messageExtensionRangeTypeTag       = int32(5)
	messageExtensionRangeStartTypeTag  = int32(1)
	messageExtensionRangeEndTypeTag    = int32(2)
	messageExtensionRangeOptionTypeTag = int32(3)
	messageReservedRangeTypeTag        = int32(9)
	messageReservedNameTypeTag         = int32(10)
)

var (
	terminalOneOfTokens = []int32{
		messageOneOfNameTypeTag,
	}
	terminalExtensionRangeTokens = []int32{
		messageExtensionRangeStartTypeTag,
		messageExtensionRangeEndTypeTag,
	}
)

func messages(
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
			childAssociatedPath(sourcePath, i, messageNameTypeTag),
		)
	}
	if len(sourcePath) == i+1 {
		// This does not extend beyond the message declaration, return associated paths and
		// terminate here.
		return nil, associatedPaths, nil
	}
	return message, associatedPaths, nil
}

func message(token int32, sourcePath protoreflect.SourcePath, i int, _ bool) (state, []protoreflect.SourcePath, error) {
	switch token {
	case messageNameTypeTag:
		// The path for message name has already been added, can terminate here immediately.
		return nil, nil, nil
	case mesasgeFieldsTypeTag:
		if len(sourcePath) < i+2 {
			return nil, nil, newInvalidSourcePathError(sourcePath, "cannot have field declaration without index")
		}
		return fields, nil, nil
	case messageOneOfsTypeTag:
		if len(sourcePath) < i+2 {
			return nil, nil, newInvalidSourcePathError(sourcePath, "cannot have oneof declaration without index")
		}
		return oneOfs, nil, nil
	case nestedMessagesTypeTag:
		if len(sourcePath) < i+2 {
			return nil, nil, newInvalidSourcePathError(sourcePath, "cannot have a nested message declaration without index")
		}
		return messages, nil, nil
	case nestedEnumsTypeTag:
		if len(sourcePath) < i+2 {
			return nil, nil, newInvalidSourcePathError(sourcePath, "cannot have a nested enum declaration without index")
		}
		return enums, nil, nil
	case messageOptionTypeTag:
		// Return the entire path and then handle the option
		return options, []protoreflect.SourcePath{slicesext.Copy(sourcePath)}, nil
	case messageExtensionRangeTypeTag:
		return extensionRanges, []protoreflect.SourcePath{currentPath(sourcePath, i)}, nil
	case messageExtensionsTypeTag:
		return extensions, []protoreflect.SourcePath{currentPath(sourcePath, i)}, nil
	case messageReservedRangeTypeTag:
		return reservedRanges, []protoreflect.SourcePath{currentPath(sourcePath, i)}, nil
	case messageReservedNameTypeTag:
		return reservedNames, []protoreflect.SourcePath{currentPath(sourcePath, i)}, nil
	}
	return nil, nil, newInvalidSourcePathError(sourcePath, "invalid message path")
}

func oneOfs(
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
			childAssociatedPath(sourcePath, i, messageOneOfNameTypeTag),
		)
	}
	return oneOf, associatedPaths, nil
}

func oneOf(token int32, sourcePath protoreflect.SourcePath, i int, _ bool) (state, []protoreflect.SourcePath, error) {
	// TODO: use slices.Contains in the future
	if slicesext.ElementsContained(
		terminalOneOfTokens,
		[]int32{token},
	) {
		// Encountered a terminal one of token, can terminate here immediately.
		return nil, nil, nil
	}
	switch token {
	case messageOneOfOptionTypeTag:
		// Return the entire path and then handle the option
		return options, []protoreflect.SourcePath{slicesext.Copy(sourcePath)}, nil
	}
	return nil, nil, newInvalidSourcePathError(sourcePath, "invalid one of path")
}

func extensionRanges(
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
			childAssociatedPath(sourcePath, i, messageExtensionRangeStartTypeTag),
			childAssociatedPath(sourcePath, i, messageExtensionRangeEndTypeTag),
		)
	}
	if len(sourcePath) == i+1 {
		// This does not extend beyond the declaration, return associated paths and terminate here.
		return nil, associatedPaths, nil
	}
	return extensionRange, associatedPaths, nil
}

func extensionRange(token int32, sourcePath protoreflect.SourcePath, i int, _ bool) (state, []protoreflect.SourcePath, error) {
	// TODO: use slices.Contains in the future
	if slicesext.ElementsContained(
		terminalExtensionRangeTokens,
		[]int32{token},
	) {
		// Encountered a terminal extension range token, can terminate here immediately.
		return nil, nil, nil
	}
	switch token {
	case messageExtensionRangeOptionTypeTag:
		// Return the entire path and then handle the option
		return options, []protoreflect.SourcePath{slicesext.Copy(sourcePath)}, nil
	}
	return nil, nil, newInvalidSourcePathError(sourcePath, "invalid extension range path")
}

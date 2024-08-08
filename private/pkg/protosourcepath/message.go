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

func messages(_ int32, sourcePath protoreflect.SourcePath, i int) (state, []protoreflect.SourcePath, error) {
	// TODO(doria): should we handle the index?
	// Add message declaration and message name to aassociated paths
	associatedPaths := []protoreflect.SourcePath{
		currentPath(sourcePath, i),
		childAssociatedPath(sourcePath, i, messageNameTypeTag),
	}
	if len(sourcePath) == i+1 {
		// This does not extend beyond the declaration, return associated paths and terminate here.
		return nil, associatedPaths, nil
	}
	// Otherwise, move on to the message structure
	return message, associatedPaths, nil
}

func message(token int32, sourcePath protoreflect.SourcePath, i int) (state, []protoreflect.SourcePath, error) {
	switch token {
	case messageNameTypeTag:
		// This is the mesasge name, which is already added, can terminate here immediately.
		return nil, nil, nil
	case mesasgeFieldsTypeTag:
		if len(sourcePath) < i+2 {
			return nil, nil, newInvalidSourcePathError(sourcePath, "cannot have field declaraction without index")
		}
		return fields, nil, nil
	case messageOneOfsTypeTag:
		if len(sourcePath) < i+2 {
			return nil, nil, newInvalidSourcePathError(sourcePath, "cannot have one of declaration without index")
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
		if len(sourcePath) < i+2 {
			return nil, nil, newInvalidSourcePathError(sourcePath, "cannot have message option declaration without option number")
		}
		return options, nil, nil
	case messageExtensionRangeTypeTag:
		return extensionRanges, []protoreflect.SourcePath{currentPath(sourcePath, i)}, nil
	case messageExtensionsTypeTag:
		return extensions, []protoreflect.SourcePath{currentPath(sourcePath, i)}, nil
	case messageReservedRangeTypeTag:
		return reservedRanges, []protoreflect.SourcePath{currentPath(sourcePath, i)}, nil
	case messageReservedNameTypeTag:
		return reservedNames, []protoreflect.SourcePath{currentPath(sourcePath, i)}, nil
	}
	return nil, nil, newInvalidSourcePathError(sourcePath, "invalid source path")
}

func oneOfs(_ int32, sourcePath protoreflect.SourcePath, i int) (state, []protoreflect.SourcePath, error) {
	// TODO(doria): should we handle the index?
	associatedPaths := []protoreflect.SourcePath{
		currentPath(sourcePath, i),
		childAssociatedPath(sourcePath, i, messageOneOfNameTypeTag),
	}
	return oneOf, associatedPaths, nil
}

func oneOf(token int32, sourcePath protoreflect.SourcePath, i int) (state, []protoreflect.SourcePath, error) {
	if slices.Contains(terminalOneOfTokens, token) {
		// Encountered a terminal one of token validate the path and return here.
		if len(sourcePath) != i+1 {
			return nil, nil, newInvalidSourcePathError(sourcePath, "invalid one of path")
		}
		return nil, nil, nil
	}
	switch token {
	case messageOneOfOptionTypeTag:
		if len(sourcePath) < i+2 {
			return nil, nil, newInvalidSourcePathError(sourcePath, "cannot have one of option declaration without option number")
		}
		return options, nil, nil
	}
	return nil, nil, newInvalidSourcePathError(sourcePath, "invalid source path")
}

func extensionRanges(_ int32, sourcePath protoreflect.SourcePath, i int) (state, []protoreflect.SourcePath, error) {
	associatedPaths := []protoreflect.SourcePath{
		currentPath(sourcePath, i),
		childAssociatedPath(sourcePath, i, messageExtensionRangeStartTypeTag),
		childAssociatedPath(sourcePath, i, messageExtensionRangeEndTypeTag),
	}
	if len(sourcePath) == i+1 {
		// This does not extend beyond the declaration, return associated paths and terminate here.
		return nil, associatedPaths, nil
	}
	return extensionRange, associatedPaths, nil
}

func extensionRange(token int32, sourcePath protoreflect.SourcePath, i int) (state, []protoreflect.SourcePath, error) {
	if slices.Contains(terminalExtensionRangeTokens, token) {
		// Encountered a terminal extension range token validate the path and return here
		if len(sourcePath) != i+1 {
			return nil, nil, newInvalidSourcePathError(sourcePath, "invalid extension range path")
		}
		return nil, nil, nil
	}
	switch token {
	case messageExtensionRangeOptionTypeTag:
		if len(sourcePath) < i+2 {
			return nil, nil, newInvalidSourcePathError(sourcePath, "cannot have extension range option declaration without option number")
		}
		return options, nil, nil
	}
	return nil, nil, newInvalidSourcePathError(sourcePath, "invalid source path")
}

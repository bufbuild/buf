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
	messageNameTypeTag                 = int32(1)
	messageFieldsTypeTag               = int32(2)
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

// messages is the state when an element representing messages in the source path was parsed.
func messages(
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
			childAssociatedPath(fullSourcePath, index, messageNameTypeTag),
		)
	}
	if len(fullSourcePath) == index+1 {
		// This does not extend beyond the message declaration, return associated paths and
		// terminate here.
		return nil, associatedPaths, nil
	}
	return message, associatedPaths, nil
}

// message is the state when an element representing a specific child path of a message was parsed.
func message(token int32, fullSourcePath protoreflect.SourcePath, index int, _ bool) (state, []protoreflect.SourcePath, error) {
	switch token {
	case messageNameTypeTag:
		// The path for message name has already been added, can terminate here immediately.
		return nil, nil, nil
	case messageFieldsTypeTag:
		// We check to make sure that the length of the source path contains at least the current
		// token and an index. This is because all source paths for fields are expected
		// to have indices.
		if len(fullSourcePath) < index+2 {
			return nil, nil, newInvalidSourcePathError(fullSourcePath, "cannot have field declaration without index")
		}
		return fields, nil, nil
	case messageOneOfsTypeTag:
		// We check to make sure that the length of the source path contains at least the current
		// token and an index. This is because all source paths for oneofs are expected
		// to have indices.
		if len(fullSourcePath) < index+2 {
			return nil, nil, newInvalidSourcePathError(fullSourcePath, "cannot have oneof declaration without index")
		}
		return oneOfs, nil, nil
	case nestedMessagesTypeTag:
		// We check to make sure that the length of the source path contains at least the current
		// token and an index. This is because all source paths for nested messages are expected
		// to have indices.
		if len(fullSourcePath) < index+2 {
			return nil, nil, newInvalidSourcePathError(fullSourcePath, "cannot have a nested message declaration without index")
		}
		return messages, nil, nil
	case nestedEnumsTypeTag:
		// We check to make sure that the length of the source path contains at least the current
		// token and an index. This is because all source paths for nested enums are expected
		// to have indices.
		if len(fullSourcePath) < index+2 {
			return nil, nil, newInvalidSourcePathError(fullSourcePath, "cannot have a nested enum declaration without index")
		}
		return enums, nil, nil
	case messageOptionTypeTag:
		// For options, we add the full path and then return the options state to validate
		// the path.
		return options, []protoreflect.SourcePath{slices.Clone(fullSourcePath)}, nil
	case messageExtensionRangeTypeTag:
		// For extension ranges, we add the full path and then return the extension ranges state
		// to validate the path.
		return extensionRanges, []protoreflect.SourcePath{currentPath(fullSourcePath, index)}, nil
	case messageExtensionsTypeTag:
		// For extensions, we add the full path and then return the extensions state to
		// validate the path.
		return extensions, []protoreflect.SourcePath{currentPath(fullSourcePath, index)}, nil
	case messageReservedRangeTypeTag:
		// For reserved ranges, we add the full path and then return the reserved ranges state
		// to validate the path.
		return reservedRanges, []protoreflect.SourcePath{currentPath(fullSourcePath, index)}, nil
	case messageReservedNameTypeTag:
		// For reserved names, we add the full path and then return the reserved names state to
		// validate the path.
		return reservedNames, []protoreflect.SourcePath{currentPath(fullSourcePath, index)}, nil
	}
	return nil, nil, newInvalidSourcePathError(fullSourcePath, "invalid message path")
}

// oneOfs is the state when an element representing oneofs in the source path was parsed.
func oneOfs(
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
			childAssociatedPath(fullSourcePath, index, messageOneOfNameTypeTag),
		)
	}
	return oneOf, associatedPaths, nil
}

// oneOf is the state when an element representing a specific child path of a oneof was parsed.
func oneOf(token int32, fullSourcePath protoreflect.SourcePath, _ int, _ bool) (state, []protoreflect.SourcePath, error) {
	if slices.Contains(terminalOneOfTokens, token) {
		// Encountered a terminal one of token, can terminate here immediately.
		return nil, nil, nil
	}
	switch token {
	case messageOneOfOptionTypeTag:
		// For options, we add the full path and then return the options state to validate
		// the path.
		return options, []protoreflect.SourcePath{slices.Clone(fullSourcePath)}, nil
	}
	return nil, nil, newInvalidSourcePathError(fullSourcePath, "invalid one of path")
}

// extensionRanges is the state when an element representing extension ranges in the source path was parsed.
func extensionRanges(
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
			childAssociatedPath(fullSourcePath, index, messageExtensionRangeStartTypeTag),
			childAssociatedPath(fullSourcePath, index, messageExtensionRangeEndTypeTag),
		)
	}
	if len(fullSourcePath) == index+1 {
		// This does not extend beyond the declaration, return associated paths and terminate here.
		return nil, associatedPaths, nil
	}
	return extensionRange, associatedPaths, nil
}

// extensionRange is the state when an element representing a specific child path of an
// extension range was parsed.
func extensionRange(token int32, fullSourcePath protoreflect.SourcePath, _ int, _ bool) (state, []protoreflect.SourcePath, error) {
	if slices.Contains(terminalExtensionRangeTokens, token) {
		// Encountered a terminal extension range token, can terminate here immediately.
		return nil, nil, nil
	}
	switch token {
	case messageExtensionRangeOptionTypeTag:
		// For options, we add the full path and then return the options state to validate
		// the path.
		return options, []protoreflect.SourcePath{slices.Clone(fullSourcePath)}, nil
	}
	return nil, nil, newInvalidSourcePathError(fullSourcePath, "invalid extension range path")
}

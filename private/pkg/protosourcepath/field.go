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
	fieldNameTypeTag         = int32(1)
	fieldNumberTypeTag       = int32(3)
	fieldLabelTypeTag        = int32(4)
	fieldTypeTypeTag         = int32(5)
	fieldTypeNameTypeTag     = int32(6)
	fieldOptionTypeTag       = int32(8)
	extensionExtendeeTypeTag = int32(2)
	fieldDefaultValueTypeTag = int32(7)
)

var (
	terminalFieldTokens = []int32{
		fieldNameTypeTag,
		fieldNumberTypeTag,
		fieldLabelTypeTag,
		fieldTypeTypeTag,
		fieldTypeNameTypeTag,
		extensionExtendeeTypeTag,
	}
)

// fields is the state when an element representing fields in the source path was parsed.
func fields(
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
			childAssociatedPath(fullSourcePath, index, fieldNameTypeTag),
			childAssociatedPath(fullSourcePath, index, fieldNumberTypeTag),
			childAssociatedPath(fullSourcePath, index, fieldLabelTypeTag),
			childAssociatedPath(fullSourcePath, index, fieldTypeTypeTag),
			childAssociatedPath(fullSourcePath, index, fieldTypeNameTypeTag),
		)
	}
	if len(fullSourcePath) == index+1 {
		// This does not extend beyond the field declaration, return the associated paths and
		// terminate here.
		return nil, associatedPaths, nil
	}
	return field, associatedPaths, nil
}

// field is the state when an element representing a specific child path of a field was parsed.
func field(token int32, fullSourcePath protoreflect.SourcePath, index int, _ bool) (state, []protoreflect.SourcePath, error) {
	if slices.Contains(terminalFieldTokens, token) {
		// Encountered a terminal field token, can terminate here immediately.
		return nil, nil, nil
	}
	switch token {
	case fieldOptionTypeTag:
		// For options, we add the full path and then return the options state to validate
		// the path.
		return options, []protoreflect.SourcePath{slices.Clone(fullSourcePath)}, nil
	case fieldDefaultValueTypeTag:
		// Default value is a terminal path, but was not already added to our associated paths,
		// since default values are specific to proto2. Add the path and terminate.
		return nil, []protoreflect.SourcePath{currentPath(fullSourcePath, index)}, nil
	}
	return nil, nil, newInvalidSourcePathError(fullSourcePath, "invalid field path")
}

// extensions is the state when an element representing extensions in the source path was parsed.
func extensions(
	token int32,
	fullSourcePath protoreflect.SourcePath,
	index int,
	excludeChildAssociatedPaths bool,
) (state, []protoreflect.SourcePath, error) {
	// Extensions share the same descriptor proto definition as fields, so we can parse them
	// using the same states.
	field, associatedPaths, err := fields(token, fullSourcePath, index, excludeChildAssociatedPaths)
	if err != nil {
		return nil, nil, err
	}
	if !excludeChildAssociatedPaths {
		associatedPaths = append(
			associatedPaths,
			childAssociatedPath(fullSourcePath, index, extensionExtendeeTypeTag),
		)
	}
	return field, associatedPaths, nil
}

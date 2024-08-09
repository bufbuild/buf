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

func fields(
	_ int32,
	sourcePath protoreflect.SourcePath,
	i int,
	excludeChildAssociatedPaths bool,
) (state, []protoreflect.SourcePath, error) {
	// TODO(doria): should we handle the index?
	// Add current path, field name, number, label, type, type name to associated paths
	associatedPaths := []protoreflect.SourcePath{
		currentPath(sourcePath, i),
	}
	if !excludeChildAssociatedPaths {
		associatedPaths = append(
			associatedPaths,
			childAssociatedPath(sourcePath, i, fieldNameTypeTag),
			childAssociatedPath(sourcePath, i, fieldNumberTypeTag),
			childAssociatedPath(sourcePath, i, fieldLabelTypeTag),
			childAssociatedPath(sourcePath, i, fieldTypeTypeTag),
			childAssociatedPath(sourcePath, i, fieldTypeNameTypeTag),
		)
	}
	if len(sourcePath) == i+1 {
		// If this does not extend beyond the declaration, return the name, number, label, type, type_name
		// as associated paths and terminate here:
		return nil, associatedPaths, nil
	}
	// Otherwise, continue to the field structure
	return field, associatedPaths, nil
}

func field(token int32, sourcePath protoreflect.SourcePath, i int, _ bool) (state, []protoreflect.SourcePath, error) {
	// TODO: use slices.Contains in the future
	if slicesext.ElementsContained(
		terminalFieldTokens,
		[]int32{token},
	) {
		// Encountered a terminal field token, validate the path and return here.
		if len(sourcePath) != i+1 {
			return nil, nil, newInvalidSourcePathError(sourcePath, "invalid field path")
		}
		return nil, nil, nil
	}
	switch token {
	case fieldOptionTypeTag:
		if len(sourcePath) < i+2 {
			return nil, nil, newInvalidSourcePathError(sourcePath, "cannot have field option declaration without option number")
		}
		return options, nil, nil
	case fieldDefaultValueTypeTag:
		return nil, []protoreflect.SourcePath{currentPath(sourcePath, i)}, nil
	}
	return nil, nil, newInvalidSourcePathError(sourcePath, "invalid or unimplemented source path")
}

func extensions(
	token int32,
	sourcePath protoreflect.SourcePath,
	i int,
	excludeChildAssociatedPaths bool,
) (state, []protoreflect.SourcePath, error) {
	// An extension is effectively a field descriptor, so we start by getting all paths for fields.
	field, associatedPaths, err := fields(token, sourcePath, i, excludeChildAssociatedPaths)
	if err != nil {
		return nil, nil, err
	}
	if !excludeChildAssociatedPaths {
		associatedPaths = append(
			associatedPaths,
			childAssociatedPath(sourcePath, i, extensionExtendeeTypeTag),
		)
	}
	return field, associatedPaths, nil
}

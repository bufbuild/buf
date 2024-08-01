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
	fieldNameTypeTag     = int32(1)
	fieldNumberTypeTag   = int32(3)
	fieldLabelTypeTag    = int32(4)
	fieldTypeTypeTag     = int32(5)
	fieldTypeNameTypeTag = int32(6)
)

var (
	terminalFieldTokens = []int32{
		fieldNameTypeTag,
		fieldNumberTypeTag,
		fieldLabelTypeTag,
		fieldTypeTypeTag,
		fieldTypeNameTypeTag,
	}
)

func fields(_ int32, sourcePath protoreflect.SourcePath, i int) (state, []protoreflect.SourcePath, error) {
	// TODO(doria): should we handle the index?
	// Add current path, field name, number, label, type, type name to associated paths
	associatedPaths := []protoreflect.SourcePath{
		currentPath(sourcePath, i),
		childAssociatedPath(sourcePath, i, fieldNameTypeTag),
		childAssociatedPath(sourcePath, i, fieldNumberTypeTag),
		childAssociatedPath(sourcePath, i, fieldLabelTypeTag),
		childAssociatedPath(sourcePath, i, fieldTypeTypeTag),
		childAssociatedPath(sourcePath, i, fieldTypeNameTypeTag),
	}
	if len(sourcePath) == i+1 {
		// If this does not extend beyond the declaration, return the name, number, label, type, type_name
		// as associated paths and terminate here:
		return nil, associatedPaths, nil
	}
	// Otherwise, continue to the field structure
	return field, associatedPaths, nil
}

func field(token int32, sourcePath protoreflect.SourcePath, i int) (state, []protoreflect.SourcePath, error) {
	if slices.Contains(terminalFieldTokens, token) {
		// Encountered a terminal field token, terminate here.
		return nil, nil, nil
	}
	// TODO(doria): implement non-terminal field tokens
	return nil, nil, newInvalidSourcePathError(sourcePath, "invalid or unimplemented source path")
}

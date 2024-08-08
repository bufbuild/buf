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
	methodNameTypeTag            = int32(1)
	methodInputTypeTypeTag       = int32(2)
	methodOutputTypeTypeTag      = int32(3)
	methodClientStreamingTypeTag = int32(5)
	methodServerStreamingTypeTag = int32(6)
	methodOptionTypeTag          = int32(4)
)

var (
	terminalMethodTokens = []int32{
		methodNameTypeTag,
		methodInputTypeTypeTag,
		methodOutputTypeTypeTag,
		methodClientStreamingTypeTag,
		methodServerStreamingTypeTag,
	}
)

func methods(_ int32, sourcePath protoreflect.SourcePath, i int) (state, []protoreflect.SourcePath, error) {
	// TODO(doria): should we handle the index?
	// Add current path, method name, method input type, method output type, client streaming, and server
	// streaming as associated paths
	associatedPaths := []protoreflect.SourcePath{
		currentPath(sourcePath, i),
		childAssociatedPath(sourcePath, i, methodNameTypeTag),
		childAssociatedPath(sourcePath, i, methodInputTypeTypeTag),
		childAssociatedPath(sourcePath, i, methodOutputTypeTypeTag),
		childAssociatedPath(sourcePath, i, methodClientStreamingTypeTag),
		childAssociatedPath(sourcePath, i, methodServerStreamingTypeTag),
	}
	if len(sourcePath) == i+1 {
		// If this does not extend beyond the method declaration, return associated paths and
		// terminate.
		return nil, associatedPaths, nil
	}
	// Otherwise, continue to method structure
	return method, associatedPaths, nil
}

func method(token int32, sourcePath protoreflect.SourcePath, i int) (state, []protoreflect.SourcePath, error) {
	if slices.Contains(terminalMethodTokens, token) {
		// Encountered a terminal method token, validate and terminate here.
		if len(sourcePath) != i+1 {
			return nil, nil, newInvalidSourcePathError(sourcePath, "invalid method path")
		}
		return nil, nil, nil
	}
	switch token {
	case methodOptionTypeTag:
		if len(sourcePath) < i+2 {
			return nil, nil, newInvalidSourcePathError(sourcePath, "cannot have method option declaration without option number")
		}
		return options, nil, nil
	}
	// TODO(doria): implement non-terminal method tokens
	return nil, nil, newInvalidSourcePathError(sourcePath, "invalid or unimplemented source path")
}

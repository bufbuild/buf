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

// methods is the state when an element representing methods in the source path was parsed.
func methods(
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
			childAssociatedPath(fullSourcePath, index, methodNameTypeTag),
			childAssociatedPath(fullSourcePath, index, methodInputTypeTypeTag),
			childAssociatedPath(fullSourcePath, index, methodOutputTypeTypeTag),
			childAssociatedPath(fullSourcePath, index, methodClientStreamingTypeTag),
			childAssociatedPath(fullSourcePath, index, methodServerStreamingTypeTag),
		)
	}
	if len(fullSourcePath) == index+1 {
		// This does not extend beyond the method declaration, return associated paths and
		// terminate here.
		return nil, associatedPaths, nil
	}
	return method, associatedPaths, nil
}

// method is the state when an element representing a specific child path of a method was parsed.
func method(token int32, fullSourcePath protoreflect.SourcePath, _ int, _ bool) (state, []protoreflect.SourcePath, error) {
	if slices.Contains(terminalMethodTokens, token) {
		// Encountered a terminal method token, can terminate here immediately.
		return nil, nil, nil
	}
	switch token {
	case methodOptionTypeTag:
		// For options, we add the full path and then return the options state to validate
		// the path.
		return options, []protoreflect.SourcePath{slices.Clone(fullSourcePath)}, nil
	}
	return nil, nil, newInvalidSourcePathError(fullSourcePath, "invalid method path")
}

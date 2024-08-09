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

func methods(
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
			childAssociatedPath(sourcePath, i, methodNameTypeTag),
			childAssociatedPath(sourcePath, i, methodInputTypeTypeTag),
			childAssociatedPath(sourcePath, i, methodOutputTypeTypeTag),
			childAssociatedPath(sourcePath, i, methodClientStreamingTypeTag),
			childAssociatedPath(sourcePath, i, methodServerStreamingTypeTag),
		)
	}
	if len(sourcePath) == i+1 {
		// This does not extend beyond the method declaration, return associated paths and
		// terminate here.
		return nil, associatedPaths, nil
	}
	return method, associatedPaths, nil
}

func method(token int32, sourcePath protoreflect.SourcePath, i int, _ bool) (state, []protoreflect.SourcePath, error) {
	// TODO: use slices.Contains in the future
	if slicesext.ElementsContained(
		terminalMethodTokens,
		[]int32{token},
	) {
		// Encountered a terminal method token, can terminate here immediately.
		return nil, nil, nil
	}
	switch token {
	case methodOptionTypeTag:
		// Return the entire path and then handle the option
		return options, []protoreflect.SourcePath{slicesext.Copy(sourcePath)}, nil
	}
	return nil, nil, newInvalidSourcePathError(sourcePath, "invalid method path")
}

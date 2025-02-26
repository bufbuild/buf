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
	serviceNameTypeTag    = int32(1)
	serviceMethodsTypeTag = int32(2)
	serviceOptionTypeTag  = int32(3)
)

// services is the state when an element representing services in the source path was parsed.
func services(
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
			childAssociatedPath(fullSourcePath, index, serviceNameTypeTag),
		)
	}
	if len(fullSourcePath) == +1 {
		// This does not extend beyond the declaration, return associated paths and
		// terminate here.
		return nil, associatedPaths, nil
	}
	return service, associatedPaths, nil
}

// service is the state when an element representing a specific child path of a service was parsed.
func service(token int32, fullSourcePath protoreflect.SourcePath, index int, _ bool) (state, []protoreflect.SourcePath, error) {
	switch token {
	case serviceNameTypeTag:
		// The path for service name has already been added, can terminate here immediately.
		return nil, nil, nil
	case serviceMethodsTypeTag:
		// We check to make sure that the length of the source path contains at least the current
		// token and an index. This is because all source paths for methods are expected
		// to have indices.
		if len(fullSourcePath) < index+2 {
			return nil, nil, newInvalidSourcePathError(fullSourcePath, "cannot have method declaration without index")
		}
		return methods, nil, nil
	case serviceOptionTypeTag:
		// For options, we add the full path and then return the options state to validate
		// the path.
		return options, []protoreflect.SourcePath{slices.Clone(fullSourcePath)}, nil
	}
	return nil, nil, newInvalidSourcePathError(fullSourcePath, "invalid service path")
}

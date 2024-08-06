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
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	serviceNameTypeTag    = int32(1)
	serviceMethodsTypeTag = int32(2)
	serviceOptionTypeTag  = int32(3)
)

func services(_ int32, sourcePath protoreflect.SourcePath, i int) (state, []protoreflect.SourcePath, error) {
	// TODO(doria): should we handle the index?
	// Add service declaration and name to associated paths
	associatedPaths := []protoreflect.SourcePath{
		currentPath(sourcePath, i),
		childAssociatedPath(sourcePath, i, serviceNameTypeTag),
	}
	if len(sourcePath) == +1 {
		// This does not extend beyond the declaration, return associated paths and terminate here
		return nil, associatedPaths, nil
	}
	// Otherwsise, move on to the service structure
	return service, associatedPaths, nil
}

func service(token int32, sourcePath protoreflect.SourcePath, i int) (state, []protoreflect.SourcePath, error) {
	switch token {
	case serviceNameTypeTag:
		// This is the service name, which is already added, can terminate here immediately.
		return nil, nil, nil
	case serviceMethodsTypeTag:
		if len(sourcePath) < i+2 {
			return nil, nil, newInvalidSourcePathError(sourcePath, "cannot have method declaration without index")
		}
		return methods, nil, nil
	case serviceOptionTypeTag:
		if len(sourcePath) < i+2 {
			return nil, nil, newInvalidSourcePathError(sourcePath, "cannot have service option declaration without index")
		}
		return options, nil, nil
	}
	return nil, nil, newInvalidSourcePathError(sourcePath, "invalid or unimplemented source path")
}

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
	"fmt"

	"github.com/bufbuild/buf/private/pkg/slicesext"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	packageTypeTag      = int32(2)
	dependenciesTypeTag = int32(3)
	syntaxTypeTag       = int32(12)
	editionTypeTag      = int32(14)
	messagesTypeTag     = int32(4)
	enumsTypeTag        = int32(5)
	servicesTypeTag     = int32(6)
	fileOptionsTypeTag  = int32(8)
	extensionsTypeTag   = int32(7)
)

type AssociatedSourcePaths struct {
	ParentPaths  []protoreflect.SourcePath
	SiblingPaths []protoreflect.SourcePath
	ChildPaths   []protoreflect.SourcePath
}

// GetAssociatedSourcePaths...
//
// TODO(doria): write docs
func GetAssociatedSourcePaths(sourcePath protoreflect.SourcePath) ([]protoreflect.SourcePath, error) {
	var result []protoreflect.SourcePath
	currentState := start
	var associatedSourcePaths []protoreflect.SourcePath
	var err error
	for i, token := range sourcePath {
		if currentState == nil {
			return nil, fmt.Errorf("state has already been terminated before index %v for path %v", i, sourcePath)
		}
		currentState, associatedSourcePaths, err = currentState(token, sourcePath, i)
		if err != nil {
			return nil, err
		}
		if associatedSourcePaths != nil {
			result = append(result, associatedSourcePaths...)
		}
	}

	return result, nil
}

// *** PRIVATE ***

type state func(int32, protoreflect.SourcePath, int) (state, []protoreflect.SourcePath, error)

func start(token int32, sourcePath protoreflect.SourcePath, i int) (state, []protoreflect.SourcePath, error) {
	switch token {
	case packageTypeTag, syntaxTypeTag, editionTypeTag:
		// package, syntax, and edition are terminal paths
		return nil, []protoreflect.SourcePath{currentPath(sourcePath, i)}, nil
	case dependenciesTypeTag:
		if len(sourcePath) < i+2 {
			return nil, nil, newInvalidSourcePathError(sourcePath, "cannot have dependency declaration without index")
		}
		return dependencies, nil, nil
	case messagesTypeTag:
		if len(sourcePath) < i+2 {
			return nil, nil, newInvalidSourcePathError(sourcePath, "cannot have message declaration without index")
		}
		return messages, nil, nil
	case enumsTypeTag:
		if len(sourcePath) < i+2 {
			return nil, nil, newInvalidSourcePathError(sourcePath, "cannot have enum declaration without index")
		}
		return enums, nil, nil
	case servicesTypeTag:
		if len(sourcePath) < i+2 {
			return nil, nil, newInvalidSourcePathError(sourcePath, "cannot have service declaration without index")
		}
		return services, nil, nil
	case fileOptionsTypeTag:
		if len(sourcePath) < i+2 {
			return nil, nil, newInvalidSourcePathError(sourcePath, "cannot have file options declaration without option path")
		}
		return options, nil, nil
	}
	// TODO(doria): continuing implementing source paths
	return nil, nil, newInvalidSourcePathError(sourcePath, "invalid or unimplemented source path")
}

func dependencies(token int32, sourcePath protoreflect.SourcePath, i int) (state, []protoreflect.SourcePath, error) {
	// dependencies are a terminal path
	return nil, []protoreflect.SourcePath{currentPath(sourcePath, i)}, nil
}

// TODO(doria): make the error better
func newInvalidSourcePathError(sourcePath protoreflect.SourcePath, s string) error {
	return fmt.Errorf("%s: %v", s, sourcePath)
}

// childAssociatedPath makes a copy of the source path at the given index (inclusive)
// and appends a child path tag.
// This is a helper function, the caller is expected to manage providing an index within
// range.
func childAssociatedPath(sourcePath protoreflect.SourcePath, i int, tag int32) protoreflect.SourcePath {
	return append(slicesext.Copy(sourcePath)[:i+1], tag)
}

// currentPath makes a copy of the source path at the given index (inclusive).
// This is a helper function, the caller is expected to manage providing an index within
// range.
func currentPath(sourcePath protoreflect.SourcePath, i int) protoreflect.SourcePath {
	return slicesext.Copy(sourcePath)[:i+1]
}

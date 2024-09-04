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
	packageTypeTag             = int32(2)
	dependenciesTypeTag        = int32(3)
	syntaxTypeTag              = int32(12)
	editionTypeTag             = int32(14)
	messagesTypeTag            = int32(4)
	enumsTypeTag               = int32(5)
	servicesTypeTag            = int32(6)
	fileOptionsTypeTag         = int32(8)
	extensionsTypeTag          = int32(7)
	reservedRangeStartTypeTag  = int32(1)
	reservedRangeEndTypeTag    = int32(2)
	uninterpretedOptionTypeTag = int32(999)
)

var (
	terminalReservedRangeTokens = []int32{
		reservedRangeStartTypeTag,
		reservedRangeEndTypeTag,
	}
)

// GetAssociatedSourcePaths takes a protoreflect.SourcePath and the option to exclude child
// associated paths, and returns a list of associated paths, []protoreflect.SourcePath.
//
// We should expect at least one associated path for a valid path input.
//
// Excluding child associated paths will only return associated paths for complete/top-level
// declarations. For example,
//
// Input: [4, 0, 2, 0] (.message[0].field[0])
//
// excludeChildAssociatedPaths == false:
// Associated paths: [
//
//	[4, 0] (.message[0])
//	[4, 0, 1] (.message[0].name)
//	[4, 0, 2, 0] (.message[0].field[0])
//	[4, 0, 2, 0, 1] (.message[0].field[0].name)
//	[4, 0, 2, 0, 3] (.message[0].field[0].number)
//	[4, 0, 2, 0, 4] (.message[0].field[0].label)
//	[4, 0, 2, 0, 5] (.message[0].field[0].type)
//	[4, 0, 2, 0, 6] (.message[0].field[0].type_name)
//
// ]
//
// excludeChildAssociatedPaths == true:
// Associated paths: [
//
//	[4, 0] (.message[0])
//	[4, 0, 2, 0] (.message[0].field[0])
//
// ]
//
// More details are available with the README for this package.
func GetAssociatedSourcePaths(sourcePath protoreflect.SourcePath) ([]protoreflect.SourcePath, error) {
	return getAssociatedSourcePaths(sourcePath, true)
}

func getAssociatedSourcePaths(
	sourcePath protoreflect.SourcePath,
	excludeChildAssociatedPaths bool,
) ([]protoreflect.SourcePath, error) {
	var result []protoreflect.SourcePath
	currentState := start
	var associatedSourcePaths []protoreflect.SourcePath
	var err error
	for i, token := range sourcePath {
		if currentState == nil {
			// We returned an unexpected terminal state, this is considered an invalid source path.
			return nil, newInvalidSourcePathError(sourcePath, "unexpected termination, invalid source path")
		}
		currentState, associatedSourcePaths, err = currentState(token, sourcePath, i, excludeChildAssociatedPaths)
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

type state func(
	token int32,
	sourcePath protoreflect.SourcePath,
	index int,
	excludeChildAssociatedPaths bool,
) (state, []protoreflect.SourcePath, error)

func start(token int32, sourcePath protoreflect.SourcePath, i int, _ bool) (state, []protoreflect.SourcePath, error) {
	switch token {
	case packageTypeTag, syntaxTypeTag, editionTypeTag:
		// package, syntax, and edition are terminal paths, return the path and terminate here.
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
		// Return the entire path and then handle the option
		return options, []protoreflect.SourcePath{slicesext.Copy(sourcePath)}, nil
	case extensionsTypeTag:
		return extensions, []protoreflect.SourcePath{currentPath(sourcePath, i)}, nil
	}
	return nil, nil, newInvalidSourcePathError(sourcePath, "invalid or unimplemented source path")
}

func dependencies(token int32, sourcePath protoreflect.SourcePath, i int, _ bool) (state, []protoreflect.SourcePath, error) {
	// dependencies are a terminal path, retrun the path and terminate here.
	return nil, []protoreflect.SourcePath{currentPath(sourcePath, i)}, nil
}

func options(token int32, sourcePath protoreflect.SourcePath, i int, _ bool) (state, []protoreflect.SourcePath, error) {
	// The entire path has alreaduy been returned, we just need to handle the terminal state here
	if len(sourcePath) == i+1 {
		return nil, nil, nil
	}
	return options, nil, nil
}

func reservedRanges(
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
			childAssociatedPath(sourcePath, i, reservedRangeStartTypeTag),
			childAssociatedPath(sourcePath, i, reservedRangeEndTypeTag),
		)
	}
	return reservedRange, associatedPaths, nil
}

func reservedRange(token int32, sourcePath protoreflect.SourcePath, i int, _ bool) (state, []protoreflect.SourcePath, error) {
	// All reserved range paths are considered a terminal, so validate the path and terminate here.
	// TODO: use slices.Contains in the future
	if !slicesext.ElementsContained(
		terminalReservedRangeTokens,
		[]int32{token},
	) {
		return nil, nil, newInvalidSourcePathError(sourcePath, "invalid reserved range path")
	}
	return nil, nil, nil
}

func reservedNames(_ int32, sourcePath protoreflect.SourcePath, i int, _ bool) (state, []protoreflect.SourcePath, error) {
	associatedPaths := []protoreflect.SourcePath{
		currentPath(sourcePath, i),
	}
	// All reserved name paths are considered terminal, can terminate here immediately.
	return nil, associatedPaths, nil
}

func newInvalidSourcePathError(sourcePath protoreflect.SourcePath, s string) error {
	return fmt.Errorf("%s: %v", s, sourcePath)
}

// childAssociatedPath makes a copy of the source path at the given index (inclusive)
// and appends a child path tag.
// This is a helper function, the caller is expected to manage providing an index within range.
func childAssociatedPath(sourcePath protoreflect.SourcePath, i int, tag int32) protoreflect.SourcePath {
	return append(slicesext.Copy(sourcePath)[:i+1], tag)
}

// currentPath makes a copy of the source path at the given index (inclusive).
// This is a helper function, the caller is expected to manage providing an index within range.
func currentPath(sourcePath protoreflect.SourcePath, i int) protoreflect.SourcePath {
	return slicesext.Copy(sourcePath)[:i+1]
}

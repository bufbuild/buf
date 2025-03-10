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
	"fmt"
	"slices"

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

// GetAssociatedSourcePaths takes a protoreflect.SourcePath and returns a list of associated
// paths, []protoreflect.SourcePath.
//
// We should expect at least one associated path for a valid path input.
//
// More details on associated paths are available in the README.md.
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
			// We have not parsed the entire source path, but received a terminal state, this is
			// considered an invalid source path, return an error.
			return nil, newInvalidSourcePathError(sourcePath, "unexpected termination, invalid source path")
		}
		// Check the currentState and then set the next state.
		currentState, associatedSourcePaths, err = currentState(token, sourcePath, i, excludeChildAssociatedPaths)
		if err != nil {
			return nil, err
		}
		// Add all associated paths found to the result.
		if associatedSourcePaths != nil {
			result = append(result, associatedSourcePaths...)
		}
	}

	return result, nil
}

// *** PRIVATE ***

// state represents a single state in a deterministic finite automaton (DFA). A DFA is used
// to parse the given source path. Each element of the source path (token) is checked with a
// state, and each state either returns the state to check the next element or it terminates
// the DFA. When the DFA is terminated, we do not expect to have additional elements that need
// to be parsed and all associated paths found are turned.
type state func(
	// token is the element of the source path that is currently being checked.
	token int32,
	// fullSourcePath is the full source path being parsed. This is needed to construct associated
	// source paths based on the token being checked.
	fullSourcePath protoreflect.SourcePath,
	// index is the index of the token that we are currently checking on the source path.
	index int,
	// excludeChildAssociatedPaths, when set to true, will exclude child paths, which are not
	// complete Protobuf declarations, from the associated source paths returned.
	excludeChildAssociatedPaths bool,
) (state, []protoreflect.SourcePath, error)

// start is the starting state and is used to parse the first element of the source path.
// It returns the subsequent state based on the token that was parsed.
func start(token int32, fullSourcePath protoreflect.SourcePath, index int, _ bool) (state, []protoreflect.SourcePath, error) {
	switch token {
	case packageTypeTag, syntaxTypeTag, editionTypeTag:
		// package, syntax, and edition are terminal paths, return the path and terminate here.
		return nil, []protoreflect.SourcePath{currentPath(fullSourcePath, index)}, nil
	case dependenciesTypeTag:
		// We check to make sure that the length of the source path contains at least the current
		// token and an index. This is because all source paths for dependencies are expected
		// to have indices.
		if len(fullSourcePath) < index+2 {
			return nil, nil, newInvalidSourcePathError(fullSourcePath, "cannot have dependency declaration without index")
		}
		return dependencies, nil, nil
	case messagesTypeTag:
		// We check to make sure that the length of the source path contains at least the current
		// token and an index. This is because all source paths for messages are expected
		// to have indices.
		if len(fullSourcePath) < index+2 {
			return nil, nil, newInvalidSourcePathError(fullSourcePath, "cannot have message declaration without index")
		}
		return messages, nil, nil
	case enumsTypeTag:
		// We check to make sure that the length of the source path contains at least the current
		// token and an index. This is because all source paths for enums are expected
		// to have indices.
		if len(fullSourcePath) < index+2 {
			return nil, nil, newInvalidSourcePathError(fullSourcePath, "cannot have enum declaration without index")
		}
		return enums, nil, nil
	case servicesTypeTag:
		// We check to make sure that the length of the source path contains at least the current
		// token and an index. This is because all source paths for services are expected
		// to have indices.
		if len(fullSourcePath) < index+2 {
			return nil, nil, newInvalidSourcePathError(fullSourcePath, "cannot have service declaration without index")
		}
		return services, nil, nil
	case fileOptionsTypeTag:
		// For options, we add the full path and then return the options state to validate
		// the path.
		return options, []protoreflect.SourcePath{slices.Clone(fullSourcePath)}, nil
	case extensionsTypeTag:
		// For extensions, we add the full path and then return the extensions state to validate
		// the path.
		return extensions, []protoreflect.SourcePath{currentPath(fullSourcePath, index)}, nil
	}
	return nil, nil, newInvalidSourcePathError(fullSourcePath, "invalid source path")
}

// dependencies is the state when an element representing dependencies in the source path
// was parsed.
func dependencies(token int32, sourcePath protoreflect.SourcePath, index int, _ bool) (state, []protoreflect.SourcePath, error) {
	// Dependencies are considered a terminal path, we add the current path and then return.
	return nil, []protoreflect.SourcePath{currentPath(sourcePath, index)}, nil
}

// options is the state when an element representing options in the source path was parsed.
func options(token int32, fullSourcePath protoreflect.SourcePath, index int, _ bool) (state, []protoreflect.SourcePath, error) {
	// We already added the full options path, this is considered a terminal state without
	// additional information on the option for the source path.
	if len(fullSourcePath) == index+1 {
		return nil, nil, nil
	}
	// If there are additional path elements, we loop through them here.
	return options, nil, nil
}

// reservedRanges is the state when an element representing reserved ranges in the source
// path was parsed.
func reservedRanges(
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
			childAssociatedPath(fullSourcePath, index, reservedRangeStartTypeTag),
			childAssociatedPath(fullSourcePath, index, reservedRangeEndTypeTag),
		)
	}
	return reservedRange, associatedPaths, nil
}

// reservedRange is the state when an element representing a specific child path of a reserved
// range was parsed.
func reservedRange(token int32, fullSourcePath protoreflect.SourcePath, _ int, _ bool) (state, []protoreflect.SourcePath, error) {
	// Reserved ranges are considered a terminal path, we validate the token to ensure that it
	// is an expected element and return here.
	if !slices.Contains(terminalReservedRangeTokens, token) {
		return nil, nil, newInvalidSourcePathError(fullSourcePath, "invalid reserved range path")
	}
	return nil, nil, nil
}

// reservedNames is the state when an element representing reserved names in the source
// path was parsed.
func reservedNames(_ int32, fullSourcePath protoreflect.SourcePath, index int, _ bool) (state, []protoreflect.SourcePath, error) {
	associatedPaths := []protoreflect.SourcePath{
		currentPath(fullSourcePath, index),
	}
	// Reserved names are considered a terminal path, we can terminal immediately.
	return nil, associatedPaths, nil
}

func newInvalidSourcePathError(sourcePath protoreflect.SourcePath, s string) error {
	return fmt.Errorf("%s: %v", s, sourcePath)
}

// childAssociatedPath makes a copy of the source path at the given index (inclusive)
// and appends a child path tag.
// This is a helper function, the caller is expected to manage providing an index within range.
func childAssociatedPath(sourcePath protoreflect.SourcePath, i int, tag int32) protoreflect.SourcePath {
	return append(slices.Clone(sourcePath)[:i+1], tag)
}

// currentPath makes a copy of the source path at the given index (inclusive).
// This is a helper function, the caller is expected to manage providing an index within range.
func currentPath(sourcePath protoreflect.SourcePath, i int) protoreflect.SourcePath {
	return slices.Clone(sourcePath)[:i+1]
}

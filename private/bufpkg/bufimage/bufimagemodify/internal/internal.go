// Copyright 2020-2023 Buf Technologies, Inc.
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

package internal

import (
	"fmt"

	"google.golang.org/protobuf/types/descriptorpb"
)

// fileOptionPath is the path prefix used for FileOptions.
// All file option locations are preceded by a location
// with a path set to the fileOptionPath.
// https://github.com/protocolbuffers/protobuf/blob/053966b4959bdd21e4a24e657bcb97cb9de9e8a4/src/google/protobuf/descriptor.proto#L80
var fileOptionPath = []int32{8}

func RemoveLocationsFromSourceCodeInfo(sourceCodeInfo *descriptorpb.SourceCodeInfo, paths map[string]struct{}) error {
	// We can't just match on an exact path match because the target
	// file option's parent path elements would remain (i.e [8]).
	// Instead, we perform an initial pass to validate that the paths
	// are structured as expect, and collect all of the indices that
	// we need to delete.
	indices := make(map[int]struct{}, len(paths)*2)
	for i, location := range sourceCodeInfo.Location {
		if _, ok := paths[GetPathKey(location.Path)]; !ok {
			continue
		}
		if i == 0 {
			return fmt.Errorf("path %v must have a preceding parent path", location.Path)
		}
		if !Int32SliceIsEqual(sourceCodeInfo.Location[i-1].Path, fileOptionPath) {
			return fmt.Errorf("path %v must have a preceding parent path equal to %v", location.Path, fileOptionPath)
		}
		// Add the target path and its parent.
		indices[i-1] = struct{}{}
		indices[i] = struct{}{}
	}
	// Now that we know exactly which indices to exclude, we can
	// filter the SourceCodeInfo_Locations as needed.
	locations := make(
		[]*descriptorpb.SourceCodeInfo_Location,
		0,
		len(sourceCodeInfo.Location)-len(indices),
	)
	for i, location := range sourceCodeInfo.Location {
		if _, ok := indices[i]; ok {
			continue
		}
		locations = append(locations, location)
	}
	sourceCodeInfo.Location = locations
	return nil
}

// GetPathKey returns a unique key for the given path.
func GetPathKey(path []int32) string {
	key := make([]byte, len(path)*4)
	j := 0
	for _, elem := range path {
		key[j] = byte(elem)
		key[j+1] = byte(elem >> 8)
		key[j+2] = byte(elem >> 16)
		key[j+3] = byte(elem >> 24)
		j += 4
	}
	return string(key)
}

// Int32SliceIsEqual returns true if x and y contain the same elements.
func Int32SliceIsEqual(x []int32, y []int32) bool {
	if len(x) != len(y) {
		return false
	}
	for i, elem := range x {
		if elem != y[i] {
			return false
		}
	}
	return true
}

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

package internal

import (
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"google.golang.org/protobuf/types/descriptorpb"
)

type markSweeper struct {
	image bufimage.Image
	// Filepath -> SourceCodeInfo_Location.Path keys.
	sourceCodeInfoPaths map[string]map[string]struct{}
}

func newMarkSweeper(image bufimage.Image) *markSweeper {
	return &markSweeper{
		image:               image,
		sourceCodeInfoPaths: make(map[string]map[string]struct{}),
	}
}

func (s *markSweeper) Mark(imageFile bufimage.ImageFile, path []int32) {
	paths, ok := s.sourceCodeInfoPaths[imageFile.Path()]
	if !ok {
		paths = make(map[string]struct{})
		s.sourceCodeInfoPaths[imageFile.Path()] = paths
	}
	paths[getPathKey(path)] = struct{}{}
}

func (s *markSweeper) Sweep() error {
	for _, imageFile := range s.image.Files() {
		descriptor := imageFile.FileDescriptorProto()
		if descriptor.SourceCodeInfo == nil {
			continue
		}
		paths, ok := s.sourceCodeInfoPaths[imageFile.Path()]
		if !ok {
			continue
		}
		err := removeLocationsFromSourceCodeInfo(descriptor.SourceCodeInfo, paths)
		if err != nil {
			return err
		}
	}
	return nil
}

// removeLocationsFromSourceCodeInfo removes paths from the given sourceCodeInfo.
// Each path must be for either a file option or a field option.
func removeLocationsFromSourceCodeInfo(
	sourceCodeInfo *descriptorpb.SourceCodeInfo,
	pathsToRemove map[string]struct{},
) error {
	// TODO FUTURE: in v1 there is no need to check for field options, maybe v1 and v2
	// don't need to share this function.

	// We can't just match on an exact path match because the target
	// file option's parent path elements would remain (i.e [8]),
	// or the target field option's parent path has no other child left.
	// Instead, we perform an initial pass to validate that the paths
	// are structured as expected, and collect all of the indices that
	// we need to delete.
	indices := make(map[int]struct{}, len(pathsToRemove)*2)
	// each path in this trie is for a FieldOptions message (not for a singular option)
	var fieldOptionsPaths fieldOptionsTrie
	for i, location := range sourceCodeInfo.Location {
		path := location.Path
		pathType := getPathType(path)
		if pathType == pathTypeFieldOptionsRoot {
			fieldOptionsPaths.insert(path, i)
		}
		if _, ok := pathsToRemove[getPathKey(path)]; !ok {
			if pathType == pathTypeFieldOption {
				// This field option path will not be removed, register it to its
				// parent FieldOptions.
				fieldOptionsPaths.registerDescendant(path)
			}
			continue
		}
		if i == 0 {
			return fmt.Errorf("path %v must have a preceding parent path", location.Path)
		}
		if isPathForFileOption(location.Path) {
			if !slicesext.ElementsEqual(sourceCodeInfo.Location[i-1].Path, fileOptionPath) {
				return fmt.Errorf("file option path %v must have a preceding parent path equal to %v", location.Path, fileOptionPath)
			}
			// Add the target path and its parent.
			indices[i-1] = struct{}{}
			indices[i] = struct{}{}
			continue
		}
		if pathType == pathTypeFieldOption {
			// Note that there is a difference between the generated file option paths and field options paths.
			// For example, for:
			// ...
			// option java_package = "com.example";
			// option go_package = "github.com/hello/world";
			// ...
			// the generated paths are
			// [8], [8,1], [8], [8,11]
			// where each file option declared has a parent.
			// However, for different field options of the same field, they share the same parent. For
			// ...
			// optional string id2 = 2 [jstype = JS_STRING, ctype = CORD];
			// required fixed64 first = 1 [
			//   (foo.bar.baz.aaa).foo = "hello",
			//   (foo.bar.baz.bbb).a.foo = "hey",
			//   (foo.bar.baz.ccc) = 123, // ccc is a repeated option
			//   jstype = JS_STRING
			// ];
			// ...
			// the generated paths are
			// [4,0,2,0,8],[4,0,2,0,8,50000,1],[4,0,2,0,8,50002,1,1],[4,0,2,0,8,50003,0],[4,0,2,0,8,6]
			// where two field options share the same parent.
			// Therefore, do not remove the parent path yet.
			indices[i] = struct{}{}
			continue
		}
		return fmt.Errorf("path %v is neither a file option path nor a field option path", location.Path)
	}
	for _, emptyFieldOptions := range fieldOptionsPaths.indicesWithoutDescendant() {
		indices[emptyFieldOptions] = struct{}{}
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

func isPathForFileOption(path []int32) bool {
	// a file option's path is {8, x}
	fileOptionPathLen := 2
	return len(path) == fileOptionPathLen && path[0] == fileOptionPath[0]
}

// getPathKey returns a unique key for the given path.
func getPathKey(path []int32) string {
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

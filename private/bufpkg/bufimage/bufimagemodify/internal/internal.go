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

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/gen/data/datawkt"
	"google.golang.org/protobuf/types/descriptorpb"
)

var (
	// CCEnableArenas is the SourceCodeInfo path for the cc_enable_arenas option.
	// https://github.com/protocolbuffers/protobuf/blob/29152fbc064921ca982d64a3a9eae1daa8f979bb/src/google/protobuf/descriptor.proto#L420
	CCEnableArenasPath = []int32{8, 31}
	// CsharpNamespacePath is the SourceCodeInfo path for the csharp_namespace option.
	// https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L428
	CsharpNamespacePath = []int32{8, 37}
	// GoPackagePath is the SourceCodeInfo path for the go_package option.
	// https://github.com/protocolbuffers/protobuf/blob/ee04809540c098718121e092107fbc0abc231725/src/google/protobuf/descriptor.proto#L392
	GoPackagePath = []int32{8, 11}
	// JavaMultipleFilesPath is the SourceCodeInfo path for the java_multiple_files option.
	// https://github.com/protocolbuffers/protobuf/blob/ee04809540c098718121e092107fbc0abc231725/src/google/protobuf/descriptor.proto#L364
	JavaMultipleFilesPath = []int32{8, 10}
	// JavaOuterClassnamePath is the SourceCodeInfo path for the java_outer_classname option.
	// https://github.com/protocolbuffers/protobuf/blob/87d140f851131fb8a6e8a80449cf08e73e568259/src/google/protobuf/descriptor.proto#L356
	JavaOuterClassnamePath = []int32{8, 8}
	// JavaPackagePath is the SourceCodeInfo path for the java_package option.
	// https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L348
	JavaPackagePath = []int32{8, 1}
	// JavaStringCheckUtf8Path is the SourceCodeInfo path for the java_string_check_utf8 option.
	// https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L375
	JavaStringCheckUtf8Path = []int32{8, 27}
	// ObjcClassPrefixPath is the SourceCodeInfo path for the objc_class_prefix option.
	// https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L425
	ObjcClassPrefixPath = []int32{8, 36}
	// optimizeFor is the SourceCodeInfo path for the optimize_for option.
	// https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L385
	OptimizeForPath = []int32{8, 9}
	// PhpMetadataNamespacePath is the SourceCodeInfo path for the php_metadata_namespace option.
	// Ref: https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L448
	PhpMetadataNamespacePath = []int32{8, 44}
	// PhpNamespacePath is the SourceCodeInfo path for the php_namespace option.
	// Ref: https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L443
	PhpNamespacePath = []int32{8, 41}
	// RubyPackagePath is the SourceCodeInfo path for the ruby_package option.
	// https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L453
	RubyPackagePath = []int32{8, 45}
	// JSTypePackageSuffix is the SourceCodeInfo sub path for the jstype field option.
	// https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L567
	JSTypePackageSuffix = []int32{8, 6}
)

// https://github.com/protocolbuffers/protobuf/blob/29152fbc064921ca982d64a3a9eae1daa8f979bb/src/google/protobuf/descriptor.proto#L215
const tagForFieldOptionsInField = 8

// fileOptionPath is the path prefix used for FileOptions.
// All file option locations are preceded by a location
// with a path set to the fileOptionPath.
// https://github.com/protocolbuffers/protobuf/blob/053966b4959bdd21e4a24e657bcb97cb9de9e8a4/src/google/protobuf/descriptor.proto#L80
var fileOptionPath = []int32{8}

// RemoveLocationsFromSourceCodeInfo removes paths from the given sourceCodeInfo.
// Each path must be for either a file option or a field option.
func RemoveLocationsFromSourceCodeInfo(sourceCodeInfo *descriptorpb.SourceCodeInfo, paths map[string]struct{}) error {
	// We can't just match on an exact path match because the target
	// file option's parent path elements would remain (i.e [8]),
	// or the target field option's parent path has no other child left.
	// Instead, we perform an initial pass to validate that the paths
	// are structured as expected, and collect all of the indices that
	// we need to delete.
	indices := make(map[int]struct{}, len(paths)*2)
	for i, location := range sourceCodeInfo.Location {
		if _, ok := paths[GetPathKey(location.Path)]; !ok {
			continue
		}
		if i == 0 {
			return fmt.Errorf("path %v must have a preceding parent path", location.Path)
		}
		if isPathForFileOption(location.Path) {
			if !Int32SliceIsEqual(sourceCodeInfo.Location[i-1].Path, fileOptionPath) {
				return fmt.Errorf("file option path %v must have a preceding parent path equal to %v", location.Path, fileOptionPath)
			}
			// Add the target path and its parent.
			indices[i-1] = struct{}{}
			indices[i] = struct{}{}
			continue
		}
		if isPathMaybeForFieldOption(location.Path) {
			// Now the path must be for a field option
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
			// ...
			// the generated paths are
			// [4,0,2,1,8], [4,0,2,1,8,6], [4,0,2,1,8,1]
			// where two field options share the same parent.
			// Therefore, do not remove the parent path.
			indices[i] = struct{}{}
			// TODO: remove the parent path when this field option is the only child.
			continue
		}
		return fmt.Errorf("path %v is neither a file option path nor a field option path", location.Path)
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

// isPathMaybeForFieldOption is a best-effort guess on
// whether the path looks like a field option.
func isPathMaybeForFieldOption(path []int32) bool {
	// a field option's path is {4, messageIndex, ..., 2, fieldIndex, 8, optionTag}
	// or {7, extensionIndex, 8, optionTag}
	minFieldOptionpathLen := 4
	return len(path) >= minFieldOptionpathLen && path[len(path)-2] == tagForFieldOptionsInField
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

// IsWellKnownType returns true if the given path is one of the well-known types.
func IsWellKnownType(imageFile bufimage.ImageFile) bool {
	return datawkt.Exists(imageFile.Path())
}

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

package bufimagegenerate

import (
	"fmt"

	"github.com/bufbuild/buf/private/gen/data/datawkt"
	"github.com/bufbuild/protoplugin/protopluginutil"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// ImagesToCodeGeneratorRequests converts the Images to CodeGeneratorRequests.
//
// All non-imports are added as files to generate.
// If includeImports is set, all non-well-known-type imports are also added as files to generate.
// If includeImports is set, only one CodeGeneratorRequest will contain any given file as a FileToGenerate.
// If includeWellKnownTypes is set, well-known-type imports are also added as files to generate.
// includeWellKnownTypes has no effect if includeImports is not set.
func ImagesToCodeGeneratorRequests(
	images []ImageForGeneration,
	parameter string,
	compilerVersion *pluginpb.Version,
	includeImports bool,
	includeWellKnownTypes bool,
) ([]*pluginpb.CodeGeneratorRequest, error) {
	requests := make([]*pluginpb.CodeGeneratorRequest, 0, len(images))
	// alreadyUsedPaths is a map of paths that have already been added to an image.
	//
	// We track this if includeImports is set, so that when we find an import, we can
	// see if the import was already added to a CodeGeneratorRequest via another Image
	// in the Image slice. If the import was already added, we do not add duplicates
	// across CodeGeneratorRequests.
	var alreadyUsedPaths map[string]struct{}
	// nonImportPaths is a map of non-import paths.
	//
	// We track this if includeImports is set. If we find a non-import file in Image A
	// and this file is an import in Image B, the file will have already been added to
	// a CodeGeneratorRequest via Image A, so do not add the duplicate to any other
	// CodeGeneratorRequest.
	var nonImportPaths map[string]struct{}
	if includeImports {
		// We don't need to track these if includeImports is false, so we only populate
		// the maps if includeImports is true. If includeImports is false, only non-imports
		// will be added to each CodeGeneratorRequest, so figuring out whether or not
		// we should add a given import to a given CodeGeneratorRequest is unnecessary.
		//
		// imageToCodeGeneratorRequest checks if these maps are nil before every access.
		alreadyUsedPaths = make(map[string]struct{})
		nonImportPaths = make(map[string]struct{})
		for _, image := range images {
			for _, imageFile := range image.files() {
				if !imageFile.IsImport() {
					nonImportPaths[imageFile.Path()] = struct{}{}
				}
			}
		}
	}
	for _, image := range images {
		var err error
		request, err := imageToCodeGeneratorRequest(
			image,
			parameter,
			compilerVersion,
			includeImports,
			includeWellKnownTypes,
			alreadyUsedPaths,
			nonImportPaths,
		)
		if err != nil {
			return nil, err
		}
		if len(request.FileToGenerate) == 0 {
			continue
		}
		requests = append(requests, request)
	}
	return requests, nil
}

// ImageToCodeGeneratorRequest returns a new CodeGeneratorRequest for the Image.
//
// All non-imports are added as files to generate.
// If includeImports is set, all non-well-known-type imports are also added as files to generate.
// If includeWellKnownTypes is set, well-known-type imports are also added as files to generate.
// includeWellKnownTypes has no effect if includeImports is not set.
func ImageToCodeGeneratorRequest(
	image ImageForGeneration,
	parameter string,
	compilerVersion *pluginpb.Version,
	includeImports bool,
	includeWellKnownTypes bool,
) (*pluginpb.CodeGeneratorRequest, error) {
	return imageToCodeGeneratorRequest(
		image,
		parameter,
		compilerVersion,
		includeImports,
		includeWellKnownTypes,
		nil,
		nil,
	)
}

// *** PRIVATE ***

func imageToCodeGeneratorRequest(
	image ImageForGeneration,
	parameter string,
	compilerVersion *pluginpb.Version,
	includeImports bool,
	includeWellKnownTypes bool,
	alreadyUsedPaths map[string]struct{},
	nonImportPaths map[string]struct{},
) (*pluginpb.CodeGeneratorRequest, error) {
	imageFiles := image.files()
	request := &pluginpb.CodeGeneratorRequest{
		ProtoFile:       make([]*descriptorpb.FileDescriptorProto, len(imageFiles)),
		CompilerVersion: compilerVersion,
	}
	if parameter != "" {
		request.Parameter = proto.String(parameter)
	}
	for i, imageFile := range imageFiles {
		fileDescriptorProto := imageFile.FileDescriptorProto()
		// ProtoFile should include only runtime-retained options for files to generate.
		if isFileToGenerate(
			imageFile,
			alreadyUsedPaths,
			nonImportPaths,
			includeImports,
			includeWellKnownTypes,
		) {
			request.FileToGenerate = append(request.FileToGenerate, imageFile.Path())
			// Source-retention options for items in FileToGenerate are provided in SourceFileDescriptors.
			request.SourceFileDescriptors = append(request.SourceFileDescriptors, fileDescriptorProto)
			// And the corresponding descriptor in ProtoFile will have source-retention options stripped.
			var err error
			fileDescriptorProto, err = protopluginutil.StripSourceRetentionOptions(fileDescriptorProto)
			if err != nil {
				return nil, fmt.Errorf("failed to strip source-retention options for file %q when constructing a CodeGeneratorRequest: %w", imageFile.Path(), err)
			}
		}
		request.ProtoFile[i] = fileDescriptorProto
	}
	return request, nil
}

func isFileToGenerate(
	imageFile imageFileForGeneration,
	alreadyUsedPaths map[string]struct{},
	nonImportPaths map[string]struct{},
	includeImports bool,
	includeWellKnownTypes bool,
) bool {
	if !imageFile.toGenerate {
		return false
	}
	path := imageFile.Path()
	if !imageFile.IsImport() {
		if alreadyUsedPaths != nil {
			// set as already used
			alreadyUsedPaths[path] = struct{}{}
		}
		// this is a non-import in this image, we always want to generate
		return true
	}
	if !includeImports {
		// we don't want to include imports
		return false
	}
	if !includeWellKnownTypes && datawkt.Exists(path) {
		// we don't want to generate wkt even if includeImports is set unless
		// includeWellKnownTypes is set
		return false
	}
	if alreadyUsedPaths != nil {
		if _, ok := alreadyUsedPaths[path]; ok {
			// this was already added for generate to another image
			return false
		}
	}
	if nonImportPaths != nil {
		if _, ok := nonImportPaths[path]; ok {
			// this is a non-import in another image so it will be generated
			// from another image
			return false
		}
	}
	// includeImports is set, this isn't a wkt, and it won't be generated in another image
	if alreadyUsedPaths != nil {
		// set as already used
		alreadyUsedPaths[path] = struct{}{}
	}
	return true
}

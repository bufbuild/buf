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

package bufimage

import (
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/gen/data/datawkt"
	imagev1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/image/v1"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/bufbuild/protoplugin/protopluginutil"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// Must match the tag number for ImageFile.buf_extensions defined in proto/buf/alpha/image/v1/image.proto.
const bufExtensionFieldNumber = 8042

// paths can be either files (ending in .proto) or directories
// paths must be normalized and validated, and not duplicated
// if a directory, all .proto files underneath will be included
func imageWithOnlyPaths(image Image, fileOrDirPaths []string, excludeFileOrDirPaths []string, allowNotExist bool) (Image, error) {
	if err := normalpath.ValidatePathsNormalizedValidatedUnique(fileOrDirPaths); err != nil {
		return nil, err
	}
	if err := normalpath.ValidatePathsNormalizedValidatedUnique(excludeFileOrDirPaths); err != nil {
		return nil, err
	}
	excludeFileOrDirPathMap := slicesext.ToStructMap(excludeFileOrDirPaths)
	// These are the files that fileOrDirPaths actually reference and will
	// result in the non-imports in our resulting Image. The Image will also include
	// the ImageFiles that the nonImportImageFiles import
	nonImportPaths := make(map[string]struct{})
	var nonImportImageFiles []ImageFile
	// We have only exclude paths, and therefore all other paths are target paths.
	if len(fileOrDirPaths) == 0 && len(excludeFileOrDirPaths) > 0 {
		for _, imageFile := range image.Files() {
			if !imageFile.IsImport() {
				if !normalpath.MapHasEqualOrContainingPath(excludeFileOrDirPathMap, imageFile.Path(), normalpath.Relative) {
					nonImportPaths[imageFile.Path()] = struct{}{}
					nonImportImageFiles = append(nonImportImageFiles, imageFile)
				}
			}
		}
		// Finally, before we construct the image, we need to validate that all exclude paths
		// provided adhere to the allowNotExist flag.
		if !allowNotExist {
			if err := checkExcludePathsExistInImage(image, excludeFileOrDirPaths); err != nil {
				return nil, err
			}
		}
		return getImageWithImports(image, nonImportPaths, nonImportImageFiles)
	}
	// We do a check here to ensure that no paths are duplicated as a target and an exclude.
	for _, fileOrDirPath := range fileOrDirPaths {
		if _, ok := excludeFileOrDirPathMap[fileOrDirPath]; ok {
			return nil, fmt.Errorf(
				"cannot set the same path for both --path and --exclude-path flags: %s",
				normalpath.Unnormalize(fileOrDirPath),
			)
		}
	}
	// potentialDirPaths are paths that we need to check if they are directories
	// these are any files that do not end in .proto, as well as files that
	// end in .proto but do not have a corresponding ImageFile - if there
	// is not an ImageFile, the path ending in .proto could be a directory
	// that itself contains ImageFiles, i.e. a/b.proto/c.proto is valid if not dumb
	var potentialDirPaths []string
	for _, fileOrDirPath := range fileOrDirPaths {
		// this is not allowed, this is the equivalent of a root
		if fileOrDirPath == "." {
			return nil, errors.New(`"." is not a valid path value`)
		}
		if normalpath.Ext(fileOrDirPath) != ".proto" {
			// not a .proto file, therefore must be a directory
			potentialDirPaths = append(potentialDirPaths, fileOrDirPath)
		} else {
			if imageFile := image.GetFile(fileOrDirPath); imageFile != nil {
				// We do not need to check excludes here, since we already checked for duplicated
				// paths, and target files that resolve to a specific image file are always a leaf,
				// thus, we would always include it if it's specified.
				// We have an ImageFile, therefore the fileOrDirPath was a file path
				// add to the nonImportImageFiles if does not already exist
				if _, ok := nonImportPaths[fileOrDirPath]; !ok {
					nonImportPaths[fileOrDirPath] = struct{}{}
					nonImportImageFiles = append(nonImportImageFiles, imageFile)
				}
			} else {
				// we do not have an image file, so even though this path ends
				// in .proto,  this could be a directory - we need to check it
				potentialDirPaths = append(potentialDirPaths, fileOrDirPath)
			}
		}
	}
	if len(potentialDirPaths) == 0 {
		// We had no potential directory paths as we were able to get
		// an ImageFile for all fileOrDirPaths, so we can return an Image now.
		// This means we do not have to do the expensive O(image.Files()) operation
		// to check to see if each file is within a potential directory path.
		//
		// We do not need to check the excluded paths for the allowNotExist flag because all target
		// paths were image files, therefore the exclude paths would not apply in this case.
		//
		// Unfortunately, we need to do the expensive operation of checking to make sure the exclude
		// paths exist in the case where `allowNotExist == false`.
		if !allowNotExist {
			if err := checkExcludePathsExistInImage(image, excludeFileOrDirPaths); err != nil {
				return nil, err
			}
		}
		return getImageWithImports(image, nonImportPaths, nonImportImageFiles)
	}
	// we have potential directory paths, do the expensive operation
	// make a map of the directory paths
	// note that we do not make this a map to begin with as maps are unordered,
	// and we want to make sure we iterate over the paths in a deterministic order
	potentialDirPathMap := slicesext.ToStructMap(potentialDirPaths)

	// map of all paths based on the imageFiles
	// the map of paths within potentialDirPath that matches a file in image.Files()
	// this needs to contain all paths in potentialDirPathMap at the end for us to
	// have had matches for every inputted fileOrDirPath
	matchingPotentialDirPathMap := make(map[string]struct{})
	// the same thing is done for exclude paths
	matchingPotentialExcludePathMap := make(map[string]struct{})
	for _, imageFile := range image.Files() {
		imageFilePath := imageFile.Path()
		fileMatchingExcludePathMap := normalpath.MapAllEqualOrContainingPathMap(
			excludeFileOrDirPathMap,
			imageFilePath,
			normalpath.Relative,
		)
		if len(fileMatchingExcludePathMap) > 0 {
			for key := range fileMatchingExcludePathMap {
				matchingPotentialExcludePathMap[key] = struct{}{}
			}
		}
		// get the paths in potentialDirPathMap that match this imageFilePath
		fileMatchingPathMap := normalpath.MapAllEqualOrContainingPathMap(
			potentialDirPathMap,
			imageFilePath,
			normalpath.Relative,
		)
		if shouldExcludeFile(fileMatchingPathMap, fileMatchingExcludePathMap) {
			continue
		}
		if len(fileMatchingPathMap) > 0 {
			// we had a match, this means that some path in potentialDirPaths matched
			// the imageFilePath, add all the paths in potentialDirPathMap that
			// matched to matchingPotentialDirPathMap
			for key := range fileMatchingPathMap {
				matchingPotentialDirPathMap[key] = struct{}{}
			}
			// then, add the file to non-imports if it is not added
			if _, ok := nonImportPaths[imageFilePath]; !ok {
				nonImportPaths[imageFilePath] = struct{}{}
				nonImportImageFiles = append(nonImportImageFiles, imageFile)
			}
		}
	}
	// if !allowNotExist, i.e. if all fileOrDirPaths must have a matching ImageFile,
	// we check the matchingPotentialDirPathMap against the potentialDirPathMap
	// to make sure that potentialDirPathMap is covered
	if !allowNotExist {
		for potentialDirPath := range potentialDirPathMap {
			if _, ok := matchingPotentialDirPathMap[potentialDirPath]; !ok {
				// no match, this is an error given that allowNotExist is false
				return nil, fmt.Errorf("path %q has no matching file in the image", potentialDirPath)
			}
		}
		for excludeFileOrDirPath := range excludeFileOrDirPathMap {
			if _, ok := matchingPotentialExcludePathMap[excludeFileOrDirPath]; !ok {
				// no match, this is an error given that allowNotExist is false
				return nil, fmt.Errorf("path %q has no matching file in the image", excludeFileOrDirPath)
			}
		}
	}
	// we finally have all files that match fileOrDirPath that we can find, make the image
	return getImageWithImports(image, nonImportPaths, nonImportImageFiles)
}

// shouldExcludeFile takes the map of all the matching target paths and the map of all the matching
// exclude paths for an image file and takes the union of the two sets of matches to return
// a bool on whether or not we should exclude the file from the image.
func shouldExcludeFile(
	fileMatchingPathMap map[string]struct{},
	fileMatchingExcludePathMap map[string]struct{},
) bool {
	for fileMatchingPath := range fileMatchingPathMap {
		for fileMatchingExcludePath := range fileMatchingExcludePathMap {
			if normalpath.EqualsOrContainsPath(fileMatchingPath, fileMatchingExcludePath, normalpath.Relative) {
				delete(fileMatchingPathMap, fileMatchingPath)
				continue
			}
		}
	}
	// If there are no potential paths remaining,
	// then the file should be excluded.
	return len(fileMatchingPathMap) == 0
}

func getImageWithImports(
	image Image,
	nonImportPaths map[string]struct{},
	nonImportImageFiles []ImageFile,
) (Image, error) {
	var imageFiles []ImageFile
	seenPaths := make(map[string]struct{})
	for _, nonImportImageFile := range nonImportImageFiles {
		imageFiles = addFileWithImports(
			imageFiles,
			image,
			nonImportPaths,
			seenPaths,
			nonImportImageFile,
		)
	}
	return NewImage(imageFiles)
}

// returns accumulated files in correct order
func addFileWithImports(
	accumulator []ImageFile,
	image Image,
	nonImportPaths map[string]struct{},
	seenPaths map[string]struct{},
	imageFile ImageFile,
) []ImageFile {
	path := imageFile.Path()
	// if seen already, skip
	if _, ok := seenPaths[path]; ok {
		return accumulator
	}
	seenPaths[path] = struct{}{}

	// then, add imports first, for proper ordering
	for _, importPath := range imageFile.FileDescriptorProto().GetDependency() {
		if importFile := image.GetFile(importPath); importFile != nil {
			accumulator = addFileWithImports(
				accumulator,
				image,
				nonImportPaths,
				seenPaths,
				importFile,
			)
		}
	}

	// finally, add this file
	// check if this is an import or not
	_, isNotImport := nonImportPaths[path]
	accumulator = append(
		accumulator,
		ImageFileWithIsImport(imageFile, !isNotImport),
	)
	return accumulator
}

func checkExcludePathsExistInImage(image Image, excludeFileOrDirPaths []string) error {
	for _, excludeFileOrDirPath := range excludeFileOrDirPaths {
		var foundPath bool
		for _, imageFile := range image.Files() {
			if normalpath.EqualsOrContainsPath(excludeFileOrDirPath, imageFile.Path(), normalpath.Relative) {
				foundPath = true
				break
			}
		}
		if !foundPath {
			// no match, this is an error given that allowNotExist is false
			return fmt.Errorf("path %q has no matching file in the image", excludeFileOrDirPath)
		}
	}
	return nil
}

func imageFilesToFileDescriptorProtos(imageFiles []ImageFile) []*descriptorpb.FileDescriptorProto {
	fileDescriptorProtos := make([]*descriptorpb.FileDescriptorProto, len(imageFiles))
	for i, imageFile := range imageFiles {
		fileDescriptorProtos[i] = imageFile.FileDescriptorProto()
	}
	return fileDescriptorProtos
}

func imageFileToProtoImageFile(imageFile ImageFile) (*imagev1.ImageFile, error) {
	var protoCommitID string
	if !imageFile.CommitID().IsNil() {
		protoCommitID = uuidutil.ToDashless(imageFile.CommitID())
	}
	return fileDescriptorProtoToProtoImageFile(
		imageFile.FileDescriptorProto(),
		imageFile.IsImport(),
		imageFile.IsSyntaxUnspecified(),
		imageFile.UnusedDependencyIndexes(),
		imageFile.ModuleFullName(),
		protoCommitID,
	), nil
}

func fileDescriptorProtoToProtoImageFile(
	fileDescriptorProto *descriptorpb.FileDescriptorProto,
	isImport bool,
	isSyntaxUnspecified bool,
	unusedDependencyIndexes []int32,
	moduleFullName bufmodule.ModuleFullName,
	// Dashless
	moduleProtoCommitID string,
) *imagev1.ImageFile {
	var protoModuleInfo *imagev1.ModuleInfo
	if moduleFullName != nil {
		protoModuleInfo = &imagev1.ModuleInfo{
			Name: &imagev1.ModuleName{
				Remote:     proto.String(moduleFullName.Registry()),
				Owner:      proto.String(moduleFullName.Owner()),
				Repository: proto.String(moduleFullName.Name()),
			},
		}
		if moduleProtoCommitID != "" {
			protoModuleInfo.Commit = proto.String(moduleProtoCommitID)
		}
	}
	if len(unusedDependencyIndexes) == 0 {
		unusedDependencyIndexes = nil
	}
	resultFile := &imagev1.ImageFile{
		Name:             fileDescriptorProto.Name,
		Package:          fileDescriptorProto.Package,
		Syntax:           fileDescriptorProto.Syntax,
		Dependency:       fileDescriptorProto.GetDependency(),
		PublicDependency: fileDescriptorProto.GetPublicDependency(),
		WeakDependency:   fileDescriptorProto.GetWeakDependency(),
		MessageType:      fileDescriptorProto.GetMessageType(),
		EnumType:         fileDescriptorProto.GetEnumType(),
		Service:          fileDescriptorProto.GetService(),
		Extension:        fileDescriptorProto.GetExtension(),
		Options:          fileDescriptorProto.GetOptions(),
		SourceCodeInfo:   fileDescriptorProto.GetSourceCodeInfo(),
		Edition:          fileDescriptorProto.Edition,
		BufExtension: &imagev1.ImageFileExtension{
			// we might actually want to differentiate between unset and false
			IsImport: proto.Bool(isImport),
			// we might actually want to differentiate between unset and false
			IsSyntaxUnspecified: proto.Bool(isSyntaxUnspecified),
			UnusedDependency:    unusedDependencyIndexes,
			ModuleInfo:          protoModuleInfo,
		},
	}
	resultFile.ProtoReflect().SetUnknown(stripBufExtensionField(fileDescriptorProto.ProtoReflect().GetUnknown()))
	return resultFile
}

func stripBufExtensionField(unknownFields protoreflect.RawFields) protoreflect.RawFields {
	// We accumulate the new bytes in result. However, for efficiency, we don't do any
	// allocation/copying until we have to (i.e. until we actually see the field we're
	// trying to strip). So result will be left nil and initialized lazily if-and-only-if
	// we actually need to strip data from unknownFields.
	var result protoreflect.RawFields
	bytesRemaining := unknownFields
	for len(bytesRemaining) > 0 {
		num, wireType, n := protowire.ConsumeTag(bytesRemaining)
		if n < 0 {
			// shouldn't be possible unless explicitly set to invalid bytes via reflection
			return unknownFields
		}
		var skip bool
		if num == bufExtensionFieldNumber {
			// We need to strip this field.
			skip = true
			if result == nil {
				// Lazily initialize result to the preface that we've already examined.
				result = append(
					make(protoreflect.RawFields, 0, len(unknownFields)),
					unknownFields[:len(unknownFields)-len(bytesRemaining)]...,
				)
			}
		} else if result != nil {
			// accumulate data in result as we go
			result = append(result, bytesRemaining[:n]...)
		}
		bytesRemaining = bytesRemaining[n:]
		n = protowire.ConsumeFieldValue(num, wireType, bytesRemaining)
		if n < 0 {
			return unknownFields
		}
		if !skip && result != nil {
			result = append(result, bytesRemaining[:n]...)
		}
		bytesRemaining = bytesRemaining[n:]
	}
	if result == nil {
		// we did not have to remove anything
		return unknownFields
	}
	return result
}

func imageToCodeGeneratorRequest(
	image Image,
	parameter string,
	compilerVersion *pluginpb.Version,
	includeImports bool,
	includeWellKnownTypes bool,
	alreadyUsedPaths map[string]struct{},
	nonImportPaths map[string]struct{},
) (*pluginpb.CodeGeneratorRequest, error) {
	imageFiles := image.Files()
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
	imageFile ImageFile,
	alreadyUsedPaths map[string]struct{},
	nonImportPaths map[string]struct{},
	includeImports bool,
	includeWellKnownTypes bool,
) bool {
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

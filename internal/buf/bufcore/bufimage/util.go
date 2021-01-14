// Copyright 2020-2021 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/internal/buf/bufcore/internal/bufcorevalidate"
	imagev1 "github.com/bufbuild/buf/internal/gen/proto/go/buf/alpha/image/v1"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
)

func getImportFileIndexes(protoImage *imagev1.Image) (map[int]struct{}, error) {
	imageImportRefs := protoImage.GetBufbuildImageExtension().GetImageImportRefs()
	importFileIndexes := make(map[int]struct{}, len(imageImportRefs))
	for _, imageImportRef := range imageImportRefs {
		if imageImportRef.FileIndex == nil {
			// this should have been caught in validation but just in case
			return nil, errors.New("nil fileIndex")
		}
		importFileIndexes[int(imageImportRef.GetFileIndex())] = struct{}{}
	}
	return importFileIndexes, nil
}

// paths can be either files (ending in .proto) or directories
// paths must be normalized and validated, and not duplicated
// if a directory, all .proto files underneath will be included
func imageWithOnlyPaths(image Image, fileOrDirPaths []string, allowNotExist bool) (Image, error) {
	if err := bufcorevalidate.ValidateFileOrDirPaths(fileOrDirPaths); err != nil {
		return nil, err
	}
	// these are the files that fileOrDirPaths actually reference and will
	// result in the non-imports in our resulting Image
	// the Image will also include the ImageFiles that the nonImportImageFiles import
	var nonImportImageFiles []ImageFile
	nonImportPaths := make(map[string]struct{})
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
				// we have an ImageFile, therefile the fileOrDirPath was a file path
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
		// we had no potential directory paths as we were able to get
		// an ImageFile for all fileOrDirPaths, so we can return an Image now
		// this means we do not have to do the expensive O(image.Files()) operation
		// to check to see if each file is within a potential directory path
		return getImageWithImports(image, nonImportPaths, nonImportImageFiles)
	}
	// we have potential directory paths, do the expensive operation
	// make a map of the directory paths
	// note that we do not make this a map to begin with as maps are unordered,
	// and we want to make sure we iterate over the paths in a deterministic order
	potentialDirPathMap := stringutil.SliceToMap(potentialDirPaths)
	// the map of paths within potentialDirPath that matches a file in image.Files()
	// this needs to contain all paths in potentialDirPathMap at the end for us to
	// have had matches for every inputted fileOrDirPath
	matchingPotentialDirPathMap := make(map[string]struct{})
	for _, imageFile := range image.Files() {
		imageFilePath := imageFile.Path()
		// get the paths in potentialDirPathMap that match this imageFilePath
		fileMatchingPathMap := normalpath.MapAllEqualOrContainingPathMap(
			potentialDirPathMap,
			imageFilePath,
			normalpath.Relative,
		)
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
	}
	// we finally have all files that match fileOrDirPath that we can find, make the image
	return getImageWithImports(image, nonImportPaths, nonImportImageFiles)
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
	for _, importPath := range imageFile.ImportPaths() {
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
		imageFile.withIsImport(!isNotImport),
	)
	return accumulator
}

// Copyright 2020 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/internal/buf/bufcore/internal"
	imagev1 "github.com/bufbuild/buf/internal/gen/proto/go/buf/image/v1"
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

func imageWithOnlyPaths(
	image Image,
	paths []string,
	allowNotExist bool,
) (Image, error) {
	if err := internal.ValidateFileInfoPaths(paths); err != nil {
		return nil, err
	}
	var nonImportImageFiles []ImageFile
	nonImportPaths := make(map[string]struct{})
	for _, path := range paths {
		imageFile := image.GetFile(path)
		if imageFile == nil {
			if !allowNotExist {
				return nil, fmt.Errorf("%s is not present in the Image", path)
			}
		} else {
			nonImportImageFiles = append(nonImportImageFiles, imageFile)
			nonImportPaths[path] = struct{}{}
		}
	}
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

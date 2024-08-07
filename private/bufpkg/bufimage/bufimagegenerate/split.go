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
	"sort"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

// ImageByDirSplitImports returns multiple images split by directory.
//
// This function does not treat import files differently from non-import files.
//
// TODO: rename this and maybe it should be moved to the bufgen package.
//
// If strategy is dir, we want the files in file_to_generate inside each
// CodeGeneratorRequest to be in the same directory.
//
// For example, if we have an image with the following files:
//
// - a/a.proto -> b/b.proto
// - b/b.proto -> c/c.proto
// - b/b.proto -> d/d.proto
//
// where a/a.proto and b/b.proto are the non-imports, and c/c.proto and and d/d.proto are imports.
//
// Now, if we have includeImports set to true, we should send 4 CodeGeneratorRequests to each plugin.
// Each request should have file_to_generate equal to ["x/x.proto"] of length 1, with x being one of a, b, c or d,
// and its proto_file should include "all files in files_to_generate and everything they import", by contract.
//
// If includeImports is set to false, the request with c/c.proto to generate and the one with d/d.proto should not exist,
// and we should only send 2 requests to the plugin.
func ImageByDirSplitImports(image bufimage.Image) ([]ImageForGeneration, error) {
	dirToImageFilePaths := normalpath.ByDir(
		slicesext.Map(image.Files(), func(imageFile bufimage.ImageFile) string {
			return imageFile.Path()
		})...,
	)
	// we need this to produce a deterministic order of the returned Images
	dirs := make([]string, 0, len(dirToImageFilePaths))
	for dir := range dirToImageFilePaths {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	newImages := make([]ImageForGeneration, 0, len(dirToImageFilePaths))
	for _, dir := range dirs {
		imageFilePaths, ok := dirToImageFilePaths[dir]
		if !ok {
			// this should never happen
			return nil, syserror.Newf("no dir for %q in dirToImageFilePaths", dir)
		}
		imageFilesToGenerate, err := slicesext.MapError(imageFilePaths, func(filePath string) (bufimage.ImageFile, error) {
			imageFile := image.GetFile(filePath)
			if imageFile == nil {
				return nil, syserror.Newf("expected image file to exist at %q", filePath)
			}
			return imageFile, nil
		})
		if err != nil {
			return nil, err
		}
		newImage, err := getImageForGenerationForFilePaths(image, slicesext.ToStructMap(imageFilePaths), imageFilesToGenerate)
		if err != nil {
			return nil, err
		}
		newImages = append(newImages, newImage)
	}
	return newImages, nil
}

// *** PRIVATE ***

func getImageForGenerationForFilePaths(
	image bufimage.Image,
	generationPaths map[string]struct{},
	generationImageFiles []bufimage.ImageFile,
) (ImageForGeneration, error) {
	var imageFiles []bufimage.ImageFile
	seenPaths := make(map[string]struct{})
	for _, nonImportImageFile := range generationImageFiles {
		imageFiles = addFileForGenerationWithImports(
			imageFiles,
			image,
			generationPaths,
			seenPaths,
			nonImportImageFile,
		)
	}
	imageWithPaths, err := bufimage.NewImage(imageFiles)
	if err != nil {
		return nil, err
	}
	return newImageForGeneration(imageWithPaths, generationPaths), nil
}

// largely copied from addFileWithImports
//
// returns accumulated files in correct order
func addFileForGenerationWithImports(
	accumulator []bufimage.ImageFile,
	image bufimage.Image,
	nonImportPaths map[string]struct{},
	seenPaths map[string]struct{},
	imageFile bufimage.ImageFile,
) []bufimage.ImageFile {
	path := imageFile.Path()
	// if seen already, skip
	if _, ok := seenPaths[path]; ok {
		return accumulator
	}
	seenPaths[path] = struct{}{}

	// then, add imports first, for proper ordering
	for _, importPath := range imageFile.FileDescriptorProto().GetDependency() {
		if importFile := image.GetFile(importPath); importFile != nil {
			accumulator = addFileForGenerationWithImports(
				accumulator,
				image,
				nonImportPaths,
				seenPaths,
				importFile,
			)
		}
	}

	accumulator = append(
		accumulator,
		imageFile,
	)
	return accumulator
}

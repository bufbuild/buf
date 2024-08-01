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
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/slicesext"
)

// ImageForGeneration is a buf image to be generated.
type ImageForGeneration interface {
	// files are the files that comprise the image, but not all files should be generated.
	files() []imageFileForGeneration
}

// NewImageForGenerationFromImage creates an ImageForGeneration that works the exact same way as before
// this type is added, and this is the one of the two exported functions that construct an ImageForGeneration.
// The other one is ImageByDirSplitImports.
//
// TODO: update this doc
func NewImageForGenerationFromImage(image bufimage.Image) ImageForGeneration {
	imageFilesForGeneration := slicesext.Map(image.Files(), func(imageFile bufimage.ImageFile) imageFileForGeneration {
		return imageFileForGeneration{
			ImageFile:  imageFile,
			toGenerate: true,
		}
	})
	return imageForGeneration(imageFilesForGeneration)
}

// *** PRIVATE ***

type imageFileForGeneration struct {
	bufimage.ImageFile

	// toGenerate returns whether the file may be generated. This is not necessarily the same as IsImport(),
	// especially when strategy is set to directory. It also does not guarantee the file's inclusion in file_to_generate.
	//
	//  - If it returns true, the file will be generated unless includeImports or includeWKT says otherwise.
	//  - If it returns false, it will not be generated regardless of includeImports and includeWKT.
	toGenerate bool
}

type imageForGeneration []imageFileForGeneration

func (i imageForGeneration) files() []imageFileForGeneration {
	return []imageFileForGeneration(i)
}

func newImageForGeneration(image bufimage.Image, filesToGenerate map[string]struct{}) ImageForGeneration {
	imageFilesForGeneration := imageForGeneration(slicesext.Map(image.Files(), func(imageFile bufimage.ImageFile) imageFileForGeneration {
		_, ok := filesToGenerate[imageFile.Path()]
		return imageFileForGeneration{
			ImageFile:  imageFile,
			toGenerate: ok,
		}
	}))
	return imageForGeneration(imageFilesForGeneration)
}

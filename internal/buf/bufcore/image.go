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

package bufcore

import "fmt"

var _ Image = &image{}

type image struct {
	files           []ImageFile
	pathToImageFile map[string]ImageFile
}

func newImage(files []ImageFile) (*image, error) {
	pathToImageFile := make(map[string]ImageFile, len(files))
	for _, file := range files {
		path := file.Path()
		if _, ok := pathToImageFile[path]; ok {
			return nil, fmt.Errorf("duplicate file: %s", path)
		}
		pathToImageFile[path] = file
	}
	return &image{
		files:           files,
		pathToImageFile: pathToImageFile,
	}, nil
}

func newImageNoValidate(files []ImageFile) *image {
	pathToImageFile := make(map[string]ImageFile, len(files))
	for _, file := range files {
		path := file.Path()
		pathToImageFile[path] = file
	}
	return &image{
		files:           files,
		pathToImageFile: pathToImageFile,
	}
}

func (i *image) Files() []ImageFile {
	return i.files
}

func (i *image) NonImportFiles() []ImageFile {
	nonImportFiles := make([]ImageFile, 0, len(i.files))
	for _, file := range i.files {
		if !file.IsImport() {
			nonImportFiles = append(nonImportFiles, file)
		}
	}
	return nonImportFiles
}

func (i *image) GetFile(path string) ImageFile {
	return i.pathToImageFile[path]
}

func (*image) isImage() {}

// Copyright 2020 Buf Technologies Inc.
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

import "fmt"

type image struct {
	files                 []File
	rootRelFilePathToFile map[string]File
}

func newImage(files []File) (*image, error) {
	rootRelFilePathToFile := make(map[string]File, len(files))
	for _, file := range files {
		rootRelFilePath := file.RootRelFilePath()
		if _, ok := rootRelFilePathToFile[rootRelFilePath]; ok {
			return nil, fmt.Errorf("duplicate file: %s", rootRelFilePath)
		}
		rootRelFilePathToFile[rootRelFilePath] = file
	}
	return &image{
		files:                 files,
		rootRelFilePathToFile: rootRelFilePathToFile,
	}, nil
}

func newImageNoValidate(files []File) *image {
	rootRelFilePathToFile := make(map[string]File, len(files))
	for _, file := range files {
		rootRelFilePath := file.RootRelFilePath()
		rootRelFilePathToFile[rootRelFilePath] = file
	}
	return &image{
		files:                 files,
		rootRelFilePathToFile: rootRelFilePathToFile,
	}
}

func (i *image) Files() []File {
	return i.files
}

func (i *image) GetFile(rootRelFilePath string) File {
	return i.rootRelFilePathToFile[rootRelFilePath]
}

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

	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/gofrs/uuid/v5"
	"google.golang.org/protobuf/types/descriptorpb"
)

var _ Image = &image{}

type image struct {
	files           []ImageFile
	pathToImageFile map[string]ImageFile
	resolver        protoencoding.Resolver
}

func newImage(files []ImageFile, reorder bool, resolver protoencoding.Resolver) (*image, error) {
	if len(files) == 0 {
		return nil, errors.New("image contains no files")
	}
	pathToImageFile := make(map[string]ImageFile, len(files))
	type commitIDAndFilePath struct {
		commitID uuid.UUID
		filePath string
	}
	moduleFullNameStringToCommitIDAndFilePath := make(map[string]commitIDAndFilePath)
	for _, file := range files {
		path := file.Path()
		if _, ok := pathToImageFile[path]; ok {
			return nil, fmt.Errorf("duplicate file: %s", path)
		}
		pathToImageFile[path] = file
		if moduleFullName := file.ModuleFullName(); moduleFullName != nil {
			moduleFullNameString := moduleFullName.String()
			existing, ok := moduleFullNameStringToCommitIDAndFilePath[moduleFullNameString]
			if ok {
				if existing.commitID != file.CommitID() {
					return nil, fmt.Errorf(
						"files with different commits for the same module %s: %s:%s and %s:%s",
						moduleFullNameString,
						existing.filePath,
						uuidutil.ToDashless(existing.commitID),
						path,
						uuidutil.ToDashless(file.CommitID()),
					)
				}
			} else {
				moduleFullNameStringToCommitIDAndFilePath[moduleFullNameString] = commitIDAndFilePath{
					commitID: file.CommitID(),
					filePath: path,
				}
			}
		}
	}
	if reorder {
		files = orderImageFiles(files, pathToImageFile)
	}
	if resolver == nil {
		fileDescriptorProtos := make([]*descriptorpb.FileDescriptorProto, len(files))
		for i := range files {
			fileDescriptorProtos[i] = files[i].FileDescriptorProto()
		}
		resolver = protoencoding.NewLazyResolver(fileDescriptorProtos...)
	}
	return &image{
		files:           files,
		pathToImageFile: pathToImageFile,
		resolver:        resolver,
	}, nil
}

func newImageNoValidate(files []ImageFile, resolver protoencoding.Resolver) *image {
	pathToImageFile := make(map[string]ImageFile, len(files))
	for _, file := range files {
		path := file.Path()
		pathToImageFile[path] = file
	}
	return &image{
		files:           files,
		pathToImageFile: pathToImageFile,
		resolver:        resolver,
	}
}

func (i *image) Files() []ImageFile {
	return i.files
}

func (i *image) GetFile(path string) ImageFile {
	return i.pathToImageFile[path]
}

func (i *image) Resolver() protoencoding.Resolver {
	return i.resolver
}

func (*image) isImage() {}

// orderImageFiles re-orders the ImageFiles in DAG order.
func orderImageFiles(
	inputImageFiles []ImageFile,
	pathToImageFile map[string]ImageFile,
) []ImageFile {
	outputImageFiles := make([]ImageFile, 0, len(inputImageFiles))
	alreadySeen := map[string]struct{}{}
	for _, inputImageFile := range inputImageFiles {
		outputImageFiles = orderImageFilesRec(
			inputImageFile,
			outputImageFiles,
			pathToImageFile,
			alreadySeen,
		)
	}
	return outputImageFiles
}

func orderImageFilesRec(
	inputImageFile ImageFile,
	outputImageFiles []ImageFile,
	pathToImageFile map[string]ImageFile,
	alreadySeen map[string]struct{},
) []ImageFile {
	path := inputImageFile.Path()
	if _, ok := alreadySeen[path]; ok {
		return outputImageFiles
	}
	alreadySeen[path] = struct{}{}
	for _, dependency := range inputImageFile.FileDescriptorProto().GetDependency() {
		dependencyImageFile, ok := pathToImageFile[dependency]
		if ok {
			outputImageFiles = orderImageFilesRec(
				dependencyImageFile,
				outputImageFiles,
				pathToImageFile,
				alreadySeen,
			)
		}
	}
	return append(outputImageFiles, inputImageFile)
}

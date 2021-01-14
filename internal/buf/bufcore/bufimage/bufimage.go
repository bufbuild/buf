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
	"github.com/bufbuild/buf/internal/buf/bufcore"
	imagev1 "github.com/bufbuild/buf/internal/gen/proto/go/buf/alpha/image/v1"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/protodescriptor"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// ImageFile is a Protobuf file within an image.
type ImageFile interface {
	bufcore.FileInfo
	// Proto is the backing FileDescriptorProto for this File.
	//
	// This will never be nil.
	// The value Path() is equal to Proto.GetName() .
	// The value ImportPaths() is equal to Proto().GetDependency().
	Proto() *descriptorpb.FileDescriptorProto
	// ImportPaths returns the root relative file paths of the imports.
	//
	// The values will be normalized, validated, and never empty.
	// This is equal to Proto.GetDependency().
	ImportPaths() []string

	withIsImport(isImport bool) ImageFile
	isImageFile()
}

// NewImageFile returns a new ImageFile.
//
// If externalPath is empty, path is used.
func NewImageFile(
	fileDescriptorProto *descriptorpb.FileDescriptorProto,
	externalPath string,
	isImport bool,
) (ImageFile, error) {
	return newImageFile(
		fileDescriptorProto,
		externalPath,
		isImport,
	)
}

// Image is a buf image.
type Image interface {
	// Files are the files that comprise the image.
	//
	// This contains all files, including imports if available.
	// The returned files are in correct DAG order.
	Files() []ImageFile
	// GetFile gets the file for the root relative file path.
	//
	// If the file does not exist, nil is returned.
	// The path is expected to be normalized and validated.
	// Note that all values of GetDependency() can be used here.
	GetFile(path string) ImageFile
	isImage()
}

// NewImage returns a new Image for the given ImageFiles.
//
// The input ImageFiles are expected to be in correct DAG order!
// TODO: Consider checking the above, and if not, reordering the Files.
// If imageFiles is empty, returns error
func NewImage(imageFiles []ImageFile) (Image, error) {
	return newImage(imageFiles, false)
}

// NewMultiImage returns a new Image for the given Images.
//
// Reorders the ImageFiles to be in DAG order.
// Duplicates cannot exist across the Images.
func NewMultiImage(images ...Image) (Image, error) {
	switch len(images) {
	case 0:
		return nil, nil
	case 1:
		return images[0], nil
	default:
		var imageFiles []ImageFile
		for _, image := range images {
			imageFiles = append(imageFiles, image.Files()...)
		}
		return newImage(imageFiles, true)
	}
}

// NewImageForProto returns a new Image for the given proto Image.
//
// The input Files are expected to be in correct DAG order!
// TODO: Consider checking the above, and if not, reordering the Files.
//
// TODO: do we want to add the ability to do external path resolution here?
func NewImageForProto(protoImage *imagev1.Image) (Image, error) {
	if err := validateProtoImageExceptFileDescriptorProtos(protoImage); err != nil {
		return nil, err
	}
	importFileIndexes, err := getImportFileIndexes(protoImage)
	if err != nil {
		return nil, err
	}
	imageFiles := make([]ImageFile, len(protoImage.File))
	for i, fileDescriptorProto := range protoImage.File {
		_, isImport := importFileIndexes[i]
		imageFile, err := NewImageFile(fileDescriptorProto, fileDescriptorProto.GetName(), isImport)
		if err != nil {
			return nil, err
		}
		imageFiles[i] = imageFile
	}
	return NewImage(imageFiles)
}

// NewImageForCodeGeneratorRequest returns a new Image from a given CodeGeneratorRequest.
//
// The input Files are expected to be in correct DAG order!
// TODO: Consider checking the above, and if not, reordering the Files.
func NewImageForCodeGeneratorRequest(request *pluginpb.CodeGeneratorRequest) (Image, error) {
	if err := protodescriptor.ValidateCodeGeneratorRequestExceptFileDescriptorProtos(request); err != nil {
		return nil, err
	}
	image, err := NewImageForProto(
		&imagev1.Image{
			File: request.GetProtoFile(),
		},
	)
	if err != nil {
		return nil, err
	}
	return ImageWithOnlyPaths(
		image,
		request.GetFileToGenerate(),
	)
}

// ImageWithoutImports returns a copy of the Image without imports.
//
// The backing Files are not copied.
func ImageWithoutImports(image Image) Image {
	imageFiles := image.Files()
	newImageFiles := make([]ImageFile, 0, len(imageFiles))
	for _, imageFile := range imageFiles {
		if !imageFile.IsImport() {
			newImageFiles = append(newImageFiles, imageFile)
		}
	}
	return newImageNoValidate(newImageFiles)
}

// ImageWithOnlyPaths returns a copy of the Image that only includes the files
// with the given root relative file paths or directories.
//
// Note that paths can be either files or directories - whether or not a path
// is included is a result of normalpath.EqualsOrContainsPath.
//
// If a root relative file path does not exist, this errors.
func ImageWithOnlyPaths(
	image Image,
	paths []string,
) (Image, error) {
	return imageWithOnlyPaths(image, paths, false)
}

// ImageWithOnlyPathsAllowNotExist returns a copy of the Image that only includes the files
// with the given root relative file paths.
//
// Note that paths can be either files or directories - whether or not a path
// is included is a result of normalpath.EqualsOrContainsPath.
//
// If a root relative file path does not exist, this skips this path.
func ImageWithOnlyPathsAllowNotExist(
	image Image,
	paths []string,
) (Image, error) {
	return imageWithOnlyPaths(image, paths, true)
}

// ImageByDir returns multiple images that have non-imports split
// by directory.
//
// That is, each Image will only contain a single directoy's files
// as it's non-imports, along with all required imports for the
// files in that directory.
func ImageByDir(image Image) ([]Image, error) {
	imageFiles := image.Files()
	paths := make([]string, 0, len(imageFiles))
	for _, imageFile := range imageFiles {
		if !imageFile.IsImport() {
			paths = append(paths, imageFile.Path())
		}
	}
	dirToPaths := normalpath.ByDir(paths...)
	newImages := make([]Image, 0, len(dirToPaths))
	for _, paths := range dirToPaths {
		newImage, err := ImageWithOnlyPaths(image, paths)
		if err != nil {
			return nil, err
		}
		newImages = append(newImages, newImage)
	}
	return newImages, nil
}

// ImageToProtoImage returns a new ProtoImage for the Image.
func ImageToProtoImage(image Image) *imagev1.Image {
	imageFiles := image.Files()
	protoImage := &imagev1.Image{
		File: make([]*descriptorpb.FileDescriptorProto, len(imageFiles)),
		BufbuildImageExtension: &imagev1.ImageExtension{
			ImageImportRefs: make([]*imagev1.ImageImportRef, 0),
		},
	}
	for i, imageFile := range imageFiles {
		protoImage.File[i] = imageFile.Proto()
		if imageFile.IsImport() {
			protoImage.BufbuildImageExtension.ImageImportRefs = append(
				protoImage.BufbuildImageExtension.ImageImportRefs,
				&imagev1.ImageImportRef{
					FileIndex: proto.Uint32(uint32(i)),
				},
			)
		}
	}
	return protoImage
}

// ImageToFileDescriptorSet returns a new FileDescriptorSet for the Image.
func ImageToFileDescriptorSet(image Image) *descriptorpb.FileDescriptorSet {
	return &descriptorpb.FileDescriptorSet{
		File: ImageToFileDescriptorProtos(image),
	}
}

// ImageToFileDescriptorProtos returns a the FileDescriptorProtos for the Image.
func ImageToFileDescriptorProtos(image Image) []*descriptorpb.FileDescriptorProto {
	imageFiles := image.Files()
	fileDescriptorProtos := make([]*descriptorpb.FileDescriptorProto, len(imageFiles))
	for i, imageFile := range imageFiles {
		fileDescriptorProtos[i] = imageFile.Proto()
	}
	return fileDescriptorProtos
}

// ImageToCodeGeneratorRequest returns a new CodeGeneratorRequest for the Image.
//
// All non-imports are added as files to generate.
func ImageToCodeGeneratorRequest(image Image, parameter string) *pluginpb.CodeGeneratorRequest {
	imageFiles := image.Files()
	request := &pluginpb.CodeGeneratorRequest{
		ProtoFile: make([]*descriptorpb.FileDescriptorProto, len(imageFiles)),
	}
	if parameter != "" {
		request.Parameter = proto.String(parameter)
	}
	for i, imageFile := range imageFiles {
		request.ProtoFile[i] = imageFile.Proto()
		if !imageFile.IsImport() {
			request.FileToGenerate = append(request.FileToGenerate, imageFile.Path())
		}
	}
	return request
}

// ImagesToCodeGeneratorRequests converts the Images to CodeGeneratorRequests.
//
// All non-imports are added as files to generate.
func ImagesToCodeGeneratorRequests(images []Image, parameter string) []*pluginpb.CodeGeneratorRequest {
	requests := make([]*pluginpb.CodeGeneratorRequest, len(images))
	for i, image := range images {
		requests[i] = ImageToCodeGeneratorRequest(image, parameter)
	}
	return requests
}

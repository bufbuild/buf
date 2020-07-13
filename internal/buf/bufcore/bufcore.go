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

// Package bufcore contains core types.
package bufcore

import (
	"context"
	"errors"
	"io"

	imagev1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/image/v1"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/protodescriptor"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// ErrNoTargetFiles is the error returned if there are no target files found.
var ErrNoTargetFiles = errors.New("no .proto target files found")

// FileInfo contains protobuf file info.
type FileInfo interface {
	// Path is the path of the file relative to the root it is contained within.
	// This will be normalized, validated and never empty,
	// This will be unique within a given Image.
	Path() string
	// ExternalPath returns the path that identifies this file externally.
	//
	// This will be unnormalized.
	// Never empty. Falls back to Path if there is not an external path.
	//
	// Example:
	//	 Assume we had the input path /foo/bar which is a local directory.

	//   Path: one/one.proto
	//   RootDirPath: proto
	//   ExternalPath: /foo/bar/proto/one/one.proto
	ExternalPath() string
	// IsImport returns true if this file is an import.
	IsImport() bool
	isFileInfo()
}

// NewFileInfo returns a new FileInfo.
//
// If externalPath is empty, path is used.
func NewFileInfo(path string, externalPath string, isImport bool) (FileInfo, error) {
	return newFileInfo(path, externalPath, isImport)
}

// NewFileInfoForObjectInfo returns a new FileInfo for the storage.ObjectInfo.
//
// The same rules apply to ObjectInfos for paths as FileInfos so we do not need to validate.
func NewFileInfoForObjectInfo(objectInfo storage.ObjectInfo, isImport bool) FileInfo {
	return newFileInfoForObjectInfo(objectInfo, isImport)
}

// ImageFile is a Protobuf file within an image.
type ImageFile interface {
	FileInfo
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

// ModuleFile is a file within a Root.
type ModuleFile interface {
	FileInfo
	io.ReadCloser
	isModuleFile()
}

// Module is a Protobuf module.
type Module interface {
	// TargetFileInfos gets all FileRefs for the Module that should be built.
	//
	// This does not include imports, or if ModuleWithTargetPaths() was used, files
	// not specified with ModuleWithTargetPaths().
	TargetFileInfos(ctx context.Context) ([]FileInfo, error)
	// GetFile gets the file for the given path.
	//
	// Returns storage.IsNotExist error if the file does not exist.
	GetFile(ctx context.Context, path string) (ModuleFile, error)
	// GetFileInfo gets the FileInfo for the given path.
	//
	// This includes non-source FileInfos, i.e. imports and those not specified as
	// part of ModuleWithTargetPaths().
	//
	// Returns storage.IsNotExist error if the file does not exist.
	GetFileInfo(ctx context.Context, path string) (FileInfo, error)
	isModule()
}

// NewModule returns a new Module.
func NewModule(readBucket storage.ReadBucket, options ...ModuleOption) (Module, error) {
	return newModule(readBucket, options...)
}

// ModuleOption is an option for a new Module.
type ModuleOption func(*module)

// ModuleWithImports returns a new ModuleOption that adds the given ReadBucket for imports.
//
// This bucket CANNOT be the same or have overlap with the bucket given to NewModule.
func ModuleWithImports(importReadBucket storage.ReadBucket) ModuleOption {
	return func(module *module) {
		module.importReadBucket = importReadBucket
	}
}

// ModuleWithTargetPaths returns a new ModuleOption that specifies specific file paths to build.
//
// These paths must exist.
// These paths must be relative to any roots.
// These paths will be normalized and validated.
// Multiple calls to this option will override previous calls.
func ModuleWithTargetPaths(targetPaths ...string) ModuleOption {
	return func(module *module) {
		module.targetPaths = targetPaths
	}
}

// ModuleWithTargetPathsAllowNotExistOnWalk returns a ModuleOption that says that the
// target paths specified with ModuleWithTargetPaths may not exist on TargetFileInfos
// calls.
//
// GetFileInfo and GetFile will still operate as normal.
func ModuleWithTargetPathsAllowNotExistOnWalk() ModuleOption {
	return func(module *module) {
		module.targetPathsAllowNotExistOnWalk = true
	}
}

// ***** Helpers *****

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

// ImageWithOnlyPaths returns a copy of the Image that only includes the Files
// with the given root relative file paths.
//
// If a root relative file path does not exist, this errors.
func ImageWithOnlyPaths(
	image Image,
	paths []string,
) (Image, error) {
	return imageWithOnlyPaths(image, paths, false)
}

// ImageWithOnlyPathsAllowNotExist returns a copy of the Image that only includes the Files
// with the given root relative file paths.
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

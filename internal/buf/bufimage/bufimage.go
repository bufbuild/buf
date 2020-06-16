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
	"fmt"

	"github.com/bufbuild/buf/internal/buf/bufpath"
	imagev1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/image/v1"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// FileRef is a Protobuf file reference.
type FileRef interface {
	// RootRelFilePath is the path of the file relative to the root it is contained within.
	// This will be normalized, validated and never empty,
	// This will be unique within a given Image.
	RootRelFilePath() string
	// RootDirPath is the directory path of the root that contains this file.
	//
	// This will be normalized, validated and never empty,
	// The full relative path to the file within a given context is normalpath.Join(RootDirPath(), RootRelFilePath())
	// For FileRefs created from external images, this will always be ".".
	//
	// This path should be used for informational purposes only and not
	// used to help uniquely identify a file within an Image.
	RootDirPath() string
	// ExternalFilePath returns the path that identifies this file externally.
	//
	// Example:
	//	 Assume we had the input path /foo/bar which is a local directory.

	//   RootRelFilePath: one/one.proto
	//   RootDirPath: proto
	//   ExternalFilePath: /foo/bar/proto/one/one.proto
	ExternalFilePath() string
	isFileRef()
}

// NewFileRef returns a new FileRef.
func NewFileRef(
	rootRelFilePath string,
	rootDirPath string,
	externalPathResolver bufpath.ExternalPathResolver,
) (FileRef, error) {
	return newFileRef(
		rootRelFilePath,
		rootDirPath,
		externalPathResolver,
	)
}

// NewDirectFileRef returns a new FileRef created with a direct external file path.
//
// This should only be used in testing.
func NewDirectFileRef(
	rootRelFilePath string,
	rootDirPath string,
	externalFilePath string,
) (FileRef, error) {
	return newDirectFileRef(
		rootRelFilePath,
		rootDirPath,
		externalFilePath,
	)
}

// File is a Protobuf file within an image.
type File interface {
	FileRef
	// ImportRootRelFilePaths returns the root relative file paths of the imports.
	//
	// The values will be normalized, validated, and never empty.
	// This is equal to Proto.GetDependency().
	ImportRootRelFilePaths() []string
	// Proto is the backing FileDescriptorProto for this File.
	//
	// This will never be nil.
	// The value RootRelFilePath() is equal to Proto.GetName() .
	// The value ImportRootRelFilePaths() is equal to Proto().GetDependency().
	Proto() *descriptorpb.FileDescriptorProto
	// IsImport returns true if this file is an import in the context of the enclosing Image.
	IsImport() bool
	isFile()
}

// NewFile returns a new File.
func NewFile(
	fileDescriptorProto *descriptorpb.FileDescriptorProto,
	rootDirPath string,
	externalPathResolver bufpath.ExternalPathResolver,
	isImport bool,
) (File, error) {
	return newFile(
		fileDescriptorProto,
		rootDirPath,
		externalPathResolver,
		isImport,
	)
}

// NewDirectFile returns a new File created with a direct external file path.
//
// This should only be used in testing.
func NewDirectFile(
	fileDescriptorProto *descriptorpb.FileDescriptorProto,
	rootDirPath string,
	externalFilePath string,
	isImport bool,
) (File, error) {
	return newDirectFile(
		fileDescriptorProto,
		rootDirPath,
		externalFilePath,
		isImport,
	)
}

// Image is a buf image.
type Image interface {
	// Files are the files that comprise the image.
	//
	// This contains all files, including imports if available.
	// The returned files are in correct DAG order.
	Files() []File
	// GetFile gets the file for the root relative file path.
	//
	// If the file does not exist, nil is returned.
	// The rootRelFilePath is expected to be normalized and validated.
	// Note that all values of GetDependency() can be used here.
	GetFile(rootRelFilePath string) File
}

// NewImage returns a new Image for the given Files.
//
// The input Files are expected to be in correct DAG order!
// TODO: Consider checking the above, and if not, reordering the Files.
func NewImage(files []File) (Image, error) {
	return newImage(files)
}

// NewImageForProto returns a new Image for the given proto Image.
//
// The input Files are expected to be in correct DAG order!
// TODO: Consider checking the above, and if not, reordering the Files.
func NewImageForProto(protoImage *imagev1.Image) (Image, error) {
	if err := validateProtoImageExceptFileDescriptorProtos(protoImage); err != nil {
		return nil, err
	}
	importFileIndexes, err := getImportFileIndexes(protoImage)
	if err != nil {
		return nil, err
	}
	files := make([]File, len(protoImage.File))
	for i, fileDescriptorProto := range protoImage.File {
		_, isImport := importFileIndexes[i]
		file, err := NewFile(fileDescriptorProto, ".", bufpath.NopPathResolver, isImport)
		if err != nil {
			return nil, err
		}
		files[i] = file
	}
	return NewImage(files)
}

// NewImageForCodeGeneratorRequest returns a new Image from a given CodeGeneratorRequest.
//
// externalPathResolver may be nil.
//
// The input Files are expected to be in correct DAG order!
// TODO: Consider checking the above, and if not, reordering the Files.
func NewImageForCodeGeneratorRequest(request *pluginpb.CodeGeneratorRequest) (Image, error) {
	if err := validateCodeGeneratorRequestExceptFileDescriptorProtos(request); err != nil {
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
	return ImageWithOnlyRootRelFilePaths(
		image,
		request.GetFileToGenerate(),
	)
}

// FileRefFullRelFilePath returns the full relative file path.
//
// This is the root directory path joined with the root relative file path.
//
// Example:
//   RootRelFilePath: one/one.proto
//   RootDirPath: proto
//   FullRelFilePath: proto/one/one.proto
func FileRefFullRelFilePath(fileRef FileRef) string {
	return normalpath.Join(fileRef.RootDirPath(), fileRef.RootRelFilePath())
}

// ImageWithoutImports returns a copy of the Image without imports.
//
// The backing Files are not copied.
func ImageWithoutImports(image Image) Image {
	files := image.Files()
	newFiles := make([]File, 0, len(files))
	for _, file := range files {
		if !file.IsImport() {
			newFiles = append(newFiles, file)
		}
	}
	return newImageNoValidate(newFiles)
}

// ImageWithOnlyRootRelFilePaths returns a copy of the Image that only includes the Files
// with the given root relative file paths.
//
// If a root relative file path does not exist, this errors.
func ImageWithOnlyRootRelFilePaths(
	image Image,
	rootRelFilePaths []string,
) (Image, error) {
	return imageWithOnlyRootRelFilePaths(image, rootRelFilePaths, false)
}

// ImageWithOnlyRootRelFilePathsAllowNotExist returns a copy of the Image that only includes the Files
// with the given root relative file paths.
//
// If a root relative file path does not exist, this skips this path.
func ImageWithOnlyRootRelFilePathsAllowNotExist(
	image Image,
	rootRelFilePaths []string,
) (Image, error) {
	return imageWithOnlyRootRelFilePaths(image, rootRelFilePaths, true)
}

// ImageToProtoImage returns a new ProtoImage for the Image.
func ImageToProtoImage(image Image) *imagev1.Image {
	files := image.Files()
	protoImage := &imagev1.Image{
		File: make([]*descriptorpb.FileDescriptorProto, len(files)),
		BufbuildImageExtension: &imagev1.ImageExtension{
			ImageImportRefs: make([]*imagev1.ImageImportRef, 0),
		},
	}
	for i, file := range files {
		protoImage.File[i] = file.Proto()
		if file.IsImport() {
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
	files := image.Files()
	fileDescriptorProtos := make([]*descriptorpb.FileDescriptorProto, len(files))
	for i, file := range files {
		fileDescriptorProtos[i] = file.Proto()
	}
	return fileDescriptorProtos
}

// ImageToCodeGeneratorRequest returns a new CodeGeneratorRequest for the Image.
//
// All non-imports are added as files to generate.
func ImageToCodeGeneratorRequest(image Image, parameter string) *pluginpb.CodeGeneratorRequest {
	files := image.Files()
	request := &pluginpb.CodeGeneratorRequest{
		ProtoFile: make([]*descriptorpb.FileDescriptorProto, len(files)),
	}
	if parameter != "" {
		request.Parameter = proto.String(parameter)
	}
	for i, file := range files {
		request.ProtoFile[i] = file.Proto()
		if !file.IsImport() {
			request.FileToGenerate = append(request.FileToGenerate, file.RootRelFilePath())
		}
	}
	return request
}

func imageWithOnlyRootRelFilePaths(
	image Image,
	rootRelFilePaths []string,
	allowNotExist bool,
) (Image, error) {
	if err := validateRootRelFilePaths(rootRelFilePaths); err != nil {
		return nil, err
	}
	var nonImportFiles []File
	nonImportRootRelFilePaths := make(map[string]struct{})
	for _, rootRelFilePath := range rootRelFilePaths {
		file := image.GetFile(rootRelFilePath)
		if file == nil {
			if !allowNotExist {
				return nil, fmt.Errorf("%s is not present in the Image", rootRelFilePath)
			}
		} else {
			nonImportFiles = append(nonImportFiles, file)
			nonImportRootRelFilePaths[rootRelFilePath] = struct{}{}
		}
	}
	var files []File
	seenRootRelFilePaths := make(map[string]struct{})
	for _, nonImportFile := range nonImportFiles {
		files = addFileWithImports(
			files,
			image,
			nonImportRootRelFilePaths,
			seenRootRelFilePaths,
			nonImportFile,
		)
	}
	return NewImage(files)
}

// returns accumulated files in correct order
func addFileWithImports(
	accumulator []File,
	image Image,
	nonImportRootRelFilePaths map[string]struct{},
	seenRootRelFilePaths map[string]struct{},
	file File,
) []File {
	rootRelFilePath := file.RootRelFilePath()
	// if seen already, skip
	if _, ok := seenRootRelFilePaths[rootRelFilePath]; ok {
		return accumulator
	}
	seenRootRelFilePaths[rootRelFilePath] = struct{}{}

	// then, add imports first, for proper ordering
	for _, importRootRelFilePath := range file.ImportRootRelFilePaths() {
		if importFile := image.GetFile(importRootRelFilePath); importFile != nil {
			accumulator = addFileWithImports(
				accumulator,
				image,
				nonImportRootRelFilePaths,
				seenRootRelFilePaths,
				importFile,
			)
		}
	}

	// finally, add this file
	// check if this is an import or not
	_, isNotImport := nonImportRootRelFilePaths[rootRelFilePath]
	accumulator = append(
		accumulator,
		newFileNoValidate(
			file.Proto(),
			file.RootDirPath(),
			file.ExternalFilePath(),
			!isNotImport,
		),
	)
	return accumulator
}

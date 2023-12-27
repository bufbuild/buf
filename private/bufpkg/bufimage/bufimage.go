// Copyright 2020-2023 Buf Technologies, Inc.
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
	"context"
	"fmt"
	"sort"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	imagev1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/image/v1"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/protodescriptor"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// ImageFile is a Protobuf file within an image.
type ImageFile interface {
	storage.ObjectInfo

	// ModuleFullName returns the full name of the Module that this ImageFile came from,
	// if the ImageFile came from a Module (as opposed to a serialized Protobuf message),
	// and if the ModuleFullName was known.
	//
	// May be nil. Callers should not rely on this value being present.
	ModuleFullName() bufmodule.ModuleFullName
	// CommitID returns the BSR ID of the Commit of the Module that this ImageFile came from.
	// if the ImageFile came from a Module (as opposed to a serialized Protobuf message), and
	// if the CommitID was known..
	//
	// May be empty. Callers should not rely on this value being present. If
	// ModuleFullName is nil, this will always be empty.
	CommitID() string

	// FileDescriptorProto is the backing *descriptorpb.FileDescriptorProto for this File.
	//
	// This will never be nil.
	// The value Path() is equal to FileDescriptorProto().GetName() .
	FileDescriptorProto() *descriptorpb.FileDescriptorProto
	// IsImport returns true if this file is an import.
	IsImport() bool
	// IsSyntaxUnspecified will be true if the syntax was not explicitly specified.
	IsSyntaxUnspecified() bool
	// UnusedDependencyIndexes returns the indexes of the unused dependencies within
	// FileDescriptor.GetDependency().
	//
	// All indexes will be valid.
	// Will return nil if empty.
	UnusedDependencyIndexes() []int32

	isImageFile()
}

// NewImageFile returns a new ImageFile.
//
// If externalPath is empty, path is used.
//
// TODO: moduleFullName and commitID should be options since they are optional.
func NewImageFile(
	fileDescriptor protodescriptor.FileDescriptor,
	moduleFullName bufmodule.ModuleFullName,
	commitID string,
	externalPath string,
	isImport bool,
	isSyntaxUnspecified bool,
	unusedDependencyIndexes []int32,
) (ImageFile, error) {
	return newImageFile(
		fileDescriptor,
		moduleFullName,
		commitID,
		externalPath,
		isImport,
		isSyntaxUnspecified,
		unusedDependencyIndexes,
	)
}

// ImageFileWithIsImport returns a copy of the ImageFile with the new ImageFile
// now marked as an import.
//
// If the original ImageFile was already an import, this returns
// the original ImageFile.
func ImageFileWithIsImport(imageFile ImageFile, isImport bool) ImageFile {
	if imageFile.IsImport() == isImport {
		return imageFile
	}
	// No need to validate as ImageFile is already validated.
	return newImageFileNoValidate(
		imageFile.FileDescriptorProto(),
		imageFile.ModuleFullName(),
		imageFile.CommitID(),
		imageFile.ExternalPath(),
		isImport,
		imageFile.IsSyntaxUnspecified(),
		imageFile.UnusedDependencyIndexes(),
	)
}

// Image is a buf image.
type Image interface {
	// Files are the files that comprise the image.
	//
	// This contains all files, including imports if available.
	// The returned files are in correct DAG order.
	//
	// All files that have the same ModuleFullName will also have the same commit, or no commit.
	// This is enforced at construction time.
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

// BuildImage runs compilation.
//
// An error of type FileAnnotationSet may be returned. It is up to the caller to parse this if needed.
// FileAnnotations will use external file paths.
//
// The given ModuleReadBucket must be self-contained.
//
// A ModuleReadBucket is self-contained if it was constructed from
// ModuleSetToModuleReadBucketWithOnlyProtoFiles or
// ModuleToSelfContainedModuleReadBucketWithOnlyProtoFiles. These are likely
// the only two ways you should have a ModuleReadBucket that you pass to BuildImage.
func BuildImage(
	ctx context.Context,
	tracer tracing.Tracer,
	moduleReadBucket bufmodule.ModuleReadBucket,
	options ...BuildImageOption,
) (Image, error) {
	buildImageOptions := newBuildImageOptions()
	for _, option := range options {
		option(buildImageOptions)
	}
	return buildImage(
		ctx,
		tracer,
		moduleReadBucket,
		buildImageOptions.excludeSourceCodeInfo,
		buildImageOptions.noParallelism,
	)
}

// BuildImageOption is an option for BuildImage.
type BuildImageOption func(*buildImageOptions)

// WithExcludeSourceCodeInfo returns a new BuildImageOption that excludes sourceCodeInfo.
func WithExcludeSourceCodeInfo() BuildImageOption {
	return func(buildImageOptions *buildImageOptions) {
		buildImageOptions.excludeSourceCodeInfo = true
	}
}

// WithNoParallelism turns off parallelism for a build.
//
// The default is to use thread.Parallelism().
//
// Used for testing.
func WithNoParallelism() BuildImageOption {
	return func(buildImageOptions *buildImageOptions) {
		buildImageOptions.noParallelism = true
	}
}

// MergeImages returns a new Image for the given Images. ImageFiles
// treated as non-imports in at least one of the given Images will
// be treated as non-imports in the returned Image. The first non-import
// version of a file will be used in the result.
//
// Reorders the ImageFiles to be in DAG order.
// Duplicates can exist across the Images, but only if duplicates are non-imports.
func MergeImages(images ...Image) (Image, error) {
	switch len(images) {
	case 0:
		return nil, nil
	case 1:
		return images[0], nil
	default:
		var paths []string
		imageFileSet := make(map[string]ImageFile)
		for _, image := range images {
			for _, currentImageFile := range image.Files() {
				storedImageFile, ok := imageFileSet[currentImageFile.Path()]
				if !ok {
					imageFileSet[currentImageFile.Path()] = currentImageFile
					paths = append(paths, currentImageFile.Path())
					continue
				}
				if !storedImageFile.IsImport() && !currentImageFile.IsImport() {
					return nil, fmt.Errorf("%s is a non-import in multiple images", currentImageFile.Path())
				}
				if storedImageFile.IsImport() && !currentImageFile.IsImport() {
					imageFileSet[currentImageFile.Path()] = currentImageFile
				}
			}
		}
		// We need to preserve order for deterministic results, so we add
		// the files in the order they're given, but base our selection
		// on the imageFileSet.
		imageFiles := make([]ImageFile, 0, len(imageFileSet))
		for _, path := range paths {
			imageFiles = append(imageFiles, imageFileSet[path] /* Guaranteed to exist */)
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
func NewImageForProto(protoImage *imagev1.Image, options ...NewImageForProtoOption) (Image, error) {
	var newImageOptions newImageForProtoOptions
	for _, option := range options {
		option(&newImageOptions)
	}
	if newImageOptions.noReparse && newImageOptions.computeUnusedImports {
		return nil, fmt.Errorf("cannot use both WithNoReparse and WithComputeUnusedImports options; they are mutually exclusive")
	}
	if !newImageOptions.noReparse {
		if err := reparseImageProto(protoImage, newImageOptions.computeUnusedImports); err != nil {
			return nil, err
		}
	}
	if err := validateProtoImage(protoImage); err != nil {
		return nil, err
	}
	imageFiles := make([]ImageFile, len(protoImage.File))
	for i, protoImageFile := range protoImage.File {
		var isImport bool
		var isSyntaxUnspecified bool
		var unusedDependencyIndexes []int32
		var moduleFullName bufmodule.ModuleFullName
		var commitID string
		var err error
		if protoImageFileExtension := protoImageFile.GetBufExtension(); protoImageFileExtension != nil {
			isImport = protoImageFileExtension.GetIsImport()
			isSyntaxUnspecified = protoImageFileExtension.GetIsSyntaxUnspecified()
			unusedDependencyIndexes = protoImageFileExtension.GetUnusedDependency()
			if protoModuleInfo := protoImageFileExtension.GetModuleInfo(); protoModuleInfo != nil {
				if protoModuleName := protoModuleInfo.GetName(); protoModuleName != nil {
					moduleFullName, err = bufmodule.NewModuleFullName(
						protoModuleName.GetRemote(),
						protoModuleName.GetOwner(),
						protoModuleName.GetRepository(),
					)
					if err != nil {
						return nil, err
					}
					// we only want to set this if there is a module name
					commitID = protoModuleInfo.GetCommit()
				}
			}
		}
		imageFile, err := NewImageFile(
			protoImageFile,
			moduleFullName,
			commitID,
			protoImageFile.GetName(),
			isImport,
			isSyntaxUnspecified,
			unusedDependencyIndexes,
		)
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
func NewImageForCodeGeneratorRequest(request *pluginpb.CodeGeneratorRequest, options ...NewImageForProtoOption) (Image, error) {
	if err := protodescriptor.ValidateCodeGeneratorRequestExceptFileDescriptorProtos(request); err != nil {
		return nil, err
	}
	protoImageFiles := make([]*imagev1.ImageFile, len(request.GetProtoFile()))
	for i, fileDescriptorProto := range request.GetProtoFile() {
		// we filter whether something is an import or not in ImageWithOnlyPaths
		// we cannot determine if the syntax was unset
		protoImageFiles[i] = fileDescriptorProtoToProtoImageFile(fileDescriptorProto, false, false, nil, nil, "")
	}
	image, err := NewImageForProto(
		&imagev1.Image{
			File: protoImageFiles,
		},
		options...,
	)
	if err != nil {
		return nil, err
	}
	return ImageWithOnlyPaths(
		image,
		request.GetFileToGenerate(),
		nil,
	)
}

// NewImageForProtoOption is an option for use with NewImageForProto.
type NewImageForProtoOption func(*newImageForProtoOptions)

// WithNoReparse instructs NewImageForProto to skip the reparse step. The reparse
// step is usually needed when unmarshalling the image from bytes. It reconstitutes
// custom options, from unrecognized bytes to known extension fields.
func WithNoReparse() NewImageForProtoOption {
	return func(options *newImageForProtoOptions) {
		options.noReparse = true
	}
}

// WithUnusedImportsComputation instructs NewImageForProto to compute unused imports
// for the files. These are usually computed by the compiler and stored in the image.
// But some sources of images may not include this information, so this option can be
// used to ensure that information is present in the image and accurate.
//
// This option is NOT compatible with WithNoReparse: the image must be re-parsed for
// there to be adequate information for computing unused imports.
func WithUnusedImportsComputation() NewImageForProtoOption {
	return func(options *newImageForProtoOptions) {
		options.computeUnusedImports = true
	}
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
	excludePaths []string,
) (Image, error) {
	return imageWithOnlyPaths(image, paths, excludePaths, false)
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
	excludePaths []string,
) (Image, error) {
	return imageWithOnlyPaths(image, paths, excludePaths, true)
}

// ImageByDir returns multiple images that have non-imports split
// by directory.
//
// That is, each Image will only contain a single directory's files
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
	// we need this to produce a deterministic order of the returned Images
	dirs := make([]string, 0, len(dirToPaths))
	for dir := range dirToPaths {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	newImages := make([]Image, 0, len(dirToPaths))
	for _, dir := range dirs {
		paths, ok := dirToPaths[dir]
		if !ok {
			// this should never happen
			return nil, fmt.Errorf("no dir for %q in dirToPaths", dir)
		}
		newImage, err := ImageWithOnlyPaths(image, paths, nil)
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
		File: make([]*imagev1.ImageFile, len(imageFiles)),
	}
	for i, imageFile := range imageFiles {
		protoImage.File[i] = imageFileToProtoImageFile(imageFile)
	}
	return protoImage
}

// ImageToFileDescriptorSet returns a new FileDescriptorSet for the Image.
func ImageToFileDescriptorSet(image Image) *descriptorpb.FileDescriptorSet {
	return protodescriptor.FileDescriptorSetForFileDescriptors(ImageToFileDescriptorProtos(image)...)
}

// ImageToFileDescriptorProtos returns the FileDescriptorProtos for the Image.
func ImageToFileDescriptorProtos(image Image) []*descriptorpb.FileDescriptorProto {
	return imageFilesToFileDescriptorProtos(image.Files())
}

// ImageToCodeGeneratorRequest returns a new CodeGeneratorRequest for the Image.
//
// All non-imports are added as files to generate.
// If includeImports is set, all non-well-known-type imports are also added as files to generate.
// If includeWellKnownTypes is set, well-known-type imports are also added as files to generate.
// includeWellKnownTypes has no effect if includeImports is not set.
func ImageToCodeGeneratorRequest(
	image Image,
	parameter string,
	compilerVersion *pluginpb.Version,
	includeImports bool,
	includeWellKnownTypes bool,
) *pluginpb.CodeGeneratorRequest {
	return imageToCodeGeneratorRequest(
		image,
		parameter,
		compilerVersion,
		includeImports,
		includeWellKnownTypes,
		nil,
		nil,
	)
}

// ImagesToCodeGeneratorRequests converts the Images to CodeGeneratorRequests.
//
// All non-imports are added as files to generate.
// If includeImports is set, all non-well-known-type imports are also added as files to generate.
// If includeImports is set, only one CodeGeneratorRequest will contain any given file as a FileToGenerate.
// If includeWellKnownTypes is set, well-known-type imports are also added as files to generate.
// includeWellKnownTypes has no effect if includeImports is not set.
func ImagesToCodeGeneratorRequests(
	images []Image,
	parameter string,
	compilerVersion *pluginpb.Version,
	includeImports bool,
	includeWellKnownTypes bool,
) []*pluginpb.CodeGeneratorRequest {
	requests := make([]*pluginpb.CodeGeneratorRequest, len(images))
	// alreadyUsedPaths is a map of paths that have already been added to an image.
	//
	// We track this if includeImports is set, so that when we find an import, we can
	// see if the import was already added to a CodeGeneratorRequest via another Image
	// in the Image slice. If the import was already added, we do not add duplicates
	// across CodeGeneratorRequests.
	var alreadyUsedPaths map[string]struct{}
	// nonImportPaths is a map of non-import paths.
	//
	// We track this if includeImports is set. If we find a non-import file in Image A
	// and this file is an import in Image B, the file will have already been added to
	// a CodeGeneratorRequest via Image A, so do not add the duplicate to any other
	// CodeGeneratorRequest.
	var nonImportPaths map[string]struct{}
	if includeImports {
		// We don't need to track these if includeImports is false, so we only populate
		// the maps if includeImports is true. If includeImports is false, only non-imports
		// will be added to each CodeGeneratorRequest, so figuring out whether or not
		// we should add a given import to a given CodeGeneratorRequest is unnecessary.
		//
		// imageToCodeGeneratorRequest checks if these maps are nil before every access.
		alreadyUsedPaths = make(map[string]struct{})
		nonImportPaths = make(map[string]struct{})
		for _, image := range images {
			for _, imageFile := range image.Files() {
				if !imageFile.IsImport() {
					nonImportPaths[imageFile.Path()] = struct{}{}
				}
			}
		}
	}
	for i, image := range images {
		requests[i] = imageToCodeGeneratorRequest(
			image,
			parameter,
			compilerVersion,
			includeImports,
			includeWellKnownTypes,
			alreadyUsedPaths,
			nonImportPaths,
		)
	}
	return requests
}

type newImageForProtoOptions struct {
	noReparse            bool
	computeUnusedImports bool
}

func reparseImageProto(protoImage *imagev1.Image, computeUnusedImports bool) error {
	// TODO right now, NewResolver sets AllowUnresolvable to true all the time
	// we want to make this into a check, and we verify if we need this for the individual command
	resolver := protoencoding.NewLazyResolver(protoImage.File...)
	if err := protoencoding.ReparseUnrecognized(resolver, protoImage.ProtoReflect()); err != nil {
		return fmt.Errorf("could not reparse image: %v", err)
	}
	if computeUnusedImports {
		tracker := &importTracker{
			resolver: resolver,
			used:     map[string]map[string]struct{}{},
		}
		tracker.findUsedImports(protoImage)
		// Now we can populated list of unused dependencies
		for _, file := range protoImage.File {
			bufExt := file.BufExtension
			if bufExt == nil {
				bufExt = &imagev1.ImageFileExtension{}
				file.BufExtension = bufExt
			}
			bufExt.UnusedDependency = nil // reset
			usedImports := tracker.used[file.GetName()]
			for i, dep := range file.Dependency {
				if _, ok := usedImports[dep]; !ok {
					// it's fine if it's public
					isPublic := false
					for _, publicDepIndex := range file.PublicDependency {
						if i == int(publicDepIndex) {
							isPublic = true
							break
						}
					}
					if !isPublic {
						bufExt.UnusedDependency = append(bufExt.UnusedDependency, int32(i))
					}
				}
			}
		}
	}
	return nil
}

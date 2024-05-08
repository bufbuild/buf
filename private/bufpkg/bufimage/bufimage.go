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
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/gen/data/datawkt"
	imagev1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/image/v1"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/protodescriptor"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/gofrs/uuid/v5"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// ImageFileInfo is the minimal interface that can be fulfilled by both an ImageFile
// and (with conversion) a bufmodule.FileInfo.
//
// This is used by ls-files.
type ImageFileInfo interface {
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
	// May be empty, that is CommitID().IsNil() may be true. Callers should not rely on this
	// value being present. If ModuleFullName is nil, this will always be empty.
	CommitID() uuid.UUID
	// Imports returns the imports for this ImageFile.
	Imports() ([]string, error)
	// IsImport returns true if this file is an import.
	IsImport() bool

	isImageFileInfo()
}

// ImageFileInfoForModuleFileInfo returns a new ImageFileInfo for the bufmodule.FileInfo.
func ImageFileInfoForModuleFileInfo(moduleFileInfo bufmodule.FileInfo) ImageFileInfo {
	return newModuleImageFileInfo(moduleFileInfo)
}

// AppendWellKnownTypeImageFileInfos appends any Well-Known Types that are not already present
// in the input ImageFileInfos.
//
// For example, if imageFileInfos contains "google/protobuf/timestamp.proto", the returned
// ImageFileInfos will have all the Well-Known Types except for "google/protobuf/timestamp.proto"
// appended.
//
// This function uses the input wktBucket to determine what is a Well-Known Type.
// This bucket should contain all the Well-Known Types, and nothing else. This is used instead
// of using datawkt directly so that we can pass in a bucket backed by a cache on-disk with
// the Well-Known Types, and use its LocalPath information.
//
// The appended Well-Known Types will be in sorted order by path, and will all be marked as imports.
func AppendWellKnownTypeImageFileInfos(
	ctx context.Context,
	wktBucket storage.ReadBucket,
	imageFileInfos []ImageFileInfo,
) ([]ImageFileInfo, error) {
	pathToImageFileInfo, err := slicesext.ToUniqueValuesMap(imageFileInfos, ImageFileInfo.Path)
	if err != nil {
		return nil, err
	}
	return appendWellKnownTypeImageFileInfos(ctx, wktBucket, imageFileInfos, pathToImageFileInfo)
}

// ImageFileInfosWithOnlyTargetsAndTargetImports returns a new slice of ImageFileInfos that only
// contains the non-imports (ie targets), and the files that those non-imports themselves
// transitively import.
//
// This is used in ls-files.
//
// As an example, assume a module has files a.proto, b.proto, and it has a dependency on
// a module with files c.proto, d.proto. a.proto imports c.proto. We only target the module
// with a.proto, b.proto. The resulting slice should have a.proto, b.proto, c.proto, but not
// d.proto.
//
// It is assumed that the input ImageFileInfos are self-contained, that is every import should
// be contained within the input, except for the Well-Known Types. If a Well-Known Type is imported
// and not present in the input, an ImageFileInfo for the Well-Known Type is automatically added
// to the result from the given bucket.
//
// The result will be sorted by path.
func ImageFileInfosWithOnlyTargetsAndTargetImports(
	ctx context.Context,
	wktBucket storage.ReadBucket,
	imageFileInfos []ImageFileInfo,
) ([]ImageFileInfo, error) {
	pathToImageFileInfo, err := slicesext.ToUniqueValuesMap(imageFileInfos, ImageFileInfo.Path)
	if err != nil {
		return nil, err
	}
	imageFileInfos, err = appendWellKnownTypeImageFileInfos(ctx, wktBucket, imageFileInfos, pathToImageFileInfo)
	if err != nil {
		return nil, err
	}
	resultPaths := make(map[string]struct{}, len(imageFileInfos))
	for _, imageFileInfo := range imageFileInfos {
		if imageFileInfo.IsImport() {
			continue
		}
		if err := imageFileInfosWithOnlyTargetsAndTargetImportsRec(imageFileInfo, pathToImageFileInfo, resultPaths); err != nil {
			return nil, err
		}
	}
	resultImageFileInfos := make([]ImageFileInfo, 0, len(resultPaths))
	for resultPath := range resultPaths {
		imageFileInfo, ok := pathToImageFileInfo[resultPath]
		if !ok {
			return nil, fmt.Errorf("no ImageFileInfo for path %q", resultPath)
		}
		resultImageFileInfos = append(resultImageFileInfos, imageFileInfo)
	}
	sort.Slice(
		resultImageFileInfos,
		func(i int, j int) bool {
			return resultImageFileInfos[i].Path() < resultImageFileInfos[j].Path()
		},
	)
	return resultImageFileInfos, nil
}

// ImageFile is a Protobuf file within an image.
type ImageFile interface {
	ImageFileInfo

	// FileDescriptorProto is the backing *descriptorpb.FileDescriptorProto for this File.
	//
	// This will never be nil.
	// The value Path() is equal to FileDescriptorProto().GetName() .
	FileDescriptorProto() *descriptorpb.FileDescriptorProto
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
// TODO FUTURE: moduleFullName and commitID should be options since they are optional.
func NewImageFile(
	fileDescriptor protodescriptor.FileDescriptor,
	moduleFullName bufmodule.ModuleFullName,
	commitID uuid.UUID,
	externalPath string,
	localPath string,
	isImport bool,
	isSyntaxUnspecified bool,
	unusedDependencyIndexes []int32,
) (ImageFile, error) {
	return newImageFile(
		fileDescriptor,
		moduleFullName,
		commitID,
		externalPath,
		localPath,
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
		imageFile.LocalPath(),
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
	// Resolver returns a resolver backed by this image.
	Resolver() protoencoding.Resolver

	isImage()
}

// NewImage returns a new Image for the given ImageFiles.
//
// The input ImageFiles are expected to be in correct DAG order!
// TODO FUTURE: Consider checking the above, and if not, reordering the Files.
// If imageFiles is empty, returns error
func NewImage(imageFiles []ImageFile) (Image, error) {
	return newImage(imageFiles, false, nil)
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

// CloneImage returns a deep copy of the given image.
func CloneImage(image Image) (Image, error) {
	originalFiles := image.Files()
	imageFiles := make([]ImageFile, len(originalFiles))
	for i, originalFile := range originalFiles {
		clonedFile, err := CloneImageFile(originalFile)
		if err != nil {
			return nil, err
		}
		imageFiles[i] = clonedFile
	}
	return NewImage(imageFiles)
}

// CloneImageFile returns a deep copy of the given image file.
func CloneImageFile(imageFile ImageFile) (ImageFile, error) {
	clonedProto := proto.Clone(imageFile.FileDescriptorProto())
	clonedDescriptor, ok := clonedProto.(*descriptorpb.FileDescriptorProto)
	if !ok {
		// Shouldn't actually be possible...
		return nil, fmt.Errorf("failed to clone image file %q: input %T; clone is %T but expecting %T",
			imageFile.Path(), imageFile, clonedProto, (*descriptorpb.FileDescriptorProto)(nil))
	}
	originalUnusedDeps := imageFile.UnusedDependencyIndexes()
	unusedDeps := make([]int32, len(originalUnusedDeps))
	copy(unusedDeps, originalUnusedDeps)
	// The other attributes are already immutable, so we don't need to copy them.
	return NewImageFile(
		clonedDescriptor,
		imageFile.ModuleFullName(),
		imageFile.CommitID(),
		imageFile.ExternalPath(),
		imageFile.LocalPath(),
		imageFile.IsImport(),
		imageFile.IsSyntaxUnspecified(),
		unusedDeps,
	)
}

// NewImageForProto returns a new Image for the given proto Image.
//
// The input Files are expected to be in correct DAG order!
// TODO FUTURE: Consider checking the above, and if not, reordering the Files.
//
// TODO FUTURE: do we want to add the ability to do external path resolution here?
func NewImageForProto(protoImage *imagev1.Image, options ...NewImageForProtoOption) (Image, error) {
	var newImageOptions newImageForProtoOptions
	for _, option := range options {
		option(&newImageOptions)
	}
	if newImageOptions.noReparse && newImageOptions.computeUnusedImports {
		return nil, fmt.Errorf("cannot use both WithNoReparse and WithComputeUnusedImports options; they are mutually exclusive")
	}
	// TODO FUTURE: right now, NewResolver sets AllowUnresolvable to true all the time
	// we want to make this into a check, and we verify if we need this for the individual command
	resolver := protoencoding.NewLazyResolver(protoImage.File...)
	if !newImageOptions.noReparse {
		if err := reparseImageProto(protoImage, resolver, newImageOptions.computeUnusedImports); err != nil {
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
		var commitID uuid.UUID
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
					if protoCommitID := protoModuleInfo.GetCommit(); protoCommitID != "" {
						commitID, err = uuidutil.FromDashless(protoCommitID)
						if err != nil {
							return nil, err
						}
					}
				}
			}
		}
		imageFile, err := NewImageFile(
			protoImageFile,
			moduleFullName,
			commitID,
			protoImageFile.GetName(),
			"",
			isImport,
			isSyntaxUnspecified,
			unusedDependencyIndexes,
		)
		if err != nil {
			return nil, err
		}
		imageFiles[i] = imageFile
	}
	return newImage(imageFiles, false, resolver)
}

// NewImageForCodeGeneratorRequest returns a new Image from a given CodeGeneratorRequest.
//
// The input Files are expected to be in correct DAG order!
// TODO FUTURE: Consider checking the above, and if not, reordering the Files.
func NewImageForCodeGeneratorRequest(request *pluginpb.CodeGeneratorRequest, options ...NewImageForProtoOption) (Image, error) {
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
	return newImageNoValidate(newImageFiles, image.Resolver())
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
func ImageToProtoImage(image Image) (*imagev1.Image, error) {
	imageFiles := image.Files()
	protoImage := &imagev1.Image{
		File: make([]*imagev1.ImageFile, len(imageFiles)),
	}
	for i, imageFile := range imageFiles {
		protoImageFile, err := imageFileToProtoImageFile(imageFile)
		if err != nil {
			return nil, err
		}
		protoImage.File[i] = protoImageFile
	}
	return protoImage, nil
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
) (*pluginpb.CodeGeneratorRequest, error) {
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
) ([]*pluginpb.CodeGeneratorRequest, error) {
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
		var err error
		requests[i], err = imageToCodeGeneratorRequest(
			image,
			parameter,
			compilerVersion,
			includeImports,
			includeWellKnownTypes,
			alreadyUsedPaths,
			nonImportPaths,
		)
		if err != nil {
			return nil, err
		}
	}
	return requests, nil
}

type newImageForProtoOptions struct {
	noReparse            bool
	computeUnusedImports bool
}

func reparseImageProto(protoImage *imagev1.Image, resolver protoencoding.Resolver, computeUnusedImports bool) error {
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

// We pass in the pathToImageFileInfo here because we also call this in
// ImageFileInfosWithOnlyTargetsAndTargetImports and we don't want to have to make this map twice.
//
// This also modifies the pathToImageFileInfo map if a Well-Known Type is added.
func appendWellKnownTypeImageFileInfos(
	ctx context.Context,
	wktBucket storage.ReadBucket,
	imageFileInfos []ImageFileInfo,
	pathToImageFileInfo map[string]ImageFileInfo,
) ([]ImageFileInfo, error) {
	// Sorted.
	wktObjectInfos, err := storage.AllObjectInfos(ctx, wktBucket, "")
	if err != nil {
		return nil, err
	}
	wktPaths := slicesext.Map(wktObjectInfos, storage.ObjectInfo.Path)
	if !slicesext.Equal(datawkt.AllFilePaths, wktPaths) {
		return nil, syserror.Newf("wktBucket paths %s are not equal to datawkt.AllFilePaths %s", strings.Join(wktPaths, ","), strings.Join(datawkt.AllFilePaths, ","))
	}
	resultImageFileInfos := slicesext.Copy(imageFileInfos)
	for _, wktObjectInfo := range wktObjectInfos {
		if _, ok := pathToImageFileInfo[wktObjectInfo.Path()]; !ok {
			fileImports, ok := datawkt.FileImports(wktObjectInfo.Path())
			if !ok {
				return nil, syserror.Newf("datawkt.FileImports returned false for wkt %s", wktObjectInfo.Path())
			}
			imageFileInfo := newWellKnownTypeImageFileInfo(wktObjectInfo, fileImports, true)
			resultImageFileInfos = append(resultImageFileInfos, imageFileInfo)
			pathToImageFileInfo[wktObjectInfo.Path()] = imageFileInfo
		}
	}
	return resultImageFileInfos, nil
}

func imageFileInfosWithOnlyTargetsAndTargetImportsRec(
	imageFileInfo ImageFileInfo,
	pathToImageFileInfo map[string]ImageFileInfo,
	resultPaths map[string]struct{},
) error {
	path := imageFileInfo.Path()
	if _, ok := resultPaths[path]; ok {
		return nil
	}
	resultPaths[path] = struct{}{}
	imports, err := imageFileInfo.Imports()
	if err != nil {
		return err
	}
	for _, imp := range imports {
		importImageFileInfo, ok := pathToImageFileInfo[imp]
		if !ok {
			return fmt.Errorf("%s: import %q: %w", imageFileInfo.ExternalPath(), imp, fs.ErrNotExist)
		}
		if err := imageFileInfosWithOnlyTargetsAndTargetImportsRec(importImageFileInfo, pathToImageFileInfo, resultPaths); err != nil {
			return err
		}
	}
	return nil
}

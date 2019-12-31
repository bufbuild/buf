package extimage

import (
	"errors"
	"fmt"
	"sort"

	imagev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/image/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/ext/extdescriptor"
	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin_go "github.com/golang/protobuf/protoc-gen-go/plugin"
)

// ValidateImage validates an Image.
func ValidateImage(image *imagev1beta1.Image) error {
	if image == nil {
		return errors.New("validate error: nil Image")
	}
	if err := extdescriptor.ValidateFileDescriptorProtos(image.File); err != nil {
		return err
	}

	if image.BufbuildImageExtension != nil {
		seenFileIndexes := make(map[uint32]struct{}, len(image.BufbuildImageExtension.ImageImportRefs))
		for _, imageImportRef := range image.BufbuildImageExtension.ImageImportRefs {
			if imageImportRef == nil {
				return errors.New("validate error: nil ImageImportRef")
			}
			if imageImportRef.FileIndex == nil {
				return errors.New("validate error: nil ImageImportRef.FileIndex")
			}
			fileIndex := *imageImportRef.FileIndex
			if fileIndex >= uint32(len(image.File)) {
				return fmt.Errorf("validate error: invalid file index: %d", fileIndex)
			}
			if _, ok := seenFileIndexes[fileIndex]; ok {
				return fmt.Errorf("validate error: duplicate file index: %d", fileIndex)
			}
			seenFileIndexes[fileIndex] = struct{}{}
		}
	}
	return nil
}

// ImageImportNames returns the sorted import names.
//
// Validates the input.
func ImageImportNames(image *imagev1beta1.Image) ([]string, error) {
	if err := ValidateImage(image); err != nil {
		return nil, err
	}
	imageImportRefs := image.GetBufbuildImageExtension().GetImageImportRefs()
	if len(imageImportRefs) == 0 {
		return nil, nil
	}
	importFileIndexes := make(map[int]struct{}, len(imageImportRefs))
	for _, imageImportRef := range imageImportRefs {
		if imageImportRef.FileIndex == nil {
			// this should have been caught in validation but just in case
			return nil, errors.New("nil fileIndex")
		}
		importFileIndexes[int(imageImportRef.GetFileIndex())] = struct{}{}
	}
	importNames := make([]string, 0, len(importFileIndexes))
	for importFileIndex := range importFileIndexes {
		importNames = append(importNames, image.File[importFileIndex].GetName())
	}
	sort.Strings(importNames)
	return importNames, nil
}

// ImageWithoutImports returns a copy of the Image without imports.
//
// If GetBufbuildImageExtension() is nil, returns the original Image.
// If there are no imports, returns the original Image.
//
// Backing FileDescriptorProtos are not copied, only the references are copied.
// This will result in unknown fields being dropped from the backing Image, but not
// the backing FileDescriptorProtos.
//
// Validates the input and output.
func ImageWithoutImports(image *imagev1beta1.Image) (*imagev1beta1.Image, error) {
	if err := ValidateImage(image); err != nil {
		return nil, err
	}
	imageImportRefs := image.GetBufbuildImageExtension().GetImageImportRefs()
	// If no modifications would be made, then we return the original
	if len(imageImportRefs) == 0 {
		return image, nil
	}

	newImage := &imagev1beta1.Image{
		BufbuildImageExtension: &imagev1beta1.ImageExtension{
			ImageImportRefs: make([]*imagev1beta1.ImageImportRef, 0),
		},
	}
	importFileIndexes := make(map[int]struct{}, len(imageImportRefs))
	for _, imageImportRef := range imageImportRefs {
		if imageImportRef.FileIndex == nil {
			// this should have been caught in validation but just in case
			return nil, errors.New("nil fileIndex")
		}
		importFileIndexes[int(imageImportRef.GetFileIndex())] = struct{}{}
	}

	for i, file := range image.File {
		if _, isImport := importFileIndexes[i]; isImport {
			continue
		}
		newImage.File = append(newImage.File, file)
	}

	if len(newImage.File) == 0 {
		return nil, fmt.Errorf("no input files after stripping imports")
	}
	if err := ValidateImage(newImage); err != nil {
		return nil, err
	}
	return newImage, nil
}

// ImageWithSpecificNames returns a copy of the Image with only the Files with the given names.
//
// Names are normalized and validated.
// If allowNotExist is false, the specific names must exist on the input image.
// Backing FileDescriptorProtos are not copied, only the references are copied.
//
// Validates the input and output.
func ImageWithSpecificNames(
	image *imagev1beta1.Image,
	allowNotExist bool,
	specificNames ...string,
) (*imagev1beta1.Image, error) {
	if err := ValidateImage(image); err != nil {
		return nil, err
	}
	// If no modifications would be made, then we return the original
	if len(specificNames) == 0 {
		return image, nil
	}

	newImage := &imagev1beta1.Image{
		BufbuildImageExtension: &imagev1beta1.ImageExtension{
			ImageImportRefs: make([]*imagev1beta1.ImageImportRef, 0),
		},
	}

	imageImportRefs := image.GetBufbuildImageExtension().GetImageImportRefs()
	importFileIndexes := make(map[int]struct{}, len(imageImportRefs))
	for _, imageImportRef := range imageImportRefs {
		if imageImportRef.FileIndex == nil {
			// this should have been caught in validation but just in case
			return nil, errors.New("nil fileIndex")
		}
		importFileIndexes[int(imageImportRef.GetFileIndex())] = struct{}{}
	}

	specificNamesMap := make(map[string]struct{}, len(specificNames))
	for _, specificName := range specificNames {
		normalizedName, err := storagepath.NormalizeAndValidate(specificName)
		if err != nil {
			return nil, err
		}
		specificNamesMap[normalizedName] = struct{}{}
	}

	if !allowNotExist {
		allNamesMap := make(map[string]struct{}, len(image.File))
		for _, file := range image.File {
			allNamesMap[file.GetName()] = struct{}{}
		}
		for specificName := range specificNamesMap {
			if _, ok := allNamesMap[specificName]; !ok {
				return nil, fmt.Errorf("%s is not present in the Image", specificName)
			}
		}
	}

	for i, file := range image.File {
		// we already know that file.GetName() is normalized and validated from validation
		if _, add := specificNamesMap[file.GetName()]; !add {
			continue
		}
		newImage.File = append(newImage.File, file)
		if _, isImport := importFileIndexes[i]; isImport {
			fileIndex := uint32(len(newImage.File) - 1)
			newImage.BufbuildImageExtension.ImageImportRefs = append(
				newImage.BufbuildImageExtension.ImageImportRefs,
				&imagev1beta1.ImageImportRef{
					FileIndex: proto.Uint32(fileIndex),
				},
			)
		}
	}
	if len(newImage.File) == 0 {
		return nil, errors.New("no input files match the given names")
	}
	if err := ValidateImage(newImage); err != nil {
		return nil, err
	}
	return newImage, nil
}

// ImageToFileDescriptorSet converts the Image to a native FileDescriptorSet.
//
// This strips the backing ImageExtension.
//
// Backing FileDescriptorProtos are not copied, only the references are copied.
// This will result in unknown fields being dropped from the backing FileDescriptorSet, but not
// the backing FileDescriptorProtos.
//
// Validates the input and output.
func ImageToFileDescriptorSet(image *imagev1beta1.Image) (*descriptor.FileDescriptorSet, error) {
	if err := ValidateImage(image); err != nil {
		return nil, err
	}
	fileDescriptorSet := &descriptor.FileDescriptorSet{
		File: image.File,
	}
	if err := extdescriptor.ValidateFileDescriptorSet(fileDescriptorSet); err != nil {
		return nil, err
	}
	return fileDescriptorSet, nil
}

// ImageToCodeGeneratorRequest converts the Image to a CodeGeneratorRequest.
//
// The files to generate must be within the Image.
// Files to generate are normalized and validated.
//
// Validates the input.
func ImageToCodeGeneratorRequest(
	image *imagev1beta1.Image,
	parameter string,
	fileToGenerate ...string,
) (*plugin_go.CodeGeneratorRequest, error) {
	if err := ValidateImage(image); err != nil {
		return nil, err
	}
	if len(fileToGenerate) == 0 {
		return nil, fmt.Errorf("no file to generate")
	}
	normalizedFileToGenerate := make([]string, len(fileToGenerate))
	for i, elem := range fileToGenerate {
		normalized, err := storagepath.NormalizeAndValidate(elem)
		if err != nil {
			return nil, err
		}
		normalizedFileToGenerate[i] = normalized
	}
	namesMap := make(map[string]struct{}, len(image.File))
	for _, file := range image.File {
		namesMap[file.GetName()] = struct{}{}
	}
	for _, normalized := range normalizedFileToGenerate {
		if _, ok := namesMap[normalized]; !ok {
			return nil, fmt.Errorf("file to generate %q is not within the image", normalized)
		}
	}
	var parameterPtr *string
	if parameter != "" {
		parameterPtr = proto.String(parameter)
	}
	return &plugin_go.CodeGeneratorRequest{
		FileToGenerate: normalizedFileToGenerate,
		Parameter:      parameterPtr,
		ProtoFile:      image.File,
	}, nil
}

// CodeGeneratorRequestToImage converts the CodeGeneratorRequest to an Image.
//
// Validates the output.
func CodeGeneratorRequestToImage(request *plugin_go.CodeGeneratorRequest) (*imagev1beta1.Image, error) {
	image := &imagev1beta1.Image{
		File: request.GetProtoFile(),
	}
	return ImageWithSpecificNames(image, false, request.FileToGenerate...)
}

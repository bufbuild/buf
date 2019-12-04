package bufpb

import (
	"bytes"
	"sort"

	"github.com/bufbuild/buf/internal/buf/buferrs"
	imagev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/image/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/protodescpb"
	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin_go "github.com/golang/protobuf/protoc-gen-go/plugin"
)

var (
	jsonMarshaler       = &jsonpb.Marshaler{}
	jsonIndentMarshaler = &jsonpb.Marshaler{
		Indent: "  ",
	}
)

type image struct {
	backing *imagev1beta1.Image
}

func newImage(backing *imagev1beta1.Image) (*image, error) {
	image := &image{
		backing: backing,
	}
	if err := image.validate(); err != nil {
		return nil, err
	}
	return image, nil
}

func (f *image) GetFile() []protodescpb.FileDescriptor {
	files := make([]protodescpb.FileDescriptor, len(f.backing.File))
	for i, file := range f.backing.File {
		files[i] = file
	}
	return files
}

func (f *image) MarshalWire() ([]byte, error) {
	return proto.Marshal(f.backing)
}

func (f *image) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	if err := jsonMarshaler.Marshal(buffer, f.backing); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (f *image) MarshalJSONIndent() ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	if err := jsonIndentMarshaler.Marshal(buffer, f.backing); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (f *image) MarshalText() ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	if err := proto.MarshalText(buffer, f.backing); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (f *image) GetBufbuildImageExtension() *imagev1beta1.ImageExtension {
	return f.backing.GetBufbuildImageExtension()
}

func (f *image) ImportNames() ([]string, error) {
	imageImportRefs := f.backing.GetBufbuildImageExtension().GetImageImportRefs()
	if len(imageImportRefs) == 0 {
		return nil, nil
	}
	importFileIndexes := make(map[int]struct{}, len(imageImportRefs))
	for _, imageImportRef := range imageImportRefs {
		if imageImportRef.FileIndex == nil {
			// this should have been caught in validation but just in case
			return nil, buferrs.NewSystemError("nil fileIndex")
		}
		importFileIndexes[int(imageImportRef.GetFileIndex())] = struct{}{}
	}
	importNames := make([]string, 0, len(importFileIndexes))
	for importFileIndex := range importFileIndexes {
		importNames = append(importNames, f.backing.File[importFileIndex].GetName())
	}
	sort.Strings(importNames)
	return importNames, nil
}

func (f *image) WithoutImports() (Image, error) {
	imageImportRefs := f.backing.GetBufbuildImageExtension().GetImageImportRefs()
	// If no modifications would be made, then we return the original
	if len(imageImportRefs) == 0 {
		return f, nil
	}

	newBacking := &imagev1beta1.Image{
		BufbuildImageExtension: &imagev1beta1.ImageExtension{
			ImageImportRefs: make([]*imagev1beta1.ImageImportRef, 0),
		},
	}
	importFileIndexes := make(map[int]struct{}, len(imageImportRefs))
	for _, imageImportRef := range imageImportRefs {
		if imageImportRef.FileIndex == nil {
			// this should have been caught in validation but just in case
			return nil, buferrs.NewSystemError("nil fileIndex")
		}
		importFileIndexes[int(imageImportRef.GetFileIndex())] = struct{}{}
	}

	for i, file := range f.backing.File {
		if _, isImport := importFileIndexes[i]; isImport {
			continue
		}
		newBacking.File = append(newBacking.File, file)
	}

	if len(newBacking.File) == 0 {
		return nil, buferrs.NewUserErrorf("no input files after stripping imports")
	}
	return newImage(newBacking)
}

func (f *image) WithSpecificNames(allowNotExist bool, specificNames ...string) (Image, error) {
	// If no modifications would be made, then we return the original
	if len(specificNames) == 0 {
		return f, nil
	}

	newBacking := &imagev1beta1.Image{
		BufbuildImageExtension: &imagev1beta1.ImageExtension{
			ImageImportRefs: make([]*imagev1beta1.ImageImportRef, 0),
		},
	}

	imageImportRefs := f.backing.GetBufbuildImageExtension().GetImageImportRefs()
	importFileIndexes := make(map[int]struct{}, len(imageImportRefs))
	for _, imageImportRef := range imageImportRefs {
		if imageImportRef.FileIndex == nil {
			// this should have been caught in validation but just in case
			return nil, buferrs.NewSystemError("nil fileIndex")
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
		allNamesMap := make(map[string]struct{}, len(f.backing.File))
		for _, file := range f.backing.File {
			allNamesMap[file.GetName()] = struct{}{}
		}
		for specificName := range specificNamesMap {
			if _, ok := allNamesMap[specificName]; !ok {
				return nil, buferrs.NewUserErrorf("%s is not present in the Image", specificName)
			}
		}
	}

	for i, file := range f.backing.File {
		// we already know that file.GetName() is normalized and validated from validation
		if _, add := specificNamesMap[file.GetName()]; !add {
			continue
		}
		newBacking.File = append(newBacking.File, file)
		if _, isImport := importFileIndexes[i]; isImport {
			fileIndex := uint32(len(newBacking.File) - 1)
			newBacking.BufbuildImageExtension.ImageImportRefs = append(
				newBacking.BufbuildImageExtension.ImageImportRefs,
				&imagev1beta1.ImageImportRef{
					FileIndex: protodescpb.Uint32(fileIndex),
				},
			)
		}
	}
	if len(newBacking.File) == 0 {
		return nil, buferrs.NewUserError("no input files match the given names")
	}
	return newImage(newBacking)
}

func (f *image) ToFileDescriptorSet() (protodescpb.FileDescriptorSet, error) {
	nativeFileDescriptorSet := &descriptor.FileDescriptorSet{
		File: f.backing.File,
	}
	return protodescpb.NewFileDescriptorSet(nativeFileDescriptorSet)
}

func (f *image) ToCodeGeneratorRequest(parameter string, fileToGenerate ...string) (*plugin_go.CodeGeneratorRequest, error) {
	if len(fileToGenerate) == 0 {
		return nil, buferrs.NewUserErrorf("no file to generate")
	}
	normalizedFileToGenerate := make([]string, len(fileToGenerate))
	for i, elem := range fileToGenerate {
		normalized, err := storagepath.NormalizeAndValidate(elem)
		if err != nil {
			return nil, err
		}
		normalizedFileToGenerate[i] = normalized
	}
	namesMap := make(map[string]struct{}, len(f.backing.File))
	for _, file := range f.backing.File {
		namesMap[file.GetName()] = struct{}{}
	}
	for _, normalized := range normalizedFileToGenerate {
		if _, ok := namesMap[normalized]; !ok {
			return nil, buferrs.NewUserErrorf("file to generate %q is not within the image", normalized)
		}
	}
	var parameterPtr *string
	if parameter != "" {
		parameterPtr = protodescpb.String(parameter)
	}
	return &plugin_go.CodeGeneratorRequest{
		FileToGenerate: normalizedFileToGenerate,
		Parameter:      parameterPtr,
		ProtoFile:      f.backing.File,
	}, nil
}

func (f *image) validate() error {
	if f.backing == nil {
		return buferrs.NewSystemError("validate error: nil Image")
	}
	if len(f.backing.File) == 0 {
		return buferrs.NewSystemError("validate error: empty Image.File")
	}

	if f.backing.BufbuildImageExtension != nil {
		seenFileIndexes := make(map[uint32]struct{}, len(f.backing.BufbuildImageExtension.ImageImportRefs))
		for _, imageImportRef := range f.backing.BufbuildImageExtension.ImageImportRefs {
			if imageImportRef == nil {
				return buferrs.NewSystemError("validate error: nil ImageImportRef")
			}
			if imageImportRef.FileIndex == nil {
				return buferrs.NewSystemError("validate error: nil ImageImportRef.FileIndex")
			}
			fileIndex := *imageImportRef.FileIndex
			if fileIndex >= uint32(len(f.backing.File)) {
				return buferrs.NewSystemErrorf("validate error: invalid file index: %d", fileIndex)
			}
			if _, ok := seenFileIndexes[fileIndex]; ok {
				return buferrs.NewSystemErrorf("validate error: duplicate file index: %d", fileIndex)
			}
			seenFileIndexes[fileIndex] = struct{}{}
		}
	}

	seenNames := make(map[string]struct{}, len(f.backing.File))
	for _, file := range f.backing.File {
		if file == nil {
			return buferrs.NewSystemError("validate error: nil File")
		}
		if file.Name == nil {
			return buferrs.NewSystemError("validate error: nil FileDescriptorProto.Name")
		}
		name := *file.Name
		if name == "" {
			return buferrs.NewSystemError("validate error: empty FileDescrtiptorProto.Name")
		}
		if _, ok := seenNames[name]; ok {
			return buferrs.NewSystemErrorf("validate error: duplicate FileDescriptorProto.Name: %q", name)
		}
		seenNames[name] = struct{}{}
		normalizedName, err := storagepath.NormalizeAndValidate(name)
		if err != nil {
			return buferrs.NewSystemErrorf("validate error: %v", err)
		}
		if name != normalizedName {
			return buferrs.NewSystemErrorf("validate error: FileDescriptorProto.Name %q has normalized name %q", name, normalizedName)
		}
	}
	return nil
}

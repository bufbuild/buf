// Copyright 2020-2025 Buf Technologies, Inc.
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

package bufimageutil

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// filterImage filters the Image for the given options.
func filterImage(image bufimage.Image, options *imageFilterOptions) (bufimage.Image, error) {
	imageTypeIndex, err := newImageTypeIndex(image)
	if err != nil {
		return nil, err
	}
	b, _ := json.MarshalIndent(imageTypeIndex.TypeSet, "", "  ")
	fmt.Println("index:", string(b))
	typeFilter, err := createTypeFilter(imageTypeIndex, options)
	if err != nil {
		return nil, err
	}
	fmt.Println("\ttypeFilter:", typeFilter)
	fmt.Println("\t\tincludes:", typeFilter.include)
	fmt.Println("\t\texcludes:", typeFilter.exclude)
	optionsFilter, err := createOptionsFilter(imageTypeIndex, options)
	if err != nil {
		return nil, err
	}

	// Loop over image files in revserse DAG order. Imports that are no longer
	// imported by a previous file are dropped from the image.
	imageFiles := image.Files()
	dirty := false
	newImageFiles := make([]bufimage.ImageFile, 0, len(image.Files()))
	importsByFilePath := make(map[string]struct{})
	for i := len(image.Files()) - 1; i >= 0; i-- {
		imageFile := imageFiles[i]
		imageFilePath := imageFile.Path()
		if imageFile.IsImport() {
			// Check if this import is still used.
			if _, isImportUsed := importsByFilePath[imageFilePath]; !isImportUsed {
				continue
			}
		}
		newImageFile, err := filterImageFile(
			imageFile,
			imageTypeIndex,
			typeFilter,
			optionsFilter,
		)
		if err != nil {
			return nil, err
		}
		dirty = dirty || newImageFile != imageFile
		if newImageFile == nil {
			continue // Filtered out.
		}
		for _, filePath := range newImageFile.FileDescriptorProto().Dependency {
			importsByFilePath[filePath] = struct{}{}
		}
		newImageFiles = append(newImageFiles, newImageFile)
	}
	if dirty {
		// Reverse the image files back to DAG order.
		slices.Reverse(newImageFiles)
		return bufimage.NewImage(newImageFiles)
	}
	return image, nil
}

func filterImageFile(
	imageFile bufimage.ImageFile,
	imageTypeIndex *imageTypeIndex,
	typesFilter fullNameFilter,
	optionsFilter fullNameFilter,
) (bufimage.ImageFile, error) {
	fileDescriptor := imageFile.FileDescriptorProto()
	var sourcePathsRemap sourcePathsRemapTrie
	isIncluded, err := addRemapsForFileDescriptor(
		&sourcePathsRemap,
		fileDescriptor,
		imageTypeIndex,
		typesFilter,
		optionsFilter,
	)
	if err != nil {
		return nil, err
	}
	if !isIncluded {
		return nil, nil // Filtered out.
	}
	if len(sourcePathsRemap) == 0 {
		return imageFile, nil // No changes required.
	}
	newFileDescriptor, err := remapFileDescriptor(fileDescriptor, sourcePathsRemap)
	if err != nil {
		return nil, err
	}
	return bufimage.NewImageFile(
		newFileDescriptor,
		imageFile.FullName(),
		imageFile.CommitID(),
		imageFile.ExternalPath(),
		imageFile.LocalPath(),
		imageFile.IsImport(),
		imageFile.IsSyntaxUnspecified(),
		imageFile.UnusedDependencyIndexes(),
	)
}

type sourcePathsBuilder struct {
	filePath       string
	imageTypeIndex *imageTypeIndex
	typesFilter    fullNameFilter
	optionsFilter  fullNameFilter
	fileImports    map[string]struct{}
}

func addRemapsForFileDescriptor(
	sourcePathsRemap *sourcePathsRemapTrie,
	fileDescriptor *descriptorpb.FileDescriptorProto,
	imageTypeIndex *imageTypeIndex,
	typesFilter fullNameFilter,
	optionsFilter fullNameFilter,
) (bool, error) {
	fmt.Println("---", fileDescriptor.GetName(), "---")
	defer fmt.Println("------")
	packageName := protoreflect.FullName(fileDescriptor.GetPackage())
	if packageName != "" {
		// Check if filtered by the package name.
		isIncluded, isExplicit := typesFilter.filter(packageName)
		if !isIncluded && isExplicit {
			// The package is excluded.
			return false, nil
		}
	}

	fileImports := make(map[string]struct{})
	builder := &sourcePathsBuilder{
		filePath:       fileDescriptor.GetName(),
		imageTypeIndex: imageTypeIndex,
		typesFilter:    typesFilter,
		optionsFilter:  optionsFilter,
		fileImports:    fileImports,
	}
	sourcePath := make(protoreflect.SourcePath, 0, 8)

	// Walk the file descriptor.
	if _, err := addRemapsForSlice(sourcePathsRemap, packageName, append(sourcePath, fileMessagesTag), fileDescriptor.MessageType, builder.addRemapsForDescriptor); err != nil {
		return false, err
	}
	if _, err := addRemapsForSlice(sourcePathsRemap, packageName, append(sourcePath, fileEnumsTag), fileDescriptor.EnumType, builder.addRemapsForEnum); err != nil {
		return false, err
	}
	if _, err := addRemapsForSlice(sourcePathsRemap, packageName, append(sourcePath, fileServicesTag), fileDescriptor.Service, builder.addRemapsForService); err != nil {
		return false, err
	}
	if _, err := addRemapsForSlice(sourcePathsRemap, packageName, append(sourcePath, fileExtensionsTag), fileDescriptor.Extension, builder.addRemapsForField); err != nil {
		return false, err
	}
	if err := builder.addRemapsForOptions(sourcePathsRemap, append(sourcePath, fileOptionsTag), fileDescriptor.Options); err != nil {
		return false, err
	}

	// Fix the imports to remove any that are no longer used.
	// TODO: handle unused dependencies, and keep them?
	fmt.Println("fileImports", builder.fileImports)
	if len(fileImports) != len(fileDescriptor.Dependency) {
		indexTo := int32(0)
		dependencyPath := []int32{fileDependencyTag}
		dependencyChanges := make([]int32, len(fileDescriptor.Dependency))
		for indexFrom, dependency := range fileDescriptor.Dependency {
			path := append(dependencyPath, int32(indexFrom))
			if _, ok := fileImports[dependency]; ok {
				dependencyChanges[indexFrom] = indexTo
				if indexTo != int32(indexFrom) {
					sourcePathsRemap.markMoved(path, indexTo)
				}
				indexTo++
			} else {
				sourcePathsRemap.markDeleted(path)
				dependencyChanges[indexFrom] = -1
			}
		}
		publicDependencyPath := []int32{filePublicDependencyTag}
		for indexFrom, publicDependency := range fileDescriptor.PublicDependency {
			path := append(publicDependencyPath, int32(indexFrom))
			indexTo := dependencyChanges[publicDependency]
			if indexTo == -1 {
				sourcePathsRemap.markDeleted(path)
			} else if indexTo != int32(indexFrom) {
				sourcePathsRemap.markMoved(path, indexTo)
			}
		}
		weakDependencyPath := []int32{fileWeakDependencyTag}
		for indexFrom, weakDependency := range fileDescriptor.WeakDependency {
			path := append(weakDependencyPath, int32(indexFrom))
			indexTo := dependencyChanges[weakDependency]
			if indexTo == -1 {
				sourcePathsRemap.markDeleted(path)
			} else if indexTo != int32(indexFrom) {
				sourcePathsRemap.markMoved(path, indexTo)
			}
		}
	}
	return true, nil
}

func (b *sourcePathsBuilder) addRemapsForDescriptor(
	sourcePathsRemap *sourcePathsRemapTrie,
	parentName protoreflect.FullName,
	sourcePath protoreflect.SourcePath,
	descriptor *descriptorpb.DescriptorProto,
) (bool, error) {
	fullName := getFullName(parentName, descriptor)
	isIncluded, isExplicit := b.typesFilter.filter(fullName)
	if !isIncluded && isExplicit {
		// The type is excluded.
		return false, nil
	}
	//// If the message is only enclosing an included message remove the fields.
	//if isIncluded {
	//	if _, err := addRemapsForSlice(sourcePathsRemap, fullName, append(sourcePath, messageFieldsTag), descriptor.GetField(), b.addRemapsForField); err != nil {
	//		return false, err
	//	}
	//	if _, err := addRemapsForSlice(sourcePathsRemap, fullName, append(sourcePath, messageExtensionsTag), descriptor.GetExtension(), b.addRemapsForField); err != nil {
	//		return false, err
	//	}
	//	for index, extensionRange := range descriptor.GetExtensionRange() {
	//		fmt.Println("\textensionRange", index, extensionRange)
	//		extensionRangeOptionsPath := append(sourcePath, messageExtensionRangesTag, int32(index), extensionRangeOptionsTag)
	//		if err := b.addRemapsForOptions(sourcePathsRemap, extensionRangeOptionsPath, extensionRange.GetOptions()); err != nil {
	//			return false, err
	//		}
	//	}
	//	if err := b.addRemapsForOptions(sourcePathsRemap, append(sourcePath, messageOptionsTag), descriptor.GetOptions()); err != nil {
	//		return false, err
	//	}
	//} else {
	//	sourcePathsRemap.markDeleted(append(sourcePath, messageFieldsTag))
	//	sourcePathsRemap.markDeleted(append(sourcePath, messageExtensionsTag))
	//	for index := range descriptor.GetExtensionRange() {
	//		sourcePathsRemap.markDeleted(append(sourcePath, messageExtensionRangesTag, int32(index), extensionRangeOptionsTag))
	//	}
	//	sourcePathsRemap.markDeleted(append(sourcePath, messageOptionsTag))
	//}

	// Walk the nested types.
	hasNestedTypes, err := addRemapsForSlice(sourcePathsRemap, fullName, append(sourcePath, messageNestedMessagesTag), descriptor.NestedType, b.addRemapsForDescriptor)
	if err != nil {
		return false, err
	}
	isIncluded = isIncluded || hasNestedTypes

	// Walk the enum types.
	hasEnums, err := addRemapsForSlice(sourcePathsRemap, fullName, append(sourcePath, messageEnumsTag), descriptor.EnumType, b.addRemapsForEnum)
	if err != nil {
		return false, err
	}
	isIncluded = isIncluded || hasEnums

	// Walk the oneof types.
	hasOneofs, err := addRemapsForSlice(sourcePathsRemap, fullName, append(sourcePath, messageOneofsTag), descriptor.OneofDecl, b.addRemapsForOneof)
	if err != nil {
		return false, err
	}
	isIncluded = isIncluded || hasOneofs

	// If the message is only enclosing an included message remove the fields.
	if isIncluded {
		if _, err := addRemapsForSlice(sourcePathsRemap, fullName, append(sourcePath, messageFieldsTag), descriptor.GetField(), b.addRemapsForField); err != nil {
			return false, err
		}
		if _, err := addRemapsForSlice(sourcePathsRemap, fullName, append(sourcePath, messageExtensionsTag), descriptor.GetExtension(), b.addRemapsForField); err != nil {
			return false, err
		}
		for index, extensionRange := range descriptor.GetExtensionRange() {
			fmt.Println("\textensionRange", index, extensionRange)
			extensionRangeOptionsPath := append(sourcePath, messageExtensionRangesTag, int32(index), extensionRangeOptionsTag)
			if err := b.addRemapsForOptions(sourcePathsRemap, extensionRangeOptionsPath, extensionRange.GetOptions()); err != nil {
				return false, err
			}
		}
		if err := b.addRemapsForOptions(sourcePathsRemap, append(sourcePath, messageOptionsTag), descriptor.GetOptions()); err != nil {
			return false, err
		}
	} else {

	return isIncluded, nil
}

func (b *sourcePathsBuilder) addRemapsForEnum(
	sourcePathsRemap *sourcePathsRemapTrie,
	parentName protoreflect.FullName,
	sourcePath protoreflect.SourcePath,
	enum *descriptorpb.EnumDescriptorProto,
) (bool, error) {
	fullName := getFullName(parentName, enum)
	if isIncluded, _ := b.typesFilter.filter(fullName); !isIncluded {
		// The type is excluded, enum values cannot be excluded individually.
		return false, nil
	}

	if err := b.addRemapsForOptions(sourcePathsRemap, append(sourcePath, enumOptionsTag), enum.GetOptions()); err != nil {
		return false, err
	}

	// Walk the enum values.
	for index, enumValue := range enum.Value {
		enumValuePath := append(sourcePath, enumValuesTag, int32(index))
		enumValueOptionsPath := append(enumValuePath, enumValueOptionsTag)
		if err := b.addRemapsForOptions(sourcePathsRemap, enumValueOptionsPath, enumValue.GetOptions()); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (b *sourcePathsBuilder) addRemapsForOneof(
	sourcePathsRemap *sourcePathsRemapTrie,
	parentName protoreflect.FullName,
	sourcePath protoreflect.SourcePath,
	oneof *descriptorpb.OneofDescriptorProto,
) (bool, error) {
	fullName := getFullName(parentName, oneof)
	if isIncluded, _ := b.typesFilter.filter(fullName); !isIncluded {
		// The type is excluded, enum values cannot be excluded individually.
		return false, nil
	}
	if err := b.addRemapsForOptions(sourcePathsRemap, append(sourcePath, oneofOptionsTag), oneof.GetOptions()); err != nil {
		return false, err
	}
	return true, nil
}

func (b *sourcePathsBuilder) addRemapsForService(
	sourcePathsRemap *sourcePathsRemapTrie,
	parentName protoreflect.FullName,
	sourcePath protoreflect.SourcePath,
	service *descriptorpb.ServiceDescriptorProto,
) (bool, error) {
	fullName := getFullName(parentName, service)
	isIncluded, isExplicit := b.typesFilter.filter(fullName)
	if !isIncluded && isExplicit {
		// The type is excluded.
		return false, nil
	}
	if isIncluded {
		if err := b.addRemapsForOptions(sourcePathsRemap, append(sourcePath, serviceOptionsTag), service.GetOptions()); err != nil {
			return false, err
		}
	}
	// Walk the service methods.
	hasMethods, err := addRemapsForSlice(sourcePathsRemap, fullName, append(sourcePath, serviceMethodsTag), service.Method, b.addRemapsForMethod)
	if err != nil {
		return false, err
	}
	return isIncluded || hasMethods, nil
}

func (b *sourcePathsBuilder) addRemapsForMethod(
	sourcePathsRemap *sourcePathsRemapTrie,
	parentName protoreflect.FullName,
	sourcePath protoreflect.SourcePath,
	method *descriptorpb.MethodDescriptorProto,
) (bool, error) {
	fullName := getFullName(parentName, method)
	if isIncluded, _ := b.typesFilter.filter(fullName); !isIncluded {
		// The type is excluded.
		return false, nil
	}
	inputName := protoreflect.FullName(strings.TrimPrefix(method.GetInputType(), "."))
	if isIncluded, _ := b.typesFilter.filter(inputName); !isIncluded {
		// The input type is excluded.
		return false, fmt.Errorf("input type %s of method %s is excluded", inputName, fullName)
	}
	b.addRequiredType(inputName)
	outputName := protoreflect.FullName(strings.TrimPrefix(method.GetOutputType(), "."))
	if isIncluded, _ := b.typesFilter.filter(outputName); !isIncluded {
		// The output type is excluded.
		return false, fmt.Errorf("output type %s of method %s is excluded", outputName, fullName)
	}
	b.addRequiredType(outputName)
	if err := b.addRemapsForOptions(sourcePathsRemap, append(sourcePath, methodOptionsTag), method.GetOptions()); err != nil {
		return false, err
	}
	return true, nil
}

func (b *sourcePathsBuilder) addRemapsForField(
	sourcePathsRemap *sourcePathsRemapTrie,
	parentName protoreflect.FullName,
	sourcePath protoreflect.SourcePath,
	field *descriptorpb.FieldDescriptorProto,
) (bool, error) {
	if field.Extendee != nil {
		// This is an extension field.
		extendeeName := protoreflect.FullName(strings.TrimPrefix(field.GetExtendee(), "."))
		if isIncluded, _ := b.typesFilter.filter(extendeeName); !isIncluded {
			return false, nil
		}
		b.addRequiredType(extendeeName)
	}
	switch field.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_ENUM,
		descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
		descriptorpb.FieldDescriptorProto_TYPE_GROUP:
		typeName := protoreflect.FullName(strings.TrimPrefix(field.GetTypeName(), "."))
		if isIncluded, _ := b.typesFilter.filter(typeName); !isIncluded {
			return false, nil
		}
		b.addRequiredType(typeName)
	case descriptorpb.FieldDescriptorProto_TYPE_DOUBLE,
		descriptorpb.FieldDescriptorProto_TYPE_FLOAT,
		descriptorpb.FieldDescriptorProto_TYPE_INT64,
		descriptorpb.FieldDescriptorProto_TYPE_UINT64,
		descriptorpb.FieldDescriptorProto_TYPE_INT32,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED64,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED32,
		descriptorpb.FieldDescriptorProto_TYPE_BOOL,
		descriptorpb.FieldDescriptorProto_TYPE_STRING,
		descriptorpb.FieldDescriptorProto_TYPE_BYTES,
		descriptorpb.FieldDescriptorProto_TYPE_UINT32,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED32,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED64,
		descriptorpb.FieldDescriptorProto_TYPE_SINT32,
		descriptorpb.FieldDescriptorProto_TYPE_SINT64:
	default:
		return false, fmt.Errorf("unknown field type %d", field.GetType())
	}
	if err := b.addRemapsForOptions(sourcePathsRemap, append(sourcePath, fieldOptionsTag), field.GetOptions()); err != nil {
		return false, err
	}
	return true, nil
}

func (b *sourcePathsBuilder) addRemapsForOptions(
	sourcePathsRemap *sourcePathsRemapTrie,
	optionsPath protoreflect.SourcePath,
	optionsMessage proto.Message,
) error {
	if optionsMessage == nil {
		return nil
	}
	options := optionsMessage.ProtoReflect()
	numFieldsToKeep := 0
	options.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		isIncluded, _ := b.optionsFilter.filter(fd.FullName())
		if !isIncluded {
			// Remove this option.
			fmt.Println("\tremove option", fd.FullName())
			optionPath := append(optionsPath, int32(fd.Number()))
			sourcePathsRemap.markDeleted(optionPath)
			return true
		}
		fmt.Println("\tkeep option", fd.FullName())
		numFieldsToKeep++
		if fd.IsExtension() {
			// Add the extension type to the required types.
			b.addRequiredType(fd.FullName())
		}
		return true
	})
	if numFieldsToKeep == 0 {
		// No options to keep.
		sourcePathsRemap.markDeleted(optionsPath)
	}
	return nil
}

func (b *sourcePathsBuilder) addRequiredType(fullName protoreflect.FullName) {
	file, ok := b.imageTypeIndex.TypeToFile[fullName]
	if !ok {
		panic(fmt.Sprintf("could not find file for %s", fullName))
	}
	fmt.Println("\taddRequiredType", fullName, file.Path())
	if file.Path() != b.filePath {
		// This is an imported type.
		file := b.imageTypeIndex.TypeToFile[fullName]
		b.fileImports[file.Path()] = struct{}{}
	}
}

func addRemapsForSlice[T any](
	sourcePathsRemap *sourcePathsRemapTrie,
	parentName protoreflect.FullName,
	sourcePath protoreflect.SourcePath,
	list []T,
	addRemapsForItem func(*sourcePathsRemapTrie, protoreflect.FullName, protoreflect.SourcePath, T) (bool, error),
) (bool, error) {
	fromIndex, toIndex := int32(0), int32(0)
	for int(fromIndex) < len(list) {
		item := list[fromIndex]
		sourcePath := append(sourcePath, fromIndex)
		isIncluded, err := addRemapsForItem(sourcePathsRemap, parentName, sourcePath, item)
		if err != nil {
			return false, err
		}
		if isIncluded {
			if fromIndex != toIndex {
				sourcePathsRemap.markMoved(sourcePath, toIndex)
			}
			toIndex++
		} else {
			sourcePathsRemap.markDeleted(sourcePath)
		}
		fromIndex++
	}
	if toIndex == 0 {
		sourcePathsRemap.markDeleted(sourcePath)
	}
	return toIndex > 0, nil
}

/*
type fileDescriptorWalker struct {
	filePath       string
	imageTypeIndex *imageTypeIndex
	typesFilter    fullNameFilter
	optionsFilter  fullNameFilter

	// On walking record whether the type is inlcuded.
	includes []bool

	sourcePathsRemap sourcePathsRemapTrie
	fileImports      map[string]struct{}
}

func (f *fileDescriptorWalker) walkFile(
	fileDescriptor *descriptorpb.FileDescriptorProto,
	sourcePathsRemap sourcePathsRemapTrie,
) error {
	prefix := fileDescriptor.GetPackage()
	if prefix != "" {
		prefix += "."
	}
	isIncluded, isExplicit := f.typesFilter.filter(protoreflect.FullName(prefix))
	if !isIncluded && isExpli

	sourcePath := make(protoreflect.SourcePath, 0, 16)
	toIndex := 0
	for fromIndex := range fileDescriptor.MessageType {
		sourcePath := append(sourcePath, fileMessagesTag, int32(fromIndex))
		isIncluded, err := f.walkMessage(prefix, sourcePath, fileDescriptor.MessageType[fromIndex])
		if err != nil {
			return err
		}
		if isIncluded && fromIndex != toIndex {
			f.sourcePathsRemap.markMoved(sourcePath, int32(toIndex))
		} else {
			f.sourcePathsRemap.markDeleted(sourcePath)
		}
	}
}

func (f *fileDescriptorWalker) walkMessage(prefix string, sourcePath protoreflect.SourcePath, descriptor *descriptorpb.DescriptorProto) (bool, error) {

}

func (f *fileDescriptorWalker) enter(fullName protoreflect.FullName, sourcePath protoreflect.SourcePath, descriptor proto.Message) error {
	var isIncluded bool
	switch descriptor := descriptor.(type) {
	case *descriptorpb.EnumValueDescriptorProto, *descriptorpb.OneofDescriptorProto:
		// Added by their enclosing types.
		isIncluded = f.includes[len(f.includes)-1]
	case *descriptorpb.FieldDescriptorProto:
		isIncluded = f.isFieldIncluded(descriptor)
	default:
		isIncluded = f.typesFilter.filter(fullName)
	}
	fmt.Println("ENTER", fullName, "included?", isIncluded)
	if isIncluded {
		// If a child is included, the parent must be included.
		for index := range f.includes {
			f.includes[index] = true
		}
	}
	f.includes = append(f.includes, isIncluded)
	return nil
}

func (f *fileDescriptorWalker) exit(fullName protoreflect.FullName, sourcePath protoreflect.SourcePath, descriptor proto.Message) error {
	fmt.Println("EXIT", f.includes)
	isIncluded := f.includes[len(f.includes)-1]
	f.includes = f.includes[:len(f.includes)-1]
	if !isIncluded {
		// Mark the source path for deletion.
		f.sourcePathsRemap.markDeleted(sourcePath)
		fmt.Println("\tDELETE", fullName)
		return nil
	}
	// If the type is included, walk the options.
	switch descriptor := descriptor.(type) {
	case *descriptorpb.FileDescriptorProto:
		// File options are handled at the top level, before walking the file.
		// The FileDescriptorProto is not walked here.
		return nil
	case *descriptorpb.DescriptorProto:
		optionsPath := append(sourcePath, messageOptionsTag)
		return f.options(descriptor.GetOptions(), optionsPath)
	case *descriptorpb.FieldDescriptorProto:
		// Add the field type to the required types.
		if err := f.addFieldType(descriptor); err != nil {
			return err
		}
		optionsPath := append(sourcePath, fieldOptionsTag)
		return f.options(descriptor.GetOptions(), optionsPath)
	case *descriptorpb.OneofDescriptorProto:
		optionsPath := append(sourcePath, oneofOptionsTag)
		return f.options(descriptor.GetOptions(), optionsPath)
	case *descriptorpb.EnumDescriptorProto:
		optionsPath := append(sourcePath, enumOptionsTag)
		return f.options(descriptor.GetOptions(), optionsPath)
	case *descriptorpb.EnumValueDescriptorProto:
		optionsPath := append(sourcePath, enumValueOptionsTag)
		return f.options(descriptor.GetOptions(), optionsPath)
	case *descriptorpb.ServiceDescriptorProto:
		optionsPath := append(sourcePath, serviceOptionsTag)
		return f.options(descriptor.GetOptions(), optionsPath)
	case *descriptorpb.MethodDescriptorProto:
		optionsPath := append(sourcePath, methodOptionsTag)
		return f.options(descriptor.GetOptions(), optionsPath)
	case *descriptorpb.DescriptorProto_ExtensionRange:
		optionsPath := append(sourcePath, extensionRangeOptionsTag)
		return f.options(descriptor.GetOptions(), optionsPath)
	default:
		return fmt.Errorf("unexpected message type %T", descriptor)
	}
}

func (f *fileDescriptorWalker) options(
	optionsMessage proto.Message,
	optionsPath protoreflect.SourcePath,
) error {
	if optionsMessage == nil {
		return nil
	}
	options := optionsMessage.ProtoReflect()
	if !options.IsValid() {
		return nil // No options to strip.
	}
	numFieldsToKeep := 0
	options.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if !f.optionsFilter.filter(fd.FullName()) {
			// Remove this option.
			fmt.Println("\tremove option", fd.FullName())
			optionPath := append(optionsPath, int32(fd.Number()))
			f.sourcePathsRemap.markDeleted(optionPath)
			return true
		}
		fmt.Println("\tkeep option", fd.FullName())
		numFieldsToKeep++
		if fd.IsExtension() {
			// Add the extension type to the required types.
			f.addRequiredType(fd.FullName())
		}
		return true
	})
	if numFieldsToKeep == 0 {
		f.sourcePathsRemap.markDeleted(optionsPath) // No options to keep.
	}
	return nil
}

func (f *fileDescriptorWalker) isFieldIncluded(field *descriptorpb.FieldDescriptorProto) bool {
	isIncluded := f.includes[len(f.includes)-1]
	if field.Extendee != nil {
		// This is an extension field.
		extendee := strings.TrimPrefix(field.GetExtendee(), ".")
		isIncluded = isIncluded && f.typesFilter.filter(protoreflect.FullName(extendee))
	}
	typeName := strings.TrimPrefix(field.GetTypeName(), ".")
	isIncluded = isIncluded && f.typesFilter.filter(protoreflect.FullName(typeName))
	fmt.Println("\tisFieldIncluded", field.GetName(), typeName, isIncluded)
	return isIncluded
}

func (f *fileDescriptorWalker) addFieldType(field *descriptorpb.FieldDescriptorProto) error {
	if field.Extendee != nil {
		// This is an extension field.
		extendee := strings.TrimPrefix(field.GetExtendee(), ".")
		f.addRequiredType(protoreflect.FullName(extendee))
	}
	switch field.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_ENUM,
		descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
		descriptorpb.FieldDescriptorProto_TYPE_GROUP:
		typeName := strings.TrimPrefix(field.GetTypeName(), ".")
		f.addRequiredType(protoreflect.FullName(typeName))
	case descriptorpb.FieldDescriptorProto_TYPE_DOUBLE,
		descriptorpb.FieldDescriptorProto_TYPE_FLOAT,
		descriptorpb.FieldDescriptorProto_TYPE_INT64,
		descriptorpb.FieldDescriptorProto_TYPE_UINT64,
		descriptorpb.FieldDescriptorProto_TYPE_INT32,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED64,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED32,
		descriptorpb.FieldDescriptorProto_TYPE_BOOL,
		descriptorpb.FieldDescriptorProto_TYPE_STRING,
		descriptorpb.FieldDescriptorProto_TYPE_BYTES,
		descriptorpb.FieldDescriptorProto_TYPE_UINT32,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED32,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED64,
		descriptorpb.FieldDescriptorProto_TYPE_SINT32,
		descriptorpb.FieldDescriptorProto_TYPE_SINT64:
	// nothing to follow, custom options handled above.
	default:
		return fmt.Errorf("unknown field type %d", field.GetType())
	}
	return nil
}

func (f *fileDescriptorWalker) addRequiredType(fullName protoreflect.FullName) {
	file := f.imageTypeIndex.TypeToFile[fullName]
	fmt.Println("\taddRequiredType", fullName, file.Path())
	if file.Path() != f.filePath {
		// This is an imported type.
		file := f.imageTypeIndex.TypeToFile[fullName]
		f.fileImports[file.Path()] = struct{}{}
	}
}*/

func remapFileDescriptor(
	fileDescriptor *descriptorpb.FileDescriptorProto,
	sourcePathRemaps sourcePathsRemapTrie,
) (*descriptorpb.FileDescriptorProto, error) {
	fileDescriptorMessage := fileDescriptor.ProtoReflect()
	newFileDescriptorMessage, err := remapMessageReflect(fileDescriptorMessage, sourcePathRemaps)
	if err != nil {
		return nil, err
	}
	newFileDescriptor, ok := newFileDescriptorMessage.Interface().(*descriptorpb.FileDescriptorProto)
	if !ok {
		return nil, syserror.Newf("unexpected type %T", newFileDescriptorMessage.Interface())
	}
	// Remap the source code info.
	if locations := fileDescriptor.SourceCodeInfo.GetLocation(); len(locations) > 0 {
		newLocations := make([]*descriptorpb.SourceCodeInfo_Location, 0, len(locations))
		for _, location := range locations {
			oldPath := location.Path
			newPath, noComment := sourcePathRemaps.newPath(oldPath)
			if newPath == nil {
				continue
			}
			if !slices.Equal(oldPath, newPath) || noComment {
				location = shallowClone(location)
				location.Path = newPath
			}
			if noComment {
				location.LeadingDetachedComments = nil
				location.LeadingComments = nil
				location.TrailingComments = nil
			}
			newLocations = append(newLocations, location)
		}
		newFileDescriptor.SourceCodeInfo = &descriptorpb.SourceCodeInfo{
			Location: newLocations,
		}
	}
	return newFileDescriptor, nil
}

func remapMessageReflect(
	message protoreflect.Message,
	sourcePathRemaps sourcePathsRemapTrie,
) (protoreflect.Message, error) {
	if len(sourcePathRemaps) == 0 {
		return message, nil
	}
	if !message.IsValid() {
		return nil, fmt.Errorf("invalid message %T", message)
	}
	fieldDescriptors, err := getFieldDescriptors(message, sourcePathRemaps)
	if err != nil {
		return nil, err
	}
	message = shallowCloneReflect(message)
	for index, remapNode := range sourcePathRemaps {
		fieldDescriptor := fieldDescriptors[index]
		if fieldDescriptor == nil {
			return nil, fmt.Errorf("missing field descriptor %d on type %s", remapNode.oldIndex, message.Descriptor().FullName())
		}
		if remapNode.newIndex == -1 {
			message.Clear(fieldDescriptor)
			continue
		} else if remapNode.newIndex != remapNode.oldIndex {
			return nil, fmt.Errorf("unexpected field move %d to %d", remapNode.oldIndex, remapNode.newIndex)
		}
		value := message.Get(fieldDescriptor)
		switch {
		case fieldDescriptor.IsList():
			if len(remapNode.children) == 0 {
				break
			}
			newList := message.NewField(fieldDescriptor).List()
			if err := remapListReflect(newList, value.List(), remapNode.children); err != nil {
				return nil, err
			}
			value = protoreflect.ValueOfList(newList)
		case fieldDescriptor.IsMap():
			panic("map fields not yet supported")
		default:
			fieldMessage, err := remapMessageReflect(value.Message(), remapNode.children)
			if err != nil {
				return nil, err
			}
			value = protoreflect.ValueOfMessage(fieldMessage)
		}
		message.Set(fieldDescriptor, value)
	}
	return message, nil
}

func remapListReflect(
	dstList protoreflect.List,
	srcList protoreflect.List,
	sourcePathRemaps sourcePathsRemapTrie,
) error {
	if len(sourcePathRemaps) == 0 {
		return nil
	}
	toIndex := 0
	sourcePathIndex := 0
	for fromIndex := 0; fromIndex < srcList.Len(); fromIndex++ {
		var remapNode *sourcePathsRemapTrieNode
		for ; sourcePathIndex < len(sourcePathRemaps); sourcePathIndex++ {
			nextRemapNode := sourcePathRemaps[sourcePathIndex]
			if index := int(nextRemapNode.oldIndex); index > fromIndex {
				break
			} else if index == fromIndex {
				remapNode = nextRemapNode
				break
			}
		}
		value := srcList.Get(fromIndex)
		if remapNode == nil {
			dstList.Append(value)
			toIndex++
			continue
		}
		if remapNode.newIndex == -1 {
			continue
		}
		if fromIndex != int(remapNode.oldIndex) || toIndex != int(remapNode.newIndex) {
			return fmt.Errorf("unexpected list move %d to %d, expected %d to %d", remapNode.oldIndex, remapNode.newIndex, fromIndex, toIndex)
		}
		//if toIndex != int(remapNode.newIndex) {
		//	// Mutate the remap node to reflect the actual index.
		//	// TODO: this is a hack.
		//	remapNode.newIndex = int32(toIndex)
		//}
		// If no children, the value is unchanged.
		if len(remapNode.children) > 0 {
			// Must be a list of messages to have children.
			indexMessage, err := remapMessageReflect(value.Message(), remapNode.children)
			if err != nil {
				return err
			}
			value = protoreflect.ValueOfMessage(indexMessage)
		}
		dstList.Append(value)
		toIndex++
	}
	return nil
}

func getFieldDescriptors(
	message protoreflect.Message,
	sourcePathRemaps sourcePathsRemapTrie,
) ([]protoreflect.FieldDescriptor, error) {
	var hasExtension bool
	fieldDescriptors := make([]protoreflect.FieldDescriptor, len(sourcePathRemaps))
	fields := message.Descriptor().Fields()
	for index, remapNode := range sourcePathRemaps {
		fieldDescriptor := fields.ByNumber(protoreflect.FieldNumber(remapNode.oldIndex))
		if fieldDescriptor == nil {
			hasExtension = true
		} else {
			fieldDescriptors[index] = fieldDescriptor
		}
	}
	if !hasExtension {
		return fieldDescriptors, nil
	}
	message.Range(func(fieldDescriptor protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		if !fieldDescriptor.IsExtension() {
			return true // Skip non-extension fields.
		}
		if index, found := sort.Find(len(sourcePathRemaps), func(i int) int {
			return int(fieldDescriptor.Number()) - int(sourcePathRemaps[i].oldIndex)
		}); found {
			fieldDescriptors[index] = fieldDescriptor
		}
		return true
	})
	return fieldDescriptors, nil
}

func shallowClone[T proto.Message](src T) T {
	value, _ := shallowCloneReflect(src.ProtoReflect()).Interface().(T) // Safe to assert.
	return value
}

func shallowCloneReflect(src protoreflect.Message) protoreflect.Message {
	dst := src.New()
	src.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		dst.Set(fd, v)
		return true
	})
	return dst
}

func getFullName(parentName protoreflect.FullName, message interface{ GetName() string }) protoreflect.FullName {
	fullName := protoreflect.FullName(message.GetName())
	if parentName == "" {
		return fullName
	}
	return parentName + "." + fullName
}

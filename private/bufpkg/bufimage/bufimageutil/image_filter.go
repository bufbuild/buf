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
	"fmt"
	"slices"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// filterImage filters the Image for the given options.
func filterImage(image bufimage.Image, options *imageFilterOptions) (bufimage.Image, error) {
	imageIndex, err := newImageIndexForImage(image, options)
	if err != nil {
		return nil, err
	}
	closure := newTransitiveClosure()
	// All excludes are added first, then includes walk included all non excluded types.
	for excludeType := range options.excludeTypes {
		excludeType := protoreflect.FullName(excludeType)
		if err := closure.excludeType(excludeType, imageIndex, options); err != nil {
			return nil, err
		}
	}
	for includeType := range options.includeTypes {
		includeType := protoreflect.FullName(includeType)
		if err := closure.includeType(includeType, imageIndex, options); err != nil {
			return nil, err
		}
	}
	// TODO: No types were included, so include everything. This can be
	// removed when we are able to handle finding all required imports
	// below, when remapping the descriptor.
	if len(options.includeTypes) == 0 {
		for _, file := range image.Files() {
			if file.IsImport() {
				continue
			}
			if err := closure.addElement(file.FileDescriptorProto(), "", false, imageIndex, options); err != nil {
				return nil, err
			}
		}
	}
	// After all types are added, add their known extensions
	if err := closure.addExtensions(imageIndex, options); err != nil {
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
		_, isFileImported := importsByFilePath[imageFilePath]
		if imageFile.IsImport() && !options.allowImportedTypes {
			// Check if this import is still used.
			if !isFileImported {
				continue
			}
		}
		newImageFile, err := filterImageFile(
			imageFile,
			imageIndex,
			closure,
			options,
		)
		if err != nil {
			return nil, err
		}
		dirty = dirty || newImageFile != imageFile
		if newImageFile == nil {
			if isFileImported {
				return nil, fmt.Errorf("imported file %q was filtered out", imageFilePath)
			}
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
	imageIndex *imageIndex,
	closure *transitiveClosure,
	options *imageFilterOptions,
) (bufimage.ImageFile, error) {
	fileDescriptor := imageFile.FileDescriptorProto()
	var sourcePathsRemap sourcePathsRemapTrie
	builder := sourcePathsBuilder{
		imageIndex: imageIndex,
		closure:    closure,
		options:    options,
	}
	newFileDescriptor, err := builder.remapFileDescriptor(&sourcePathsRemap, fileDescriptor)
	if err != nil {
		return nil, err
	}
	if newFileDescriptor == nil {
		return nil, nil // Filtered out.
	}
	if newFileDescriptor == fileDescriptor {
		return imageFile, nil // No changes required.
	}

	// Remap the source code info.
	if locations := fileDescriptor.SourceCodeInfo.GetLocation(); len(locations) > 0 {
		newLocations := make([]*descriptorpb.SourceCodeInfo_Location, 0, len(locations))
		for _, location := range locations {
			oldPath := location.Path
			newPath, noComment := sourcePathsRemap.newPath(oldPath)
			if newPath == nil {
				continue
			}
			if noComment || !slices.Equal(oldPath, newPath) {
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
	imageIndex *imageIndex
	closure    *transitiveClosure
	options    *imageFilterOptions
}

func (b *sourcePathsBuilder) remapFileDescriptor(
	sourcePathsRemap *sourcePathsRemapTrie,
	fileDescriptor *descriptorpb.FileDescriptorProto,
) (*descriptorpb.FileDescriptorProto, error) {
	if !b.closure.hasType(fileDescriptor, b.options) {
		return nil, nil
	}

	sourcePath := make(protoreflect.SourcePath, 0, 8)

	// Walk the file descriptor.
	isDirty := false
	newMessages, changed, err := remapSlice(sourcePathsRemap, append(sourcePath, fileMessagesTag), fileDescriptor.MessageType, b.remapDescriptor)
	if err != nil {
		return nil, err
	}
	isDirty = isDirty || changed
	newEnums, changed, err := remapSlice(sourcePathsRemap, append(sourcePath, fileEnumsTag), fileDescriptor.EnumType, b.remapEnum)
	if err != nil {
		return nil, err
	}
	isDirty = isDirty || changed
	newServices, changed, err := remapSlice(sourcePathsRemap, append(sourcePath, fileServicesTag), fileDescriptor.Service, b.remapService)
	if err != nil {
		return nil, err
	}
	isDirty = isDirty || changed
	// TODO: extension docs
	//newExtensions, changed, err := remapSlice(sourcePathsRemap, append(sourcePath, fileExtensionsTag), fileDescriptor.Extension, b.remapField)
	newExtensions, changed, err := remapSlice(sourcePathsRemap, append(sourcePath, fileExtensionsTag), fileDescriptor.Extension, b.remapField)
	if err != nil {
		return nil, err
	}
	isDirty = isDirty || changed
	newOptions, changed, err := remapMessage(nil, append(sourcePath, fileOptionsTag), fileDescriptor.Options, b.remapOptions)
	if err != nil {
		return nil, err
	}
	isDirty = isDirty || changed

	if !isDirty {
		return fileDescriptor, nil
	}

	newFileDescriptor := shallowClone(fileDescriptor)
	newFileDescriptor.MessageType = newMessages
	newFileDescriptor.EnumType = newEnums
	newFileDescriptor.Service = newServices
	newFileDescriptor.Extension = newExtensions
	newFileDescriptor.Options = newOptions

	// Fix the imports to remove any that are no longer used.
	importsRequired := b.closure.imports[fileDescriptor.GetName()]

	indexTo := int32(0)
	newFileDescriptor.Dependency = nil
	dependencyPath := append(sourcePath, fileDependencyTag)
	dependencyChanges := make([]int32, len(fileDescriptor.Dependency))
	for indexFrom, importPath := range fileDescriptor.Dependency {
		path := append(dependencyPath, int32(indexFrom))
		if importsRequired != nil && importsRequired.index(importPath) != -1 {
			dependencyChanges[indexFrom] = indexTo
			if indexTo != int32(indexFrom) {
				sourcePathsRemap.markMoved(path, indexTo)
			}
			newFileDescriptor.Dependency = append(newFileDescriptor.Dependency, importPath)
			indexTo++
			// delete them as we go, so we know which ones weren't in the list
			importsRequired.delete(importPath)
		} else {
			sourcePathsRemap.markDeleted(path)
			dependencyChanges[indexFrom] = -1
		}
	}
	if importsRequired != nil {
		newFileDescriptor.Dependency = append(newFileDescriptor.Dependency, importsRequired.keys()...)
	}

	newFileDescriptor.PublicDependency = nil
	publicDependencyPath := append(sourcePath, filePublicDependencyTag)
	sourcePathsRemap.markDeleted(publicDependencyPath)

	// TODO: validate
	newFileDescriptor.WeakDependency = nil
	if len(fileDescriptor.WeakDependency) > 0 {
		weakDependencyPath := append(sourcePath, fileWeakDependencyTag)
		for _, indexFrom := range fileDescriptor.WeakDependency {
			path := append(weakDependencyPath, int32(indexFrom))
			indexTo := dependencyChanges[indexFrom]
			if indexTo == -1 {
				sourcePathsRemap.markDeleted(path)
			} else {
				if indexTo != int32(indexFrom) {
					sourcePathsRemap.markMoved(path, indexTo)
				}
				newFileDescriptor.WeakDependency = append(newFileDescriptor.WeakDependency, indexTo)
			}
		}
	}

	return newFileDescriptor, nil
}

func (b *sourcePathsBuilder) remapDescriptor(
	sourcePathsRemap *sourcePathsRemapTrie,
	sourcePath protoreflect.SourcePath,
	descriptor *descriptorpb.DescriptorProto,
) (*descriptorpb.DescriptorProto, bool, error) {
	if !b.closure.hasType(descriptor, b.options) {
		return nil, true, nil
	}
	var newDescriptor *descriptorpb.DescriptorProto
	isDirty := false
	if mode := b.closure.elements[descriptor]; mode == inclusionModeEnclosing {
		// If the type is only enclosing, only the namespace matters.
		isDirty = true
		newDescriptor = shallowClone(descriptor)
		sourcePathsRemap.markNoComment(sourcePath)
		sourcePathsRemap.markDeleted(append(sourcePath, messageFieldsTag))
		sourcePathsRemap.markDeleted(append(sourcePath, messageOneofsTag))
		sourcePathsRemap.markDeleted(append(sourcePath, messageExtensionRangesTag))
		sourcePathsRemap.markDeleted(append(sourcePath, messageReservedRangesTag))
		sourcePathsRemap.markDeleted(append(sourcePath, messageReservedNamesTag))
		newDescriptor.Field = nil
		newDescriptor.OneofDecl = nil
		newDescriptor.ExtensionRange = nil
		newDescriptor.ReservedRange = nil
		newDescriptor.ReservedName = nil
	} else {
		newFields, changed, err := remapSlice(sourcePathsRemap, append(sourcePath, messageFieldsTag), descriptor.GetField(), b.remapField)
		if err != nil {
			return nil, false, err
		}
		isDirty = isDirty || changed
		newOneofs, changed, err := remapSlice(sourcePathsRemap, append(sourcePath, messageOneofsTag), descriptor.OneofDecl, b.remapOneof)
		if err != nil {
			return nil, false, err
		}
		isDirty = isDirty || changed
		newExtensionRange, changed, err := remapSlice(sourcePathsRemap, append(sourcePath, messageExtensionRangesTag), descriptor.ExtensionRange, b.remapExtensionRange)
		if err != nil {
			return nil, false, err
		}
		isDirty = isDirty || changed
		if isDirty {
			newDescriptor = shallowClone(descriptor)
			newDescriptor.Field = newFields
			newDescriptor.OneofDecl = newOneofs
			newDescriptor.ExtensionRange = newExtensionRange
		}
	}
	// TODO: sourcePath might not be correct here.
	// TODO: extension docs.
	newExtensions, changed, err := remapSlice(sourcePathsRemap, append(sourcePath, messageExtensionsTag), descriptor.GetExtension(), b.remapField)
	if err != nil {
		return nil, false, err
	}
	isDirty = isDirty || changed
	newDescriptors, changed, err := remapSlice(sourcePathsRemap, append(sourcePath, messageNestedMessagesTag), descriptor.NestedType, b.remapDescriptor)
	if err != nil {
		return nil, false, err
	}
	isDirty = isDirty || changed
	newEnums, changed, err := remapSlice(sourcePathsRemap, append(sourcePath, messageEnumsTag), descriptor.EnumType, b.remapEnum)
	if err != nil {
		return nil, false, err
	}
	isDirty = isDirty || changed
	newOptions, changed, err := remapMessage(nil, append(sourcePath, messageOptionsTag), descriptor.GetOptions(), b.remapOptions)
	if err != nil {
		return nil, false, err
	}
	isDirty = isDirty || changed

	if !isDirty {
		return descriptor, false, nil
	}
	if newDescriptor == nil {
		newDescriptor = shallowClone(descriptor)
	}
	newDescriptor.Extension = newExtensions
	newDescriptor.NestedType = newDescriptors
	newDescriptor.EnumType = newEnums
	newDescriptor.Options = newOptions
	return newDescriptor, true, nil
}

func (b *sourcePathsBuilder) remapExtensionRange(
	sourcePathsRemap *sourcePathsRemapTrie,
	sourcePath protoreflect.SourcePath,
	extensionRange *descriptorpb.DescriptorProto_ExtensionRange,
) (*descriptorpb.DescriptorProto_ExtensionRange, bool, error) {
	newOptions, changed, err := remapMessage(nil, append(sourcePath, extensionRangeOptionsTag), extensionRange.GetOptions(), b.remapOptions)
	if err != nil {
		return nil, false, err
	}
	if !changed {
		return extensionRange, false, nil
	}
	newExtensionRange := shallowClone(extensionRange)
	newExtensionRange.Options = newOptions
	return newExtensionRange, true, nil
}

func (b *sourcePathsBuilder) remapEnum(
	sourcePathsRemap *sourcePathsRemapTrie,
	sourcePath protoreflect.SourcePath,
	enum *descriptorpb.EnumDescriptorProto,
) (*descriptorpb.EnumDescriptorProto, bool, error) {
	if !b.closure.hasType(enum, b.options) {
		// The type is excluded, enum values cannot be excluded individually.
		return nil, true, nil
	}
	var isDirty bool
	newOptions, changed, err := remapMessage(nil, append(sourcePath, enumOptionsTag), enum.GetOptions(), b.remapOptions)
	if err != nil {
		return nil, false, err
	}
	isDirty = changed

	// Walk the enum values.
	newEnumValues, changed, err := remapSlice(sourcePathsRemap, append(sourcePath, enumValuesTag), enum.Value, b.remapEnumValue)
	if err != nil {
		return nil, false, err
	}
	isDirty = isDirty || changed
	if !isDirty {
		return enum, true, nil
	}
	newEnum := shallowClone(enum)
	newEnum.Options = newOptions
	newEnum.Value = newEnumValues
	return newEnum, true, nil
}

func (b *sourcePathsBuilder) remapEnumValue(
	sourcePathsRemap *sourcePathsRemapTrie,
	sourcePath protoreflect.SourcePath,
	enumValue *descriptorpb.EnumValueDescriptorProto,
) (*descriptorpb.EnumValueDescriptorProto, bool, error) {
	newOptions, changed, err := remapMessage(nil, append(sourcePath, enumValueOptionsTag), enumValue.GetOptions(), b.remapOptions)
	if err != nil {
		return nil, false, err
	}
	if !changed {
		return enumValue, false, nil
	}
	newEnumValue := shallowClone(enumValue)
	newEnumValue.Options = newOptions
	return newEnumValue, true, nil
}

func (b *sourcePathsBuilder) remapOneof(
	sourcePathsRemap *sourcePathsRemapTrie,
	sourcePath protoreflect.SourcePath,
	oneof *descriptorpb.OneofDescriptorProto,
) (*descriptorpb.OneofDescriptorProto, bool, error) {
	options, changed, err := remapMessage(nil, append(sourcePath, oneofOptionsTag), oneof.GetOptions(), b.remapOptions)
	if err != nil {
		return nil, false, err
	}
	if !changed {
		return oneof, false, nil
	}
	newOneof := shallowClone(oneof)
	newOneof.Options = options
	return newOneof, true, nil
}

func (b *sourcePathsBuilder) remapService(
	sourcePathsRemap *sourcePathsRemapTrie,
	sourcePath protoreflect.SourcePath,
	service *descriptorpb.ServiceDescriptorProto,
) (*descriptorpb.ServiceDescriptorProto, bool, error) {
	if !b.closure.hasType(service, b.options) {
		return nil, true, nil
	}
	isDirty := false
	// Walk the service methods.
	newMethods, changed, err := remapSlice(sourcePathsRemap, append(sourcePath, serviceMethodsTag), service.Method, b.remapMethod)
	if err != nil {
		return nil, false, err
	}
	isDirty = isDirty || changed
	newOptions, changed, err := remapMessage(nil, append(sourcePath, serviceOptionsTag), service.GetOptions(), b.remapOptions)
	if err != nil {
		return nil, false, err
	}
	isDirty = isDirty || changed
	if !isDirty {
		return service, false, nil
	}
	newService := shallowClone(service)
	newService.Method = newMethods
	newService.Options = newOptions
	return newService, true, nil
}

func (b *sourcePathsBuilder) remapMethod(
	sourcePathsRemap *sourcePathsRemapTrie,
	sourcePath protoreflect.SourcePath,
	method *descriptorpb.MethodDescriptorProto,
) (*descriptorpb.MethodDescriptorProto, bool, error) {
	if !b.closure.hasType(method, b.options) {
		return nil, true, nil
	}
	newOptions, changed, err := remapMessage(nil, append(sourcePath, methodOptionsTag), method.GetOptions(), b.remapOptions)
	if err != nil {
		return nil, false, err
	}
	if !changed {
		return method, false, nil
	}
	newMethod := shallowClone(method)
	newMethod.Options = newOptions
	return newMethod, true, nil
}

func (b *sourcePathsBuilder) remapField(
	sourcePathsRemap *sourcePathsRemapTrie,
	sourcePath protoreflect.SourcePath,
	field *descriptorpb.FieldDescriptorProto,
) (*descriptorpb.FieldDescriptorProto, bool, error) {
	if field.Extendee != nil {
		// Extensions are filtered by type.
		if !b.closure.hasType(field, b.options) {
			return nil, true, nil
		}
	}
	switch field.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_ENUM,
		descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
		descriptorpb.FieldDescriptorProto_TYPE_GROUP:
		typeName := protoreflect.FullName(strings.TrimPrefix(field.GetTypeName(), "."))
		typeInfo := b.imageIndex.ByName[typeName]
		if !b.closure.hasType(typeInfo.element, b.options) {
			return nil, true, nil
		}
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
		return nil, false, fmt.Errorf("unknown field type %d", field.GetType())
	}
	newOption, changed, err := remapMessage(nil, append(sourcePath, fieldOptionsTag), field.GetOptions(), b.remapOptions)
	if err != nil {
		return nil, false, err
	}
	if !changed {
		return field, false, nil
	}
	newField := shallowClone(field)
	newField.Options = newOption
	return newField, true, nil
}

func (b *sourcePathsBuilder) remapOptions(
	sourcePathsRemap *sourcePathsRemapTrie,
	optionsPath protoreflect.SourcePath,
	optionsMessage proto.Message,
) (proto.Message, bool, error) {
	if optionsMessage == nil {
		return nil, false, nil
	}
	var newOptions protoreflect.Message
	options := optionsMessage.ProtoReflect()
	numFieldsToKeep := 0
	options.Range(func(fd protoreflect.FieldDescriptor, val protoreflect.Value) bool {
		if !b.closure.hasOption(fd, b.options) {
			// Remove this option.
			optionPath := append(optionsPath, int32(fd.Number()))
			sourcePathsRemap.markDeleted(optionPath)
			if newOptions == nil {
				newOptions = shallowCloneReflect(options)
			}
			newOptions.Clear(fd)
			return true
		}
		numFieldsToKeep++
		return true
	})
	if numFieldsToKeep == 0 {
		// No options to keep.
		sourcePathsRemap.markDeleted(optionsPath)
		return nil, true, nil
	}
	if newOptions == nil {
		return optionsMessage, false, nil
	}
	return newOptions.Interface(), true, nil
}

func remapMessage[T proto.Message](
	sourcePathsRemap *sourcePathsRemapTrie,
	sourcePath protoreflect.SourcePath,
	message T,
	remapMessage func(*sourcePathsRemapTrie, protoreflect.SourcePath, proto.Message) (proto.Message, bool, error),
) (T, bool, error) {
	var zeroValue T
	newMessageOpaque, changed, err := remapMessage(sourcePathsRemap, sourcePath, message)
	if err != nil {
		return zeroValue, false, err
	}
	if newMessageOpaque == nil {
		return zeroValue, true, nil
	}
	if !changed {
		return message, false, nil
	}
	newMessage, _ := newMessageOpaque.(T) // Safe to assert.
	return newMessage, true, nil

}

func remapSlice[T any](
	sourcePathsRemap *sourcePathsRemapTrie,
	sourcePath protoreflect.SourcePath,
	list []*T,
	remapItem func(*sourcePathsRemapTrie, protoreflect.SourcePath, *T) (*T, bool, error),
) ([]*T, bool, error) {
	isDirty := false
	var newList []*T
	fromIndex, toIndex := int32(0), int32(0)
	for int(fromIndex) < len(list) {
		item := list[fromIndex]
		sourcePath := append(sourcePath, fromIndex)
		newItem, changed, err := remapItem(sourcePathsRemap, sourcePath, item)
		if err != nil {
			return nil, false, err
		}
		isDirty = isDirty || changed
		if isDirty && newList == nil {
			newList = make([]*T, 0, len(list))
			newList = append(newList, list[:toIndex]...)
		}
		isIncluded := newItem != nil
		if isIncluded {
			if fromIndex != toIndex {
				sourcePathsRemap.markMoved(sourcePath, toIndex)
			}
			if isDirty {
				newList = append(newList, newItem)
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
	if isDirty {
		return newList, true, nil
	}
	return list, false, nil
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

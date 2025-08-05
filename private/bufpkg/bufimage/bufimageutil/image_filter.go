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
	imageIndex, err := newImageIndexForImage(image, options)
	if err != nil {
		return nil, err
	}
	closure := newTransitiveClosure()
	// All excludes are added first, then includes walk included all non excluded types.
	// TODO: consider supporting a glob syntax of some kind, to do more advanced pattern
	//   matching, such as ability to get a package AND all of its sub-packages.
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
			// Check if target package is filtered, this is not an error.
			// These includes are implicitly added to the closure.
			fileDescriptorProto := file.FileDescriptorProto()
			if mode := closure.elements[fileDescriptorProto]; mode == inclusionModeExcluded {
				continue
			}
			if err := closure.addElement(fileDescriptorProto, "", false, imageIndex, options); err != nil {
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
	dirty := false
	newImageFiles := make([]bufimage.ImageFile, 0, len(image.Files()))
	for _, imageFile := range slices.Backward(image.Files()) {
		imageFilePath := imageFile.Path()
		// Check if the file is used.
		if _, ok := closure.imports[imageFilePath]; !ok {
			continue // Filtered out.
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
			// The file was filtered out. Check if it was used by another file.
			for filePath, dependencies := range closure.imports {
				if _, isImported := dependencies[imageFilePath]; isImported {
					return nil, syserror.Newf("file %q was filtered out, but is still used by %q", imageFilePath, filePath)
				}
			}
			// Currently, with an explicitly included type we add the extensions
			// to the import list. If all the extension fields types are excluded,
			// the file may be empty. The import list still contains the empty file.
			// Skip the file. See: TestDeps/IncludeWithExcludeExtensions
			continue
		}
		newImageFiles = append(newImageFiles, newImageFile)
	}
	if !dirty {
		return image, nil
	}
	// Reverse the image files back to DAG order.
	slices.Reverse(newImageFiles)
	return bufimage.NewImage(newImageFiles)
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
	newFileDescriptor, changed, err := builder.remapFileDescriptor(&sourcePathsRemap, fileDescriptor)
	if err != nil {
		return nil, err
	}
	if newFileDescriptor == nil {
		return nil, nil // Filtered out.
	}
	if !changed {
		if len(sourcePathsRemap) > 0 {
			return nil, syserror.Newf("unexpected %d sourcePathsRemaps", len(sourcePathsRemap))
		}
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
				location = maybeClone(location, options)
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
		nil, // There are no unused dependencies.
	)
}

// sourcePathsBuilder is a helper for building the new source paths.
// Each method return the new value, whether it was changed, and an error if any.
// The value is nil if it was filtered out.
type sourcePathsBuilder struct {
	imageIndex *imageIndex
	closure    *transitiveClosure
	options    *imageFilterOptions
}

func (b *sourcePathsBuilder) remapFileDescriptor(
	sourcePathsRemap *sourcePathsRemapTrie,
	fileDescriptor *descriptorpb.FileDescriptorProto,
) (*descriptorpb.FileDescriptorProto, bool, error) {
	if !b.closure.hasType(fileDescriptor, b.options) {
		return nil, true, nil
	}

	sourcePath := make(protoreflect.SourcePath, 0, 8)

	// Walk the file descriptor.
	isDirty := false
	newMessages, changed, err := remapSlice(sourcePathsRemap, append(sourcePath, fileMessagesTag), fileDescriptor.MessageType, b.remapDescriptor, b.options)
	if err != nil {
		return nil, false, err
	}
	isDirty = isDirty || changed
	newEnums, changed, err := remapSlice(sourcePathsRemap, append(sourcePath, fileEnumsTag), fileDescriptor.EnumType, b.remapEnum, b.options)
	if err != nil {
		return nil, false, err
	}
	isDirty = isDirty || changed
	newServices, changed, err := remapSlice(sourcePathsRemap, append(sourcePath, fileServicesTag), fileDescriptor.Service, b.remapService, b.options)
	if err != nil {
		return nil, false, err
	}
	isDirty = isDirty || changed
	newExtensions, changed, err := remapSlice(sourcePathsRemap, append(sourcePath, fileExtensionsTag), fileDescriptor.Extension, b.remapField, b.options)
	if err != nil {
		return nil, false, err
	}
	isDirty = isDirty || changed
	newOptions, changed, err := remapMessage(sourcePathsRemap, append(sourcePath, fileOptionsTag), fileDescriptor.Options, b.remapOptions)
	if err != nil {
		return nil, false, err
	}
	isDirty = isDirty || changed
	newDependencies, newPublicDependencies, newWeakDependencies, changed, err := b.remapDependencies(sourcePathsRemap, sourcePath, fileDescriptor)
	if err != nil {
		return nil, false, err
	}
	isDirty = isDirty || changed
	if !isDirty {
		return fileDescriptor, false, nil
	}

	newFileDescriptor := maybeClone(fileDescriptor, b.options)
	newFileDescriptor.MessageType = newMessages
	newFileDescriptor.EnumType = newEnums
	newFileDescriptor.Service = newServices
	newFileDescriptor.Extension = newExtensions
	newFileDescriptor.Options = newOptions
	newFileDescriptor.Dependency = newDependencies
	newFileDescriptor.PublicDependency = newPublicDependencies
	newFileDescriptor.WeakDependency = newWeakDependencies
	return newFileDescriptor, true, nil
}

func (b *sourcePathsBuilder) remapDependencies(
	sourcePathsRemap *sourcePathsRemapTrie,
	sourcePath protoreflect.SourcePath,
	fileDescriptor *descriptorpb.FileDescriptorProto,
) ([]string, []int32, []int32, bool, error) {
	dependencies := fileDescriptor.GetDependency()
	publicDependencies := fileDescriptor.GetPublicDependency()
	weakDependencies := fileDescriptor.GetWeakDependency()

	// Check if the imports need to be remapped.
	importsRequired := b.closure.imports[fileDescriptor.GetName()]
	importsCount := len(importsRequired)
	for _, importPath := range dependencies {
		if _, ok := importsRequired[importPath]; ok {
			importsCount--
		} else {
			importsCount = -1
			break
		}
	}
	if importsCount == 0 && len(publicDependencies) == 0 {
		// Imports match and no public dependencies.
		return dependencies, publicDependencies, weakDependencies, false, nil
	}

	indexFrom, indexTo := int32(0), int32(0)
	var newDependencies []string
	if b.options.mutateInPlace {
		newDependencies = dependencies[:0]
	}
	dependencyPath := append(sourcePath, fileDependencyTag)
	dependencyChanges := make([]int32, len(dependencies))
	for _, importPath := range dependencies {
		path := append(dependencyPath, indexFrom)
		if _, ok := importsRequired[importPath]; ok {
			dependencyChanges[indexFrom] = indexTo
			if indexTo != indexFrom {
				sourcePathsRemap.markMoved(path, indexTo)
			}
			newDependencies = append(newDependencies, importPath)
			indexTo++
			// delete them as we go, so we know which ones weren't in the list
			delete(importsRequired, importPath)
		} else {
			sourcePathsRemap.markDeleted(path)
			dependencyChanges[indexFrom] = -1
		}
		indexFrom++
	}
	// Add imports picked up via a public import. The filtered files do not use public imports.
	if publicImportCount := len(importsRequired); publicImportCount > 0 {
		for importPath := range importsRequired {
			newDependencies = append(newDependencies, importPath)
		}
		// Sort the public imports to ensure the output is deterministic.
		sort.Strings(newDependencies[len(newDependencies)-publicImportCount:])
	}

	// Public dependencies are always removed on remapping.
	publicDependencyPath := append(sourcePath, filePublicDependencyTag)
	sourcePathsRemap.markDeleted(publicDependencyPath)

	var newWeakDependencies []int32
	if len(weakDependencies) > 0 {
		if b.options.mutateInPlace {
			newWeakDependencies = weakDependencies[:0]
		}
		weakDependencyPath := append(sourcePath, fileWeakDependencyTag)
		for _, indexFrom := range weakDependencies {
			path := append(weakDependencyPath, indexFrom)
			indexTo := dependencyChanges[indexFrom]
			if indexTo == -1 {
				sourcePathsRemap.markDeleted(path)
			} else {
				if indexTo != indexFrom {
					sourcePathsRemap.markMoved(path, indexTo)
				}
				newWeakDependencies = append(newWeakDependencies, indexTo)
			}
		}
	}
	return newDependencies, nil, newWeakDependencies, true, nil
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
		if len(descriptor.Field) > 0 ||
			len(descriptor.OneofDecl) > 0 ||
			len(descriptor.ExtensionRange) > 0 ||
			len(descriptor.ReservedRange) > 0 ||
			len(descriptor.ReservedName) > 0 {
			// Clear unnecessary fields.
			isDirty = true
			newDescriptor = maybeClone(descriptor, b.options)
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
		}
	} else {
		newFields, changed, err := remapSlice(sourcePathsRemap, append(sourcePath, messageFieldsTag), descriptor.GetField(), b.remapField, b.options)
		if err != nil {
			return nil, false, err
		}
		isDirty = isDirty || changed
		newOneofs, changed, err := remapSlice(sourcePathsRemap, append(sourcePath, messageOneofsTag), descriptor.OneofDecl, b.remapOneof, b.options)
		if err != nil {
			return nil, false, err
		}
		isDirty = isDirty || changed
		newExtensionRange, changed, err := remapSlice(sourcePathsRemap, append(sourcePath, messageExtensionRangesTag), descriptor.ExtensionRange, b.remapExtensionRange, b.options)
		if err != nil {
			return nil, false, err
		}
		isDirty = isDirty || changed
		if isDirty {
			newDescriptor = maybeClone(descriptor, b.options)
			newDescriptor.Field = newFields
			newDescriptor.OneofDecl = newOneofs
			newDescriptor.ExtensionRange = newExtensionRange
		}
	}
	newExtensions, changed, err := remapSlice(sourcePathsRemap, append(sourcePath, messageExtensionsTag), descriptor.GetExtension(), b.remapField, b.options)
	if err != nil {
		return nil, false, err
	}
	isDirty = isDirty || changed
	newDescriptors, changed, err := remapSlice(sourcePathsRemap, append(sourcePath, messageNestedMessagesTag), descriptor.NestedType, b.remapDescriptor, b.options)
	if err != nil {
		return nil, false, err
	}
	isDirty = isDirty || changed
	newEnums, changed, err := remapSlice(sourcePathsRemap, append(sourcePath, messageEnumsTag), descriptor.EnumType, b.remapEnum, b.options)
	if err != nil {
		return nil, false, err
	}
	isDirty = isDirty || changed
	newOptions, changed, err := remapMessage(sourcePathsRemap, append(sourcePath, messageOptionsTag), descriptor.GetOptions(), b.remapOptions)
	if err != nil {
		return nil, false, err
	}
	isDirty = isDirty || changed

	if !isDirty {
		return descriptor, false, nil
	}
	if newDescriptor == nil {
		newDescriptor = maybeClone(descriptor, b.options)
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
	newOptions, changed, err := remapMessage(sourcePathsRemap, append(sourcePath, extensionRangeOptionsTag), extensionRange.GetOptions(), b.remapOptions)
	if err != nil {
		return nil, false, err
	}
	if !changed {
		return extensionRange, false, nil
	}
	newExtensionRange := maybeClone(extensionRange, b.options)
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
	newOptions, changed, err := remapMessage(sourcePathsRemap, append(sourcePath, enumOptionsTag), enum.GetOptions(), b.remapOptions)
	if err != nil {
		return nil, false, err
	}
	isDirty = changed

	// Walk the enum values.
	newEnumValues, changed, err := remapSlice(sourcePathsRemap, append(sourcePath, enumValuesTag), enum.Value, b.remapEnumValue, b.options)
	if err != nil {
		return nil, false, err
	}
	isDirty = isDirty || changed
	if !isDirty {
		return enum, true, nil
	}
	newEnum := maybeClone(enum, b.options)
	newEnum.Options = newOptions
	newEnum.Value = newEnumValues
	return newEnum, true, nil
}

func (b *sourcePathsBuilder) remapEnumValue(
	sourcePathsRemap *sourcePathsRemapTrie,
	sourcePath protoreflect.SourcePath,
	enumValue *descriptorpb.EnumValueDescriptorProto,
) (*descriptorpb.EnumValueDescriptorProto, bool, error) {
	newOptions, changed, err := remapMessage(sourcePathsRemap, append(sourcePath, enumValueOptionsTag), enumValue.GetOptions(), b.remapOptions)
	if err != nil {
		return nil, false, err
	}
	if !changed {
		return enumValue, false, nil
	}
	newEnumValue := maybeClone(enumValue, b.options)
	newEnumValue.Options = newOptions
	return newEnumValue, true, nil
}

func (b *sourcePathsBuilder) remapOneof(
	sourcePathsRemap *sourcePathsRemapTrie,
	sourcePath protoreflect.SourcePath,
	oneof *descriptorpb.OneofDescriptorProto,
) (*descriptorpb.OneofDescriptorProto, bool, error) {
	if mode, ok := b.closure.elements[oneof]; ok && mode == inclusionModeExcluded {
		// Oneofs are implicitly excluded when all of its fields types are excluded.
		return nil, true, nil
	}
	options, changed, err := remapMessage(sourcePathsRemap, append(sourcePath, oneofOptionsTag), oneof.GetOptions(), b.remapOptions)
	if err != nil {
		return nil, false, err
	}
	if !changed {
		return oneof, false, nil
	}
	newOneof := maybeClone(oneof, b.options)
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
	newMethods, changed, err := remapSlice(sourcePathsRemap, append(sourcePath, serviceMethodsTag), service.Method, b.remapMethod, b.options)
	if err != nil {
		return nil, false, err
	}
	isDirty = isDirty || changed
	newOptions, changed, err := remapMessage(sourcePathsRemap, append(sourcePath, serviceOptionsTag), service.GetOptions(), b.remapOptions)
	if err != nil {
		return nil, false, err
	}
	isDirty = isDirty || changed
	if !isDirty {
		return service, false, nil
	}
	newService := maybeClone(service, b.options)
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
	newOptions, changed, err := remapMessage(sourcePathsRemap, append(sourcePath, methodOptionsTag), method.GetOptions(), b.remapOptions)
	if err != nil {
		return nil, false, err
	}
	if !changed {
		return method, false, nil
	}
	newMethod := maybeClone(method, b.options)
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
	newOptions, changed, err := remapMessage(sourcePathsRemap, append(sourcePath, fieldOptionsTag), field.GetOptions(), b.remapOptions)
	if err != nil {
		return nil, false, err
	}
	if !changed {
		return field, false, nil
	}
	newField := maybeClone(field, b.options)
	newField.Options = newOptions
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
		if !b.closure.hasOption(fd, b.imageIndex, b.options) {
			// Remove this option.
			optionPath := append(optionsPath, int32(fd.Number()))
			sourcePathsRemap.markDeleted(optionPath)
			if newOptions == nil {
				newOptions = maybeClone(optionsMessage, b.options).ProtoReflect()
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
	options *imageFilterOptions,
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
			if options.mutateInPlace {
				newList = list[:toIndex]
			} else {
				newList = append(newList, list[:toIndex]...)
			}
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
		isDirty = true
		sourcePathsRemap.markDeleted(sourcePath)
	}
	if isDirty {
		if len(newList) == 0 {
			return nil, true, nil
		}
		if options.mutateInPlace {
			// Zero out the remaining elements.
			for i := int(toIndex); i < len(list); i++ {
				list[i] = nil
			}
		}
		return newList, true, nil
	}
	return list, false, nil
}

func maybeClone[T proto.Message](value T, options *imageFilterOptions) T {
	if !options.mutateInPlace {
		return shallowClone(value)
	}
	return value
}

func shallowClone[T proto.Message](message T) T {
	src := message.ProtoReflect()
	dst := src.New()
	src.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		dst.Set(fd, v)
		return true
	})
	value, _ := dst.Interface().(T) // Safe to assert.
	return value
}

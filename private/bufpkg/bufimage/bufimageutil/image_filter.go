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
	imageIndex, err := newImageIndexForImage(image, options)
	if err != nil {
		return nil, err
	}
	filter, err := newFullNameFilter(imageIndex, options)
	if err != nil {
		return nil, err
	}
	b1, _ := json.MarshalIndent(filter.includes, "", "  ")
	b2, _ := json.MarshalIndent(filter.excludes, "", "  ")
	fmt.Println("includes", string(b1))
	fmt.Println("excludes", string(b2))
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
				fmt.Println("dropping import", imageFilePath)
				continue
			}
		}
		newImageFile, err := filterImageFile(
			imageFile,
			imageIndex,
			filter,
			isFileImported,
		)
		if err != nil {
			return nil, err
		}
		dirty = dirty || newImageFile != imageFile
		if newImageFile == nil {
			fmt.Println("dropping", imageFilePath)
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
	filter *fullNameFilter,
	isFileImported bool,
) (bufimage.ImageFile, error) {
	fileDescriptor := imageFile.FileDescriptorProto()
	var sourcePathsRemap sourcePathsRemapTrie
	isIncluded, err := addRemapsForFileDescriptor(
		&sourcePathsRemap,
		fileDescriptor,
		imageIndex,
		filter,
	)
	if err != nil {
		return nil, err
	}
	if len(sourcePathsRemap) == 0 {
		return imageFile, nil // No changes required.
	}
	if !isIncluded && !isFileImported {
		return nil, nil // Filtered out.
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
	filePath    string
	imageIndex  *imageIndex
	filter      *fullNameFilter
	fileImports map[string]struct{}
}

func addRemapsForFileDescriptor(
	sourcePathsRemap *sourcePathsRemapTrie,
	fileDescriptor *descriptorpb.FileDescriptorProto,
	imageIndex *imageIndex,
	filter *fullNameFilter,
) (bool, error) {
	packageName := protoreflect.FullName(fileDescriptor.GetPackage())
	if packageName != "" {
		// Check if filtered by the package name.
		if !filter.hasType(packageName) {
			return false, nil
		}
	}

	fileImports := make(map[string]struct{})
	builder := &sourcePathsBuilder{
		filePath:    fileDescriptor.GetName(),
		imageIndex:  imageIndex,
		filter:      filter,
		fileImports: fileImports,
	}
	sourcePath := make(protoreflect.SourcePath, 0, 8)

	// Walk the file descriptor.
	isIncluded := false
	hasMessages, err := addRemapsForSlice(sourcePathsRemap, packageName, append(sourcePath, fileMessagesTag), fileDescriptor.MessageType, builder.addRemapsForDescriptor)
	if err != nil {
		return false, err
	}
	isIncluded = isIncluded || hasMessages
	hasEnums, err := addRemapsForSlice(sourcePathsRemap, packageName, append(sourcePath, fileEnumsTag), fileDescriptor.EnumType, builder.addRemapsForEnum)
	if err != nil {
		return false, err
	}
	isIncluded = isIncluded || hasEnums
	hasServices, err := addRemapsForSlice(sourcePathsRemap, packageName, append(sourcePath, fileServicesTag), fileDescriptor.Service, builder.addRemapsForService)
	if err != nil {
		return false, err
	}
	isIncluded = isIncluded || hasServices
	hasExtensions, err := addRemapsForSlice(sourcePathsRemap, packageName, append(sourcePath, fileExtensionsTag), fileDescriptor.Extension, builder.addRemapsForField)
	if err != nil {
		return false, err
	}
	isIncluded = isIncluded || hasExtensions
	if err := builder.addRemapsForOptions(sourcePathsRemap, append(sourcePath, fileOptionsTag), fileDescriptor.Options); err != nil {
		return false, err
	}

	// Fix the imports to remove any that are no longer used.
	// TODO: handle unused dependencies, and keep them?
	if len(fileImports) != len(fileDescriptor.Dependency) {
		indexTo := int32(0)
		dependencyPath := append(sourcePath, fileDependencyTag)
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
		publicDependencyPath := append(sourcePath, filePublicDependencyTag)
		for indexFrom, publicDependency := range fileDescriptor.PublicDependency {
			path := append(publicDependencyPath, int32(indexFrom))
			indexTo := dependencyChanges[publicDependency]
			if indexTo == -1 {
				sourcePathsRemap.markDeleted(path)
			} else if indexTo != int32(indexFrom) {
				sourcePathsRemap.markMoved(path, indexTo)
			}
		}
		weakDependencyPath := append(sourcePath, fileWeakDependencyTag)
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
	return isIncluded, nil
}

func (b *sourcePathsBuilder) addRemapsForDescriptor(
	sourcePathsRemap *sourcePathsRemapTrie,
	parentName protoreflect.FullName,
	sourcePath protoreflect.SourcePath,
	descriptor *descriptorpb.DescriptorProto,
) (bool, error) {
	fullName := getFullName(parentName, descriptor)
	mode := b.filter.inclusionMode(fullName)
	if mode == inclusionModeNone {
		// The type is excluded.
		return false, nil
	}
	if mode == inclusionModeEnclosing {
		//fmt.Println("--- ENCLOSING:", fullName)
		// If the type is only enclosing, only the namespace matters.
		sourcePathsRemap.markDeleted(append(sourcePath, messageFieldsTag))
		sourcePathsRemap.markDeleted(append(sourcePath, messageOneofsTag))
		sourcePathsRemap.markDeleted(append(sourcePath, messageExtensionRangesTag))
		sourcePathsRemap.markDeleted(append(sourcePath, messageReservedRangesTag))
		sourcePathsRemap.markDeleted(append(sourcePath, messageReservedNamesTag))
	} else {
		if _, err := addRemapsForSlice(sourcePathsRemap, fullName, append(sourcePath, messageFieldsTag), descriptor.GetField(), b.addRemapsForField); err != nil {
			return false, err
		}
		if _, err := addRemapsForSlice(sourcePathsRemap, fullName, append(sourcePath, messageOneofsTag), descriptor.OneofDecl, b.addRemapsForOneof); err != nil {
			return false, err
		}
		for index, extensionRange := range descriptor.GetExtensionRange() {
			extensionRangeOptionsPath := append(sourcePath, messageExtensionRangesTag, int32(index), extensionRangeOptionsTag)
			if err := b.addRemapsForOptions(sourcePathsRemap, extensionRangeOptionsPath, extensionRange.GetOptions()); err != nil {
				return false, err
			}
		}
	}
	if _, err := addRemapsForSlice(sourcePathsRemap, fullName, append(sourcePath, messageExtensionsTag), descriptor.GetExtension(), b.addRemapsForField); err != nil {
		return false, err
	}
	if _, err := addRemapsForSlice(sourcePathsRemap, fullName, append(sourcePath, messageNestedMessagesTag), descriptor.NestedType, b.addRemapsForDescriptor); err != nil {
		return false, err
	}
	if _, err := addRemapsForSlice(sourcePathsRemap, fullName, append(sourcePath, messageEnumsTag), descriptor.EnumType, b.addRemapsForEnum); err != nil {
		return false, err
	}
	if err := b.addRemapsForOptions(sourcePathsRemap, append(sourcePath, messageOptionsTag), descriptor.GetOptions()); err != nil {
		return false, err
	}
	return true, nil
}

func (b *sourcePathsBuilder) addRemapsForEnum(
	sourcePathsRemap *sourcePathsRemapTrie,
	parentName protoreflect.FullName,
	sourcePath protoreflect.SourcePath,
	enum *descriptorpb.EnumDescriptorProto,
) (bool, error) {
	//fullName := b.imageIndex.ByDescriptor[enum]
	fullName := getFullName(parentName, enum)
	if !b.filter.hasType(fullName) {
		// The type is excluded, enum values cannot be excluded individually.
		return false, nil
	}
	fmt.Println("ENUM:", fullName, b.filter.inclusionMode(fullName))

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
	if !b.filter.hasType(fullName) {
		// The type is excluded, oneof fields cannot be excluded individually.
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
	if !b.filter.hasType(fullName) {
		// The type is excluded.
		return false, nil
	}
	// Walk the service methods.
	if _, err := addRemapsForSlice(sourcePathsRemap, fullName, append(sourcePath, serviceMethodsTag), service.Method, b.addRemapsForMethod); err != nil {
		return false, err
	}
	if err := b.addRemapsForOptions(sourcePathsRemap, append(sourcePath, serviceOptionsTag), service.GetOptions()); err != nil {
		return false, err
	}
	return true, nil
}

func (b *sourcePathsBuilder) addRemapsForMethod(
	sourcePathsRemap *sourcePathsRemapTrie,
	parentName protoreflect.FullName,
	sourcePath protoreflect.SourcePath,
	method *descriptorpb.MethodDescriptorProto,
) (bool, error) {
	fullName := getFullName(parentName, method)
	if !b.filter.hasType(fullName) {
		// The type is excluded.
		return false, nil
	}
	inputName := protoreflect.FullName(strings.TrimPrefix(method.GetInputType(), "."))
	if !b.filter.hasType(inputName) {
		// The input type is excluded.
		return false, fmt.Errorf("input type %s of method %s is excluded", inputName, fullName)
	}
	b.addRequiredType(inputName)
	outputName := protoreflect.FullName(strings.TrimPrefix(method.GetOutputType(), "."))
	if !b.filter.hasType(outputName) {
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
		// This is an extension field. Extensions are filtered by the name.
		extensionFullName := getFullName(parentName, field)
		if !b.filter.hasType(extensionFullName) {
			return false, nil
		}
		extendeeName := protoreflect.FullName(strings.TrimPrefix(field.GetExtendee(), "."))
		b.addRequiredType(extendeeName)

	} else if b.filter.inclusionMode(parentName) == inclusionModeEnclosing {
		return false, nil // The field is excluded.
	}
	switch field.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_ENUM,
		descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
		descriptorpb.FieldDescriptorProto_TYPE_GROUP:
		typeName := protoreflect.FullName(strings.TrimPrefix(field.GetTypeName(), "."))
		if !b.filter.hasType(typeName) {
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
	var err error
	options := optionsMessage.ProtoReflect()
	numFieldsToKeep := 0
	options.Range(func(fd protoreflect.FieldDescriptor, val protoreflect.Value) bool {
		if !b.filter.hasOption(fd.FullName(), fd.IsExtension()) {
			// Remove this option.
			optionPath := append(optionsPath, int32(fd.Number()))
			sourcePathsRemap.markDeleted(optionPath)
			return true
		}
		numFieldsToKeep++
		if fd.IsExtension() {
			// Add the extension type to the required types.
			b.addRequiredType(fd.FullName())
		}
		err = b.addOptionValue(fd, val)
		return err == nil
	})
	if numFieldsToKeep == 0 {
		// No options to keep.
		sourcePathsRemap.markDeleted(optionsPath)
	}
	return err
}

func (b *sourcePathsBuilder) addOptionValue(
	fieldDescriptor protoreflect.FieldDescriptor,
	value protoreflect.Value,
) error {
	switch {
	case fieldDescriptor.IsMap():
		if isMessageKind(fieldDescriptor.MapValue().Kind()) {
			var err error
			value.Map().Range(func(_ protoreflect.MapKey, v protoreflect.Value) bool {
				err = b.addOptionSingularValueForAny(v.Message())
				return err == nil
			})
			return err
		}
		return nil
	case isMessageKind(fieldDescriptor.Kind()):
		if fieldDescriptor.IsList() {
			listVal := value.List()
			for i := 0; i < listVal.Len(); i++ {
				if err := b.addOptionSingularValueForAny(listVal.Get(i).Message()); err != nil {
					return err
				}
			}
			return nil
		}
		return b.addOptionSingularValueForAny(value.Message())
	default:
		return nil
	}
}

func (b *sourcePathsBuilder) addOptionSingularValueForAny(message protoreflect.Message) error {
	md := message.Descriptor()
	if md.FullName() == anyFullName {
		// Found one!
		typeURLFd := md.Fields().ByNumber(1)
		if typeURLFd.Kind() != protoreflect.StringKind || typeURLFd.IsList() {
			// should not be possible...
			return nil
		}
		typeURL := message.Get(typeURLFd).String()
		pos := strings.LastIndexByte(typeURL, '/')
		msgType := protoreflect.FullName(typeURL[pos+1:])
		b.addRequiredType(msgType)
		// TODO: unmarshal the bytes to see if there are any nested Any messages
		return nil
	}
	// keep digging
	var err error
	message.Range(func(fd protoreflect.FieldDescriptor, val protoreflect.Value) bool {
		err = b.addOptionValue(fd, val)
		return err == nil
	})
	return err
}

func (b *sourcePathsBuilder) addRequiredType(fullName protoreflect.FullName) {
	info, ok := b.imageIndex.ByName[fullName]
	if !ok {
		panic(fmt.Sprintf("could not find file for %s", fullName))
	}
	file := info.imageFile
	if file.Path() != b.filePath {
		// This is an imported type.
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

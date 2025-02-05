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
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/protocompile/walk"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// ExcludeOptions returns an Image that omits all provided options.
//
// If the Image has none of the provided options, the original Image is returned.
// If the Image has any of the provided options, a new Image is returned.
// If an import is no longer imported by any file, it is removed from the Image.
//
// The returned Image will be a shallow copy of the original Image.
// The FileDescriptors in the returned Image.Files will be a shallow copy of the original FileDescriptorProto.
//
// The options are specified by their type name, e.g. "buf.validate.message".
func ExcludeOptions(image bufimage.Image, options ...string) (bufimage.Image, error) {
	if len(options) == 0 {
		return image, nil
	}
	imageFiles := image.Files()
	newImageFiles := make([]bufimage.ImageFile, 0, len(image.Files()))
	importsByFilePath := make(map[string]struct{})

	optionsByName := make(map[string]struct{}, len(options))
	for _, option := range options {
		optionsByName[option] = struct{}{}
	}
	optionsFilter := func(fd protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		_, ok := optionsByName[string(fd.FullName())]
		return !ok
	}
	filter := newOptionsFilter(image.Resolver(), optionsFilter)

	// Loop over image files in revserse DAG order. Imports that are no longer
	// imported by a previous file are dropped from the image.
	dirty := false
	for i := len(image.Files()) - 1; i >= 0; i-- {
		imageFile := imageFiles[i]
		imageFilePath := imageFile.Path()
		if imageFile.IsImport() {
			// Check if this import is still used.
			if _, isImportUsed := importsByFilePath[imageFilePath]; !isImportUsed {
				continue
			}
		}
		oldFileDescriptor := imageFile.FileDescriptorProto()
		newFileDescriptor, err := filter.fileDescriptor(oldFileDescriptor, imageFilePath)
		if err != nil {
			return nil, err
		}
		for _, filePath := range newFileDescriptor.Dependency {
			importsByFilePath[filePath] = struct{}{}
		}
		// Create a new image file with the modified file descriptor.
		if newFileDescriptor != oldFileDescriptor {
			dirty = true
			imageFile, err = bufimage.NewImageFile(
				newFileDescriptor,
				imageFile.FullName(),
				imageFile.CommitID(),
				imageFile.ExternalPath(),
				imageFile.LocalPath(),
				imageFile.IsImport(),
				imageFile.IsSyntaxUnspecified(),
				imageFile.UnusedDependencyIndexes(),
			)
			if err != nil {
				return nil, err
			}
		}
		newImageFiles = append(newImageFiles, imageFile)
	}
	if dirty {
		// Revserse the image files back to DAG order.
		slices.Reverse(newImageFiles)
		return bufimage.NewImage(newImageFiles)
	}
	return image, nil
}

// protoOptionsFilter is a filter for options.
//
// This filter is applied to FileDescriptorProto to remove options. A new shallow
// copy of the FileDescriptorProto is returned with the options removed. If no
// options are removed, the original FileDescriptorProto is returned.
type protoOptionsFilter struct {
	// resolver is used to resvole parent files for required types.
	resolver protodesc.Resolver
	// optionsFilter returns true if the option should be kept.
	optionsFilter func(protoreflect.FieldDescriptor, protoreflect.Value) bool

	// sourcePathsRemaps is a trie of source path remaps. Reset per file iteration.
	sourcePathRemaps sourcePathsRemapTrie
	// requiredTypes is a set of required types. Reset per file iteration.
	requiredTypes map[protoreflect.FullName]struct{}
}

func newOptionsFilter(resolver protoencoding.Resolver, optionsFilter func(protoreflect.FieldDescriptor, protoreflect.Value) bool) *protoOptionsFilter {
	return &protoOptionsFilter{
		resolver:      resolver,
		optionsFilter: optionsFilter,
	}
}

func (f *protoOptionsFilter) fileDescriptor(
	fileDescriptor *descriptorpb.FileDescriptorProto,
	filePath string,
) (*descriptorpb.FileDescriptorProto, error) {
	if f.requiredTypes == nil {
		f.requiredTypes = make(map[protoreflect.FullName]struct{})
	}
	// Check file options.
	if options := fileDescriptor.GetOptions(); options != nil {
		optionsPath := []int32{fileOptionsTag}
		if err := f.options(options, optionsPath); err != nil {
			return nil, err
		}
	}
	// Walk the file descriptor, collecting required types and marking options for deletion.
	if err := walk.DescriptorProtosWithPath(fileDescriptor, f.visit); err != nil {
		return nil, err
	}
	if len(f.sourcePathRemaps) == 0 {
		return fileDescriptor, nil // No changes required.
	}

	// Recursively apply the source path remaps.
	newFileDescriptor := shallowClone(fileDescriptor)
	if err := remapFileDescriptor(
		newFileDescriptor,
		f.sourcePathRemaps,
	); err != nil {
		return nil, err
	}
	// Convert the required types to imports.
	fileImports := make(map[string]struct{}, len(fileDescriptor.Dependency))
	for requiredType := range f.requiredTypes {
		descriptor, err := f.resolver.FindDescriptorByName(requiredType)
		if err != nil {
			return nil, fmt.Errorf("couldn't find type %s: %w", requiredType, err)
		}
		importPath := descriptor.ParentFile().Path()
		if _, ok := fileImports[importPath]; ok || importPath == filePath {
			continue
		}
		fileImports[importPath] = struct{}{}
	}
	// Fix imports.
	if len(fileImports) != len(fileDescriptor.Dependency) {
		i := 0
		dependencyPath := []int32{fileDependencyTag}
		dependencyChanges := make([]int32, len(fileDescriptor.Dependency))
		newFileDescriptor.Dependency = make([]string, 0, len(fileImports))
		for index, dependency := range fileDescriptor.Dependency {
			path := append(dependencyPath, int32(index))
			if _, ok := fileImports[dependency]; ok {
				newFileDescriptor.Dependency = append(newFileDescriptor.Dependency, dependency)
				dependencyChanges[index] = int32(i)
				if i != index {
					f.sourcePathRemaps.markMoved(path, int32(i))
				}
				i++
			} else {
				f.sourcePathRemaps.markDeleted(path)
				dependencyChanges[index] = -1
			}
		}
		publicDependencyPath := []int32{filePublicDependencyTag}
		newFileDescriptor.PublicDependency = make([]int32, 0, len(fileDescriptor.PublicDependency))
		for index, publicDependency := range fileDescriptor.PublicDependency {
			path := append(publicDependencyPath, int32(index))
			newPublicDependency := dependencyChanges[publicDependency]
			if newPublicDependency == -1 {
				f.sourcePathRemaps.markDeleted(path)
			} else {
				newFileDescriptor.PublicDependency = append(newFileDescriptor.PublicDependency, newPublicDependency)
				if newPublicDependency != int32(index) {
					f.sourcePathRemaps.markMoved(path, newPublicDependency)
				}
			}
		}
		weakDependencyPath := []int32{fileWeakDependencyTag}
		newFileDescriptor.WeakDependency = make([]int32, 0, len(fileDescriptor.WeakDependency))
		for index, weakDependency := range fileDescriptor.WeakDependency {
			path := append(weakDependencyPath, int32(index))
			newWeakDependency := dependencyChanges[weakDependency]
			if newWeakDependency == -1 {
				f.sourcePathRemaps.markDeleted(path)
			} else {
				newFileDescriptor.WeakDependency = append(newFileDescriptor.WeakDependency, newWeakDependency)
				if newWeakDependency != int32(index) {
					f.sourcePathRemaps.markMoved(path, newWeakDependency)
				}
			}
		}
	}
	// Remap the source code info.
	if locations := fileDescriptor.SourceCodeInfo.GetLocation(); len(locations) > 0 {
		newLocations := make([]*descriptorpb.SourceCodeInfo_Location, 0, len(locations))
		for _, location := range locations {
			oldPath := location.Path
			newPath, noComment := f.sourcePathRemaps.newPath(oldPath)
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
	// Cleanup.
	f.sourcePathRemaps = f.sourcePathRemaps[:0]
	for key := range f.requiredTypes {
		delete(f.requiredTypes, key)
	}
	return newFileDescriptor, nil
}

func (f *protoOptionsFilter) visit(fullName protoreflect.FullName, sourcePath protoreflect.SourcePath, descriptor proto.Message) error {
	switch descriptor := descriptor.(type) {
	case *descriptorpb.FileDescriptorProto:
		// File options are handled at the top level, before walking the file.
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

func (f *protoOptionsFilter) options(
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
		if !f.optionsFilter(fd, v) {
			// Remove this option.
			optionPath := append(optionsPath, int32(fd.Number()))
			f.sourcePathRemaps.markDeleted(optionPath)
			return true
		}
		numFieldsToKeep++
		if fd.IsExtension() {
			// Add the extension type to the required types.
			f.requiredTypes[fd.FullName()] = struct{}{}
		}
		return true
	})
	if numFieldsToKeep == 0 {
		// No options to keep.
		f.sourcePathRemaps.markDeleted(optionsPath)
	}
	return nil
}

func (f *protoOptionsFilter) addFieldType(field *descriptorpb.FieldDescriptorProto) error {
	if field.Extendee != nil {
		// This is an extension field.
		extendee := strings.TrimPrefix(field.GetExtendee(), ".")
		f.requiredTypes[protoreflect.FullName(extendee)] = struct{}{}
	}
	switch field.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_ENUM,
		descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
		descriptorpb.FieldDescriptorProto_TYPE_GROUP:
		typeName := strings.TrimPrefix(field.GetTypeName(), ".")
		f.requiredTypes[protoreflect.FullName(typeName)] = struct{}{}
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

// TODO: rewrite using protoreflect.
func remapFileDescriptor(
	fileDescriptor *descriptorpb.FileDescriptorProto,
	sourcePathRemaps sourcePathsRemapTrie,
) error {
	for _, remapNode := range sourcePathRemaps {
		switch remapNode.oldIndex {
		case fileDependencyTag:
			// Dependencies are handled after message remaps.
			return fmt.Errorf("unexpected dependency move %d to %d", remapNode.oldIndex, remapNode.newIndex)
		case filePublicDependencyTag:
			return fmt.Errorf("unexpected public dependency move %d to %d", remapNode.oldIndex, remapNode.newIndex)
		case fileWeakDependencyTag:
			return fmt.Errorf("unexpected weak dependency move %d to %d", remapNode.oldIndex, remapNode.newIndex)
		case fileMessagesTag:
			if err := remapSlice(
				&fileDescriptor.MessageType,
				remapNode,
				remapMessageDescriptor,
			); err != nil {
				return err
			}
		case fileEnumsTag:
			if err := remapSlice(
				&fileDescriptor.EnumType,
				remapNode,
				remapEnumDescriptor,
			); err != nil {
				return err
			}
		case fileServicesTag:
			if err := remapSlice(
				&fileDescriptor.Service,
				remapNode,
				remapServiceDescriptor,
			); err != nil {
				return err
			}
		case fileExtensionsTag:
			if err := remapSlice(
				&fileDescriptor.Extension,
				remapNode,
				remapFieldDescriptor,
			); err != nil {
				return err
			}
		case fileOptionsTag:
			if err := remapMessage(
				&fileDescriptor.Options,
				remapNode,
				remapOptionsDescriptor,
			); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpected file index %d", remapNode.oldIndex)
		}
	}
	return nil
}

func remapMessageDescriptor(
	messageDescriptor *descriptorpb.DescriptorProto,
	sourcePathRemaps sourcePathsRemapTrie,
) error {
	for _, remapNode := range sourcePathRemaps {
		switch remapNode.oldIndex {
		case messageFieldsTag:
			if err := remapSlice(
				&messageDescriptor.Field,
				remapNode,
				remapFieldDescriptor,
			); err != nil {
				return err
			}
		case messageNestedMessagesTag:
			if err := remapSlice(
				&messageDescriptor.NestedType,
				remapNode,
				remapMessageDescriptor,
			); err != nil {
				return err
			}
		case messageEnumsTag:
			if err := remapSlice(
				&messageDescriptor.EnumType,
				remapNode,
				remapEnumDescriptor,
			); err != nil {
				return err
			}
		case messageExtensionsTag:
			if err := remapSlice(
				&messageDescriptor.Extension,
				remapNode,
				remapFieldDescriptor,
			); err != nil {
				return err
			}
		case messageOptionsTag:
			if err := remapMessage(
				&messageDescriptor.Options,
				remapNode,
				remapOptionsDescriptor,
			); err != nil {
				return err
			}
		case messageOneofsTag:
			if err := remapSlice(
				&messageDescriptor.OneofDecl,
				remapNode,
				remapOneofDescriptor,
			); err != nil {
				return err
			}
		case messageExtensionRangesTag:
			if err := remapSlice(
				&messageDescriptor.ExtensionRange,
				remapNode,
				remapExtensionRangeDescriptor,
			); err != nil {
				return err
			}
		case messageReservedRangesTag:
			// TODO: handle reserved ranges tag.
			return fmt.Errorf("unexpected reserved tags move %d to %d", remapNode.oldIndex, remapNode.newIndex)
		case messageReservedNamesTag:
			// TODO: handle reserved names.
			return fmt.Errorf("unexpected reserved names move %d to %d", remapNode.oldIndex, remapNode.newIndex)
		default:
			return fmt.Errorf("unexpected message index %d", remapNode.oldIndex)
		}
	}
	return nil
}

func remapFieldDescriptor(
	fieldDescriptor *descriptorpb.FieldDescriptorProto,
	sourcePathRemaps sourcePathsRemapTrie,
) error {
	for _, remapNode := range sourcePathRemaps {
		switch remapNode.oldIndex {
		case fieldOptionsTag:
			if err := remapMessage(
				&fieldDescriptor.Options,
				remapNode,
				remapOptionsDescriptor,
			); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpected field index %d", remapNode.oldIndex)
		}
	}
	return nil
}

func remapOneofDescriptor(
	oneofDescriptor *descriptorpb.OneofDescriptorProto,
	sourcePathRemaps sourcePathsRemapTrie,
) error {
	for _, remapNode := range sourcePathRemaps {
		switch remapNode.oldIndex {
		case oneofOptionsTag:
			if err := remapMessage(
				&oneofDescriptor.Options,
				remapNode,
				remapOptionsDescriptor,
			); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpected oneof index %d", remapNode.oldIndex)
		}
	}
	return nil
}

func remapEnumDescriptor(
	enumDescriptor *descriptorpb.EnumDescriptorProto,
	sourcePathRemaps sourcePathsRemapTrie,
) error {
	for _, remapNode := range sourcePathRemaps {
		switch remapNode.oldIndex {
		case enumValuesTag:
			if err := remapSlice(
				&enumDescriptor.Value,
				remapNode,
				remapEnumValueDescriptor,
			); err != nil {
				return err
			}
		case enumOptionsTag:
			if err := remapMessage(
				&enumDescriptor.Options,
				remapNode,
				remapOptionsDescriptor,
			); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpected enum index %d", remapNode.oldIndex)
		}
	}
	return nil
}

func remapEnumValueDescriptor(
	enumValueDescriptor *descriptorpb.EnumValueDescriptorProto,
	sourcePathRemaps sourcePathsRemapTrie,
) error {
	for _, remapNode := range sourcePathRemaps {
		switch remapNode.oldIndex {
		case enumValueOptionsTag:
			if err := remapMessage(
				&enumValueDescriptor.Options,
				remapNode,
				remapOptionsDescriptor,
			); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpected enum value index %d", remapNode.oldIndex)
		}
	}
	return nil
}

func remapExtensionRangeDescriptor(
	extensionRangeDescriptor *descriptorpb.DescriptorProto_ExtensionRange,
	sourcePathRemaps sourcePathsRemapTrie,
) error {
	for _, remapNode := range sourcePathRemaps {
		switch remapNode.oldIndex {
		case extensionRangeOptionsTag:
			if err := remapMessage(
				&extensionRangeDescriptor.Options,
				remapNode,
				remapOptionsDescriptor,
			); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpected extension range index %d", remapNode.oldIndex)
		}
	}
	return nil
}

func remapServiceDescriptor(
	serviceDescriptor *descriptorpb.ServiceDescriptorProto,
	sourcePathRemaps sourcePathsRemapTrie,
) error {
	for _, remapNode := range sourcePathRemaps {
		switch remapNode.oldIndex {
		case serviceMethodsTag:
			if err := remapSlice(
				&serviceDescriptor.Method,
				remapNode,
				remapMethodDescriptor,
			); err != nil {
				return err
			}
		case serviceOptionsTag:
			if err := remapMessage(
				&serviceDescriptor.Options,
				remapNode,
				remapOptionsDescriptor,
			); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpected service index %d", remapNode.oldIndex)
		}
	}
	return nil
}

func remapMethodDescriptor(
	methodDescriptor *descriptorpb.MethodDescriptorProto,
	sourcePathRemaps sourcePathsRemapTrie,
) error {
	for _, remapNode := range sourcePathRemaps {
		switch remapNode.oldIndex {
		case methodOptionsTag:
			if err := remapMessage(
				&methodDescriptor.Options,
				remapNode,
				remapOptionsDescriptor,
			); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpected method index %d", remapNode.oldIndex)
		}
	}
	return nil
}

func remapOptionsDescriptor[T proto.Message](
	optionsMessage T,
	sourcePathRemaps sourcePathsRemapTrie,
) error {
	options := optionsMessage.ProtoReflect()
	if !options.IsValid() {
		return fmt.Errorf("invalid options %T", optionsMessage)
	}

	// Create a mapping of fields to handle extensions. Extensions are not in the descriptor.
	fields := make(map[int32]protoreflect.FieldDescriptor)
	options.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		fields[int32(fd.Number())] = fd
		return true
	})
	for _, remapNode := range sourcePathRemaps {
		fd := fields[remapNode.oldIndex]
		if fd == nil {
			return fmt.Errorf("unexpected field number %d", remapNode.oldIndex)
		}
		if remapNode.newIndex != -1 {
			return fmt.Errorf("unexpected options move %d to %d", remapNode.oldIndex, remapNode.newIndex)
		}
		options.Clear(fd)
	}
	return nil
}

func remapMessage[T proto.Message](
	ptr *T,
	remapNode *sourcePathsRemapTrieNode,
	updateFn func(T, sourcePathsRemapTrie) error,
) error {
	var zero T
	if remapNode.newIndex == -1 {
		// Remove the message.
		*ptr = zero
		return nil
	}
	message := shallowClone(*ptr)
	if !message.ProtoReflect().IsValid() {
		*ptr = zero
		return nil // Nothing to update.
	}
	if remapNode.oldIndex != remapNode.newIndex {
		return fmt.Errorf("unexpected message move %d to %d", remapNode.oldIndex, remapNode.newIndex)
	}
	if err := updateFn(message, remapNode.children); err != nil {
		return err
	}
	*ptr = message
	return nil
}

func remapSlice[T proto.Message](
	slice *[]T,
	remapNode *sourcePathsRemapTrieNode,
	updateFn func(T, sourcePathsRemapTrie) error,
) error {
	if remapNode.newIndex == -1 {
		// Remove the slice.
		*slice = nil
		return nil
	}
	*slice = slices.Clone(*slice) // Shallow clone
	for _, child := range remapNode.children {
		if child.oldIndex != child.newIndex {
			// TODO: add support for deleting and moving elements.
			// If children are present, the element must be copied.
			// Otherwise the element can only be moved.
			return fmt.Errorf("unexpected child slice move %d to %d", child.oldIndex, child.newIndex)
		}
		index := child.oldIndex
		item := shallowClone((*slice)[index])
		if err := updateFn(item, child.children); err != nil {
			return err
		}
		(*slice)[index] = item
	}
	return nil
}

func shallowClone[T proto.Message](src T) T {
	sm := src.ProtoReflect()
	dm := sm.New()
	sm.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		dm.Set(fd, v)
		return true
	})
	return dm.Interface().(T)
}

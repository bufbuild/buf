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
	f.sourcePathRemaps = f.sourcePathRemaps[:0]
	for key := range f.requiredTypes {
		delete(f.requiredTypes, key)
	}
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
		indexTo := int32(0)
		dependencyPath := []int32{fileDependencyTag}
		dependencyChanges := make([]int32, len(fileDescriptor.Dependency))
		for indexFrom, dependency := range fileDescriptor.Dependency {
			path := append(dependencyPath, int32(indexFrom))
			if _, ok := fileImports[dependency]; ok {
				dependencyChanges[indexFrom] = indexTo
				if indexTo != int32(indexFrom) {
					f.sourcePathRemaps.markMoved(path, indexTo)
				}
				indexTo++
			} else {
				f.sourcePathRemaps.markDeleted(path)
				dependencyChanges[indexFrom] = -1
			}
		}
		publicDependencyPath := []int32{filePublicDependencyTag}
		for indexFrom, publicDependency := range fileDescriptor.PublicDependency {
			path := append(publicDependencyPath, int32(indexFrom))
			indexTo := dependencyChanges[publicDependency]
			if indexTo == -1 {
				f.sourcePathRemaps.markDeleted(path)
			} else if indexTo != int32(indexFrom) {
				f.sourcePathRemaps.markMoved(path, indexTo)
			}
		}
		weakDependencyPath := []int32{fileWeakDependencyTag}
		for indexFrom, weakDependency := range fileDescriptor.WeakDependency {
			path := append(weakDependencyPath, int32(indexFrom))
			indexTo := dependencyChanges[weakDependency]
			if indexTo == -1 {
				f.sourcePathRemaps.markDeleted(path)
			} else if indexTo != int32(indexFrom) {
				f.sourcePathRemaps.markMoved(path, indexTo)
			}
		}
	}
	// Remap the source code info.
	fileDescriptorMessage := fileDescriptor.ProtoReflect()
	newFileDescriptorMessage, err := remapMessageReflect(fileDescriptorMessage, f.sourcePathRemaps)
	if err != nil {
		return nil, err
	}
	newFileDescriptor, _ := newFileDescriptorMessage.Interface().(*descriptorpb.FileDescriptorProto)
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

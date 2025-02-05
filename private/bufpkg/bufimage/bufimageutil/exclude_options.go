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
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// ExcludeOptions returns a new Image that omits all provided options.
//
// If the Image has none of the provided options, the original Image is returned.
// If the Image has any of the provided options, a new Image is returned.
//
// The returned Image will be a shallow copy of the original Image.
// The FileDescriptors in the returned Image.Files will be a shallow copy of the original FileDescriptorProto.
func ExcludeOptions(image bufimage.Image, options []string) (bufimage.Image, error) {
	resolver := image.Resolver()
	dirty := false

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

	filter := &protoFilter{
		resolver:      resolver,
		imports:       importsByFilePath,
		optionsFilter: optionsFilter,
	}

	// Loop over image files in revserse DAG order. Imports that are no longer
	// imported by a previous file will then be dropped.
	for i := len(image.Files()) - 1; i >= 0; i-- {
		imageFile := imageFiles[i]
		imageFilePath := imageFile.Path()
		fmt.Println("---\nFILE", imageFile.FullName(), imageFile.Path(), imageFile.IsImport())
		if imageFile.IsImport() {
			// Check if this import is still used.
			if _, isImportUsed := importsByFilePath[imageFilePath]; !isImportUsed {
				fmt.Println("dropping import", imageFilePath)
				continue
			}
		}
		//
		fileDescriptor, err := filter.fileDescriptor(
			imageFile.FileDescriptorProto(),
			imageFilePath,
		)
		if err != nil {
			return nil, err
		}
		for _, filePath := range fileDescriptor.Dependency {
			importsByFilePath[filePath] = struct{}{}
		}

		// Create a new image file with the modified file descriptor.
		if fileDescriptor != imageFile.FileDescriptorProto() {
			dirty = true
			imageFile, err = bufimage.NewImageFile(
				fileDescriptor,
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

type protoFilter struct {
	resolver      protoencoding.Resolver
	imports       map[string]struct{}
	optionsFilter func(protoreflect.FieldDescriptor, protoreflect.Value) bool

	sourcePathRemaps sourcePathsRemapTrie
	requiredTypes    map[protoreflect.FullName]struct{}
}

func (f *protoFilter) fileDescriptor(
	fileDescriptor *descriptorpb.FileDescriptorProto,
	filePath string,
) (*descriptorpb.FileDescriptorProto, error) {
	if f.requiredTypes == nil {
		f.requiredTypes = make(map[protoreflect.FullName]struct{})
	}
	if err := walk.DescriptorProtosWithPathEnterAndExit(fileDescriptor, f.enter, f.exit); err != nil {
		fmt.Println("walk error", err)
		return nil, err
	}

	if len(f.sourcePathRemaps) == 0 {
		// No changes required.
		fmt.Println("no changes for", filePath)
		return fileDescriptor, nil
	}

	// FileDescriptor has updates, shallow clone and apply the updates.
	newFileDescriptor := shallowClone(fileDescriptor)

	// Recursively apply the source path remaps.
	fmt.Println("sourcePathRemaps", f.sourcePathRemaps)
	fmt.Println("SOURCE PATH REMAP")
	for _, sourcePathRemap := range f.sourcePathRemaps {
		fmt.Println("\t", sourcePathRemap)
	}
	if err := remapFileDescriptor(
		newFileDescriptor,
		f.sourcePathRemaps,
	); err != nil {
		return nil, err
	}

	// Convert the required types to imports.
	fileImports := make(map[string]struct{}, len(fileDescriptor.Dependency))
	for requiredType := range f.requiredTypes {
		fmt.Println("REQUIRED TYPE", requiredType)
		descriptor, err := f.resolver.FindDescriptorByName(requiredType)
		if err != nil {
			return nil, fmt.Errorf("couldn't find type %s: %w", requiredType, err)
		}
		importPath := descriptor.ParentFile().Path()
		if _, ok := fileImports[importPath]; ok || importPath == filePath {
			continue
		}
		fileImports[importPath] = struct{}{}
		if f.imports == nil {
			f.imports = make(map[string]struct{})
		}
		f.imports[importPath] = struct{}{}
		fmt.Println("IMPORT", descriptor.ParentFile().Path())
	}

	// Fix imports.
	if len(fileImports) != len(fileDescriptor.Dependency) {
		fmt.Println("Fixing imports", len(fileImports), len(fileDescriptor.Dependency))
		fmt.Println("fileImports", fileImports)
		fmt.Println("fileDescriptor.Dependency", fileDescriptor.Dependency)
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

func (f *protoFilter) enter(fullName protoreflect.FullName, sourcePath protoreflect.SourcePath, descriptor proto.Message) error {
	//fmt.Println("enter", fullName)
	switch descriptor := descriptor.(type) {
	case *descriptorpb.FileDescriptorProto:
		// We don't filter file options.
		return nil
	case *descriptorpb.DescriptorProto:
		fmt.Printf("descriptor options %+v\n", descriptor.GetOptions())

		optionsPath := append(sourcePath, messageOptionsTag)
		if err := f.options(descriptor.GetOptions(), optionsPath); err != nil {
			return err
		}
		return nil
	case *descriptorpb.FieldDescriptorProto:
		// Add the field type to the required types.
		if err := f.addFieldType(descriptor); err != nil {
			return err
		}

		optionsPath := append(sourcePath, fieldOptionsTag)
		if err := f.options(descriptor.GetOptions(), optionsPath); err != nil {
			return err
		}
		return nil
	case *descriptorpb.OneofDescriptorProto:
		optionsPath := append(sourcePath, oneofOptionsTag)
		if err := f.options(descriptor.GetOptions(), optionsPath); err != nil {
			return err
		}
		return nil
	case *descriptorpb.EnumDescriptorProto:
		optionsPath := append(sourcePath, enumOptionsTag)
		if err := f.options(descriptor.GetOptions(), optionsPath); err != nil {
			return err
		}
		return nil
	case *descriptorpb.EnumValueDescriptorProto:
		optionsPath := append(sourcePath, enumValueOptionsTag)
		if err := f.options(descriptor.GetOptions(), optionsPath); err != nil {
			return err
		}
		return nil
	case *descriptorpb.ServiceDescriptorProto:
		optionsPath := append(sourcePath, serviceOptionsTag)
		if err := f.options(descriptor.GetOptions(), optionsPath); err != nil {
			return err
		}
		return nil
	case *descriptorpb.MethodDescriptorProto:
		optionsPath := append(sourcePath, methodOptionsTag)
		if err := f.options(descriptor.GetOptions(), optionsPath); err != nil {
			return err
		}
		return nil
	case *descriptorpb.DescriptorProto_ExtensionRange:
		return nil
	default:
		return fmt.Errorf("unexpected message type %T", descriptor)
	}
}

func (f *protoFilter) exit(fullName protoreflect.FullName, southPath protoreflect.SourcePath, message proto.Message) error {
	//fmt.Println("exit", fullName)
	return nil
}

func (f *protoFilter) options(
	optionsMessage proto.Message,
	optionsPath protoreflect.SourcePath,
) error {
	if optionsMessage == nil {
		return nil
	}
	options := optionsMessage.ProtoReflect()
	if !options.IsValid() {
		f.sourcePathRemaps.markDeleted(optionsPath)
		return nil // No options to strip.
	}
	b, _ := protojson.Marshal(optionsMessage)
	fmt.Println("options", string(b))
	optionsName := string(options.Descriptor().FullName())
	fmt.Println("optionsName", optionsName, options)

	numFieldsToKeep := 0
	options.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		fmt.Println("field", fd.FullName(), fd.Number(), v)
		if !f.optionsFilter(fd, v) {
			// Remove this option.
			optionPath := append(optionsPath, int32(fd.Number()))
			f.sourcePathRemaps.markDeleted(optionPath)
			fmt.Println("removing", fd.FullName(), optionsPath)
			return true
		}
		numFieldsToKeep++
		if !fd.IsExtension() {
			return true
		}
		// Add the extension type to the required types.
		switch fd.Kind() {
		case protoreflect.MessageKind, protoreflect.GroupKind:
			fmt.Println("ADDING MESSAGE EXTENSION", fd.Message().FullName(), "for", fd.FullName())
			// TODO: get the required types.
			f.requiredTypes[fd.Message().FullName()] = struct{}{}
		case protoreflect.EnumKind:
			fmt.Println("ADDING ENUM EXTENSION", fd.Enum().FullName(), "for", fd.FullName())
			f.requiredTypes[fd.Enum().FullName()] = struct{}{}
		}
		return true
	})
	if numFieldsToKeep == 0 {
		// No options to keep.
		f.sourcePathRemaps.markDeleted(optionsPath)
	}
	return nil
}

func (f *protoFilter) addFieldType(field *descriptorpb.FieldDescriptorProto) error {
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
	// nothing to follow, custom options handled below.
	default:
		return fmt.Errorf("unknown field type %d", field.GetType())
	}
	return nil
}

func remapFileDescriptor(
	fileDescriptor *descriptorpb.FileDescriptorProto,
	sourcePathRemaps sourcePathsRemapTrie,
) error {
	fmt.Println("REMAP FILE DESCRIPTOR")
	for _, remapNode := range sourcePathRemaps {
		fmt.Println("\tremapNode", remapNode)
		switch remapNode.oldIndex {
		case fileDependencyTag:
			return fmt.Errorf("unexpected dependency move %d -> %d", remapNode.oldIndex, remapNode.newIndex)
		case filePublicDependencyTag:
			return fmt.Errorf("unexpected public dependency move %d -> %d", remapNode.oldIndex, remapNode.newIndex)
		case fileWeakDependencyTag:
			return fmt.Errorf("unexpected weak dependency move %d -> %d", remapNode.oldIndex, remapNode.newIndex)
		case fileMessagesTag:
			if err := remapSlice(
				&fileDescriptor.MessageType,
				remapNode,
				remapMessageDescriptor,
			); err != nil {
				return err
			}
		case fileEnumsTag:
			fmt.Println("REMAP ENUMS???")
		case fileServicesTag:
			fmt.Println("REMAP SERVICES")
		case fileExtensionsTag:
			fmt.Println("REMAP EXTENSIONS")
		case fileOptionsTag:
			fmt.Println("REMAP OPTIONS???")
		default:
			panic(fmt.Errorf("unexpected file index %d", remapNode.oldIndex))
		}
	}
	return nil
}

func remapMessageDescriptor(
	messageDescriptor *descriptorpb.DescriptorProto,
	sourcePathRemaps sourcePathsRemapTrie,
) error {
	fmt.Println("REMAP MESSAGE DESCRIPTOR")
	for _, remapNode := range sourcePathRemaps {
		fmt.Println("\tremapNode", remapNode)
		switch remapNode.oldIndex {
		case messageFieldsTag:
		case messageNestedMessagesTag:
		case messageEnumsTag:
		case messageExtensionsTag:
		case messageOptionsTag:
			fmt.Println("GOT THE OPTIONS TO EDIT")
			if messageDescriptor.Options == nil {
				continue
			}
			if remapNode.newIndex == -1 {
				// Remove the options.
				fmt.Println("REMOVING ALL OPTIONS")
				messageDescriptor.Options = nil
				continue
			}
			messageDescriptor.Options = shallowClone(messageDescriptor.Options)
			if err := remapOptionsDescriptor(
				messageDescriptor.Options,
				remapNode.children,
			); err != nil {
				return err
			}
		case messageOneofsTag:
		case messageExtensionRangesTag:
		case messageReservedRangesTag:
		case messageReservedNamesTag:
		default:
			panic(fmt.Errorf("unexpected message index %d", remapNode.oldIndex))
		}
	}
	return nil
}

func remapOptionsDescriptor(
	optionsMessage proto.Message,
	sourcePathRemaps sourcePathsRemapTrie,
) error {
	fmt.Println("REMAP OPTIONS DESCRIPTOR!!", optionsMessage)
	options := optionsMessage.ProtoReflect()
	if !options.IsValid() {
		return fmt.Errorf("unexpected invalid options %T", optionsMessage)
	}

	// TODO: do this better. How do we get an extension by field number?
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
			return fmt.Errorf("unexpected options move %d -> %d", remapNode.oldIndex, remapNode.newIndex)
		}
		options.Clear(fd)
	}
	return nil
}

func remapSlice[T proto.Message](
	slice *[]T,
	remapNode *sourcePathsRemapTrieNode,
	updateFn func(T, sourcePathsRemapTrie) error,
) error {
	if remapNode.newIndex == -1 {
		// Remove the slice. Supported but not used.
		*slice = nil
		return nil
	}
	*slice = slices.Clone(*slice) // Shallow clone
	for _, child := range remapNode.children {
		if child.oldIndex != child.newIndex {
			// Moving of elements is not supported. This would
			// require knowing if a child was edited, to determine
			// if we have to copy the item, vs just moving it.
			return fmt.Errorf("unexpected child slice move %d -> %d", child.oldIndex, child.newIndex)
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

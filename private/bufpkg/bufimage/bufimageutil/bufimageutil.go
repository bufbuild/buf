// Copyright 2020-2022 Buf Technologies, Inc.
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
	"context"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/protosource"
	"google.golang.org/protobuf/types/descriptorpb"
)

// NewInputFiles converts the ImageFiles to InputFiles.
//
// Since protosource is a pkg package, it cannot depend on bufmoduleref, which has the
// definition for bufmoduleref.ModuleIdentity, so we have our own interfaces for this
// in protosource. Given Go's type system, we need to do a conversion here.
func NewInputFiles(imageFiles []bufimage.ImageFile) []protosource.InputFile {
	inputFiles := make([]protosource.InputFile, len(imageFiles))
	for i, imageFile := range imageFiles {
		inputFiles[i] = newInputFile(imageFile)
	}
	return inputFiles
}

// FreeMessageRangeStrings gets the free MessageRange strings for the target files.
//
// Recursive.
func FreeMessageRangeStrings(
	ctx context.Context,
	filePaths []string,
	image bufimage.Image,
) ([]string, error) {
	var s []string
	for _, filePath := range filePaths {
		imageFile := image.GetFile(filePath)
		if imageFile == nil {
			return nil, fmt.Errorf("unexpected nil image file: %q", filePath)
		}
		file, err := protosource.NewFile(newInputFile(imageFile))
		if err != nil {
			return nil, err
		}
		for _, message := range file.Messages() {
			s = freeMessageRangeStringsRec(s, message)
		}
	}
	return s, nil
}

// ImageFilteredByTypes returns a minimal image containing only the descriptors
// required to interpret types.
func ImageFilteredByTypes(image bufimage.Image, types []string) (bufimage.Image, error) {
	imageIndex, err := newImageIndexForImage(image)
	if err != nil {
		return nil, err
	}
	// Check types exist
	startingDescriptors := make([]protosource.NamedDescriptor, 0, len(types))
	for _, typeName := range types {
		descriptor, ok := imageIndex.NameToDescriptor[typeName]
		if !ok {
			return nil, fmt.Errorf("not found: %q", typeName)
		}
		if image.GetFile(descriptor.File().Path()).IsImport() {
			return nil, fmt.Errorf("type %q is in an import", typeName)
		}
		startingDescriptors = append(startingDescriptors, descriptor)
	}
	//
	seen := make(map[string]struct{})
	neededDescriptors := []descriptorAndDirects{}
	descriptorsByFile := make(map[string][]descriptorAndDirects)
	for _, startingDescriptor := range startingDescriptors {
		closure, err := descriptorTransitiveClosure(startingDescriptor, imageIndex, seen)
		if err != nil {
			return nil, err
		}
		neededDescriptors = append(neededDescriptors, closure...)
	}
	for _, descriptor := range neededDescriptors {
		descriptorsByFile[descriptor.Descriptor.File().Path()] = append(
			descriptorsByFile[descriptor.Descriptor.File().Path()],
			descriptor,
		)
	}
	// Now create a thinned image.
	var includedFiles []bufimage.ImageFile
	for file, descriptors := range descriptorsByFile {
		importsMap := make(map[string]struct{})
		descriptorNames := make(map[string]struct{})
		for _, descriptor := range descriptors {
			descriptorNames[descriptor.Descriptor.FullName()] = struct{}{}
			for _, importedDescdescriptor := range descriptor.Directs {
				if importedDescdescriptor.File() != descriptor.Descriptor.File() {
					importsMap[importedDescdescriptor.File().Path()] = struct{}{}
				}
			}
		}

		imageFile := image.GetFile(file)
		includedFiles = append(includedFiles, imageFile)
		// the comment on `.Proto` says its for modifying. Probably
		// thats easier to do than recreating the image from scratch,
		// trying that out first.
		imageFileDescriptor := imageFile.Proto()
		// https://github.com/golang/go/wiki/SliceTricks#filter-in-place
		indexFromTo := make(map[int32]int32)
		indexTo := 0
		for indexFrom, importPath := range imageFileDescriptor.GetDependency() {
			// TODO: this only filters the existing imports down to
			// the ones requested, if there was a type we picked up
			// through a public import in a dependent file may
			// filter out that file here, a type not to be found. We
			// may need to add the file directly (or have a file
			// with public import only inserted in the middle).
			// We should check if all keys in importmap get looked up.
			if _, ok := importsMap[importPath]; ok {
				indexFromTo[int32(indexFrom)] = int32(indexTo)
				imageFileDescriptor.Dependency[indexTo] = importPath
				indexTo++
			}
		}
		imageFileDescriptor.Dependency = imageFileDescriptor.Dependency[:indexTo]
		i := 0
		for _, indexFrom := range imageFileDescriptor.PublicDependency {
			if indexTo, ok := indexFromTo[indexFrom]; ok {
				imageFileDescriptor.PublicDependency[i] = indexTo
				i++
			}
		}
		imageFileDescriptor.PublicDependency = imageFileDescriptor.PublicDependency[:i]
		i = 0
		for _, indexFrom := range imageFileDescriptor.WeakDependency {
			if indexTo, ok := indexFromTo[indexFrom]; ok {
				imageFileDescriptor.WeakDependency[i] = indexTo
				i++
			}
		}
		imageFileDescriptor.WeakDependency = imageFileDescriptor.WeakDependency[:i]

		// all the below is a mess, and doesn't work for nested
		// messages/enums as intended, going to make this nice once
		// validated that rewriting the descriptor proto is the way to
		// go.
		prefix := ""
		if imageFileDescriptor.Package != nil {
			prefix = imageFileDescriptor.GetPackage() + "."
		}
		trimMessages, err := trimMessageDescriptor(imageFileDescriptor.MessageType, prefix, descriptorNames)
		if err != nil {
			return nil, err
		}
		imageFileDescriptor.MessageType = trimMessages
		trimEnums, err := trimEnumDescriptor(imageFileDescriptor.EnumType, prefix, descriptorNames)
		if err != nil {
			return nil, err
		}
		imageFileDescriptor.EnumType = trimEnums

		i = 0
		for _, serviceDescriptor := range imageFileDescriptor.Service {
			name := prefix + *serviceDescriptor.Name
			if _, ok := descriptorNames[name]; ok {
				imageFileDescriptor.Service[i] = serviceDescriptor
				i++
			}
		}
		imageFileDescriptor.Service = imageFileDescriptor.Service[:i]

		i = 0
		for _, extensionDescriptor := range imageFileDescriptor.Extension {
			name := prefix + *extensionDescriptor.Name
			if _, ok := descriptorNames[name]; ok {
				imageFileDescriptor.Extension[i] = extensionDescriptor
				i++
			}
		}
		imageFileDescriptor.Extension = imageFileDescriptor.Extension[:i]

		// With some from/to mappings, perhaps even sourcecodeinfo isn't too bad.
		imageFileDescriptor.SourceCodeInfo = nil
	}
	return bufimage.NewImage(includedFiles)
}

func trimMessageDescriptor(in []*descriptorpb.DescriptorProto, prefix string, toKeep map[string]struct{}) ([]*descriptorpb.DescriptorProto, error) {
	i := 0
	for _, messageDescriptor := range in {
		name := prefix + *messageDescriptor.Name
		if _, ok := toKeep[name]; ok {
			trimMessages, err := trimMessageDescriptor(messageDescriptor.NestedType, name+".", toKeep)
			if err != nil {
				return nil, err
			}
			messageDescriptor.NestedType = trimMessages
			trimEnums, err := trimEnumDescriptor(messageDescriptor.EnumType, name+".", toKeep)
			if err != nil {
				return nil, err
			}
			messageDescriptor.EnumType = trimEnums
			in[i] = messageDescriptor
			i++
		}
	}
	return in[:i], nil
}

func trimEnumDescriptor(in []*descriptorpb.EnumDescriptorProto, prefix string, toKeep map[string]struct{}) ([]*descriptorpb.EnumDescriptorProto, error) {
	i := 0
	for _, enumDescriptor := range in {
		name := prefix + *enumDescriptor.Name
		if _, ok := toKeep[name]; ok {
			in[i] = enumDescriptor
			i++
		}
	}
	return in[:i], nil
}

type descriptorAndDirects struct {
	Descriptor protosource.NamedDescriptor
	Directs    []protosource.NamedDescriptor // maybe make this a []string and store imported paths directly?
}

func descriptorTransitiveClosure(namedDescriptor protosource.NamedDescriptor, imageIndex *imageIndex, seen map[string]struct{}) ([]descriptorAndDirects, error) {
	if _, ok := seen[namedDescriptor.FullName()]; ok {
		return nil, nil
	}
	seen[namedDescriptor.FullName()] = struct{}{}

	directDependencies := []protosource.NamedDescriptor{}
	transitiveDependencies := []descriptorAndDirects{}
	switch typedDesctriptor := namedDescriptor.(type) {
	case protosource.Message:
		for _, field := range typedDesctriptor.Fields() {
			switch field.Type() {
			case protosource.FieldDescriptorProtoTypeEnum,
				protosource.FieldDescriptorProtoTypeMessage,
				protosource.FieldDescriptorProtoTypeGroup:
				inputDescriptor, ok := imageIndex.NameToDescriptor[strings.TrimPrefix(field.TypeName(), ".")]
				if !ok {
					return nil, fmt.Errorf("missing %q", field.TypeName())
				}
				directDependencies = append(directDependencies, inputDescriptor)
				recursiveDescriptors, err := descriptorTransitiveClosure(inputDescriptor, imageIndex, seen)
				if err != nil {
					return nil, err
				}
				transitiveDependencies = append(transitiveDependencies, recursiveDescriptors...)
			default:
				// add known types and error here
			}
			// options
			explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(field, imageIndex, seen)
			if err != nil {
				return nil, err
			}
			directDependencies = append(directDependencies, explicitOptionDeps...)
			transitiveDependencies = append(transitiveDependencies, recursedOptionDeps...)
		}

		// Extensions
		for _, extendsDescriptor := range imageIndex.NameToExtensions[namedDescriptor.FullName()] {
			directDependencies = append(directDependencies, extendsDescriptor)
			recursiveDescriptors, err := descriptorTransitiveClosure(extendsDescriptor, imageIndex, seen)
			if err != nil {
				return nil, err
			}
			transitiveDependencies = append(transitiveDependencies, recursiveDescriptors...)
		}

		for _, oneOfDescriptor := range typedDesctriptor.Oneofs() {
			explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(oneOfDescriptor, imageIndex, seen)
			if err != nil {
				return nil, err
			}
			directDependencies = append(directDependencies, explicitOptionDeps...)
			transitiveDependencies = append(transitiveDependencies, recursedOptionDeps...)
		}

		// Parent messages
		if typedDesctriptor.Parent() != nil {
			directDependencies = append(directDependencies, typedDesctriptor.Parent())
			recursiveDescriptors, err := descriptorTransitiveClosure(typedDesctriptor.Parent(), imageIndex, seen)
			if err != nil {
				return nil, err
			}
			transitiveDependencies = append(transitiveDependencies, recursiveDescriptors...)
		}

		// Options
		explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(typedDesctriptor, imageIndex, seen)
		if err != nil {
			return nil, err
		}
		directDependencies = append(directDependencies, explicitOptionDeps...)
		transitiveDependencies = append(transitiveDependencies, recursedOptionDeps...)
	case protosource.Enum:
		// Parent messages
		if typedDesctriptor.Parent() != nil {
			directDependencies = append(directDependencies, typedDesctriptor.Parent())
			recursiveDescriptors, err := descriptorTransitiveClosure(typedDesctriptor.Parent(), imageIndex, seen)
			if err != nil {
				return nil, err
			}
			transitiveDependencies = append(transitiveDependencies, recursiveDescriptors...)
		}

		for _, enumValue := range typedDesctriptor.Values() {
			explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(enumValue, imageIndex, seen)
			if err != nil {
				return nil, err
			}
			directDependencies = append(directDependencies, explicitOptionDeps...)
			transitiveDependencies = append(transitiveDependencies, recursedOptionDeps...)
		}

		// Options
		explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(typedDesctriptor, imageIndex, seen)
		if err != nil {
			return nil, err
		}
		directDependencies = append(directDependencies, explicitOptionDeps...)
		transitiveDependencies = append(transitiveDependencies, recursedOptionDeps...)
	case protosource.EnumValue:
		panic("shouldnt reach") // should be handled in protosource.Enum case
	case protosource.Oneof:
		panic("shouldnt reach") // should be handled in protosource.Message case
	case protosource.Service:
		for _, method := range typedDesctriptor.Methods() {
			inputDescriptor, ok := imageIndex.NameToDescriptor[strings.TrimPrefix(method.InputTypeName(), ".")]
			if !ok {
				return nil, fmt.Errorf("missing %q", method.InputTypeName())
			}
			recursiveDescriptorsIn, err := descriptorTransitiveClosure(inputDescriptor, imageIndex, seen)
			if err != nil {
				return nil, err
			}
			transitiveDependencies = append(transitiveDependencies, recursiveDescriptorsIn...)
			directDependencies = append(directDependencies, inputDescriptor)

			outputDescriptor, ok := imageIndex.NameToDescriptor[strings.TrimPrefix(method.OutputTypeName(), ".")]
			if !ok {
				return nil, fmt.Errorf("missing %q", method.OutputTypeName())
			}
			recursiveDescriptorsOut, err := descriptorTransitiveClosure(outputDescriptor, imageIndex, seen)
			if err != nil {
				return nil, err
			}
			transitiveDependencies = append(transitiveDependencies, recursiveDescriptorsOut...)
			directDependencies = append(directDependencies, outputDescriptor)

			// options
			explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(method, imageIndex, seen)
			if err != nil {
				return nil, err
			}
			directDependencies = append(directDependencies, explicitOptionDeps...)
			transitiveDependencies = append(transitiveDependencies, recursedOptionDeps...)
		}
		// Options
		explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(typedDesctriptor, imageIndex, seen)
		if err != nil {
			return nil, err
		}
		directDependencies = append(directDependencies, explicitOptionDeps...)
		transitiveDependencies = append(transitiveDependencies, recursedOptionDeps...)
	case protosource.Field: // regular fields should be handled by protosource.Message, this is for extends.
		switch typedDesctriptor.Type() {
		case protosource.FieldDescriptorProtoTypeEnum, protosource.FieldDescriptorProtoTypeMessage, protosource.FieldDescriptorProtoTypeGroup:
			inputDescriptor, ok := imageIndex.NameToDescriptor[strings.TrimPrefix(typedDesctriptor.TypeName(), ".")]
			if !ok {
				return nil, fmt.Errorf("missing %q", typedDesctriptor.TypeName())
			}
			directDependencies = append(directDependencies, inputDescriptor)
			recursiveDescriptors, err := descriptorTransitiveClosure(inputDescriptor, imageIndex, seen)
			if err != nil {
				return nil, err
			}
			transitiveDependencies = append(transitiveDependencies, recursiveDescriptors...)
		}
		if typedDesctriptor.Extendee() != "" {
			extendeeDescriptor, ok := imageIndex.NameToDescriptor[strings.TrimPrefix(typedDesctriptor.Extendee(), ".")]
			if !ok {
				return nil, fmt.Errorf("missing %q", typedDesctriptor.Extendee())
			}
			directDependencies = append(directDependencies, extendeeDescriptor)
			recursiveDescriptors, err := descriptorTransitiveClosure(extendeeDescriptor, imageIndex, seen)
			if err != nil {
				return nil, err
			}
			transitiveDependencies = append(transitiveDependencies, recursiveDescriptors...)
		}
		explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(typedDesctriptor, imageIndex, seen)
		if err != nil {
			return nil, err
		}
		directDependencies = append(directDependencies, explicitOptionDeps...)
		transitiveDependencies = append(transitiveDependencies, recursedOptionDeps...)
	default:
		panic(typedDesctriptor)
	}

	// todo: remove others
	// if optionsDescriptor, ok := starting.(protosource.OptionExtensionDescriptor); ok {
	// 	explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(optionsDescriptor, imageIndex, seen)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	explicitDescriptorDependencies = append(explicitDescriptorDependencies, explicitOptionDeps...)
	// 	recursedDescriptorsWithDependencies = append(recursedDescriptorsWithDependencies, recursedOptionDeps...)
	// }

	explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(namedDescriptor.File(), imageIndex, seen)
	if err != nil {
		return nil, err
	}
	directDependencies = append(directDependencies, explicitOptionDeps...)
	transitiveDependencies = append(transitiveDependencies, recursedOptionDeps...)

	return append(
		[]descriptorAndDirects{{Descriptor: namedDescriptor, Directs: directDependencies}},
		transitiveDependencies...,
	), nil
}

func exploreCustomOptions(descriptor protosource.OptionExtensionDescriptor, imageIndex *imageIndex, seen map[string]struct{}) ([]protosource.NamedDescriptor, []descriptorAndDirects, error) {
	explicitDescriptorDependencies := []protosource.NamedDescriptor{}
	recursedDescriptorsWithDependencies := []descriptorAndDirects{}

	var optionName string
	switch descriptor.(type) {
	case protosource.File:
		optionName = "google.protobuf.FileOptions"
	case protosource.Message:
		optionName = "google.protobuf.MessageOptions"
	case protosource.Field:
		optionName = "google.protobuf.FieldOptions"
	case protosource.Oneof:
		optionName = "google.protobuf.OneofOptions"
	case protosource.Enum:
		optionName = "google.protobuf.EnumOptions"
	case protosource.EnumValue:
		optionName = "google.protobuf.EnumValueOptions"
	case protosource.Service:
		optionName = "google.protobuf.ServiceOptions"
	case protosource.Method:
		optionName = "google.protobuf.MethodOptions"
	}

	for _, no := range descriptor.PresentExtensionNumbers() {
		opts := imageIndex.NameToOptions[optionName]
		field, ok := opts[no]
		if !ok {
			return nil, nil, fmt.Errorf("cannot find ext no %d on %s", no, "google.protobuf.FileOptions")
		}

		explicitDescriptorDependencies = append(explicitDescriptorDependencies, field)
		recursiveDescriptors, err := descriptorTransitiveClosure(field, imageIndex, seen)
		if err != nil {
			return nil, nil, err
		}
		recursedDescriptorsWithDependencies = append(recursedDescriptorsWithDependencies, recursiveDescriptors...)
	}
	return explicitDescriptorDependencies, recursedDescriptorsWithDependencies, nil
}

func freeMessageRangeStringsRec(
	s []string,
	message protosource.Message,
) []string {
	for _, nestedMessage := range message.Messages() {
		s = freeMessageRangeStringsRec(s, nestedMessage)
	}
	if e := protosource.FreeMessageRangeString(message); e != "" {
		return append(s, e)
	}
	return s
}

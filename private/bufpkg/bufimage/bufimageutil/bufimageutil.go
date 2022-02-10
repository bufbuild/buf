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

type imageIndex struct {
	NameToDescriptor map[string]protosource.NamedDescriptor
	NameToExtensions map[string][]protosource.Field
	NameToOptions    map[string]map[int32]protosource.Field
}

// ImageFilteredByTypes returns a minimal image containing only the descriptors
// required to interpet types.
func ImageFilteredByTypes(image bufimage.Image, types []string) (bufimage.Image, error) {
	imageIndex := &imageIndex{
		NameToDescriptor: make(map[string]protosource.NamedDescriptor),
		NameToExtensions: make(map[string][]protosource.Field),
		NameToOptions:    make(map[string]map[int32]protosource.Field),
	}
	for _, file := range image.Files() {
		protosourceFile, err := protosource.NewFile(newInputFile(file))
		if err != nil {
			return nil, err
		}
		for _, field := range protosourceFile.Extensions() {
			imageIndex.NameToDescriptor[field.FullName()] = field
			imageIndex.NameToExtensions[strings.TrimPrefix(field.Extendee(), ".")] = append(imageIndex.NameToExtensions[strings.TrimPrefix(field.Extendee(), ".")], field)
		}
		if err := protosource.ForEachMessage(func(message protosource.Message) error {
			if storedDescriptor, ok := imageIndex.NameToDescriptor[message.FullName()]; ok && storedDescriptor != message {
				return fmt.Errorf("duplicate for %q: %#v != %#v", message.FullName(), storedDescriptor, message)
			}
			imageIndex.NameToDescriptor[message.FullName()] = message

			for _, field := range message.Extensions() {
				imageIndex.NameToDescriptor[field.FullName()] = field
				imageIndex.NameToExtensions[field.Extendee()] = append(imageIndex.NameToExtensions[field.Extendee()], field)
			}
			return nil
		}, protosourceFile); err != nil {
			return nil, err
		}
		if err = protosource.ForEachEnum(func(enum protosource.Enum) error {
			if storedDescriptor, ok := imageIndex.NameToDescriptor[enum.FullName()]; ok {
				return fmt.Errorf("duplicate for %q: %#v != %#v", enum.FullName(), storedDescriptor, enum)
			}
			imageIndex.NameToDescriptor[enum.FullName()] = enum
			return nil
		}, protosourceFile); err != nil {
			return nil, err
		}
		for _, service := range protosourceFile.Services() {
			if storedDescriptor, ok := imageIndex.NameToDescriptor[service.FullName()]; ok {
				return nil, fmt.Errorf("duplicate for %q: %#v != %#v", service.FullName(), storedDescriptor, service)
			}
			imageIndex.NameToDescriptor[service.FullName()] = service
		}
	}
	// should probably do this when constructing the imageIndex
	optionNames := map[string]struct{}{
		"google.protobuf.FileOptions":      {},
		"google.protobuf.MessageOptions":   {},
		"google.protobuf.FieldOptions":     {},
		"google.protobuf.OneofOptions":     {},
		"google.protobuf.EnumOptions":      {},
		"google.protobuf.EnumValueOptions": {},
		"google.protobuf.ServiceOptions":   {},
		"google.protobuf.MethodOptions":    {},
	}
	for name, extensions := range imageIndex.NameToExtensions {
		if _, ok := optionNames[name]; !ok {
			continue
		}
		for _, ext := range extensions {
			if _, ok := imageIndex.NameToOptions[name]; !ok {
				imageIndex.NameToOptions[name] = make(map[int32]protosource.Field)
			}
			imageIndex.NameToOptions[name][int32(ext.Number())] = ext
		}
		delete(imageIndex.NameToExtensions, name)
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

	// Find all the types they refer to
	seen := make(map[string]struct{})
	neededDescriptors := []namedDescriptorAndExplicitDeps{}
	for _, startingDescriptor := range startingDescriptors {
		new, err := descriptorTransitiveClosure(startingDescriptor, imageIndex, seen)
		if err != nil {
			return nil, err
		}
		neededDescriptors = append(neededDescriptors, new...)
	}
	// for _, d := range neededDescriptors {
	// 	fmt.Println(d.Descriptor.FullName())
	// 	for _, e := range d.ExplicitDeps {
	// 		fmt.Println("\t" + e.FullName())
	// 	}
	// }

	// Now create a thinned image.
	descriptorsByFile := make(map[string][]namedDescriptorAndExplicitDeps)
	for _, descriptor := range neededDescriptors {
		descriptorsByFile[descriptor.Descriptor.File().Path()] = append(
			descriptorsByFile[descriptor.Descriptor.File().Path()],
			descriptor,
		)
	}
	var includedFiles []bufimage.ImageFile
	for file, descriptors := range descriptorsByFile {
		importsMap := make(map[string]struct{})
		descriptorNames := make(map[string]struct{})
		for _, descriptor := range descriptors {
			descriptorNames[descriptor.Descriptor.FullName()] = struct{}{}
			for _, importedDescdescriptor := range descriptor.ExplicitDeps {
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
			prefix = *imageFileDescriptor.Package + "."
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
		// Won't work for nested
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
		// Won't work for nested
		if _, ok := toKeep[name]; ok {
			in[i] = enumDescriptor
			i++
		}
	}
	return in[:i], nil
}

type namedDescriptorAndExplicitDeps struct {
	Descriptor   protosource.NamedDescriptor
	ExplicitDeps []protosource.NamedDescriptor // maybe make this a []string and store imported paths directly?
}

func descriptorTransitiveClosure(starting protosource.NamedDescriptor, imageIndex *imageIndex, seen map[string]struct{}) ([]namedDescriptorAndExplicitDeps, error) {
	if _, ok := seen[starting.FullName()]; ok {
		return nil, nil
	}
	seen[starting.FullName()] = struct{}{}

	explicitDescriptorDependencies := []protosource.NamedDescriptor{}
	recursedDescriptorsWithDependencies := []namedDescriptorAndExplicitDeps{}
	switch x := starting.(type) {
	case protosource.Message:
		fields := x.Fields()
		for _, field := range fields {
			switch field.Type() {
			case protosource.FieldDescriptorProtoTypeEnum, protosource.FieldDescriptorProtoTypeMessage, protosource.FieldDescriptorProtoTypeGroup:
				inputDescriptor, ok := imageIndex.NameToDescriptor[strings.TrimPrefix(field.TypeName(), ".")]
				if !ok {
					return nil, fmt.Errorf("missing %q", field.TypeName())
				}
				explicitDescriptorDependencies = append(explicitDescriptorDependencies, inputDescriptor)
				recursiveDescriptors, err := descriptorTransitiveClosure(inputDescriptor, imageIndex, seen)
				if err != nil {
					return nil, err
				}
				recursedDescriptorsWithDependencies = append(recursedDescriptorsWithDependencies, recursiveDescriptors...)
			default:
				// add known types and error here
			}
			// options
			explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(field, imageIndex, seen)
			if err != nil {
				return nil, err
			}
			explicitDescriptorDependencies = append(explicitDescriptorDependencies, explicitOptionDeps...)
			recursedDescriptorsWithDependencies = append(recursedDescriptorsWithDependencies, recursedOptionDeps...)
		}

		// Extensions
		for _, extendsDescriptor := range imageIndex.NameToExtensions[starting.FullName()] {
			explicitDescriptorDependencies = append(explicitDescriptorDependencies, extendsDescriptor)
			recursiveDescriptors, err := descriptorTransitiveClosure(extendsDescriptor, imageIndex, seen)
			if err != nil {
				return nil, err
			}
			recursedDescriptorsWithDependencies = append(recursedDescriptorsWithDependencies, recursiveDescriptors...)
		}

		for _, oneOfDescriptor := range x.Oneofs() {
			explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(oneOfDescriptor, imageIndex, seen)
			if err != nil {
				return nil, err
			}
			explicitDescriptorDependencies = append(explicitDescriptorDependencies, explicitOptionDeps...)
			recursedDescriptorsWithDependencies = append(recursedDescriptorsWithDependencies, recursedOptionDeps...)
		}

		// Parent messages
		if x.Parent() != nil {
			explicitDescriptorDependencies = append(explicitDescriptorDependencies, x.Parent())
			recursiveDescriptors, err := descriptorTransitiveClosure(x.Parent(), imageIndex, seen)
			if err != nil {
				return nil, err
			}
			recursedDescriptorsWithDependencies = append(recursedDescriptorsWithDependencies, recursiveDescriptors...)
		}

		// Options
		explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(x, imageIndex, seen)
		if err != nil {
			return nil, err
		}
		explicitDescriptorDependencies = append(explicitDescriptorDependencies, explicitOptionDeps...)
		recursedDescriptorsWithDependencies = append(recursedDescriptorsWithDependencies, recursedOptionDeps...)
	case protosource.Enum:
		// Parent messages
		if x.Parent() != nil {
			explicitDescriptorDependencies = append(explicitDescriptorDependencies, x.Parent())
			recursiveDescriptors, err := descriptorTransitiveClosure(x.Parent(), imageIndex, seen)
			if err != nil {
				return nil, err
			}
			recursedDescriptorsWithDependencies = append(recursedDescriptorsWithDependencies, recursiveDescriptors...)
		}

		for _, enumValue := range x.Values() {
			explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(enumValue, imageIndex, seen)
			if err != nil {
				return nil, err
			}
			explicitDescriptorDependencies = append(explicitDescriptorDependencies, explicitOptionDeps...)
			recursedDescriptorsWithDependencies = append(recursedDescriptorsWithDependencies, recursedOptionDeps...)
		}

		// Options
		explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(x, imageIndex, seen)
		if err != nil {
			return nil, err
		}
		explicitDescriptorDependencies = append(explicitDescriptorDependencies, explicitOptionDeps...)
		recursedDescriptorsWithDependencies = append(recursedDescriptorsWithDependencies, recursedOptionDeps...)
	case protosource.EnumValue:
		panic("shouldnt reach") // should be handled in protosource.Enum case
	case protosource.Oneof:
		panic("shouldnt reach") // should be handled in protosource.Message case
	case protosource.Service:
		for _, method := range x.Methods() {
			inputDescriptor, ok := imageIndex.NameToDescriptor[strings.TrimPrefix(method.InputTypeName(), ".")]
			if !ok {
				return nil, fmt.Errorf("missing %q", method.InputTypeName())
			}
			recursiveDescriptorsIn, err := descriptorTransitiveClosure(inputDescriptor, imageIndex, seen)
			if err != nil {
				return nil, err
			}
			recursedDescriptorsWithDependencies = append(recursedDescriptorsWithDependencies, recursiveDescriptorsIn...)
			explicitDescriptorDependencies = append(explicitDescriptorDependencies, inputDescriptor)

			outputDescriptor, ok := imageIndex.NameToDescriptor[strings.TrimPrefix(method.OutputTypeName(), ".")]
			if !ok {
				return nil, fmt.Errorf("missing %q", method.OutputTypeName())
			}
			recursiveDescriptorsOut, err := descriptorTransitiveClosure(outputDescriptor, imageIndex, seen)
			if err != nil {
				return nil, err
			}
			recursedDescriptorsWithDependencies = append(recursedDescriptorsWithDependencies, recursiveDescriptorsOut...)
			explicitDescriptorDependencies = append(explicitDescriptorDependencies, outputDescriptor)

			// options
			explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(method, imageIndex, seen)
			if err != nil {
				return nil, err
			}
			explicitDescriptorDependencies = append(explicitDescriptorDependencies, explicitOptionDeps...)
			recursedDescriptorsWithDependencies = append(recursedDescriptorsWithDependencies, recursedOptionDeps...)
		}
		// Options
		explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(x, imageIndex, seen)
		if err != nil {
			return nil, err
		}
		explicitDescriptorDependencies = append(explicitDescriptorDependencies, explicitOptionDeps...)
		recursedDescriptorsWithDependencies = append(recursedDescriptorsWithDependencies, recursedOptionDeps...)
	case protosource.Field: // regular fields should be handled by protosource.Message, this is for extends.
		switch x.Type() {
		case protosource.FieldDescriptorProtoTypeEnum, protosource.FieldDescriptorProtoTypeMessage, protosource.FieldDescriptorProtoTypeGroup:
			inputDescriptor, ok := imageIndex.NameToDescriptor[strings.TrimPrefix(x.TypeName(), ".")]
			if !ok {
				return nil, fmt.Errorf("missing %q", x.TypeName())
			}
			explicitDescriptorDependencies = append(explicitDescriptorDependencies, inputDescriptor)
			recursiveDescriptors, err := descriptorTransitiveClosure(inputDescriptor, imageIndex, seen)
			if err != nil {
				return nil, err
			}
			recursedDescriptorsWithDependencies = append(recursedDescriptorsWithDependencies, recursiveDescriptors...)

		}
		if x.Extendee() != "" {
			extendeeDescriptor, ok := imageIndex.NameToDescriptor[strings.TrimPrefix(x.Extendee(), ".")]
			if !ok {
				return nil, fmt.Errorf("missing %q", x.Extendee())
			}
			explicitDescriptorDependencies = append(explicitDescriptorDependencies, extendeeDescriptor)
			recursiveDescriptors, err := descriptorTransitiveClosure(extendeeDescriptor, imageIndex, seen)
			if err != nil {
				return nil, err
			}
			recursedDescriptorsWithDependencies = append(recursedDescriptorsWithDependencies, recursiveDescriptors...)
		}
		explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(x, imageIndex, seen)
		if err != nil {
			return nil, err
		}
		explicitDescriptorDependencies = append(explicitDescriptorDependencies, explicitOptionDeps...)
		recursedDescriptorsWithDependencies = append(recursedDescriptorsWithDependencies, recursedOptionDeps...)
	default:
		panic(x)
	}

	explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(starting.File(), imageIndex, seen)
	if err != nil {
		return nil, err
	}
	explicitDescriptorDependencies = append(explicitDescriptorDependencies, explicitOptionDeps...)
	recursedDescriptorsWithDependencies = append(recursedDescriptorsWithDependencies, recursedOptionDeps...)

	return append(
		[]namedDescriptorAndExplicitDeps{{Descriptor: starting, ExplicitDeps: explicitDescriptorDependencies}},
		recursedDescriptorsWithDependencies...,
	), nil
}

func exploreCustomOptions(descriptor protosource.OptionExtensionDescriptor, imageIndex *imageIndex, seen map[string]struct{}) ([]protosource.NamedDescriptor, []namedDescriptorAndExplicitDeps, error) {
	explicitDescriptorDependencies := []protosource.NamedDescriptor{}
	recursedDescriptorsWithDependencies := []namedDescriptorAndExplicitDeps{}

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

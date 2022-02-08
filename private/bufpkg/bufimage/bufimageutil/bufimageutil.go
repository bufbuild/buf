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
// required to interpet types.
func ImageFilteredByTypes(image bufimage.Image, types []string) (bufimage.Image, error) {
	nameToDescriptor := make(map[string]protosource.NamedDescriptor)
	nameToExtensions := make(map[string][]protosource.OptionExtensionDescriptor)
	_ = nameToExtensions // we need a map of string->[]int
	for _, file := range image.Files() {
		protosourceFile, err := protosource.NewFile(newInputFile(file))
		if err != nil {
			return nil, err
		}
		if err := protosource.ForEachMessage(func(message protosource.Message) error {
			if storedDescriptor, ok := nameToDescriptor[message.FullName()]; ok && storedDescriptor != message {
				return fmt.Errorf("duplicate for %q: %#v != %#v", message.FullName(), storedDescriptor, message)
			}
			nameToDescriptor[message.FullName()] = message
			return nil
		}, protosourceFile); err != nil {
			return nil, err
		}
		if err = protosource.ForEachEnum(func(enum protosource.Enum) error {
			if storedDescriptor, ok := nameToDescriptor[enum.FullName()]; ok {
				return fmt.Errorf("duplicate for %q: %#v != %#v", enum.FullName(), storedDescriptor, enum)
			}
			nameToDescriptor[enum.FullName()] = enum
			return nil
		}, protosourceFile); err != nil {
			return nil, err
		}
		for _, service := range protosourceFile.Services() {
			if storedDescriptor, ok := nameToDescriptor[service.FullName()]; ok {
				return nil, fmt.Errorf("duplicate for %q: %#v != %#v", service.FullName(), storedDescriptor, service)
			}
			nameToDescriptor[service.FullName()] = service
		}
	}

	// Check types exist
	startingDescriptors := make([]protosource.NamedDescriptor, 0, len(types))
	for _, typeName := range types {
		descriptor, ok := nameToDescriptor[typeName]
		if !ok {
			return nil, fmt.Errorf("not found: %q", typeName)
		}
		// imageFile, ok := descriptor.File().(bufimage.ImageFile)
		// if !ok {
		// 	return nil, fmt.Errorf("expected file to be a imagefile (was %T)", descriptor.File())
		// }
		// if imageFile.IsImport() {
		// 	return nil, fmt.Errorf("type %q is in an import", typeName)
		// }
		startingDescriptors = append(startingDescriptors, descriptor)
	}

	// Find all the types they refer to
	seen := make(map[string]struct{})
	neededDescriptors := []namedDescriptorAndExplicitDeps{}
	for _, startingDescriptor := range startingDescriptors {
		new, err := descriptorTransitiveClosure(startingDescriptor, nameToDescriptor, seen)
		if err != nil {
			return nil, err
		}
		neededDescriptors = append(neededDescriptors, new...)
	}

	// Now create a thinned image.
	descriptorsByFile := make(map[string][]namedDescriptorAndExplicitDeps)
	for _, descriptor := range neededDescriptors {
		descriptorsByFile[descriptor.Descriptor.File().Path()] = append(
			descriptorsByFile[descriptor.Descriptor.File().Path()],
			descriptor,
		)
	}
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

		// the comment on `.Proto` says its for modifying. Probably
		// thats easier to do than recreating the image from scratch,
		// trying that out first.
		imageFileDescriptor := image.GetFile(file).Proto()
		// https://github.com/golang/go/wiki/SliceTricks#filter-in-place
		i := 0
		for _, importPath := range imageFileDescriptor.GetDependency() {
			if _, ok := importsMap[importPath]; ok {
				imageFileDescriptor.Dependency[i] = importPath
				i++
			}
		}
		imageFileDescriptor.Dependency = imageFileDescriptor.Dependency[:i]
		// TODO: fixup these
		imageFileDescriptor.PublicDependency = nil
		imageFileDescriptor.WeakDependency = nil

		i = 0
		for _, messageDescriptor := range imageFileDescriptor.MessageType {
			// Won't work for nested
			if _, ok := descriptorNames[*imageFileDescriptor.Package+"."+*messageDescriptor.Name]; ok {
				imageFileDescriptor.MessageType[i] = messageDescriptor
				i++
			}
		}
		imageFileDescriptor.MessageType = imageFileDescriptor.MessageType[:i]

		i = 0
		for _, enumDescriptor := range imageFileDescriptor.EnumType {
			// Won't work for nested
			if _, ok := descriptorNames[*imageFileDescriptor.Package+"."+*enumDescriptor.Name]; ok {
				imageFileDescriptor.EnumType[i] = enumDescriptor
				i++
			}
		}

		i = 0
		for _, serviceDescriptor := range imageFileDescriptor.Service {
			// Won't work for nested
			if _, ok := descriptorNames[*imageFileDescriptor.Package+"."+*serviceDescriptor.Name]; ok {
				imageFileDescriptor.Service[i] = serviceDescriptor
				i++
			}
		}
		imageFileDescriptor.Extension = nil
		// TODO: options
		imageFileDescriptor.Options = nil
		imageFileDescriptor.SourceCodeInfo = nil
	}

	// spew.Dump(nameToDescriptor)
	return image, nil
}

type namedDescriptorAndExplicitDeps struct {
	Descriptor   protosource.NamedDescriptor
	ExplicitDeps []protosource.NamedDescriptor // maybe make this a []string and store imported paths directly?
}

func descriptorTransitiveClosure(starting protosource.NamedDescriptor, all map[string]protosource.NamedDescriptor, seen map[string]struct{}) ([]namedDescriptorAndExplicitDeps, error) {
	if _, ok := seen[starting.FullName()]; ok {
		return nil, nil
	}
	seen[starting.FullName()] = struct{}{}

	explicitDescriptorDependencies := []protosource.NamedDescriptor{}
	recursedDescriptorsWithDependencies := []namedDescriptorAndExplicitDeps{}

	switch x := starting.(type) {
	case protosource.Message:
		for _, field := range x.Fields() {
			switch field.Type() {
			case protosource.FieldDescriptorProtoTypeEnum, protosource.FieldDescriptorProtoTypeMessage, protosource.FieldDescriptorProtoTypeGroup:
				inputDescriptor, ok := all[strings.TrimPrefix(field.TypeName(), ".")]
				if !ok {
					return nil, fmt.Errorf("missing %q", field.TypeName())
				}
				explicitDescriptorDependencies = append(explicitDescriptorDependencies, inputDescriptor)
				recursiveDescriptors, err := descriptorTransitiveClosure(inputDescriptor, all, seen)
				if err != nil {
					return nil, err
				}
				recursedDescriptorsWithDependencies = append(recursedDescriptorsWithDependencies, recursiveDescriptors...)
			default:
				// add known types and error here
			}
		}
		// TODO: if nested message, message in which this was declared.
		// TODO: Extensions
		// TODO: Options
	case protosource.Enum:
		// TODO: Options
	case protosource.Service:
		for _, method := range x.Methods() {
			inputDescriptor, ok := all[strings.TrimPrefix(method.InputTypeName(), ".")]
			if !ok {
				return nil, fmt.Errorf("missing %q", method.InputTypeName())
			}
			recursiveDescriptorsIn, err := descriptorTransitiveClosure(inputDescriptor, all, seen)
			if err != nil {
				return nil, err
			}
			recursedDescriptorsWithDependencies = append(recursedDescriptorsWithDependencies, recursiveDescriptorsIn...)
			explicitDescriptorDependencies = append(explicitDescriptorDependencies, inputDescriptor)

			outputDescriptor, ok := all[strings.TrimPrefix(method.OutputTypeName(), ".")]
			if !ok {
				return nil, fmt.Errorf("missing %q", method.OutputTypeName())
			}
			recursiveDescriptorsOut, err := descriptorTransitiveClosure(outputDescriptor, all, seen)
			if err != nil {
				return nil, err
			}
			recursedDescriptorsWithDependencies = append(recursedDescriptorsWithDependencies, recursiveDescriptorsOut...)
			explicitDescriptorDependencies = append(explicitDescriptorDependencies, outputDescriptor)
		}
	default:
		panic(x)
	}
	return append(
		[]namedDescriptorAndExplicitDeps{{Descriptor: starting, ExplicitDeps: explicitDescriptorDependencies}},
		recursedDescriptorsWithDependencies...,
	), nil
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

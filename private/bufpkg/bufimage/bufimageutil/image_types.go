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
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/protocompile/walk"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type typeLink int

const (
	typeLinkNone    typeLink = iota // 0
	typeLinkChild                   // 1
	typeLinkDepends                 // 2
	typeLinkOption                  // 3
)

type imageTypeIndex struct {
	// TypeSet maps fully qualified type names to their children.
	TypeSet map[protoreflect.FullName]map[protoreflect.FullName]typeLink
	// TypeToFile maps fully qualified type names to their image file.
	TypeToFile map[protoreflect.FullName]bufimage.ImageFile
}

func newImageTypeIndex(image bufimage.Image) (*imageTypeIndex, error) {
	index := &imageTypeIndex{
		TypeSet:    make(map[protoreflect.FullName]map[protoreflect.FullName]typeLink),
		TypeToFile: make(map[protoreflect.FullName]bufimage.ImageFile),
	}
	for _, file := range image.Files() {
		if err := index.addFile(file); err != nil {
			return nil, err
		}
	}
	return index, nil
}

func (i *imageTypeIndex) addFile(file bufimage.ImageFile) error {
	fileDescriptor := file.FileDescriptorProto()
	packageName := protoreflect.FullName(fileDescriptor.GetPackage())
	// Add all parent packages to point to this package.
	for packageName := packageName; packageName != ""; {
		sep := strings.LastIndex(string(packageName), ".")
		if sep == -1 {
			break
		}
		parentPackageName := protoreflect.FullName(packageName[:sep])
		i.linkType(parentPackageName, packageName, typeLinkChild)
		packageName = parentPackageName
	}

	stack := make([]protoreflect.FullName, 0, 10)
	stack = append(stack, packageName)
	i.addType(stack[0])
	i.addOptionTypes(stack[0], fileDescriptor.GetOptions())
	enter := func(fullName protoreflect.FullName, descriptor proto.Message) error {
		fmt.Println("stack", stack, "->", fullName)
		parentFullName := stack[len(stack)-1]
		switch descriptor := descriptor.(type) {
		case *descriptorpb.DescriptorProto:
			i.addOptionTypes(parentFullName, descriptor.GetOptions())
		case *descriptorpb.FieldDescriptorProto:
			i.addFieldType(parentFullName, descriptor)
			i.addOptionTypes(parentFullName, descriptor.GetOptions())
		case *descriptorpb.OneofDescriptorProto:
			i.addOptionTypes(parentFullName, descriptor.GetOptions())
		case *descriptorpb.EnumDescriptorProto:
			i.addOptionTypes(parentFullName, descriptor.GetOptions())
		case *descriptorpb.EnumValueDescriptorProto:
			i.addOptionTypes(parentFullName, descriptor.GetOptions())
		case *descriptorpb.ServiceDescriptorProto:
			i.addOptionTypes(parentFullName, descriptor.GetOptions())
		case *descriptorpb.MethodDescriptorProto:
			inputName := protoreflect.FullName(strings.TrimPrefix(descriptor.GetInputType(), "."))
			outputName := protoreflect.FullName(strings.TrimPrefix(descriptor.GetOutputType(), "."))
			i.linkType(parentFullName, inputName, typeLinkDepends)
			i.linkType(parentFullName, outputName, typeLinkDepends)
			i.addOptionTypes(parentFullName, descriptor.GetOptions())
		case *descriptorpb.DescriptorProto_ExtensionRange:
			i.addOptionTypes(parentFullName, descriptor.GetOptions())
		default:
			return fmt.Errorf("unexpected message type %T", descriptor)
		}
		i.TypeToFile[fullName] = file
		if isDescriptorType(descriptor) {
			i.linkType(parentFullName, fullName, typeLinkChild)
			i.addType(fullName)
			stack = append(stack, fullName)
		}
		return nil
	}
	exit := func(fullName protoreflect.FullName, descriptor proto.Message) error {
		if isDescriptorType(descriptor) {
			stack = stack[:len(stack)-1]
		}
		fmt.Println("exit ", stack, "->", fullName)
		return nil
	}
	if err := walk.DescriptorProtosEnterAndExit(fileDescriptor, enter, exit); err != nil {
		return err
	}
	return nil
}

func (i *imageTypeIndex) addType(fullName protoreflect.FullName) {
	if _, ok := i.TypeSet[fullName]; !ok {
		i.TypeSet[fullName] = nil
	}
}

func (i *imageTypeIndex) linkType(parentFullName protoreflect.FullName, fullName protoreflect.FullName, link typeLink) {
	if typeSet := i.TypeSet[parentFullName]; typeSet != nil {
		typeSet[fullName] = link
	} else {
		i.TypeSet[parentFullName] = map[protoreflect.FullName]typeLink{fullName: link}
	}
}

func (i *imageTypeIndex) addFieldType(parentFullName protoreflect.FullName, fieldDescriptor *descriptorpb.FieldDescriptorProto) {
	if extendee := fieldDescriptor.GetExtendee(); extendee != "" {
		// This is an extension field.
		extendeeFullName := protoreflect.FullName(strings.TrimPrefix(extendee, "."))
		i.linkType(parentFullName, extendeeFullName, typeLinkDepends)
	}
	// Add the field type.
	switch fieldDescriptor.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_ENUM,
		descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
		descriptorpb.FieldDescriptorProto_TYPE_GROUP:
		// Add links to the type of the field.
		typeFullName := protoreflect.FullName(
			strings.TrimPrefix(fieldDescriptor.GetTypeName(), "."),
		)
		i.linkType(parentFullName, typeFullName, typeLinkDepends)
	}
}

func (i *imageTypeIndex) addOptionTypes(parentFullName protoreflect.FullName, optionsMessage proto.Message) {
	if optionsMessage == nil {
		return
	}
	options := optionsMessage.ProtoReflect()
	if !options.IsValid() {
		return
	}
	options.Range(func(fieldDescriptor protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		if fieldDescriptor.IsExtension() {
			i.linkType(parentFullName, fieldDescriptor.FullName(), typeLinkOption)
		}
		return true
	})
}

type fullNameFilter struct {
	include map[protoreflect.FullName]struct{}
	exclude map[protoreflect.FullName]struct{}
}

func (f *fullNameFilter) filter(fullName protoreflect.FullName) (isIncluded bool, isExplicit bool) {
	if f.exclude != nil {
		if _, ok := f.exclude[fullName]; ok {
			return false, true
		}
	}
	if f.include != nil {
		if _, ok := f.include[fullName]; ok {
			return true, true
		}
		return false, false
	}
	return true, false
}

func createTypeFilter(index *imageTypeIndex, options *imageFilterOptions) (fullNameFilter, error) {
	var filter fullNameFilter
	var excludeList []protoreflect.FullName
	for excludeType := range options.excludeTypes {
		excludeType := protoreflect.FullName(excludeType)
		if _, ok := index.TypeSet[excludeType]; !ok {
			return filter, fmt.Errorf("filtering by excluded type %q: %w", excludeType, ErrImageFilterTypeNotFound)
		}
		file := index.TypeToFile[excludeType]
		if file.IsImport() && !options.allowImportedTypes {
			return filter, fmt.Errorf("filtering by excluded type %q: %w", excludeType, ErrImageFilterTypeIsImport)
		}
		excludeList = append(excludeList, excludeType)
	}
	if len(excludeList) > 0 {
		filter.exclude = make(map[protoreflect.FullName]struct{})
	}
	for len(excludeList) > 0 {
		excludeType := excludeList[len(excludeList)-1]
		excludeList = excludeList[:len(excludeList)-1]
		if _, ok := filter.exclude[excludeType]; ok {
			continue
		}
		for childType, childLink := range index.TypeSet[excludeType] {
			if childLink == typeLinkChild {
				excludeList = append(excludeList, childType)
			}
		}
		filter.exclude[excludeType] = struct{}{}
	}

	var includeList []protoreflect.FullName
	for includeType := range options.includeTypes {
		includeType := protoreflect.FullName(includeType)
		if _, ok := index.TypeSet[includeType]; !ok {
			return filter, fmt.Errorf("filtering by included type %q: %w", includeType, ErrImageFilterTypeNotFound)
		}
		file := index.TypeToFile[includeType]
		if file.IsImport() && !options.allowImportedTypes {
			return filter, fmt.Errorf("filtering by included type %q: %w", includeType, ErrImageFilterTypeIsImport)
		}
		if _, ok := filter.exclude[includeType]; ok {
			continue // Skip already excluded.
		}
		includeList = append(includeList, includeType)
	}
	if len(includeList) > 0 {
		filter.include = make(map[protoreflect.FullName]struct{})
	}
	for len(includeList) > 0 {
		includeType := includeList[len(includeList)-1]
		includeList = includeList[:len(includeList)-1]
		if _, ok := filter.include[includeType]; ok {
			continue
		}
		for childType, childLink := range index.TypeSet[includeType] {
			if _, ok := filter.exclude[includeType]; ok {
				continue // Skip already excluded.
			}
			switch childLink {
			case typeLinkChild:
				includeList = append(includeList, childType)
			case typeLinkDepends:
				includeList = append(includeList, childType)
			case typeLinkOption:
				if options.includeKnownExtensions || (options.includeCustomOptions && isOptionsTypeName(string(childType))) {
					includeList = append(includeList, childType)
				}
			}
		}
		filter.include[includeType] = struct{}{}
	}
	return filter, nil
}

func createOptionsFilter(index *imageTypeIndex, options *imageFilterOptions) (fullNameFilter, error) {
	var filter fullNameFilter
	for includeOption := range options.includeOptions {
		includeOption := protoreflect.FullName(includeOption)
		if _, ok := index.TypeSet[includeOption]; !ok {
			return filter, fmt.Errorf("filtering by included option %q: %w", includeOption, ErrImageFilterTypeNotFound)
		}
		// TODO: check for imported type filter?
		if filter.include == nil {
			filter.include = make(map[protoreflect.FullName]struct{})
		}
		filter.include[includeOption] = struct{}{}
	}
	for excludeOption := range options.excludeOptions {
		excludeOption := protoreflect.FullName(excludeOption)
		if _, ok := index.TypeSet[excludeOption]; !ok {
			return filter, fmt.Errorf("filtering by excluded option %q: %w", excludeOption, ErrImageFilterTypeNotFound)
		}
		// TODO: check for imported type filter?
		if filter.exclude == nil {
			filter.exclude = make(map[protoreflect.FullName]struct{})
		}
		filter.exclude[excludeOption] = struct{}{}
	}
	return filter, nil
}

func isDescriptorType(descriptor proto.Message) bool {
	switch descriptor := descriptor.(type) {
	case *descriptorpb.EnumValueDescriptorProto, *descriptorpb.OneofDescriptorProto:
		// Added by their enclosing types.
		return false
	case *descriptorpb.FieldDescriptorProto:
		return descriptor.Extendee != nil
	default:
		return true
	}
}

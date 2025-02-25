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

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type inclusionMode int

const (
	inclusionModeNone inclusionMode = iota
	inclusionModeEnclosing
	inclusionModeExplicit
)

type fullNameFilter struct {
	options  *imageFilterOptions
	index    *imageIndex
	includes map[protoreflect.FullName]inclusionMode
	excludes map[protoreflect.FullName]struct{}
	depth    int
}

func newFullNameFilter(
	imageIndex *imageIndex,
	options *imageFilterOptions,
) (*fullNameFilter, error) {
	filter := &fullNameFilter{
		options: options,
		index:   imageIndex,
	}
	if !options.includeCustomOptions && len(options.includeOptions) > 0 {
		return nil, fmt.Errorf("cannot include options without including custom options")
	}
	for excludeType := range options.excludeTypes {
		excludeType := protoreflect.FullName(excludeType)
		if err := filter.checkFilterType(excludeType); err != nil {
			return nil, err
		}
		if err := filter.exclude(excludeType); err != nil {
			return nil, err
		}
	}
	for includeType := range options.includeTypes {
		includeType := protoreflect.FullName(includeType)
		if err := filter.checkFilterType(includeType); err != nil {
			return nil, err
		}
		if err := filter.include(includeType); err != nil {
			return nil, err
		}
	}
	if err := filter.includeExtensions(); err != nil {
		return nil, err
	}
	return filter, nil
}

func (f *fullNameFilter) inclusionMode(fullName protoreflect.FullName) inclusionMode {
	if f.excludes != nil {
		if _, ok := f.excludes[fullName]; ok {
			return inclusionModeNone
		}
	}
	if f.includes != nil {
		return f.includes[fullName]
	}
	return inclusionModeExplicit
}

func (f *fullNameFilter) hasType(fullName protoreflect.FullName) (isIncluded bool) {
	defer fmt.Println("hasType", fullName, isIncluded)
	return f.inclusionMode(fullName) != inclusionModeNone
}

func (f *fullNameFilter) hasOption(fullName protoreflect.FullName, isExtension bool) (isIncluded bool) {
	defer fmt.Println("hasOption", fullName, isIncluded)
	if f.options.excludeOptions != nil {
		if _, ok := f.options.excludeOptions[string(fullName)]; ok {
			return false
		}
	}
	if !f.options.includeCustomOptions {
		return !isExtension
	}
	if f.options.includeOptions != nil {
		_, ok := f.options.includeOptions[string(fullName)]
		return ok
	}
	return true
}

func (f *fullNameFilter) isExplicitExclude(fullName protoreflect.FullName) bool {
	if f.excludes == nil {
		return false
	}
	_, ok := f.excludes[fullName]
	return ok
}

func (f *fullNameFilter) exclude(fullName protoreflect.FullName) error {
	if _, excluded := f.excludes[fullName]; excluded {
		return nil
	}
	if descriptorInfo, ok := f.index.ByName[fullName]; ok {
		return f.excludeElement(fullName, descriptorInfo.element)
	}
	packageInfo, ok := f.index.Packages[string(fullName)]
	if !ok {
		return fmt.Errorf("type %q: %w", fullName, ErrImageFilterTypeNotFound)
	}
	for _, file := range packageInfo.files {
		// Remove the package name from the excludes since it is not unique per file.
		delete(f.excludes, fullName)
		if err := f.excludeElement(fullName, file.FileDescriptorProto()); err != nil {
			return err
		}
	}
	for _, subPackage := range packageInfo.subPackages {
		if err := f.exclude(subPackage.fullName); err != nil {
			return err
		}
	}
	return nil
}

func (f *fullNameFilter) excludeElement(fullName protoreflect.FullName, descriptor namedDescriptor) error {
	if _, excluded := f.excludes[fullName]; excluded {
		return nil
	}
	if f.excludes == nil {
		f.excludes = make(map[protoreflect.FullName]struct{})
	}
	f.excludes[fullName] = struct{}{}
	switch descriptor := descriptor.(type) {
	case *descriptorpb.FileDescriptorProto:
		if err := forEachDescriptor(fullName, descriptor.GetMessageType(), f.excludeElement); err != nil {
			return err
		}
		if err := forEachDescriptor(fullName, descriptor.GetEnumType(), f.excludeElement); err != nil {
			return err
		}
		if err := forEachDescriptor(fullName, descriptor.GetService(), f.excludeElement); err != nil {
			return err
		}
		return nil
	case *descriptorpb.DescriptorProto:
		// Exclude all sub-elements
		if err := forEachDescriptor(fullName, descriptor.GetNestedType(), f.excludeElement); err != nil {
			return err
		}
		if err := forEachDescriptor(fullName, descriptor.GetEnumType(), f.excludeElement); err != nil {
			return err
		}
		if err := forEachDescriptor(fullName, descriptor.GetOneofDecl(), f.excludeElement); err != nil {
			return err
		}
		return nil
	case *descriptorpb.EnumDescriptorProto:
		// Value is excluded by parent.
		return nil
	case *descriptorpb.OneofDescriptorProto:
		return nil
	case *descriptorpb.ServiceDescriptorProto:
		if err := forEachDescriptor(fullName, descriptor.GetMethod(), f.excludeElement); err != nil {
			return err
		}
		return nil
	case *descriptorpb.MethodDescriptorProto:
		return nil
	default:
		return errorUnsupportedFilterType(descriptor, fullName)
	}
}

func (f *fullNameFilter) include(fullName protoreflect.FullName) error {
	if _, included := f.includes[fullName]; included {
		return nil
	}
	if descriptorInfo, ok := f.index.ByName[fullName]; ok {
		if err := f.includeElement(fullName, descriptorInfo.element); err != nil {
			return err
		}
		// Include the enclosing parent options.
		fileDescriptor := descriptorInfo.imageFile.FileDescriptorProto()
		if err := f.includeOptions(fileDescriptor); err != nil {
			return err
		}
		// loop through all enclosing parents since nesting level
		// could be arbitrarily deep
		for parentName := descriptorInfo.parentName; parentName != ""; {
			if isIncluded := f.hasType(parentName); isIncluded {
				break
			}
			f.includes[parentName] = inclusionModeEnclosing
			parentInfo, ok := f.index.ByName[parentName]
			if !ok {
				break
			}
			if err := f.includeOptions(parentInfo.element); err != nil {
				return err
			}
			parentName = parentInfo.parentName
		}
		return nil
	}
	packageInfo, ok := f.index.Packages[string(fullName)]
	if !ok {
		return fmt.Errorf("type %q: %w", fullName, ErrImageFilterTypeNotFound)
	}
	for _, file := range packageInfo.files {
		// Remove the package name from the includes since it is not unique per file.
		delete(f.includes, fullName)
		if err := f.includeElement(fullName, file.FileDescriptorProto()); err != nil {
			return err
		}
	}
	for _, subPackage := range packageInfo.subPackages {
		if err := f.include(subPackage.fullName); err != nil {
			return err
		}
	}
	return nil
}

func (f *fullNameFilter) includeElement(fullName protoreflect.FullName, descriptor namedDescriptor) error {
	if _, included := f.includes[fullName]; included {
		return nil
	}
	if f.isExplicitExclude(fullName) {
		return nil // already excluded
	}
	if f.includes == nil {
		f.includes = make(map[protoreflect.FullName]inclusionMode)
	}
	f.includes[fullName] = inclusionModeExplicit

	if err := f.includeOptions(descriptor); err != nil {
		return err
	}

	switch descriptor := descriptor.(type) {
	case *descriptorpb.FileDescriptorProto:
		if err := forEachDescriptor(fullName, descriptor.GetMessageType(), f.includeElement); err != nil {
			return err
		}
		if err := forEachDescriptor(fullName, descriptor.GetEnumType(), f.includeElement); err != nil {
			return err
		}
		if err := forEachDescriptor(fullName, descriptor.GetService(), f.includeElement); err != nil {
			return err
		}
		return nil
	case *descriptorpb.DescriptorProto:
		if err := forEachDescriptor(fullName, descriptor.GetNestedType(), f.includeElement); err != nil {
			return err
		}
		if err := forEachDescriptor(fullName, descriptor.GetEnumType(), f.includeElement); err != nil {
			return err
		}
		if err := forEachDescriptor(fullName, descriptor.GetOneofDecl(), f.includeElement); err != nil {
			return err
		}
		if err := forEachDescriptor(fullName, descriptor.GetField(), f.includeElement); err != nil {
			return err
		}
		// Extensions are handled after all elements are included.
		// This allows us to ensure that the extendee is included first.
		return nil
	case *descriptorpb.FieldDescriptorProto:
		if descriptor.Extendee != nil {
			// This is an extension field.
			extendeeName := protoreflect.FullName(strings.TrimPrefix(descriptor.GetExtendee(), "."))
			if err := f.include(extendeeName); err != nil {
				return err
			}
		}
		switch descriptor.GetType() {
		case descriptorpb.FieldDescriptorProto_TYPE_ENUM,
			descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
			descriptorpb.FieldDescriptorProto_TYPE_GROUP:
			typeName := protoreflect.FullName(strings.TrimPrefix(descriptor.GetTypeName(), "."))
			if err := f.include(typeName); err != nil {
				return err
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
			return fmt.Errorf("unknown field type %d", descriptor.GetType())
		}
		return nil
	case *descriptorpb.EnumDescriptorProto:
		for _, enumValue := range descriptor.GetValue() {
			if err := f.includeOptions(enumValue); err != nil {
				return err
			}
		}
		return nil
	case *descriptorpb.OneofDescriptorProto:
		return nil
	case *descriptorpb.ServiceDescriptorProto:
		if err := forEachDescriptor(fullName, descriptor.GetMethod(), f.includeElement); err != nil {
			return err
		}
		return nil
	case *descriptorpb.MethodDescriptorProto:
		inputName := protoreflect.FullName(strings.TrimPrefix(descriptor.GetInputType(), "."))
		inputInfo, ok := f.index.ByName[inputName]
		if !ok {
			return fmt.Errorf("missing %q", inputName)
		}
		if err := f.includeElement(inputName, inputInfo.element); err != nil {
			return err
		}

		outputName := protoreflect.FullName(strings.TrimPrefix(descriptor.GetOutputType(), "."))
		outputInfo, ok := f.index.ByName[outputName]
		if !ok {
			return fmt.Errorf("missing %q", outputName)
		}
		if err := f.includeElement(outputName, outputInfo.element); err != nil {
			return err
		}
		return nil
	default:
		return errorUnsupportedFilterType(descriptor, fullName)
	}
}

func (f *fullNameFilter) includeExtensions() error {
	if !f.options.includeKnownExtensions || len(f.options.includeTypes) == 0 {
		return nil // nothing to do
	}
	for extendeeName, extensions := range f.index.NameToExtensions {
		extendeeName := protoreflect.FullName(extendeeName)
		if f.inclusionMode(extendeeName) != inclusionModeExplicit {
			continue
		}
		for _, extension := range extensions {
			info := f.index.ByDescriptor[extension]
			if f.hasType(info.fullName) {
				continue
			}
			typeName := protoreflect.FullName(strings.TrimPrefix(extension.GetTypeName(), "."))
			if typeName != "" && !f.hasType(typeName) {
				continue
			}
			if err := f.includeElement(info.fullName, extension); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *fullNameFilter) includeOptions(descriptor proto.Message) (err error) {
	if !f.options.includeCustomOptions {
		return nil
	}
	var optionsMessage proto.Message
	switch descriptor := descriptor.(type) {
	case *descriptorpb.FileDescriptorProto:
		optionsMessage = descriptor.GetOptions()
	case *descriptorpb.DescriptorProto:
		optionsMessage = descriptor.GetOptions()
	case *descriptorpb.FieldDescriptorProto:
		optionsMessage = descriptor.GetOptions()
	case *descriptorpb.OneofDescriptorProto:
		optionsMessage = descriptor.GetOptions()
	case *descriptorpb.EnumDescriptorProto:
		optionsMessage = descriptor.GetOptions()
	case *descriptorpb.EnumValueDescriptorProto:
		optionsMessage = descriptor.GetOptions()
	case *descriptorpb.ServiceDescriptorProto:
		optionsMessage = descriptor.GetOptions()
	case *descriptorpb.MethodDescriptorProto:
		optionsMessage = descriptor.GetOptions()
	case *descriptorpb.DescriptorProto_ExtensionRange:
		optionsMessage = descriptor.GetOptions()
	default:
		return fmt.Errorf("unexpected type for exploring options %T", descriptor)
	}
	if optionsMessage == nil {
		return nil
	}
	options := optionsMessage.ProtoReflect()
	optionsName := options.Descriptor().FullName()
	optionsByNumber := f.index.NameToOptions[string(optionsName)]
	options.Range(func(fieldDescriptor protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		if !f.hasOption(fieldDescriptor.FullName(), fieldDescriptor.IsExtension()) {
			return true
		}
		if err = f.includeOptionValue(fieldDescriptor, value); err != nil {
			return false
		}
		if !fieldDescriptor.IsExtension() {
			return true
		}
		extensionField, ok := optionsByNumber[int32(fieldDescriptor.Number())]
		if !ok {
			err = fmt.Errorf("cannot find ext no %d on %s", fieldDescriptor.Number(), optionsName)
			return false
		}
		info := f.index.ByDescriptor[extensionField]
		err = f.includeElement(info.fullName, extensionField)
		return err == nil
	})
	return err
}

func (f *fullNameFilter) includeOptionValue(fieldDescriptor protoreflect.FieldDescriptor, value protoreflect.Value) error {
	// If the value contains an Any message, we should add the message type
	// therein to the closure.
	switch {
	case fieldDescriptor.IsMap():
		if isMessageKind(fieldDescriptor.MapValue().Kind()) {
			var err error
			value.Map().Range(func(_ protoreflect.MapKey, v protoreflect.Value) bool {
				err = f.includeOptionSingularValueForAny(v.Message())
				return err == nil
			})
			return err
		}
		return nil
	case isMessageKind(fieldDescriptor.Kind()):
		if fieldDescriptor.IsList() {
			listVal := value.List()
			for i := 0; i < listVal.Len(); i++ {
				if err := f.includeOptionSingularValueForAny(listVal.Get(i).Message()); err != nil {
					return err
				}
			}
			return nil
		}
		return f.includeOptionSingularValueForAny(value.Message())
	default:
		return nil
	}
}

func (f *fullNameFilter) includeOptionSingularValueForAny(message protoreflect.Message) error {
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
		d, _ := f.index.ByName[msgType].element.(*descriptorpb.DescriptorProto)
		if d != nil {
			if err := f.includeElement(msgType, d); err != nil {
				return err
			}
		}
		// TODO: unmarshal the bytes to see if there are any nested Any messages
		return nil
	}
	// keep digging
	var err error
	message.Range(func(fd protoreflect.FieldDescriptor, val protoreflect.Value) bool {
		err = f.includeOptionValue(fd, val)
		return err == nil
	})
	return err
}

func (f *fullNameFilter) checkFilterType(fullName protoreflect.FullName) error {
	info, ok := f.index.ByName[fullName]
	if !ok {
		return fmt.Errorf("type %q: %w", fullName, ErrImageFilterTypeNotFound)
	}
	if !f.options.allowImportedTypes && info.imageFile.IsImport() {
		return fmt.Errorf("type %q: %w", fullName, ErrImageFilterTypeIsImport)
	}
	return nil
}

func forEachDescriptor[T namedDescriptor](
	parentName protoreflect.FullName,
	list []T,
	fn func(protoreflect.FullName, namedDescriptor) error,
) error {
	for _, element := range list {
		if err := fn(getFullName(parentName, element), element); err != nil {
			return err
		}
	}
	return nil
}

func isMessageKind(k protoreflect.Kind) bool {
	return k == protoreflect.MessageKind || k == protoreflect.GroupKind
}

func errorUnsupportedFilterType(descriptor namedDescriptor, fullName protoreflect.FullName) error {
	var descriptorType string
	switch d := descriptor.(type) {
	case *descriptorpb.FileDescriptorProto:
		descriptorType = "file"
	case *descriptorpb.DescriptorProto:
		descriptorType = "message"
	case *descriptorpb.FieldDescriptorProto:
		if d.Extendee != nil {
			descriptorType = "extension field"
		} else {
			descriptorType = "non-extension field"
		}
	case *descriptorpb.OneofDescriptorProto:
		descriptorType = "oneof"
	case *descriptorpb.EnumDescriptorProto:
		descriptorType = "enum"
	case *descriptorpb.EnumValueDescriptorProto:
		descriptorType = "enum value"
	case *descriptorpb.ServiceDescriptorProto:
		descriptorType = "service"
	case *descriptorpb.MethodDescriptorProto:
		descriptorType = "method"
	default:
		descriptorType = fmt.Sprintf("%T", d)
	}
	return fmt.Errorf("%s is unsupported filter type: %s", fullName, descriptorType)
}

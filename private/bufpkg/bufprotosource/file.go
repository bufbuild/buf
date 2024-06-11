// Copyright 2020-2024 Buf Technologies, Inc.
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

package bufprotosource

import (
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/protodescriptor"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"google.golang.org/protobuf/types/descriptorpb"
)

type file struct {
	FileInfo
	descriptor
	optionExtensionDescriptor

	resolver       protoencoding.Resolver
	fileDescriptor protodescriptor.FileDescriptor
	syntax         Syntax
	fileImports    []FileImport
	messages       []Message
	enums          []Enum
	services       []Service
	extensions     []Field
	edition        descriptorpb.Edition
	optimizeMode   descriptorpb.FileOptions_OptimizeMode
}

func (f *file) FileDescriptor() protodescriptor.FileDescriptor {
	return f.fileDescriptor
}

func (f *file) Syntax() Syntax {
	return f.syntax
}

func (f *file) Package() string {
	return f.fileDescriptor.GetPackage()
}

func (f *file) FileImports() []FileImport {
	return f.fileImports
}

func (f *file) Messages() []Message {
	return f.messages
}

func (f *file) Enums() []Enum {
	return f.enums
}

func (f *file) Services() []Service {
	return f.services
}

func (f *file) Extensions() []Field {
	return f.extensions
}

func (f *file) Edition() descriptorpb.Edition {
	return f.edition
}

func (f *file) CsharpNamespace() string {
	return f.fileDescriptor.GetOptions().GetCsharpNamespace()
}

func (f *file) Deprecated() bool {
	return f.fileDescriptor.GetOptions().GetDeprecated()
}

func (f *file) GoPackage() string {
	return f.fileDescriptor.GetOptions().GetGoPackage()
}

func (f *file) JavaMultipleFiles() bool {
	return f.fileDescriptor.GetOptions().GetJavaMultipleFiles()
}

func (f *file) JavaOuterClassname() string {
	return f.fileDescriptor.GetOptions().GetJavaOuterClassname()
}

func (f *file) JavaPackage() string {
	return f.fileDescriptor.GetOptions().GetJavaPackage()
}

func (f *file) JavaStringCheckUtf8() bool {
	return f.fileDescriptor.GetOptions().GetJavaStringCheckUtf8()
}

func (f *file) ObjcClassPrefix() string {
	return f.fileDescriptor.GetOptions().GetObjcClassPrefix()
}

func (f *file) PhpClassPrefix() string {
	return f.fileDescriptor.GetOptions().GetPhpClassPrefix()
}

func (f *file) PhpNamespace() string {
	return f.fileDescriptor.GetOptions().GetPhpNamespace()
}

func (f *file) PhpMetadataNamespace() string {
	return f.fileDescriptor.GetOptions().GetPhpMetadataNamespace()
}

func (f *file) RubyPackage() string {
	return f.fileDescriptor.GetOptions().GetRubyPackage()
}

func (f *file) SwiftPrefix() string {
	return f.fileDescriptor.GetOptions().GetSwiftPrefix()
}

func (f *file) OptimizeFor() descriptorpb.FileOptions_OptimizeMode {
	return f.optimizeMode
}

func (f *file) CcGenericServices() bool {
	return f.fileDescriptor.GetOptions().GetCcGenericServices()
}

func (f *file) JavaGenericServices() bool {
	return f.fileDescriptor.GetOptions().GetJavaGenericServices()
}

func (f *file) PyGenericServices() bool {
	return f.fileDescriptor.GetOptions().GetPyGenericServices()
}

func (f *file) CcEnableArenas() bool {
	return f.fileDescriptor.GetOptions().GetCcEnableArenas()
}

func (f *file) PackageLocation() Location {
	return f.getLocationByPathKey(packagePathKey)
}

func (f *file) CsharpNamespaceLocation() Location {
	return f.getLocationByPathKey(csharpNamespacePathKey)
}

func (f *file) GoPackageLocation() Location {
	return f.getLocationByPathKey(goPackagePathKey)
}

func (f *file) JavaMultipleFilesLocation() Location {
	return f.getLocationByPathKey(javaMultipleFilesPathKey)
}

func (f *file) JavaOuterClassnameLocation() Location {
	return f.getLocationByPathKey(javaOuterClassnamePathKey)
}

func (f *file) JavaPackageLocation() Location {
	return f.getLocationByPathKey(javaPackagePathKey)
}

func (f *file) JavaStringCheckUtf8Location() Location {
	return f.getLocationByPathKey(javaStringCheckUtf8PathKey)
}

func (f *file) ObjcClassPrefixLocation() Location {
	return f.getLocationByPathKey(objcClassPrefixPathKey)
}

func (f *file) PhpClassPrefixLocation() Location {
	return f.getLocationByPathKey(phpClassPrefixPathKey)
}

func (f *file) PhpNamespaceLocation() Location {
	return f.getLocationByPathKey(phpNamespacePathKey)
}

func (f *file) PhpMetadataNamespaceLocation() Location {
	return f.getLocationByPathKey(phpMetadataNamespacePathKey)
}

func (f *file) RubyPackageLocation() Location {
	return f.getLocationByPathKey(rubyPackagePathKey)
}

func (f *file) SwiftPrefixLocation() Location {
	return f.getLocationByPathKey(swiftPrefixPathKey)
}

func (f *file) OptimizeForLocation() Location {
	return f.getLocationByPathKey(optimizeForPathKey)
}

func (f *file) CcGenericServicesLocation() Location {
	return f.getLocationByPathKey(ccGenericServicesPathKey)
}

func (f *file) JavaGenericServicesLocation() Location {
	return f.getLocationByPathKey(javaGenericServicesPathKey)
}

func (f *file) PyGenericServicesLocation() Location {
	return f.getLocationByPathKey(pyGenericServicesPathKey)
}

func (f *file) CcEnableArenasLocation() Location {
	return f.getLocationByPathKey(ccEnableArenasPathKey)
}

func (f *file) SyntaxLocation() Location {
	return f.getLocationByPathKey(syntaxPathKey)
}

// does not validation of the fileDescriptorProto - this is assumed to be done elsewhere
// does no duplicate checking by name - could just have maps ie importToFileImport, enumNameToEnum, etc
func newFile(imageFile bufimage.ImageFile, resolver protoencoding.Resolver) (*file, error) {
	locationStore := newLocationStore(imageFile.FileDescriptorProto().GetSourceCodeInfo().GetLocation())
	f := &file{
		FileInfo:       imageFile,
		resolver:       resolver,
		fileDescriptor: imageFile.FileDescriptorProto(),
		optionExtensionDescriptor: newOptionExtensionDescriptor(
			imageFile.FileDescriptorProto().GetOptions(),
			[]int32{8},
			locationStore,
			50,
		),
		edition: imageFile.FileDescriptorProto().GetEdition(),
	}
	descriptor := newDescriptor(
		f,
		locationStore,
	)
	f.descriptor = descriptor

	if imageFile.IsSyntaxUnspecified() {
		// if the syntax is "proto2", protoc and buf will not set the syntax
		// field even if it was explicitly set, this is why we have
		// IsSyntaxUnspecified
		f.syntax = SyntaxUnspecified
	} else {
		switch syntaxString := f.fileDescriptor.GetSyntax(); syntaxString {
		case "", "proto2":
			f.syntax = SyntaxProto2
		case "proto3":
			f.syntax = SyntaxProto3
		case "editions":
			f.syntax = SyntaxEditions
		default:
			return nil, fmt.Errorf("unknown syntax: %q", syntaxString)
		}
	}

	for dependencyIndex, dependency := range f.fileDescriptor.GetDependency() {
		fileImport, err := newFileImport(
			f.descriptor,
			dependency,
			getDependencyPath(dependencyIndex),
		)
		if err != nil {
			return nil, err
		}
		f.fileImports = append(f.fileImports, fileImport)
	}
	for _, dependencyIndex := range f.fileDescriptor.GetPublicDependency() {
		if int(dependencyIndex) < 0 || len(f.fileImports) <= int(dependencyIndex) {
			return nil, fmt.Errorf("got dependency index of %d but length of imports is %d", dependencyIndex, len(f.fileImports))
		}
		fileImport, ok := f.fileImports[dependencyIndex].(*fileImport)
		if !ok {
			return nil, fmt.Errorf("could not cast %T to a *fileImport", f.fileImports[dependencyIndex])
		}
		fileImport.setIsPublic()
	}
	for _, dependencyIndex := range f.fileDescriptor.GetWeakDependency() {
		if int(dependencyIndex) < 0 || len(f.fileImports) <= int(dependencyIndex) {
			return nil, fmt.Errorf("got dependency index of %d but length of imports is %d", dependencyIndex, len(f.fileImports))
		}
		fileImport, ok := f.fileImports[dependencyIndex].(*fileImport)
		if !ok {
			return nil, fmt.Errorf("could not cast %T to a *fileImport", f.fileImports[dependencyIndex])
		}
		fileImport.setIsWeak()
	}
	for _, dependencyIndex := range imageFile.UnusedDependencyIndexes() {
		if int(dependencyIndex) < 0 || len(f.fileImports) <= int(dependencyIndex) {
			return nil, fmt.Errorf("got dependency index of %d but length of imports is %d", dependencyIndex, len(f.fileImports))
		}
		fileImport, ok := f.fileImports[dependencyIndex].(*fileImport)
		if !ok {
			return nil, fmt.Errorf("could not cast %T to a *fileImport", f.fileImports[dependencyIndex])
		}
		fileImport.setIsUnused()
	}
	for enumIndex, enumDescriptorProto := range f.fileDescriptor.GetEnumType() {
		enum, err := f.populateEnum(
			enumDescriptorProto,
			enumIndex,
			nil,
			nil,
			nil,
		)
		if err != nil {
			return nil, err
		}
		f.enums = append(f.enums, enum)
	}
	for messageIndex, descriptorProto := range f.fileDescriptor.GetMessageType() {
		message, err := f.populateMessage(
			descriptorProto,
			messageIndex,
			nil,
			nil,
			nil,
		)
		if err != nil {
			return nil, err
		}
		f.messages = append(f.messages, message)
	}
	for serviceIndex, serviceDescriptorProto := range f.fileDescriptor.GetService() {
		service, err := f.populateService(
			serviceDescriptorProto,
			serviceIndex,
		)
		if err != nil {
			return nil, err
		}
		f.services = append(f.services, service)
	}
	for extensionIndex, extensionDescriptorProto := range f.fileDescriptor.GetExtension() {
		extension, err := f.populateExtension(
			extensionDescriptorProto,
			extensionIndex,
		)
		if err != nil {
			return nil, err
		}
		f.extensions = append(f.extensions, extension)
	}
	f.optimizeMode = f.fileDescriptor.GetOptions().GetOptimizeFor()
	return f, nil
}

func (f *file) populateEnum(
	enumDescriptorProto *descriptorpb.EnumDescriptorProto,
	enumIndex int,
	// all message indexes leading to this enum
	nestedMessageIndexes []int,
	// all message names leading to this enum
	nestedMessageNames []string,
	parent Message,
) (Enum, error) {
	enumNamedDescriptor, err := newNamedDescriptor(
		newLocationDescriptor(
			f.descriptor,
			getEnumPath(enumIndex, nestedMessageIndexes...),
		),
		enumDescriptorProto.GetName(),
		getEnumNamePath(enumIndex, nestedMessageIndexes...),
		nestedMessageNames,
	)
	if err != nil {
		return nil, err
	}
	enum := newEnum(
		enumNamedDescriptor,
		newOptionExtensionDescriptor(
			enumDescriptorProto.GetOptions(),
			getEnumOptionsPath(enumIndex, nestedMessageIndexes...),
			f.descriptor.locationStore,
			7,
		),
		enumDescriptorProto.GetOptions().GetAllowAlias(),
		enumDescriptorProto.GetOptions().GetDeprecatedLegacyJsonFieldConflicts(),
		enumDescriptorProto.GetOptions().GetDeprecated(),
		getEnumAllowAliasPath(enumIndex, nestedMessageIndexes...),
		parent,
	)

	for enumValueIndex, enumValueDescriptorProto := range enumDescriptorProto.GetValue() {
		enumValueNamedDescriptor, err := newNamedDescriptor(
			newLocationDescriptor(
				f.descriptor,
				getEnumValuePath(enumIndex, enumValueIndex, nestedMessageIndexes...),
			),
			enumValueDescriptorProto.GetName(),
			getEnumValueNamePath(enumIndex, enumValueIndex, nestedMessageIndexes...),
			slicesext.Concat(nestedMessageNames, []string{enum.Name()}),
		)
		if err != nil {
			return nil, err
		}
		enumValue := newEnumValue(
			enumValueNamedDescriptor,
			newOptionExtensionDescriptor(
				enumValueDescriptorProto.GetOptions(),
				getEnumValueOptionsPath(enumIndex, enumValueIndex, nestedMessageIndexes...),
				f.descriptor.locationStore,
				2,
			),
			enum,
			int(enumValueDescriptorProto.GetNumber()),
			enumValueDescriptorProto.GetOptions().GetDeprecated(),
			getEnumValueNumberPath(enumIndex, enumValueIndex, nestedMessageIndexes...),
		)
		enum.addValue(enumValue)
	}

	for reservedRangeIndex, reservedRangeDescriptorProto := range enumDescriptorProto.GetReservedRange() {
		reservedRangeLocationDescriptor := newLocationDescriptor(
			f.descriptor,
			getEnumReservedRangePath(enumIndex, reservedRangeIndex, nestedMessageIndexes...),
		)
		reservedEnumRange := newEnumRange(
			reservedRangeLocationDescriptor,
			enum,
			int(reservedRangeDescriptorProto.GetStart()),
			int(reservedRangeDescriptorProto.GetEnd()),
		)
		enum.addReservedEnumRange(reservedEnumRange)
	}
	for reservedNameIndex, reservedNameValue := range enumDescriptorProto.GetReservedName() {
		reservedNameLocationDescriptor := newLocationDescriptor(
			f.descriptor,
			getEnumReservedNamePath(enumIndex, reservedNameIndex, nestedMessageIndexes...),
		)
		reservedName, err := newReservedName(
			reservedNameLocationDescriptor,
			reservedNameValue,
		)
		if err != nil {
			return nil, err
		}
		enum.addReservedName(reservedName)
	}
	return enum, nil
}

func (f *file) populateMessage(
	descriptorProto *descriptorpb.DescriptorProto,
	// always stays the same on every recursive call
	topLevelMessageIndex int,
	// includes descriptorProto index
	nestedMessageIndexes []int,
	// does NOT include descriptorProto.GetName()
	nestedMessageNames []string,
	parent Message,
) (Message, error) {
	messageNamedDescriptor, err := newNamedDescriptor(
		newLocationDescriptor(
			f.descriptor,
			getMessagePath(topLevelMessageIndex, nestedMessageIndexes...),
		),
		descriptorProto.GetName(),
		getMessageNamePath(topLevelMessageIndex, nestedMessageIndexes...),
		nestedMessageNames,
	)
	if err != nil {
		return nil, err
	}
	message := newMessage(
		messageNamedDescriptor,
		newOptionExtensionDescriptor(
			descriptorProto.GetOptions(),
			getMessageOptionsPath(topLevelMessageIndex, nestedMessageIndexes...),
			f.descriptor.locationStore,
			12,
		),
		parent,
		descriptorProto.GetOptions().GetMapEntry(),
		descriptorProto.GetOptions().GetMessageSetWireFormat(),
		descriptorProto.GetOptions().GetNoStandardDescriptorAccessor(),
		descriptorProto.GetOptions().GetDeprecatedLegacyJsonFieldConflicts(),
		descriptorProto.GetOptions().GetDeprecated(),
		getMessageMessageSetWireFormatPath(topLevelMessageIndex, nestedMessageIndexes...),
		getMessageNoStandardDescriptorAccessorPath(topLevelMessageIndex, nestedMessageIndexes...),
	)
	oneofIndexToOneof := make(map[int]*oneof)
	for oneofIndex, oneofDescriptorProto := range descriptorProto.GetOneofDecl() {
		oneofNamedDescriptor, err := newNamedDescriptor(
			newLocationDescriptor(
				f.descriptor,
				getMessageOneofPath(oneofIndex, topLevelMessageIndex, nestedMessageIndexes...),
			),
			oneofDescriptorProto.GetName(),
			getMessageOneofNamePath(oneofIndex, topLevelMessageIndex, nestedMessageIndexes...),
			slicesext.Concat(nestedMessageNames, []string{message.Name()}),
		)
		if err != nil {
			return nil, err
		}
		oneof := newOneof(
			oneofNamedDescriptor,
			newOptionExtensionDescriptor(
				oneofDescriptorProto.GetOptions(),
				getMessageOneofOptionsPath(oneofIndex, topLevelMessageIndex, nestedMessageIndexes...),
				f.descriptor.locationStore,
				1,
			),
			message,
		)
		message.addOneof(oneof)
		oneofIndexToOneof[oneofIndex] = oneof
	}
	for fieldIndex, fieldDescriptorProto := range descriptorProto.GetField() {
		// TODO: not working for map entries
		fieldNamedDescriptor, err := newNamedDescriptor(
			newLocationDescriptor(
				f.descriptor,
				getMessageFieldPath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			),
			fieldDescriptorProto.GetName(),
			getMessageFieldNamePath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			slicesext.Concat(nestedMessageNames, []string{message.Name()}),
		)
		if err != nil {
			return nil, err
		}
		var packed *bool
		if fieldDescriptorProto.Options != nil {
			packed = fieldDescriptorProto.GetOptions().Packed
		}
		var oneof *oneof
		var ok bool
		if fieldDescriptorProto.OneofIndex != nil {
			oneofIndex := int(*fieldDescriptorProto.OneofIndex)
			oneof, ok = oneofIndexToOneof[oneofIndex]
			if !ok {
				return nil, fmt.Errorf("no oneof for index %d", oneofIndex)
			}
		}
		field := newField(
			fieldNamedDescriptor,
			newOptionExtensionDescriptor(
				fieldDescriptorProto.GetOptions(),
				getMessageFieldOptionsPath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
				f.descriptor.locationStore,
				21,
			),
			message,
			int(fieldDescriptorProto.GetNumber()),
			fieldDescriptorProto.GetLabel(),
			fieldDescriptorProto.GetType(),
			strings.TrimPrefix(fieldDescriptorProto.GetTypeName(), "."),
			strings.TrimPrefix(fieldDescriptorProto.GetExtendee(), "."),
			oneof,
			fieldDescriptorProto.GetProto3Optional(),
			fieldDescriptorProto.GetJsonName(),
			fieldDescriptorProto.GetOptions().GetJstype(),
			fieldDescriptorProto.GetOptions().GetCtype(),
			fieldDescriptorProto.GetOptions().GetRetention(),
			fieldDescriptorProto.GetOptions().GetTargets(),
			fieldDescriptorProto.GetOptions().GetDebugRedact(),
			packed,
			fieldDescriptorProto.GetDefaultValue(),
			fieldDescriptorProto.GetOptions().GetDeprecated(),
			getMessageFieldNumberPath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			getMessageFieldTypePath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			getMessageFieldTypeNamePath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			getMessageFieldJSONNamePath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			getMessageFieldJSTypePath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			getMessageFieldCTypePath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			getMessageFieldPackedPath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			getMessageFieldDefaultPath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			getMessageFieldExtendeePath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
		)
		message.addField(field)
		if oneof != nil {
			oneof.addField(field)
		}
	}
	for fieldIndex, fieldDescriptorProto := range descriptorProto.GetExtension() {
		fieldNamedDescriptor, err := newNamedDescriptor(
			newLocationDescriptor(
				f.descriptor,
				getMessageExtensionPath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			),
			fieldDescriptorProto.GetName(),
			getMessageExtensionNamePath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			slicesext.Concat(nestedMessageNames, []string{message.Name()}),
		)
		if err != nil {
			return nil, err
		}
		var packed *bool
		if fieldDescriptorProto.Options != nil {
			packed = fieldDescriptorProto.GetOptions().Packed
		}
		var oneof *oneof
		var ok bool
		if fieldDescriptorProto.OneofIndex != nil {
			oneofIndex := int(*fieldDescriptorProto.OneofIndex)
			oneof, ok = oneofIndexToOneof[oneofIndex]
			if !ok {
				return nil, fmt.Errorf("no oneof for index %d", oneofIndex)
			}
		}
		field := newField(
			fieldNamedDescriptor,
			newOptionExtensionDescriptor(
				fieldDescriptorProto.GetOptions(),
				getMessageExtensionOptionsPath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
				f.descriptor.locationStore,
				21,
			),
			message,
			int(fieldDescriptorProto.GetNumber()),
			fieldDescriptorProto.GetLabel(),
			fieldDescriptorProto.GetType(),
			strings.TrimPrefix(fieldDescriptorProto.GetTypeName(), "."),
			strings.TrimPrefix(fieldDescriptorProto.GetExtendee(), "."),
			oneof,
			fieldDescriptorProto.GetProto3Optional(),
			fieldDescriptorProto.GetJsonName(),
			fieldDescriptorProto.GetOptions().GetJstype(),
			fieldDescriptorProto.GetOptions().GetCtype(),
			fieldDescriptorProto.GetOptions().GetRetention(),
			fieldDescriptorProto.GetOptions().GetTargets(),
			fieldDescriptorProto.GetOptions().GetDebugRedact(),
			packed,
			fieldDescriptorProto.GetDefaultValue(),
			fieldDescriptorProto.GetOptions().GetDeprecated(),
			getMessageExtensionNumberPath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			getMessageExtensionTypePath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			getMessageExtensionTypeNamePath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			getMessageExtensionJSONNamePath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			getMessageExtensionJSTypePath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			getMessageExtensionCTypePath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			getMessageExtensionPackedPath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			getMessageExtensionDefaultPath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			getMessageExtensionExtendeePath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
		)
		message.addExtension(field)
		if oneof != nil {
			oneof.addField(field)
		}
	}
	for reservedRangeIndex, reservedRangeDescriptorProto := range descriptorProto.GetReservedRange() {
		reservedRangeLocationDescriptor := newLocationDescriptor(
			f.descriptor,
			getMessageReservedRangePath(reservedRangeIndex, topLevelMessageIndex, nestedMessageIndexes...),
		)
		reservedMessageRange := newMessageRange(
			reservedRangeLocationDescriptor,
			message,
			int(reservedRangeDescriptorProto.GetStart()),
			int(reservedRangeDescriptorProto.GetEnd()),
		)
		message.addReservedMessageRange(reservedMessageRange)
	}
	for reservedNameIndex, reservedNameValue := range descriptorProto.GetReservedName() {
		reservedNameLocationDescriptor := newLocationDescriptor(
			f.descriptor,
			getMessageReservedNamePath(reservedNameIndex, topLevelMessageIndex, nestedMessageIndexes...),
		)
		reservedName, err := newReservedName(
			reservedNameLocationDescriptor,
			reservedNameValue,
		)
		if err != nil {
			return nil, err
		}
		message.addReservedName(reservedName)
	}
	for extensionRangeIndex, extensionRangeDescriptorProto := range descriptorProto.GetExtensionRange() {
		extensionRangeLocationDescriptor := newLocationDescriptor(
			f.descriptor,
			getMessageExtensionRangePath(extensionRangeIndex, topLevelMessageIndex, nestedMessageIndexes...),
		)
		extensionMessageRange := newExtensionRange(
			extensionRangeLocationDescriptor,
			message,
			int(extensionRangeDescriptorProto.GetStart()),
			int(extensionRangeDescriptorProto.GetEnd()),
			newOptionExtensionDescriptor(
				extensionRangeDescriptorProto.GetOptions(),
				getMessageExtensionRangeOptionsPath(extensionRangeIndex, topLevelMessageIndex, nestedMessageIndexes...),
				f.descriptor.locationStore,
				50,
			),
		)
		message.addExtensionRange(extensionMessageRange)
	}
	for enumIndex, enumDescriptorProto := range descriptorProto.GetEnumType() {
		nestedEnum, err := f.populateEnum(
			enumDescriptorProto,
			enumIndex,
			// this is all of the message indexes including this one
			// TODO we should refactor get.*Path messages to be more consistent
			append([]int{topLevelMessageIndex}, nestedMessageIndexes...),
			slicesext.Concat(nestedMessageNames, []string{message.Name()}),
			message,
		)
		if err != nil {
			return nil, err
		}
		message.addNestedEnum(nestedEnum)
	}
	for nestedMessageIndex, nestedMessageDescriptorProto := range descriptorProto.GetNestedType() {
		nestedMessage, err := f.populateMessage(
			nestedMessageDescriptorProto,
			topLevelMessageIndex,
			slicesext.Concat(nestedMessageIndexes, []int{nestedMessageIndex}),
			slicesext.Concat(nestedMessageNames, []string{message.Name()}),
			message,
		)
		if err != nil {
			return nil, err
		}
		message.addNestedMessage(nestedMessage)
	}
	return message, nil
}

func (f *file) populateService(
	serviceDescriptorProto *descriptorpb.ServiceDescriptorProto,
	serviceIndex int,
) (Service, error) {
	serviceNamedDescriptor, err := newNamedDescriptor(
		newLocationDescriptor(
			f.descriptor,
			getServicePath(serviceIndex),
		),
		serviceDescriptorProto.GetName(),
		getServiceNamePath(serviceIndex),
		nil,
	)
	if err != nil {
		return nil, err
	}
	service := newService(
		serviceNamedDescriptor,
		newOptionExtensionDescriptor(
			serviceDescriptorProto.GetOptions(),
			getServiceOptionsPath(serviceIndex),
			f.descriptor.locationStore,
			34,
		),
		serviceDescriptorProto.GetOptions().GetDeprecated(),
	)
	for methodIndex, methodDescriptorProto := range serviceDescriptorProto.GetMethod() {
		methodNamedDescriptor, err := newNamedDescriptor(
			newLocationDescriptor(
				f.descriptor,
				getMethodPath(serviceIndex, methodIndex),
			),
			methodDescriptorProto.GetName(),
			getMethodNamePath(serviceIndex, methodIndex),
			[]string{service.Name()},
		)
		if err != nil {
			return nil, err
		}
		method, err := newMethod(
			methodNamedDescriptor,
			newOptionExtensionDescriptor(
				methodDescriptorProto.GetOptions(),
				getMethodOptionsPath(serviceIndex, methodIndex),
				f.descriptor.locationStore,
				35,
			),
			service,
			strings.TrimPrefix(methodDescriptorProto.GetInputType(), "."),
			strings.TrimPrefix(methodDescriptorProto.GetOutputType(), "."),
			methodDescriptorProto.GetClientStreaming(),
			methodDescriptorProto.GetServerStreaming(),
			methodDescriptorProto.GetOptions().GetDeprecated(),
			getMethodInputTypePath(serviceIndex, methodIndex),
			getMethodOutputTypePath(serviceIndex, methodIndex),
			methodDescriptorProto.GetOptions().GetIdempotencyLevel(),
			getMethodIdempotencyLevelPath(serviceIndex, methodIndex),
		)
		if err != nil {
			return nil, err
		}
		service.addMethod(method)
	}
	return service, nil
}

func (f *file) populateExtension(
	fieldDescriptorProto *descriptorpb.FieldDescriptorProto,
	fieldIndex int,
) (Field, error) {
	fieldNamedDescriptor, err := newNamedDescriptor(
		newLocationDescriptor(
			f.descriptor,
			getFileExtensionPath(fieldIndex),
		),
		fieldDescriptorProto.GetName(),
		getFileExtensionNamePath(fieldIndex),
		nil,
	)
	if err != nil {
		return nil, err
	}
	var packed *bool
	if fieldDescriptorProto.Options != nil {
		packed = fieldDescriptorProto.GetOptions().Packed
	}
	return newField(
		fieldNamedDescriptor,
		newOptionExtensionDescriptor(
			fieldDescriptorProto.GetOptions(),
			getFileExtensionOptionsPath(fieldIndex),
			f.descriptor.locationStore,
			21,
		),
		nil,
		int(fieldDescriptorProto.GetNumber()),
		fieldDescriptorProto.GetLabel(),
		fieldDescriptorProto.GetType(),
		strings.TrimPrefix(fieldDescriptorProto.GetTypeName(), "."),
		strings.TrimPrefix(fieldDescriptorProto.GetExtendee(), "."),
		nil,
		fieldDescriptorProto.GetProto3Optional(),
		fieldDescriptorProto.GetJsonName(),
		fieldDescriptorProto.GetOptions().GetJstype(),
		fieldDescriptorProto.GetOptions().GetCtype(),
		fieldDescriptorProto.GetOptions().GetRetention(),
		fieldDescriptorProto.GetOptions().GetTargets(),
		fieldDescriptorProto.GetOptions().GetDebugRedact(),
		packed,
		fieldDescriptorProto.GetDefaultValue(),
		fieldDescriptorProto.GetOptions().GetDeprecated(),
		getFileExtensionNumberPath(fieldIndex),
		getFileExtensionTypePath(fieldIndex),
		getFileExtensionTypeNamePath(fieldIndex),
		getFileExtensionJSONNamePath(fieldIndex),
		getFileExtensionJSTypePath(fieldIndex),
		getFileExtensionCTypePath(fieldIndex),
		getFileExtensionPackedPath(fieldIndex),
		getFileExtensionDefaultPath(fieldIndex),
		getFileExtensionExtendeePath(fieldIndex),
	), nil
}

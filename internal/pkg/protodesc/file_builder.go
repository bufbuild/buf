package protodesc

import (
	"github.com/bufbuild/buf/internal/pkg/errs"
	"github.com/bufbuild/buf/internal/pkg/protodescpb"
	protobufdescriptor "github.com/golang/protobuf/protoc-gen-go/descriptor"
)

type fileBuilder struct {
	fileDescriptor protodescpb.FileDescriptor

	descriptor  descriptor
	syntax      Syntax
	fileImports []FileImport
	messages    []Message
	enums       []Enum
	services    []Service
}

func newFileBuilder(fileDescriptor protodescpb.FileDescriptor) *fileBuilder {
	return &fileBuilder{
		fileDescriptor: fileDescriptor,
	}
}

// TODO: does no duplicate checking by name, add? could just have maps ie importToFileImport, enumNameToEnum, etc
func (f *fileBuilder) toFile() (*file, error) {
	descriptor, err := newDescriptor(
		f.fileDescriptor.GetName(),
		f.fileDescriptor.GetPackage(),
		newLocationStore(f.fileDescriptor.GetSourceCodeInfo().GetLocation()),
	)
	if err != nil {
		return nil, err
	}
	f.descriptor = descriptor

	syntaxString := f.fileDescriptor.GetSyntax()
	if syntaxString == "" || syntaxString == "proto2" {
		f.syntax = SyntaxProto2
	} else if syntaxString == "proto3" {
		f.syntax = SyntaxProto3
	} else {
		return nil, errs.NewInternalf("unknown syntax: %q", syntaxString)
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
		if len(f.fileImports) <= int(dependencyIndex) {
			return nil, errs.NewInternalf("got dependency index of %d but length of imports is %d", dependencyIndex, len(f.fileImports))
		}
		fileImport, ok := f.fileImports[dependencyIndex].(*fileImport)
		if !ok {
			return nil, errs.NewInternalf("could not cast %T to a *fileImport", f.fileImports[dependencyIndex])
		}
		fileImport.setIsPublic()
	}
	for _, dependencyIndex := range f.fileDescriptor.GetWeakDependency() {
		if len(f.fileImports) <= int(dependencyIndex) {
			return nil, errs.NewInternalf("got dependency index of %d but length of imports is %d", dependencyIndex, len(f.fileImports))
		}
		fileImport, ok := f.fileImports[dependencyIndex].(*fileImport)
		if !ok {
			return nil, errs.NewInternalf("could not cast %T to a *fileImport", f.fileImports[dependencyIndex])
		}
		fileImport.setIsWeak()
	}
	for enumIndex, enumDescriptorProto := range f.fileDescriptor.GetEnumType() {
		enum, err := f.populateEnum(
			enumDescriptorProto,
			enumIndex,
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
	optimizeMode, err := getFileOptionsOptimizeMode(f.fileDescriptor.GetOptions().GetOptimizeFor())
	if err != nil {
		return nil, err
	}
	return &file{
		descriptor:     f.descriptor,
		fileDescriptor: f.fileDescriptor,
		syntax:         f.syntax,
		fileImports:    f.fileImports,
		messages:       f.messages,
		enums:          f.enums,
		services:       f.services,
		optimizeMode:   optimizeMode,
	}, nil
}

func (f *fileBuilder) populateEnum(
	enumDescriptorProto *protobufdescriptor.EnumDescriptorProto,
	enumIndex int,
	// all message indexes leading to this enum
	nestedMessageIndexes []int,
	// all message names leading to this enum
	nestedMessageNames []string,
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
		enumDescriptorProto.GetOptions().GetAllowAlias(),
		getEnumAllowAliasPath(enumIndex, nestedMessageIndexes...),
	)

	for enumValueIndex, enumValueDescriptorProto := range enumDescriptorProto.GetValue() {
		enumValueNamedDescriptor, err := newNamedDescriptor(
			newLocationDescriptor(
				f.descriptor,
				getEnumValuePath(enumIndex, enumValueIndex, nestedMessageIndexes...),
			),
			enumValueDescriptorProto.GetName(),
			getEnumValueNamePath(enumIndex, enumValueIndex, nestedMessageIndexes...),
			append(nestedMessageNames, enum.Name()),
		)
		if err != nil {
			return nil, err
		}
		enumValue := newEnumValue(
			enumValueNamedDescriptor,
			enum,
			int(enumValueDescriptorProto.GetNumber()),
			getEnumValueNumberPath(enumIndex, enumValueIndex, nestedMessageIndexes...),
		)
		enum.addValue(enumValue)
	}

	for reservedRangeIndex, reservedRangeDescriptorProto := range enumDescriptorProto.GetReservedRange() {
		reservedRangeLocationDescriptor := newLocationDescriptor(
			f.descriptor,
			getEnumReservedRangePath(enumIndex, reservedRangeIndex, nestedMessageIndexes...),
		)
		reservedRange := newReservedRange(
			reservedRangeLocationDescriptor,
			int(reservedRangeDescriptorProto.GetStart()),
			int(reservedRangeDescriptorProto.GetEnd()),
			false,
		)
		enum.addReservedRange(reservedRange)
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

func (f *fileBuilder) populateMessage(
	descriptorProto *protobufdescriptor.DescriptorProto,
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
		parent,
		descriptorProto.GetOptions().GetMapEntry(),
		descriptorProto.GetOptions().GetMessageSetWireFormat(),
		descriptorProto.GetOptions().GetNoStandardDescriptorAccessor(),
		getMessageMessageSetWireFormatPath(topLevelMessageIndex, nestedMessageIndexes...),
		getMessageNoStandardDescriptorAccessorPath(topLevelMessageIndex, nestedMessageIndexes...),
	)
	for fieldIndex, fieldDescriptorProto := range descriptorProto.GetField() {
		// TODO: not working for map entries
		fieldNamedDescriptor, err := newNamedDescriptor(
			newLocationDescriptor(
				f.descriptor,
				getMessageFieldPath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			),
			fieldDescriptorProto.GetName(),
			getMessageFieldNamePath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			append(nestedMessageNames, message.Name()),
		)
		if err != nil {
			return nil, err
		}
		var packed *bool
		if fieldDescriptorProto.Options != nil {
			packed = fieldDescriptorProto.GetOptions().Packed
		}
		label, err := getFieldDescriptorProtoLabel(fieldDescriptorProto.GetLabel())
		if err != nil {
			return nil, err
		}
		typ, err := getFieldDescriptorProtoType(fieldDescriptorProto.GetType())
		if err != nil {
			return nil, err
		}
		jsType, err := getFieldOptionsJSType(fieldDescriptorProto.GetOptions().GetJstype())
		if err != nil {
			return nil, err
		}
		cType, err := getFieldOptionsCType(fieldDescriptorProto.GetOptions().GetCtype())
		if err != nil {
			return nil, err
		}
		field := newField(
			fieldNamedDescriptor,
			message,
			int(fieldDescriptorProto.GetNumber()),
			label,
			typ,
			fieldDescriptorProto.GetTypeName(),
			fieldDescriptorProto.OneofIndex,
			fieldDescriptorProto.GetJsonName(),
			jsType,
			cType,
			packed,
			getMessageFieldNumberPath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			getMessageFieldTypePath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			getMessageFieldTypeNamePath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			getMessageFieldJSONNamePath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			getMessageFieldJSTypePath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			getMessageFieldCTypePath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			getMessageFieldPackedPath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
		)
		message.addField(field)
	}
	// TODO: is this right?
	for fieldIndex, fieldDescriptorProto := range descriptorProto.GetExtension() {
		fieldNamedDescriptor, err := newNamedDescriptor(
			newLocationDescriptor(
				f.descriptor,
				getMessageExtensionPath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			),
			fieldDescriptorProto.GetName(),
			getMessageExtensionNamePath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			append(nestedMessageNames, message.Name()),
		)
		if err != nil {
			return nil, err
		}
		var packed *bool
		if fieldDescriptorProto.Options != nil {
			packed = fieldDescriptorProto.GetOptions().Packed
		}
		label, err := getFieldDescriptorProtoLabel(fieldDescriptorProto.GetLabel())
		if err != nil {
			return nil, err
		}
		typ, err := getFieldDescriptorProtoType(fieldDescriptorProto.GetType())
		if err != nil {
			return nil, err
		}
		jsType, err := getFieldOptionsJSType(fieldDescriptorProto.GetOptions().GetJstype())
		if err != nil {
			return nil, err
		}
		cType, err := getFieldOptionsCType(fieldDescriptorProto.GetOptions().GetCtype())
		if err != nil {
			return nil, err
		}
		field := newField(
			fieldNamedDescriptor,
			message,
			int(fieldDescriptorProto.GetNumber()),
			label,
			typ,
			fieldDescriptorProto.GetTypeName(),
			fieldDescriptorProto.OneofIndex,
			fieldDescriptorProto.GetJsonName(),
			jsType,
			cType,
			packed,
			getMessageExtensionNumberPath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			getMessageExtensionTypePath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			getMessageExtensionTypeNamePath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			getMessageExtensionJSONNamePath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			getMessageExtensionJSTypePath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			getMessageExtensionCTypePath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
			getMessageExtensionPackedPath(fieldIndex, topLevelMessageIndex, nestedMessageIndexes...),
		)
		message.addExtension(field)
	}
	for oneofIndex, oneofDescriptorProto := range descriptorProto.GetOneofDecl() {
		oneofNamedDescriptor, err := newNamedDescriptor(
			newLocationDescriptor(
				f.descriptor,
				getMessageOneofPath(oneofIndex, topLevelMessageIndex, nestedMessageIndexes...),
			),
			oneofDescriptorProto.GetName(),
			getMessageOneofNamePath(oneofIndex, topLevelMessageIndex, nestedMessageIndexes...),
			append(nestedMessageNames, message.Name()),
		)
		if err != nil {
			return nil, err
		}
		oneof := newOneof(
			oneofNamedDescriptor,
			message,
		)
		message.addOneof(oneof)
	}
	for reservedRangeIndex, reservedRangeDescriptorProto := range descriptorProto.GetReservedRange() {
		reservedRangeLocationDescriptor := newLocationDescriptor(
			f.descriptor,
			getMessageReservedRangePath(reservedRangeIndex, topLevelMessageIndex, nestedMessageIndexes...),
		)
		reservedRange := newReservedRange(
			reservedRangeLocationDescriptor,
			int(reservedRangeDescriptorProto.GetStart()),
			int(reservedRangeDescriptorProto.GetEnd()),
			true,
		)
		message.addReservedRange(reservedRange)
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
		extensionRange := newExtensionRange(
			extensionRangeLocationDescriptor,
			int(extensionRangeDescriptorProto.GetStart()),
			int(extensionRangeDescriptorProto.GetEnd()),
		)
		message.addExtensionRange(extensionRange)
	}
	for enumIndex, enumDescriptorProto := range descriptorProto.GetEnumType() {
		nestedEnum, err := f.populateEnum(
			enumDescriptorProto,
			enumIndex,
			// this is all of the message indexes including this one
			// TODO we should refactor get.*Path messages to be more consistent
			append([]int{topLevelMessageIndex}, nestedMessageIndexes...),
			append(nestedMessageNames, message.Name()),
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
			append(nestedMessageIndexes, nestedMessageIndex),
			append(nestedMessageNames, message.Name()),
			message,
		)
		if err != nil {
			return nil, err
		}
		message.addNestedMessage(nestedMessage)
	}
	return message, nil
}

func (f *fileBuilder) populateService(
	serviceDescriptorProto *protobufdescriptor.ServiceDescriptorProto,
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
		idempotencyLevel, err := getMethodOptionsIdempotencyLevel(methodDescriptorProto.GetOptions().GetIdempotencyLevel())
		if err != nil {
			return nil, err
		}
		method, err := newMethod(
			methodNamedDescriptor,
			service,
			methodDescriptorProto.GetInputType(),
			methodDescriptorProto.GetOutputType(),
			methodDescriptorProto.GetClientStreaming(),
			methodDescriptorProto.GetServerStreaming(),
			getMethodInputTypePath(serviceIndex, methodIndex),
			getMethodOutputTypePath(serviceIndex, methodIndex),
			idempotencyLevel,
			getMethodIdempotencyLevelPath(serviceIndex, methodIndex),
		)
		if err != nil {
			return nil, err
		}
		service.addMethod(method)
	}
	return service, nil
}

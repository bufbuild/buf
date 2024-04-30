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
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type field struct {
	namedDescriptor
	optionExtensionDescriptor

	parentMessage Message
	number        int
	label         descriptorpb.FieldDescriptorProto_Label
	typ           descriptorpb.FieldDescriptorProto_Type
	typeName      string
	// if the field is an extension, this is the type being extended
	extendee string
	// this has to be the pointer to the private struct or you have the bug where the
	// interface is nil but value == nil is false
	oneof          *oneof
	proto3Optional bool
	jsonName       string
	jsType         descriptorpb.FieldOptions_JSType
	cType          descriptorpb.FieldOptions_CType
	retention      descriptorpb.FieldOptions_OptionRetention
	targets        []descriptorpb.FieldOptions_OptionTargetType
	debugRedact    bool
	packed         *bool
	defaultVal     string
	deprecated     bool
	numberPath     []int32
	typePath       []int32
	typeNamePath   []int32
	jsonNamePath   []int32
	jsTypePath     []int32
	cTypePath      []int32
	packedPath     []int32
	defaultPath    []int32
	extendeePath   []int32
}

func newField(
	namedDescriptor namedDescriptor,
	optionExtensionDescriptor optionExtensionDescriptor,
	parentMessage Message,
	number int,
	label descriptorpb.FieldDescriptorProto_Label,
	typ descriptorpb.FieldDescriptorProto_Type,
	typeName string,
	extendee string,
	oneof *oneof,
	proto3Optional bool,
	jsonName string,
	jsType descriptorpb.FieldOptions_JSType,
	cType descriptorpb.FieldOptions_CType,
	retention descriptorpb.FieldOptions_OptionRetention,
	targets []descriptorpb.FieldOptions_OptionTargetType,
	debugRedact bool,
	packed *bool,
	defaultVal string,
	deprecated bool,
	numberPath []int32,
	typePath []int32,
	typeNamePath []int32,
	jsonNamePath []int32,
	jsTypePath []int32,
	cTypePath []int32,
	packedPath []int32,
	defaultPath []int32,
	extendeePath []int32,
) *field {
	return &field{
		namedDescriptor:           namedDescriptor,
		optionExtensionDescriptor: optionExtensionDescriptor,
		parentMessage:             parentMessage,
		number:                    number,
		label:                     label,
		typ:                       typ,
		typeName:                  typeName,
		extendee:                  extendee,
		oneof:                     oneof,
		proto3Optional:            proto3Optional,
		jsonName:                  jsonName,
		jsType:                    jsType,
		cType:                     cType,
		retention:                 retention,
		targets:                   targets,
		debugRedact:               debugRedact,
		packed:                    packed,
		defaultVal:                defaultVal,
		deprecated:                deprecated,
		numberPath:                numberPath,
		typePath:                  typePath,
		typeNamePath:              typeNamePath,
		jsonNamePath:              jsonNamePath,
		jsTypePath:                jsTypePath,
		cTypePath:                 cTypePath,
		packedPath:                packedPath,
		defaultPath:               defaultPath,
		extendeePath:              extendeePath,
	}
}

func (f *field) ParentMessage() Message {
	return f.parentMessage
}

func (f *field) Number() int {
	return f.number
}

func (f *field) Label() descriptorpb.FieldDescriptorProto_Label {
	return f.label
}

func (f *field) Type() descriptorpb.FieldDescriptorProto_Type {
	return f.typ
}

func (f *field) TypeName() string {
	return f.typeName
}

func (f *field) Extendee() string {
	return f.extendee
}

func (f *field) Oneof() Oneof {
	// this has to be done or you have the bug where the interface is nil
	// but value == nil is false
	if f.oneof == nil {
		return nil
	}
	return f.oneof
}

func (f *field) Proto3Optional() bool {
	return f.proto3Optional
}

func (f *field) JSONName() string {
	return f.jsonName
}

func (f *field) JSType() descriptorpb.FieldOptions_JSType {
	return f.jsType
}

func (f *field) CType() descriptorpb.FieldOptions_CType {
	return f.cType
}

func (f *field) Retention() descriptorpb.FieldOptions_OptionRetention {
	return f.retention
}

func (f *field) Targets() []descriptorpb.FieldOptions_OptionTargetType {
	return f.targets
}

func (f *field) DebugRedact() bool {
	return f.debugRedact
}

func (f *field) Packed() *bool {
	return f.packed
}

func (f *field) Deprecated() bool {
	return f.deprecated
}

func (f *field) Default() string {
	return f.defaultVal
}

func (f *field) NumberLocation() Location {
	return f.getLocation(f.numberPath)
}

func (f *field) TypeLocation() Location {
	loc := f.getLocation(f.typePath)
	if loc == nil {
		return f.maybeMapEntryLocation()
	}
	return loc
}

func (f *field) TypeNameLocation() Location {
	loc := f.getLocation(f.typeNamePath)
	if loc == nil {
		return f.maybeMapEntryLocation()
	}
	return loc
}

func (f *field) JSONNameLocation() Location {
	return f.getLocation(f.jsonNamePath)
}

func (f *field) JSTypeLocation() Location {
	return f.getLocation(f.jsTypePath)
}

func (f *field) CTypeLocation() Location {
	return f.getLocation(f.cTypePath)
}

func (f *field) PackedLocation() Location {
	return f.getLocation(f.packedPath)
}

func (f *field) DefaultLocation() Location {
	return f.getLocation(f.defaultPath)
}

func (f *field) ExtendeeLocation() Location {
	return f.getLocation(f.extendeePath)
}

func (f *field) AsDescriptor() (protoreflect.FieldDescriptor, error) {
	return asDescriptor[protoreflect.FieldDescriptor](&f.descriptor, f.FullName(), "a field")
}

func (f *field) Location() Location {
	loc := f.namedDescriptor.Location()
	if loc == nil {
		return f.maybeMapEntryLocation()
	}
	return loc
}

func (f *field) NameLocation() Location {
	loc := f.namedDescriptor.NameLocation()
	if loc == nil {
		return f.maybeMapEntryLocation()
	}
	return loc
}

func (f *field) maybeMapEntryLocation() Location {
	if message, _ := f.parentMessage.(*message); message != nil {
		return message.maybeMapEntryLocation()
	}
	return nil
}

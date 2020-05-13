// Copyright 2020 Buf Technologies Inc.
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

package protosrc

type field struct {
	namedDescriptor

	message      Message
	number       int
	label        FieldDescriptorProtoLabel
	typ          FieldDescriptorProtoType
	typeName     string
	oneofIndex   *int32
	jsonName     string
	jsType       FieldOptionsJSType
	cType        FieldOptionsCType
	packed       *bool
	numberPath   []int32
	typePath     []int32
	typeNamePath []int32
	jsonNamePath []int32
	jsTypePath   []int32
	cTypePath    []int32
	packedPath   []int32
}

func newField(
	namedDescriptor namedDescriptor,
	message Message,
	number int,
	label FieldDescriptorProtoLabel,
	typ FieldDescriptorProtoType,
	typeName string,
	oneofIndex *int32,
	jsonName string,
	jsType FieldOptionsJSType,
	cType FieldOptionsCType,
	packed *bool,
	numberPath []int32,
	typePath []int32,
	typeNamePath []int32,
	jsonNamePath []int32,
	jsTypePath []int32,
	cTypePath []int32,
	packedPath []int32,
) *field {
	return &field{
		namedDescriptor: namedDescriptor,
		message:         message,
		number:          number,
		label:           label,
		typ:             typ,
		typeName:        typeName,
		oneofIndex:      oneofIndex,
		jsonName:        jsonName,
		jsType:          jsType,
		cType:           cType,
		packed:          packed,
		numberPath:      numberPath,
		typePath:        typePath,
		typeNamePath:    typeNamePath,
		jsonNamePath:    jsonNamePath,
		jsTypePath:      jsTypePath,
		cTypePath:       cTypePath,
		packedPath:      packedPath,
	}
}

func (f *field) Message() Message {
	return f.message
}

func (f *field) Number() int {
	return f.number
}

func (f *field) Label() FieldDescriptorProtoLabel {
	return f.label
}

func (f *field) Type() FieldDescriptorProtoType {
	return f.typ
}

func (f *field) TypeName() string {
	return f.typeName
}

func (f *field) OneofIndex() (int, bool) {
	if f.oneofIndex == nil {
		return 0, false
	}
	return int(*f.oneofIndex), true
}

func (f *field) JSONName() string {
	return f.jsonName
}

func (f *field) JSType() FieldOptionsJSType {
	return f.jsType
}

func (f *field) CType() FieldOptionsCType {
	return f.cType
}

func (f *field) Packed() *bool {
	return f.packed
}

func (f *field) NumberLocation() Location {
	return f.getLocation(f.numberPath)
}

func (f *field) TypeLocation() Location {
	return f.getLocation(f.typePath)
}

func (f *field) TypeNameLocation() Location {
	return f.getLocation(f.typeNamePath)
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

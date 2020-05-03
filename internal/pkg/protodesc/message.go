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

package protodesc

type message struct {
	namedDescriptor

	fields                           []Field
	extensions                       []Field
	nestedMessages                   []Message
	nestedEnums                      []Enum
	oneofs                           []Oneof
	reservedRanges                   []ReservedRange
	reservedNames                    []ReservedName
	extensionRanges                  []ExtensionRange
	parent                           Message
	isMapEntry                       bool
	messageSetWireFormat             bool
	noStandardDescriptorAccessor     bool
	messageSetWireFormatPath         []int32
	noStandardDescriptorAccessorPath []int32
}

func newMessage(
	namedDescriptor namedDescriptor,
	parent Message,
	isMapEntry bool,
	messageSetWireFormat bool,
	noStandardDescriptorAccessor bool,
	messageSetWireFormatPath []int32,
	noStandardDescriptorAccessorPath []int32,
) *message {
	return &message{
		namedDescriptor:                  namedDescriptor,
		isMapEntry:                       isMapEntry,
		messageSetWireFormat:             messageSetWireFormat,
		noStandardDescriptorAccessor:     noStandardDescriptorAccessor,
		messageSetWireFormatPath:         messageSetWireFormatPath,
		noStandardDescriptorAccessorPath: noStandardDescriptorAccessorPath,
	}
}

func (m *message) Fields() []Field {
	return m.fields
}

func (m *message) Extensions() []Field {
	return m.extensions
}

func (m *message) Messages() []Message {
	return m.nestedMessages
}

func (m *message) Enums() []Enum {
	return m.nestedEnums
}

func (m *message) Oneofs() []Oneof {
	return m.oneofs
}

func (m *message) ReservedRanges() []ReservedRange {
	return m.reservedRanges
}

func (m *message) ReservedNames() []ReservedName {
	return m.reservedNames
}

func (m *message) ExtensionRanges() []ExtensionRange {
	return m.extensionRanges
}

func (m *message) Parent() Message {
	return m.parent
}

func (m *message) IsMapEntry() bool {
	return m.isMapEntry
}

func (m *message) MessageSetWireFormat() bool {
	return m.messageSetWireFormat
}

func (m *message) NoStandardDescriptorAccessor() bool {
	return m.noStandardDescriptorAccessor
}

func (m *message) MessageSetWireFormatLocation() Location {
	return m.getLocation(m.messageSetWireFormatPath)
}

func (m *message) NoStandardDescriptorAccessorLocation() Location {
	return m.getLocation(m.noStandardDescriptorAccessorPath)
}

func (m *message) addField(field Field) {
	m.fields = append(m.fields, field)
}

func (m *message) addExtension(extension Field) {
	m.extensions = append(m.extensions, extension)
}

func (m *message) addNestedMessage(nestedMessage Message) {
	m.nestedMessages = append(m.nestedMessages, nestedMessage)
}

func (m *message) addNestedEnum(nestedEnum Enum) {
	m.nestedEnums = append(m.nestedEnums, nestedEnum)
}

func (m *message) addOneof(oneof Oneof) {
	m.oneofs = append(m.oneofs, oneof)
}

func (m *message) addReservedRange(reservedRange ReservedRange) {
	m.reservedRanges = append(m.reservedRanges, reservedRange)
}

func (m *message) addReservedName(reservedName ReservedName) {
	m.reservedNames = append(m.reservedNames, reservedName)
}

func (m *message) addExtensionRange(extensionRange ExtensionRange) {
	m.extensionRanges = append(m.extensionRanges, extensionRange)
}

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

type message struct {
	namedDescriptor
	optionExtensionDescriptor

	fields                             []Field
	extensions                         []Field
	nestedMessages                     []Message
	nestedEnums                        []Enum
	oneofs                             []Oneof
	reservedMessageRanges              []MessageRange
	reservedNames                      []ReservedName
	extensionRanges                    []ExtensionRange
	parent                             Message
	isMapEntry                         bool
	messageSetWireFormat               bool
	noStandardDescriptorAccessor       bool
	deprecatedLegacyJSONFieldConflicts bool
	deprecated                         bool
	messageSetWireFormatPath           []int32
	noStandardDescriptorAccessorPath   []int32
}

func newMessage(
	namedDescriptor namedDescriptor,
	optionExtensionDescriptor optionExtensionDescriptor,
	parent Message,
	isMapEntry bool,
	messageSetWireFormat bool,
	noStandardDescriptorAccessor bool,
	deprecatedLegacyJSONFieldConflicts bool,
	deprecated bool,
	messageSetWireFormatPath []int32,
	noStandardDescriptorAccessorPath []int32,
) *message {
	return &message{
		namedDescriptor:                    namedDescriptor,
		optionExtensionDescriptor:          optionExtensionDescriptor,
		parent:                             parent,
		isMapEntry:                         isMapEntry,
		messageSetWireFormat:               messageSetWireFormat,
		noStandardDescriptorAccessor:       noStandardDescriptorAccessor,
		deprecatedLegacyJSONFieldConflicts: deprecatedLegacyJSONFieldConflicts,
		deprecated:                         deprecated,
		messageSetWireFormatPath:           messageSetWireFormatPath,
		noStandardDescriptorAccessorPath:   noStandardDescriptorAccessorPath,
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

func (m *message) ReservedMessageRanges() []MessageRange {
	return m.reservedMessageRanges
}

func (m *message) ReservedTagRanges() []TagRange {
	tagRanges := make([]TagRange, len(m.reservedMessageRanges))
	for i, reservedMessageRange := range m.reservedMessageRanges {
		tagRanges[i] = reservedMessageRange
	}
	return tagRanges
}

func (m *message) ReservedNames() []ReservedName {
	return m.reservedNames
}

func (m *message) ExtensionRanges() []ExtensionRange {
	return m.extensionRanges
}

func (m *message) ExtensionMessageRanges() []MessageRange {
	extMsgRanges := make([]MessageRange, len(m.extensionRanges))
	for i, extensionRange := range m.extensionRanges {
		extMsgRanges[i] = extensionRange
	}
	return extMsgRanges
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

func (m *message) DeprecatedLegacyJSONFieldConflicts() bool {
	return m.deprecatedLegacyJSONFieldConflicts
}

func (m *message) Deprecated() bool {
	return m.deprecated
}

func (m *message) MessageSetWireFormatLocation() Location {
	return m.getLocation(m.messageSetWireFormatPath)
}

func (m *message) NoStandardDescriptorAccessorLocation() Location {
	return m.getLocation(m.noStandardDescriptorAccessorPath)
}

func (m *message) Location() Location {
	loc := m.namedDescriptor.Location()
	if loc == nil {
		return m.maybeMapEntryLocation()
	}
	return loc
}

func (m *message) NameLocation() Location {
	loc := m.namedDescriptor.NameLocation()
	if loc == nil {
		return m.maybeMapEntryLocation()
	}
	return loc
}

func (m *message) maybeMapEntryLocation() Location {
	parent, _ := m.parent.(*message)
	if !m.isMapEntry || parent == nil || m.namedDescriptor.locationStore.isEmpty() {
		// not a map entry
		return nil
	}
	// Synthetic map messages come from the type of the corresponding
	// map field. So report that location.
	if field := parent.findMapField(m.FullName()); field != nil {
		return field.TypeNameLocation()
	}
	return nil
}

func (m *message) findMapField(entryName string) Field {
	for _, field := range m.fields {
		if field.Type() == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE &&
			field.Label() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED &&
			field.TypeName() == entryName {
			return field
		}
	}
	return nil
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

func (m *message) addReservedMessageRange(reservedMessageRange MessageRange) {
	m.reservedMessageRanges = append(m.reservedMessageRanges, reservedMessageRange)
}

func (m *message) addReservedName(reservedName ReservedName) {
	m.reservedNames = append(m.reservedNames, reservedName)
}

func (m *message) addExtensionRange(extensionRange ExtensionRange) {
	m.extensionRanges = append(m.extensionRanges, extensionRange)
}

func (m *message) AsDescriptor() (protoreflect.MessageDescriptor, error) {
	return asDescriptor[protoreflect.MessageDescriptor](&m.descriptor, m.FullName(), "a message")
}

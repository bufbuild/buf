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

package protoencoding

import (
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// stripLegacyOptions modifies the given slice of file descriptors so that the slice does
// not contain legacy elements that the Go protobuf runtime does not support -- namely
// messages that use the message set wire format (and any extensions whose tags are too
// high to use outside of message set wire format) and weak fields.
//
// It does this by simply clearing any message_set_wire_format message options and weak
// field options encountered, by omitting extensions whose tags are too high, and by
// modifying extension ranges with numbers that are too high (which means omitting
// ranges whose start value is too high).
//
// This does not actually mutate the given descriptors (though it will mutate the given
// slice). If any of the descriptors needs to be modified, it is first cloned and its
// replacement will be stored in the slice. So it is safe to provide descriptor protos
// that are potentially shared with/referenced from other state.
func stripLegacyOptions(files []*descriptorpb.FileDescriptorProto) error {
	for i, file := range files {
		newDescriptor, err := stripLegacyOptionsFromFile(file)
		if err != nil {
			return err
		}
		if newDescriptor != nil {
			files[i] = newDescriptor
		}
	}
	return nil
}

// stripLegacyOptionsFromFile strips legacy options from the given file descriptor.
// If none are encountered this returns nil to indicate no changes were made. Otherwise,
// it returns a clone of file with legacy options removed.
func stripLegacyOptionsFromFile(file *descriptorpb.FileDescriptorProto) (*descriptorpb.FileDescriptorProto, error) {
	var cloned bool
	for i, message := range file.MessageType {
		newDescriptor, err := stripLegacyOptionsFromMessage(message)
		if err != nil {
			return nil, err
		}
		if newDescriptor == nil {
			continue
		}
		if !cloned {
			newFile, err := clone(file)
			if err != nil {
				return nil, err
			}
			file = newFile
			cloned = true
		}
		file.MessageType[i] = newDescriptor
	}
	newExts, err := stripLegacyOptionsFromExtensions(file.Extension)
	if err != nil {
		return nil, err
	}
	if newExts != nil {
		if !cloned {
			newFile, err := clone(file)
			if err != nil {
				return nil, err
			}
			file = newFile
			cloned = true
		}
		file.Extension = newExts
	}
	if cloned {
		return file, nil
	}
	return nil, nil // nothing changed
}

// stripLegacyOptionsFromMessage strips legacy options from the given message descriptor.
// If none are encountered this returns nil to indicate no changes were made. Otherwise,
// it returns a clone of message with legacy options removed.
func stripLegacyOptionsFromMessage(message *descriptorpb.DescriptorProto) (*descriptorpb.DescriptorProto, error) {
	var cloned bool
	if message.GetOptions().GetMessageSetWireFormat() {
		// Strip this option since the Go runtime does not support
		// creating protoreflect.Descriptor instances with this set.
		newMessage, err := clone(message)
		if err != nil {
			return nil, err
		}
		message = newMessage
		cloned = true
		message.Options.MessageSetWireFormat = nil
	}
	for i, field := range message.Field {
		newDescriptor, err := stripLegacyOptionsFromField(field)
		if err != nil {
			return nil, err
		}
		if newDescriptor == nil {
			continue
		}
		if !cloned {
			newMessage, err := clone(message)
			if err != nil {
				return nil, err
			}
			message = newMessage
			cloned = true
		}
		message.Field[i] = newDescriptor
	}

	for i, nested := range message.NestedType {
		newDescriptor, err := stripLegacyOptionsFromMessage(nested)
		if err != nil {
			return nil, err
		}
		if newDescriptor == nil {
			continue
		}
		if !cloned {
			newMessage, err := clone(message)
			if err != nil {
				return nil, err
			}
			message = newMessage
			cloned = true
		}
		message.NestedType[i] = newDescriptor
	}
	newExtRanges, err := stripLegacyOptionsFromExtensionRanges(message.ExtensionRange)
	if err != nil {
		return nil, err
	}
	if newExtRanges != nil {
		if !cloned {
			newMessage, err := clone(message)
			if err != nil {
				return nil, err
			}
			message = newMessage
			cloned = true
		}
		message.ExtensionRange = newExtRanges
	}
	newExts, err := stripLegacyOptionsFromExtensions(message.Extension)
	if err != nil {
		return nil, err
	}
	if newExts != nil {
		if !cloned {
			newMessage, err := clone(message)
			if err != nil {
				return nil, err
			}
			message = newMessage
			cloned = true
		}
		message.Extension = newExts
	}
	if cloned {
		return message, nil
	}
	return nil, nil // nothing changed
}

// stripLegacyOptionsFromField strips legacy options from the given field descriptor.
// If none are encountered this returns nil to indicate no changes were made. Otherwise,
// it returns a clone of field with legacy options removed.
func stripLegacyOptionsFromField(field *descriptorpb.FieldDescriptorProto) (*descriptorpb.FieldDescriptorProto, error) {
	if !field.GetOptions().GetWeak() {
		return nil, nil
	}
	// Strip this option since the Go runtime does not support
	// creating protoreflect.Descriptor instances with this set.
	// Buf CLI doesn't actually support weak dependencies, so
	// there should be no practical consequences of removing this.
	newField, err := clone(field)
	if err != nil {
		return nil, err
	}
	newField.Options.Weak = nil
	return newField, nil
}

// stripLegacyOptionsFromExtensions strips legacy options and values from the
// given slice of extension descriptor. If none are encountered this returns
// nil to indicate no changes were made. Otherwise, it returns a new slice
// that omits invalid legacy extensions (those whose tag number is too high
// because they extended a message with message set wire format) and replaces
// items that had legacy options with clones that have the options removed.
func stripLegacyOptionsFromExtensions(exts []*descriptorpb.FieldDescriptorProto) ([]*descriptorpb.FieldDescriptorProto, error) {
	// We leave this nil unless we are actually changing the extensions
	// (by removing one with a tag that is too high are or by stripping
	// the weak option from one).
	var newExts []*descriptorpb.FieldDescriptorProto
	for i, ext := range exts {
		// Message-set extensions could be out of range. We simply remove them.
		// This could possibly
		if ext.GetNumber() > maxTagNumber {
			if newExts == nil {
				// initialize to everything so far except current item (which we're dropping)
				newExts = make([]*descriptorpb.FieldDescriptorProto, i, len(exts)-1)
				copy(newExts, exts)
			}
			continue
		}
		newDescriptor, err := stripLegacyOptionsFromField(ext)
		if err != nil {
			return nil, err
		}
		if newDescriptor != nil {
			if newExts == nil {
				// initialize to everything so far except current item (that we're replacing)
				newExts = make([]*descriptorpb.FieldDescriptorProto, i, len(exts))
				copy(newExts, exts)
			}
			newExts = append(newExts, newDescriptor)
			continue
		}
		if newExts != nil {
			newExts = append(newExts, ext)
		}
	}
	return newExts, nil
}

// stripLegacyOptionsFromExtensionRanges strips legacy values from the given
// slice of extension descriptor. If none are encountered this returns nil
// to indicate no changes were made. Otherwise, it returns a new slice that
// is updated to exclude invalid legacy extension ranges (those referencing
// tag numbers that are too high because they extended a message with message
// set wire format). If an extension range has a start tag that is too high,
// it is omitted entirely. If it has an end tag that is too high, the end tag
// is changed to the maximum valid end tag.
func stripLegacyOptionsFromExtensionRanges(extRanges []*descriptorpb.DescriptorProto_ExtensionRange) ([]*descriptorpb.DescriptorProto_ExtensionRange, error) {
	// We leave this nil unless we are actually changing the extensions
	// (by removing one with a tag that is too high are or by stripping
	// the weak option from one).
	var newExtRanges []*descriptorpb.DescriptorProto_ExtensionRange
	for i, extRange := range extRanges {
		// Message-set extensions could be out of range. We simply remove them.
		// This could possibly
		if extRange.GetStart() > maxTagNumber {
			if newExtRanges == nil {
				// initialize to everything so far except current item (which we're dropping)
				newExtRanges = make([]*descriptorpb.DescriptorProto_ExtensionRange, i, len(extRanges)-1)
				copy(newExtRanges, extRanges)
			}
			continue
		}
		if extRange.GetEnd() > maxTagNumber+1 /* extension range end is exclusive */ {
			newExtRange, err := clone(extRange)
			if err != nil {
				return nil, err
			}
			newExtRange.End = proto.Int32(maxTagNumber + 1)
			if newExtRanges == nil {
				// initialize to everything so far except current item (that we're replacing)
				newExtRanges = make([]*descriptorpb.DescriptorProto_ExtensionRange, i, len(extRanges))
				copy(newExtRanges, extRanges)
			}
			newExtRanges = append(newExtRanges, newExtRange)
			continue
		}
		if newExtRanges != nil {
			newExtRanges = append(newExtRanges, extRange)
		}
	}
	return newExtRanges, nil
}

func clone[M proto.Message](message M) (M, error) {
	clone := proto.Clone(message)
	newMessage, isFileProto := clone.(M)
	if !isFileProto {
		var zero M
		return zero, fmt.Errorf("proto.Clone returned unexpected value: %T instead of %T", clone, message)
	}
	return newMessage, nil
}

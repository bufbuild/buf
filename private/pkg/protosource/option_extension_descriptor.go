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

package protosource

import (
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type optionExtensionDescriptor struct {
	message proto.Message
}

func newOptionExtensionDescriptor(message proto.Message) optionExtensionDescriptor {
	return optionExtensionDescriptor{
		message: message,
	}
}

func (o *optionExtensionDescriptor) OptionExtension(extensionType protoreflect.ExtensionType) (interface{}, bool) {
	if extensionType.TypeDescriptor().ContainingMessage().FullName() != o.message.ProtoReflect().Descriptor().FullName() {
		return nil, false
	}
	if !proto.HasExtension(o.message, extensionType) {
		return nil, false
	}
	return proto.GetExtension(o.message, extensionType), true
}

func (o *optionExtensionDescriptor) RangeExtensions(callback func(protoreflect.ExtensionType, interface{}) bool) {
	proto.RangeExtensions(o.message, callback)
}

func (o *optionExtensionDescriptor) PresentExtensionNumbers() []int32 {
	fieldNumbersSet := map[int32]struct{}{}
	var fieldNumbers []int32
	addFieldNumber := func(fieldNo int32) {
		if _, ok := fieldNumbersSet[fieldNo]; !ok {
			fieldNumbersSet[fieldNo] = struct{}{}
			fieldNumbers = append(fieldNumbers, fieldNo)
		}
	}
	msg := o.message.ProtoReflect()
	extensionRanges := msg.Descriptor().ExtensionRanges()
	for b := msg.GetUnknown(); len(b) > 0; {
		fieldNo, _, n := protowire.ConsumeField(b)
		if extensionRanges.Has(fieldNo) {
			addFieldNumber(int32(fieldNo))
		}
		b = b[n:]
	}
	// Extensions for google.protobuf.*Options are a bit of a special case
	// as the extensions in a FileDescriptorSet message may differ with
	// the extensions defined in the proto with which buf is compiled.
	//
	// Also loop through known extensions here to get extension numbers.
	msg.Range(func(fieldDescriptor protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		if fieldDescriptor.IsExtension() {
			addFieldNumber(int32(fieldDescriptor.Number()))
		}
		return true
	})

	return fieldNumbers
}

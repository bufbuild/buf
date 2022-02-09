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
	if !proto.HasExtension(o.message, extensionType) {
		return nil, false
	}
	return proto.GetExtension(o.message, extensionType), true
}

func (o *optionExtensionDescriptor) ExtendedFieldNo() []int32 {
	var extended []int32
	for b := o.message.ProtoReflect().GetUnknown(); len(b) > 0; {
		fieldNo, _, n := protowire.ConsumeField(b)
		// We should filter these to only take the ones in the message's
		// declared extension ranges , but I haven't figured out how to
		// get those yet.
		extended = append(extended, int32(fieldNo))
		b = b[n:]
	}
	return extended
}

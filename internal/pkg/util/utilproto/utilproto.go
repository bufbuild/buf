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

package utilproto

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

var (
	marshalOptions              = proto.MarshalOptions{}
	marshalOptionsDeterministic = proto.MarshalOptions{
		Deterministic: true,
	}
	unmarshalOptions           = proto.UnmarshalOptions{}
	jsonMarshalOptions         = protojson.MarshalOptions{}
	jsonMarshalOptionsOrigName = protojson.MarshalOptions{
		UseProtoNames: true,
	}
	jsonMarshalOptionsIndent = protojson.MarshalOptions{
		Indent: "  ",
	}
	jsonUnmarshalOptions = protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	textMarshalOptions = prototext.MarshalOptions{}
)

// MarshalWire marshals the message to wire format.
func MarshalWire(message proto.Message) ([]byte, error) {
	return marshalOptions.Marshal(message)
}

// MarshalWireDeterministic marshals the message to wire format deterministically.
func MarshalWireDeterministic(message proto.Message) ([]byte, error) {
	return marshalOptionsDeterministic.Marshal(message)
}

// MarshalJSON marshals the message to JSON format.
func MarshalJSON(message proto.Message) ([]byte, error) {
	return jsonMarshalOptions.Marshal(message)
}

// MarshalJSONOrigName marshals the message to JSON format with original .proto names as keys.
func MarshalJSONOrigName(message proto.Message) ([]byte, error) {
	return jsonMarshalOptionsOrigName.Marshal(message)
}

// MarshalJSONIndent marshals the message to JSON format with indents.
func MarshalJSONIndent(message proto.Message) ([]byte, error) {
	return jsonMarshalOptionsIndent.Marshal(message)
}

// MarshalText marshals the message to text format.
func MarshalText(message proto.Message) ([]byte, error) {
	return textMarshalOptions.Marshal(message)
}

// UnmarshalWire unmarshals the message from wire format.
func UnmarshalWire(data []byte, message proto.Message) error {
	return unmarshalOptions.Unmarshal(data, message)
}

// UnmarshalJSON unmarshals the message from JSON format.
func UnmarshalJSON(data []byte, message proto.Message) error {
	return jsonUnmarshalOptions.Unmarshal(data, message)
}

// Equal checks if the two messages are equal.
func Equal(one proto.Message, two proto.Message) bool {
	return proto.Equal(one, two)
}

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
	"bytes"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

var (
	jsonMarshaler         = &jsonpb.Marshaler{}
	jsonMarshalerOrigName = &jsonpb.Marshaler{
		OrigName: true,
	}
	jsonMarshalerIndent = &jsonpb.Marshaler{
		Indent: "  ",
	}
	jsonUnmarshaler = &jsonpb.Unmarshaler{
		AllowUnknownFields: true,
	}
)

// MarshalWire marshals the message to wire format.
func MarshalWire(message proto.Message) ([]byte, error) {
	return proto.Marshal(message)
}

// MarshalWireDeterministic marshals the message to wire format deterministically.
func MarshalWireDeterministic(message proto.Message) ([]byte, error) {
	buffer := proto.NewBuffer(nil)
	buffer.SetDeterministic(true)
	if err := buffer.Marshal(message); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// MarshalJSON marshals the message to JSON format.
func MarshalJSON(message proto.Message) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	if err := jsonMarshaler.Marshal(buffer, message); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// MarshalJSONOrigName marshals the message to JSON format with original .proto names as keys.
func MarshalJSONOrigName(message proto.Message) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	if err := jsonMarshalerOrigName.Marshal(buffer, message); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// MarshalJSONIndent marshals the message to JSON format with indents.
func MarshalJSONIndent(message proto.Message) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	if err := jsonMarshalerIndent.Marshal(buffer, message); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// MarshalText marshals the message to text format.
func MarshalText(message proto.Message) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	if err := proto.MarshalText(buffer, message); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// UnmarshalWire unmarshals the message from wire format.
func UnmarshalWire(data []byte, message proto.Message) error {
	return proto.Unmarshal(data, message)
}

// UnmarshalJSON unmarshals the message from JSON format.
func UnmarshalJSON(data []byte, message proto.Message) error {
	return jsonUnmarshaler.Unmarshal(bytes.NewReader(data), message)
}

// Equal checks if the two messages are equal.
func Equal(one proto.Message, two proto.Message) bool {
	return proto.Equal(one, two)
}

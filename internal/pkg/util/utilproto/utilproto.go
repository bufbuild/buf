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
	"encoding/json"

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
	return marshalJSON(jsonMarshalOptions, message)
}

// MarshalJSONOrigName marshals the message to JSON format with original .proto names as keys.
func MarshalJSONOrigName(message proto.Message) ([]byte, error) {
	return marshalJSON(jsonMarshalOptionsOrigName, message)
}

// MarshalJSONIndent marshals the message to JSON format with indents.
func MarshalJSONIndent(message proto.Message) ([]byte, error) {
	return marshalJSON(jsonMarshalOptionsIndent, message)
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

// marshalJSON marshals the message as JSON with the given MarshalOptions.
//
// This is needed due to the instability of protojson output.
//
// https://github.com/golang/protobuf/issues/1121
// https://go-review.googlesource.com/c/protobuf/+/151340
// https://developers.google.com/protocol-buffers/docs/reference/go/faq#unstable-json
//
// We may need to do a full encoding/json encode/decode in the future if protojson
// produces non-deterministic output.
//
// Naming "options" to make sure there is no overlap with the global variables
func marshalJSON(options protojson.MarshalOptions, message proto.Message) ([]byte, error) {
	data, err := options.Marshal(message)
	if err != nil {
		return nil, err
	}
	buffer := bytes.NewBuffer(nil)
	if err := json.Compact(buffer, data); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

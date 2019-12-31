package utilproto

import (
	"bytes"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

var (
	jsonMarshaler       = &jsonpb.Marshaler{}
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

// MarshalJSON marshals the message to JSON format.
func MarshalJSON(message proto.Message) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	if err := jsonMarshaler.Marshal(buffer, message); err != nil {
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

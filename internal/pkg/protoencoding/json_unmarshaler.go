package protoencoding

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type jsonUnmarshaler struct {
	resolver Resolver
}

func newJSONUnmarshaler(resolver Resolver) Unmarshaler {
	return &jsonUnmarshaler{
		resolver: resolver,
	}
}

func (m *jsonUnmarshaler) Unmarshal(data []byte, message proto.Message) error {
	options := protojson.UnmarshalOptions{
		Resolver: m.resolver,
	}
	return options.Unmarshal(data, message)
}

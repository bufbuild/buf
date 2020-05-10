package protoencoding

import (
	"google.golang.org/protobuf/proto"
)

type wireUnmarshaler struct {
	resolver Resolver
}

func newWireUnmarshaler(resolver Resolver) Unmarshaler {
	return &wireUnmarshaler{
		resolver: resolver,
	}
}

func (m *wireUnmarshaler) Unmarshal(data []byte, message proto.Message) error {
	options := proto.UnmarshalOptions{
		Resolver: m.resolver,
	}
	return options.Unmarshal(data, message)
}

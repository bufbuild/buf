package protoencoding

import (
	"google.golang.org/protobuf/proto"
)

type wireMarshaler struct{}

func newWireMarshaler() Marshaler {
	return &wireMarshaler{}
}

func (m *wireMarshaler) Marshal(message proto.Message) ([]byte, error) {
	options := proto.MarshalOptions{
		Deterministic: true,
	}
	return options.Marshal(message)
}

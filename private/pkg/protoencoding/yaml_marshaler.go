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
	"github.com/bufbuild/protoyaml-go"
	"google.golang.org/protobuf/proto"
)

type yamlMarshaler struct {
	resolver        Resolver
	indent          int
	useProtoNames   bool
	useEnumNumbers  bool
	emitUnpopulated bool
}

func newYAMLMarshaler(resolver Resolver, options ...YAMLMarshalerOption) Marshaler {
	yamlMarshaler := &yamlMarshaler{
		resolver: resolver,
	}
	for _, option := range options {
		option(yamlMarshaler)
	}
	return yamlMarshaler
}

func (m *yamlMarshaler) Marshal(message proto.Message) ([]byte, error) {
	if err := ReparseUnrecognized(m.resolver, message.ProtoReflect()); err != nil {
		return nil, err
	}
	options := protoyaml.MarshalOptions{
		Indent:          m.indent,
		Resolver:        m.resolver,
		UseProtoNames:   m.useProtoNames,
		UseEnumNumbers:  m.useEnumNumbers,
		EmitUnpopulated: m.emitUnpopulated,
	}
	return options.Marshal(message)
}

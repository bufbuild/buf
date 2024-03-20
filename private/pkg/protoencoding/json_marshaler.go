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
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type jsonMarshaler struct {
	resolver        Resolver
	indent          string
	useProtoNames   bool
	useEnumNumbers  bool
	emitUnpopulated bool
}

func newJSONMarshaler(resolver Resolver, options ...JSONMarshalerOption) Marshaler {
	jsonMarshaler := &jsonMarshaler{
		resolver: resolver,
	}
	for _, option := range options {
		option(jsonMarshaler)
	}
	return jsonMarshaler
}

func (m *jsonMarshaler) Marshal(message proto.Message) ([]byte, error) {
	if err := ReparseUnrecognized(m.resolver, message.ProtoReflect()); err != nil {
		return nil, err
	}
	options := protojson.MarshalOptions{
		Resolver:        m.resolver,
		UseProtoNames:   m.useProtoNames,
		UseEnumNumbers:  m.useEnumNumbers,
		EmitUnpopulated: m.emitUnpopulated,
		Indent:          m.indent,
	}
	return options.Marshal(message)
}

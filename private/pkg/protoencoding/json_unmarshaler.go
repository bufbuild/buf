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

type jsonUnmarshaler struct {
	resolver        Resolver
	disallowUnknown bool
}

func newJSONUnmarshaler(resolver Resolver, options ...JSONUnmarshalerOption) Unmarshaler {
	jsonUnmarshaler := &jsonUnmarshaler{
		resolver: resolver,
	}
	for _, option := range options {
		option(jsonUnmarshaler)
	}
	return jsonUnmarshaler
}

func (m *jsonUnmarshaler) Unmarshal(data []byte, message proto.Message) error {
	options := protojson.UnmarshalOptions{
		Resolver:       m.resolver,
		DiscardUnknown: !m.disallowUnknown,
	}
	return options.Unmarshal(data, message)
}

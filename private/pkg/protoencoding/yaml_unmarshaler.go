// Copyright 2020-2025 Buf Technologies, Inc.
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
	"fmt"

	"buf.build/go/protoyaml"
	"google.golang.org/protobuf/proto"
)

type yamlUnmarshaler struct {
	resolver  Resolver
	path      string
	validator protoyaml.Validator
}

func newYAMLUnmarshaler(resolver Resolver, options ...YAMLUnmarshalerOption) Unmarshaler {
	if resolver == nil {
		resolver = EmptyResolver
	}
	result := &yamlUnmarshaler{
		resolver: resolver,
	}
	for _, option := range options {
		option(result)
	}
	return result
}

func (m *yamlUnmarshaler) Unmarshal(data []byte, message proto.Message) error {
	options := protoyaml.UnmarshalOptions{
		Resolver:  m.resolver,
		Validator: m.validator,
		Path:      m.path,
	}
	if err := options.Unmarshal(data, message); err != nil {
		return fmt.Errorf("yaml unmarshal: %w", err)
	}
	return nil
}

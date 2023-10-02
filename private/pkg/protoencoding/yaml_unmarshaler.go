// Copyright 2020-2023 Buf Technologies, Inc.
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
	"github.com/bufbuild/protovalidate-go"
	"github.com/bufbuild/protoyaml-go"
	"google.golang.org/protobuf/proto"
)

type yamlUnmarshaler struct {
	resolver Resolver
	path     string
}

func newYAMLUnmarshaler(resolver Resolver, options ...YAMLUnmarshalerOption) Unmarshaler {
	result := &yamlUnmarshaler{
		resolver: resolver,
	}
	for _, option := range options {
		option(result)
	}
	return result
}

func (m *yamlUnmarshaler) Unmarshal(data []byte, message proto.Message) error {
	validator, err := protovalidate.New()
	if err != nil {
		return err
	}
	options := protoyaml.UnmarshalOptions{
		Resolver:  m.resolver,
		Validator: validator,
		Path:      m.path,
	}
	return options.Unmarshal(data, message)
}

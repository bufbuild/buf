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

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

type txtpbUnmarshaler struct {
	resolver Resolver
}

func newTxtpbUnmarshaler(resolver Resolver) Unmarshaler {
	if resolver == nil {
		resolver = EmptyResolver
	}
	return &txtpbUnmarshaler{
		resolver: resolver,
	}
}

func (m *txtpbUnmarshaler) Unmarshal(data []byte, message proto.Message) error {
	options := prototext.UnmarshalOptions{
		Resolver: m.resolver,
		// TODO: make this an option
		DiscardUnknown: true,
	}
	if err := options.Unmarshal(data, message); err != nil {
		return fmt.Errorf("txtpb unmarshal: %w", err)
	}
	return nil
}

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

	"google.golang.org/protobuf/proto"
)

type wireUnmarshaler struct {
	resolver Resolver
}

func newWireUnmarshaler(resolver Resolver) Unmarshaler {
	if resolver == nil {
		resolver = EmptyResolver
	}
	return &wireUnmarshaler{
		resolver: resolver,
	}
}

func (m *wireUnmarshaler) Unmarshal(data []byte, message proto.Message) error {
	options := proto.UnmarshalOptions{
		Resolver: m.resolver,
	}
	if err := options.Unmarshal(data, message); err != nil {
		return fmt.Errorf("wire unmarshal: %w", err)
	}
	return nil
}

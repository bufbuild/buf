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
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// If our call to detrand.Disable ever stops working, these tests will result in
// the data having random extra whitespaces on some test build.

func TestJSONStable(t *testing.T) {
	t.Parallel()

	fileDescriptorProto := &descriptorpb.FileDescriptorProto{Name: proto.String("a.proto")}
	data, err := NewJSONMarshaler(nil, JSONMarshalerWithIndent()).Marshal(fileDescriptorProto)
	require.NoError(t, err)
	require.Equal(
		t,
		"{\n  \"name\": \"a.proto\"\n}",
		string(data),
	)
}

func TestTxtpbStable(t *testing.T) {
	t.Parallel()

	fileDescriptorProto := &descriptorpb.FileDescriptorProto{Name: proto.String("a.proto")}
	data, err := NewTxtpbMarshaler(nil).Marshal(fileDescriptorProto)
	require.NoError(t, err)
	require.Equal(
		t,
		"name: \"a.proto\"\n",
		string(data),
	)
}

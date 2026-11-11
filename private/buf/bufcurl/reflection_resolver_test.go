// Copyright 2020-2026 Buf Technologies, Inc.
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

package bufcurl

import (
	"testing"

	reflectionv1 "github.com/bufbuild/buf/private/gen/proto/go/grpc/reflection/v1"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/jsonenumvalueoptionspb"
)

func TestDescriptorsInResponseEnumValueJSONName(t *testing.T) {
	t.Parallel()
	options := &descriptorpb.EnumValueOptions{}
	proto.SetExtension(options, jsonenumvalueoptionspb.E_Json, jsonenumvalueoptionspb.JsonEnumValueOptions_builder{
		String: new("red"),
	}.Build())
	data, err := protoencoding.NewWireMarshaler().Marshal(&descriptorpb.FileDescriptorProto{
		Name: new("a.proto"),
		EnumType: []*descriptorpb.EnumDescriptorProto{{
			Name:  new("Color"),
			Value: []*descriptorpb.EnumValueDescriptorProto{{Name: new("COLOR_RED"), Number: new(int32(0)), Options: options}},
		}},
	})
	require.NoError(t, err)
	files, err := descriptorsInResponse(reflectionv1.ServerReflectionResponse_builder{
		FileDescriptorResponse: reflectionv1.FileDescriptorResponse_builder{
			FileDescriptorProto: [][]byte{data},
		}.Build(),
	}.Build())
	require.NoError(t, err)
	require.Len(t, files, 1)
	// protojson only reads the option when it is parsed, not left as unknown fields.
	gotOptions := files[0].GetEnumType()[0].GetValue()[0].GetOptions()
	assert.Empty(t, gotOptions.ProtoReflect().GetUnknown())
	jsonOptions := gotOptions.ProtoReflect().Get(jsonenumvalueoptionspb.E_Json.TypeDescriptor()).Message()
	assert.Equal(t, "red", jsonOptions.Get(jsonOptions.Descriptor().Fields().ByName("string")).String())
}

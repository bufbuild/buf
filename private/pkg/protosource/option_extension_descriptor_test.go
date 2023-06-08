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

package protosource

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestOptionExtensionLocation(t *testing.T) {
	t.Parallel()
	locations := []*descriptorpb.SourceCodeInfo_Location{
		{
			Path:            []int32{1, 2, 3, 4, 5, 1099, 100, 101, 102},
			Span:            []int32{99, 100, 101},
			LeadingComments: proto.String("inside custom option 1099 (100)"),
		},
		{
			Path:            []int32{1, 2, 3, 4, 5, 1099, 100, 102, 103},
			Span:            []int32{99, 100, 102},
			LeadingComments: proto.String("inside custom option 1099 (100 again)"),
		},
		{
			Path:            []int32{1, 2, 3, 4, 5, 1099, 200},
			Span:            []int32{99, 200, 201},
			LeadingComments: proto.String("inside custom option 1099 (200)"),
		},
		{
			Path:            []int32{1, 2, 3, 4, 5, 1089},
			Span:            []int32{89, 1, 2},
			LeadingComments: proto.String("custom option 1089"),
		},
		{
			Path:            []int32{1, 2, 3, 4, 5, 1079},
			Span:            []int32{79, 1, 2},
			LeadingComments: proto.String("custom option 1079"),
		},
		{
			Path:            []int32{1, 2, 3, 4, 5, 1079},
			Span:            []int32{79, 11, 12},
			LeadingComments: proto.String("custom option 1079 (again)"),
		},
		{
			Path:            []int32{1, 2, 3, 4, 5},
			Span:            []int32{5, 1, 2},
			LeadingComments: proto.String("options"),
		},
	}
	locationStore := newLocationStore(locations)
	descriptor := newOptionExtensionDescriptor(&descriptorpb.MessageOptions{}, []int32{1, 2, 3, 4, 5}, locationStore)
	customOption1079 := makeCustomOption(t, 1079)
	customOption1089 := makeCustomOption(t, 1089)
	customOption1099 := makeCustomOption(t, 1099)
	customOption1109 := makeCustomOption(t, 1109)

	assert.Nil(t, descriptor.OptionExtensionLocation(customOption1109))
	assert.Nil(t, descriptor.OptionExtensionLocation(customOption1099, 100, 103))
	assert.Nil(t, descriptor.OptionExtensionLocation(customOption1099, 300))

	loc := descriptor.OptionExtensionLocation(customOption1099)
	checkLocation(t, loc, locations[0])
	loc = descriptor.OptionExtensionLocation(customOption1099, 100)
	checkLocation(t, loc, locations[0])
	loc = descriptor.OptionExtensionLocation(customOption1099, 100, 101)
	checkLocation(t, loc, locations[0])
	loc = descriptor.OptionExtensionLocation(customOption1099, 100, 101, 102)
	checkLocation(t, loc, locations[0])
	loc = descriptor.OptionExtensionLocation(customOption1099, 100, 101, 102, 103)
	checkLocation(t, loc, locations[0])

	loc = descriptor.OptionExtensionLocation(customOption1099, 100, 102)
	checkLocation(t, loc, locations[1])
	loc = descriptor.OptionExtensionLocation(customOption1099, 100, 102, 103)
	checkLocation(t, loc, locations[1])
	loc = descriptor.OptionExtensionLocation(customOption1099, 100, 102, 103, 1, 2, 3, 4)
	checkLocation(t, loc, locations[1])

	loc = descriptor.OptionExtensionLocation(customOption1099, 200)
	checkLocation(t, loc, locations[2])
	loc = descriptor.OptionExtensionLocation(customOption1099, 200, 0)
	checkLocation(t, loc, locations[2])

	loc = descriptor.OptionExtensionLocation(customOption1089)
	checkLocation(t, loc, locations[3])
	loc = descriptor.OptionExtensionLocation(customOption1089, 10)
	checkLocation(t, loc, locations[3])

	loc = descriptor.OptionExtensionLocation(customOption1079)
	checkLocation(t, loc, locations[4])
	loc = descriptor.OptionExtensionLocation(customOption1079, 1, 2, 3)
	checkLocation(t, loc, locations[4])
}

func checkLocation(t *testing.T, loc Location, sourceCodeInfoLoc *descriptorpb.SourceCodeInfo_Location) {
	t.Helper()
	assert.Equal(t, sourceCodeInfoLoc.GetLeadingComments(), loc.LeadingComments())
	span := []int32{int32(loc.StartLine() - 1), int32(loc.StartColumn() - 1)}
	if loc.EndLine() != loc.StartLine() {
		span = append(span, int32(loc.EndLine()-1))
	}
	span = append(span, int32(loc.EndColumn()-1))
	assert.Equal(t, sourceCodeInfoLoc.Span, span)
}

func makeCustomOption(t *testing.T, tag int32) protoreflect.ExtensionType {
	t.Helper()
	fileDescriptorProto := &descriptorpb.FileDescriptorProto{
		Name:       proto.String("test.proto"),
		Syntax:     proto.String("proto2"),
		Dependency: []string{"google/protobuf/descriptor.proto"},
		Extension: []*descriptorpb.FieldDescriptorProto{
			{
				Name:     proto.String("test"),
				Number:   proto.Int32(tag),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Extendee: proto.String(".google.protobuf.MessageOptions"),
			},
		},
	}
	fileDescriptor, err := protodesc.NewFile(fileDescriptorProto, protoregistry.GlobalFiles)
	require.NoError(t, err)
	return dynamicpb.NewExtensionType(fileDescriptor.Extensions().Get(0))
}

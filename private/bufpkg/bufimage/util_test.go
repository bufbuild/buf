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

package bufimage

import (
	"bytes"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	imagev1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/image/v1"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestStripBufExtensionField(t *testing.T) {
	t.Parallel()
	file := &imagev1.ImageFile{
		BufExtension: &imagev1.ImageFileExtension{
			IsImport:         proto.Bool(true),
			UnusedDependency: []int32{1, 3, 5},
			ModuleInfo: &imagev1.ModuleInfo{
				Name: &imagev1.ModuleName{
					Remote:     proto.String("buf.build"),
					Owner:      proto.String("foo"),
					Repository: proto.String("bar"),
				},
				Commit: proto.String("1234981234123412341234"),
			},
		},
	}
	dataToBeStripped, err := proto.Marshal(file)
	require.NoError(t, err)

	otherData := protowire.AppendTag(nil, 122, protowire.BytesType)
	otherData = protowire.AppendBytes(otherData, []byte{1, 18, 28, 123, 5, 3, 1})
	otherData = protowire.AppendTag(otherData, 123, protowire.VarintType)
	otherData = protowire.AppendVarint(otherData, 23456)
	otherData = protowire.AppendTag(otherData, 124, protowire.Fixed32Type)
	otherData = protowire.AppendFixed32(otherData, 23456)
	otherData = protowire.AppendTag(otherData, 125, protowire.Fixed64Type)
	otherData = protowire.AppendFixed64(otherData, 23456)
	otherData = protowire.AppendTag(otherData, 126, protowire.StartGroupType)
	{
		otherData = protowire.AppendTag(otherData, 1, protowire.VarintType)
		otherData = protowire.AppendVarint(otherData, 123)
		otherData = protowire.AppendTag(otherData, 2, protowire.BytesType)
		otherData = protowire.AppendBytes(otherData, []byte("foo-bar-baz"))
	}
	otherData = protowire.AppendTag(otherData, 126, protowire.EndGroupType)

	testCases := []struct {
		name           string
		input          []byte
		expectedOutput []byte
	}{
		{
			name:           "nothing to strip",
			input:          otherData,
			expectedOutput: otherData,
		},
		{
			name:           "nothing left after strip",
			input:          dataToBeStripped,
			expectedOutput: []byte{},
		},
		{
			name:           "stripped field at start",
			input:          bytes.Join([][]byte{dataToBeStripped, otherData}, nil),
			expectedOutput: otherData,
		},
		{
			name:           "stripped field at end",
			input:          bytes.Join([][]byte{otherData, dataToBeStripped}, nil),
			expectedOutput: otherData,
		},
		{
			name:           "stripped field in the middle",
			input:          bytes.Join([][]byte{otherData, dataToBeStripped, otherData}, nil),
			expectedOutput: bytes.Repeat(otherData, 2),
		},
	}
	for i := range testCases {
		testCase := testCases[i]
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			stripped := stripBufExtensionField(testCase.input)
			require.Equal(t, testCase.expectedOutput, []byte(stripped))
		})
	}
}

func TestImageToProtoPreservesUnrecognizedFields(t *testing.T) {
	t.Parallel()
	fileDescriptor := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("foo/bar/baz.proto"),
		Package: proto.String("foo.bar.baz"),
		Syntax:  proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("Foo"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     proto.String("id"),
						Number:   proto.Int32(1),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum(),
						JsonName: proto.String("id"),
					},
					{
						Name:     proto.String("name"),
						Number:   proto.Int32(2),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
						JsonName: proto.String("name"),
					},
				},
			},
		},
	}
	otherData := protowire.AppendTag(nil, 122, protowire.BytesType)
	otherData = protowire.AppendBytes(otherData, []byte{1, 18, 28, 123, 5, 3, 1})
	otherData = protowire.AppendTag(otherData, 123, protowire.VarintType)
	otherData = protowire.AppendVarint(otherData, 23456)
	otherData = protowire.AppendTag(otherData, 124, protowire.Fixed32Type)
	otherData = protowire.AppendFixed32(otherData, 23456)
	fileDescriptor.ProtoReflect().SetUnknown(otherData)

	module, err := bufmoduleref.ModuleIdentityForString("buf.build/foo/bar")
	require.NoError(t, err)
	imageFile, err := NewImageFile(
		fileDescriptor,
		module,
		"1234123451235",
		"foo/bar/baz.proto",
		false,
		false,
		nil,
	)
	require.NoError(t, err)

	protoImageFile := imageFileToProtoImageFile(imageFile)
	// make sure unrecognized bytes survived
	require.Equal(t, otherData, []byte(protoImageFile.ProtoReflect().GetUnknown()))

	// now round-trip it back through
	imageFileBytes, err := proto.Marshal(protoImageFile)
	require.NoError(t, err)

	roundTrippedFileDescriptor := &descriptorpb.FileDescriptorProto{}
	err = proto.Unmarshal(imageFileBytes, roundTrippedFileDescriptor)
	require.NoError(t, err)
	// unrecognized now includes image file's buf extension
	require.Greater(t, len(roundTrippedFileDescriptor.ProtoReflect().GetUnknown()), len(otherData))

	// if we go back through an image file, we should strip out the
	// buf extension unknown bytes but preserve the rest
	module, err = bufmoduleref.ModuleIdentityForString("buf.build/abc/def")
	require.NoError(t, err)
	// NB: intentionally different metadata
	imageFile, err = NewImageFile(
		fileDescriptor,
		module,
		"987654321",
		"abc/def/xyz.proto",
		false,
		true,
		[]int32{1, 2, 3},
	)
	require.NoError(t, err)

	protoImageFile = imageFileToProtoImageFile(imageFile)
	// make sure unrecognized bytes survived and extraneous buf extension is not present
	require.Equal(t, otherData, []byte(protoImageFile.ProtoReflect().GetUnknown()))

	// double-check via round-trip, to make sure resulting image file equals the input
	// (to verify that the original unknown bytes byf extension didn't interfere)
	imageFileBytes, err = proto.Marshal(protoImageFile)
	require.NoError(t, err)

	roundTrippedImageFile := &imagev1.ImageFile{}
	err = proto.Unmarshal(imageFileBytes, roundTrippedImageFile)
	require.NoError(t, err)

	diff := cmp.Diff(protoImageFile, roundTrippedImageFile, protocmp.Transform())
	require.Empty(t, diff)
}

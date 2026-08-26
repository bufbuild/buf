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

package bufimage

import (
	"reflect"
	"testing"

	imagev1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/image/v1"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestCloneImage(t *testing.T) {
	t.Parallel()
	protoImage := imagev1.Image_builder{
		File: []*imagev1.ImageFile{
			imagev1.ImageFile_builder{
				Syntax:     new("proto3"),
				Name:       new("a.proto"),
				Dependency: []string{"b.proto", "c.proto"},
				Package:    new("abc.def"),
				MessageType: []*descriptorpb.DescriptorProto{
					{
						Name: new("Msg"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:     new("id"),
								Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
								Number:   proto.Int32(1),
								Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
								JsonName: new("id"),
							},
							{
								Name:     new("en"),
								Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
								Number:   proto.Int32(2),
								Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
								TypeName: new(".abc.def.Enum"),
								JsonName: new("en"),
							},
						},
					},
				},
				BufExtension: imagev1.ImageFileExtension_builder{
					IsImport:            new(false),
					IsSyntaxUnspecified: new(false),
					UnusedDependency:    []int32{1},
					ModuleInfo: imagev1.ModuleInfo_builder{
						Name: imagev1.ModuleName_builder{
							Remote:     new("buf.build"),
							Owner:      new("foo"),
							Repository: new("bar"),
						}.Build(),
						Commit: new("9876543210fedcba9876543210fedcba"),
					}.Build(),
				}.Build(),
			}.Build(),
			imagev1.ImageFile_builder{
				Syntax:  new("proto3"),
				Name:    new("b.proto"),
				Package: new("abc.def"),
				EnumType: []*descriptorpb.EnumDescriptorProto{
					{
						Name: new("Enum"),
						Value: []*descriptorpb.EnumValueDescriptorProto{
							{
								Name:   new("ZERO"),
								Number: proto.Int32(0),
							},
							{
								Name:   new("ONE"),
								Number: proto.Int32(1),
							},
						},
					},
				},
				BufExtension: imagev1.ImageFileExtension_builder{
					IsImport:            new(true),
					IsSyntaxUnspecified: new(false),
					ModuleInfo: imagev1.ModuleInfo_builder{
						Name: imagev1.ModuleName_builder{
							Remote:     new("buf.build"),
							Owner:      new("foo"),
							Repository: new("baz"),
						}.Build(),
						Commit: new("0123456789abcdef0123456789abcdef"),
					}.Build(),
				}.Build(),
			}.Build(),
			imagev1.ImageFile_builder{
				Syntax: new("proto2"),
				Name:   new("c.proto"),
				BufExtension: imagev1.ImageFileExtension_builder{
					IsImport:            new(true),
					IsSyntaxUnspecified: new(true),
					ModuleInfo: imagev1.ModuleInfo_builder{
						Name: imagev1.ModuleName_builder{
							Remote:     new("buf.build"),
							Owner:      new("foo"),
							Repository: new("baz"),
						}.Build(),
						Commit: new("0123456789abcdef0123456789abcdef"),
					}.Build(),
				}.Build(),
			}.Build(),
		},
	}.Build()

	image, err := NewImageForProto(protoImage)
	require.NoError(t, err)

	clone, err := CloneImage(image)
	require.NoError(t, err)

	// Test that they are equal by comparing their proto versions
	protoClone, err := ImageToProtoImage(clone)
	require.NoError(t, err)

	require.Empty(t, cmp.Diff(protoImage, protoClone, protocmp.Transform()))

	// Verify the pointer values are different
	for _, imageFile := range image.Files() {
		cloneFile := clone.GetFile(imageFile.Path())
		require.NotSame(t, imageFile.FileDescriptorProto(), cloneFile.FileDescriptorProto())
		unused := reflect.ValueOf(imageFile.UnusedDependencyIndexes()).Pointer()
		cloneUnused := reflect.ValueOf(cloneFile.UnusedDependencyIndexes()).Pointer()
		if unused != 0 || cloneUnused != 0 {
			// They can both be nil. But otherwise must not be equal since that
			// means the backing arrays are shared.
			require.NotEqual(t, unused, cloneUnused)
		}
	}
}

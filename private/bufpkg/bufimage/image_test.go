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

package bufimage

import (
	"reflect"
	"testing"

	imagev1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/image/v1"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestMergeImagesWithImports(t *testing.T) {
	t.Parallel()
	firstProtoImage := &imagev1.Image{
		File: []*imagev1.ImageFile{
			{
				Syntax:     proto.String("proto3"),
				Name:       proto.String("a.proto"),
				Dependency: []string{"b.proto"},
			},
			{
				Syntax: proto.String("proto3"),
				Name:   proto.String("b.proto"),
				BufExtension: &imagev1.ImageFileExtension{
					IsImport: proto.Bool(true),
				},
			},
			{
				Syntax: proto.String("proto3"),
				Name:   proto.String("c.proto"),
				BufExtension: &imagev1.ImageFileExtension{
					IsImport: proto.Bool(true),
				},
			},
		},
	}
	secondProtoImage := &imagev1.Image{
		File: []*imagev1.ImageFile{
			{
				Syntax: proto.String("proto3"),
				Name:   proto.String("b.proto"),
			},
		},
	}

	firstImage, err := NewImageForProto(firstProtoImage)
	require.NoError(t, err)
	secondImage, err := NewImageForProto(secondProtoImage)
	require.NoError(t, err)
	mergedImage, err := MergeImages(firstImage, secondImage)
	require.NoError(t, err)

	imageFiles := mergedImage.Files()
	require.Len(t, imageFiles, 3)
	assert.False(t, mergedImage.GetFile("a.proto").IsImport())
	assert.False(t, mergedImage.GetFile("b.proto").IsImport())
	assert.True(t, mergedImage.GetFile("c.proto").IsImport())
}

func TestMergeImagesWithDuplicateFile(t *testing.T) {
	t.Parallel()
	firstProtoImage := &imagev1.Image{
		File: []*imagev1.ImageFile{
			{
				Syntax: proto.String("proto3"),
				Name:   proto.String("a.proto"),
			},
		},
	}
	secondProtoImage := &imagev1.Image{
		File: []*imagev1.ImageFile{
			{
				Syntax: proto.String("proto3"),
				Name:   proto.String("a.proto"),
			},
		},
	}

	firstImage, err := NewImageForProto(firstProtoImage)
	require.NoError(t, err)
	secondImage, err := NewImageForProto(secondProtoImage)
	require.NoError(t, err)
	_, err = MergeImages(firstImage, secondImage)
	require.Error(t, err)
}

func TestMergeImagesOrdered(t *testing.T) {
	t.Parallel()
	firstProtoImage := &imagev1.Image{
		File: []*imagev1.ImageFile{
			{
				Syntax: proto.String("proto3"),
				Name:   proto.String("a.proto"),
			},
			{
				Syntax: proto.String("proto3"),
				Name:   proto.String("b.proto"),
			},
		},
	}
	secondProtoImage := &imagev1.Image{
		File: []*imagev1.ImageFile{
			{
				Syntax: proto.String("proto3"),
				Name:   proto.String("c.proto"),
			},
			{
				Syntax: proto.String("proto3"),
				Name:   proto.String("d.proto"),
			},
		},
	}

	firstImage, err := NewImageForProto(firstProtoImage)
	require.NoError(t, err)
	secondImage, err := NewImageForProto(secondProtoImage)
	require.NoError(t, err)
	image, err := MergeImages(firstImage, secondImage)
	require.NoError(t, err)
	var paths []string
	for _, imageFile := range image.Files() {
		paths = append(paths, imageFile.Path())
	}
	assert.Equal(t, []string{"a.proto", "b.proto", "c.proto", "d.proto"}, paths)
}

func TestCloneImage(t *testing.T) {
	t.Parallel()
	protoImage := &imagev1.Image{
		File: []*imagev1.ImageFile{
			{
				Syntax:     proto.String("proto3"),
				Name:       proto.String("a.proto"),
				Dependency: []string{"b.proto", "c.proto"},
				Package:    proto.String("abc.def"),
				MessageType: []*descriptorpb.DescriptorProto{
					{
						Name: proto.String("Msg"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:     proto.String("id"),
								Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
								Number:   proto.Int32(1),
								Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
								JsonName: proto.String("id"),
							},
							{
								Name:     proto.String("en"),
								Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
								Number:   proto.Int32(2),
								Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
								TypeName: proto.String(".abc.def.Enum"),
								JsonName: proto.String("en"),
							},
						},
					},
				},
				BufExtension: &imagev1.ImageFileExtension{
					IsImport:            proto.Bool(false),
					IsSyntaxUnspecified: proto.Bool(false),
					UnusedDependency:    []int32{1},
					ModuleInfo: &imagev1.ModuleInfo{
						Name: &imagev1.ModuleName{
							Remote:     proto.String("buf.build"),
							Owner:      proto.String("foo"),
							Repository: proto.String("bar"),
						},
					},
				},
			},
			{
				Syntax:  proto.String("proto3"),
				Name:    proto.String("b.proto"),
				Package: proto.String("abc.def"),
				EnumType: []*descriptorpb.EnumDescriptorProto{
					{
						Name: proto.String("Enum"),
						Value: []*descriptorpb.EnumValueDescriptorProto{
							{
								Name:   proto.String("ZERO"),
								Number: proto.Int32(0),
							},
							{
								Name:   proto.String("ONE"),
								Number: proto.Int32(1),
							},
						},
					},
				},
				BufExtension: &imagev1.ImageFileExtension{
					IsImport:            proto.Bool(true),
					IsSyntaxUnspecified: proto.Bool(false),
					ModuleInfo: &imagev1.ModuleInfo{
						Name: &imagev1.ModuleName{
							Remote:     proto.String("buf.build"),
							Owner:      proto.String("foo"),
							Repository: proto.String("baz"),
						},
					},
				},
			},
			{
				Syntax: proto.String("proto2"),
				Name:   proto.String("c.proto"),
				BufExtension: &imagev1.ImageFileExtension{
					IsImport:            proto.Bool(true),
					IsSyntaxUnspecified: proto.Bool(true),
					ModuleInfo: &imagev1.ModuleInfo{
						Name: &imagev1.ModuleName{
							Remote:     proto.String("buf.build"),
							Owner:      proto.String("foo"),
							Repository: proto.String("baz"),
						},
					},
				},
			},
		},
	}

	image, err := NewImageForProto(protoImage)
	require.NoError(t, err)

	clone, err := CloneImage(image)
	require.NoError(t, err)

	// Test that they are equal by comparing their proto versions
	protoClone := ImageToProtoImage(clone)

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

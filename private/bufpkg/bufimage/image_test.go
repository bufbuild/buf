// Copyright 2020-2022 Buf Technologies, Inc.
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
	"testing"

	imagev1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/image/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
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

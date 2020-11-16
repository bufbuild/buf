// Copyright 2020 Buf Technologies, Inc.
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

package bufimagetesting

import (
	"fmt"
	"testing"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage"
	imagev1 "github.com/bufbuild/buf/internal/gen/proto/go/buf/image/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func BenchmarkNewImageWithOnlyPathsAllowNotExistFileOnly(b *testing.B) {
	var imageFiles []bufimage.ImageFile
	for i := 0; i < 3000; i++ {
		imageFiles = append(
			imageFiles,
			NewImageFile(
				b,
				NewFileDescriptorProto(
					b,
					fmt.Sprintf("a%d.proto/a%d.proto", i, i),
				),
				fmt.Sprintf("foo/two/a%d.proto/a%d.proto", i, i),
				false,
			),
		)
	}
	image, err := bufimage.NewImage(imageFiles)
	require.NoError(b, err)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		newImage, err := bufimage.ImageWithOnlyPathsAllowNotExist(image, []string{"a1.proto/a1.proto"})
		// this does increase the time but we're just looking for order of magnitude
		// between this and the below benchmark function
		require.NoError(b, err)
		require.Equal(b, 1, len(newImage.Files()))
	}
}

func BenchmarkNewImageWithOnlyPathsAllowNotExistDirOnly(b *testing.B) {
	var imageFiles []bufimage.ImageFile
	for i := 0; i < 3000; i++ {
		imageFiles = append(
			imageFiles,
			NewImageFile(
				b,
				NewFileDescriptorProto(
					b,
					fmt.Sprintf("a%d.proto/a%d.proto", i, i),
				),
				fmt.Sprintf("foo/two/a%d.proto/a%d.proto", i, i),
				false,
			),
		)
	}
	image, err := bufimage.NewImage(imageFiles)
	require.NoError(b, err)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		newImage, err := bufimage.ImageWithOnlyPathsAllowNotExist(image, []string{"a1.proto"})
		require.NoError(b, err)
		require.Equal(b, 1, len(newImage.Files()))
	}
}

func TestBasic(t *testing.T) {
	t.Parallel()

	fileDescriptorProtoImport := NewFileDescriptorProto(
		t,
		"import.proto",
	)
	fileDescriptorProtoAA := NewFileDescriptorProto(
		t,
		"a/a.proto",
	)
	fileDescriptorProtoAB := NewFileDescriptorProto(
		t,
		"a/b.proto",
		"import.proto",
	)
	fileDescriptorProtoBA := NewFileDescriptorProto(
		t,
		"b/a.proto",
		"a/a.proto",
		"a/b.proto",
	)
	fileDescriptorProtoBB := NewFileDescriptorProto(
		t,
		"b/b.proto",
		"a/a.proto",
		"a/b.proto",
		"b/b.proto",
	)
	fileDescriptorProtoOutlandishDirectoryName := NewFileDescriptorProto(
		t,
		"d/d.proto/d.proto",
		"import.proto",
	)

	fileImport := NewImageFile(
		t,
		fileDescriptorProtoImport,
		"some/import/import.proto",
		true,
	)
	fileOneAA := NewImageFile(
		t,
		fileDescriptorProtoAA,
		"foo/one/a/a.proto",
		false,
	)
	fileOneAB := NewImageFile(
		t,
		fileDescriptorProtoAB,
		"foo/one/a/b.proto",
		false,
	)
	fileTwoBA := NewImageFile(
		t,
		fileDescriptorProtoBA,
		"foo/two/b/a.proto",
		false,
	)
	fileTwoBB := NewImageFile(
		t,
		fileDescriptorProtoBB,
		"foo/two/b/b.proto",
		false,
	)
	fileOutlandishDirectoryName := NewImageFile(
		t,
		fileDescriptorProtoOutlandishDirectoryName,
		"foo/three/d/d.proto/d.proto",
		false,
	)

	image, err := bufimage.NewImage(
		[]bufimage.ImageFile{
			fileOneAA,
			fileImport,
			fileOneAB,
			fileTwoBA,
			fileTwoBB,
			fileOutlandishDirectoryName,
		},
	)
	require.NoError(t, err)
	AssertImageFilesEqual(
		t,
		[]bufimage.ImageFile{
			NewImageFile(t, fileDescriptorProtoAA, "foo/one/a/a.proto", false),
			NewImageFile(t, fileDescriptorProtoImport, "some/import/import.proto", true),
			NewImageFile(t, fileDescriptorProtoAB, "foo/one/a/b.proto", false),
			NewImageFile(t, fileDescriptorProtoBA, "foo/two/b/a.proto", false),
			NewImageFile(t, fileDescriptorProtoBB, "foo/two/b/b.proto", false),
			NewImageFile(t, fileDescriptorProtoOutlandishDirectoryName, "foo/three/d/d.proto/d.proto", false),
		},
		image.Files(),
	)
	require.NotNil(t, image.GetFile("a/a.proto"))
	require.Nil(t, image.GetFile("one/a/a.proto"))
	AssertImageFilesEqual(
		t,
		[]bufimage.ImageFile{
			NewImageFile(t, fileDescriptorProtoAA, "foo/one/a/a.proto", false),
			NewImageFile(t, fileDescriptorProtoAB, "foo/one/a/b.proto", false),
			NewImageFile(t, fileDescriptorProtoBA, "foo/two/b/a.proto", false),
			NewImageFile(t, fileDescriptorProtoBB, "foo/two/b/b.proto", false),
			NewImageFile(t, fileDescriptorProtoOutlandishDirectoryName, "foo/three/d/d.proto/d.proto", false),
		},
		bufimage.ImageWithoutImports(image).Files(),
	)

	newImage, err := bufimage.ImageWithOnlyPaths(
		image,
		[]string{
			"b/a.proto",
			"a/b.proto",
		},
	)
	require.NoError(t, err)
	AssertImageFilesEqual(
		t,
		[]bufimage.ImageFile{
			NewImageFile(t, fileDescriptorProtoAA, "foo/one/a/a.proto", true),
			NewImageFile(t, fileDescriptorProtoImport, "some/import/import.proto", true),
			NewImageFile(t, fileDescriptorProtoAB, "foo/one/a/b.proto", false),
			NewImageFile(t, fileDescriptorProtoBA, "foo/two/b/a.proto", false),
		},
		newImage.Files(),
	)

	_, err = bufimage.ImageWithOnlyPaths(
		image,
		[]string{
			"b/a.proto",
			"a/b.proto",
			"foo.proto",
		},
	)
	require.Error(t, err)

	newImage, err = bufimage.ImageWithOnlyPathsAllowNotExist(
		image,
		[]string{
			"b/a.proto",
			"a/b.proto",
			"foo.proto",
		},
	)
	require.NoError(t, err)
	AssertImageFilesEqual(
		t,
		[]bufimage.ImageFile{
			NewImageFile(t, fileDescriptorProtoAA, "foo/one/a/a.proto", true),
			NewImageFile(t, fileDescriptorProtoImport, "some/import/import.proto", true),
			NewImageFile(t, fileDescriptorProtoAB, "foo/one/a/b.proto", false),
			NewImageFile(t, fileDescriptorProtoBA, "foo/two/b/a.proto", false),
		},
		newImage.Files(),
	)
	newImage, err = bufimage.ImageWithOnlyPaths(
		image,
		[]string{
			"a",
		},
	)
	require.NoError(t, err)
	AssertImageFilesEqual(
		t,
		[]bufimage.ImageFile{
			NewImageFile(t, fileDescriptorProtoAA, "foo/one/a/a.proto", false),
			NewImageFile(t, fileDescriptorProtoImport, "some/import/import.proto", true),
			NewImageFile(t, fileDescriptorProtoAB, "foo/one/a/b.proto", false),
		},
		newImage.Files(),
	)
	newImage, err = bufimage.ImageWithOnlyPaths(
		image,
		[]string{
			"b",
		},
	)
	require.NoError(t, err)
	AssertImageFilesEqual(
		t,
		[]bufimage.ImageFile{
			NewImageFile(t, fileDescriptorProtoAA, "foo/one/a/a.proto", true),
			NewImageFile(t, fileDescriptorProtoImport, "some/import/import.proto", true),
			NewImageFile(t, fileDescriptorProtoAB, "foo/one/a/b.proto", true),
			NewImageFile(t, fileDescriptorProtoBA, "foo/two/b/a.proto", false),
			NewImageFile(t, fileDescriptorProtoBB, "foo/two/b/b.proto", false),
		},
		newImage.Files(),
	)
	newImage, err = bufimage.ImageWithOnlyPaths(
		image,
		[]string{
			"a",
			"b/a.proto",
		},
	)
	require.NoError(t, err)
	AssertImageFilesEqual(
		t,
		[]bufimage.ImageFile{
			NewImageFile(t, fileDescriptorProtoAA, "foo/one/a/a.proto", false),
			NewImageFile(t, fileDescriptorProtoImport, "some/import/import.proto", true),
			NewImageFile(t, fileDescriptorProtoAB, "foo/one/a/b.proto", false),
			NewImageFile(t, fileDescriptorProtoBA, "foo/two/b/a.proto", false),
		},
		newImage.Files(),
	)
	_, err = bufimage.ImageWithOnlyPaths(
		image,
		[]string{
			"a",
			"b/a.proto",
			"c",
		},
	)
	require.Error(t, err)
	newImage, err = bufimage.ImageWithOnlyPathsAllowNotExist(
		image,
		[]string{
			"a",
			"b/a.proto",
			"c",
		},
	)
	require.NoError(t, err)
	AssertImageFilesEqual(
		t,
		[]bufimage.ImageFile{
			NewImageFile(t, fileDescriptorProtoAA, "foo/one/a/a.proto", false),
			NewImageFile(t, fileDescriptorProtoImport, "some/import/import.proto", true),
			NewImageFile(t, fileDescriptorProtoAB, "foo/one/a/b.proto", false),
			NewImageFile(t, fileDescriptorProtoBA, "foo/two/b/a.proto", false),
		},
		newImage.Files(),
	)
	newImage, err = bufimage.ImageWithOnlyPaths(
		image,
		[]string{
			"a",
			"b/a.proto",
			"d/d.proto/d.proto",
		},
	)
	require.NoError(t, err)
	AssertImageFilesEqual(
		t,
		[]bufimage.ImageFile{
			NewImageFile(t, fileDescriptorProtoAA, "foo/one/a/a.proto", false),
			NewImageFile(t, fileDescriptorProtoImport, "some/import/import.proto", true),
			NewImageFile(t, fileDescriptorProtoAB, "foo/one/a/b.proto", false),
			NewImageFile(t, fileDescriptorProtoBA, "foo/two/b/a.proto", false),
			NewImageFile(t, fileDescriptorProtoOutlandishDirectoryName, "foo/three/d/d.proto/d.proto", false),
		},
		newImage.Files(),
	)
	newImage, err = bufimage.ImageWithOnlyPaths(
		image,
		[]string{
			"a",
			"b/a.proto",
			"d/d.proto",
		},
	)
	require.NoError(t, err)
	AssertImageFilesEqual(
		t,
		[]bufimage.ImageFile{
			NewImageFile(t, fileDescriptorProtoAA, "foo/one/a/a.proto", false),
			NewImageFile(t, fileDescriptorProtoImport, "some/import/import.proto", true),
			NewImageFile(t, fileDescriptorProtoAB, "foo/one/a/b.proto", false),
			NewImageFile(t, fileDescriptorProtoBA, "foo/two/b/a.proto", false),
			NewImageFile(t, fileDescriptorProtoOutlandishDirectoryName, "foo/three/d/d.proto/d.proto", false),
		},
		newImage.Files(),
	)

	protoImage := &imagev1.Image{
		File: []*descriptorpb.FileDescriptorProto{
			fileDescriptorProtoAA,
			fileDescriptorProtoImport,
			fileDescriptorProtoAB,
			fileDescriptorProtoBA,
			fileDescriptorProtoBB,
			fileDescriptorProtoOutlandishDirectoryName,
		},
		BufbuildImageExtension: &imagev1.ImageExtension{
			ImageImportRefs: []*imagev1.ImageImportRef{
				{
					FileIndex: proto.Uint32(1),
				},
			},
		},
	}
	newImage, err = bufimage.NewImageForProto(protoImage)
	require.NoError(t, err)
	AssertImageFilesEqual(
		t,
		[]bufimage.ImageFile{
			NewImageFile(t, fileDescriptorProtoAA, "a/a.proto", false),
			NewImageFile(t, fileDescriptorProtoImport, "import.proto", true),
			NewImageFile(t, fileDescriptorProtoAB, "a/b.proto", false),
			NewImageFile(t, fileDescriptorProtoBA, "b/a.proto", false),
			NewImageFile(t, fileDescriptorProtoBB, "b/b.proto", false),
			NewImageFile(t, fileDescriptorProtoOutlandishDirectoryName, "d/d.proto/d.proto", false),
		},
		newImage.Files(),
	)
	require.Equal(
		t,
		protoImage,
		bufimage.ImageToProtoImage(newImage),
	)
	require.Equal(
		t,
		&descriptorpb.FileDescriptorSet{
			File: []*descriptorpb.FileDescriptorProto{
				fileDescriptorProtoAA,
				fileDescriptorProtoImport,
				fileDescriptorProtoAB,
				fileDescriptorProtoBA,
				fileDescriptorProtoBB,
				fileDescriptorProtoOutlandishDirectoryName,
			},
		},
		bufimage.ImageToFileDescriptorSet(image),
	)
	codeGeneratorRequest := &pluginpb.CodeGeneratorRequest{
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			fileDescriptorProtoAA,
			fileDescriptorProtoImport,
			fileDescriptorProtoAB,
			fileDescriptorProtoBA,
			fileDescriptorProtoBB,
			fileDescriptorProtoOutlandishDirectoryName,
		},
		Parameter: proto.String("foo"),
		FileToGenerate: []string{
			"a/a.proto",
			"a/b.proto",
			"b/a.proto",
			"b/b.proto",
			"d/d.proto/d.proto",
		},
	}
	require.Equal(
		t,
		codeGeneratorRequest,
		bufimage.ImageToCodeGeneratorRequest(image, "foo"),
	)
	newImage, err = bufimage.NewImageForCodeGeneratorRequest(codeGeneratorRequest)
	require.NoError(t, err)
	AssertImageFilesEqual(
		t,
		[]bufimage.ImageFile{
			NewImageFile(t, fileDescriptorProtoAA, "a/a.proto", false),
			NewImageFile(t, fileDescriptorProtoImport, "import.proto", true),
			NewImageFile(t, fileDescriptorProtoAB, "a/b.proto", false),
			NewImageFile(t, fileDescriptorProtoBA, "b/a.proto", false),
			NewImageFile(t, fileDescriptorProtoBB, "b/b.proto", false),
			NewImageFile(t, fileDescriptorProtoOutlandishDirectoryName, "d/d.proto/d.proto", false),
		},
		newImage.Files(),
	)
}

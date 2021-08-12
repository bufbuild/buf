// Copyright 2020-2021 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/internal/buf/bufimage"
	imagev1 "github.com/bufbuild/buf/internal/gen/proto/go/buf/alpha/image/v1"
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
				NewProtoImageFile(
					b,
					fmt.Sprintf("a%d.proto/a%d.proto", i, i),
				),
				nil,
				"",
				fmt.Sprintf("foo/two/a%d.proto/a%d.proto", i, i),
				false,
				false,
				nil,
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
				NewProtoImageFile(
					b,
					fmt.Sprintf("a%d.proto/a%d.proto", i, i),
				),
				nil,
				"",
				fmt.Sprintf("foo/two/a%d.proto/a%d.proto", i, i),
				false,
				false,
				nil,
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

	protoImageFileImport := NewProtoImageFileIsImport(
		t,
		"import.proto",
	)
	protoImageFileAA := NewProtoImageFile(
		t,
		"a/a.proto",
	)
	protoImageFileAB := NewProtoImageFile(
		t,
		"a/b.proto",
		"import.proto",
	)
	protoImageFileBA := NewProtoImageFile(
		t,
		"b/a.proto",
		"a/a.proto",
		"a/b.proto",
	)
	protoImageFileBB := NewProtoImageFile(
		t,
		"b/b.proto",
		"a/a.proto",
		"a/b.proto",
		"b/b.proto",
	)
	protoImageFileOutlandishDirectoryName := NewProtoImageFile(
		t,
		"d/d.proto/d.proto",
		"import.proto",
	)

	fileImport := NewImageFile(
		t,
		protoImageFileImport,
		nil,
		"",
		"some/import/import.proto",
		true,
		false,
		nil,
	)
	fileOneAA := NewImageFile(
		t,
		protoImageFileAA,
		nil,
		"",
		"foo/one/a/a.proto",
		false,
		false,
		nil,
	)
	fileOneAB := NewImageFile(
		t,
		protoImageFileAB,
		nil,
		"",
		"foo/one/a/b.proto",
		false,
		false,
		nil,
	)
	fileTwoBA := NewImageFile(
		t,
		protoImageFileBA,
		nil,
		"",
		"foo/two/b/a.proto",
		false,
		false,
		nil,
	)
	fileTwoBB := NewImageFile(
		t,
		protoImageFileBB,
		nil,
		"",
		"foo/two/b/b.proto",
		false,
		false,
		nil,
	)
	fileOutlandishDirectoryName := NewImageFile(
		t,
		protoImageFileOutlandishDirectoryName,
		nil,
		"",
		"foo/three/d/d.proto/d.proto",
		false,
		false,
		nil,
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
			NewImageFile(t, protoImageFileAA, nil, "", "foo/one/a/a.proto", false, false, nil),
			NewImageFile(t, protoImageFileImport, nil, "", "some/import/import.proto", true, false, nil),
			NewImageFile(t, protoImageFileAB, nil, "", "foo/one/a/b.proto", false, false, nil),
			NewImageFile(t, protoImageFileBA, nil, "", "foo/two/b/a.proto", false, false, nil),
			NewImageFile(t, protoImageFileBB, nil, "", "foo/two/b/b.proto", false, false, nil),
			NewImageFile(t, protoImageFileOutlandishDirectoryName, nil, "", "foo/three/d/d.proto/d.proto", false, false, nil),
		},
		image.Files(),
	)
	require.NotNil(t, image.GetFile("a/a.proto"))
	require.Nil(t, image.GetFile("one/a/a.proto"))
	AssertImageFilesEqual(
		t,
		[]bufimage.ImageFile{
			NewImageFile(t, protoImageFileAA, nil, "", "foo/one/a/a.proto", false, false, nil),
			NewImageFile(t, protoImageFileAB, nil, "", "foo/one/a/b.proto", false, false, nil),
			NewImageFile(t, protoImageFileBA, nil, "", "foo/two/b/a.proto", false, false, nil),
			NewImageFile(t, protoImageFileBB, nil, "", "foo/two/b/b.proto", false, false, nil),
			NewImageFile(t, protoImageFileOutlandishDirectoryName, nil, "", "foo/three/d/d.proto/d.proto", false, false, nil),
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
			NewImageFile(t, protoImageFileAA, nil, "", "foo/one/a/a.proto", true, false, nil),
			NewImageFile(t, protoImageFileImport, nil, "", "some/import/import.proto", true, false, nil),
			NewImageFile(t, protoImageFileAB, nil, "", "foo/one/a/b.proto", false, false, nil),
			NewImageFile(t, protoImageFileBA, nil, "", "foo/two/b/a.proto", false, false, nil),
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
			NewImageFile(t, protoImageFileAA, nil, "", "foo/one/a/a.proto", true, false, nil),
			NewImageFile(t, protoImageFileImport, nil, "", "some/import/import.proto", true, false, nil),
			NewImageFile(t, protoImageFileAB, nil, "", "foo/one/a/b.proto", false, false, nil),
			NewImageFile(t, protoImageFileBA, nil, "", "foo/two/b/a.proto", false, false, nil),
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
			NewImageFile(t, protoImageFileAA, nil, "", "foo/one/a/a.proto", false, false, nil),
			NewImageFile(t, protoImageFileImport, nil, "", "some/import/import.proto", true, false, nil),
			NewImageFile(t, protoImageFileAB, nil, "", "foo/one/a/b.proto", false, false, nil),
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
			NewImageFile(t, protoImageFileAA, nil, "", "foo/one/a/a.proto", true, false, nil),
			NewImageFile(t, protoImageFileImport, nil, "", "some/import/import.proto", true, false, nil),
			NewImageFile(t, protoImageFileAB, nil, "", "foo/one/a/b.proto", true, false, nil),
			NewImageFile(t, protoImageFileBA, nil, "", "foo/two/b/a.proto", false, false, nil),
			NewImageFile(t, protoImageFileBB, nil, "", "foo/two/b/b.proto", false, false, nil),
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
			NewImageFile(t, protoImageFileAA, nil, "", "foo/one/a/a.proto", false, false, nil),
			NewImageFile(t, protoImageFileImport, nil, "", "some/import/import.proto", true, false, nil),
			NewImageFile(t, protoImageFileAB, nil, "", "foo/one/a/b.proto", false, false, nil),
			NewImageFile(t, protoImageFileBA, nil, "", "foo/two/b/a.proto", false, false, nil),
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
			NewImageFile(t, protoImageFileAA, nil, "", "foo/one/a/a.proto", false, false, nil),
			NewImageFile(t, protoImageFileImport, nil, "", "some/import/import.proto", true, false, nil),
			NewImageFile(t, protoImageFileAB, nil, "", "foo/one/a/b.proto", false, false, nil),
			NewImageFile(t, protoImageFileBA, nil, "", "foo/two/b/a.proto", false, false, nil),
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
			NewImageFile(t, protoImageFileAA, nil, "", "foo/one/a/a.proto", false, false, nil),
			NewImageFile(t, protoImageFileImport, nil, "", "some/import/import.proto", true, false, nil),
			NewImageFile(t, protoImageFileAB, nil, "", "foo/one/a/b.proto", false, false, nil),
			NewImageFile(t, protoImageFileBA, nil, "", "foo/two/b/a.proto", false, false, nil),
			NewImageFile(t, protoImageFileOutlandishDirectoryName, nil, "", "foo/three/d/d.proto/d.proto", false, false, nil),
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
			NewImageFile(t, protoImageFileAA, nil, "", "foo/one/a/a.proto", false, false, nil),
			NewImageFile(t, protoImageFileImport, nil, "", "some/import/import.proto", true, false, nil),
			NewImageFile(t, protoImageFileAB, nil, "", "foo/one/a/b.proto", false, false, nil),
			NewImageFile(t, protoImageFileBA, nil, "", "foo/two/b/a.proto", false, false, nil),
			NewImageFile(t, protoImageFileOutlandishDirectoryName, nil, "", "foo/three/d/d.proto/d.proto", false, false, nil),
		},
		newImage.Files(),
	)

	protoImage := &imagev1.Image{
		File: []*imagev1.ImageFile{
			protoImageFileAA,
			protoImageFileImport,
			protoImageFileAB,
			protoImageFileBA,
			protoImageFileBB,
			protoImageFileOutlandishDirectoryName,
		},
	}
	newImage, err = bufimage.NewImageForProto(protoImage)
	require.NoError(t, err)
	AssertImageFilesEqual(
		t,
		[]bufimage.ImageFile{
			NewImageFile(t, protoImageFileAA, nil, "", "a/a.proto", false, false, nil),
			NewImageFile(t, protoImageFileImport, nil, "", "import.proto", true, false, nil),
			NewImageFile(t, protoImageFileAB, nil, "", "a/b.proto", false, false, nil),
			NewImageFile(t, protoImageFileBA, nil, "", "b/a.proto", false, false, nil),
			NewImageFile(t, protoImageFileBB, nil, "", "b/b.proto", false, false, nil),
			NewImageFile(t, protoImageFileOutlandishDirectoryName, nil, "", "d/d.proto/d.proto", false, false, nil),
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
				testProtoImageFileToFileDescriptorProto(protoImageFileAA),
				testProtoImageFileToFileDescriptorProto(protoImageFileImport),
				testProtoImageFileToFileDescriptorProto(protoImageFileAB),
				testProtoImageFileToFileDescriptorProto(protoImageFileBA),
				testProtoImageFileToFileDescriptorProto(protoImageFileBB),
				testProtoImageFileToFileDescriptorProto(protoImageFileOutlandishDirectoryName),
			},
		},
		bufimage.ImageToFileDescriptorSet(image),
	)
	codeGeneratorRequest := &pluginpb.CodeGeneratorRequest{
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			testProtoImageFileToFileDescriptorProto(protoImageFileAA),
			testProtoImageFileToFileDescriptorProto(protoImageFileImport),
			testProtoImageFileToFileDescriptorProto(protoImageFileAB),
			testProtoImageFileToFileDescriptorProto(protoImageFileBA),
			testProtoImageFileToFileDescriptorProto(protoImageFileBB),
			testProtoImageFileToFileDescriptorProto(protoImageFileOutlandishDirectoryName),
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
		bufimage.ImageToCodeGeneratorRequest(image, "foo", nil, false),
	)
	newImage, err = bufimage.NewImageForCodeGeneratorRequest(codeGeneratorRequest)
	require.NoError(t, err)
	AssertImageFilesEqual(
		t,
		[]bufimage.ImageFile{
			NewImageFile(t, protoImageFileAA, nil, "", "a/a.proto", false, false, nil),
			NewImageFile(t, protoImageFileImport, nil, "", "import.proto", true, false, nil),
			NewImageFile(t, protoImageFileAB, nil, "", "a/b.proto", false, false, nil),
			NewImageFile(t, protoImageFileBA, nil, "", "b/a.proto", false, false, nil),
			NewImageFile(t, protoImageFileBB, nil, "", "b/b.proto", false, false, nil),
			NewImageFile(t, protoImageFileOutlandishDirectoryName, nil, "", "d/d.proto/d.proto", false, false, nil),
		},
		newImage.Files(),
	)
}

func testProtoImageFileToFileDescriptorProto(imageFile *imagev1.ImageFile) *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:       imageFile.Name,
		Dependency: imageFile.Dependency,
	}
}

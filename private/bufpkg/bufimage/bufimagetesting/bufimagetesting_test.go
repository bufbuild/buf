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

package bufimagetesting

import (
	"fmt"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	imagev1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/image/v1"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
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
		newImage, err := bufimage.ImageWithOnlyPathsAllowNotExist(image, []string{"a1.proto/a1.proto"}, nil)
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
		newImage, err := bufimage.ImageWithOnlyPathsAllowNotExist(image, []string{"a1.proto"}, nil)
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
	protoImageFileWellKnownTypeImport := NewProtoImageFileIsImport(
		t,
		"google/protobuf/timestamp.proto",
	)
	protoImageFileAA := NewProtoImageFile(
		t,
		"a/a.proto",
	)
	protoImageFileAB := NewProtoImageFile(
		t,
		"a/b.proto",
		"import.proto",
		"google/protobuf/timestamp.proto",
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
		"b/a.proto",
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
	fileWellKnownTypeImport := NewImageFile(
		t,
		protoImageFileWellKnownTypeImport,
		nil,
		"",
		"google/protobuf/timestamp.proto",
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
			fileWellKnownTypeImport,
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
			NewImageFile(t, protoImageFileWellKnownTypeImport, nil, "", "google/protobuf/timestamp.proto", true, false, nil),
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
		nil,
	)
	require.NoError(t, err)
	AssertImageFilesEqual(
		t,
		[]bufimage.ImageFile{
			NewImageFile(t, protoImageFileAA, nil, "", "foo/one/a/a.proto", true, false, nil),
			NewImageFile(t, protoImageFileImport, nil, "", "some/import/import.proto", true, false, nil),
			NewImageFile(t, protoImageFileWellKnownTypeImport, nil, "", "google/protobuf/timestamp.proto", true, false, nil),
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
		nil,
	)
	require.Error(t, err)

	newImage, err = bufimage.ImageWithOnlyPathsAllowNotExist(
		image,
		[]string{
			"b/a.proto",
			"a/b.proto",
			"foo.proto",
		},
		nil,
	)
	require.NoError(t, err)
	AssertImageFilesEqual(
		t,
		[]bufimage.ImageFile{
			NewImageFile(t, protoImageFileAA, nil, "", "foo/one/a/a.proto", true, false, nil),
			NewImageFile(t, protoImageFileImport, nil, "", "some/import/import.proto", true, false, nil),
			NewImageFile(t, protoImageFileWellKnownTypeImport, nil, "", "google/protobuf/timestamp.proto", true, false, nil),
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
		nil,
	)
	require.NoError(t, err)
	AssertImageFilesEqual(
		t,
		[]bufimage.ImageFile{
			NewImageFile(t, protoImageFileAA, nil, "", "foo/one/a/a.proto", false, false, nil),
			NewImageFile(t, protoImageFileImport, nil, "", "some/import/import.proto", true, false, nil),
			NewImageFile(t, protoImageFileWellKnownTypeImport, nil, "", "google/protobuf/timestamp.proto", true, false, nil),
			NewImageFile(t, protoImageFileAB, nil, "", "foo/one/a/b.proto", false, false, nil),
		},
		newImage.Files(),
	)
	newImage, err = bufimage.ImageWithOnlyPaths(
		image,
		[]string{
			"b",
		},
		nil,
	)
	require.NoError(t, err)
	AssertImageFilesEqual(
		t,
		[]bufimage.ImageFile{
			NewImageFile(t, protoImageFileAA, nil, "", "foo/one/a/a.proto", true, false, nil),
			NewImageFile(t, protoImageFileImport, nil, "", "some/import/import.proto", true, false, nil),
			NewImageFile(t, protoImageFileWellKnownTypeImport, nil, "", "google/protobuf/timestamp.proto", true, false, nil),
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
		nil,
	)
	require.NoError(t, err)
	AssertImageFilesEqual(
		t,
		[]bufimage.ImageFile{
			NewImageFile(t, protoImageFileAA, nil, "", "foo/one/a/a.proto", false, false, nil),
			NewImageFile(t, protoImageFileImport, nil, "", "some/import/import.proto", true, false, nil),
			NewImageFile(t, protoImageFileWellKnownTypeImport, nil, "", "google/protobuf/timestamp.proto", true, false, nil),
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
		nil,
	)
	require.Error(t, err)
	newImage, err = bufimage.ImageWithOnlyPathsAllowNotExist(
		image,
		[]string{
			"a",
			"b/a.proto",
			"c",
		},
		nil,
	)
	require.NoError(t, err)
	AssertImageFilesEqual(
		t,
		[]bufimage.ImageFile{
			NewImageFile(t, protoImageFileAA, nil, "", "foo/one/a/a.proto", false, false, nil),
			NewImageFile(t, protoImageFileImport, nil, "", "some/import/import.proto", true, false, nil),
			NewImageFile(t, protoImageFileWellKnownTypeImport, nil, "", "google/protobuf/timestamp.proto", true, false, nil),
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
		nil,
	)
	require.NoError(t, err)
	AssertImageFilesEqual(
		t,
		[]bufimage.ImageFile{
			NewImageFile(t, protoImageFileAA, nil, "", "foo/one/a/a.proto", false, false, nil),
			NewImageFile(t, protoImageFileImport, nil, "", "some/import/import.proto", true, false, nil),
			NewImageFile(t, protoImageFileWellKnownTypeImport, nil, "", "google/protobuf/timestamp.proto", true, false, nil),
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
		nil,
	)
	require.NoError(t, err)
	AssertImageFilesEqual(
		t,
		[]bufimage.ImageFile{
			NewImageFile(t, protoImageFileAA, nil, "", "foo/one/a/a.proto", false, false, nil),
			NewImageFile(t, protoImageFileImport, nil, "", "some/import/import.proto", true, false, nil),
			NewImageFile(t, protoImageFileWellKnownTypeImport, nil, "", "google/protobuf/timestamp.proto", true, false, nil),
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
			protoImageFileWellKnownTypeImport,
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
			NewImageFile(t, protoImageFileWellKnownTypeImport, nil, "", "google/protobuf/timestamp.proto", true, false, nil),
			NewImageFile(t, protoImageFileAB, nil, "", "a/b.proto", false, false, nil),
			NewImageFile(t, protoImageFileBA, nil, "", "b/a.proto", false, false, nil),
			NewImageFile(t, protoImageFileBB, nil, "", "b/b.proto", false, false, nil),
			NewImageFile(t, protoImageFileOutlandishDirectoryName, nil, "", "d/d.proto/d.proto", false, false, nil),
		},
		newImage.Files(),
	)
	diff := cmp.Diff(protoImage, bufimage.ImageToProtoImage(newImage), protocmp.Transform())
	require.Equal(t, "", diff)
	fileDescriptorSet := &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			testProtoImageFileToFileDescriptorProto(protoImageFileAA),
			testProtoImageFileToFileDescriptorProto(protoImageFileImport),
			testProtoImageFileToFileDescriptorProto(protoImageFileWellKnownTypeImport),
			testProtoImageFileToFileDescriptorProto(protoImageFileAB),
			testProtoImageFileToFileDescriptorProto(protoImageFileBA),
			testProtoImageFileToFileDescriptorProto(protoImageFileBB),
			testProtoImageFileToFileDescriptorProto(protoImageFileOutlandishDirectoryName),
		},
	}
	diff = cmp.Diff(fileDescriptorSet, bufimage.ImageToFileDescriptorSet(image), protocmp.Transform())
	require.Equal(t, "", diff)
	codeGeneratorRequest := &pluginpb.CodeGeneratorRequest{
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			testProtoImageFileToFileDescriptorProto(protoImageFileAA),
			testProtoImageFileToFileDescriptorProto(protoImageFileImport),
			testProtoImageFileToFileDescriptorProto(protoImageFileWellKnownTypeImport),
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
	diff = cmp.Diff(
		codeGeneratorRequest,
		bufimage.ImageToCodeGeneratorRequest(image, "foo", nil, false, false),
		protocmp.Transform(),
	)
	require.Equal(t, "", diff)

	// verify that includeWellKnownTypes is a no-op if includeImports is false
	diff = cmp.Diff(
		codeGeneratorRequest,
		bufimage.ImageToCodeGeneratorRequest(image, "foo", nil, false, true),
		protocmp.Transform(),
	)
	require.Equal(t, "", diff)

	codeGeneratorRequestIncludeImports := &pluginpb.CodeGeneratorRequest{
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			testProtoImageFileToFileDescriptorProto(protoImageFileAA),
			testProtoImageFileToFileDescriptorProto(protoImageFileImport),
			testProtoImageFileToFileDescriptorProto(protoImageFileWellKnownTypeImport),
			testProtoImageFileToFileDescriptorProto(protoImageFileAB),
			testProtoImageFileToFileDescriptorProto(protoImageFileBA),
			testProtoImageFileToFileDescriptorProto(protoImageFileBB),
			testProtoImageFileToFileDescriptorProto(protoImageFileOutlandishDirectoryName),
		},
		Parameter: proto.String("foo"),
		FileToGenerate: []string{
			"a/a.proto",
			"import.proto",
			// no WKT
			"a/b.proto",
			"b/a.proto",
			"b/b.proto",
			"d/d.proto/d.proto",
		},
	}
	diff = cmp.Diff(
		codeGeneratorRequestIncludeImports,
		bufimage.ImageToCodeGeneratorRequest(image, "foo", nil, true, false),
		protocmp.Transform(),
	)
	require.Equal(t, "", diff)
	newImage, err = bufimage.NewImageForCodeGeneratorRequest(codeGeneratorRequest)
	require.NoError(t, err)
	AssertImageFilesEqual(
		t,
		[]bufimage.ImageFile{
			NewImageFile(t, protoImageFileAA, nil, "", "a/a.proto", false, false, nil),
			NewImageFile(t, protoImageFileImport, nil, "", "import.proto", true, false, nil),
			NewImageFile(t, protoImageFileWellKnownTypeImport, nil, "", "google/protobuf/timestamp.proto", true, false, nil),
			NewImageFile(t, protoImageFileAB, nil, "", "a/b.proto", false, false, nil),
			NewImageFile(t, protoImageFileBA, nil, "", "b/a.proto", false, false, nil),
			NewImageFile(t, protoImageFileBB, nil, "", "b/b.proto", false, false, nil),
			NewImageFile(t, protoImageFileOutlandishDirectoryName, nil, "", "d/d.proto/d.proto", false, false, nil),
		},
		newImage.Files(),
	)
	codeGeneratorRequestIncludeImportsAndWellKnownTypes := &pluginpb.CodeGeneratorRequest{
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			testProtoImageFileToFileDescriptorProto(protoImageFileAA),
			testProtoImageFileToFileDescriptorProto(protoImageFileImport),
			testProtoImageFileToFileDescriptorProto(protoImageFileWellKnownTypeImport),
			testProtoImageFileToFileDescriptorProto(protoImageFileAB),
			testProtoImageFileToFileDescriptorProto(protoImageFileBA),
			testProtoImageFileToFileDescriptorProto(protoImageFileBB),
			testProtoImageFileToFileDescriptorProto(protoImageFileOutlandishDirectoryName),
		},
		Parameter: proto.String("foo"),
		FileToGenerate: []string{
			"a/a.proto",
			"import.proto",
			"google/protobuf/timestamp.proto",
			"a/b.proto",
			"b/a.proto",
			"b/b.proto",
			"d/d.proto/d.proto",
		},
	}
	diff = cmp.Diff(
		codeGeneratorRequestIncludeImportsAndWellKnownTypes,
		bufimage.ImageToCodeGeneratorRequest(image, "foo", nil, true, true),
		protocmp.Transform(),
	)
	require.Equal(t, "", diff)
	// imagesByDir and multiple Image tests
	imagesByDir, err := bufimage.ImageByDir(image)
	require.NoError(t, err)
	require.Equal(t, 3, len(imagesByDir))
	AssertImageFilesEqual(
		t,
		[]bufimage.ImageFile{
			NewImageFile(t, protoImageFileAA, nil, "", "foo/one/a/a.proto", false, false, nil),
			NewImageFile(t, protoImageFileImport, nil, "", "some/import/import.proto", true, false, nil),
			NewImageFile(t, protoImageFileWellKnownTypeImport, nil, "", "google/protobuf/timestamp.proto", true, false, nil),
			NewImageFile(t, protoImageFileAB, nil, "", "foo/one/a/b.proto", false, false, nil),
		},
		imagesByDir[0].Files(),
	)
	AssertImageFilesEqual(
		t,
		[]bufimage.ImageFile{
			NewImageFile(t, protoImageFileAA, nil, "", "foo/one/a/a.proto", true, false, nil),
			NewImageFile(t, protoImageFileImport, nil, "", "some/import/import.proto", true, false, nil),
			NewImageFile(t, protoImageFileWellKnownTypeImport, nil, "", "google/protobuf/timestamp.proto", true, false, nil),
			NewImageFile(t, protoImageFileAB, nil, "", "foo/one/a/b.proto", true, false, nil),
			NewImageFile(t, protoImageFileBA, nil, "", "foo/two/b/a.proto", false, false, nil),
			NewImageFile(t, protoImageFileBB, nil, "", "foo/two/b/b.proto", false, false, nil),
		},
		imagesByDir[1].Files(),
	)
	AssertImageFilesEqual(
		t,
		[]bufimage.ImageFile{
			NewImageFile(t, protoImageFileImport, nil, "", "some/import/import.proto", true, false, nil),
			NewImageFile(t, protoImageFileOutlandishDirectoryName, nil, "", "foo/three/d/d.proto/d.proto", false, false, nil),
		},
		imagesByDir[2].Files(),
	)
	codeGeneratorRequests := []*pluginpb.CodeGeneratorRequest{
		{
			ProtoFile: []*descriptorpb.FileDescriptorProto{
				testProtoImageFileToFileDescriptorProto(protoImageFileAA),
				testProtoImageFileToFileDescriptorProto(protoImageFileImport),
				testProtoImageFileToFileDescriptorProto(protoImageFileWellKnownTypeImport),
				testProtoImageFileToFileDescriptorProto(protoImageFileAB),
			},
			Parameter: proto.String("foo"),
			FileToGenerate: []string{
				"a/a.proto",
				"a/b.proto",
			},
		},
		{
			ProtoFile: []*descriptorpb.FileDescriptorProto{
				testProtoImageFileToFileDescriptorProto(protoImageFileAA),
				testProtoImageFileToFileDescriptorProto(protoImageFileImport),
				testProtoImageFileToFileDescriptorProto(protoImageFileWellKnownTypeImport),
				testProtoImageFileToFileDescriptorProto(protoImageFileAB),
				testProtoImageFileToFileDescriptorProto(protoImageFileBA),
				testProtoImageFileToFileDescriptorProto(protoImageFileBB),
			},
			Parameter: proto.String("foo"),
			FileToGenerate: []string{
				"b/a.proto",
				"b/b.proto",
			},
		},
		{
			ProtoFile: []*descriptorpb.FileDescriptorProto{
				testProtoImageFileToFileDescriptorProto(protoImageFileImport),
				testProtoImageFileToFileDescriptorProto(protoImageFileOutlandishDirectoryName),
			},
			Parameter: proto.String("foo"),
			FileToGenerate: []string{
				"d/d.proto/d.proto",
			},
		},
	}
	requestsFromImages := bufimage.ImagesToCodeGeneratorRequests(imagesByDir, "foo", nil, false, false)
	require.Equal(t, len(codeGeneratorRequests), len(requestsFromImages))
	for i := range codeGeneratorRequests {
		diff = cmp.Diff(codeGeneratorRequests[i], requestsFromImages[i], protocmp.Transform())
		require.Equal(t, "", diff)
	}
	codeGeneratorRequestsIncludeImports := []*pluginpb.CodeGeneratorRequest{
		{
			ProtoFile: []*descriptorpb.FileDescriptorProto{
				testProtoImageFileToFileDescriptorProto(protoImageFileAA),
				testProtoImageFileToFileDescriptorProto(protoImageFileImport),
				testProtoImageFileToFileDescriptorProto(protoImageFileWellKnownTypeImport),
				testProtoImageFileToFileDescriptorProto(protoImageFileAB),
			},
			Parameter: proto.String("foo"),
			FileToGenerate: []string{
				"a/a.proto",
				"import.proto",
				"a/b.proto",
			},
		},
		{
			ProtoFile: []*descriptorpb.FileDescriptorProto{
				testProtoImageFileToFileDescriptorProto(protoImageFileAA),
				testProtoImageFileToFileDescriptorProto(protoImageFileImport),
				testProtoImageFileToFileDescriptorProto(protoImageFileWellKnownTypeImport),
				testProtoImageFileToFileDescriptorProto(protoImageFileAB),
				testProtoImageFileToFileDescriptorProto(protoImageFileBA),
				testProtoImageFileToFileDescriptorProto(protoImageFileBB),
			},
			Parameter: proto.String("foo"),
			FileToGenerate: []string{
				"b/a.proto",
				"b/b.proto",
			},
		},
		{
			ProtoFile: []*descriptorpb.FileDescriptorProto{
				testProtoImageFileToFileDescriptorProto(protoImageFileImport),
				testProtoImageFileToFileDescriptorProto(protoImageFileOutlandishDirectoryName),
			},
			Parameter: proto.String("foo"),
			FileToGenerate: []string{
				"d/d.proto/d.proto",
			},
		},
	}
	requestsFromImages = bufimage.ImagesToCodeGeneratorRequests(imagesByDir, "foo", nil, true, false)
	require.Equal(t, len(codeGeneratorRequestsIncludeImports), len(requestsFromImages))
	for i := range codeGeneratorRequestsIncludeImports {
		diff = cmp.Diff(codeGeneratorRequestsIncludeImports[i], requestsFromImages[i], protocmp.Transform())
		require.Equal(t, "", diff)
	}
}

func testProtoImageFileToFileDescriptorProto(imageFile *imagev1.ImageFile) *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:       imageFile.Name,
		Dependency: imageFile.Dependency,
	}
}

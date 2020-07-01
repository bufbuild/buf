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

package bufcoretesting

import (
	"testing"

	"github.com/bufbuild/buf/internal/buf/bufcore"
	imagev1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/image/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

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
	// imports a/a.proto
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
	fileOneBA := NewImageFile(
		t,
		fileDescriptorProtoBA,
		"foo/two/b/a.proto",
		false,
	)
	fileOneBB := NewImageFile(
		t,
		fileDescriptorProtoBB,
		"foo/two/b/b.proto",
		false,
	)

	image, err := bufcore.NewImage(
		[]bufcore.ImageFile{
			fileOneAA,
			fileImport,
			fileOneAB,
			fileOneBA,
			fileOneBB,
		},
	)
	require.NoError(t, err)
	AssertImageFilesEqual(
		t,
		[]bufcore.ImageFile{
			NewImageFile(t, fileDescriptorProtoAA, "foo/one/a/a.proto", false),
			NewImageFile(t, fileDescriptorProtoImport, "some/import/import.proto", true),
			NewImageFile(t, fileDescriptorProtoAB, "foo/one/a/b.proto", false),
			NewImageFile(t, fileDescriptorProtoBA, "foo/two/b/a.proto", false),
			NewImageFile(t, fileDescriptorProtoBB, "foo/two/b/b.proto", false),
		},
		image.Files(),
	)
	require.NotNil(t, image.GetFile("a/a.proto"))
	require.Nil(t, image.GetFile("one/a/a.proto"))
	AssertImageFilesEqual(
		t,
		[]bufcore.ImageFile{
			NewImageFile(t, fileDescriptorProtoAA, "foo/one/a/a.proto", false),
			NewImageFile(t, fileDescriptorProtoAB, "foo/one/a/b.proto", false),
			NewImageFile(t, fileDescriptorProtoBA, "foo/two/b/a.proto", false),
			NewImageFile(t, fileDescriptorProtoBB, "foo/two/b/b.proto", false),
		},
		bufcore.ImageWithoutImports(image).Files(),
	)

	image, err = bufcore.ImageWithOnlyPaths(
		image,
		[]string{
			"b/a.proto",
			"a/b.proto",
		},
	)
	require.NoError(t, err)
	AssertImageFilesEqual(
		t,
		[]bufcore.ImageFile{
			NewImageFile(t, fileDescriptorProtoAA, "foo/one/a/a.proto", true),
			NewImageFile(t, fileDescriptorProtoImport, "some/import/import.proto", true),
			NewImageFile(t, fileDescriptorProtoAB, "foo/one/a/b.proto", false),
			NewImageFile(t, fileDescriptorProtoBA, "foo/two/b/a.proto", false),
		},
		image.Files(),
	)

	_, err = bufcore.ImageWithOnlyPaths(
		image,
		[]string{
			"b/a.proto",
			"a/b.proto",
			"foo.proto",
		},
	)
	require.Error(t, err)

	image, err = bufcore.ImageWithOnlyPathsAllowNotExist(
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
		[]bufcore.ImageFile{
			NewImageFile(t, fileDescriptorProtoAA, "foo/one/a/a.proto", true),
			NewImageFile(t, fileDescriptorProtoImport, "some/import/import.proto", true),
			NewImageFile(t, fileDescriptorProtoAB, "foo/one/a/b.proto", false),
			NewImageFile(t, fileDescriptorProtoBA, "foo/two/b/a.proto", false),
		},
		image.Files(),
	)

	protoImage := &imagev1.Image{
		File: []*descriptorpb.FileDescriptorProto{
			fileDescriptorProtoAA,
			fileDescriptorProtoImport,
			fileDescriptorProtoAB,
			fileDescriptorProtoBA,
			fileDescriptorProtoBB,
		},
		BufbuildImageExtension: &imagev1.ImageExtension{
			ImageImportRefs: []*imagev1.ImageImportRef{
				{
					FileIndex: proto.Uint32(1),
				},
			},
		},
	}
	image, err = bufcore.NewImageForProto(protoImage)
	require.NoError(t, err)
	AssertImageFilesEqual(
		t,
		[]bufcore.ImageFile{
			NewImageFile(t, fileDescriptorProtoAA, "a/a.proto", false),
			NewImageFile(t, fileDescriptorProtoImport, "import.proto", true),
			NewImageFile(t, fileDescriptorProtoAB, "a/b.proto", false),
			NewImageFile(t, fileDescriptorProtoBA, "b/a.proto", false),
			NewImageFile(t, fileDescriptorProtoBB, "b/b.proto", false),
		},
		image.Files(),
	)
	require.Equal(
		t,
		protoImage,
		bufcore.ImageToProtoImage(image),
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
			},
		},
		bufcore.ImageToFileDescriptorSet(image),
	)
	codeGeneratorRequest := &pluginpb.CodeGeneratorRequest{
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			fileDescriptorProtoAA,
			fileDescriptorProtoImport,
			fileDescriptorProtoAB,
			fileDescriptorProtoBA,
			fileDescriptorProtoBB,
		},
		Parameter: proto.String("foo"),
		FileToGenerate: []string{
			"a/a.proto",
			"a/b.proto",
			"b/a.proto",
			"b/b.proto",
		},
	}
	require.Equal(
		t,
		codeGeneratorRequest,
		bufcore.ImageToCodeGeneratorRequest(image, "foo"),
	)
	image, err = bufcore.NewImageForCodeGeneratorRequest(codeGeneratorRequest)
	require.NoError(t, err)
	AssertImageFilesEqual(
		t,
		[]bufcore.ImageFile{
			NewImageFile(t, fileDescriptorProtoAA, "a/a.proto", false),
			NewImageFile(t, fileDescriptorProtoImport, "import.proto", true),
			NewImageFile(t, fileDescriptorProtoAB, "a/b.proto", false),
			NewImageFile(t, fileDescriptorProtoBA, "b/a.proto", false),
			NewImageFile(t, fileDescriptorProtoBB, "b/b.proto", false),
		},
		image.Files(),
	)
}

// Copyright 2020 Buf Technologies Inc.
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
	"testing"

	"github.com/bufbuild/buf/internal/buf/bufimage"
	"github.com/bufbuild/buf/internal/buf/bufpath"
	imagev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/image/v1beta1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestBasic(t *testing.T) {
	t.Parallel()

	importPathResolver := bufpath.NewDirPathResolver("some/import")
	pathResolver := bufpath.NewDirPathResolver("foo")

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

	fileImport := NewFile(
		t,
		fileDescriptorProtoImport,
		".",
		importPathResolver,
		true,
	)
	fileOneAA := NewFile(
		t,
		fileDescriptorProtoAA,
		"one",
		pathResolver,
		false,
	)
	fileOneAB := NewFile(
		t,
		fileDescriptorProtoAB,
		"one",
		pathResolver,
		false,
	)
	fileOneBA := NewFile(
		t,
		fileDescriptorProtoBA,
		"two",
		pathResolver,
		false,
	)
	fileOneBB := NewFile(
		t,
		fileDescriptorProtoBB,
		"two",
		pathResolver,
		false,
	)

	image, err := bufimage.NewImage(
		[]bufimage.File{
			fileOneAA,
			fileImport,
			fileOneAB,
			fileOneBA,
			fileOneBB,
		},
	)
	require.NoError(t, err)
	AssertFilesEqual(
		t,
		[]bufimage.File{
			NewDirectFile(t, fileDescriptorProtoAA, "one", "foo/one/a/a.proto", false),
			NewDirectFile(t, fileDescriptorProtoImport, ".", "some/import/import.proto", true),
			NewDirectFile(t, fileDescriptorProtoAB, "one", "foo/one/a/b.proto", false),
			NewDirectFile(t, fileDescriptorProtoBA, "two", "foo/two/b/a.proto", false),
			NewDirectFile(t, fileDescriptorProtoBB, "two", "foo/two/b/b.proto", false),
		},
		image.Files(),
	)
	require.NotNil(t, image.GetFile("a/a.proto"))
	require.Nil(t, image.GetFile("one/a/a.proto"))
	AssertFilesEqual(
		t,
		[]bufimage.File{
			NewDirectFile(t, fileDescriptorProtoAA, "one", "foo/one/a/a.proto", false),
			NewDirectFile(t, fileDescriptorProtoAB, "one", "foo/one/a/b.proto", false),
			NewDirectFile(t, fileDescriptorProtoBA, "two", "foo/two/b/a.proto", false),
			NewDirectFile(t, fileDescriptorProtoBB, "two", "foo/two/b/b.proto", false),
		},
		bufimage.ImageWithoutImports(image).Files(),
	)

	image, err = bufimage.ImageWithOnlyRootRelFilePaths(
		image,
		[]string{
			"b/a.proto",
			"a/b.proto",
		},
	)
	require.NoError(t, err)
	AssertFilesEqual(
		t,
		[]bufimage.File{
			NewDirectFile(t, fileDescriptorProtoAA, "one", "foo/one/a/a.proto", true),
			NewDirectFile(t, fileDescriptorProtoImport, ".", "some/import/import.proto", true),
			NewDirectFile(t, fileDescriptorProtoAB, "one", "foo/one/a/b.proto", false),
			NewDirectFile(t, fileDescriptorProtoBA, "two", "foo/two/b/a.proto", false),
		},
		image.Files(),
	)

	_, err = bufimage.ImageWithOnlyRootRelFilePaths(
		image,
		[]string{
			"b/a.proto",
			"a/b.proto",
			"foo.proto",
		},
	)
	require.Error(t, err)

	image, err = bufimage.ImageWithOnlyRootRelFilePathsAllowNotExist(
		image,
		[]string{
			"b/a.proto",
			"a/b.proto",
			"foo.proto",
		},
	)
	require.NoError(t, err)
	AssertFilesEqual(
		t,
		[]bufimage.File{
			NewDirectFile(t, fileDescriptorProtoAA, "one", "foo/one/a/a.proto", true),
			NewDirectFile(t, fileDescriptorProtoImport, ".", "some/import/import.proto", true),
			NewDirectFile(t, fileDescriptorProtoAB, "one", "foo/one/a/b.proto", false),
			NewDirectFile(t, fileDescriptorProtoBA, "two", "foo/two/b/a.proto", false),
		},
		image.Files(),
	)

	protoImage := &imagev1beta1.Image{
		File: []*descriptorpb.FileDescriptorProto{
			fileDescriptorProtoAA,
			fileDescriptorProtoImport,
			fileDescriptorProtoAB,
			fileDescriptorProtoBA,
			fileDescriptorProtoBB,
		},
		BufbuildImageExtension: &imagev1beta1.ImageExtension{
			ImageImportRefs: []*imagev1beta1.ImageImportRef{
				{
					FileIndex: proto.Uint32(1),
				},
			},
		},
	}
	image, err = bufimage.NewImageForProto(protoImage)
	require.NoError(t, err)
	AssertFilesEqual(
		t,
		[]bufimage.File{
			NewDirectFile(t, fileDescriptorProtoAA, ".", "a/a.proto", false),
			NewDirectFile(t, fileDescriptorProtoImport, ".", "import.proto", true),
			NewDirectFile(t, fileDescriptorProtoAB, ".", "a/b.proto", false),
			NewDirectFile(t, fileDescriptorProtoBA, ".", "b/a.proto", false),
			NewDirectFile(t, fileDescriptorProtoBB, ".", "b/b.proto", false),
		},
		image.Files(),
	)
	require.Equal(
		t,
		protoImage,
		bufimage.ImageToProtoImage(image),
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
		bufimage.ImageToFileDescriptorSet(image),
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
		bufimage.ImageToCodeGeneratorRequest(image, "foo"),
	)
	image, err = bufimage.NewImageForCodeGeneratorRequest(codeGeneratorRequest)
	require.NoError(t, err)
	AssertFilesEqual(
		t,
		[]bufimage.File{
			NewDirectFile(t, fileDescriptorProtoAA, ".", "a/a.proto", false),
			NewDirectFile(t, fileDescriptorProtoImport, ".", "import.proto", true),
			NewDirectFile(t, fileDescriptorProtoAB, ".", "a/b.proto", false),
			NewDirectFile(t, fileDescriptorProtoBA, ".", "b/a.proto", false),
			NewDirectFile(t, fileDescriptorProtoBB, ".", "b/b.proto", false),
		},
		image.Files(),
	)
}

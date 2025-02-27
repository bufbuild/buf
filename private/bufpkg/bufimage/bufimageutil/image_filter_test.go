// Copyright 2020-2025 Buf Technologies, Inc.
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

package bufimageutil

/*import (
	"bytes"
	"context"
	"sort"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/slogtestext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/jhump/protoreflect/v2/protoprint"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/txtar"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestExcludeOptions(t *testing.T) {
	t.Parallel()

	t.Run("NoneExcluded", func(t *testing.T) {
		t.Parallel()
		testFilterOptions(
			t, "testdata/excludeoptions",
		)
	})
	t.Run("ExcludeMessage", func(t *testing.T) {
		t.Parallel()
		testFilterOptions(
			t, "testdata/excludeoptions", WithExcludeOptions(
				"message_foo", "message_bar", "message_baz",
			),
		)
	})
	t.Run("ExcludeFoo", func(t *testing.T) {
		t.Parallel()
		testFilterOptions(
			t, "testdata/excludeoptions", WithExcludeOptions(
				"message_foo",
				"field_foo",
				"oneof_foo",
				"enum_foo",
				"enum_value_foo",
				"service_foo",
				"method_foo",
				"UsedOption.file_foo",
			),
		)
	})
	t.Run("OnlyFile", func(t *testing.T) {
		t.Parallel()
		testFilterOptions(
			t, "testdata/excludeoptions", WithExcludeOptions(
				"message_foo", "message_bar", "message_baz",
				"field_foo", "field_bar", "field_baz",
				"oneof_foo", "oneof_bar", "oneof_baz",
				"enum_foo", "enum_bar", "enum_baz",
				"enum_value_foo", "enum_value_bar", "enum_value_baz",
				"service_foo", "service_bar", "service_baz",
				"method_foo", "method_bar", "method_baz",
			),
		)
	})
	t.Run("OnlyOneOf", func(t *testing.T) {
		t.Parallel()
		testFilterOptions(
			t, "testdata/excludeoptions", WithExcludeOptions(
				"message_foo", "message_bar", "message_baz",
				"field_foo", "field_bar", "field_baz",
				"enum_foo", "enum_bar", "enum_baz",
				"enum_value_foo", "enum_value_bar", "enum_value_baz",
				"service_foo", "service_bar", "service_baz",
				"method_foo", "method_bar", "method_baz",
				"UsedOption.file_foo", "UsedOption.file_bar", "UsedOption.file_baz",
			),
		)
	})
	t.Run("OnlyEnumValue", func(t *testing.T) {
		t.Parallel()
		testFilterOptions(
			t, "testdata/excludeoptions", WithExcludeOptions(
				"message_foo", "message_bar", "message_baz",
				"field_foo", "field_bar", "field_baz",
				"oneof_foo", "oneof_bar", "oneof_baz",
				"enum_foo", "enum_bar", "enum_baz",
				"service_foo", "service_bar", "service_baz",
				"method_foo", "method_bar", "method_baz",
				"UsedOption.file_foo", "UsedOption.file_bar", "UsedOption.file_baz",
			),
		)
	})
	t.Run("ExcludeAll", func(t *testing.T) {
		t.Parallel()
		testFilterOptions(
			t, "testdata/excludeoptions", WithExcludeOptions(
				"message_foo", "message_bar", "message_baz",
				"field_foo", "field_bar", "field_baz",
				"oneof_foo", "oneof_bar", "oneof_baz",
				"enum_foo", "enum_bar", "enum_baz",
				"enum_value_foo", "enum_value_bar", "enum_value_baz",
				"service_foo", "service_bar", "service_baz",
				"method_foo", "method_bar", "method_baz",
				"UsedOption.file_foo", "UsedOption.file_bar", "UsedOption.file_baz",
			),
		)
	})
}

func TestExcludeOptionImports(t *testing.T) {
	t.Parallel()

	// This checks that when excluding options the imports are correctly dropped.
	// For this case when both options are removed only a.proto should be left.
	testdataDir := "testdata/excludeoptionimports"
	bucket, err := storageos.NewProvider().NewReadWriteBucket(testdataDir)
	require.NoError(t, err)
	testModuleData := []bufmoduletesting.ModuleData{
		{
			Bucket: storage.FilterReadBucket(bucket, storage.MatchPathEqual("a.proto")),
		},
		{
			Bucket:      storage.FilterReadBucket(bucket, storage.MatchPathEqual("options.proto")),
			NotTargeted: true,
		},
	}
	moduleSet, err := bufmoduletesting.NewModuleSet(testModuleData...)
	require.NoError(t, err)

	// Safe to filter the image concurrently as its not being modified.
	image, err := bufimage.BuildImage(
		context.Background(),
		slogtestext.NewLogger(t),
		bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(moduleSet),
		bufimage.WithExcludeSourceCodeInfo(),
	)
	require.NoError(t, err)

	t.Run("NoneExcluded", func(t *testing.T) {
		t.Parallel()
		testFilterOptionsForImage(t, bucket, image)
	})
	t.Run("ExcludeFoo", func(t *testing.T) {
		t.Parallel()
		testFilterOptionsForImage(t, bucket, image, WithExcludeOptions("message_foo"))
	})
	t.Run("ExcludeFooBar", func(t *testing.T) {
		t.Parallel()
		testFilterOptionsForImage(t, bucket, image, WithExcludeOptions("message_foo", "message_bar"))
	})
	t.Run("ExcludeBar", func(t *testing.T) {
		t.Parallel()
		testFilterOptionsForImage(t, bucket, image, WithExcludeOptions("message_bar"))
	})
}

func TestFilterTypes(t *testing.T) {
	t.Parallel()

	testdataDir := "testdata/nesting"
	bucket, err := storageos.NewProvider().NewReadWriteBucket(testdataDir)
	require.NoError(t, err)
	testModuleData := []bufmoduletesting.ModuleData{
		{
			Bucket: storage.FilterReadBucket(bucket, storage.MatchPathEqual("a.proto")),
		},
		{
			Bucket:      storage.FilterReadBucket(bucket, storage.MatchPathEqual("options.proto")),
			NotTargeted: true,
		},
	}
	moduleSet, err := bufmoduletesting.NewModuleSet(testModuleData...)
	require.NoError(t, err)

	// Safe to filter the image concurrently as its not being modified.
	image, err := bufimage.BuildImage(
		context.Background(),
		slogtestext.NewLogger(t),
		bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(moduleSet),
		bufimage.WithExcludeSourceCodeInfo(),
	)
	require.NoError(t, err)

	t.Run("ExcludeBar", func(t *testing.T) {
		t.Parallel()
		testFilterOptionsForImage(t, bucket, image, WithExcludeTypes("pkg.Bar"))
	})
	t.Run("IncludeBar", func(t *testing.T) {
		t.Parallel()
		testFilterOptionsForImage(t, bucket, image, WithIncludeTypes("pkg.Bar"))
	})

}

func testFilterOptions(t *testing.T, testdataDir string, options ...ImageFilterOption) {
	bucket, err := storageos.NewProvider().NewReadWriteBucket(testdataDir)
	require.NoError(t, err)
	testFilterOptionsForModuleData(t, bucket, nil, options...)
}

func testFilterOptionsForModuleData(t *testing.T, bucket storage.ReadWriteBucket, moduleData []bufmoduletesting.ModuleData, options ...ImageFilterOption) {
	ctx := context.Background()
	if len(moduleData) == 0 {
		moduleData = append(moduleData, bufmoduletesting.ModuleData{
			Bucket: bucket,
		})
	}
	moduleSet, err := bufmoduletesting.NewModuleSet(moduleData...)
	require.NoError(t, err)

	image, err := bufimage.BuildImage(
		ctx,
		slogtestext.NewLogger(t),
		bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(moduleSet),
		bufimage.WithExcludeSourceCodeInfo(),
	)
	require.NoError(t, err)

	testFilterOptionsForImage(t, bucket, image, options...)
}

func testFilterOptionsForImage(t *testing.T, bucket storage.ReadWriteBucket, image bufimage.Image, options ...ImageFilterOption) {
	ctx := context.Background()
	filteredImage, err := FilterImage(image, options...)
	require.NoError(t, err)

	files, err := protodesc.NewFiles(&descriptorpb.FileDescriptorSet{
		File: slicesext.Map(filteredImage.Files(), func(imageFile bufimage.ImageFile) *descriptorpb.FileDescriptorProto {
			return imageFile.FileDescriptorProto()
		}),
	})
	require.NoError(t, err)

	archive := &txtar.Archive{}
	printer := protoprint.Printer{
		SortElements: true,
		Compact:      true,
	}
	files.RangeFiles(func(fileDescriptor protoreflect.FileDescriptor) bool {
		fileBuilder := &bytes.Buffer{}
		require.NoError(t, printer.PrintProtoFile(fileDescriptor, fileBuilder), "expected no error while printing %q", fileDescriptor.Path())
		archive.Files = append(
			archive.Files,
			txtar.File{
				Name: fileDescriptor.Path(),
				Data: fileBuilder.Bytes(),
			},
		)
		return true
	})
	sort.SliceStable(archive.Files, func(i, j int) bool {
		return archive.Files[i].Name < archive.Files[j].Name
	})
	generated := txtar.Format(archive)
	expectedFile := t.Name() + ".txtar"
	checkExpectation(t, ctx, generated, bucket, expectedFile)
}*/

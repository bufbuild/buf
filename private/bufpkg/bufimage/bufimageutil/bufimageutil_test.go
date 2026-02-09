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

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"sort"
	"testing"

	"buf.build/go/app/appext"
	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/slogtestext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/google/uuid"
	"github.com/jhump/protoreflect/v2/protoprint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/txtar"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// IF YOU HAVE ANY FAILING TESTS IN HERE, ESPECIALLY AFTER A PROTOC UPGRADE,
// RUN THE FOLLOWING:
// make bufimageutilupdateexpectations

var shouldUpdateExpectations = os.Getenv("BUFBUILD_BUF_BUFIMAGEUTIL_SHOULD_UPDATE_EXPECTATIONS")

func TestTypes(t *testing.T) {
	t.Parallel()
	t.Run("message", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", "pkg.Foo.txtar", WithIncludeTypes("pkg.Foo"))
	})
	t.Run("enum", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", "pkg.FooEnum.txtar", WithIncludeTypes("pkg.FooEnum"))
	})
	t.Run("service", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", "pkg.FooService.txtar", WithIncludeTypes("pkg.FooService"))
	})
	t.Run("method", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", "pkg.FooService.Do.txtar", WithIncludeTypes("pkg.FooService.Do"))
	})
	t.Run("all", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", "all.txtar", WithIncludeTypes("pkg.Foo", "pkg.FooEnum", "pkg.FooService"))
	})
	t.Run("exclude-options", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", "all-exclude-options.txtar", WithIncludeTypes("pkg.Foo", "pkg.FooEnum", "pkg.FooService"), WithExcludeCustomOptions())
	})
	t.Run("files", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", "Files.txtar", WithIncludeTypes("Files"))
	})
	t.Run("all-with-files", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", "all-with-Files.txtar", WithIncludeTypes("pkg.Foo", "pkg.FooEnum", "pkg.FooService", "Files"))
	})

	t.Run("exclude-message", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", "pkg.Foo.exclude.txtar", WithExcludeTypes("pkg.Foo"))
	})
	t.Run("exclude-enum", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", "pkg.FooEnum.exclude.txtar", WithExcludeTypes("pkg.FooEnum"))
	})
	t.Run("exclude-service", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", "pkg.FooService.exclude.txtar", WithExcludeTypes("pkg.FooService"))
	})
	t.Run("exclude-method", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", "pkg.FooService.Do.exclude.txtar", WithExcludeTypes("pkg.FooService.Do"))
	})
	t.Run("exclude-package", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", "pkg.Pkg.exclude.txtar", WithExcludeTypes("pkg"))
	})
	t.Run("exclude-all", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", "all.exclude.txtar", WithExcludeTypes("pkg.Foo", "pkg.FooEnum", "pkg.FooService"))
	})

	t.Run("mixed-service-method", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", "pkg.FooService.mixed.txtar", WithIncludeTypes("pkg.FooService"), WithExcludeTypes("pkg.FooService.Do"))
	})
	t.Run("include-service-exclude-method-types", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", "pkg.FooService.exclude-method-types.txtar", WithIncludeTypes("pkg.FooService"), WithExcludeTypes("pkg.Empty"))
	})
	t.Run("include-method-exclude-method-types", func(t *testing.T) {
		t.Parallel()
		_, image, err := getImage(context.Background(), slogtestext.NewLogger(t), "testdata/options", bufimage.WithExcludeSourceCodeInfo())
		require.NoError(t, err)
		_, err = FilterImage(image, WithIncludeTypes("pkg.FooService", "pkg.FooService.Do"), WithExcludeTypes("pkg.Empty"))
		require.Error(t, err)
		assert.ErrorContains(t, err, "cannot include method \"pkg.FooService.Do\"")
	})

	t.Run("include-extension-exclude-extendee", func(t *testing.T) {
		t.Parallel()
		_, image, err := getImage(context.Background(), slogtestext.NewLogger(t), "testdata/options", bufimage.WithExcludeSourceCodeInfo())
		require.NoError(t, err)
		_, err = FilterImage(image, WithIncludeTypes("pkg.extension"), WithExcludeTypes("pkg.Foo"))
		require.Error(t, err)
		assert.ErrorContains(t, err, "cannot include extension field \"pkg.extension\" as the extendee type \"pkg.Foo\" is excluded")
	})
}

func TestNesting(t *testing.T) {
	t.Parallel()
	t.Run("message", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/nesting", "message.txtar", WithIncludeTypes("pkg.Foo"))
	})
	t.Run("recursenested", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/nesting", "recursenested.txtar", WithIncludeTypes("pkg.Foo.NestedFoo.NestedNestedFoo"))
	})
	t.Run("enum", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/nesting", "enum.txtar", WithIncludeTypes("pkg.FooEnum"))
	})
	t.Run("usingother", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/nesting", "usingother.txtar", WithIncludeTypes("pkg.Baz"))
	})

	t.Run("exclude_message", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/nesting", "message.exclude.txtar", WithExcludeTypes("pkg.Foo"))
	})
	t.Run("exclude_recursenested", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/nesting", "recursenested.exclude.txtar", WithExcludeTypes("pkg.Foo.NestedFoo.NestedNestedFoo"))
	})
	t.Run("exclude_enum", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/nesting", "enum.exclude.txtar", WithExcludeTypes("pkg.FooEnum"))
	})
	t.Run("exclude_usingother", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/nesting", "usingother.exclude.txtar", WithExcludeTypes("pkg.Baz"))
	})

	t.Run("mixed", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/nesting", "mixed.txtar", WithIncludeTypes("pkg.Foo", "pkg.FooEnum"), WithExcludeTypes("pkg.Foo.NestedFoo", "pkg.Baz"))
	})

	t.Run("include-excluded", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		_, image, err := getImage(ctx, slogtestext.NewLogger(t), "testdata/nesting", bufimage.WithExcludeSourceCodeInfo())
		require.NoError(t, err)
		_, err = FilterImage(image, WithIncludeTypes("pkg.Foo.NestedFoo"), WithExcludeTypes("pkg.Foo"))
		require.ErrorContains(t, err, "inclusion of excluded type \"pkg.Foo.NestedFoo\"")
		_, err = FilterImage(image, WithIncludeTypes("pkg.Foo.NestedButNotUsed"), WithExcludeTypes("pkg.Foo"))
		require.ErrorContains(t, err, "inclusion of excluded type \"pkg.Foo.NestedButNotUsed\"")
	})
}

func TestOneof(t *testing.T) {
	t.Parallel()
	t.Run("exclude-partial", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/oneofs", "pkg.Foo.exclude-partial.txtar", WithIncludeTypes("pkg.Foo"), WithExcludeTypes("pkg.FooEnum", "pkg.Bar.BarNested"))
	})
	t.Run("exclude-bar", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/oneofs", "pkg.Foo.exclude-bar.txtar", WithIncludeTypes("pkg.Foo"), WithExcludeTypes("pkg.FooEnum", "pkg.Bar"))
	})
}

func TestOptions(t *testing.T) {
	t.Parallel()
	t.Run("include_option", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", "options.foo.include.txtar", WithIncludeTypes(
			"message_foo",
			"field_foo",
			"oneof_foo",
			"enum_foo",
			"enum_value_foo",
			"service_foo",
			"method_foo",
			"UsedOption.file_foo",
		))
	})
	t.Run("exclude_foo", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", "options.foo.exclude.txtar", WithExcludeTypes(
			"message_foo",
			"field_foo",
			"oneof_foo",
			"enum_foo",
			"enum_value_foo",
			"service_foo",
			"method_foo",
			"UsedOption.file_foo",
		))
	})
	t.Run("only_file", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", "options.only_file.txtar", WithExcludeTypes(
			"message_foo", "message_bar", "message_baz",
			"field_foo", "field_bar", "field_baz",
			"oneof_foo", "oneof_bar", "oneof_baz",
			"enum_foo", "enum_bar", "enum_baz",
			"enum_value_foo", "enum_value_bar", "enum_value_baz",
			"service_foo", "service_bar", "service_baz",
			"method_foo", "method_bar", "method_baz",
		))
	})
}

func TestOptionImports(t *testing.T) {
	t.Parallel()

	// This checks that when excluding options the imports are correctly dropped.
	// For this case when both options are removed only a.proto should be left.
	testdataDir := "testdata/imports"
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

	t.Run("exclude_foo", func(t *testing.T) {
		t.Parallel()
		generated, _ := runFilterImage(t, image, WithExcludeTypes("message_foo"))
		checkExpectation(t, context.Background(), generated, bucket, "foo.txtar")
	})
	t.Run("exclude_foo_bar", func(t *testing.T) {
		t.Parallel()
		generated, _ := runFilterImage(t, image, WithExcludeTypes("message_foo", "message_bar"))
		checkExpectation(t, context.Background(), generated, bucket, "foo_bar.txtar")
	})
	t.Run("exclude_bar", func(t *testing.T) {
		t.Parallel()
		generated, _ := runFilterImage(t, image, WithIncludeTypes("pkg.Foo"), WithExcludeTypes("message_bar"))
		checkExpectation(t, context.Background(), generated, bucket, "bar.txtar")
	})
}

func TestImportModifiers(t *testing.T) {
	t.Parallel()
	t.Run("regular_weak", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/importmods", "regular_weak.txtar", WithIncludeTypes("ImportRegular", "ImportWeak"))
	})
	t.Run("weak_public", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/importmods", "weak_public.txtar", WithIncludeTypes("ImportWeak", "ImportPublic"))
	})
	t.Run("regular_public", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/importmods", "regular_public.txtar", WithIncludeTypes("ImportRegular", "ImportPublic"))
	})
	t.Run("noimports", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/importmods", "noimports.txtar", WithIncludeTypes("NoImports"))
	})
}

func TestExtensions(t *testing.T) {
	t.Parallel()
	runDiffTest(t, "testdata/extensions", "extensions.txtar", WithIncludeTypes("pkg.Foo"))
	runDiffTest(t, "testdata/extensions", "extensions-excluded.txtar", WithExcludeKnownExtensions(), WithIncludeTypes("pkg.Foo"))
}

func TestPackages(t *testing.T) {
	t.Parallel()
	runDiffTest(t, "testdata/packages", "root.txtar", WithIncludeTypes(""))
	runDiffTest(t, "testdata/packages", "foo.txtar", WithIncludeTypes("foo"))
	runDiffTest(t, "testdata/packages", "foo.bar.txtar", WithIncludeTypes("foo.bar"))
	runDiffTest(t, "testdata/packages", "foo.bar.baz.txtar", WithIncludeTypes("foo.bar.baz"))
}

func TestAny(t *testing.T) {
	t.Parallel()
	runDiffTest(t, "testdata/any", "c1.txtar", WithIncludeTypes("ExtendedAnySyntax"))
	runDiffTest(t, "testdata/any", "c2.txtar", WithIncludeTypes("ExtendedAnySyntax_InField"))
	runDiffTest(t, "testdata/any", "c3.txtar", WithIncludeTypes("ExtendedAnySyntax_InList"))
	runDiffTest(t, "testdata/any", "c4.txtar", WithIncludeTypes("ExtendedAnySyntax_InMap"))
	runDiffTest(t, "testdata/any", "d.txtar", WithIncludeTypes("NormalMessageSyntaxValidType"))
	runDiffTest(t, "testdata/any", "e.txtar", WithIncludeTypes("NormalMessageSyntaxInvalidType"))
}

func TestSourceCodeInfo(t *testing.T) {
	t.Parallel()
	noExts := []ImageFilterOption{WithExcludeCustomOptions(), WithExcludeKnownExtensions()}
	runSourceCodeInfoTest(t, "foo.bar.Foo", "Foo.txtar", noExts...)
	runSourceCodeInfoTest(t, "foo.bar.Foo", "Foo+Ext.txtar")
	runSourceCodeInfoTest(t, "foo.bar.Foo.NestedFoo", "NestedFoo.txtar", noExts...)
	runSourceCodeInfoTest(t, "foo.bar.Bar", "Bar.txtar", noExts...)
	runSourceCodeInfoTest(t, "foo.bar.Baz", "Baz.txtar", noExts...)
	runSourceCodeInfoTest(t, "foo.bar.Quz", "Quz.txtar", noExts...)
	runSourceCodeInfoTest(t, "foo.bar.Svc", "Svc.txtar", noExts...)
	runSourceCodeInfoTest(t, "foo.bar.Svc.Do", "Do.txtar", noExts...)
	runSourceCodeInfoTest(t, "foo.bar", "all.txtar")
}

func TestUnusedDeps(t *testing.T) {
	t.Parallel()
	runDiffTest(t, "testdata/unuseddeps", "a.txtar", WithIncludeTypes("a.A"))
	runDiffTest(t, "testdata/unuseddeps", "ab.txtar", WithIncludeTypes("a.A"), WithExcludeTypes("b.B"))
}

func TestTransitivePublic(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	moduleSet, err := bufmoduletesting.NewModuleSetForPathToData(
		map[string][]byte{
			"a.proto": []byte(`syntax = "proto3";package a;message Foo{}`),
			"b.proto": []byte(`syntax = "proto3";package b;import public "a.proto";message Bar {}`),
			"c.proto": []byte(`syntax = "proto3";package c;import "b.proto";message Baz{ a.Foo foo = 1; }`),
		},
	)
	require.NoError(t, err)
	image, err := bufimage.BuildImage(
		ctx,
		slogtestext.NewLogger(t),
		bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(moduleSet),
		bufimage.WithExcludeSourceCodeInfo(),
	)
	require.NoError(t, err)

	filteredImage, err := FilterImage(image, WithIncludeTypes("c.Baz"))
	require.NoError(t, err)

	_, err = protodesc.NewFiles(bufimage.ImageToFileDescriptorSet(filteredImage))
	require.NoError(t, err)
}

func TestTypesFromMainModule(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	moduleSet, err := bufmoduletesting.NewOmniProvider(
		bufmoduletesting.ModuleData{
			Name: "buf.build/repo/main",
			PathToData: map[string][]byte{
				"a.proto": []byte(`syntax = "proto3";import "b.proto";package pkg;message Foo { dependency.Dep bar = 1;}`),
			},
		},
		bufmoduletesting.ModuleData{
			Name: "buf.build/repo/dep",
			PathToData: map[string][]byte{
				"b.proto": []byte(`syntax = "proto3";package dependency; message Dep{}`),
			},
			NotTargeted: true,
		},
	)
	require.NoError(t, err)
	image, err := bufimage.BuildImage(
		ctx,
		slogtestext.NewLogger(t),
		bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(moduleSet),
		bufimage.WithExcludeSourceCodeInfo(),
	)
	require.NoError(t, err)

	dep := moduleSet.GetModuleForOpaqueID("buf.build/repo/dep")
	require.NotNil(t, dep)
	bProtoFileInfo, err := dep.StatFileInfo(ctx, "b.proto")
	require.NoError(t, err)
	require.False(t, bProtoFileInfo.IsTargetFile())
	_, err = FilterImage(image, WithIncludeTypes("dependency.Dep"))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrImageFilterTypeIsImport)

	// allowed if we specify option
	_, err = FilterImage(image, WithIncludeTypes("dependency.Dep"), WithAllowIncludeOfImportedType())
	require.NoError(t, err)

	_, err = FilterImage(image, WithIncludeTypes("nonexisting"))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrImageFilterTypeNotFound)
}

func TestMutateInPlace(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	_, image, err := getImage(ctx, slogtestext.NewLogger(t), "testdata/options")
	require.NoError(t, err)

	aProtoFile := image.GetFile("a.proto")
	aFileDescriptorProto := aProtoFile.FileDescriptorProto()
	assert.Len(t, aFileDescriptorProto.MessageType, 2) // Foo, Empty
	assert.Len(t, aFileDescriptorProto.EnumType, 1)    // FooEnum
	assert.Len(t, aFileDescriptorProto.Service, 1)

	locationLen := len(aFileDescriptorProto.SourceCodeInfo.Location)

	// Shallow copy
	shallowFilteredImage, err := FilterImage(image, WithIncludeTypes("pkg.Foo"))
	require.NoError(t, err)

	filteredAFileDescriptorProto := shallowFilteredImage.GetFile("a.proto").FileDescriptorProto()
	assert.NotSame(t, aFileDescriptorProto, filteredAFileDescriptorProto)
	filterLocationLen := len(filteredAFileDescriptorProto.SourceCodeInfo.Location)
	assert.Less(t, filterLocationLen, locationLen)

	// Mutate in place
	mutateFilteredImage, err := FilterImage(image, WithIncludeTypes("pkg.Foo"), WithMutateInPlace())
	require.NoError(t, err)

	// Check that the original image was mutated
	assert.Same(t, aFileDescriptorProto, mutateFilteredImage.GetFile("a.proto").FileDescriptorProto())
	assert.Len(t, aFileDescriptorProto.MessageType, 1) // Foo
	if assert.GreaterOrEqual(t, cap(aFileDescriptorProto.MessageType), 2) {
		slice := aFileDescriptorProto.MessageType[1:cap(aFileDescriptorProto.MessageType)]
		for _, elem := range slice {
			assert.Nil(t, elem) // Empty
		}
	}
	assert.Nil(t, aFileDescriptorProto.EnumType)
	assert.Nil(t, aFileDescriptorProto.Service)
	assert.Equal(t, filterLocationLen, len(aFileDescriptorProto.SourceCodeInfo.Location))
}

func TestConsecutiveFilters(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, image, err := getImage(ctx, slogtestext.NewLogger(t), "testdata/options")
	require.NoError(t, err)

	t.Run("options", func(t *testing.T) {
		t.Parallel()
		filteredImage, err := FilterImage(image, WithIncludeTypes("pkg.Foo"), WithExcludeTypes("message_baz"))
		require.NoError(t, err)
		_, err = FilterImage(filteredImage, WithExcludeTypes("message_foo"))
		require.NoError(t, err)
	})
}

func TestDependencies(t *testing.T) {
	t.Parallel()
	// Test referred file for options of imported types.
	t.Run("FieldA", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/deps", "test.FieldA.txtar", WithIncludeTypes("test.FieldA"))
	})
	t.Run("EnumA", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/deps", "test.EnumA.txtar", WithIncludeTypes("test.EnumA"))
	})
	t.Run("PublicOrder", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/deps", "test.PublicOrder.txtar", WithIncludeTypes("test.PublicOrder"))
	})
	// Test an included type with implicitly excluded extensions fields.
	t.Run("IncludeWithExcludeExtensions", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/deps", "test.IncludeWithExcludeExt.txtar", WithIncludeTypes("google.protobuf.MessageOptions"), WithExcludeTypes("a", "b", "c"), WithAllowIncludeOfImportedType())
	})
}

func TestEmptyFiles(t *testing.T) {
	t.Parallel()
	t.Run("include_empty_file", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/empty", "empty.include.txtar", WithIncludeTypes("include"))
	})
	t.Run("exclude_empty_file", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/empty", "empty.exclude.txtar", WithExcludeTypes("include"))
	})
}

func getImage(ctx context.Context, logger *slog.Logger, testdataDir string, options ...bufimage.BuildImageOption) (storage.ReadWriteBucket, bufimage.Image, error) {
	bucket, err := storageos.NewProvider().NewReadWriteBucket(testdataDir)
	if err != nil {
		return nil, nil, err
	}
	moduleSet, err := bufmoduletesting.NewModuleSetForBucket(bucket)
	if err != nil {
		return nil, nil, err
	}
	image, err := bufimage.BuildImage(
		ctx,
		logger,
		bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(moduleSet),
		options...,
	)
	if err != nil {
		return nil, nil, err
	}
	return bucket, image, nil
}

func runDiffTest(t *testing.T, testdataDir string, expectedFile string, opts ...ImageFilterOption) {
	ctx := context.Background()
	bucket, image, err := getImage(ctx, slogtestext.NewLogger(t), testdataDir, bufimage.WithExcludeSourceCodeInfo())
	require.NoError(t, err)
	generated, _ := runFilterImage(t, image, opts...)
	checkExpectation(t, ctx, generated, bucket, expectedFile)
}

func runFilterImage(t *testing.T, image bufimage.Image, opts ...ImageFilterOption) ([]byte, bufimage.Image) {
	filteredImage, err := FilterImage(image, opts...)
	require.NoError(t, err)
	assert.NotNil(t, filteredImage)
	assert.True(t, imageIsDependencyOrdered(filteredImage), "image files not in dependency order")

	// Convert the filtered image back to a proto image and then back to an image to ensure that the
	// image is still valid after filtering.
	protoImage, err := bufimage.ImageToProtoImage(filteredImage)
	require.NoError(t, err)
	// Clone here as `bufimage.NewImageForProto` mutates protoImage.
	protoImage = proto.CloneOf(protoImage)
	filteredImage, err = bufimage.NewImageForProto(protoImage)
	require.NoError(t, err)

	// We may have filtered out custom options from the set in the step above. However, the options messages
	// still contain extension fields that refer to the custom options, as a result of building the image.
	// So we serialize and then de-serialize, and use only the filtered results to parse extensions. That way,
	// the result will omit custom options that aren't present in the filtered set (as they will be considered
	// unrecognized fields).
	fileDescriptorSet := &descriptorpb.FileDescriptorSet{
		File: xslices.Map(filteredImage.Files(), func(imageFile bufimage.ImageFile) *descriptorpb.FileDescriptorProto {
			return imageFile.FileDescriptorProto()
		}),
	}

	files, err := protodesc.NewFiles(fileDescriptorSet)
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
	return generated, filteredImage
}

func checkExpectation(t *testing.T, ctx context.Context, actual []byte, bucket storage.ReadWriteBucket, expectedFile string) {
	if shouldUpdateExpectations != "" {
		writer, err := bucket.Put(ctx, expectedFile)
		require.NoError(t, err)
		_, err = writer.Write(actual)
		require.NoError(t, err)
		require.NoError(t, writer.Close())
	} else {
		expectedReader, err := bucket.Get(ctx, expectedFile)
		require.NoError(t, err)
		expected, err := io.ReadAll(expectedReader)
		require.NoError(t, err)
		assert.Equal(t, string(expected), string(actual))
	}
}

func runSourceCodeInfoTest(t *testing.T, typename string, expectedFile string, opts ...ImageFilterOption) {
	ctx := context.Background()
	bucket, image, err := getImage(ctx, slogtestext.NewLogger(t), "testdata/sourcecodeinfo")
	require.NoError(t, err)

	opts = append(opts, WithIncludeTypes(typename))
	generated, image := runFilterImage(t, image, opts...)
	filteredImage, err := FilterImage(image, opts...)
	require.NoError(t, err)

	imageFile := filteredImage.GetFile("test.proto")
	sourceCodeInfo := imageFile.FileDescriptorProto().GetSourceCodeInfo()
	actual, err := protoencoding.NewJSONMarshaler(nil, protoencoding.JSONMarshalerWithIndent()).Marshal(sourceCodeInfo)
	require.NoError(t, err)
	generated = append(generated, []byte("-- source_code_info.json --\n")...)
	generated = append(generated, actual...)

	checkExpectation(t, ctx, generated, bucket, expectedFile)

	resolver, err := protoencoding.NewResolver(bufimage.ImageToFileDescriptorProtos(filteredImage)...)
	require.NoError(t, err)
	file, err := resolver.FindFileByPath("test.proto")
	require.NoError(t, err)
	examineComments(t, file)
}

func imageIsDependencyOrdered(image bufimage.Image) bool {
	seen := make(map[string]struct{})
	for _, imageFile := range image.Files() {
		for _, importName := range imageFile.FileDescriptorProto().Dependency {
			if _, ok := seen[importName]; !ok {
				return false
			}
		}
		seen[imageFile.Path()] = struct{}{}
	}
	return true
}

func examineComments(t *testing.T, file protoreflect.FileDescriptor) {
	examineCommentsInTypeContainer(t, file, file)
	svcs := file.Services()
	for i, numSvcs := 0, svcs.Len(); i < numSvcs; i++ {
		svc := svcs.Get(i)
		examineComment(t, file, svc)
		methods := svc.Methods()
		for j, numMethods := 0, methods.Len(); j < numMethods; j++ {
			method := methods.Get(j)
			examineComment(t, file, method)
		}
	}
}

type typeContainer interface {
	Messages() protoreflect.MessageDescriptors
	Enums() protoreflect.EnumDescriptors
	Extensions() protoreflect.ExtensionDescriptors
}

func examineCommentsInTypeContainer(t *testing.T, file protoreflect.FileDescriptor, descriptor typeContainer) {
	msgs := descriptor.Messages()
	for i, numMsgs := 0, msgs.Len(); i < numMsgs; i++ {
		msg := msgs.Get(i)
		examineComment(t, file, msg)
		fields := msg.Fields()
		for j, numFields := 0, fields.Len(); j < numFields; j++ {
			field := fields.Get(j)
			examineComment(t, file, field)
		}
		oneofs := msg.Oneofs()
		for j, numOneofs := 0, oneofs.Len(); j < numOneofs; j++ {
			oneof := oneofs.Get(j)
			examineComment(t, file, oneof)
		}
		examineCommentsInTypeContainer(t, file, msg)
	}
	enums := descriptor.Enums()
	for i, numEnums := 0, enums.Len(); i < numEnums; i++ {
		enum := enums.Get(i)
		examineComment(t, file, enum)
		vals := enum.Values()
		for j, numVals := 0, vals.Len(); j < numVals; j++ {
			val := vals.Get(j)
			examineComment(t, file, val)
		}
	}
	exts := descriptor.Extensions()
	for i, numExts := 0, exts.Len(); i < numExts; i++ {
		ext := exts.Get(i)
		examineComment(t, file, ext)
	}
}

func examineComment(t *testing.T, file protoreflect.FileDescriptor, descriptor protoreflect.Descriptor) {
	loc := file.SourceLocations().ByDescriptor(descriptor)
	if loc.LeadingComments == "" {
		// Messages that are only present because they are namespaces that contains a retained
		// type will not have a comment. So we can skip the comment check for that case.
		if msg, ok := descriptor.(protoreflect.MessageDescriptor); ok && msg.Fields().Len() == 0 {
			return
		}
	}
	// Verify we got the correct location by checking that the comment contains the element's name
	require.Contains(t, loc.LeadingComments, string(descriptor.Name()))
}

func BenchmarkFilterImage_WithoutSourceCodeInfo(b *testing.B) {
	benchmarkFilterImage(b, bufimage.WithExcludeSourceCodeInfo())
}

func BenchmarkFilterImage_WithSourceCodeInfo(b *testing.B) {
	benchmarkFilterImage(b)
}

func benchmarkFilterImage(b *testing.B, opts ...bufimage.BuildImageOption) {
	benchmarkCases := []*struct {
		folder string
		image  bufimage.Image
		types  []string
	}{
		{
			folder: "testdata/extensions",
			types:  []string{"pkg.Foo"},
		},
		{
			folder: "testdata/importmods",
			types:  []string{"ImportRegular", "ImportWeak", "ImportPublic", "NoImports"},
		},
		{
			folder: "testdata/nesting",
			types:  []string{"pkg.Foo", "pkg.Foo.NestedFoo.NestedNestedFoo", "pkg.Baz", "pkg.FooEnum"},
		},
		{
			folder: "testdata/options",
			types:  []string{"pkg.Foo", "pkg.FooEnum", "pkg.FooService", "pkg.FooService.Do"},
		},
	}
	ctx := context.Background()
	for _, benchmarkCase := range benchmarkCases {
		_, image, err := getImage(ctx, slogtestext.NewLogger(b, slogtestext.WithLogLevel(appext.LogLevelError)), benchmarkCase.folder, opts...)
		require.NoError(b, err)
		benchmarkCase.image = image
	}
	b.ResetTimer()

	i := 0
	for {
		for _, benchmarkCase := range benchmarkCases {
			for _, typeName := range benchmarkCase.types {
				// filtering is destructive, so we have to make a copy
				b.StopTimer()
				imageFiles := make([]bufimage.ImageFile, len(benchmarkCase.image.Files()))
				for j, imageFile := range benchmarkCase.image.Files() {
					clone := proto.CloneOf(imageFile.FileDescriptorProto())
					var err error
					imageFiles[j], err = bufimage.NewImageFile(clone, nil, uuid.Nil, "", "", false, false, nil)
					require.NoError(b, err)
				}
				image, err := bufimage.NewImage(imageFiles)
				require.NoError(b, err)
				b.StartTimer()

				_, err = FilterImage(image, WithIncludeTypes(typeName), WithMutateInPlace())
				require.NoError(b, err)
				i++
				if i == b.N {
					return
				}
			}
		}
	}
}

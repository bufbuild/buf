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

package bufimageutil

import (
	"context"
	"os"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"github.com/gofrs/uuid/v5"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/descriptorpb"
)

// IF YOU HAVE ANY FAILING TESTS IN HERE, ESPECIALLY AFTER A PROTOC UPGRADE,
// RUN THE FOLLOWING:
// make bufimageutilupdateexpectations

var shouldUpdateExpectations = os.Getenv("BUFBUILD_BUF_BUFIMAGEUTIL_SHOULD_UPDATE_EXPECTATIONS")

func TestOptions(t *testing.T) {
	t.Parallel()
	t.Run("message", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", []string{"pkg.Foo"}, "pkg.Foo.binpb")
	})
	t.Run("enum", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", []string{"pkg.FooEnum"}, "pkg.FooEnum.binpb")
	})
	t.Run("service", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", []string{"pkg.FooService"}, "pkg.FooService.binpb")
	})
	t.Run("method", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", []string{"pkg.FooService.Do"}, "pkg.FooService.Do.binpb")
	})
	t.Run("all", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", []string{"pkg.Foo", "pkg.FooEnum", "pkg.FooService"}, "all.binpb")
	})
	t.Run("exclude-options", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", []string{"pkg.Foo", "pkg.FooEnum", "pkg.FooService"}, "all-exclude-options.binpb", WithExcludeCustomOptions())
	})
	t.Run("files", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", []string{"Files"}, "Files.binpb")
	})
	t.Run("all-with-files", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", []string{"pkg.Foo", "pkg.FooEnum", "pkg.FooService", "Files"}, "all-with-Files.binpb")
	})
}

func TestNesting(t *testing.T) {
	t.Parallel()
	t.Run("message", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/nesting", []string{"pkg.Foo"}, "message.binpb")
	})
	t.Run("recursenested", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/nesting", []string{"pkg.Foo.NestedFoo.NestedNestedFoo"}, "recursenested.binpb")
	})
	t.Run("enum", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/nesting", []string{"pkg.FooEnum"}, "enum.binpb")
	})
	t.Run("usingother", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/nesting", []string{"pkg.Baz"}, "usingother.binpb")
	})
}

func TestImportModifiers(t *testing.T) {
	t.Parallel()
	t.Run("regular_weak", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/importmods", []string{"ImportRegular", "ImportWeak"}, "regular_weak.binpb")
	})
	t.Run("weak_public", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/importmods", []string{"ImportWeak", "ImportPublic"}, "weak_public.binpb")
	})
	t.Run("regular_public", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/importmods", []string{"ImportRegular", "ImportPublic"}, "regular_public.binpb")
	})
	t.Run("noimports", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/importmods", []string{"NoImports"}, "noimports.binpb")
	})
}

func TestExtensions(t *testing.T) {
	t.Parallel()
	runDiffTest(t, "testdata/extensions", []string{"pkg.Foo"}, "extensions.binpb")
	runDiffTest(t, "testdata/extensions", []string{"pkg.Foo"}, "extensions-excluded.binpb", WithExcludeKnownExtensions())
}

func TestPackages(t *testing.T) {
	t.Parallel()
	runDiffTest(t, "testdata/packages", []string{""}, "root.binpb")
	runDiffTest(t, "testdata/packages", []string{"foo"}, "foo.binpb")
	runDiffTest(t, "testdata/packages", []string{"foo.bar"}, "foo.bar.binpb")
	runDiffTest(t, "testdata/packages", []string{"foo.bar.baz"}, "foo.bar.baz.binpb")
}

func TestAny(t *testing.T) {
	t.Parallel()
	runDiffTest(t, "testdata/any", []string{"ExtendedAnySyntax"}, "c1.binpb")
	runDiffTest(t, "testdata/any", []string{"ExtendedAnySyntax_InField"}, "c2.binpb")
	runDiffTest(t, "testdata/any", []string{"ExtendedAnySyntax_InList"}, "c3.binpb")
	runDiffTest(t, "testdata/any", []string{"ExtendedAnySyntax_InMap"}, "c4.binpb")
	runDiffTest(t, "testdata/any", []string{"NormalMessageSyntaxValidType"}, "d.binpb")
	runDiffTest(t, "testdata/any", []string{"NormalMessageSyntaxInvalidType"}, "e.binpb")
}

func TestSourceCodeInfo(t *testing.T) {
	t.Parallel()
	noExts := []ImageFilterOption{WithExcludeCustomOptions(), WithExcludeKnownExtensions()}
	runSourceCodeInfoTest(t, "foo.bar.Foo", "Foo.json", noExts...)
	runSourceCodeInfoTest(t, "foo.bar.Foo", "Foo+Ext.json")
	runSourceCodeInfoTest(t, "foo.bar.Foo.NestedFoo", "NestedFoo.json", noExts...)
	runSourceCodeInfoTest(t, "foo.bar.Bar", "Bar.json", noExts...)
	runSourceCodeInfoTest(t, "foo.bar.Baz", "Baz.json", noExts...)
	runSourceCodeInfoTest(t, "foo.bar.Quz", "Quz.json", noExts...)
	runSourceCodeInfoTest(t, "foo.bar.Svc", "Svc.json", noExts...)
	runSourceCodeInfoTest(t, "foo.bar.Svc.Do", "Do.json", noExts...)
	runSourceCodeInfoTest(t, "foo.bar", "all.json")
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
		tracing.NopTracer,
		bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(moduleSet),
		bufimage.WithExcludeSourceCodeInfo(),
	)
	require.NoError(t, err)

	filteredImage, err := ImageFilteredByTypes(image, "c.Baz")
	require.NoError(t, err)

	_, err = protoencoding.NewWireMarshaler().Marshal(bufimage.ImageToFileDescriptorSet(filteredImage))
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
		tracing.NopTracer,
		bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(moduleSet),
		bufimage.WithExcludeSourceCodeInfo(),
	)
	require.NoError(t, err)

	dep := moduleSet.GetModuleForOpaqueID("buf.build/repo/dep")
	require.NotNil(t, dep)
	bProtoFileInfo, err := dep.StatFileInfo(ctx, "b.proto")
	require.NoError(t, err)
	require.False(t, bProtoFileInfo.IsTargetFile())
	_, err = ImageFilteredByTypes(image, "dependency.Dep")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrImageFilterTypeIsImport)

	// allowed if we specify option
	_, err = ImageFilteredByTypesWithOptions(image, []string{"dependency.Dep"}, WithAllowFilterByImportedType())
	require.NoError(t, err)

	_, err = ImageFilteredByTypes(image, "nonexisting")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrImageFilterTypeNotFound)
}

func getImage(ctx context.Context, logger *zap.Logger, testdataDir string, options ...bufimage.BuildImageOption) (storage.ReadWriteBucket, bufimage.Image, error) {
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
		tracing.NopTracer,
		bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(moduleSet),
		options...,
	)
	if err != nil {
		return nil, nil, err
	}
	return bucket, image, nil
}

func runDiffTest(t *testing.T, testdataDir string, typenames []string, expectedFile string, opts ...ImageFilterOption) {
	ctx := context.Background()
	bucket, image, err := getImage(ctx, zaptest.NewLogger(t), testdataDir, bufimage.WithExcludeSourceCodeInfo())
	require.NoError(t, err)

	filteredImage, err := ImageFilteredByTypesWithOptions(image, typenames, opts...)
	require.NoError(t, err)
	assert.NotNil(t, image)
	assert.True(t, imageIsDependencyOrdered(filteredImage), "image files not in dependency order")

	// We may have filtered out custom options from the set in the step above. However, the options messages
	// still contain extension fields that refer to the custom options, as a result of building the image.
	// So we serialize and then de-serialize, and use only the filtered results to parse extensions. That
	// way, the result will omit custom options that aren't present in the filtered set (as they will be
	// considered unrecognized fields).
	data, err := proto.Marshal(bufimage.ImageToFileDescriptorSet(filteredImage))
	require.NoError(t, err)
	fileDescriptorSet := &descriptorpb.FileDescriptorSet{}
	err = proto.UnmarshalOptions{Resolver: filteredImage.Resolver()}.Unmarshal(data, fileDescriptorSet)
	require.NoError(t, err)

	checkExpectation(
		t, ctx, fileDescriptorSet, bucket, expectedFile,
		protoencoding.NewWireMarshaler().Marshal,
		protoencoding.NewWireUnmarshaler(filteredImage.Resolver()).Unmarshal,
	)
}

func checkExpectation(
	t *testing.T, ctx context.Context, actual proto.Message, bucket storage.ReadWriteBucket, expectedFile string,
	marshalFunc func(proto.Message) ([]byte, error),
	unmarshalFunc func([]byte, proto.Message) error,
) {
	if shouldUpdateExpectations != "" {
		data, err := marshalFunc(actual)
		require.NoError(t, err)
		require.NoError(t, storage.PutPath(ctx, bucket, expectedFile, data))
	} else {
		expected, err := storage.ReadPath(ctx, bucket, expectedFile)
		require.NoError(t, err)
		expect := actual.ProtoReflect().New().Interface()
		require.NoError(t, unmarshalFunc(expected, expect))
		assert.Empty(t, cmp.Diff(expect, actual, protocmp.Transform()))
	}
}

func runSourceCodeInfoTest(t *testing.T, typename string, expectedFile string, opts ...ImageFilterOption) {
	ctx := context.Background()
	bucket, image, err := getImage(ctx, zaptest.NewLogger(t), "testdata/sourcecodeinfo")
	require.NoError(t, err)

	filteredImage, err := ImageFilteredByTypesWithOptions(image, []string{typename}, opts...)
	require.NoError(t, err)

	imageFile := filteredImage.GetFile("test.proto")
	sourceCodeInfo := imageFile.FileDescriptorProto().GetSourceCodeInfo()
	resolver := filteredImage.Resolver()

	checkExpectation(
		t, ctx, sourceCodeInfo, bucket, expectedFile,
		protoencoding.NewJSONMarshaler(resolver, protoencoding.JSONMarshalerWithIndent()).Marshal,
		protoencoding.NewJSONUnmarshaler(resolver).Unmarshal,
	)

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
		_, image, err := getImage(ctx, zaptest.NewLogger(b), benchmarkCase.folder, opts...)
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
					clone, ok := proto.Clone(imageFile.FileDescriptorProto()).(*descriptorpb.FileDescriptorProto)
					require.True(b, ok)
					var err error
					imageFiles[j], err = bufimage.NewImageFile(clone, nil, uuid.Nil, "", "", false, false, nil)
					require.NoError(b, err)
				}
				image, err := bufimage.NewImage(imageFiles)
				require.NoError(b, err)
				b.StartTimer()

				_, err = ImageFilteredByTypes(image, typeName)
				require.NoError(b, err)
				i++
				if i == b.N {
					return
				}
			}
		}
	}
}

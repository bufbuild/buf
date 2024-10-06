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
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"sort"
	"testing"

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

func TestOptions(t *testing.T) {
	t.Parallel()
	t.Run("message", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", []string{"pkg.Foo"}, "pkg.Foo.txtar")
	})
	t.Run("enum", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", []string{"pkg.FooEnum"}, "pkg.FooEnum.txtar")
	})
	t.Run("service", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", []string{"pkg.FooService"}, "pkg.FooService.txtar")
	})
	t.Run("method", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", []string{"pkg.FooService.Do"}, "pkg.FooService.Do.txtar")
	})
	t.Run("all", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", []string{"pkg.Foo", "pkg.FooEnum", "pkg.FooService"}, "all.txtar")
	})
	t.Run("exclude-options", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", []string{"pkg.Foo", "pkg.FooEnum", "pkg.FooService"}, "all-exclude-options.txtar", WithExcludeCustomOptions())
	})
	t.Run("files", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", []string{"Files"}, "Files.txtar")
	})
	t.Run("all-with-files", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/options", []string{"pkg.Foo", "pkg.FooEnum", "pkg.FooService", "Files"}, "all-with-Files.txtar")
	})
}

func TestNesting(t *testing.T) {
	t.Parallel()
	t.Run("message", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/nesting", []string{"pkg.Foo"}, "message.txtar")
	})
	t.Run("recursenested", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/nesting", []string{"pkg.Foo.NestedFoo.NestedNestedFoo"}, "recursenested.txtar")
	})
	t.Run("enum", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/nesting", []string{"pkg.FooEnum"}, "enum.txtar")
	})
	t.Run("usingother", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/nesting", []string{"pkg.Baz"}, "usingother.txtar")
	})
}

func TestImportModifiers(t *testing.T) {
	t.Parallel()
	t.Run("regular_weak", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/importmods", []string{"ImportRegular", "ImportWeak"}, "regular_weak.txtar")
	})
	t.Run("weak_public", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/importmods", []string{"ImportWeak", "ImportPublic"}, "weak_public.txtar")
	})
	t.Run("regular_public", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/importmods", []string{"ImportRegular", "ImportPublic"}, "regular_public.txtar")
	})
	t.Run("noimports", func(t *testing.T) {
		t.Parallel()
		runDiffTest(t, "testdata/importmods", []string{"NoImports"}, "noimports.txtar")
	})
}

func TestExtensions(t *testing.T) {
	t.Parallel()
	runDiffTest(t, "testdata/extensions", []string{"pkg.Foo"}, "extensions.txtar")
	runDiffTest(t, "testdata/extensions", []string{"pkg.Foo"}, "extensions-excluded.txtar", WithExcludeKnownExtensions())
}

func TestPackages(t *testing.T) {
	t.Parallel()
	runDiffTest(t, "testdata/packages", []string{""}, "root.txtar")
	runDiffTest(t, "testdata/packages", []string{"foo"}, "foo.txtar")
	runDiffTest(t, "testdata/packages", []string{"foo.bar"}, "foo.bar.txtar")
	runDiffTest(t, "testdata/packages", []string{"foo.bar.baz"}, "foo.bar.baz.txtar")
}

func TestAny(t *testing.T) {
	t.Parallel()
	runDiffTest(t, "testdata/any", []string{"ExtendedAnySyntax"}, "c1.txtar")
	runDiffTest(t, "testdata/any", []string{"ExtendedAnySyntax_InField"}, "c2.txtar")
	runDiffTest(t, "testdata/any", []string{"ExtendedAnySyntax_InList"}, "c3.txtar")
	runDiffTest(t, "testdata/any", []string{"ExtendedAnySyntax_InMap"}, "c4.txtar")
	runDiffTest(t, "testdata/any", []string{"NormalMessageSyntaxValidType"}, "d.txtar")
	runDiffTest(t, "testdata/any", []string{"NormalMessageSyntaxInvalidType"}, "e.txtar")
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

	filteredImage, err := ImageFilteredByTypes(image, "c.Baz")
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

func runDiffTest(t *testing.T, testdataDir string, typenames []string, expectedFile string, opts ...ImageFilterOption) {
	ctx := context.Background()
	bucket, image, err := getImage(ctx, slogtestext.NewLogger(t), testdataDir, bufimage.WithExcludeSourceCodeInfo())
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
	data, err := protoencoding.NewWireMarshaler().Marshal(bufimage.ImageToFileDescriptorSet(filteredImage))
	require.NoError(t, err)
	fileDescriptorSet := &descriptorpb.FileDescriptorSet{}
	err = protoencoding.NewWireUnmarshaler(filteredImage.Resolver()).Unmarshal(data, fileDescriptorSet)
	require.NoError(t, err)

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
	checkExpectation(t, ctx, generated, bucket, expectedFile)
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

	filteredImage, err := ImageFilteredByTypesWithOptions(image, []string{typename}, opts...)
	require.NoError(t, err)

	imageFile := filteredImage.GetFile("test.proto")
	sourceCodeInfo := imageFile.FileDescriptorProto().GetSourceCodeInfo()
	actual, err := protoencoding.NewJSONMarshaler(nil, protoencoding.JSONMarshalerWithIndent()).Marshal(sourceCodeInfo)
	require.NoError(t, err)

	checkExpectation(t, ctx, actual, bucket, expectedFile)

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
		_, image, err := getImage(ctx, slogtestext.NewLogger(b), benchmarkCase.folder, opts...)
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

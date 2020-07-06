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

package bufbuild

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufcore"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufcoreutil"
	"github.com/bufbuild/buf/internal/buf/bufmod"
	"github.com/bufbuild/buf/internal/buf/internal/buftesting"
	"github.com/bufbuild/buf/internal/pkg/protosource"
	"github.com/bufbuild/buf/internal/pkg/prototesting"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

var buftestingDirPath = filepath.Join(
	"..",
	"internal",
	"buftesting",
)

func TestBuildGoogleapis(t *testing.T) {
	t.Parallel()
	//for _, includeSourceInfo := range []bool{true, false} {
	for _, includeSourceInfo := range []bool{false} {
		t.Run(
			fmt.Sprintf("includeSourceInfo:%v", includeSourceInfo),
			func(t *testing.T) {
				t.Parallel()
				testBuildGoogleapis(t, includeSourceInfo)
			},
		)
	}
}

func TestBufProtocGoogleapis(t *testing.T) {
	t.Parallel()
	//for _, includeSourceInfo := range []bool{true, false} {
	for _, includeSourceInfo := range []bool{false} {
		t.Run(
			fmt.Sprintf("includeSourceInfo:%v", includeSourceInfo),
			func(t *testing.T) {
				t.Parallel()
				testBuildBufProtocGoogleapis(t, includeSourceInfo)
			},
		)
	}
}

func TestActualProtocGoogleapis(t *testing.T) {
	t.Parallel()
	//for _, includeSourceInfo := range []bool{true, false} {
	for _, includeSourceInfo := range []bool{false} {
		t.Run(
			fmt.Sprintf("includeSourceInfo:%v", includeSourceInfo),
			func(t *testing.T) {
				t.Parallel()
				testBuildActualProtocGoogleapis(t, includeSourceInfo)
			},
		)
	}
}

func TestCompareGoogleapis(t *testing.T) {
	t.Parallel()
	//for _, includeSourceInfo := range []bool{true, false} {
	for _, includeSourceInfo := range []bool{false} {
		t.Run(
			fmt.Sprintf("includeSourceInfo:%v", includeSourceInfo),
			func(t *testing.T) {
				t.Parallel()
				image := testBuildGoogleapis(t, includeSourceInfo)
				fileDescriptorSet := bufcore.ImageToFileDescriptorSet(image)
				actualProtocFileDescriptorSet := testBuildActualProtocGoogleapis(t, includeSourceInfo)
				assertFileDescriptorSetsEqualJSON(t, fileDescriptorSet, actualProtocFileDescriptorSet)
				// Cannot compare due to unknown field issue
				//assertFileDescriptorSetsEqualProto(t, fileDescriptorSet, protocFileDescriptorSet)
			},
		)
	}
}

func TestCompareBufProtocGoogleapis(t *testing.T) {
	t.Parallel()
	//for _, includeSourceInfo := range []bool{true, false} {
	for _, includeSourceInfo := range []bool{false} {
		t.Run(
			fmt.Sprintf("includeSourceInfo:%v", includeSourceInfo),
			func(t *testing.T) {
				t.Parallel()
				bufProtocFileDescriptorSet := testBuildBufProtocGoogleapis(t, includeSourceInfo)
				actualProtocFileDescriptorSet := testBuildActualProtocGoogleapis(t, includeSourceInfo)
				assertFileDescriptorSetsEqualJSON(t, bufProtocFileDescriptorSet, actualProtocFileDescriptorSet)
				// Cannot compare due to unknown field issue
				//assertFileDescriptorSetsEqualProto(t, fileDescriptorSet, protocFileDescriptorSet)
			},
		)
	}
}

func TestCompareCustomOptions1(t *testing.T) {
	testCompare(t, "customoptions1")
}

func TestCompareProto3Optional1(t *testing.T) {
	testCompare(t, "proto3optional1")
}

func TestCustomOptionsError1(t *testing.T) {
	t.Parallel()
	_, fileAnnotations := testBuild(t, false, filepath.Join("testdata", "customoptionserror1"))
	require.Equal(t, 1, len(fileAnnotations), fileAnnotations)
	require.Equal(
		t,
		"testdata/customoptionserror1/b.proto:9:27:field a.Baz.bat: option (a.foo).bat: field bat of a.Foo does not exist",
		fileAnnotations[0].String(),
	)
}

func testCompare(t *testing.T, relDirPath string) {
	t.Parallel()
	dirPath := filepath.Join("testdata", relDirPath)
	image, fileAnnotations := testBuild(t, false, dirPath)
	require.Equal(t, 0, len(fileAnnotations), fileAnnotations)
	image = bufcore.ImageWithoutImports(image)
	fileDescriptorSet := bufcore.ImageToFileDescriptorSet(image)
	actualProtocFileDescriptorSet := buftesting.GetActualProtocFileDescriptorSet(t, false, false, dirPath)
	assertFileDescriptorSetsEqualWire(t, fileDescriptorSet, actualProtocFileDescriptorSet)
	assertFileDescriptorSetsEqualJSON(t, fileDescriptorSet, actualProtocFileDescriptorSet)
	assertFileDescriptorSetsEqualProto(t, fileDescriptorSet, actualProtocFileDescriptorSet)
}

func testBuildGoogleapis(t *testing.T, includeSourceInfo bool) bufcore.Image {
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	image, fileAnnotations := testBuild(t, includeSourceInfo, googleapisDirPath)

	assert.Equal(t, 0, len(fileAnnotations), fileAnnotations)
	assert.Equal(t, buftesting.NumGoogleapisFilesWithImports, len(image.Files()))
	assert.Equal(
		t,
		[]string{
			"google/protobuf/any.proto",
			"google/protobuf/api.proto",
			"google/protobuf/descriptor.proto",
			"google/protobuf/duration.proto",
			"google/protobuf/empty.proto",
			"google/protobuf/field_mask.proto",
			"google/protobuf/source_context.proto",
			"google/protobuf/struct.proto",
			"google/protobuf/timestamp.proto",
			"google/protobuf/type.proto",
			"google/protobuf/wrappers.proto",
		},
		testGetImageImportPaths(image),
	)

	imageWithoutImports := bufcore.ImageWithoutImports(image)
	assert.Equal(t, buftesting.NumGoogleapisFiles, len(imageWithoutImports.Files()))
	imageWithoutImports = bufcore.ImageWithoutImports(imageWithoutImports)
	assert.Equal(t, buftesting.NumGoogleapisFiles, len(imageWithoutImports.Files()))

	imageWithSpecificNames, err := bufcore.ImageWithOnlyPathsAllowNotExist(
		image,
		[]string{
			"google/protobuf/descriptor.proto",
			"google/protobuf/api.proto",
			"google/type/date.proto",
			"google/foo/nonsense.proto",
		},
	)
	assert.NoError(t, err)
	assert.Equal(
		t,
		[]string{
			"google/protobuf/any.proto",
			"google/protobuf/api.proto",
			"google/protobuf/descriptor.proto",
			"google/protobuf/source_context.proto",
			"google/protobuf/type.proto",
			"google/type/date.proto",
		},
		testGetImageFilePaths(imageWithSpecificNames),
	)
	imageWithoutImports = bufcore.ImageWithoutImports(imageWithSpecificNames)
	assert.Equal(
		t,
		[]string{
			"google/protobuf/api.proto",
			"google/protobuf/descriptor.proto",
			"google/type/date.proto",
		},
		testGetImageFilePaths(imageWithoutImports),
	)
	_, err = bufcore.ImageWithOnlyPaths(
		image,
		[]string{
			"google/protobuf/descriptor.proto",
			"google/protobuf/api.proto",
			"google/type/date.proto",
			"google/foo/nonsense.proto",
		},
	)
	// TODO
	assert.Equal(t, errors.New("google/foo/nonsense.proto is not present in the Image"), err)

	assert.Equal(t, buftesting.NumGoogleapisFilesWithImports, len(image.Files()))
	// basic check to make sure there is no error at this scale
	_, err = protosource.NewFilesUnstable(context.Background(), bufcoreutil.NewInputFiles(image.Files())...)
	assert.NoError(t, err)
	return image
}

func testBuildBufProtocGoogleapis(t *testing.T, includeSourceInfo bool) *descriptorpb.FileDescriptorSet {
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	fileDescriptorSet := testBuildBufProtoc(t, true, includeSourceInfo, googleapisDirPath)
	assert.Equal(t, buftesting.NumGoogleapisFilesWithImports, len(fileDescriptorSet.GetFile()))
	return fileDescriptorSet
}

func testBuildActualProtocGoogleapis(t *testing.T, includeSourceInfo bool) *descriptorpb.FileDescriptorSet {
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	fileDescriptorSet := buftesting.GetActualProtocFileDescriptorSet(t, true, includeSourceInfo, googleapisDirPath)
	assert.Equal(t, buftesting.NumGoogleapisFilesWithImports, len(fileDescriptorSet.GetFile()))
	return fileDescriptorSet
}

func testBuild(t *testing.T, includeSourceInfo bool, dirPath string) (bufcore.Image, []bufanalysis.FileAnnotation) {
	module := testGetModule(t, dirPath)
	var options []BuildOption
	if !includeSourceInfo {
		options = append(options, WithExcludeSourceCodeInfo())
	}
	image, fileAnnotations, err := NewBuilder(zap.NewNop()).Build(
		context.Background(),
		module,
		options...,
	)
	require.NoError(t, err)
	return image, fileAnnotations
}

func testBuildBufProtoc(
	t *testing.T,
	includeImports bool,
	includeSourceInfo bool,
	dirPath string,
) *descriptorpb.FileDescriptorSet {
	module := testGetModuleBufProtoc(t, dirPath)
	var options []BuildOption
	if !includeSourceInfo {
		options = append(options, WithExcludeSourceCodeInfo())
	}
	image, fileAnnotations, err := NewBuilder(zap.NewNop()).Build(
		context.Background(),
		module,
		options...,
	)
	require.NoError(t, err)
	require.Empty(t, fileAnnotations)
	if !includeImports {
		image = bufcore.ImageWithoutImports(image)
	}
	return bufcore.ImageToFileDescriptorSet(image)
}

func testGetModule(t *testing.T, dirPath string) bufcore.Module {
	readWriteBucket, err := storageos.NewReadWriteBucket(dirPath)
	require.NoError(t, err)
	config, err := bufmod.NewConfig(bufmod.ExternalConfig{})
	require.NoError(t, err)
	module, err := bufmod.NewBucketBuilder(zap.NewNop()).BuildForBucket(
		context.Background(),
		readWriteBucket,
		config,
	)
	require.NoError(t, err)
	return module
}

func testGetModuleBufProtoc(t *testing.T, dirPath string) bufcore.Module {
	module, err := bufmod.NewIncludeBuilder(zap.NewNop()).BuildForIncludes(
		context.Background(),
		[]string{dirPath},
	)
	require.NoError(t, err)
	return module
}

func testGetImageFilePaths(image bufcore.Image) []string {
	var fileNames []string
	for _, file := range image.Files() {
		fileNames = append(fileNames, file.Path())
	}
	sort.Strings(fileNames)
	return fileNames
}

func testGetImageImportPaths(image bufcore.Image) []string {
	var importNames []string
	for _, file := range image.Files() {
		if file.IsImport() {
			importNames = append(importNames, file.Path())
		}
	}
	sort.Strings(importNames)
	return importNames
}

func assertFileDescriptorSetsEqualWire(t *testing.T, one *descriptorpb.FileDescriptorSet, two *descriptorpb.FileDescriptorSet) {
	diffTwo, err := prototesting.DiffFileDescriptorSetsWire(one, two, "protoparse-protoc")
	assert.NoError(t, err)
	assert.Equal(t, "", diffTwo, "Wire diff:\n%s", diffTwo)
}

func assertFileDescriptorSetsEqualJSON(t *testing.T, one *descriptorpb.FileDescriptorSet, two *descriptorpb.FileDescriptorSet) {
	// TODO: test with resolver?
	// This also has the effect of verifying output order
	diffOne, err := prototesting.DiffFileDescriptorSetsJSON(one, two, "protoparse-protoc")
	assert.NoError(t, err)
	assert.Equal(t, "", diffOne, "JSON diff:\n%s", diffOne)
}

func assertFileDescriptorSetsEqualProto(t *testing.T, one *descriptorpb.FileDescriptorSet, two *descriptorpb.FileDescriptorSet) {
	equal := proto.Equal(one, two)
	assert.True(t, equal, "proto.Equal returned false")
}

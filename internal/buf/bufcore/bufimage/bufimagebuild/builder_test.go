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

package bufimagebuild

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"testing"

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage/bufimageutil"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/internal/buf/internal/buftesting"
	"github.com/bufbuild/buf/internal/pkg/protosource"
	"github.com/bufbuild/buf/internal/pkg/prototesting"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/descriptorpb"
)

var buftestingDirPath = filepath.Join(
	"..",
	"..",
	"..",
	"internal",
	"buftesting",
)

func TestGoogleapis(t *testing.T) {
	t.Parallel()
	image := testBuildGoogleapis(t, true)
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

	imageWithoutImports := bufimage.ImageWithoutImports(image)
	assert.Equal(t, buftesting.NumGoogleapisFiles, len(imageWithoutImports.Files()))
	imageWithoutImports = bufimage.ImageWithoutImports(imageWithoutImports)
	assert.Equal(t, buftesting.NumGoogleapisFiles, len(imageWithoutImports.Files()))

	imageWithSpecificNames, err := bufimage.ImageWithOnlyPathsAllowNotExist(
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
	imageWithoutImports = bufimage.ImageWithoutImports(imageWithSpecificNames)
	assert.Equal(
		t,
		[]string{
			"google/protobuf/api.proto",
			"google/protobuf/descriptor.proto",
			"google/type/date.proto",
		},
		testGetImageFilePaths(imageWithoutImports),
	)
	_, err = bufimage.ImageWithOnlyPaths(
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
	_, err = protosource.NewFilesUnstable(context.Background(), bufimageutil.NewInputFiles(image.Files())...)
	assert.NoError(t, err)
}

func TestCompareGoogleapis(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	t.Parallel()
	// cannot directly compare with source code info as buf protoc creates additional source
	// code infos that protoc does not
	image := testBuildGoogleapis(t, false)
	fileDescriptorSet := bufimage.ImageToFileDescriptorSet(image)
	actualProtocFileDescriptorSet := testBuildActualProtocGoogleapis(t, false)
	prototesting.AssertFileDescriptorSetsEqual(
		t,
		fileDescriptorSet,
		actualProtocFileDescriptorSet,
	)
}

func TestCompareCustomOptions1(t *testing.T) {
	t.Parallel()
	testCompare(t, "customoptions1")
}

func TestCompareProto3Optional1(t *testing.T) {
	t.Parallel()
	testCompare(t, "proto3optional1")
}

func TestCustomOptionsError1(t *testing.T) {
	t.Parallel()
	_, fileAnnotations := testBuild(t, false, filepath.Join("testdata", "customoptionserror1"))
	require.Equal(t, 1, len(fileAnnotations), fileAnnotations)
	require.Equal(
		t,
		"field a.Baz.bat: option (a.foo).bat: field bat of a.Foo does not exist",
		fileAnnotations[0].Message(),
	)
}

func TestOptionPanic(t *testing.T) {
	t.Parallel()
	require.NotPanics(t, func() {
		moduleFileSet := testGetModuleFileSet(t, filepath.Join("testdata", "optionpanic"))
		_, _, err := NewBuilder(zap.NewNop()).Build(
			context.Background(),
			moduleFileSet,
		)
		require.NoError(t, err)
	})
}

func testCompare(t *testing.T, relDirPath string) {
	t.Helper()
	dirPath := filepath.Join("testdata", relDirPath)
	image, fileAnnotations := testBuild(t, false, dirPath)
	require.Equal(t, 0, len(fileAnnotations), fileAnnotations)
	image = bufimage.ImageWithoutImports(image)
	fileDescriptorSet := bufimage.ImageToFileDescriptorSet(image)
	filePaths := buftesting.GetProtocFilePaths(t, dirPath, 0)
	actualProtocFileDescriptorSet := buftesting.GetActualProtocFileDescriptorSet(t, false, false, dirPath, filePaths)
	prototesting.AssertFileDescriptorSetsEqual(t, fileDescriptorSet, actualProtocFileDescriptorSet)
}

func testBuildGoogleapis(t *testing.T, includeSourceInfo bool) bufimage.Image {
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	image, fileAnnotations := testBuild(t, includeSourceInfo, googleapisDirPath)
	require.Equal(t, 0, len(fileAnnotations), fileAnnotations)
	return image
}

func testBuildActualProtocGoogleapis(t *testing.T, includeSourceInfo bool) *descriptorpb.FileDescriptorSet {
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	filePaths := buftesting.GetProtocFilePaths(t, googleapisDirPath, 0)
	fileDescriptorSet := buftesting.GetActualProtocFileDescriptorSet(t, true, includeSourceInfo, googleapisDirPath, filePaths)
	assert.Equal(t, buftesting.NumGoogleapisFilesWithImports, len(fileDescriptorSet.GetFile()))

	return fileDescriptorSet
}

func testBuild(t *testing.T, includeSourceInfo bool, dirPath string) (bufimage.Image, []bufanalysis.FileAnnotation) {
	moduleFileSet := testGetModuleFileSet(t, dirPath)
	var options []BuildOption
	if !includeSourceInfo {
		options = append(options, WithExcludeSourceCodeInfo())
	}
	image, fileAnnotations, err := NewBuilder(zap.NewNop()).Build(
		context.Background(),
		moduleFileSet,
		options...,
	)
	require.NoError(t, err)
	return image, fileAnnotations
}

func testGetModuleFileSet(t *testing.T, dirPath string) bufmodule.ModuleFileSet {
	readWriteBucket, err := storageos.NewReadWriteBucket(dirPath)
	require.NoError(t, err)
	config, err := bufmodulebuild.NewConfig(bufmodulebuild.ExternalConfig{})
	require.NoError(t, err)
	module, err := bufmodulebuild.NewModuleBucketBuilder(zap.NewNop()).BuildForBucket(
		context.Background(),
		readWriteBucket,
		config,
	)
	require.NoError(t, err)
	moduleFileSet, err := bufmodulebuild.NewModuleFileSetBuilder(
		zap.NewNop(),
		bufmodule.NewNopModuleReader(),
	).Build(
		context.Background(),
		module,
	)
	require.NoError(t, err)
	return moduleFileSet
}

func testGetImageFilePaths(image bufimage.Image) []string {
	var fileNames []string
	for _, file := range image.Files() {
		fileNames = append(fileNames, file.Path())
	}
	sort.Strings(fileNames)
	return fileNames
}

func testGetImageImportPaths(image bufimage.Image) []string {
	var importNames []string
	for _, file := range image.Files() {
		if file.IsImport() {
			importNames = append(importNames, file.Path())
		}
	}
	sort.Strings(importNames)
	return importNames
}

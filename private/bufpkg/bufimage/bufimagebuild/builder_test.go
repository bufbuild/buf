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

package bufimagebuild

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimageutil"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleconfig"
	"github.com/bufbuild/buf/private/bufpkg/buftesting"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/protosource"
	"github.com/bufbuild/buf/private/pkg/prototesting"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/testingext"
	"github.com/bufbuild/buf/private/pkg/thread"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var buftestingDirPath = filepath.Join(
	"..",
	"..",
	"buftesting",
)

func TestGoogleapis(t *testing.T) {
	testingext.SkipIfShort(t)
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
		nil,
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
	imageWithSpecificNames, err = bufimage.ImageWithOnlyPathsAllowNotExist(
		image,
		[]string{
			"google/protobuf/descriptor.proto",
			"google/protobuf/api.proto",
			"google/type",
			"google/foo",
		},
		nil,
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
			"google/protobuf/wrappers.proto",
			"google/type/calendar_period.proto",
			"google/type/color.proto",
			"google/type/date.proto",
			"google/type/dayofweek.proto",
			"google/type/expr.proto",
			"google/type/fraction.proto",
			"google/type/latlng.proto",
			"google/type/money.proto",
			"google/type/postal_address.proto",
			"google/type/quaternion.proto",
			"google/type/timeofday.proto",
		},
		testGetImageFilePaths(imageWithSpecificNames),
	)
	imageWithoutImports = bufimage.ImageWithoutImports(imageWithSpecificNames)
	assert.Equal(
		t,
		[]string{
			"google/protobuf/api.proto",
			"google/protobuf/descriptor.proto",
			"google/type/calendar_period.proto",
			"google/type/color.proto",
			"google/type/date.proto",
			"google/type/dayofweek.proto",
			"google/type/expr.proto",
			"google/type/fraction.proto",
			"google/type/latlng.proto",
			"google/type/money.proto",
			"google/type/postal_address.proto",
			"google/type/quaternion.proto",
			"google/type/timeofday.proto",
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
		nil,
	)
	assert.Equal(t, errors.New(`path "google/foo/nonsense.proto" has no matching file in the image`), err)
	_, err = bufimage.ImageWithOnlyPaths(
		image,
		[]string{
			"google/protobuf/descriptor.proto",
			"google/protobuf/api.proto",
			"google/type/date.proto",
			"google/foo",
		},
		nil,
	)
	assert.Equal(t, errors.New(`path "google/foo" has no matching file in the image`), err)

	imageWithPathsAndExcludes, err := bufimage.ImageWithOnlyPaths(
		image,
		[]string{
			"google/type",
		},
		[]string{
			"google/type/calendar_period.proto",
			"google/type/date.proto",
		},
	)
	assert.NoError(t, err)
	assert.Equal(t,
		[]string{
			"google/protobuf/wrappers.proto",
			"google/type/color.proto",
			"google/type/dayofweek.proto",
			"google/type/expr.proto",
			"google/type/fraction.proto",
			"google/type/latlng.proto",
			"google/type/money.proto",
			"google/type/postal_address.proto",
			"google/type/quaternion.proto",
			"google/type/timeofday.proto",
		},
		testGetImageFilePaths(imageWithPathsAndExcludes),
	)

	excludePaths := []string{
		"google/type/calendar_period.proto",
		"google/type/quaternion.proto",
		"google/type/money.proto",
		"google/type/color.proto",
		"google/type/date.proto",
	}
	imageWithExcludes, err := bufimage.ImageWithOnlyPaths(image, []string{}, excludePaths)
	assert.NoError(t, err)
	testImageWithExcludedFilePaths(t, imageWithExcludes, excludePaths)

	assert.Equal(t, buftesting.NumGoogleapisFilesWithImports, len(image.Files()))
	// basic check to make sure there is no error at this scale
	_, err = protosource.NewFilesUnstable(context.Background(), bufimageutil.NewInputFiles(image.Files())...)
	assert.NoError(t, err)
}

func TestCompareCustomOptions1(t *testing.T) {
	t.Parallel()
	runner := command.NewRunner()
	testCompare(t, runner, "customoptions1")
}

func TestCompareProto3Optional1(t *testing.T) {
	t.Parallel()
	runner := command.NewRunner()
	testCompare(t, runner, "proto3optional1")
}

func TestCompareTrailingComments(t *testing.T) {
	t.Parallel()
	runner := command.NewRunner()
	testCompare(t, runner, "trailingcomments")
}

func TestCustomOptionsError1(t *testing.T) {
	t.Parallel()
	testFileAnnotations(
		t,
		"customoptionserror1",
		filepath.FromSlash("testdata/customoptionserror1/b.proto:9:27:field a.Baz.bat: option (a.foo).bat: field bat of a.Foo does not exist"),
	)
}

func TestNotAMessageType(t *testing.T) {
	t.Parallel()
	testFileAnnotations(
		t,
		"notamessagetype",
		filepath.FromSlash("testdata/notamessagetype/a.proto:9:11:method a.MyService.Foo: invalid request type: a.MyService.Foo is a method, not a message"),
	)
}

func TestSpaceBetweenNumberAndID(t *testing.T) {
	t.Parallel()
	testFileAnnotations(
		t,
		"spacebetweennumberandid",
		filepath.FromSlash("testdata/spacebetweennumberandid/a.proto:6:14:invalid syntax in integer value: 10to"),
		filepath.FromSlash("testdata/spacebetweennumberandid/a.proto:6:14:syntax error: unexpected error, expecting int literal"),
	)
}

func TestCyclicImport(t *testing.T) {
	t.Parallel()
	testFileAnnotations(
		t,
		"cyclicimport",
		// Since the compiler is multi-threaded, order of file compilation can happen one of two ways
		fmt.Sprintf(`%s:5:8:cycle found in imports: "a/a.proto" -> "b/b.proto" -> "a/a.proto"
				|| %s:5:8:cycle found in imports: "b/b.proto" -> "a/a.proto" -> "b/b.proto"`,
			filepath.FromSlash("testdata/cyclicimport/a/a.proto"),
			filepath.FromSlash("testdata/cyclicimport/b/b.proto"),
		),
	)
}

func TestDuplicateSyntheticOneofs(t *testing.T) {
	// https://github.com/bufbuild/buf/issues/1071
	t.Parallel()
	testFileAnnotations(
		t,
		"duplicatesyntheticoneofs",
		// Since the compiler is multi-threaded, order of file compilation can happen one of two ways
		filepath.FromSlash(`testdata/duplicatesyntheticoneofs/a1.proto:5:9:symbol "a.Foo" already defined at a2.proto:5:9
				|| testdata/duplicatesyntheticoneofs/a2.proto:5:9:symbol "a.Foo" already defined at a1.proto:5:9`),
		filepath.FromSlash(`testdata/duplicatesyntheticoneofs/a1.proto:6:19:symbol "a.Foo._bar" already defined at a2.proto:6:19
				|| testdata/duplicatesyntheticoneofs/a2.proto:6:19:symbol "a.Foo._bar" already defined at a1.proto:6:19`),
		filepath.FromSlash(`testdata/duplicatesyntheticoneofs/a1.proto:6:19:symbol "a.Foo.bar" already defined at a2.proto:6:19
				|| testdata/duplicatesyntheticoneofs/a2.proto:6:19:symbol "a.Foo.bar" already defined at a1.proto:6:19`),
	)
}

func TestOptionPanic(t *testing.T) {
	t.Parallel()
	require.NotPanics(t, func() {
		module := testGetModule(t, filepath.Join("testdata", "optionpanic"))
		_, _, err := NewBuilder(zap.NewNop(), bufmodule.NewNopModuleReader()).Build(
			context.Background(),
			module,
		)
		require.NoError(t, err)
	})
}

func TestCompareSemicolons(t *testing.T) {
	t.Parallel()
	runner := command.NewRunner()
	testCompare(t, runner, "semicolons")
}

func testCompare(t *testing.T, runner command.Runner, relDirPath string) {
	dirPath := filepath.Join("testdata", relDirPath)
	image, fileAnnotations := testBuild(t, false, dirPath)
	require.Equal(t, 0, len(fileAnnotations), fileAnnotations)
	image = bufimage.ImageWithoutImports(image)
	fileDescriptorSet := bufimage.ImageToFileDescriptorSet(image)
	filePaths := buftesting.GetProtocFilePaths(t, dirPath, 0)
	actualProtocFileDescriptorSet := buftesting.GetActualProtocFileDescriptorSet(t, runner, false, false, dirPath, filePaths)
	prototesting.AssertFileDescriptorSetsEqual(t, runner, fileDescriptorSet, actualProtocFileDescriptorSet)
}

func testBuildGoogleapis(t *testing.T, includeSourceInfo bool) bufimage.Image {
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	image, fileAnnotations := testBuild(t, includeSourceInfo, googleapisDirPath)
	require.Equal(t, 0, len(fileAnnotations), fileAnnotations)
	return image
}

func testBuild(t *testing.T, includeSourceInfo bool, dirPath string) (bufimage.Image, []bufanalysis.FileAnnotation) {
	module := testGetModule(t, dirPath)
	var options []BuildOption
	if !includeSourceInfo {
		options = append(options, WithExcludeSourceCodeInfo())
	}
	image, fileAnnotations, err := NewBuilder(zap.NewNop(), bufmodule.NewNopModuleReader()).Build(
		context.Background(),
		module,
		options...,
	)
	require.NoError(t, err)
	return image, fileAnnotations
}

func testGetModule(t *testing.T, dirPath string) bufmodule.Module {
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(
		dirPath,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	require.NoError(t, err)
	config, err := bufmoduleconfig.NewConfigV1(bufmoduleconfig.ExternalConfigV1{})
	require.NoError(t, err)
	module, err := bufmodulebuild.NewModuleBucketBuilder().BuildForBucket(
		context.Background(),
		readWriteBucket,
		config,
	)
	require.NoError(t, err)
	return module
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

func testFileAnnotations(t *testing.T, relDirPath string, want ...string) {
	t.Helper()

	// Allowing real parallelism makes some test expectations too complicated to express and assert
	previousParallelism := thread.Parallelism()
	thread.SetParallelism(1)
	defer func() {
		thread.SetParallelism(previousParallelism)
	}()

	_, fileAnnotations := testBuild(t, false, filepath.Join("testdata", filepath.FromSlash(relDirPath)))
	got := make([]string, len(fileAnnotations))
	for i, annotation := range fileAnnotations {
		got[i] = annotation.String()
	}
	require.Equal(t, len(want), len(got))
	for i := range want {
		options := strings.Split(want[i], "||")
		matched := false
		for _, option := range options {
			option = strings.TrimSpace(option)
			if got[i] == option {
				matched = true
				break
			}
		}
		require.True(t, matched, "annotation at index %d: wanted %q ; got %q", i, want[i], got[i])
	}
}

func testImageWithExcludedFilePaths(t *testing.T, image bufimage.Image, excludePaths []string) {
	t.Helper()
	for _, imageFile := range image.Files() {
		if !imageFile.IsImport() {
			for _, excludePath := range excludePaths {
				assert.False(t, normalpath.EqualsOrContainsPath(excludePath, imageFile.Path(), normalpath.Relative), "paths: %s, %s", imageFile.Path(), excludePath)
			}
		}
	}
}

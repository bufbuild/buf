// Copyright 2020-2026 Buf Technologies, Inc.
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

package bufimage_test

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"buf.build/go/standard/xtesting"
	"github.com/bufbuild/buf/private/buf/buftesting"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/prototesting"
	"github.com/bufbuild/buf/private/pkg/slogtestext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var buftestingDirPath = filepath.Join(
	"..",
	"..",
	"buf",
	"buftesting",
)

func TestGoogleapis(t *testing.T) {
	xtesting.SkipIfShort(t)
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
	_, err = bufprotosource.NewFiles(t.Context(), image.Files(), image.Resolver())
	assert.NoError(t, err)
}

func TestCompareCustomOptions1(t *testing.T) {
	t.Parallel()
	testCompare(t, "customoptions1")
}

func TestCompareProto3Optional1(t *testing.T) {
	t.Parallel()
	testCompare(t, "proto3optional1")
}

func TestCompareTrailingComments(t *testing.T) {
	t.Parallel()
	testCompare(t, "trailingcomments")
}

func TestCustomOptionsError1(t *testing.T) {
	t.Parallel()
	testFileAnnotations(
		t,
		"customoptionserror1",
		true,
		filepath.FromSlash("testdata/customoptionserror1/b.proto:9:26:cannot find message field `bat` in `a.Foo`"),
	)
}

func TestNotAMessageType(t *testing.T) {
	t.Parallel()
	testFileAnnotations(
		t,
		"notamessagetype",
		true,
		filepath.FromSlash("testdata/notamessagetype/a.proto:9:11:expected message type, found service method `a.MyService.Foo`"),
	)
}

func TestSpaceBetweenNumberAndID(t *testing.T) {
	t.Parallel()
	testFileAnnotations(
		t,
		"spacebetweennumberandid",
		true,
		filepath.FromSlash("testdata/spacebetweennumberandid/a.proto:6:14:field number out of range"),
		filepath.FromSlash("testdata/spacebetweennumberandid/a.proto:6:16:invalid digit in decimal integer literal"),
		filepath.FromSlash("testdata/spacebetweennumberandid/a.proto:6:19:unexpected `max` in extension range"),
	)
}

func TestCyclicImport(t *testing.T) {
	t.Parallel()
	testFileAnnotations(
		t,
		"cyclicimport",
		false,
		fmt.Sprintf(
			`%s:5:1:detected cyclic import while importing "a/a.proto"`,
			filepath.FromSlash("testdata/cyclicimport/b/b.proto"),
		),
	)
}

func TestDuplicateSyntheticOneofs(t *testing.T) {
	// https://github.com/bufbuild/buf/issues/1071
	//
	// However, this issue no longer applies to the new compiler, since the new compiler does
	// not produce a symbol for the synthetic one-of types for its IR (these are generated
	// for the file descriptor set on-demand). It also only surfaces duplicate symbol fqn's
	// for the highest level, which is the message in the case of this test.
	t.Parallel()
	testFileAnnotations(
		t,
		"duplicatesyntheticoneofs",
		false,
		filepath.FromSlash("testdata/duplicatesyntheticoneofs/a1.proto:5:9:`Foo` declared multiple times"),
	)
}

func TestOptionPanic(t *testing.T) {
	t.Parallel()
	require.NotPanics(t, func() {
		moduleSet, err := bufmoduletesting.NewModuleSetForDirPath(filepath.Join("testdata", "optionpanic"))
		require.NoError(t, err)
		_, err = bufimage.BuildImage(
			t.Context(),
			slogtestext.NewLogger(t),
			bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(moduleSet),
		)
		require.NoError(t, err)
	})
}

func TestCompareSemicolons(t *testing.T) {
	t.Parallel()
	testCompare(t, "semicolons")
}

func TestModuleTargetFiles(t *testing.T) {
	t.Parallel()
	moduleSet, err := bufmoduletesting.NewModuleSet(
		bufmoduletesting.ModuleData{
			Name: "buf.build/foo/a",
			PathToData: map[string][]byte{
				"a.proto": []byte(
					`syntax = "proto3"; package a; import "b.proto";`,
				),
			},
		},
		bufmoduletesting.ModuleData{
			Name: "buf.build/foo/b",
			PathToData: map[string][]byte{
				"b.proto": []byte(
					`syntax = "proto3"; package b; import "c.proto";`,
				),
			},
		},
		bufmoduletesting.ModuleData{
			Name: "buf.build/foo/c",
			PathToData: map[string][]byte{
				"c.proto": []byte(
					`syntax = "proto3"; package c;`,
				),
			},
		},
	)
	require.NoError(t, err)
	testTargetImageFiles := func(t *testing.T, want []string, opaqueID ...string) {
		targetModuleSet := moduleSet
		if len(opaqueID) > 0 {
			var err error
			targetModuleSet, err = moduleSet.WithTargetOpaqueIDs(opaqueID...)
			require.NoError(t, err)
		}
		image, err := bufimage.BuildImage(
			t.Context(),
			slogtestext.NewLogger(t),
			bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(targetModuleSet),
		)
		require.NoError(t, err)
		assert.Equal(t, want, testGetImageFilePaths(image))
	}
	testTargetImageFiles(t, []string{"a.proto", "b.proto", "c.proto"})
	testTargetImageFiles(t, []string{"a.proto", "b.proto", "c.proto"}, "buf.build/foo/a")
	testTargetImageFiles(t, []string{"b.proto", "c.proto"}, "buf.build/foo/b")
	testTargetImageFiles(t, []string{"c.proto"}, "buf.build/foo/c")
	testTargetImageFiles(t, []string{"b.proto", "c.proto"}, "buf.build/foo/b", "buf.build/foo/c")
}

func testCompare(t *testing.T, relDirPath string) {
	dirPath := filepath.Join("testdata", relDirPath)
	image, fileAnnotations := testBuild(t, false, dirPath, false)
	require.Equal(t, 0, len(fileAnnotations), fileAnnotations)
	image = bufimage.ImageWithoutImports(image)
	fileDescriptorSet := bufimage.ImageToFileDescriptorSet(image)
	filePaths := buftesting.GetProtocFilePaths(t, dirPath, 0)
	actualProtocFileDescriptorSet := buftesting.GetActualProtocFileDescriptorSet(t, false, false, dirPath, filePaths)
	prototesting.AssertFileDescriptorSetsEqual(t, fileDescriptorSet, actualProtocFileDescriptorSet)
}

func testBuildGoogleapis(t *testing.T, includeSourceInfo bool) bufimage.Image {
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	image, fileAnnotations := testBuild(t, includeSourceInfo, googleapisDirPath, true)
	require.Equal(t, 0, len(fileAnnotations), fileAnnotations)
	return image
}

func testBuild(t *testing.T, includeSourceInfo bool, dirPath string, parallelism bool) (bufimage.Image, []bufanalysis.FileAnnotation) {
	moduleSet, err := bufmoduletesting.NewModuleSetForDirPath(dirPath)
	require.NoError(t, err)
	var options []bufimage.BuildImageOption
	if !includeSourceInfo {
		options = append(options, bufimage.WithExcludeSourceCodeInfo())
	}
	if !parallelism {
		options = append(options, bufimage.WithNoParallelism())
	}
	image, err := bufimage.BuildImage(
		t.Context(),
		slogtestext.NewLogger(t),
		bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(moduleSet),
		options...,
	)
	var fileAnnotationSet bufanalysis.FileAnnotationSet
	if errors.As(err, &fileAnnotationSet) {
		return image, fileAnnotationSet.FileAnnotations()
	}
	require.NoError(t, err)
	return image, nil
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

func testFileAnnotations(t *testing.T, relDirPath string, parallelism bool, want ...string) {
	t.Helper()

	_, fileAnnotations := testBuild(t, false, filepath.Join("testdata", filepath.FromSlash(relDirPath)), parallelism)
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

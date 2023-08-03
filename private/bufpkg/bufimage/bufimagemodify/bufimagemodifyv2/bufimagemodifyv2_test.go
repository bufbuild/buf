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

package bufimagemodifyv2

import (
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal/bufimagemodifytesting"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestModifyJavaPackage(t *testing.T) {
	t.Parallel()
	baseDir := filepath.Join("..", "testdata")
	tests := []struct {
		description     string
		subDir          string
		file            string
		modifyOptions   []ModifyJavaPackageOption
		expectedValue   string
		shouldNotModify bool
	}{
		{
			description:   "Modify Java Package with prefix on file with this option but without a proto package",
			subDir:        "alloptions",
			file:          "a.proto",
			modifyOptions: []ModifyJavaPackageOption{ModifyJavaPackageWithPrefix("prefix")},
			// The orignal value from the file should be preserved because this file
			// has no proto package and we cannot resolve a proper value for java_package.
			expectedValue:   "foo",
			shouldNotModify: true,
		},
		{
			description:   "Modify Java Package with value on file with this option but without a proto package",
			subDir:        "alloptions",
			file:          "a.proto",
			modifyOptions: []ModifyJavaPackageOption{ModifyJavaPackageWithValue("com.example")},
			expectedValue: "com.example",
		},
		{
			description: "Modify Java Package with prefix and suffix on file without a proto package",
			subDir:      "alloptions",
			file:        "a.proto",
			modifyOptions: []ModifyJavaPackageOption{
				ModifyJavaPackageWithPrefix("prefix"),
				ModifyJavaPackageWithSuffix("suffix"),
			},
			// The orignal value from the file should be preserved because this file
			// has no proto package and we cannot resolve a proper value for java_package.
			expectedValue:   "foo",
			shouldNotModify: true,
		},
		{
			description: "Modify Java Package without override on file without a proto package",
			subDir:      "alloptions",
			file:        "a.proto",
			// The orignal value from the file should be preserved because this file
			// has no proto package and we cannot resolve a proper value for java_package.
			modifyOptions:   []ModifyJavaPackageOption{},
			expectedValue:   "foo",
			shouldNotModify: true,
		},
		{
			description:     "Modify Java Package with override value the same as java package",
			subDir:          "javaoptions",
			file:            "java_file.proto",
			modifyOptions:   []ModifyJavaPackageOption{ModifyJavaPackageWithValue("foo")},
			expectedValue:   "foo",
			shouldNotModify: true,
		},
		{
			description:   "Modify Java Package with value on file with a proto package",
			subDir:        "javaoptions",
			file:          "java_file.proto",
			modifyOptions: []ModifyJavaPackageOption{ModifyJavaPackageWithValue("bar")},
			expectedValue: "bar",
		},
		{
			description:   "Modify Java Package without override on a file without option but with a proto package",
			subDir:        "javaemptyoptions",
			file:          "a.proto",
			modifyOptions: []ModifyJavaPackageOption{},
			expectedValue: "foo",
		},
		{
			description:   "Modify Java Package with prefix on file with a proto package",
			subDir:        "javaoptions",
			file:          "java_file.proto",
			modifyOptions: []ModifyJavaPackageOption{ModifyJavaPackageWithPrefix("prefix.override")},
			expectedValue: "prefix.override.acme.weather",
		},
		{
			description: "Modify Java Package with prefix and suffix on file with a proto package",
			subDir:      "javaoptions",
			file:        "java_file.proto",
			modifyOptions: []ModifyJavaPackageOption{
				ModifyJavaPackageWithPrefix("prefix.override"),
				ModifyJavaPackageWithSuffix("override.suffix"),
			},
			expectedValue: "prefix.override.acme.weather.override.suffix",
		},
		{
			description:   "Modify Java Package with suffix on file with a proto package",
			subDir:        "javaoptions",
			file:          "java_file.proto",
			modifyOptions: []ModifyJavaPackageOption{ModifyJavaPackageWithSuffix("override.suffix")},
			expectedValue: "acme.weather.override.suffix",
		},
		{
			description: "Modify Java Package with prefix and suffix on a wkt file",
			subDir:      "wktimport",
			file:        "google/protobuf/timestamp.proto",
			modifyOptions: []ModifyJavaPackageOption{
				ModifyJavaPackageWithPrefix("prefix.override"),
				ModifyJavaPackageWithSuffix("override.suffix"),
			},
			expectedValue:   "com.google.protobuf",
			shouldNotModify: true,
		},
		{
			description:     "Modify Java Package with value on a wkt file",
			subDir:          "wktimport",
			file:            "google/protobuf/timestamp.proto",
			modifyOptions:   []ModifyJavaPackageOption{ModifyJavaPackageWithValue("value")},
			expectedValue:   "com.google.protobuf",
			shouldNotModify: true,
		},
	}
	for _, test := range tests {
		test := test
		for _, includeSourceCodeInfo := range []bool{true, false} {
			includeSourceCodeInfo := includeSourceCodeInfo
			t.Run(test.description, func(t *testing.T) {
				t.Parallel()
				dirPath := filepath.Join(baseDir, test.subDir)
				image := bufimagemodifytesting.GetTestImage(
					t,
					dirPath,
					includeSourceCodeInfo,
				)
				markSweeper := newMarkSweeper(image)
				require.NotNil(t, markSweeper)
				imageFile := image.GetFile(test.file)
				require.NotNil(t, imageFile)
				_, ok := markSweeper.sourceCodeInfoPaths[imageFile.Path()]
				require.False(t, ok)
				err := ModifyJavaPackage(markSweeper, imageFile, test.modifyOptions...)
				require.NoError(t, err)
				err = markSweeper.Sweep()
				require.NoError(t, err)
				require.NotNil(t, imageFile.Proto())
				require.Equal(
					t,
					test.expectedValue,
					imageFile.Proto().GetOptions().GetJavaPackage(),
				)
				fileKeys, ok := markSweeper.sourceCodeInfoPaths[imageFile.Path()]
				if test.shouldNotModify {
					require.False(t, ok)
					require.Equal(
						t,
						bufimagemodifytesting.GetTestImage(
							t,
							dirPath,
							includeSourceCodeInfo,
						),
						image,
					)
					return
				}
				require.True(t, ok)
				_, ok = fileKeys[internal.GetPathKey(internal.JavaPackagePath)]
				require.True(t, ok)
			})
		}
	}
}

func TestModifyJavaMultipleFiles(t *testing.T) {
	t.Parallel()
	baseDir := filepath.Join("..", "testdata")
	tests := []struct {
		description     string
		subDir          string
		file            string
		override        bool
		expectedValue   bool
		shouldNotModify bool
	}{
		{
			description:   "java multiple files on a file without this option",
			subDir:        "emptyoptions",
			file:          "a.proto",
			override:      true,
			expectedValue: true,
		},
		{
			description:     "java multiple files to default value on a file without this option",
			subDir:          "emptyoptions",
			file:            "a.proto",
			override:        false,
			expectedValue:   false,
			shouldNotModify: true,
		},
		{
			description:   "java multiple files on a file with this option",
			subDir:        "alloptions",
			file:          "a.proto",
			override:      true,
			expectedValue: true,
		},
		{
			description:     "java multiple files with override on a file with this option equal to the same value",
			subDir:          "alloptions",
			file:            "a.proto",
			override:        false,
			expectedValue:   false,
			shouldNotModify: true,
		},
		{
			description:     "java multiple files with override on a wkt",
			subDir:          "wktimport",
			file:            "google/protobuf/timestamp.proto",
			override:        false,
			expectedValue:   true,
			shouldNotModify: true,
		},
	}
	for _, test := range tests {
		test := test
		for _, includeSourceCodeInfo := range []bool{true, false} {
			includeSourceCodeInfo := includeSourceCodeInfo
			t.Run(test.description, func(t *testing.T) {
				t.Parallel()
				dirPath := filepath.Join(baseDir, test.subDir)
				image := bufimagemodifytesting.GetTestImage(
					t,
					dirPath,
					includeSourceCodeInfo,
				)
				markSweeper := newMarkSweeper(image)
				require.NotNil(t, markSweeper)
				imageFile := image.GetFile(test.file)
				require.NotNil(t, imageFile)
				_, ok := markSweeper.sourceCodeInfoPaths[imageFile.Path()]
				require.False(t, ok)
				err := ModifyJavaMultipleFiles(
					markSweeper,
					imageFile,
					test.override,
				)
				require.NoError(t, err)
				err = markSweeper.Sweep()
				require.NoError(t, err)
				require.NotNil(t, imageFile.Proto())
				require.Equal(t, test.expectedValue, imageFile.Proto().GetOptions().GetJavaMultipleFiles())
				fileKeys, ok := markSweeper.sourceCodeInfoPaths[imageFile.Path()]
				if test.shouldNotModify {
					require.False(t, ok)
					require.Equal(
						t,
						bufimagemodifytesting.GetTestImage(
							t,
							dirPath,
							includeSourceCodeInfo,
						),
						image,
					)
					return
				}
				require.True(t, ok)
				_, ok = fileKeys[internal.GetPathKey(internal.JavaMultipleFilesPath)]
				require.True(t, ok)
			})
		}
	}
}

func TestModifyJavaOuterClassnmae(t *testing.T) {
	t.Parallel()
	baseDir := filepath.Join("..", "testdata")
	tests := []struct {
		description     string
		subDir          string
		file            string
		modifyOptions   []ModifyJavaOuterClassnameOption
		expectedValue   string
		shouldNotModify bool
	}{
		{
			description:   "java outer classname without override on a file without this option",
			subDir:        "emptyoptions",
			file:          "a.proto",
			modifyOptions: []ModifyJavaOuterClassnameOption{},
			expectedValue: "AProto",
		},
		{
			description:   "java outer classname with override on a file with this option",
			subDir:        "alloptions",
			file:          "a.proto",
			modifyOptions: []ModifyJavaOuterClassnameOption{ModifyJavaOuterClassnameWithValue("OverrideProto")},
			expectedValue: "OverrideProto",
		},
		{
			description:     "java outer classname with override on a file with this option equal to the same value",
			subDir:          "alloptions",
			file:            "a.proto",
			modifyOptions:   []ModifyJavaOuterClassnameOption{ModifyJavaOuterClassnameWithValue("foo")},
			expectedValue:   "foo",
			shouldNotModify: true,
		},
		{
			description:     "java outer classname with override on a wkt",
			subDir:          "wktimport",
			file:            "google/protobuf/timestamp.proto",
			modifyOptions:   []ModifyJavaOuterClassnameOption{ModifyJavaOuterClassnameWithValue("foo")},
			expectedValue:   "TimestampProto",
			shouldNotModify: true,
		},
	}
	for _, test := range tests {
		test := test
		for _, includeSourceCodeInfo := range []bool{true, false} {
			includeSourceCodeInfo := includeSourceCodeInfo
			t.Run(test.description, func(t *testing.T) {
				t.Parallel()
				dirPath := filepath.Join(baseDir, test.subDir)
				image := bufimagemodifytesting.GetTestImage(
					t,
					dirPath,
					includeSourceCodeInfo,
				)
				markSweeper := newMarkSweeper(image)
				require.NotNil(t, markSweeper)
				imageFile := image.GetFile(test.file)
				require.NotNil(t, imageFile)
				_, ok := markSweeper.sourceCodeInfoPaths[imageFile.Path()]
				require.False(t, ok)
				err := ModifyJavaOuterClassname(
					markSweeper,
					imageFile,
					test.modifyOptions...,
				)
				require.NoError(t, err)
				err = markSweeper.Sweep()
				require.NoError(t, err)
				require.NotNil(t, imageFile.Proto())
				require.Equal(t, test.expectedValue, imageFile.Proto().GetOptions().GetJavaOuterClassname())
				fileKeys, ok := markSweeper.sourceCodeInfoPaths[imageFile.Path()]
				if test.shouldNotModify {
					require.False(t, ok)
					require.Equal(
						t,
						bufimagemodifytesting.GetTestImage(
							t,
							dirPath,
							includeSourceCodeInfo,
						),
						image,
					)
					return
				}
				require.True(t, ok)
				_, ok = fileKeys[internal.GetPathKey(internal.JavaOuterClassnamePath)]
				require.True(t, ok)
			})
		}
	}
}

func TestModifyJavaStringCheckUtf8(t *testing.T) {
	t.Parallel()
	baseDir := filepath.Join("..", "testdata")
	tests := []struct {
		description     string
		subDir          string
		file            string
		override        bool
		expectedValue   bool
		shouldNotModify bool
	}{
		{
			description:   "java string check utf8 on a file without this option",
			subDir:        "emptyoptions",
			file:          "a.proto",
			override:      true,
			expectedValue: true,
		},
		{
			description:     "java string check utf8 to default value on a file without this option",
			subDir:          "emptyoptions",
			file:            "a.proto",
			override:        false,
			expectedValue:   false,
			shouldNotModify: true,
		},
		{
			description:   "java string check utf8 on a file with this option",
			subDir:        "alloptions",
			file:          "a.proto",
			override:      true,
			expectedValue: true,
		},
		{
			description:     "java string check utf8 with override on a file with this option equal to the same value",
			subDir:          "alloptions",
			file:            "a.proto",
			override:        false,
			expectedValue:   false,
			shouldNotModify: true,
		},
		{
			description:     "java string check utf8 with override on a wkt",
			subDir:          "wktimport",
			file:            "google/protobuf/timestamp.proto",
			override:        true,
			expectedValue:   false,
			shouldNotModify: true,
		},
	}
	for _, test := range tests {
		test := test
		for _, includeSourceCodeInfo := range []bool{true, false} {
			includeSourceCodeInfo := includeSourceCodeInfo
			t.Run(test.description, func(t *testing.T) {
				t.Parallel()
				dirPath := filepath.Join(baseDir, test.subDir)
				image := bufimagemodifytesting.GetTestImage(
					t,
					dirPath,
					includeSourceCodeInfo,
				)
				markSweeper := newMarkSweeper(image)
				require.NotNil(t, markSweeper)
				imageFile := image.GetFile(test.file)
				require.NotNil(t, imageFile)
				_, ok := markSweeper.sourceCodeInfoPaths[imageFile.Path()]
				require.False(t, ok)
				err := ModifyJavaStringCheckUtf8(
					markSweeper,
					imageFile,
					test.override,
				)
				require.NoError(t, err)
				err = markSweeper.Sweep()
				require.NoError(t, err)
				require.NotNil(t, imageFile.Proto())
				require.Equal(t, test.expectedValue, imageFile.Proto().GetOptions().GetJavaStringCheckUtf8())
				fileKeys, ok := markSweeper.sourceCodeInfoPaths[imageFile.Path()]
				if test.shouldNotModify {
					require.False(t, ok)
					require.Equal(
						t,
						bufimagemodifytesting.GetTestImage(
							t,
							dirPath,
							includeSourceCodeInfo,
						),
						image,
					)
					return
				}
				require.True(t, ok)
				_, ok = fileKeys[internal.GetPathKey(internal.JavaStringCheckUtf8Path)]
				require.True(t, ok)
			})
		}
	}
}

func TestModifyGoPackage(t *testing.T) {
	t.Parallel()
	baseDir := filepath.Join("..", "testdata")
	tests := []struct {
		description     string
		subDir          string
		file            string
		modifyOption    ModifyGoPackageOption
		expectedValue   string
		shouldNotModify bool
	}{
		{
			description:   "go package prefix on a file without this option without a package",
			subDir:        "emptyoptions",
			file:          "a.proto",
			modifyOption:  ModifyGoPackageWithPrefix("override/prefix"),
			expectedValue: "override/prefix",
		},
		{
			description:   "go package prefix on a file with this option without a package",
			subDir:        "alloptions",
			file:          "a.proto",
			modifyOption:  ModifyGoPackageWithPrefix("override/prefix"),
			expectedValue: "override/prefix",
		},
		{
			description:   "go package prefix on a file without this option but with a package",
			subDir:        "wktimport",
			file:          "a.proto",
			modifyOption:  ModifyGoPackageWithPrefix("override/prefix"),
			expectedValue: "override/prefix;weatherv1alpha1",
		},
		{
			description:     "go package prefix on a wkt",
			subDir:          "wktimport",
			file:            "google/protobuf/timestamp.proto",
			modifyOption:    ModifyGoPackageWithPrefix("override/prefix"),
			expectedValue:   "google.golang.org/protobuf/types/known/timestamppb",
			shouldNotModify: true,
		},
		{
			description:   "go package on a file without this option",
			subDir:        "emptyoptions",
			file:          "a.proto",
			modifyOption:  ModifyGoPackageWithValue("override/value"),
			expectedValue: "override/value",
		},
		{
			description:   "go package on a file with this option",
			subDir:        "alloptions",
			file:          "a.proto",
			modifyOption:  ModifyGoPackageWithValue("override/value"),
			expectedValue: "override/value",
		},
		{
			description:     "go package on a file with this option with equal value",
			subDir:          "alloptions",
			file:            "a.proto",
			modifyOption:    ModifyGoPackageWithValue("foo"),
			expectedValue:   "foo",
			shouldNotModify: true,
		},
		{
			description:     "go package on a wkt",
			subDir:          "wktimport",
			file:            "google/protobuf/timestamp.proto",
			modifyOption:    ModifyGoPackageWithValue("override/value"),
			expectedValue:   "google.golang.org/protobuf/types/known/timestamppb",
			shouldNotModify: true,
		},
	}
	for _, test := range tests {
		test := test
		for _, includeSourceCodeInfo := range []bool{true, false} {
			includeSourceCodeInfo := includeSourceCodeInfo
			t.Run(test.description, func(t *testing.T) {
				t.Parallel()
				dirPath := filepath.Join(baseDir, test.subDir)
				image := bufimagemodifytesting.GetTestImage(
					t,
					dirPath,
					includeSourceCodeInfo,
				)
				markSweeper := newMarkSweeper(image)
				require.NotNil(t, markSweeper)
				imageFile := image.GetFile(test.file)
				require.NotNil(t, imageFile)
				_, ok := markSweeper.sourceCodeInfoPaths[imageFile.Path()]
				require.False(t, ok)
				err := ModifyGoPackage(markSweeper, imageFile, test.modifyOption)
				require.NoError(t, err)
				err = markSweeper.Sweep()
				require.NoError(t, err)
				require.NotNil(t, imageFile.Proto())
				require.Equal(
					t,
					test.expectedValue,
					imageFile.Proto().GetOptions().GetGoPackage(),
				)
				fileKeys, ok := markSweeper.sourceCodeInfoPaths[imageFile.Path()]
				if test.shouldNotModify {
					require.False(t, ok)
					require.Equal(
						t,
						bufimagemodifytesting.GetTestImage(
							t,
							dirPath,
							includeSourceCodeInfo,
						),
						image,
					)
					return
				}
				require.True(t, ok)
				_, ok = fileKeys[internal.GetPathKey(internal.GoPackagePath)]
				require.True(t, ok)
			})
		}
	}
}

func TestModifyOptimizeFor(t *testing.T) {
	t.Parallel()
	baseDir := filepath.Join("..", "testdata")
	tests := []struct {
		description     string
		subDir          string
		file            string
		override        descriptorpb.FileOptions_OptimizeMode
		expectedValue   descriptorpb.FileOptions_OptimizeMode
		shouldNotModify bool
	}{
		{
			description:   "optimize for on a file without this option",
			subDir:        "emptyoptions",
			file:          "a.proto",
			override:      descriptorpb.FileOptions_CODE_SIZE,
			expectedValue: descriptorpb.FileOptions_CODE_SIZE,
		},
		{
			description:     "optimize for with default value on a file without this option",
			subDir:          "emptyoptions",
			file:            "a.proto",
			override:        descriptorpb.FileOptions_SPEED,
			expectedValue:   descriptorpb.FileOptions_SPEED,
			shouldNotModify: true,
		},
		{
			description:   "optimize for on a file without this option",
			subDir:        "alloptions",
			file:          "a.proto",
			override:      descriptorpb.FileOptions_CODE_SIZE,
			expectedValue: descriptorpb.FileOptions_CODE_SIZE,
		},
		{
			description:     "optimize for on a file with this option with equal value",
			subDir:          "alloptions",
			file:            "a.proto",
			override:        descriptorpb.FileOptions_SPEED,
			expectedValue:   descriptorpb.FileOptions_SPEED,
			shouldNotModify: true,
		},
		{
			description:     "optmize for on a wkt",
			subDir:          "wktimport",
			file:            "google/protobuf/timestamp.proto",
			override:        descriptorpb.FileOptions_LITE_RUNTIME,
			expectedValue:   descriptorpb.FileOptions_SPEED,
			shouldNotModify: true,
		},
	}
	for _, test := range tests {
		test := test
		for _, includeSourceCodeInfo := range []bool{true, false} {
			includeSourceCodeInfo := includeSourceCodeInfo
			t.Run(test.description, func(t *testing.T) {
				t.Parallel()
				dirPath := filepath.Join(baseDir, test.subDir)
				image := bufimagemodifytesting.GetTestImage(
					t,
					dirPath,
					includeSourceCodeInfo,
				)
				markSweeper := newMarkSweeper(image)
				require.NotNil(t, markSweeper)
				imageFile := image.GetFile(test.file)
				require.NotNil(t, imageFile)
				_, ok := markSweeper.sourceCodeInfoPaths[imageFile.Path()]
				require.False(t, ok)
				err := ModifyOptimizeFor(
					markSweeper,
					imageFile,
					test.override,
				)
				require.NoError(t, err)
				err = markSweeper.Sweep()
				require.NoError(t, err)
				require.NotNil(t, imageFile.Proto())
				require.Equal(t, test.expectedValue, imageFile.Proto().GetOptions().GetOptimizeFor())
				fileKeys, ok := markSweeper.sourceCodeInfoPaths[imageFile.Path()]
				if test.shouldNotModify {
					require.False(t, ok)
					require.Equal(
						t,
						bufimagemodifytesting.GetTestImage(
							t,
							dirPath,
							includeSourceCodeInfo,
						),
						image,
					)
					return
				}
				require.True(t, ok)
				_, ok = fileKeys[internal.GetPathKey(internal.OptimizeForPath)]
				require.True(t, ok)
			})
		}
	}
}

func TestModifyObjcClassPrefix(t *testing.T) {
	t.Parallel()
	baseDir := filepath.Join("..", "testdata")
	tests := []struct {
		description     string
		subDir          string
		file            string
		modifyOptions   []ModifyObjcClassPrefixOption
		expectedValue   string
		shouldNotModify bool
	}{
		{
			description:     "objc class prefix without override on a file without this option",
			subDir:          "emptyoptions",
			file:            "a.proto",
			modifyOptions:   []ModifyObjcClassPrefixOption{},
			expectedValue:   "",
			shouldNotModify: true,
		},
		{
			description:   "objc class prefix without override on a file with this option",
			subDir:        "alloptions",
			file:          "a.proto",
			modifyOptions: []ModifyObjcClassPrefixOption{},
			// the file is not modified because we cannot resolve a non-empty objc_class_prefix.
			expectedValue:   "foo",
			shouldNotModify: true,
		},
		{
			description:   "objc class prefix without override on a file with this option and a package",
			subDir:        filepath.Join("objcoptions", "double"),
			file:          "objc.proto",
			modifyOptions: []ModifyObjcClassPrefixOption{},
			expectedValue: "AWX",
		},
		{
			description:   "objc namespace without override on a file without this option but with a package",
			subDir:        "wktimport",
			file:          "a.proto",
			modifyOptions: []ModifyObjcClassPrefixOption{},
			expectedValue: "AWX",
		},
		{
			description:   "objc class prefix with override on a file with this option",
			subDir:        "alloptions",
			file:          "a.proto",
			modifyOptions: []ModifyObjcClassPrefixOption{ModifyObjcClassPrefixWithValue("OPX")},
			expectedValue: "OPX",
		},
		{
			description:     "objc class prefix with override on a file with this option equal to the same value",
			subDir:          "alloptions",
			file:            "a.proto",
			modifyOptions:   []ModifyObjcClassPrefixOption{ModifyObjcClassPrefixWithValue("foo")},
			expectedValue:   "foo",
			shouldNotModify: true,
		},
		{
			description:     "objc class prefix with override on a wkt",
			subDir:          "wktimport",
			file:            "google/protobuf/timestamp.proto",
			modifyOptions:   []ModifyObjcClassPrefixOption{ModifyObjcClassPrefixWithValue("foo")},
			expectedValue:   "GPB",
			shouldNotModify: true,
		},
	}
	for _, test := range tests {
		test := test
		for _, includeSourceCodeInfo := range []bool{true, false} {
			includeSourceCodeInfo := includeSourceCodeInfo
			t.Run(test.description, func(t *testing.T) {
				t.Parallel()
				dirPath := filepath.Join(baseDir, test.subDir)
				image := bufimagemodifytesting.GetTestImage(
					t,
					dirPath,
					includeSourceCodeInfo,
				)
				markSweeper := newMarkSweeper(image)
				require.NotNil(t, markSweeper)
				imageFile := image.GetFile(test.file)
				require.NotNil(t, imageFile)
				_, ok := markSweeper.sourceCodeInfoPaths[imageFile.Path()]
				require.False(t, ok)
				err := ModifyObjcClassPrefix(
					markSweeper,
					imageFile,
					test.modifyOptions...,
				)
				require.NoError(t, err)
				err = markSweeper.Sweep()
				require.NoError(t, err)
				require.NotNil(t, imageFile.Proto())
				require.Equal(t, test.expectedValue, imageFile.Proto().GetOptions().GetObjcClassPrefix())
				fileKeys, ok := markSweeper.sourceCodeInfoPaths[imageFile.Path()]
				if test.shouldNotModify {
					require.False(t, ok)
					require.Equal(
						t,
						bufimagemodifytesting.GetTestImage(
							t,
							dirPath,
							includeSourceCodeInfo,
						),
						image,
					)
					return
				}
				require.True(t, ok)
				_, ok = fileKeys[internal.GetPathKey(internal.ObjcClassPrefixPath)]
				require.True(t, ok)
			})
		}
	}
}

func TestModifyCsharpNamespace(t *testing.T) {
	t.Parallel()
	baseDir := filepath.Join("..", "testdata")
	tests := []struct {
		description     string
		subDir          string
		file            string
		modifyOptions   []ModifyCsharpNamespaceOption
		expectedValue   string
		shouldNotModify bool
	}{
		{
			description:     "csharp namespace without override on a file without this option",
			subDir:          "emptyoptions",
			file:            "a.proto",
			modifyOptions:   []ModifyCsharpNamespaceOption{},
			expectedValue:   "",
			shouldNotModify: true,
		},
		{
			description:     "csharp namespace without override on a file with this option",
			subDir:          "alloptions",
			file:            "a.proto",
			modifyOptions:   []ModifyCsharpNamespaceOption{},
			expectedValue:   "foo",
			shouldNotModify: true,
		},
		{
			description:   "csharp namespace without override on a file with this option and a package",
			subDir:        filepath.Join("csharpoptions", "triple"),
			file:          "csharp.proto",
			modifyOptions: []ModifyCsharpNamespaceOption{},
			expectedValue: "Acme.Weather.Data.V1",
		},
		{
			description:   "csharp namespace without override on a file without this option but with a package",
			subDir:        "wktimport",
			file:          "a.proto",
			modifyOptions: []ModifyCsharpNamespaceOption{},
			expectedValue: "Acme.Weather.V1alpha1",
		},
		{
			description:   "csharp namespace with value override",
			subDir:        "alloptions",
			file:          "a.proto",
			modifyOptions: []ModifyCsharpNamespaceOption{ModifyCsharpNamespaceWithValue("Override.Value")},
			expectedValue: "Override.Value",
		},
		{
			description:     "csharp namespace prefix on a file without package",
			subDir:          "alloptions",
			file:            "a.proto",
			modifyOptions:   []ModifyCsharpNamespaceOption{ModifyCsharpNamespaceWithPrefix("Override.Prefix")},
			expectedValue:   "foo",
			shouldNotModify: true,
		},
		{
			description:   "csharp namespace prefix on a file with package",
			subDir:        filepath.Join("csharpoptions", "double"),
			file:          "csharp.proto",
			modifyOptions: []ModifyCsharpNamespaceOption{ModifyCsharpNamespaceWithPrefix("Override.Prefix")},
			expectedValue: "Override.Prefix.Acme.Weather.V1",
		},
		{
			description:     "csharp namespace with value override equal to the same value from file",
			subDir:          "alloptions",
			file:            "a.proto",
			modifyOptions:   []ModifyCsharpNamespaceOption{ModifyCsharpNamespaceWithValue("foo")},
			expectedValue:   "foo",
			shouldNotModify: true,
		},
		{
			description:     "csharp namespace with value override on a wkt",
			subDir:          "wktimport",
			file:            "google/protobuf/timestamp.proto",
			modifyOptions:   []ModifyCsharpNamespaceOption{ModifyCsharpNamespaceWithValue("foo")},
			expectedValue:   "Google.Protobuf.WellKnownTypes",
			shouldNotModify: true,
		},
		{
			description:     "csharp namespace prefix on a wkt",
			subDir:          "wktimport",
			file:            "google/protobuf/timestamp.proto",
			modifyOptions:   []ModifyCsharpNamespaceOption{ModifyCsharpNamespaceWithPrefix("foo")},
			expectedValue:   "Google.Protobuf.WellKnownTypes",
			shouldNotModify: true,
		},
	}
	for _, test := range tests {
		test := test
		for _, includeSourceCodeInfo := range []bool{true, false} {
			includeSourceCodeInfo := includeSourceCodeInfo
			t.Run(test.description, func(t *testing.T) {
				t.Parallel()
				dirPath := filepath.Join(baseDir, test.subDir)
				image := bufimagemodifytesting.GetTestImage(
					t,
					dirPath,
					includeSourceCodeInfo,
				)
				markSweeper := newMarkSweeper(image)
				require.NotNil(t, markSweeper)
				imageFile := image.GetFile(test.file)
				require.NotNil(t, imageFile)
				_, ok := markSweeper.sourceCodeInfoPaths[imageFile.Path()]
				require.False(t, ok)
				err := ModifyCsharpNamespace(markSweeper, imageFile, test.modifyOptions...)
				require.NoError(t, err)
				err = markSweeper.Sweep()
				require.NoError(t, err)
				require.NotNil(t, imageFile.Proto())
				require.Equal(
					t,
					test.expectedValue,
					imageFile.Proto().GetOptions().GetCsharpNamespace(),
				)
				fileKeys, ok := markSweeper.sourceCodeInfoPaths[imageFile.Path()]
				if test.shouldNotModify {
					require.False(t, ok)
					require.Equal(
						t,
						bufimagemodifytesting.GetTestImage(
							t,
							dirPath,
							includeSourceCodeInfo,
						),
						image,
					)
					return
				}
				require.True(t, ok)
				_, ok = fileKeys[internal.GetPathKey(internal.CsharpNamespacePath)]
				require.True(t, ok)
			})
		}
	}
}

func TestModifyPhpNamespace(t *testing.T) {
	t.Parallel()
	baseDir := filepath.Join("..", "testdata")
	tests := []struct {
		description     string
		subDir          string
		file            string
		modifyOptions   []ModifyPhpNamespaceOption
		expectedValue   string
		shouldNotModify bool
	}{
		{
			description:   "php namespace on a file without this option and without a package",
			subDir:        "emptyoptions",
			file:          "a.proto",
			modifyOptions: []ModifyPhpNamespaceOption{ModifyPhpNamespaceWithValue(`Foo\Bar`)},
			expectedValue: `Foo\Bar`,
		},
		{
			description:     "php namespace without override on a file with this option but without a package",
			subDir:          "alloptions",
			file:            "a.proto",
			modifyOptions:   []ModifyPhpNamespaceOption{},
			expectedValue:   "foo",
			shouldNotModify: true,
		},
		{
			description:   "php namespace without override on a file with this option and a package",
			subDir:        filepath.Join("phpoptions", "underscore"),
			file:          "php.proto",
			modifyOptions: []ModifyPhpNamespaceOption{},
			expectedValue: `Acme\Weather\FooBar\V1`,
		},
		{
			description:   "php namespace with value override",
			subDir:        "alloptions",
			file:          "a.proto",
			modifyOptions: []ModifyPhpNamespaceOption{ModifyPhpNamespaceWithValue(`Override\Value`)},
			expectedValue: `Override\Value`,
		},
		{
			description:     "php namespace with value equal to the same value from file",
			subDir:          "alloptions",
			file:            "a.proto",
			modifyOptions:   []ModifyPhpNamespaceOption{ModifyPhpNamespaceWithValue(`foo`)},
			expectedValue:   "foo",
			shouldNotModify: true,
		},
		{
			description:     "php namespace with value override on a wkt",
			subDir:          "wktimport",
			file:            "google/protobuf/timestamp.proto",
			modifyOptions:   []ModifyPhpNamespaceOption{ModifyPhpNamespaceWithValue(`Foo`)},
			expectedValue:   "",
			shouldNotModify: true,
		},
	}
	for _, test := range tests {
		test := test
		for _, includeSourceCodeInfo := range []bool{true, false} {
			includeSourceCodeInfo := includeSourceCodeInfo
			t.Run(test.description, func(t *testing.T) {
				t.Parallel()
				dirPath := filepath.Join(baseDir, test.subDir)
				image := bufimagemodifytesting.GetTestImage(
					t,
					dirPath,
					includeSourceCodeInfo,
				)
				markSweeper := newMarkSweeper(image)
				require.NotNil(t, markSweeper)
				imageFile := image.GetFile(test.file)
				require.NotNil(t, imageFile)
				_, ok := markSweeper.sourceCodeInfoPaths[imageFile.Path()]
				require.False(t, ok)
				err := ModifyPhpNamespace(
					markSweeper,
					imageFile,
					test.modifyOptions...,
				)
				require.NoError(t, err)
				err = markSweeper.Sweep()
				require.NoError(t, err)
				require.NotNil(t, imageFile.Proto())
				require.Equal(t, test.expectedValue, imageFile.Proto().GetOptions().GetPhpNamespace())
				fileKeys, ok := markSweeper.sourceCodeInfoPaths[imageFile.Path()]
				if test.shouldNotModify {
					require.False(t, ok)
					require.Equal(
						t,
						bufimagemodifytesting.GetTestImage(
							t,
							dirPath,
							includeSourceCodeInfo,
						),
						image,
					)
					return
				}
				require.True(t, ok)
				_, ok = fileKeys[internal.GetPathKey(internal.PhpNamespacePath)]
				require.True(t, ok)
			})
		}
	}
}

func TestModifyPhpMetadataNamespace(t *testing.T) {
	t.Parallel()
	baseDir := filepath.Join("..", "testdata")
	tests := []struct {
		description     string
		subDir          string
		file            string
		modifyOptions   []ModifyPhpMetadataNamespaceOption
		expectedValue   string
		shouldNotModify bool
	}{
		{
			description:     "php metadata namespace without override on a file without this option",
			subDir:          "emptyoptions",
			file:            "a.proto",
			modifyOptions:   []ModifyPhpMetadataNamespaceOption{},
			expectedValue:   "",
			shouldNotModify: true,
		},
		{
			description:     "php metadata namespace with suffix override on a file without this option",
			subDir:          "emptyoptions",
			file:            "a.proto",
			modifyOptions:   []ModifyPhpMetadataNamespaceOption{ModifyPhpMetadataNamespaceWithSuffix("Suffix")},
			expectedValue:   "",
			shouldNotModify: true,
		},
		{
			description:   "php metadata namespace without override on a file with this option and a package",
			subDir:        filepath.Join("phpoptions", "underscore"),
			file:          "php.proto",
			modifyOptions: []ModifyPhpMetadataNamespaceOption{},
			expectedValue: `Acme\Weather\FooBar\V1`,
		},
		{
			description:   "php metadata namespace with value override on a file with this option and a package",
			subDir:        filepath.Join("phpoptions", "underscore"),
			file:          "php.proto",
			modifyOptions: []ModifyPhpMetadataNamespaceOption{ModifyPhpMetadataNamespaceWithValue("Override")},
			expectedValue: "Override",
		},
		{
			description:   "php metadata namespace with suffix override on a file with this option and a package",
			subDir:        filepath.Join("phpoptions", "underscore"),
			file:          "php.proto",
			modifyOptions: []ModifyPhpMetadataNamespaceOption{ModifyPhpMetadataNamespaceWithSuffix("Metadata")},
			expectedValue: `Acme\Weather\FooBar\V1\Metadata`,
		},
		{
			description:   "php metadata namespace with suffix override on a file with this option but without package",
			subDir:        "alloptions",
			file:          "a.proto",
			modifyOptions: []ModifyPhpMetadataNamespaceOption{ModifyPhpMetadataNamespaceWithSuffix("Metadata")},
			// The namespace resolves to empty because the proto package is empty, and the image should not be modified.
			expectedValue:   "foo",
			shouldNotModify: true,
		},
		{
			description:   "php metadata namespace with suffix override on a file without this option but with a package",
			subDir:        "wktimport",
			file:          "a.proto",
			modifyOptions: []ModifyPhpMetadataNamespaceOption{ModifyPhpMetadataNamespaceWithSuffix("SpecialMetadata")},
			expectedValue: `Acme\Weather\V1alpha1\SpecialMetadata`,
		},
		{
			description:     "php metadata namespace with value override equal to the same value from file",
			subDir:          "alloptions",
			file:            "a.proto",
			modifyOptions:   []ModifyPhpMetadataNamespaceOption{ModifyPhpMetadataNamespaceWithValue("foo")},
			expectedValue:   "foo",
			shouldNotModify: true,
		},
		{
			description:     "php metadata namespace with value override on a wkt",
			subDir:          "wktimport",
			file:            "google/protobuf/timestamp.proto",
			modifyOptions:   []ModifyPhpMetadataNamespaceOption{ModifyPhpMetadataNamespaceWithValue("foo")},
			expectedValue:   "",
			shouldNotModify: true,
		},
		{
			description:     "php metadata namespace suffix on a wkt",
			subDir:          "wktimport",
			file:            "google/protobuf/timestamp.proto",
			modifyOptions:   []ModifyPhpMetadataNamespaceOption{ModifyPhpMetadataNamespaceWithSuffix("foo")},
			expectedValue:   "",
			shouldNotModify: true,
		},
	}
	for _, test := range tests {
		test := test
		for _, includeSourceCodeInfo := range []bool{true, false} {
			includeSourceCodeInfo := includeSourceCodeInfo
			t.Run(test.description, func(t *testing.T) {
				t.Parallel()
				dirPath := filepath.Join(baseDir, test.subDir)
				image := bufimagemodifytesting.GetTestImage(
					t,
					dirPath,
					includeSourceCodeInfo,
				)
				markSweeper := newMarkSweeper(image)
				require.NotNil(t, markSweeper)
				imageFile := image.GetFile(test.file)
				require.NotNil(t, imageFile)
				_, ok := markSweeper.sourceCodeInfoPaths[imageFile.Path()]
				require.False(t, ok)
				err := ModifyPhpMetadataNamespace(markSweeper, imageFile, test.modifyOptions...)
				require.NoError(t, err)
				err = markSweeper.Sweep()
				require.NoError(t, err)
				require.NotNil(t, imageFile.Proto())
				require.Equal(
					t,
					test.expectedValue,
					imageFile.Proto().GetOptions().GetPhpMetadataNamespace(),
				)
				fileKeys, ok := markSweeper.sourceCodeInfoPaths[imageFile.Path()]
				if test.shouldNotModify {
					require.False(t, ok)
					require.Equal(
						t,
						bufimagemodifytesting.GetTestImage(
							t,
							dirPath,
							includeSourceCodeInfo,
						),
						image,
					)
					return
				}
				require.True(t, ok)
				_, ok = fileKeys[internal.GetPathKey(internal.PhpMetadataNamespacePath)]
				require.True(t, ok)
			})
		}
	}
}

func TestModifyRubyPackage(t *testing.T) {
	t.Parallel()
	baseDir := filepath.Join("..", "testdata")
	tests := []struct {
		description     string
		subDir          string
		file            string
		modifyOptions   []ModifyRubyPackageOption
		expectedValue   string
		shouldNotModify bool
	}{
		{
			description:     "ruby package without override on a file without this option and without a package",
			subDir:          "emptyoptions",
			file:            "a.proto",
			modifyOptions:   []ModifyRubyPackageOption{},
			expectedValue:   "",
			shouldNotModify: true,
		},
		{
			description:     "ruby package without override on a file with this option bu without a package",
			subDir:          "alloptions",
			file:            "a.proto",
			modifyOptions:   []ModifyRubyPackageOption{},
			expectedValue:   "foo",
			shouldNotModify: true,
		},
		{
			description:   "ruby package without override on a file with this option and a package",
			subDir:        filepath.Join("rubyoptions", "underscore"),
			file:          "ruby.proto",
			modifyOptions: []ModifyRubyPackageOption{},
			expectedValue: "Acme::Weather::FooBar::V1",
		},
		{
			description:   "ruby package with value override on a file with this option and a package",
			subDir:        filepath.Join("rubyoptions", "underscore"),
			file:          "ruby.proto",
			modifyOptions: []ModifyRubyPackageOption{ModifyRubyPackageWithValue("Override")},
			expectedValue: "Override",
		},
		{
			description:   "ruby package with value override on a file without this option or a package",
			subDir:        "emptyoptions",
			file:          "a.proto",
			modifyOptions: []ModifyRubyPackageOption{ModifyRubyPackageWithValue("Override")},
			expectedValue: "Override",
		},
		{
			description:   "ruby package with suffix override on a file with this option and a package",
			subDir:        filepath.Join("rubyoptions", "underscore"),
			file:          "ruby.proto",
			modifyOptions: []ModifyRubyPackageOption{ModifyRubyPackageWithSuffix("Protos")},
			expectedValue: "Acme::Weather::FooBar::V1::Protos",
		},
		{
			description:     "ruby package with suffix override on a file with this option but without a package",
			subDir:          "alloptions",
			file:            "a.proto",
			modifyOptions:   []ModifyRubyPackageOption{ModifyRubyPackageWithSuffix("Protos")},
			expectedValue:   "foo",
			shouldNotModify: true,
		},
		{
			description:     "ruby package with value equal to the same value from file",
			subDir:          "alloptions",
			file:            "a.proto",
			modifyOptions:   []ModifyRubyPackageOption{ModifyRubyPackageWithValue("foo")},
			expectedValue:   "foo",
			shouldNotModify: true,
		},
		{
			description:     "ruby package value on a wkt",
			subDir:          "wktimport",
			file:            "google/protobuf/timestamp.proto",
			modifyOptions:   []ModifyRubyPackageOption{ModifyRubyPackageWithValue("foo")},
			expectedValue:   "",
			shouldNotModify: true,
		},
		{
			description:     "ruby package suffix on a wkt",
			subDir:          "wktimport",
			file:            "google/protobuf/timestamp.proto",
			modifyOptions:   []ModifyRubyPackageOption{ModifyRubyPackageWithSuffix("foo")},
			expectedValue:   "",
			shouldNotModify: true,
		},
	}
	for _, test := range tests {
		test := test
		for _, includeSourceCodeInfo := range []bool{true, false} {
			includeSourceCodeInfo := includeSourceCodeInfo
			t.Run(test.description, func(t *testing.T) {
				t.Parallel()
				dirPath := filepath.Join(baseDir, test.subDir)
				image := bufimagemodifytesting.GetTestImage(
					t,
					dirPath,
					includeSourceCodeInfo,
				)
				markSweeper := newMarkSweeper(image)
				require.NotNil(t, markSweeper)
				imageFile := image.GetFile(test.file)
				require.NotNil(t, imageFile)
				_, ok := markSweeper.sourceCodeInfoPaths[imageFile.Path()]
				require.False(t, ok)
				ModifyRubyPackage(markSweeper, imageFile, test.modifyOptions...)
				err := markSweeper.Sweep()
				require.NoError(t, err)
				require.NotNil(t, imageFile.Proto())
				require.Equal(
					t,
					test.expectedValue,
					imageFile.Proto().GetOptions().GetRubyPackage(),
				)
				fileKeys, ok := markSweeper.sourceCodeInfoPaths[imageFile.Path()]
				if test.shouldNotModify {
					require.False(t, ok)
					require.Equal(
						t,
						bufimagemodifytesting.GetTestImage(
							t,
							dirPath,
							includeSourceCodeInfo,
						),
						image,
					)
					return
				}
				require.True(t, ok)
				_, ok = fileKeys[internal.GetPathKey(internal.RubyPackagePath)]
				require.True(t, ok)
			})
		}
	}
}

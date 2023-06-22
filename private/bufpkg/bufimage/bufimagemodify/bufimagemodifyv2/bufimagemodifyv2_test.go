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

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/bufimagemodifytesting"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestModifySingleOption(t *testing.T) {
	t.Parallel()
	baseDir := filepath.Join("..", "testdata")
	tests := []struct {
		description             string
		subDir                  string
		file                    string
		fileHasNoSourceCodeInfo bool
		modifyFunc              func(Marker, bufimage.ImageFile, Override) error
		fileOptionPath          []int32
		override                Override
		expectedValue           interface{}
		// This should be set to true when an override has no effect,
		// i.e. override is the same as defined in proto file.
		shouldKeepSourceCodeInfo bool
		assertFunc               func(*testing.T, interface{}, *descriptorpb.FileDescriptorProto)
	}{
		{
			description:             "Modify Java Package with value on file with empty options",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyJavaPackage,
			fileOptionPath:          internal.JavaPackagePath,
			override:                NewValueOverride("valueoverride"),
			expectedValue:           "valueoverride",
			assertFunc:              assertJavaPackage,
		},
		{
			description:             "Modify Java Package with prefix on file with empty options and empty proto package",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyJavaPackage,
			fileOptionPath:          internal.JavaPackagePath,
			override:                NewPrefixOverride("prefixoverride"),
			// emptyoptions/a.proto does not have a proto package, thus the result is an empty string
			expectedValue: "",
			assertFunc:    assertJavaPackage,
		},
		{
			description:    "Modify Java Package with prefix on file with all options and empty proto package",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			override:       NewPrefixOverride("prefixoverride"),
			// all/options/a.proto does not have a proto package, thus the result is an empty string
			expectedValue: "",
			assertFunc:    assertJavaPackage,
		},
		{
			description:    "Modify Java Package with value on file with all options and empty proto package",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			override:       NewValueOverride("alloverride"),
			expectedValue:  "alloverride",
			assertFunc:     assertJavaPackage,
		},
		{
			description:             "Modify Java Package with value on file with empty options and a proto package",
			subDir:                  "javaemptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyJavaPackage,
			fileOptionPath:          internal.JavaPackagePath,
			override:                NewPrefixOverride("override.pre"),
			expectedValue:           "override.pre.foo",
			assertFunc:              assertJavaPackage,
		},
		{
			description:    "Modify Java Package with prefix on file with java options and a proto package",
			subDir:         "javaoptions",
			file:           "java_file.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			override:       NewPrefixOverride("prefix"),
			expectedValue:  "prefix.acme.weather",
			assertFunc:     assertJavaPackage,
		},
		{
			description:              "Modify Java Package with override value the same as java package",
			subDir:                   "javaoptions",
			file:                     "java_file.proto",
			modifyFunc:               ModifyJavaPackage,
			fileOptionPath:           internal.JavaPackagePath,
			override:                 NewValueOverride("foo"),
			expectedValue:            "foo",
			shouldKeepSourceCodeInfo: true,
			assertFunc:               assertJavaPackage,
		},
		{
			description:             "Modify Java Package with wkt",
			subDir:                  "wktimport",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyJavaPackage,
			fileOptionPath:          internal.JavaPackagePath,
			override:                NewValueOverride("override.value"),
			expectedValue:           "override.value",
			assertFunc:              assertJavaPackage,
		},
		{
			description:    "Modify Java Package with empty prefix on file with java options and a proto package",
			subDir:         "javaoptions",
			file:           "java_file.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			override:       NewPrefixOverride(""),
			// use the package name when prefix is empty
			expectedValue: "acme.weather",
			assertFunc:    assertJavaPackage,
		},
		{
			description:    "Modify Java Package with value on file with java options and a proto package",
			subDir:         "javaoptions",
			file:           "java_file.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			override:       NewValueOverride("pkg.pkg"),
			expectedValue:  "pkg.pkg",
			assertFunc:     assertJavaPackage,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()
			{
				// Get image with source code info.
				image := bufimagemodifytesting.GetTestImage(
					t,
					filepath.Join(baseDir, test.subDir),
					true,
				)
				if test.fileHasNoSourceCodeInfo {
					bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(
						t,
						image,
						test.fileOptionPath,
						true,
						bufimagemodifytesting.AssertSourceCodeInfoWithIgnoreWKT(),
					)
				} else {
					bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(
						t,
						image,
						test.fileOptionPath,
					)
				}
				markSweeper := NewMarkSweeper(image)
				require.NotNil(t, markSweeper)
				imageFile := image.GetFile(test.file)
				require.NotNil(t, imageFile)
				err := test.modifyFunc(
					markSweeper,
					imageFile,
					test.override,
				)
				require.NoError(t, err)
				err = markSweeper.Sweep()
				require.NoError(t, err)
				require.NotNil(t, imageFile.Proto())
				test.assertFunc(t, test.expectedValue, imageFile.Proto())
				if test.shouldKeepSourceCodeInfo {
					bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(
						t,
						image,
						test.fileOptionPath,
					)
				} else {
					bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmptyForFile(
						t,
						imageFile,
						test.fileOptionPath,
						true,
					)
				}
			}
			{
				// Get image without source code info.
				image := bufimagemodifytesting.GetTestImage(
					t,
					filepath.Join(baseDir, test.subDir),
					false,
				)
				bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(
					t,
					image,
					test.fileOptionPath,
					false,
				)
				markSweeper := NewMarkSweeper(image)
				require.NotNil(t, markSweeper)
				imageFile := image.GetFile(test.file)
				require.NotNil(t, imageFile)
				err := test.modifyFunc(
					markSweeper,
					imageFile,
					test.override,
				)
				require.NoError(t, err)
				err = markSweeper.Sweep()
				require.NoError(t, err)
				require.NotNil(t, imageFile.Proto())
				test.assertFunc(t, test.expectedValue, imageFile.Proto())
				bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(
					t,
					image,
					test.fileOptionPath,
					false,
				)
			}
		})
	}
}

func TestModifyError(t *testing.T) {
	t.Parallel()
	baseDir := filepath.Join("..", "testdata")
	tests := []struct {
		description        string
		subDir             string
		file               string
		modifyFunc         func(Marker, bufimage.ImageFile, Override) error
		override           Override
		expectedErrMessage string
	}{
		{
			description:        "Test bool override for java package",
			subDir:             "javaoptions",
			file:               "java_file.proto",
			modifyFunc:         ModifyJavaPackage,
			override:           NewValueOverride(true),
			expectedErrMessage: "a valid override is required for java_package",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()
			image := bufimagemodifytesting.GetTestImage(
				t,
				filepath.Join(baseDir, test.subDir),
				true,
			)
			markSweeper := NewMarkSweeper(image)
			require.NotNil(t, markSweeper)
			imageFile := image.GetFile(test.file)
			require.NotNil(t, imageFile)
			err := test.modifyFunc(
				markSweeper,
				imageFile,
				test.override,
			)
			require.ErrorContains(t, err, test.expectedErrMessage)
		})
	}
}

func assertJavaPackage(t *testing.T, expectedValue interface{}, descriptor *descriptorpb.FileDescriptorProto) {
	assert.Equal(t, expectedValue, descriptor.GetOptions().GetJavaPackage())
}

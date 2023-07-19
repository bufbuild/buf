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
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal/bufimagemodifytesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestModifySingleOption(t *testing.T) {
	t.Parallel()
	baseDir := filepath.Join("..", "testdata")
	tests := []struct {
		description    string
		subDir         string
		file           string
		modifyFunc     func(Marker, bufimage.ImageFile, ...ModifyOption) error
		fileOptionPath []int32
		modifyOptions  []ModifyOption
		expectedValue  interface{}
		assertFunc     func(*testing.T, interface{}, *descriptorpb.FileDescriptorProto)
		shouldNotMark  bool
	}{
		{
			description:    "Modify Java Package with prefix on file without a proto package",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			modifyOptions:  modifyWithPrefix(t, "prefix"),
			expectedValue:  "",
			assertFunc:     assertJavaPackage,
		},
		{
			description:    "Modify Java Package with value on file without a proto package",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			modifyOptions:  modifyWithStringValue(t, "com.example"),
			expectedValue:  "com.example",
			assertFunc:     assertJavaPackage,
		},
		{
			description:    "Modify Java Package with prefix and suffix on file without a proto package",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			modifyOptions:  modifyWithPrefixAndSuffix(t, "prefix", "suffix"),
			expectedValue:  "",
			assertFunc:     assertJavaPackage,
		},
		{
			description:    "Modify Java Package without override on file without a proto package",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			modifyOptions:  make([]ModifyOption, 0),
			expectedValue:  "",
			assertFunc:     assertJavaPackage,
		},
		{
			description:    "Modify Java Package with override value the same as java package",
			subDir:         "javaoptions",
			file:           "java_file.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			modifyOptions:  modifyWithStringValue(t, "foo"),
			expectedValue:  "foo",
			assertFunc:     assertJavaPackage,
			shouldNotMark:  true,
		},
		{
			description:    "Modify Java Package with value on file with a proto package",
			subDir:         "javaoptions",
			file:           "java_file.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			modifyOptions:  modifyWithStringValue(t, "bar"),
			expectedValue:  "bar",
			assertFunc:     assertJavaPackage,
		},
		{
			description:    "Modify Java Package without override on a file with a proto package",
			subDir:         "javaemptyoptions",
			file:           "a.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			modifyOptions:  make([]ModifyOption, 0),
			expectedValue:  "foo",
			assertFunc:     assertJavaPackage,
		},
		{
			description:    "Modify Java Package with prefix on file with a proto package",
			subDir:         "javaoptions",
			file:           "java_file.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			modifyOptions:  modifyWithPrefix(t, "prefix.override"),
			expectedValue:  "prefix.override.acme.weather",
			assertFunc:     assertJavaPackage,
		},
		{
			description:    "Modify Java Package with prefix and suffix on file with a proto package",
			subDir:         "javaoptions",
			file:           "java_file.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			modifyOptions:  modifyWithPrefixAndSuffix(t, "prefix.override", "override.suffix"),
			expectedValue:  "prefix.override.acme.weather.override.suffix",
			assertFunc:     assertJavaPackage,
		},
		{
			description:    "Modify Java Package with suffix on file with a proto package",
			subDir:         "javaoptions",
			file:           "java_file.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			modifyOptions:  modifyWithSuffix(t, "override.suffix"),
			expectedValue:  "acme.weather.override.suffix",
			assertFunc:     assertJavaPackage,
		},
		{
			description:    "Modify Java Package with prefix and suffix on a wkt file",
			subDir:         "wktimport",
			file:           "google/protobuf/timestamp.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			modifyOptions:  modifyWithPrefixAndSuffix(t, "prefix.override", "override.suffix"),
			expectedValue:  "com.google.protobuf",
			assertFunc:     assertJavaPackage,
			shouldNotMark:  true,
		},
		{
			description:    "Modify Java Package with value on a wkt file",
			subDir:         "wktimport",
			file:           "google/protobuf/timestamp.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			modifyOptions:  modifyWithStringValue(t, "value"),
			expectedValue:  "com.google.protobuf",
			assertFunc:     assertJavaPackage,
			shouldNotMark:  true,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()
			// Get image with source code info.
			image := bufimagemodifytesting.GetTestImage(
				t,
				filepath.Join(baseDir, test.subDir),
				true,
			)
			markSweeper := newMarkSweeper(image)
			require.NotNil(t, markSweeper)
			imageFile := image.GetFile(test.file)
			require.NotNil(t, imageFile)
			_, ok := markSweeper.sourceCodeInfoPaths[imageFile.Path()]
			require.False(t, ok)
			err := test.modifyFunc(
				markSweeper,
				imageFile,
				test.modifyOptions...,
			)
			require.NoError(t, err)
			err = markSweeper.Sweep()
			require.NoError(t, err)
			require.NotNil(t, imageFile.Proto())
			test.assertFunc(t, test.expectedValue, imageFile.Proto())
			fileKeys, ok := markSweeper.sourceCodeInfoPaths[imageFile.Path()]
			if test.shouldNotMark {
				require.False(t, ok)
			} else {
				require.True(t, ok)
				_, ok = fileKeys[internal.GetPathKey(test.fileOptionPath)]
				require.True(t, ok)
			}
		})
	}
}

func TestSweep(t *testing.T) {
	// TODO
}

func modifyWithStringValue(t *testing.T, value string) []ModifyOption {
	option, err := ModifyWithOverride(NewValueOverride(value))
	require.NoError(t, err)
	return []ModifyOption{option}
}

func modifyWithPrefix(t *testing.T, prefix string) []ModifyOption {
	option, err := ModifyWithOverride(NewPrefixOverride(prefix))
	require.NoError(t, err)
	return []ModifyOption{option}
}

func modifyWithSuffix(t *testing.T, prefix string) []ModifyOption {
	option, err := ModifyWithOverride(NewSuffixOverride(prefix))
	require.NoError(t, err)
	return []ModifyOption{option}
}

func modifyWithPrefixAndSuffix(t *testing.T, prefix string, suffix string) []ModifyOption {
	option, err := ModifyWithOverride(
		NewPrefixSuffixOverride(
			prefix,
			suffix,
		),
	)
	require.NoError(t, err)
	return []ModifyOption{option}
}

func assertJavaPackage(t *testing.T, expectedValue interface{}, descriptor *descriptorpb.FileDescriptorProto) {
	assert.Equal(t, expectedValue, descriptor.GetOptions().GetJavaPackage())
}

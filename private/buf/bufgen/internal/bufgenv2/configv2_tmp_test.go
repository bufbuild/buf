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

package bufgenv2

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufgen/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/bufimagemodifyv2"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestManagedConfigTmp(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	nopLogger := zap.NewNop()
	readBucket, err := storagemem.NewReadBucket(nil)
	require.NoError(t, err)
	tests := []struct {
		testName string
		file     string
		// true means disabled
		expectedDisableResults  map[fileOption]map[imageFileIdentity]bool
		expectedOverrideResults map[fileOptionGroup]map[imageFileIdentity]bufimagemodifyv2.Override
	}{
		{
			testName: "java outer class name",
			file:     filepath.Join("managed", "java_outer_classname"),
			expectedOverrideResults: map[fileOptionGroup]map[imageFileIdentity]bufimagemodifyv2.Override{
				groupJavaOuterClassname: {
					&fakeImageFileIdentity{
						path: "random.proto",
					}: bufimagemodifyv2.NewValueOverride("DefaultProto"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "petapis"),
					}: bufimagemodifyv2.NewValueOverride("PathProto"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
					}: bufimagemodifyv2.NewValueOverride("ModProto"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: bufimagemodifyv2.NewValueOverride("PathProto"),
					&fakeImageFileIdentity{
						path:   "a/b/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: bufimagemodifyv2.NewValueOverride("Both"),
				},
			},
		},
		{
			testName: "java multiple files",
			file:     filepath.Join("managed", "java_multiple_files"),
			expectedOverrideResults: map[fileOptionGroup]map[imageFileIdentity]bufimagemodifyv2.Override{
				groupJavaMultipleFiles: {
					&fakeImageFileIdentity{
						path: "random.proto",
					}: bufimagemodifyv2.NewValueOverride(true),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "petapis"),
					}: bufimagemodifyv2.NewValueOverride(false),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
					}: bufimagemodifyv2.NewValueOverride(false),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: bufimagemodifyv2.NewValueOverride(false),
					&fakeImageFileIdentity{
						path:   "a/b/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: bufimagemodifyv2.NewValueOverride(true),
				},
			},
		},
		{
			testName: "java string check utf 8",
			file:     filepath.Join("managed", "java_string_check_utf8"),
			expectedOverrideResults: map[fileOptionGroup]map[imageFileIdentity]bufimagemodifyv2.Override{
				groupJavaStringCheckUtf8: {
					&fakeImageFileIdentity{
						path: "random.proto",
					}: bufimagemodifyv2.NewValueOverride(true),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "petapis"),
					}: bufimagemodifyv2.NewValueOverride(false),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
					}: bufimagemodifyv2.NewValueOverride(false),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: bufimagemodifyv2.NewValueOverride(false),
					&fakeImageFileIdentity{
						path:   "a/b/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: bufimagemodifyv2.NewValueOverride(true),
				},
			},
		},
		{
			testName: "optimize for",
			file:     filepath.Join("managed", "optimize_for"),
			expectedOverrideResults: map[fileOptionGroup]map[imageFileIdentity]bufimagemodifyv2.Override{
				groupOptimizeFor: {
					&fakeImageFileIdentity{
						path: "random.proto",
					}: bufimagemodifyv2.NewValueOverride(descriptorpb.FileOptions_SPEED),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "petapis"),
					}: bufimagemodifyv2.NewValueOverride(descriptorpb.FileOptions_CODE_SIZE),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
					}: bufimagemodifyv2.NewValueOverride(descriptorpb.FileOptions_CODE_SIZE),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: bufimagemodifyv2.NewValueOverride(descriptorpb.FileOptions_CODE_SIZE),
					&fakeImageFileIdentity{
						path:   "a/b/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: bufimagemodifyv2.NewValueOverride(descriptorpb.FileOptions_LITE_RUNTIME),
				},
			},
		},
		{
			testName: "go package",
			file:     filepath.Join("managed", "go_package"),
			expectedOverrideResults: map[fileOptionGroup]map[imageFileIdentity]bufimagemodifyv2.Override{
				groupGoPackage: {
					&fakeImageFileIdentity{
						path: "random.proto",
					}: bufimagemodifyv2.NewPrefixOverride("github.com/example/protos"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "petapis"),
					}: bufimagemodifyv2.NewValueOverride("package1"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
					}: bufimagemodifyv2.NewValueOverride("weather"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: bufimagemodifyv2.NewValueOverride("package1"),
					&fakeImageFileIdentity{
						path:   "a/b/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: bufimagemodifyv2.NewPrefixOverride("special/prefix"),
				},
			},
		},
		{
			testName: "objc class prefix",
			file:     filepath.Join("managed", "objc_class_prefix"),
			expectedOverrideResults: map[fileOptionGroup]map[imageFileIdentity]bufimagemodifyv2.Override{
				groupObjcClassPrefix: {
					&fakeImageFileIdentity{
						path: "random.proto",
					}: bufimagemodifyv2.NewValueOverride("foo"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "petapis"),
					}: bufimagemodifyv2.NewValueOverride("bar"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
					}: bufimagemodifyv2.NewValueOverride("baz"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: bufimagemodifyv2.NewValueOverride("bar"),
					&fakeImageFileIdentity{
						path:   "a/b/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: bufimagemodifyv2.NewValueOverride("qux"),
				},
			},
		},
		{
			testName: "csharp namespace",
			file:     filepath.Join("managed", "csharp_namespace"),
			expectedOverrideResults: map[fileOptionGroup]map[imageFileIdentity]bufimagemodifyv2.Override{
				groupCsharpNamespace: {
					&fakeImageFileIdentity{
						path: "random.proto",
					}: bufimagemodifyv2.NewPrefixOverride("foo"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "petapis"),
					}: bufimagemodifyv2.NewValueOverride("bar"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
					}: bufimagemodifyv2.NewValueOverride("baz"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: bufimagemodifyv2.NewValueOverride("bar"),
					&fakeImageFileIdentity{
						path:   "a/b/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: bufimagemodifyv2.NewPrefixOverride("qux"),
				},
			},
		},
		{
			testName: "php namespace",
			file:     filepath.Join("managed", "php_namespace"),
			expectedOverrideResults: map[fileOptionGroup]map[imageFileIdentity]bufimagemodifyv2.Override{
				groupPhpNamespace: {
					&fakeImageFileIdentity{
						path: "random.proto",
					}: bufimagemodifyv2.NewValueOverride(`Foo\Bar`),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "petapis"),
					}: bufimagemodifyv2.NewValueOverride(`Bar\Baz`),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
					}: bufimagemodifyv2.NewValueOverride(`Baz\Qux`),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: bufimagemodifyv2.NewValueOverride(`Bar\Baz`),
					&fakeImageFileIdentity{
						path:   "a/b/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: bufimagemodifyv2.NewValueOverride(`Qux\Quux`),
				},
			},
		},
		{
			testName: "php metadata namespace",
			file:     filepath.Join("managed", "php_metadata_namespace"),
			expectedOverrideResults: map[fileOptionGroup]map[imageFileIdentity]bufimagemodifyv2.Override{
				groupPhpMetadataNamespace: {
					&fakeImageFileIdentity{
						path: "random.proto",
					}: bufimagemodifyv2.NewSuffixOverride("DefaultMetadata"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "petapis"),
					}: bufimagemodifyv2.NewValueOverride(`Foo\Bar`),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
					}: bufimagemodifyv2.NewValueOverride(`Bar\Baz`),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: bufimagemodifyv2.NewValueOverride(`Foo\Bar`),
					&fakeImageFileIdentity{
						path:   "a/b/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: bufimagemodifyv2.NewSuffixOverride("SpecialMetadata"),
				},
			},
		},
		{
			testName: "ruby package",
			file:     filepath.Join("managed", "ruby_package"),
			expectedOverrideResults: map[fileOptionGroup]map[imageFileIdentity]bufimagemodifyv2.Override{
				groupRubyPackage: {
					&fakeImageFileIdentity{
						path: "random.proto",
					}: bufimagemodifyv2.NewSuffixOverride("protos"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "petapis"),
					}: bufimagemodifyv2.NewValueOverride("Foo::Bar"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
					}: bufimagemodifyv2.NewValueOverride("Bar::Baz"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: bufimagemodifyv2.NewValueOverride("Foo::Bar"),
					&fakeImageFileIdentity{
						path:   "a/b/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: bufimagemodifyv2.NewSuffixOverride("pbs"),
				},
			},
		},
	}

	for _, test := range tests {
		test := test
		for _, fileExtension := range []string{".yaml" /* , ".json" */} { // TODO: add json back
			fileExtension := fileExtension
			t.Run(fmt.Sprintf("%s with extension %s", test.testName, fileExtension), func(t *testing.T) {
				t.Parallel()
				file := filepath.Join("testdata", test.file+fileExtension)
				config, err := readConfigV2(
					ctx,
					nopLogger,
					readBucket,
					internal.ReadConfigWithOverride(file),
				)
				require.NoError(t, err)
				require.NotNil(t, config)
				require.NotNil(t, config.Managed)
				require.NotNil(t, config.Managed.DisabledFunc)
				for fileOption, resultsForFiles := range test.expectedDisableResults {
					for imageFile, expectedResult := range resultsForFiles {
						actual := config.Managed.DisabledFunc(fileOption, imageFile)
						require.Equal(
							t,
							expectedResult,
							actual,
							"whether to disable %v for %v should be %v", fileOption, imageFile, expectedResult,
						)
					}
				}
				for fileOptionGroup, resultsForFiles := range test.expectedOverrideResults {
					for imageFile, expectedOverride := range resultsForFiles {
						overrideFunc, ok := config.Managed.FileOptionGroupToOverrideFunc[fileOptionGroup]
						require.True(t, ok)
						actual := overrideFunc(imageFile)
						require.Equal(
							t,
							expectedOverride,
							actual,
							"override for %v should be %v, not %v", imageFile, expectedOverride, actual,
						)
					}
				}
			})
		}
	}
}

func TestConfigErrorTmp(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	nopLogger := zap.NewNop()
	readBucket, err := storagemem.NewReadBucket(nil)
	require.NoError(t, err)

	tests := []struct {
		testName      string
		file          string
		expectedError string
	}{
		{
			testName:      "Test compression is not allowed for Git",
			file:          filepath.Join("input", "git_error1"),
			expectedError: newOptionNotAllowedForInputMessage("compression", "git_repo"),
		},
	}

	for _, test := range tests {
		test := test
		for _, fileExtension := range []string{".yaml" /* ".json" */} { // TODO: add json back
			fileExtension := fileExtension
			t.Run(fmt.Sprintf("%s with extension %s", test.testName, fileExtension), func(t *testing.T) {
				t.Parallel()
				file := filepath.Join("testdata", test.file+fileExtension)
				config, err := readConfigV2(
					ctx,
					nopLogger,
					readBucket,
					internal.ReadConfigWithOverride(file),
				)
				require.Nil(t, config)
				require.ErrorContains(t, err, test.expectedError)
			})
		}
	}
}

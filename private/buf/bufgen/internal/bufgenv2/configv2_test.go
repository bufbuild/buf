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
	"os"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/buf/bufgen/internal"
	"github.com/bufbuild/buf/private/buf/bufgen/internal/bufgenplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/bufimagemodifyv2"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestConfigSuccess(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	nopLogger := zap.NewNop()
	refBuilder := buffetch.NewRefBuilder()
	readBucket, err := storagemem.NewReadBucket(nil)
	require.NoError(t, err)
	placeHolderPlugins := []bufgenplugin.PluginConfig{
		mustCreateBinaryPlugin(
			t,
			"",
			[]string{"protoc-gen-go"},
			internal.StrategyDirectory,
			"gen/out",
			"",
			false,
			false,
		),
	}

	tests := []struct {
		testName       string
		file           string
		expectedConfig *Config
	}{
		{
			testName: "Test git",
			file:     filepath.Join("input", "git_success"),
			expectedConfig: &Config{
				Plugins: placeHolderPlugins,
				Inputs: []*InputConfig{
					{
						InputRef: mustGetGitRef(
							t,
							ctx,
							refBuilder,
							"github.com/acme/weather0.git",
							buffetch.WithGetGitRefBranch("main"),
							buffetch.WithGetGitRefRef("fdafaewafe"),
							buffetch.WithGetGitRefDepth(1),
							buffetch.WithGetGitRefSubDir("protos"),
							buffetch.WithGetGitRefRecurseSubmodules(),
						),
						Types: []string{
							"a.b.c1",
							"a.b.c2",
						},
						ExcludePaths: []string{
							"x/y/z1",
							"x/y/z2",
						},
						IncludePaths: []string{
							"x/y/w1",
							"x/y/w2",
						},
					},
					{
						InputRef: mustGetGitRef(
							t,
							ctx,
							refBuilder,
							"github.com/acme/weather1.git",
							buffetch.WithGetGitRefTag("v123"),
							buffetch.WithGetGitRefDepth(10),
							buffetch.WithGetGitRefSubDir("proto"),
						),
						Types:        nil,
						ExcludePaths: nil,
						IncludePaths: nil,
					},
					{
						InputRef: mustGetGitRef(
							t,
							ctx,
							refBuilder,
							"github.com/acme/weather2.git",
						),
						Types:        nil,
						ExcludePaths: nil,
						IncludePaths: nil,
					},
				},
			},
		},
		{
			testName: "Test module",
			file:     filepath.Join("input", "module_success"),
			expectedConfig: &Config{
				Plugins: placeHolderPlugins,
				Inputs: []*InputConfig{
					{
						InputRef: mustGetModuleRef(
							t,
							ctx,
							refBuilder,
							"buf.build/acme/weather",
						),
						Types: []string{
							"foo.v1.User",
							"foo.v1.UserService",
						},
						IncludePaths: []string{
							"protos/a",
							"protos/b",
						},
						ExcludePaths: []string{
							"protos/c",
							"protos/d",
						},
					},
					{
						InputRef: mustGetModuleRef(
							t,
							ctx,
							refBuilder,
							"buf.build/acme/weather",
						),
					},
				},
			},
		},
		{
			testName: "Test proto file",
			file:     filepath.Join("input", "proto_file_success"),
			expectedConfig: &Config{
				Plugins: placeHolderPlugins,
				Inputs: []*InputConfig{
					{
						InputRef: mustGetProtoFileRef(
							t,
							ctx,
							refBuilder,
							"a.proto",
							buffetch.WithGetProtoFileRefIncludePackageFiles(),
						),
						Types: []string{
							"a.b.c1",
							"a.b.c2",
						},
						ExcludePaths: []string{
							"x/y/z1",
							"x/y/z2",
						},
						IncludePaths: []string{
							"x/y/w1",
							"x/y/w2",
						},
					},
					{
						InputRef: mustGetProtoFileRef(
							t,
							ctx,
							refBuilder,
							"b.proto",
						),
					},
					{
						InputRef: mustGetProtoFileRef(
							t,
							ctx,
							refBuilder,
							"c.proto",
						),
					},
				},
			},
		},
		{
			testName: "Test directory",
			file:     filepath.Join("input", "dir_success"),
			expectedConfig: &Config{
				Plugins: placeHolderPlugins,
				Inputs: []*InputConfig{
					{
						InputRef: mustGetDirRef(
							t,
							ctx,
							refBuilder,
							"a/b",
						),
						Types: []string{
							"a.b.c1",
							"a.b.c2",
						},
						ExcludePaths: []string{
							"x/y/z1",
							"x/y/z2",
						},
						IncludePaths: []string{
							"x/y/w1",
							"x/y/w2",
						},
					},
					{
						InputRef: mustGetDirRef(
							t,
							ctx,
							refBuilder,
							"/c/d",
						),
					},
					{
						InputRef: mustGetDirRef(
							t,
							ctx,
							refBuilder,
							"/e/f/g/",
						),
					},
				},
			},
		},
		{
			testName: "Test tarball",
			file:     filepath.Join("input", "tar_success"),
			expectedConfig: &Config{
				Plugins: placeHolderPlugins,
				Inputs: []*InputConfig{
					{
						InputRef: mustGetTarballRef(
							t,
							ctx,
							refBuilder,
							"a/b/c",
							buffetch.WithGetTarballRefCompression("gzip"),
							buffetch.WithGetTarballRefStripComponents(6),
							buffetch.WithGetTarballRefSubDir("x/y"),
						),
						Types: []string{
							"a.b.c1",
							"a.b.c2",
						},
						ExcludePaths: []string{
							"x/y/z1",
							"x/y/z2",
						},
						IncludePaths: []string{
							"x/y/w1",
							"x/y/w2",
						},
					},
					{
						InputRef: mustGetTarballRef(
							t,
							ctx,
							refBuilder,
							"a/b/c.tar.gz",
							buffetch.WithGetTarballRefCompression("gzip"),
						),
					},
					{
						InputRef: mustGetTarballRef(
							t,
							ctx,
							refBuilder,
							"a/b/c.tgz",
							buffetch.WithGetTarballRefCompression("gzip"),
						),
					},
					{
						InputRef: mustGetTarballRef(
							t,
							ctx,
							refBuilder,
							"a/b/c.tar.zst",
							buffetch.WithGetTarballRefCompression("zstd"),
						),
					},
					{
						InputRef: mustGetTarballRef(
							t,
							ctx,
							refBuilder,
							"a/b/c.tar",
							buffetch.WithGetTarballRefCompression("none"),
						),
					},
					{
						InputRef: mustGetTarballRef(
							t,
							ctx,
							refBuilder,
							"a/b/c",
							buffetch.WithGetTarballRefCompression("none"),
						),
					},
					{
						InputRef: mustGetTarballRef(
							t,
							ctx,
							refBuilder,
							"-",
						),
					},
				},
			},
		},
		{
			testName: "Test zip archive",
			file:     filepath.Join("input", "zip_success"),
			expectedConfig: &Config{
				Plugins: placeHolderPlugins,
				Inputs: []*InputConfig{
					{
						InputRef: mustGetZipArchiveRef(
							t,
							ctx,
							refBuilder,
							"a/b/c.zip",
							buffetch.WithGetZipArchiveRefStripComponents(10),
							buffetch.WithGetZipArchiveRefSubDir("x/y"),
						),
						Types: []string{
							"a.b.c1",
							"a.b.c2",
						},
						ExcludePaths: []string{
							"x/y/z1",
							"x/y/z2",
						},
						IncludePaths: []string{
							"x/y/w1",
							"x/y/w2",
						},
					},
					{
						InputRef: mustGetZipArchiveRef(
							t,
							ctx,
							refBuilder,
							"a/b/c",
						),
					},
					{
						InputRef: mustGetZipArchiveRef(
							t,
							ctx,
							refBuilder,
							"-",
						),
					},
				},
			},
		},
		{
			testName: "Test json image",
			file:     filepath.Join("input", "json_success"),
			expectedConfig: &Config{
				Plugins: placeHolderPlugins,
				Inputs: []*InputConfig{
					{
						InputRef: mustGetJSONImageRef(
							t,
							ctx,
							refBuilder,
							"a/b/c",
							buffetch.WithGetImageRefOption("gzip"),
						),
						Types: []string{
							"a.b.c1",
							"a.b.c2",
						},
						ExcludePaths: []string{
							"x/y/z1",
							"x/y/z2",
						},
						IncludePaths: []string{
							"x/y/w1",
							"x/y/w2",
						},
					},
					{
						InputRef: mustGetJSONImageRef(
							t,
							ctx,
							refBuilder,
							"a/b/c.json.gz",
							buffetch.WithGetImageRefOption("gzip"),
						),
					},
					{
						InputRef: mustGetJSONImageRef(
							t,
							ctx,
							refBuilder,
							"a/b/c.json.zst",
							buffetch.WithGetImageRefOption("zstd"),
						),
					},
					{
						InputRef: mustGetJSONImageRef(
							t,
							ctx,
							refBuilder,
							"a/b/c.json",
							buffetch.WithGetImageRefOption("none"),
						),
					},
					{
						InputRef: mustGetJSONImageRef(
							t,
							ctx,
							refBuilder,
							"a/b/c",
							buffetch.WithGetImageRefOption("none"),
						),
					},
					{
						InputRef: mustGetJSONImageRef(
							t,
							ctx,
							refBuilder,
							"-",
							buffetch.WithGetImageRefOption("none"),
						),
					},
				},
			},
		},
		{
			testName: "Test bin image",
			file:     filepath.Join("input", "bin_success"),
			expectedConfig: &Config{
				Plugins: placeHolderPlugins,
				Inputs: []*InputConfig{
					{
						InputRef: mustGetBinaryImageRef(
							t,
							ctx,
							refBuilder,
							"a/b/c",
							buffetch.WithGetImageRefOption("gzip"),
						),
						Types: []string{
							"a.b.c1",
							"a.b.c2",
						},
						ExcludePaths: []string{
							"x/y/z1",
							"x/y/z2",
						},
						IncludePaths: []string{
							"x/y/w1",
							"x/y/w2",
						},
					},
					{
						InputRef: mustGetBinaryImageRef(
							t,
							ctx,
							refBuilder,
							"a/b/c.bin.gz",
							buffetch.WithGetImageRefOption("gzip"),
						),
					},
					{
						InputRef: mustGetBinaryImageRef(
							t,
							ctx,
							refBuilder,
							"a/b/c.bin.zst",
							buffetch.WithGetImageRefOption("zstd"),
						),
					},
					{
						InputRef: mustGetBinaryImageRef(
							t,
							ctx,
							refBuilder,
							"a/b/c.bin",
							buffetch.WithGetImageRefOption("none"),
						),
					},
					{
						InputRef: mustGetBinaryImageRef(
							t,
							ctx,
							refBuilder,
							"a/b/c",
							buffetch.WithGetImageRefOption("none"),
						),
					},
					{
						InputRef: mustGetBinaryImageRef(
							t,
							ctx,
							refBuilder,
							"/dev/null",
						),
					},
					{
						InputRef: mustGetBinaryImageRef(
							t,
							ctx,
							refBuilder,
							"-",
						),
					},
				},
			},
		},
		{
			testName: "Test binary plugin",
			file:     filepath.Join("plugin", "binary_success"),
			expectedConfig: &Config{
				Plugins: []bufgenplugin.PluginConfig{
					mustCreateBinaryPlugin(
						t,
						"",
						[]string{"protoc-gen-go", "arg1", "arg2"},
						internal.StrategyDirectory,
						"gen/out/bin",
						"paths=source_relative,foo=bar,baz",
						true,
						true,
					),
					mustCreateBinaryPlugin(
						t,
						"",
						[]string{"./relative", "argX", "argY"},
						internal.StrategyAll,
						"gen/out/bin2",
						"a=b,x=y",
						false,
						false,
					),
					mustCreateBinaryPlugin(
						t,
						"",
						[]string{"/absolute-plugin"},
						internal.StrategyDirectory,
						"/some/out",
						"paths=source_relative",
						false,
						false,
					),
					mustCreateBinaryPlugin(
						t,
						"",
						[]string{"protoc-gen-go"},
						internal.StrategyDirectory,
						"gen/out",
						"",
						false,
						false,
					),
				},
			},
		},
		{
			testName: "Test protoc built-in plugin",
			file:     filepath.Join("plugin", "protoc_success"),
			expectedConfig: &Config{
				Plugins: []bufgenplugin.PluginConfig{
					mustCreateProtocBuiltinPluginConfig(
						t,
						"java",
						"relative/protoc",
						"gen/out/builtin",
						"paths=source_relative",
						true,
						true,
						internal.StrategyDirectory,
					),
					mustCreateProtocBuiltinPluginConfig(
						t,
						"cpp",
						"/absolute",
						"gen/out/builtin2",
						"a=b,x=y",
						false,
						false,
						internal.StrategyAll,
					),
					mustCreateProtocBuiltinPluginConfig(
						t,
						"cpp",
						"",
						"gen/out",
						"",
						false,
						false,
						internal.StrategyDirectory,
					),
				},
			},
		},
		{
			testName: "Test remote plugins",
			file:     filepath.Join("plugin", "remote_success"),
			expectedConfig: &Config{
				Plugins: []bufgenplugin.PluginConfig{
					mustCreateNewCuratedPluginConfig(
						t,
						"buf.build/protocolbuffers/go",
						2,
						"gen/out/remote",
						"paths=source_relative",
						true,
						true,
					),
					mustCreateNewCuratedPluginConfig(
						t,
						"buf.build/bufbuild/connect-go:v1.8.0",
						0,
						"gen/out/remote2",
						"a=b,x=y",
						false,
						false,
					),
					mustCreateNewCuratedPluginConfig(
						t,
						"buf.build/protocolbuffers/go",
						0,
						"gen/out",
						"",
						false,
						false,
					),
				},
			},
		},
	}

	for _, test := range tests {
		test := test
		for _, fileExtension := range []string{".yaml", ".json"} {
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
				require.Nil(t, err)
				require.Equal(t, test.expectedConfig, config)
				data, err := os.ReadFile(file)
				require.NoError(t, err)
				config, err = readConfigV2(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
				require.NoError(t, err)
				require.Equal(t, test.expectedConfig, config)
			})
		}
	}
}

func TestManagedConfig(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	nopLogger := zap.NewNop()
	readBucket, err := storagemem.NewReadBucket(nil)
	require.NoError(t, err)
	tests := []struct {
		testName                string
		file                    string
		expectedDisableResults  map[FileOption]map[ImageFileIdentity]bool
		expectedOverrideResults map[FileOption]map[ImageFileIdentity]bufimagemodifyv2.Override
	}{
		{
			testName: "test override and disable",
			file:     filepath.Join("managed", "java_package"),
			expectedDisableResults: map[FileOption]map[ImageFileIdentity]bool{
				FileOptionJavaPackage: {
					&fakeImageFileIdentity{
						path: "ok.proto",
					}: true,
					&fakeImageFileIdentity{
						path: "ok.protooo",
					}: false,
					&fakeImageFileIdentity{
						path: "notok.proto",
					}: false,
					&fakeImageFileIdentity{
						path: "a/b/x.proto",
					}: true,
					&fakeImageFileIdentity{
						path: "a/y.proto",
					}: true,
					&fakeImageFileIdentity{
						path: "m/x.proto",
					}: false,
					&fakeImageFileIdentity{
						path: "m/n/y.proto",
					}: false,
					&fakeImageFileIdentity{
						path: "a.proto",
					}: false,
					&fakeImageFileIdentity{
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
					}: false,
					&fakeImageFileIdentity{}: false,
				},
			},
			expectedOverrideResults: map[FileOption]map[ImageFileIdentity]bufimagemodifyv2.Override{
				FileOptionJavaPackage: {
					&fakeImageFileIdentity{
						path: "file.proto",
					}: bufimagemodifyv2.NewPrefixOverride("workspace.default"),
					&fakeImageFileIdentity{
						module: mustCreateModuleIdentity(t, "buf.build", "owner1", "mod1"),
					}: bufimagemodifyv2.NewPrefixOverride("workspace.default"),
					&fakeImageFileIdentity{
						path: "b/c/d/file.proto",
					}: bufimagemodifyv2.NewPrefixOverride("bcd.prefix"),
					&fakeImageFileIdentity{
						path: "b/c/d/x.proto",
					}: bufimagemodifyv2.NewValueOverride("x.override"),
					&fakeImageFileIdentity{
						path: "b/c/d/e.proto",
					}: bufimagemodifyv2.NewPrefixOverride("bcd.prefix"),
					&fakeImageFileIdentity{
						path: "b/c/d/e/file.proto",
					}: bufimagemodifyv2.NewPrefixOverride("bcde"),
					&fakeImageFileIdentity{
						path: "b/c/d/e/f.proto",
					}: bufimagemodifyv2.NewPrefixOverride("bcde"),
					&fakeImageFileIdentity{
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
						path:   "b/c/d/e/f.proto",
					}: bufimagemodifyv2.NewValueOverride("override.value.bcdef"),
				},
			},
		},
		{
			testName: "test cc enable arenas success",
			file:     filepath.Join("managed", "cc_enable_arenas"),
			expectedDisableResults: map[FileOption]map[ImageFileIdentity]bool{
				FileOptionCcEnableArenas: {
					&fakeImageFileIdentity{
						path: "ok.proto",
					}: true,
					&fakeImageFileIdentity{
						path: "ok.protooo",
					}: false,
					&fakeImageFileIdentity{
						path: "notok.proto",
					}: false,
					&fakeImageFileIdentity{
						path: "a/b/x.proto",
					}: true,
					&fakeImageFileIdentity{
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
					}: false,
					&fakeImageFileIdentity{
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "petapis"),
					}: true,
					&fakeImageFileIdentity{
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "petapis2"),
					}: false,
					&fakeImageFileIdentity{
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "petapis2"),
						path:   "x/y.proto",
					}: true,
					&fakeImageFileIdentity{}: false,
				},
			},
			expectedOverrideResults: map[FileOption]map[ImageFileIdentity]bufimagemodifyv2.Override{
				FileOptionCcEnableArenas: {
					&fakeImageFileIdentity{
						path: "file.proto",
					}: bufimagemodifyv2.NewValueOverride(true),
					&fakeImageFileIdentity{
						path: "m/n/a.proto",
					}: bufimagemodifyv2.NewValueOverride(true),
					&fakeImageFileIdentity{
						path: "m/k/b.proto",
					}: bufimagemodifyv2.NewValueOverride(false),
				},
			},
		},
		{
			testName: "test csharp namespace success",
			file:     filepath.Join("managed", "csharp_namespace"),
			expectedOverrideResults: map[FileOption]map[ImageFileIdentity]bufimagemodifyv2.Override{
				FileOptionCsharpNamespace: {
					&fakeImageFileIdentity{
						path: "file.proto",
					}: bufimagemodifyv2.NewValueOverride("ValueDefault"),
					&fakeImageFileIdentity{
						path: "dir1/a.proto",
					}: bufimagemodifyv2.NewValueOverride("ValuePath"),
					&fakeImageFileIdentity{
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
					}: bufimagemodifyv2.NewValueOverride("ValueMod"),
				},
			},
		},
		{
			testName: "test go package success",
			file:     filepath.Join("managed", "go_package"),
			expectedOverrideResults: map[FileOption]map[ImageFileIdentity]bufimagemodifyv2.Override{
				FileOptionGoPackage: {
					&fakeImageFileIdentity{
						path: "file.proto",
					}: bufimagemodifyv2.NewValueOverride("val"),
					&fakeImageFileIdentity{
						path: "dir1/a.proto",
					}: bufimagemodifyv2.NewPrefixOverride("pre"),
				},
			},
		},
		{
			testName: "test java multiple files",
			file:     filepath.Join("managed", "go_package"),
			expectedOverrideResults: map[FileOption]map[ImageFileIdentity]bufimagemodifyv2.Override{
				FileOptionGoPackage: {
					&fakeImageFileIdentity{
						path: "file.proto",
					}: bufimagemodifyv2.NewValueOverride("val"),
					&fakeImageFileIdentity{
						path: "dir1/a.proto",
					}: bufimagemodifyv2.NewPrefixOverride("pre"),
				},
			},
		},
		{
			testName: "test java outer classname",
			file:     filepath.Join("managed", "java_outer_classname"),
			expectedOverrideResults: map[FileOption]map[ImageFileIdentity]bufimagemodifyv2.Override{
				FileOptionJavaOuterClassname: {
					&fakeImageFileIdentity{
						path: "file.proto",
					}: bufimagemodifyv2.NewValueOverride("OverrideVal"),
				},
			},
		},
		{
			testName: "test java string check utf 8",
			file:     filepath.Join("managed", "java_string_check_utf8"),
			expectedOverrideResults: map[FileOption]map[ImageFileIdentity]bufimagemodifyv2.Override{
				FileOptionJavaStringCheckUtf8: {
					&fakeImageFileIdentity{
						path: "file.proto",
					}: bufimagemodifyv2.NewValueOverride(true),
					&fakeImageFileIdentity{
						path: "dir1/a.proto",
					}: bufimagemodifyv2.NewValueOverride(false),
					&fakeImageFileIdentity{
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "petapis"),
					}: bufimagemodifyv2.NewValueOverride(true),
				},
			},
		},
		{
			testName: "test objc prefix",
			file:     filepath.Join("managed", "objc_class_prefix"),
			expectedOverrideResults: map[FileOption]map[ImageFileIdentity]bufimagemodifyv2.Override{
				FileOptionObjcClassPrefix: {
					&fakeImageFileIdentity{
						path: "file.proto",
					}: bufimagemodifyv2.NewValueOverride("pre"),
				},
			},
		},
		{
			testName: "test optimize for",
			file:     filepath.Join("managed", "optimize_for"),
			expectedOverrideResults: map[FileOption]map[ImageFileIdentity]bufimagemodifyv2.Override{
				FileOptionOptimizeFor: {
					&fakeImageFileIdentity{
						path: "file.proto",
					}: bufimagemodifyv2.NewValueOverride(descriptorpb.FileOptions_CODE_SIZE),
					&fakeImageFileIdentity{
						path: "dir1/a.proto",
					}: bufimagemodifyv2.NewValueOverride(descriptorpb.FileOptions_LITE_RUNTIME),
					&fakeImageFileIdentity{
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
					}: bufimagemodifyv2.NewValueOverride(descriptorpb.FileOptions_SPEED),
				},
			},
		},
		{
			testName: "test php metadata namespace",
			file:     filepath.Join("managed", "php_metadata_namespace"),
			expectedOverrideResults: map[FileOption]map[ImageFileIdentity]bufimagemodifyv2.Override{
				FileOptionPhpMetadataNamespace: {
					&fakeImageFileIdentity{
						path: "file.proto",
					}: bufimagemodifyv2.NewValueOverride("namespaceValue"),
				},
			},
		},
		{
			testName: "test php namespace",
			file:     filepath.Join("managed", "php_namespace"),
			expectedOverrideResults: map[FileOption]map[ImageFileIdentity]bufimagemodifyv2.Override{
				FileOptionPhpNamespace: {
					&fakeImageFileIdentity{
						path: "file.proto",
					}: bufimagemodifyv2.NewValueOverride("namespaceValue"),
				},
			},
		},
	}

	for _, test := range tests {
		test := test
		for _, fileExtension := range []string{".yaml", ".json"} {
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
				for fileOption, resultsForFiles := range test.expectedOverrideResults {
					for imageFile, expectedOverride := range resultsForFiles {
						overrideFunc, ok := config.Managed.FileOptionToOverrideFunc[fileOption]
						require.True(t, ok)
						actual := overrideFunc(imageFile)
						require.Equal(
							t,
							expectedOverride,
							actual,
							"%v override for %v should be %v, not %v", fileOption, imageFile, expectedOverride, actual,
						)
					}
				}
			})
		}
	}
}

type fakeImageFileIdentity struct {
	path   string
	module bufmoduleref.ModuleIdentity
}

func (f *fakeImageFileIdentity) Path() string {
	return f.path
}

func (f *fakeImageFileIdentity) ModuleIdentity() bufmoduleref.ModuleIdentity {
	return f.module
}

func mustCreateModuleIdentity(
	t *testing.T,
	remote string,
	owner string,
	repository string,
) bufmoduleref.ModuleIdentity {
	moduleIdentity, err := bufmoduleref.NewModuleIdentity(remote, owner, repository)
	require.NoError(t, err)
	return moduleIdentity
}

func TestConfigError(t *testing.T) {
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
		{
			testName:      "Test subdir is validated for Git",
			file:          filepath.Join("input", "git_error2"),
			expectedError: "/a/b: expected to be relative",
		},
		{
			testName:      "Test two input types not allowed",
			file:          filepath.Join("input", "two_types_error"),
			expectedError: "each input can only have one format, already have format json_image",
		},
		{
			testName:      "Test input type required",
			file:          filepath.Join("input", "no_type_error"),
			expectedError: "must specify input type",
		},
		{
			testName:      "Test depth not allowed for module",
			file:          filepath.Join("input", "module_error1"),
			expectedError: newOptionNotAllowedForInputMessage("depth", "module"),
		},
		{
			testName:      "Test recurse_submodules not allowed for module",
			file:          filepath.Join("input", "module_error2"),
			expectedError: newOptionNotAllowedForInputMessage("recurse_submodules", "module"),
		},
		{
			testName:      "Test tag is not allowed for proto file",
			file:          filepath.Join("input", "proto_file_error1"),
			expectedError: newOptionNotAllowedForInputMessage("tag", "proto_file"),
		},
		{
			testName:      "Test subdir is not allowed for directory",
			file:          filepath.Join("input", "dir_error1"),
			expectedError: newOptionNotAllowedForInputMessage("subdir", "directory"),
		},
		{
			testName:      "Test stdin is not allowed for directory",
			file:          filepath.Join("input", "dir_error2"),
			expectedError: `invalid directory path: "-"`,
		},
		{
			testName:      "Test invalid compression not allowed",
			file:          filepath.Join("input", "tar_error1"),
			expectedError: `unknown compression: "abcd" (valid values are "none,gzip,zstd")`,
		},
		{
			testName:      "Test depth not allowed for tarball",
			file:          filepath.Join("input", "tar_error2"),
			expectedError: newOptionNotAllowedForInputMessage("depth", "tarball"),
		},
		{
			testName:      "Test subdir is validated for tarball",
			file:          filepath.Join("input", "tar_error3"),
			expectedError: "/x/y: expected to be relative",
		},
		{
			testName:      "Test compression not allowed for zip archive",
			file:          filepath.Join("input", "zip_error1"),
			expectedError: newOptionNotAllowedForInputMessage("compression", "zip_archive"),
		},
		{
			testName:      "Test subdir is validated for zip archive",
			file:          filepath.Join("input", "zip_error2"),
			expectedError: "/m/n: expected to be relative",
		},
		{
			testName:      "Test compression is validated for JSON image",
			file:          filepath.Join("input", "json_error1"),
			expectedError: `unknown compression: "xyz" (valid values are "none,gzip,zstd")`,
		},
		{
			testName:      "Test include package files is not allowed for JSON image",
			file:          filepath.Join("input", "json_error2"),
			expectedError: newOptionNotAllowedForInputMessage("include_package_files", "json_image"),
		},
		{
			testName:      "Test parsing invalid strategy",
			file:          filepath.Join("plugin", "binary_error1"),
			expectedError: newUnknowStategyMessage("invalid"),
		},
		{
			testName:      "Test include imports and include WKT for binary plugin",
			file:          filepath.Join("plugin", "binary_error2"),
			expectedError: "cannot include well-known types without including imports",
		},
		{
			testName:      "Test revision not allowed for binary plugin",
			file:          filepath.Join("plugin", "binary_error3"),
			expectedError: newOptionNotAllowedForPluginMessage("revision", "binary"),
		},
		{
			testName:      "Test protoc path not allowed for binary plugin",
			file:          filepath.Join("plugin", "binary_error4"),
			expectedError: newOptionNotAllowedForPluginMessage("protoc_path", "binary"),
		},
		{
			testName:      "Test invalid strategy for protoc built-in plugin",
			file:          filepath.Join("plugin", "protoc_error1"),
			expectedError: "unknown strategy: invalid",
		},
		{
			testName:      "Test invalid include imports and include WKT for protoc built-in plugin",
			file:          filepath.Join("plugin", "protoc_error2"),
			expectedError: "cannot include well-known types without including imports",
		},
		{
			testName:      "Test revision with protoc built-in plugin",
			file:          filepath.Join("plugin", "protoc_error3"),
			expectedError: newOptionNotAllowedForPluginMessage("revision", "protoc_builtin"),
		},
		{
			testName:      "Test invalid revision for remote plugin",
			file:          filepath.Join("plugin", "remote_error1"),
			expectedError: "revision -1 is out of accepted range 0-2147483647",
		},
		{
			testName:      "Test invalid include imports and include WKT for remote plugin",
			file:          filepath.Join("plugin", "remote_error2"),
			expectedError: "cannot include well-known types without including imports",
		},
		{
			testName:      "Test strategy for remote plugin",
			file:          filepath.Join("plugin", "remote_error3"),
			expectedError: newOptionNotAllowedForPluginMessage("strategy", "remote"),
		},
		{
			testName:      "Test protoc path for remote plugin",
			file:          filepath.Join("plugin", "remote_error4"),
			expectedError: newOptionNotAllowedForPluginMessage("protoc_path", "remote"),
		},
		{
			testName:      "Test bool is not allowed for java package",
			file:          filepath.Join("managed", "java_package_error_1"),
			expectedError: newInvalidOverrideMessage("java_package"),
		},
		{
			testName:      "Test prefix and value cannot be both present",
			file:          filepath.Join("managed", "java_package_error_2"),
			expectedError: "only one of value and prefix can be set for java_package",
		},
		{
			testName:      "Test one of prefix and value must be set",
			file:          filepath.Join("managed", "java_package_error_3"),
			expectedError: "must provide override value or prefix for java_package",
		},
		{
			testName:      "Test value must be bool for cc enable arenas",
			file:          filepath.Join("managed", "cc_enable_arenas_error_1"),
			expectedError: newInvalidOverrideMessage("cc_enable_arenas"),
		},
		{
			testName:      "Test prefix cannot appear cc enable arenas",
			file:          filepath.Join("managed", "cc_enable_arenas_error_2"),
			expectedError: newPrefixNotAllowedMessage("cc_enable_arenas"),
		},
		{
			testName:      "Test prefix cannot appear along with value for cc enable arenas",
			file:          filepath.Join("managed", "cc_enable_arenas_error_3"),
			expectedError: newPrefixNotAllowedMessage("cc_enable_arenas"),
		},
		{
			testName:      "Test prefix not allowed for csharp namespace",
			file:          filepath.Join("managed", "csharp_namespace_error_1"),
			expectedError: newPrefixNotAllowedMessage("csharp_namespace"),
		},
		{
			testName:      "Test prefix not allowed for csharp namespace",
			file:          filepath.Join("managed", "csharp_namespace_error_2"),
			expectedError: newInvalidOverrideMessage("csharp_namespace"),
		},
		{
			testName:      "Test bool not allowed for go package",
			file:          filepath.Join("managed", "go_package_error_1"),
			expectedError: newInvalidOverrideMessage("go_package"),
		},
		{
			testName:      "Test prefix not allowed for java multiple files",
			file:          filepath.Join("managed", "java_multiple_files_error"),
			expectedError: newPrefixNotAllowedMessage("java_multiple_files"),
		},
		{
			testName:      "Test prefix not allowed for java outer classname",
			file:          filepath.Join("managed", "java_outer_classname_error"),
			expectedError: newPrefixNotAllowedMessage("java_outer_classname"),
		},
		{
			testName:      "Test prefix not allowed for java string check utf8",
			file:          filepath.Join("managed", "java_string_check_utf8_error_1"),
			expectedError: newPrefixNotAllowedMessage("java_string_check_utf8"),
		},
		{
			testName:      "Test integer not allowed for java string check utf8",
			file:          filepath.Join("managed", "java_string_check_utf8_error_2"),
			expectedError: newInvalidOverrideMessage("java_string_check_utf8"),
		},
		{
			testName:      "Test prefix not allowed for objc class prefix",
			file:          filepath.Join("managed", "objc_class_prefix_error_1"),
			expectedError: newPrefixNotAllowedMessage("objc_class_prefix"),
		},
		{
			testName:      "Test prefix not allowed for optmize for",
			file:          filepath.Join("managed", "optimize_for_error_1"),
			expectedError: newPrefixNotAllowedMessage("optimize_for"),
		},
		{
			testName:      "Test random string not allowed for optimze for",
			file:          filepath.Join("managed", "optimize_for_error_2"),
			expectedError: "optimize_for: xyz is not a valid optmize_for value",
		},
		{
			testName:      "Test bool is not allowed for optimize for",
			file:          filepath.Join("managed", "optimize_for_error_3"),
			expectedError: newInvalidOverrideMessage("optimize_for"),
		},
		{
			testName:      "Test prefix is not allowed for php metadata namespace",
			file:          filepath.Join("managed", "php_metadata_namespace_error_1"),
			expectedError: newPrefixNotAllowedMessage("php_metadata_namespace"),
		},
		{
			testName:      "Test prefix is not allowed for php namespace",
			file:          filepath.Join("managed", "php_namespace_error_1"),
			expectedError: newPrefixNotAllowedMessage("php_namespace"),
		},
	}

	for _, test := range tests {
		test := test
		for _, fileExtension := range []string{".yaml", ".json"} {
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

func mustGetGitRef(t *testing.T, ctx context.Context, refBuilder buffetch.RefBuilder, path string, options ...buffetch.GetGitRefOption) buffetch.Ref {
	ref, err := refBuilder.GetGitRef(ctx, "git_repo", path, options...)
	require.NoError(t, err)
	return ref
}

func mustGetModuleRef(t *testing.T, ctx context.Context, refBuilder buffetch.RefBuilder, path string) buffetch.Ref {
	ref, err := refBuilder.GetModuleRef(ctx, "module", path)
	require.NoError(t, err)
	return ref
}

func mustGetProtoFileRef(t *testing.T, ctx context.Context, refBuilder buffetch.RefBuilder, path string, options ...buffetch.GetProtoFileRefOption) buffetch.Ref {
	ref, err := refBuilder.GetProtoFileRef(ctx, "proto_file", path, options...)
	require.NoError(t, err)
	return ref
}

func mustGetDirRef(t *testing.T, ctx context.Context, refBuilder buffetch.RefBuilder, path string) buffetch.Ref {
	ref, err := refBuilder.GetDirRef(ctx, "directory", path)
	require.NoError(t, err)
	return ref
}

func mustGetTarballRef(t *testing.T, ctx context.Context, refBuilder buffetch.RefBuilder, path string, options ...buffetch.GetTarballRefOption) buffetch.Ref {
	ref, err := refBuilder.GetTarballRef(ctx, "tarball", path, options...)
	require.NoError(t, err)
	return ref
}

func mustGetZipArchiveRef(t *testing.T, ctx context.Context, refBuilder buffetch.RefBuilder, path string, options ...buffetch.GetZipArchiveRefOption) buffetch.Ref {
	ref, err := refBuilder.GetZipArchiveRef(ctx, "zip_archive", path, options...)
	require.NoError(t, err)
	return ref
}

func mustGetBinaryImageRef(t *testing.T, ctx context.Context, refBuilder buffetch.RefBuilder, path string, options ...buffetch.GetImageRefOption) buffetch.Ref {
	ref, err := refBuilder.GetBinaryImageRef(ctx, "binary_image", path, options...)
	require.NoError(t, err)
	return ref
}

func mustGetJSONImageRef(t *testing.T, ctx context.Context, refBuilder buffetch.RefBuilder, path string, options ...buffetch.GetImageRefOption) buffetch.Ref {
	ref, err := refBuilder.GetJSONImageRef(ctx, "json_image", path, options...)
	require.NoError(t, err)
	return ref
}

func mustCreateBinaryPlugin(
	t *testing.T,
	name string,
	path []string,
	strategy internal.Strategy,
	out string,
	opt string,
	includeImports bool,
	includeWKT bool,
) bufgenplugin.BinaryPluginConfig {
	config, err := bufgenplugin.NewBinaryPluginConfig(
		name,
		path,
		strategy,
		out,
		opt,
		includeImports,
		includeWKT,
	)
	require.NoError(t, err)
	return config
}

func mustCreateProtocBuiltinPluginConfig(
	t *testing.T,
	name string,
	protocPath string,
	out string,
	opt string,
	includeImports bool,
	includeWKT bool,
	strategy internal.Strategy,
) bufgenplugin.ProtocBuiltinPluginConfig {
	config, err := bufgenplugin.NewProtocBuiltinPluginConfig(
		name,
		protocPath,
		out,
		opt,
		includeImports,
		includeWKT,
		strategy,
	)
	require.NoError(t, err)
	return config
}

func mustCreateNewCuratedPluginConfig(
	t *testing.T,
	plugin string,
	revision int,
	out string,
	opt string,
	includeImports bool,
	includeWKT bool,
) bufgenplugin.CuratedPluginConfig {
	config, err := bufgenplugin.NewCuratedPluginConfig(
		plugin,
		revision,
		out,
		opt,
		includeImports,
		includeImports,
	)
	require.NoError(t, err)
	return config
}

func newOptionNotAllowedForInputMessage(option string, input string) string {
	return fmt.Sprintf("option %s is not allowed for format %s", option, input)
}

func newOptionNotAllowedForPluginMessage(option string, pluginType string) string {
	return fmt.Sprintf("%s is not allowed for %s", option, pluginType)
}

func newUnknowStategyMessage(strategy string) string {
	return fmt.Sprintf("unknown strategy: %s", strategy)
}

func newInvalidOverrideMessage(fileOption string) string {
	return fmt.Sprintf("invalid override for %s", fileOption)
}

func newPrefixNotAllowedMessage(fileOption string) string {
	return fmt.Sprintf("prefix is not allowed for %s", fileOption)
}

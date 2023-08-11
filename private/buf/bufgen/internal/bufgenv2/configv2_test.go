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
			testName: "Test text image",
			file:     filepath.Join("input", "txt_success"),
			expectedConfig: &Config{
				Plugins: placeHolderPlugins,
				Inputs: []*InputConfig{
					{
						InputRef: mustGetTextImageRef(
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
						InputRef: mustGetTextImageRef(
							t,
							ctx,
							refBuilder,
							"a/b/c.txtpb.gz",
							buffetch.WithGetImageRefOption("gzip"),
						),
					},
					{
						InputRef: mustGetTextImageRef(
							t,
							ctx,
							refBuilder,
							"a/b/c.txtpb.zst",
							buffetch.WithGetImageRefOption("zstd"),
						),
					},
					{
						InputRef: mustGetTextImageRef(
							t,
							ctx,
							refBuilder,
							"a/b/c.txtpb",
							buffetch.WithGetImageRefOption("none"),
						),
					},
					{
						InputRef: mustGetTextImageRef(
							t,
							ctx,
							refBuilder,
							"a/b/c",
							buffetch.WithGetImageRefOption("none"),
						),
					},
					{
						InputRef: mustGetTextImageRef(
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

func TestManagedConfigSuccess(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	nopLogger := zap.NewNop()
	readBucket, err := storagemem.NewReadBucket(nil)
	require.NoError(t, err)
	type fileAndField struct {
		imageFileIdentity fakeImageFileIdentity
		field             string
	}
	tests := []struct {
		testName string
		file     string
		// true means disabled
		expectedDisableResults       map[FileOption]map[imageFileIdentity]bool
		expectedOverrideResults      map[fileOptionGroup]map[imageFileIdentity]override
		expectedFieldDisableResults  map[fieldOption]map[fileAndField]bool
		expectedFieldOverrideResults map[fieldOption]map[fileAndField]override
	}{
		{
			testName: "test override and disable matching",
			file:     filepath.Join("managed", "match"),
			expectedDisableResults: map[FileOption]map[imageFileIdentity]bool{
				FileOptionJavaPackage: {
					&fakeImageFileIdentity{
						path: "excluded/a.proto",
					}: true,
					&fakeImageFileIdentity{
						path: "notexcluded/a.proto",
					}: false,
					&fakeImageFileIdentity{
						path: "java-excluded/x/a.proto",
					}: true,
					&fakeImageFileIdentity{
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
						path:   "random.proto",
					}: true,
					&fakeImageFileIdentity{
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "petapis"),
						path:   "random.proto",
					}: false,
					&fakeImageFileIdentity{
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "petapis"),
						path:   "x/y/z.proto",
					}: true,
					&fakeImageFileIdentity{
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "random"),
						path:   "x/y/z.proto",
					}: false,
					&fakeImageFileIdentity{
						path: "exact.proto",
					}: true,
					&fakeImageFileIdentity{}: false,
				},
			},
			expectedOverrideResults: map[fileOptionGroup]map[imageFileIdentity]override{
				groupJavaPackage: {
					&fakeImageFileIdentity{
						path: "file.proto",
					}: newValueOverride("global.default"),
					&fakeImageFileIdentity{
						module: mustCreateModuleIdentity(t, "buf.build", "owner1", "mod1"),
					}: newValueOverride("global.default"),
					&fakeImageFileIdentity{
						path: "b/c/d/file.proto",
					}: newValueOverride("bcd.override"),
					&fakeImageFileIdentity{
						path: "b/c/d/x.proto",
					}: newValueOverride("x.override"),
					&fakeImageFileIdentity{
						path: "b/c/d/e.proto",
					}: newValueOverride("bcd.override"),
					&fakeImageFileIdentity{
						path: "b/c/d/e/file.proto",
					}: newValueOverride("net.bcde"),
					&fakeImageFileIdentity{
						path: "b/c/d/e/f.proto",
					}: newValueOverride("net.bcde"),
					&fakeImageFileIdentity{
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
						path:   "b/c/d/e/f.proto",
					}: newValueOverride("override.value.bcdef"),
					&fakeImageFileIdentity{
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
						path:   "m/n/xyz.proto",
					}: newValueOverride("m.override"),
				},
			},
		},
		{
			testName: "test java package options",
			file:     filepath.Join("managed", "java_package"),
			expectedOverrideResults: map[fileOptionGroup]map[imageFileIdentity]override{
				groupJavaPackage: {
					&fakeImageFileIdentity{
						path: "file.proto",
					}: newPrefixSuffixOverride(
						"org",
						"proto",
					),
					&fakeImageFileIdentity{
						path: "special/file.proto",
					}: newPrefixSuffixOverride(
						"special.prefix",
						"special.suffix",
					),
					// test the last two overrides are suffix and value
					&fakeImageFileIdentity{
						path: "a/b/c.proto",
					}: newValueOverride("com.example.pb"),
					&fakeImageFileIdentity{
						path: "special/x/file.proto",
					}: newValueOverride("com.special.x"),
					&fakeImageFileIdentity{
						path: "special/p/file.proto",
					}: newValueOverride("net.example"),
					&fakeImageFileIdentity{
						path: "special/s/file.proto",
					}: newSuffixOverride("s.suffix"),
					// test the last two overrides are both values
					&fakeImageFileIdentity{
						path: "special/p/something.proto",
					}: newValueOverride("com.something"),
					// test the last two overrides are suffix and prefix
					&fakeImageFileIdentity{
						path: "special/s/xyz/file.proto",
					}: newPrefixSuffixOverride(
						"xyz.prefix",
						"s.suffix",
					),
					// test the last two overrides are prefix and suffix
					&fakeImageFileIdentity{
						path: "special/s/xyz/final/file.proto",
					}: newPrefixSuffixOverride(
						"xyz.prefix",
						"final.suffix",
					),
					// test the last two overrides are both prefixes
					&fakeImageFileIdentity{
						path: "double/pre/pre/file.proto",
					}: newPrefixOverride("dp2"),
					// test the last two overrides are both suffixes
					&fakeImageFileIdentity{
						path: "double/suf/suf/file.proto",
					}: newSuffixOverride("ds2"),
					// test the last two overrides are value and prefix
					&fakeImageFileIdentity{
						path: "double/pre/file.proto",
					}: newPrefixOverride("dp1"),
					// test the last two overrides are value and suffix
					&fakeImageFileIdentity{
						path: "double/suf/file.proto",
					}: newSuffixOverride("ds1"),
				},
			},
		},
		{
			testName: "test jstype field option",
			file:     filepath.Join("managed", "jstype"),
			expectedFieldDisableResults: map[fieldOption]map[fileAndField]bool{
				fieldOptionJsType: {
					{
						imageFileIdentity: fakeImageFileIdentity{
							path: "random/random.proto",
						},
						field: "Ignore.me",
					}: true,
					{
						imageFileIdentity: fakeImageFileIdentity{
							path: "random/random.proto",
						},
						field: "Some.otherField",
					}: false,
					{
						imageFileIdentity: fakeImageFileIdentity{
							path: "excluded/a.proto",
						},
						field: "regular.Message1.field_3",
					}: true,
					{
						imageFileIdentity: fakeImageFileIdentity{
							module: mustCreateModuleIdentity(t, "buf.build", "googleapis", "googleapis"),
							path:   "ok.proto",
						},
						field: "regular.Message2.field_4",
					}: true,
					{
						imageFileIdentity: fakeImageFileIdentity{
							module: mustCreateModuleIdentity(t, "buf.build", "acme", "petapis"),
							path:   "ok.proto",
						},
						field: "Foo.bar",
					}: false,
					{
						imageFileIdentity: fakeImageFileIdentity{
							module: mustCreateModuleIdentity(t, "buf.build", "acme", "notjstype"),
							path:   "ok.proto",
						},
						field: "Foo.bar",
					}: true,
					{
						imageFileIdentity: fakeImageFileIdentity{
							module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
							path:   "exact.proto",
						},
						field: "not.the.Specified.fieldToDisable",
					}: false,
					{
						imageFileIdentity: fakeImageFileIdentity{
							module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
							path:   "exact.proto",
						},
						field: "Foo.bar",
					}: true,
					{
						imageFileIdentity: fakeImageFileIdentity{
							module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
							path:   "match/field/but/not/path",
						},
						field: "Foo.bar",
					}: false,
				},
			},
			expectedFieldOverrideResults: map[fieldOption]map[fileAndField]override{
				fieldOptionJsType: {
					{
						imageFileIdentity: fakeImageFileIdentity{
							path: "any/path.proto",
						},
						field: "Regular.field_name",
					}: newValueOverride(descriptorpb.FieldOptions_JS_STRING),
					{
						imageFileIdentity: fakeImageFileIdentity{
							module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
							path:   "any/path.proto",
						},
						field: "Regular.field_name",
					}: newValueOverride(descriptorpb.FieldOptions_JS_NUMBER),
					{
						imageFileIdentity: fakeImageFileIdentity{
							module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
							path:   "b/c/d/x.proto",
						},
						field: "Regular.field_name",
					}: newValueOverride(descriptorpb.FieldOptions_JS_NORMAL),
					{
						imageFileIdentity: fakeImageFileIdentity{
							module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
							path:   "b/c/d/x.proto",
						},
						field: "Should_1.Be_2.num",
					}: newValueOverride(descriptorpb.FieldOptions_JS_NUMBER),
					{
						imageFileIdentity: fakeImageFileIdentity{
							module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
							path:   "b/c/d/x.proto",
						},
						field: "package1.subpackage2.Message1.NestedMessage2.field_5",
					}: newValueOverride(descriptorpb.FieldOptions_JS_NORMAL),
				},
			},
		},
		{
			testName: "java outer class name",
			file:     filepath.Join("managed", "java_outer_classname"),
			expectedOverrideResults: map[fileOptionGroup]map[imageFileIdentity]override{
				groupJavaOuterClassname: {
					&fakeImageFileIdentity{
						path: "random.proto",
					}: newValueOverride("DefaultProto"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "petapis"),
					}: newValueOverride("PathProto"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
					}: newValueOverride("ModProto"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: newValueOverride("PathProto"),
					&fakeImageFileIdentity{
						path:   "a/b/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: newValueOverride("Both"),
				},
			},
		},
		{
			testName: "java multiple files",
			file:     filepath.Join("managed", "java_multiple_files"),
			expectedOverrideResults: map[fileOptionGroup]map[imageFileIdentity]override{
				groupJavaMultipleFiles: {
					&fakeImageFileIdentity{
						path: "random.proto",
					}: newValueOverride(true),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "petapis"),
					}: newValueOverride(false),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
					}: newValueOverride(false),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: newValueOverride(false),
					&fakeImageFileIdentity{
						path:   "a/b/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: newValueOverride(true),
				},
			},
		},
		{
			testName: "java string check utf 8",
			file:     filepath.Join("managed", "java_string_check_utf8"),
			expectedOverrideResults: map[fileOptionGroup]map[imageFileIdentity]override{
				groupJavaStringCheckUtf8: {
					&fakeImageFileIdentity{
						path: "random.proto",
					}: newValueOverride(true),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "petapis"),
					}: newValueOverride(false),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
					}: newValueOverride(false),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: newValueOverride(false),
					&fakeImageFileIdentity{
						path:   "a/b/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: newValueOverride(true),
				},
			},
		},
		{
			testName: "optimize for",
			file:     filepath.Join("managed", "optimize_for"),
			expectedOverrideResults: map[fileOptionGroup]map[imageFileIdentity]override{
				groupOptimizeFor: {
					&fakeImageFileIdentity{
						path: "random.proto",
					}: newValueOverride(descriptorpb.FileOptions_SPEED),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "petapis"),
					}: newValueOverride(descriptorpb.FileOptions_CODE_SIZE),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
					}: newValueOverride(descriptorpb.FileOptions_CODE_SIZE),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: newValueOverride(descriptorpb.FileOptions_CODE_SIZE),
					&fakeImageFileIdentity{
						path:   "a/b/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: newValueOverride(descriptorpb.FileOptions_LITE_RUNTIME),
				},
			},
		},
		{
			testName: "go package",
			file:     filepath.Join("managed", "go_package"),
			expectedOverrideResults: map[fileOptionGroup]map[imageFileIdentity]override{
				groupGoPackage: {
					&fakeImageFileIdentity{
						path: "random.proto",
					}: newPrefixOverride("github.com/example/protos"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "petapis"),
					}: newValueOverride("package1"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
					}: newValueOverride("weather"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: newValueOverride("package1"),
					&fakeImageFileIdentity{
						path:   "a/b/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: newPrefixOverride("special/prefix"),
				},
			},
		},
		{
			testName: "cc enable arenas",
			file:     filepath.Join("managed", "cc_enable_arenas"),
			expectedOverrideResults: map[fileOptionGroup]map[imageFileIdentity]override{
				groupCcEnableArenas: {
					&fakeImageFileIdentity{
						path: "random.proto",
					}: newValueOverride(false),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "petapis"),
					}: newValueOverride(true),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
					}: newValueOverride(true),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: newValueOverride(true),
					&fakeImageFileIdentity{
						path:   "a/b/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: newValueOverride(false),
				},
			},
		},
		{
			testName: "objc class prefix",
			file:     filepath.Join("managed", "objc_class_prefix"),
			expectedOverrideResults: map[fileOptionGroup]map[imageFileIdentity]override{
				groupObjcClassPrefix: {
					&fakeImageFileIdentity{
						path: "random.proto",
					}: newValueOverride("foo"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "petapis"),
					}: newValueOverride("bar"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
					}: newValueOverride("baz"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: newValueOverride("bar"),
					&fakeImageFileIdentity{
						path:   "a/b/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: newValueOverride("qux"),
				},
			},
		},
		{
			testName: "csharp namespace",
			file:     filepath.Join("managed", "csharp_namespace"),
			expectedOverrideResults: map[fileOptionGroup]map[imageFileIdentity]override{
				groupCsharpNamespace: {
					&fakeImageFileIdentity{
						path: "random.proto",
					}: newPrefixOverride("foo"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "petapis"),
					}: newValueOverride("bar"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
					}: newValueOverride("baz"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: newValueOverride("bar"),
					&fakeImageFileIdentity{
						path:   "a/b/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: newPrefixOverride("qux"),
				},
			},
		},
		{
			testName: "php namespace",
			file:     filepath.Join("managed", "php_namespace"),
			expectedOverrideResults: map[fileOptionGroup]map[imageFileIdentity]override{
				groupPhpNamespace: {
					&fakeImageFileIdentity{
						path: "random.proto",
					}: newValueOverride(`Foo\Bar`),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "petapis"),
					}: newValueOverride(`Bar\Baz`),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
					}: newValueOverride(`Baz\Qux`),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: newValueOverride(`Bar\Baz`),
					&fakeImageFileIdentity{
						path:   "a/b/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: newValueOverride(`Qux\Quux`),
				},
			},
		},
		{
			testName: "php metadata namespace",
			file:     filepath.Join("managed", "php_metadata_namespace"),
			expectedOverrideResults: map[fileOptionGroup]map[imageFileIdentity]override{
				groupPhpMetadataNamespace: {
					&fakeImageFileIdentity{
						path: "random.proto",
					}: newSuffixOverride("DefaultMetadata"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "petapis"),
					}: newValueOverride(`Foo\Bar`),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
					}: newValueOverride(`Bar\Baz`),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: newValueOverride(`Foo\Bar`),
					&fakeImageFileIdentity{
						path:   "a/b/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: newSuffixOverride("SpecialMetadata"),
				},
			},
		},
		{
			testName: "ruby package",
			file:     filepath.Join("managed", "ruby_package"),
			expectedOverrideResults: map[fileOptionGroup]map[imageFileIdentity]override{
				groupRubyPackage: {
					&fakeImageFileIdentity{
						path: "random.proto",
					}: newSuffixOverride("protos"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "petapis"),
					}: newValueOverride("Foo::Bar"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "weather"),
					}: newValueOverride("Bar::Baz"),
					&fakeImageFileIdentity{
						path:   "dir/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: newValueOverride("Foo::Bar"),
					&fakeImageFileIdentity{
						path:   "a/b/a.proto",
						module: mustCreateModuleIdentity(t, "buf.build", "acme", "payment"),
					}: newSuffixOverride("pbs"),
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
				for fileOption, resultsForFiles := range test.expectedOverrideResults {
					for imageFile, expectedOverride := range resultsForFiles {
						overrideFunc, ok := config.Managed.FileOptionGroupToOverrideFunc[fileOption]
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
				for fieldOption, fileAndFieldToResults := range test.expectedFieldDisableResults {
					for fileAndField, expectedResult := range fileAndFieldToResults {
						actual := config.Managed.FieldDisableFunc(
							fieldOption,
							&fileAndField.imageFileIdentity,
							fileAndField.field,
						)
						require.Equal(
							t,
							expectedResult,
							actual,
							"whether to disable %v for %v should be %v", fieldOption, fileAndField, expectedResult,
						)
					}
				}
				for fieldOption, fileAndFieldToOverrides := range test.expectedFieldOverrideResults {
					for fileAndField, expectedOverride := range fileAndFieldToOverrides {
						overrideFunc, ok := config.Managed.FieldOptionToOverrideFunc[fieldOption]
						require.True(t, ok)
						actual := overrideFunc(&fileAndField.imageFileIdentity, fileAndField.field)
						require.Equal(
							t,
							expectedOverride,
							actual,
							"override for %v for %v should be %v", fieldOption, fileAndField, expectedOverride,
						)
					}
				}
			})
		}
	}
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
			testName:      "Test dev null is not allowed for json image",
			file:          filepath.Join("input", "json_error3"),
			expectedError: "/dev/null is not allowed for json_image",
		},
		{
			testName:      "Test compression is validated for text image",
			file:          filepath.Join("input", "txt_error1"),
			expectedError: `unknown compression: "xyz" (valid values are "none,gzip,zstd")`,
		},
		{
			testName:      "Test depth is not allowed for text image",
			file:          filepath.Join("input", "txt_error2"),
			expectedError: newOptionNotAllowedForInputMessage("depth", "text_image"),
		},
		{
			testName:      "Test dev null is not allowed for text image",
			file:          filepath.Join("input", "txt_error3"),
			expectedError: "/dev/null is not allowed for text_image",
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
			testName:      "Test disable has none of path module and file option",
			file:          filepath.Join("managed", "invalid_disable"),
			expectedError: "must set one of file_option, field option, module, path and field for a disable rule",
		},
		{
			testName:      "Test override has none of path module and file option",
			file:          filepath.Join("managed", "invalid_override"),
			expectedError: "must set one of file option and field option to override",
		},
		{
			testName:      "Test invalid field option",
			file:          filepath.Join("managed", "invalid_field_option"),
			expectedError: `unknown field option: "not_a_real_field_option"`,
		},
		{
			testName:      "Test invalid jstype value",
			file:          filepath.Join("managed", "invalid_jstype_value"),
			expectedError: `"not_a_valid_jstype_value" is not a valid jstype value, must be one of JS_NORMAL, JS_STRING and JS_NUMBER`,
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

func mustGetTextImageRef(t *testing.T, ctx context.Context, refBuilder buffetch.RefBuilder, path string, options ...buffetch.GetImageRefOption) buffetch.Ref {
	ref, err := refBuilder.GetTextImageRef(ctx, "text_image", path, options...)
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

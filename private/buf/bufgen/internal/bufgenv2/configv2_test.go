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
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestConfigSuccess(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	nopLogger := zap.NewNop()
	refBuilder := buffetch.NewRefBuilder()
	readBucket, err := storagemem.NewReadBucket(nil)
	require.NoError(t, err)
	placeHolderPlugins := []internal.PluginConfig{
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
				Plugins: []internal.PluginConfig{
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
				Plugins: []internal.PluginConfig{
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
				Plugins: []internal.PluginConfig{
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
				config, err := ReadConfigV2(
					ctx,
					nopLogger,
					readBucket,
					internal.ReadConfigWithOverride(file),
				)
				require.Nil(t, err)
				require.Equal(t, test.expectedConfig, config)
				data, err := os.ReadFile(file)
				require.NoError(t, err)
				config, err = ReadConfigV2(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
				require.NoError(t, err)
				require.Equal(t, test.expectedConfig, config)
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
	}

	for _, test := range tests {
		test := test
		for _, fileExtension := range []string{".yaml", ".json"} {
			fileExtension := fileExtension
			t.Run(fmt.Sprintf("%s with extension %s", test.testName, fileExtension), func(t *testing.T) {
				t.Parallel()
				file := filepath.Join("testdata", test.file+fileExtension)
				config, err := ReadConfigV2(
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
) internal.BinaryPluginConfig {
	config, err := internal.NewBinaryPluginConfig(
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
) internal.ProtocBuiltinPluginConfig {
	config, err := internal.NewProtocBuiltinPluginConfig(
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
) internal.CuratedPluginConfig {
	config, err := internal.NewCuratedPluginConfig(
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

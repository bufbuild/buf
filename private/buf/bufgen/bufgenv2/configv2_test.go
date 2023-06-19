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
	"github.com/bufbuild/buf/private/buf/bufgen"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/bufimagemodifyv2"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestIntputConfigSuccess(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	nopLogger := zap.NewNop()
	refBuilder := buffetch.NewRefBuilder()
	provider := bufgen.NewConfigDataProvider(zap.NewNop())
	readBucket, err := storagemem.NewReadBucket(nil)
	require.NoError(t, err)

	tests := []struct {
		testName       string
		file           string
		expectedConfig *Config
	}{
		{
			testName: "Test git",
			file:     "git_success",
			expectedConfig: &Config{
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
			file:     "module_success",
			expectedConfig: &Config{
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
			file:     "proto_file_success",
			expectedConfig: &Config{
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
			file:     "dir_success",
			expectedConfig: &Config{
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
			file:     "tar_success",
			expectedConfig: &Config{
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
			file:     "zip_success",
			expectedConfig: &Config{
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
			file:     "json_success",
			expectedConfig: &Config{
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
			file:     "bin_success",
			expectedConfig: &Config{
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
	}

	for _, test := range tests {
		test := test
		for _, fileExtension := range []string{".yaml", ".json"} {
			fileExtension := fileExtension
			t.Run(fmt.Sprintf("%s with extension %s", test.testName, fileExtension), func(t *testing.T) {
				t.Parallel()
				file := filepath.Join("testdata", "input", test.file+fileExtension)
				config, err := ReadConfigV2(
					ctx,
					nopLogger,
					provider,
					readBucket,
					bufgen.ReadConfigWithOverride(file),
				)
				require.Nil(t, err)
				require.Equal(t, test.expectedConfig, config)
				data, err := os.ReadFile(file)
				require.NoError(t, err)
				config, err = ReadConfigV2(ctx, nopLogger, provider, readBucket, bufgen.ReadConfigWithOverride(string(data)))
				require.NoError(t, err)
				require.Equal(t, test.expectedConfig, config)
			})
		}
	}
}

func TestInputConfigError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	nopLogger := zap.NewNop()
	provider := bufgen.NewConfigDataProvider(zap.NewNop())
	readBucket, err := storagemem.NewReadBucket(nil)
	require.NoError(t, err)

	tests := []struct {
		testName      string
		file          string
		expectedError string
	}{
		{
			testName:      "Test compression is not allowed for Git",
			file:          "git_error1",
			expectedError: newOptionNotAllowedForIntMessage("compression", "git_repo"),
		},
		{
			testName:      "Test subdir is validated for Git",
			file:          "git_error2",
			expectedError: "/a/b: expected to be relative",
		},
		{
			testName:      "Test two input types not allowed",
			file:          "two_types_error",
			expectedError: "each input can only have one format, already have format json_image",
		},
		{
			testName:      "Test input type required",
			file:          "no_type_error",
			expectedError: "must specify input type",
		},
		{
			testName:      "Test depth not allowed for module",
			file:          "module_error1",
			expectedError: newOptionNotAllowedForIntMessage("depth", "module"),
		},
		{
			testName:      "Test recurse_submodules not allowed for module",
			file:          "module_error2",
			expectedError: newOptionNotAllowedForIntMessage("recurse_submodules", "module"),
		},
		{
			testName:      "Test tag is not allowed for proto file",
			file:          "proto_file_error1",
			expectedError: newOptionNotAllowedForIntMessage("tag", "proto_file"),
		},
		{
			testName:      "Test subdir is not allowed for directory",
			file:          "dir_error1",
			expectedError: newOptionNotAllowedForIntMessage("subdir", "directory"),
		},
		{
			testName:      "Test stdin is not allowed for directory",
			file:          "dir_error2",
			expectedError: `invalid directory path: "-"`,
		},
		{
			testName:      "Test invalid compression not allowed",
			file:          "tar_error1",
			expectedError: `unknown compression: "abcd" (valid values are "none,gzip,zstd")`,
		},
		{
			testName:      "Test depth not allowed for tarball",
			file:          "tar_error2",
			expectedError: newOptionNotAllowedForIntMessage("depth", "tarball"),
		},
		{
			testName:      "Test subdir is validated for tarball",
			file:          "tar_error3",
			expectedError: "/x/y: expected to be relative",
		},
		{
			testName:      "Test compression not allowed for zip archive",
			file:          "zip_error1",
			expectedError: newOptionNotAllowedForIntMessage("compression", "zip_archive"),
		},
		{
			testName:      "Test subdir is validated for zip archive",
			file:          "zip_error2",
			expectedError: "/m/n: expected to be relative",
		},
		{
			testName:      "Test compression is validated for JSON image",
			file:          "json_error1",
			expectedError: `unknown compression: "xyz" (valid values are "none,gzip,zstd")`,
		},
		{
			testName:      "Test include package files is not allowed for JSON image",
			file:          "json_error2",
			expectedError: newOptionNotAllowedForIntMessage("include_package_files", "json_image"),
		},
	}

	for _, test := range tests {
		test := test
		for _, fileExtension := range []string{".yaml", ".json"} {
			fileExtension := fileExtension
			t.Run(fmt.Sprintf("%s with extension %s", test.testName, fileExtension), func(t *testing.T) {
				t.Parallel()
				file := filepath.Join("testdata", "input", test.file+fileExtension)
				config, err := ReadConfigV2(
					ctx,
					nopLogger,
					provider,
					readBucket,
					bufgen.ReadConfigWithOverride(file),
				)
				require.Nil(t, config)
				require.ErrorContains(t, err, test.expectedError)
			})
		}
	}
}

func TestManagedConfig(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	nopLogger := zap.NewNop()
	provider := bufgen.NewConfigDataProvider(zap.NewNop())
	readBucket, err := storagemem.NewReadBucket(nil)
	require.NoError(t, err)
	tests := []struct {
		testName                string
		file                    string
		expectedDisableResults  map[FileOption]map[ImageFileIdentity]bool
		expectedOverrideResults map[FileOption]map[ImageFileIdentity]bufimagemodifyv2.Override
	}{
		{
			testName: "Test override",
			file:     filepath.Join("managed", "buf.gen"),
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
	}

	for _, test := range tests {
		test := test
		for _, fileExtension := range []string{".yaml" /* ,".json" */} {
			fileExtension := fileExtension
			t.Run(fmt.Sprintf("%s with extension %s", test.testName, fileExtension), func(t *testing.T) {
				t.Parallel()
				file := filepath.Join("testdata", test.file+fileExtension)
				config, err := ReadConfigV2(
					ctx,
					nopLogger,
					provider,
					readBucket,
					bufgen.ReadConfigWithOverride(file),
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

func newOptionNotAllowedForIntMessage(option string, input string) string {
	return fmt.Sprintf("option %s is not allowed for format %s", option, input)
}

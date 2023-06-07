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
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestIntputConfigSuccess(t *testing.T) {
	ctx := context.Background()
	nopLogger := zap.NewNop()
	refParser := buffetch.NewRefParser(nopLogger)
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
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
							"github.com/acme/weather0.git#branch=main,ref=fdafaewafe,depth=1,subdir=protos,recurse_submodules=false",
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
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
							"github.com/acme/weather1.git#tag=v123,depth=10,subdir=proto,recurse_submodules=true",
						),
						Types:        nil,
						ExcludePaths: nil,
						IncludePaths: nil,
					},
					{
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
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
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
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
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
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
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
							"a.proto#include_package_files=true",
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
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
							"b.proto",
						),
					},
					{
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
							"c.proto#include_package_files=false",
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
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
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
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
							"/c/d",
						),
					},
					{
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
							"/e/f/g",
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
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
							"a/b/c#format=tar,compression=gzip,strip_components=6,subdir=x/y",
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
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
							"a/b/c.tar.gz#format=tar,compression=gzip",
						),
					},
					{
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
							"a/b/c.tgz#format=tar,compression=gzip",
						),
					},
					{
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
							"a/b/c.tar.zst#format=tar,compression=zstd",
						),
					},
					{
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
							"a/b/c.tar#format=tar,compression=none",
						),
					},
					{
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
							"a/b/c#format=tar,compression=none",
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
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
							"a/b/c.zip#format=zip,strip_components=10,subdir=x/y",
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
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
							"a/b/c#format=zip",
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
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
							"a/b/c#format=json,compression=gzip",
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
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
							"a/b/c.json.gz#format=json,compression=gzip",
						),
					},
					{
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
							"a/b/c.json.zst#format=json,compression=zstd",
						),
					},
					{
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
							"a/b/c.json#format=json,compression=none",
						),
					},
					{
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
							"a/b/c#format=json,compression=none",
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
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
							"a/b/c#format=bin,compression=gzip",
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
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
							"a/b/c.bin.gz#format=bin,compression=gzip",
						),
					},
					{
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
							"a/b/c.bin.zst#format=bin,compression=zstd",
						),
					},
					{
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
							"a/b/c.bin#format=bin,compression=none",
						),
					},
					{
						InputRef: mustParseRefFromValue(
							t,
							ctx,
							refParser,
							"a/b/c#format=bin,compression=none",
						),
					},
				},
			},
		},
	}

	for _, test := range tests {
		for _, fileExtension := range []string{"yaml" /* ,"yml", "json" */} { // TODO: enable all
			t.Run(test.testName, func(t *testing.T) {
				test := test
				file := filepath.Join("testdata", "input", test.file+"."+fileExtension)
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
			expectedError: "each input can only have one format", // TODO: each input can only have one format, already of type: xyz
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
			testName:      "Test tag not allowed for proto file",
			file:          "proto_file_error1",
			expectedError: newOptionNotAllowedForIntMessage("tag", "proto_file"),
		},
		{
			testName:      "Test subdir not allowed for directory",
			file:          "dir_error1",
			expectedError: newOptionNotAllowedForIntMessage("subdir", "directory"),
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
		for _, fileExtension := range []string{"yaml" /* ,"yml", "json" */} { // TODO: enable all
			t.Run(test.testName, func(t *testing.T) {
				test := test
				file := filepath.Join("testdata", "input", test.file+"."+fileExtension)

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

func mustParseRefFromValue(t *testing.T, ctx context.Context, refParser buffetch.RefParser, value string) buffetch.Ref {
	ref, err := refParser.GetRef(ctx, value)
	require.NoError(t, err, "invalid test case: error trying to parse value %q to a Ref: %v", value, err)
	return ref
}

func newOptionNotAllowedForIntMessage(option string, input string) string {
	return fmt.Sprintf("option %s is not allowed for format %s", option, input)
}

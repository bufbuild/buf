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

package buffetch

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/bufbuild/buf/private/buf/buffetch/internal"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/stretchr/testify/require"
)

func TestGetGitRef(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	refBuilder := newRefBuilder()

	tests := []struct {
		name          string
		path          string
		format        string
		options       []GetGitRefOption
		expectedError error
		expectedRef   Ref
	}{
		{
			name:   "Test all fields but tag",
			path:   "local.git",
			format: "git_repo",
			options: []GetGitRefOption{
				WithGetGitRefBranch("b1"),
				WithGetGitRefDepth(2),
				WithGetGitRefRecurseSubmodules(),
				WithGetGitRefSubDir("c/d"),
				WithGetGitRefRef("refname"),
			},
			expectedRef: newSourceRef(
				internal.NewDirectParsedGitRef(
					"git_repo",
					"local.git",
					internal.GitSchemeLocal,
					git.NewRefNameWithBranch("refname", "b1"),
					true,
					2,
					"c/d",
				),
			),
		},
		{
			name:   "Test all fields but branch and ref",
			path:   "local.git",
			format: "git_repo",
			options: []GetGitRefOption{
				WithGetGitRefTag("tag1"),
				WithGetGitRefDepth(2),
				WithGetGitRefRecurseSubmodules(),
				WithGetGitRefSubDir("c/d"),
			},
			expectedRef: newSourceRef(
				internal.NewDirectParsedGitRef(
					"git_repo",
					"local.git",
					internal.GitSchemeLocal,
					git.NewTagName("tag1"),
					true,
					2,
					"c/d",
				),
			),
		},
		{
			name:   "Test default depth for when ref is specified",
			path:   "local.git",
			format: "git_repo",
			options: []GetGitRefOption{
				WithGetGitRefTag("tag1"),
				WithGetGitRefRecurseSubmodules(),
				WithGetGitRefSubDir("c/d"),
			},
			expectedRef: newSourceRef(
				internal.NewDirectParsedGitRef(
					"git_repo",
					"local.git",
					internal.GitSchemeLocal,
					git.NewTagName("tag1"),
					true,
					1,
					"c/d",
				),
			),
		},
		{
			name:   "Test default depth is 50 when ref is specified",
			path:   "local.git",
			format: "git_repo",
			options: []GetGitRefOption{
				WithGetGitRefRef("refname"),
				WithGetGitRefRecurseSubmodules(),
				WithGetGitRefSubDir("c/d"),
			},
			expectedRef: newSourceRef(
				internal.NewDirectParsedGitRef(
					"git_repo",
					"local.git",
					internal.GitSchemeLocal,
					git.NewRefName("refname"),
					true,
					50,
					"c/d",
				),
			),
		},
		{
			name:   "Test default depth is 1 when ref is not specified",
			path:   "local.git",
			format: "git_repo",
			options: []GetGitRefOption{
				WithGetGitRefBranch("branch1"),
			},
			expectedRef: newSourceRef(
				internal.NewDirectParsedGitRef(
					"git_repo",
					"local.git",
					internal.GitSchemeLocal,
					git.NewBranchName("branch1"),
					false,
					1,
					"",
				),
			),
		},
		{
			name:   "Test stdin is not allowed for git",
			path:   "-",
			format: "git_repo",
			options: []GetGitRefOption{
				WithGetGitRefBranch("branch1"),
			},
			expectedError: errors.New(`invalid git_repo path: "-"`),
		},
		{
			name:   "Test dev null is not allowed for git",
			path:   app.DevNullFilePath,
			format: "git_repo",
			options: []GetGitRefOption{
				WithGetGitRefBranch("branch1"),
			},
			expectedError: fmt.Errorf("%s is not allowed for git_repo", app.DevNullFilePath),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ref, err := refBuilder.GetGitRef(
				ctx,
				test.format,
				test.path,
				test.options...,
			)
			if test.expectedError != nil {
				require.Nil(t, ref)
				require.Equal(t, test.expectedError, err)
			} else {
				require.Nil(t, err)
				require.Equal(t, test.expectedRef, ref)
			}
		})
	}
}

func TestGetModuleRef(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	refBuilder := newRefBuilder()

	tests := []struct {
		name          string
		path          string
		format        string
		expectedError error
		expectedRef   Ref
	}{
		{
			name:   "Test module",
			path:   "buf.build/acme/weather",
			format: "module",
			expectedRef: newModuleRef(
				internal.NewDirectParsedModuleRef(
					"module",
					testNewModuleReference(
						t,
						"buf.build",
						"acme",
						"weather",
						"main",
					),
				),
			),
		},
		{
			name:          "Test invalid module name",
			path:          "abcde",
			format:        "module",
			expectedError: errors.New(`invalid module path: "abcde"`),
		},
		{
			name:          "Test stdin is not allowed for module",
			path:          "-",
			format:        "module",
			expectedError: errors.New(`invalid module path: "-"`),
		},
		{
			name:          "Test dev null is not allowed for module",
			path:          app.DevNullFilePath,
			format:        "module",
			expectedError: fmt.Errorf("%s is not allowed for module", app.DevNullFilePath),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ref, err := refBuilder.GetModuleRef(
				ctx,
				test.format,
				test.path,
			)
			if test.expectedError != nil {
				require.Nil(t, ref)
				require.Equal(t, test.expectedError, err)
			} else {
				require.Nil(t, err)
				require.Equal(t, test.expectedRef, ref)
			}
		})
	}
}

func TestGetProtoFileRef(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	refBuilder := newRefBuilder()

	tests := []struct {
		name          string
		path          string
		format        string
		options       []GetProtoFileRefOption
		expectedError error
		expectedRef   Ref
	}{
		{
			name:   "Test relative path to proto file",
			path:   "a/b/c.proto",
			format: "proto_file",
			options: []GetProtoFileRefOption{
				WithGetProtoFileRefIncludePackageFiles(),
			},
			expectedRef: newProtoFileRef(
				internal.NewDirectParsedProtoFileRef(
					"proto_file",
					"a/b/c.proto",
					true,
				),
			),
		},
		{
			name:   "Test absolute path to proto file",
			path:   "/x/y/z.proto",
			format: "proto_file",
			expectedRef: newProtoFileRef(
				internal.NewDirectParsedProtoFileRef(
					"proto_file",
					"/x/y/z.proto",
					false,
				),
			),
		},
		{
			name:          "Test stdin not allowed proto file",
			path:          "-",
			format:        "proto_file",
			expectedError: errors.New(`invalid proto_file path: "-" (protofiles cannot be read or written to or from stdio)`),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ref, err := refBuilder.GetProtoFileRef(
				ctx,
				test.format,
				test.path,
				test.options...,
			)
			if test.expectedError != nil {
				require.Nil(t, ref)
				require.Equal(t, test.expectedError, err)
			} else {
				require.Nil(t, err)
				require.Equal(t, test.expectedRef, ref)
			}
		})
	}
}

func TestGetDirRef(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	refBuilder := newRefBuilder()

	tests := []struct {
		name          string
		path          string
		format        string
		expectedError error
		expectedRef   Ref
	}{
		{
			name:   "Test relative path is allowed",
			path:   ".",
			format: "directory",
			expectedRef: newSourceRef(
				internal.NewDirectParsedDirRef(
					"directory",
					".",
				),
			),
		},
		{
			name:   "Test absolute path is allowed",
			path:   "/x/y",
			format: "directory",
			expectedRef: newSourceRef(
				internal.NewDirectParsedDirRef(
					"directory",
					"/x/y",
				),
			),
		},
		{
			name:          "Test stdin is not allowed for directory",
			path:          "-",
			format:        "directory",
			expectedError: errors.New(`invalid directory path: "-"`),
		},
		{
			name:          "Test dev null is not allowed for directory",
			path:          app.DevNullFilePath,
			format:        "directory",
			expectedError: fmt.Errorf("%s is not allowed for directory", app.DevNullFilePath),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ref, err := refBuilder.GetDirRef(
				ctx,
				test.format,
				test.path,
			)
			if test.expectedError != nil {
				require.Nil(t, ref)
				require.Equal(t, test.expectedError, err)
			} else {
				require.Nil(t, err)
				require.Equal(t, test.expectedRef, ref)
			}
		})
	}
}

func TestGetTarballRef(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	refBuilder := newRefBuilder()

	tests := []struct {
		name          string
		path          string
		format        string
		options       []GetTarballRefOption
		expectedError error
		expectedRef   Ref
	}{
		{
			name:   "Test tarball local path",
			path:   "a/b/c.tar",
			format: "tarball",
			options: []GetTarballRefOption{
				WithGetTarballRefCompression("gzip"),
				WithGetTarballRefStripComponents(5),
				WithGetTarballRefSubDir("m/n"),
			},
			expectedRef: newSourceRef(
				internal.NewDirectParsedArchiveRef(
					"tarball",
					"a/b/c.tar",
					internal.FileSchemeLocal,
					internal.ArchiveTypeTar,
					internal.CompressionTypeGzip,
					5,
					"m/n",
				),
			),
		},
		{
			name:   "Test tarball remote path",
			path:   "https://github.com/googleapis/googleapis/archive/master.tar",
			format: "tarball",
			expectedRef: newSourceRef(
				internal.NewDirectParsedArchiveRef(
					"tarball",
					"github.com/googleapis/googleapis/archive/master.tar",
					internal.FileSchemeHTTPS,
					internal.ArchiveTypeTar,
					internal.CompressionTypeNone,
					0,
					"",
				),
			),
		},
		{
			name:   "Test tarball auto detect .tgz",
			path:   "a.tgz",
			format: "tarball",
			expectedRef: newSourceRef(
				internal.NewDirectParsedArchiveRef(
					"tarball",
					"a.tgz",
					internal.FileSchemeLocal,
					internal.ArchiveTypeTar,
					internal.CompressionTypeGzip,
					0,
					"",
				),
			),
		},
		{
			name:   "Test tarball auto detect .gz",
			path:   "a.gz",
			format: "tarball",
			expectedRef: newSourceRef(
				internal.NewDirectParsedArchiveRef(
					"tarball",
					"a.gz",
					internal.FileSchemeLocal,
					internal.ArchiveTypeTar,
					internal.CompressionTypeGzip,
					0,
					"",
				),
			),
		},
		{
			name:   "Test tarball auto detect .zst",
			path:   "a.zst",
			format: "tarball",
			expectedRef: newSourceRef(
				internal.NewDirectParsedArchiveRef(
					"tarball",
					"a.zst",
					internal.FileSchemeLocal,
					internal.ArchiveTypeTar,
					internal.CompressionTypeZstd,
					0,
					"",
				),
			),
		},
		{
			name:   "Test tarball option overwrites assumed compression",
			path:   "a.zst",
			format: "tarball",
			options: []GetTarballRefOption{
				WithGetTarballRefCompression("none"),
			},
			expectedRef: newSourceRef(
				internal.NewDirectParsedArchiveRef(
					"tarball",
					"a.zst",
					internal.FileSchemeLocal,
					internal.ArchiveTypeTar,
					internal.CompressionTypeNone,
					0,
					"",
				),
			),
		},
		{
			name:   "Test tarball from stdin",
			path:   "-",
			format: "tarball",
			expectedRef: newSourceRef(
				internal.NewDirectParsedArchiveRef(
					"tarball",
					"",
					internal.FileSchemeStdio,
					internal.ArchiveTypeTar,
					internal.CompressionTypeNone,
					0,
					"",
				),
			),
		},
		{
			name:          "Test dev null not allowed for tarball",
			path:          app.DevNullFilePath,
			format:        "tarball",
			expectedError: fmt.Errorf("%s is not allowed for tarball", app.DevNullFilePath),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ref, err := refBuilder.GetTarballRef(
				ctx,
				test.format,
				test.path,
				test.options...,
			)
			if test.expectedError != nil {
				require.Nil(t, ref)
				require.Equal(t, test.expectedError, err)
			} else {
				require.Nil(t, err)
				require.Equal(t, test.expectedRef, ref)
			}
		})
	}
}

func TestGetZipArchiveRef(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	refBuilder := newRefBuilder()

	tests := []struct {
		name          string
		path          string
		format        string
		options       []GetZipArchiveRefOption
		expectedError error
		expectedRef   Ref
	}{
		{
			name:   "Test local zip with all options",
			path:   "a/b/c.zip",
			format: "zip_archive",
			options: []GetZipArchiveRefOption{
				WithGetZipArchiveRefStripComponents(6),
				WithGetZipArchiveRefSubDir("m/n"),
			},
			expectedRef: newSourceRef(
				internal.NewDirectParsedArchiveRef(
					"zip_archive",
					"a/b/c.zip",
					internal.FileSchemeLocal,
					internal.ArchiveTypeZip,
					internal.CompressionTypeNone,
					6,
					"m/n",
				),
			),
		},
		{
			name:   "Test zip remote path",
			path:   "https://github.com/googleapis/googleapis/archive/master.zip",
			format: "zip_archive",
			options: []GetZipArchiveRefOption{
				WithGetZipArchiveRefStripComponents(6),
				WithGetZipArchiveRefSubDir("m/n"),
			},
			expectedRef: newSourceRef(
				internal.NewDirectParsedArchiveRef(
					"zip_archive",
					"github.com/googleapis/googleapis/archive/master.zip",
					internal.FileSchemeHTTPS,
					internal.ArchiveTypeZip,
					internal.CompressionTypeNone,
					6,
					"m/n",
				),
			),
		},
		{
			name:   "Test zip_archive from stdin",
			path:   "-",
			format: "zip_archive",
			expectedRef: newSourceRef(
				internal.NewDirectParsedArchiveRef(
					"zip_archive",
					"",
					internal.FileSchemeStdio,
					internal.ArchiveTypeZip,
					internal.CompressionTypeNone,
					0,
					"",
				),
			),
		},
		{
			name:          "Test dev null not allowed for zip_archive",
			path:          app.DevNullFilePath,
			format:        "zip_archive",
			expectedError: fmt.Errorf("%s is not allowed for zip_archive", app.DevNullFilePath),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ref, err := refBuilder.GetZipArchiveRef(
				ctx,
				test.format,
				test.path,
				test.options...,
			)
			if test.expectedError != nil {
				require.Nil(t, ref)
				require.Equal(t, test.expectedError, err)
			} else {
				require.Nil(t, err)
				require.Equal(t, test.expectedRef, ref)
			}
		})
	}
}

func TestGetBinaryImageRef(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	refBuilder := newRefBuilder()

	tests := []struct {
		name          string
		path          string
		format        string
		options       []GetImageRefOption
		expectedError error
		expectedRef   Ref
	}{
		{
			name:   "Test local binary image",
			path:   "a/b/c",
			format: "binary_image",
			options: []GetImageRefOption{
				WithGetImageRefOption("gzip"),
			},
			expectedRef: newImageRef(
				internal.NewDirectParsedSingleRef(
					"binary_image",
					"a/b/c",
					internal.FileSchemeLocal,
					internal.CompressionTypeGzip,
				),
				ImageEncodingBin,
			),
		},
		{
			name:   "Test remote binary image",
			path:   "https://github.com/googleapis/googleapis/archive/master.bin.gz",
			format: "binary_image",
			expectedRef: newImageRef(
				internal.NewDirectParsedSingleRef(
					"binary_image",
					"github.com/googleapis/googleapis/archive/master.bin.gz",
					internal.FileSchemeHTTPS,
					internal.CompressionTypeGzip,
				),
				ImageEncodingBin,
			),
		},
		{
			name:   "Test binary image from stdin",
			path:   "-",
			format: "binary_image",
			options: []GetImageRefOption{
				WithGetImageRefOption("zstd"),
			},
			expectedRef: newImageRef(
				internal.NewDirectParsedSingleRef(
					"binary_image",
					"",
					internal.FileSchemeStdio,
					internal.CompressionTypeZstd,
				),
				ImageEncodingBin,
			),
		},
		{
			name:   "Test dev null is allowed for zip_archive",
			path:   app.DevNullFilePath,
			format: "binary_image",
			expectedRef: newImageRef(
				internal.NewDirectParsedSingleRef(
					"binary_image",
					"",
					internal.FileSchemeNull,
					internal.CompressionTypeNone,
				),
				ImageEncodingBin,
			),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ref, err := refBuilder.GetBinaryImageRef(
				ctx,
				test.format,
				test.path,
				test.options...,
			)
			if test.expectedError != nil {
				require.Nil(t, ref)
				require.Equal(t, test.expectedError, err)
			} else {
				require.Nil(t, err)
				require.Equal(t, test.expectedRef, ref)
			}
		})
	}
}

func TestGetJSONImageRef(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	refBuilder := newRefBuilder()

	tests := []struct {
		name          string
		path          string
		format        string
		options       []GetImageRefOption
		expectedError error
		expectedRef   Ref
	}{
		{
			name:   "Test local json image",
			path:   "a/b/c",
			format: "json_image",
			options: []GetImageRefOption{
				WithGetImageRefOption("gzip"),
			},
			expectedRef: newImageRef(
				internal.NewDirectParsedSingleRef(
					"json_image",
					"a/b/c",
					internal.FileSchemeLocal,
					internal.CompressionTypeGzip,
				),
				ImageEncodingJSON,
			),
		},
		{
			name:   "Test remote json image",
			path:   "https://github.com/googleapis/googleapis/archive/master.json.gz",
			format: "json_image",
			expectedRef: newImageRef(
				internal.NewDirectParsedSingleRef(
					"json_image",
					"github.com/googleapis/googleapis/archive/master.json.gz",
					internal.FileSchemeHTTPS,
					internal.CompressionTypeGzip,
				),
				ImageEncodingJSON,
			),
		},
		{
			name:   "Test json image from stdin",
			path:   "-",
			format: "json_image",
			options: []GetImageRefOption{
				WithGetImageRefOption("zstd"),
			},
			expectedRef: newImageRef(
				internal.NewDirectParsedSingleRef(
					"json_image",
					"",
					internal.FileSchemeStdio,
					internal.CompressionTypeZstd,
				),
				ImageEncodingJSON,
			),
		},
		{
			name:          "Test dev null is not allowed for zip_archive",
			path:          app.DevNullFilePath,
			format:        "json_image",
			expectedError: fmt.Errorf("%s is not allowed for json_image", app.DevNullFilePath),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ref, err := refBuilder.GetJSONImageRef(
				ctx,
				test.format,
				test.path,
				test.options...,
			)
			if test.expectedError != nil {
				require.Nil(t, ref)
				require.Equal(t, test.expectedError, err)
			} else {
				require.Nil(t, err)
				require.Equal(t, test.expectedRef, ref)
			}
		})
	}
}

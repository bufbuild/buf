// Copyright 2020 Buf Technologies, Inc.
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

package fetch

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/git"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

const (
	testFormatBin    = "bin"
	testFormatJSON   = "json"
	testFormatBingz  = "bingz"
	testFormatJSONGZ = "jsongz"
	testFormatTar    = "tar"
	testFormatTargz  = "targz"
	testFormatZip    = "zip"
	testFormatGit    = "git"
	testFormatDir    = "dir"
)

var (
	testGetParsedRefOptions = []GetParsedRefOption{
		WithAllowedFormats(
			testFormatBin,
			testFormatJSON,
			testFormatBingz,
			testFormatJSONGZ,
			testFormatTar,
			testFormatTargz,
			testFormatZip,
			testFormatGit,
			testFormatDir,
		),
	}
)

func TestGetParsedRefSuccess(t *testing.T) {
	testGetParsedRefSuccess(
		t,
		buildDirRef(
			testFormatDir,
			"path/to/dir",
		),
		"path/to/dir",
	)
	testGetParsedRefSuccess(
		t,
		buildDirRef(
			testFormatDir,
			".",
		),
		".",
	)
	testGetParsedRefSuccess(
		t,
		buildDirRef(
			testFormatDir,
			"/",
		),
		"/",
	)
	testGetParsedRefSuccess(
		t,
		buildDirRef(
			testFormatDir,
			".",
		),
		"foo/..",
	)
	testGetParsedRefSuccess(
		t,
		buildDirRef(
			testFormatDir,
			"../foo",
		),
		"../foo",
	)
	testGetParsedRefSuccess(
		t,
		buildDirRef(
			testFormatDir,
			"/foo",
		),
		"/foo/bar/..",
	)
	testGetParsedRefSuccess(
		t,
		buildArchiveRef(
			testFormatTar,
			"path/to/file.tar",
			FileSchemeLocal,
			ArchiveTypeTar,
			CompressionTypeNone,
			0,
		),
		"path/to/file.tar",
	)
	testGetParsedRefSuccess(
		t,
		buildArchiveRef(
			testFormatTar,
			"/path/to/file.tar",
			FileSchemeLocal,
			ArchiveTypeTar,
			CompressionTypeNone,
			0,
		),
		"file:///path/to/file.tar",
	)
	testGetParsedRefSuccess(
		t,
		buildArchiveRef(
			testFormatTar,
			"path/to/file.tar",
			FileSchemeLocal,
			ArchiveTypeTar,
			CompressionTypeNone,
			1,
		),
		"path/to/file.tar#strip_components=1",
	)
	testGetParsedRefSuccess(
		t,
		buildArchiveRef(
			testFormatTar,
			"path/to/file.tar.gz",
			FileSchemeLocal,
			ArchiveTypeTar,
			CompressionTypeGzip,
			0,
		),
		"path/to/file.tar.gz",
	)
	testGetParsedRefSuccess(
		t,
		buildArchiveRef(
			testFormatTar,
			"path/to/file.tar.gz",
			FileSchemeLocal,
			ArchiveTypeTar,
			CompressionTypeGzip,
			1,
		),
		"path/to/file.tar.gz#strip_components=1",
	)
	testGetParsedRefSuccess(
		t,
		buildArchiveRef(
			testFormatTar,
			"path/to/file.tgz",
			FileSchemeLocal,
			ArchiveTypeTar,
			CompressionTypeGzip,
			0,
		),
		"path/to/file.tgz",
	)
	testGetParsedRefSuccess(
		t,
		buildArchiveRef(
			testFormatTar,
			"path/to/file.tgz",
			FileSchemeLocal,
			ArchiveTypeTar,
			CompressionTypeGzip,
			1,
		),
		"path/to/file.tgz#strip_components=1",
	)
	testGetParsedRefSuccess(
		t,
		buildArchiveRef(
			testFormatTar,
			"path/to/file.tar",
			FileSchemeHTTP,
			ArchiveTypeTar,
			CompressionTypeNone,
			0,
		),
		"http://path/to/file.tar",
	)
	testGetParsedRefSuccess(
		t,
		buildArchiveRef(
			testFormatTar,
			"path/to/file.tar",
			FileSchemeHTTPS,
			ArchiveTypeTar,
			CompressionTypeNone,
			0,
		),
		"https://path/to/file.tar",
	)
	testGetParsedRefSuccess(
		t,
		buildArchiveRef(
			testFormatZip,
			"path/to/file.zip",
			FileSchemeLocal,
			ArchiveTypeZip,
			CompressionTypeNone,
			0,
		),
		"path/to/file.zip",
	)
	testGetParsedRefSuccess(
		t,
		buildArchiveRef(
			testFormatZip,
			"/path/to/file.zip",
			FileSchemeLocal,
			ArchiveTypeZip,
			CompressionTypeNone,
			0,
		),
		"file:///path/to/file.zip",
	)
	testGetParsedRefSuccess(
		t,
		buildArchiveRef(
			testFormatZip,
			"path/to/file.zip",
			FileSchemeLocal,
			ArchiveTypeZip,
			CompressionTypeNone,
			1,
		),
		"path/to/file.zip#strip_components=1",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"path/to/dir.git",
			GitSchemeLocal,
			nil,
			false,
			1,
		),
		"path/to/dir.git",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"path/to/dir.git",
			GitSchemeLocal,
			nil,
			false,
			40,
		),
		"path/to/dir.git#depth=40",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"path/to/dir.git",
			GitSchemeLocal,
			git.NewBranchName("master"),
			false,
			1,
		),
		"path/to/dir.git#branch=master",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"/path/to/dir.git",
			GitSchemeLocal,
			git.NewBranchName("master"),
			false,
			1,
		),
		"file:///path/to/dir.git#branch=master",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"path/to/dir.git",
			GitSchemeLocal,
			git.NewTagName("v1.0.0"),
			false,
			1,
		),
		"path/to/dir.git#tag=v1.0.0",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"hello.com/path/to/dir.git",
			GitSchemeHTTP,
			git.NewBranchName("master"),
			false,
			1,
		),
		"http://hello.com/path/to/dir.git#branch=master",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"hello.com/path/to/dir.git",
			GitSchemeHTTPS,
			git.NewBranchName("master"),
			false,
			1,
		),
		"https://hello.com/path/to/dir.git#branch=master",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"user@hello.com:path/to/dir.git",
			GitSchemeSSH,
			git.NewBranchName("master"),
			false,
			1,
		),
		"ssh://user@hello.com:path/to/dir.git#branch=master",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"user@hello.com:path/to/dir.git",
			GitSchemeSSH,
			git.NewRefName("refs/remotes/origin/HEAD"),
			false,
			50,
		),
		"ssh://user@hello.com:path/to/dir.git#ref=refs/remotes/origin/HEAD",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"user@hello.com:path/to/dir.git",
			GitSchemeSSH,
			git.NewRefNameWithBranch("refs/remotes/origin/HEAD", "master"),
			false,
			50,
		),
		"ssh://user@hello.com:path/to/dir.git#ref=refs/remotes/origin/HEAD,branch=master",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"user@hello.com:path/to/dir.git",
			GitSchemeSSH,
			git.NewRefName("refs/remotes/origin/HEAD"),
			false,
			10,
		),
		"ssh://user@hello.com:path/to/dir.git#ref=refs/remotes/origin/HEAD,depth=10",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"user@hello.com:path/to/dir.git",
			GitSchemeSSH,
			git.NewRefNameWithBranch("refs/remotes/origin/HEAD", "master"),
			false,
			10,
		),
		"ssh://user@hello.com:path/to/dir.git#ref=refs/remotes/origin/HEAD,branch=master,depth=10",
	)
	testGetParsedRefSuccess(
		t,
		buildSingleRef(
			testFormatBin,
			"path/to/file.bin",
			FileSchemeLocal,
			CompressionTypeNone,
		),
		"path/to/file.bin",
	)
	testGetParsedRefSuccess(
		t,
		buildSingleRef(
			testFormatBin,
			"path/to/file.bin.gz",
			FileSchemeLocal,
			CompressionTypeGzip,
		),
		"path/to/file.bin.gz",
	)
	testGetParsedRefSuccess(
		t,
		buildSingleRef(
			testFormatJSON,
			"path/to/file.json",
			FileSchemeLocal,
			CompressionTypeNone,
		),
		"path/to/file.json",
	)
	testGetParsedRefSuccess(
		t,
		buildSingleRef(
			testFormatJSON,
			"path/to/file.json.gz",
			FileSchemeLocal,
			CompressionTypeGzip,
		),
		"path/to/file.json.gz",
	)
	testGetParsedRefSuccess(
		t,
		buildSingleRef(
			testFormatJSON,
			"path/to/file.json.gz",
			FileSchemeLocal,
			CompressionTypeNone,
		),
		"path/to/file.json.gz#compression=none",
	)
	testGetParsedRefSuccess(
		t,
		buildSingleRef(
			testFormatJSON,
			"path/to/file.json.gz",
			FileSchemeLocal,
			CompressionTypeGzip,
		),
		"path/to/file.json.gz#compression=gzip",
	)
	testGetParsedRefSuccess(
		t,
		buildSingleRef(
			testFormatBin,
			"",
			FileSchemeStdio,
			CompressionTypeNone,
		),
		"-",
	)
	testGetParsedRefSuccess(
		t,
		buildSingleRef(
			testFormatJSON,
			"",
			FileSchemeStdio,
			CompressionTypeNone,
		),
		"-#format=json",
	)
	testGetParsedRefSuccess(
		t,
		buildSingleRef(
			testFormatBin,
			"",
			FileSchemeNull,
			CompressionTypeNone,
		),
		app.DevNullFilePath,
	)
	// TODO: this needs to be moved to a unix-only test
	testGetParsedRefSuccess(
		t,
		buildSingleRef(
			testFormatBin,
			"",
			FileSchemeStdin,
			CompressionTypeNone,
		),
		app.DevStdinFilePath,
	)
	// TODO: this needs to be moved to a unix-only test
	testGetParsedRefSuccess(
		t,
		buildSingleRef(
			testFormatBin,
			"",
			FileSchemeStdout,
			CompressionTypeNone,
		),
		app.DevStdoutFilePath,
	)
	testGetParsedRefSuccess(
		t,
		buildSingleRef(
			testFormatBin,
			"path/to/dir",
			FileSchemeLocal,
			CompressionTypeNone,
		),
		"path/to/dir#format=bin",
	)
	testGetParsedRefSuccess(
		t,
		buildSingleRef(
			testFormatBin,
			"path/to/dir",
			FileSchemeLocal,
			CompressionTypeNone,
		),
		"path/to/dir#format=bin,compression=none",
	)
	testGetParsedRefSuccess(
		t,
		buildSingleRef(
			testFormatBin,
			"path/to/dir",
			FileSchemeLocal,
			CompressionTypeGzip,
		),
		"path/to/dir#format=bin,compression=gzip",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"/path/to/dir",
			GitSchemeLocal,
			git.NewBranchName("master"),
			false,
			1,
		),
		"/path/to/dir#branch=master,format=git",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"/path/to/dir",
			GitSchemeLocal,
			git.NewBranchName("master/foo"),
			false,
			1,
		),
		"/path/to/dir#format=git,branch=master/foo",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"path/to/dir",
			GitSchemeLocal,
			git.NewTagName("master/foo"),
			false,
			1,
		),
		"path/to/dir#tag=master/foo,format=git",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"path/to/dir",
			GitSchemeLocal,
			git.NewTagName("master/foo"),
			false,
			1,
		),
		"path/to/dir#format=git,tag=master/foo",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"path/to/dir",
			GitSchemeLocal,
			git.NewTagName("master/foo"),
			true,
			1,
		),
		"path/to/dir#format=git,tag=master/foo,recurse_submodules=true",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"path/to/dir",
			GitSchemeLocal,
			git.NewTagName("master/foo"),
			false,
			1,
		),
		"path/to/dir#format=git,tag=master/foo,recurse_submodules=false",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"path/to/dir",
			GitSchemeLocal,
			git.NewRefName("refs/remotes/origin/HEAD"),
			false,
			50,
		),
		"path/to/dir#format=git,ref=refs/remotes/origin/HEAD",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"path/to/dir",
			GitSchemeLocal,
			git.NewRefName("refs/remotes/origin/HEAD"),
			false,
			10,
		),
		"path/to/dir#format=git,ref=refs/remotes/origin/HEAD,depth=10",
	)
	testGetParsedRefSuccess(
		t,
		buildArchiveRef(
			testFormatTargz,
			"path/to/file",
			FileSchemeLocal,
			ArchiveTypeTar,
			CompressionTypeGzip,
			1,
		),
		"path/to/file#format=targz,strip_components=1",
	)
	testGetParsedRefSuccess(
		t,
		buildArchiveRef(
			testFormatTar,
			"path/to/file",
			FileSchemeLocal,
			ArchiveTypeTar,
			CompressionTypeNone,
			1,
		),
		"path/to/file#format=tar,strip_components=1",
	)
	testGetParsedRefSuccess(
		t,
		buildArchiveRef(
			testFormatTar,
			"path/to/file",
			FileSchemeLocal,
			ArchiveTypeTar,
			CompressionTypeNone,
			1,
		),
		"path/to/file#format=tar,strip_components=1,compression=none",
	)
	testGetParsedRefSuccess(
		t,
		buildArchiveRef(
			testFormatTar,
			"path/to/file",
			FileSchemeLocal,
			ArchiveTypeTar,
			CompressionTypeGzip,
			1,
		),
		"path/to/file#format=tar,strip_components=1,compression=gzip",
	)
	testGetParsedRefSuccess(
		t,
		buildArchiveRef(
			testFormatZip,
			"path/to/file",
			FileSchemeLocal,
			ArchiveTypeZip,
			CompressionTypeNone,
			1,
		),
		"path/to/file#format=zip,strip_components=1",
	)
	testGetParsedRefSuccess(
		t,
		buildArchiveRef(
			testFormatTar,
			"path/to/file.tar.zst",
			FileSchemeLocal,
			ArchiveTypeTar,
			CompressionTypeZstd,
			0,
		),
		"path/to/file.tar.zst",
	)
	testGetParsedRefSuccess(
		t,
		buildArchiveRef(
			testFormatTar,
			"path/to/file.tar.zst",
			FileSchemeLocal,
			ArchiveTypeTar,
			CompressionTypeZstd,
			1,
		),
		"path/to/file.tar.zst#strip_components=1",
	)
	testGetParsedRefSuccess(
		t,
		buildArchiveRef(
			testFormatTar,
			"path/to/file",
			FileSchemeLocal,
			ArchiveTypeTar,
			CompressionTypeZstd,
			1,
		),
		"path/to/file#format=tar,strip_components=1,compression=zstd",
	)
	testGetParsedRefSuccess(
		t,
		buildSingleRef(
			testFormatBin,
			"path/to/file",
			FileSchemeLocal,
			CompressionTypeZstd,
		),
		"path/to/file#format=bin,compression=zstd",
	)
	testGetParsedRefSuccess(
		t,
		buildSingleRef(
			testFormatBin,
			"path/to/file.bin.zst",
			FileSchemeLocal,
			CompressionTypeZstd,
		),
		"path/to/file.bin.zst",
	)
}

func TestGetParsedRefError(t *testing.T) {
	testGetParsedRefError(
		t,
		newValueEmptyError(),
		"",
	)
	testGetParsedRefError(
		t,
		newValueMultipleHashtagsError("foo#format=git#branch=master"),
		"foo#format=git#branch=master",
	)
	testGetParsedRefError(
		t,
		newValueStartsWithHashtagError("#path/to/dir"),
		"#path/to/dir",
	)
	testGetParsedRefError(
		t,
		newValueEndsWithHashtagError("path/to/dir#"),
		"path/to/dir#",
	)
	testGetParsedRefError(
		t,
		newInvalidDirPathError("-"),
		"-#format=dir",
	)
	testGetParsedRefError(
		t,
		newInvalidGitPathError("-"),
		"-#format=git,branch=master",
	)
	testGetParsedRefError(
		t,
		newCannotSpecifyGitBranchAndTagError(),
		"path/to/foo#format=git,branch=foo,tag=bar",
	)
	testGetParsedRefError(
		t,
		newCannotSpecifyGitBranchAndTagError(),
		"path/to/foo#format=git,branch=foo,tag=bar,ref=baz",
	)
	testGetParsedRefError(
		t,
		newCannotSpecifyTagWithRefError(),
		"path/to/foo#format=git,tag=foo,ref=bar",
	)
	testGetParsedRefError(
		t,
		newDepthParseError("bar"),
		"path/to/foo#format=git,depth=bar",
	)
	testGetParsedRefError(
		t,
		newDepthZeroError(),
		"path/to/foo#format=git,ref=foor,depth=0",
	)
	testGetParsedRefError(
		t,
		newOptionsDuplicateKeyError("branch"),
		"path/to/foo#format=git,branch=foo,branch=bar",
	)
	testGetParsedRefError(
		t,
		newPathUnknownGzError("path/to/foo.gz"),
		"path/to/foo.gz",
	)
	testGetParsedRefError(
		t,
		newPathUnknownGzError("path/to/foo.bar.gz"),
		"path/to/foo.bar.gz",
	)
	testGetParsedRefError(
		t,
		newOptionsInvalidError("bar"),
		"path/to/foo#bar",
	)
	testGetParsedRefError(
		t,
		newOptionsInvalidError("bar="),
		"path/to/foo#bar=",
	)
	testGetParsedRefError(
		t,
		newOptionsInvalidError("format=bin,bar="),
		"path/to/foo#format=bin,bar=",
	)
	testGetParsedRefError(
		t,
		newOptionsInvalidError("format=bin,=bar"),
		"path/to/foo#format=bin,=bar",
	)
	testGetParsedRefError(
		t,
		newFormatOverrideNotAllowedForDevNullError(app.DevNullFilePath),
		fmt.Sprintf("%s#format=bin", app.DevNullFilePath),
	)
	testGetParsedRefError(
		t,
		newFormatUnknownError("bar"),
		"path/to/foo#format=bar",
	)
	testGetParsedRefError(
		t,
		newOptionsCouldNotParseStripComponentsError("foo"),
		"path/to/foo.tar.gz#strip_components=foo",
	)
	testGetParsedRefError(
		t,
		newCompressionUnknownError("foo", knownCompressionTypeStrings...),
		"path/to/foo.tar.gz#compression=foo",
	)
	testGetParsedRefError(
		t,
		newOptionsInvalidKeyError("foo"),
		"path/to/foo.tar.gz#foo=bar",
	)
	testGetParsedRefError(
		t,
		newOptionsInvalidForFormatError(testFormatTar, "path/to/foo.tar.gz#branch=master"),
		"path/to/foo.tar.gz#branch=master",
	)
	testGetParsedRefError(
		t,
		newOptionsInvalidForFormatError(testFormatDir, "path/to/foo#strip_components=1"),
		"path/to/foo#strip_components=1",
	)
	testGetParsedRefError(
		t,
		newOptionsDuplicateKeyError("strip_components"),
		"path/to/foo.tar#strip_components=0,strip_components=1",
	)
	testGetParsedRefError(
		t,
		newOptionsInvalidForFormatError(testFormatDir, "path/to/foo#compression=none"),
		"path/to/foo#compression=none",
	)
	testGetParsedRefError(
		t,
		newCannotSpecifyCompressionForZipError(),
		"path/to/foo.zip#compression=none",
	)
	testGetParsedRefError(
		t,
		newCannotSpecifyCompressionForZipError(),
		"path/to/foo.zip#compression=gzip",
	)
	testGetParsedRefError(
		t,
		newCannotSpecifyCompressionForZipError(),
		"path/to/foo#format=zip,compression=none",
	)
	testGetParsedRefError(
		t,
		newCannotSpecifyCompressionForZipError(),
		"path/to/foo#format=zip,compression=gzip",
	)
}

func testGetParsedRefSuccess(
	t *testing.T,
	expectedRef ParsedRef,
	value string,
) {
	testGetParsedRef(
		t,
		expectedRef,
		nil,
		value,
	)
}

func testGetParsedRefError(
	t *testing.T,
	expectedErr error,
	value string,
) {
	testGetParsedRef(
		t,
		nil,
		expectedErr,
		value,
	)
}

func testGetParsedRef(
	t *testing.T,
	expectedParsedRef ParsedRef,
	expectedErr error,
	value string,
) {
	t.Run(value, func(t *testing.T) {
		t.Parallel()
		parsedRef, err := testNewRefParser(zap.NewNop()).GetParsedRef(
			context.Background(),
			value,
			testGetParsedRefOptions...,
		)
		if expectedErr != nil {
			assert.Equal(t, expectedErr, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, expectedParsedRef, parsedRef)
		}
	})
}

func testNewRefParser(logger *zap.Logger) RefParser {
	return NewRefParser(
		logger,
		WithRawRefProcessor(testRawRefProcessor),
		WithSingleFormat(testFormatBin),
		WithSingleFormat(testFormatJSON),
		WithSingleFormat(
			testFormatBingz,
			WithSingleDefaultCompressionType(CompressionTypeGzip),
		),
		WithSingleFormat(
			testFormatJSONGZ,
			WithSingleDefaultCompressionType(CompressionTypeGzip),
		),
		WithArchiveFormat(
			testFormatTar,
			ArchiveTypeTar,
		),
		WithArchiveFormat(
			testFormatTargz,
			ArchiveTypeTar,
			WithArchiveDefaultCompressionType(CompressionTypeGzip),
		),
		WithArchiveFormat(
			testFormatZip,
			ArchiveTypeZip,
		),
		WithGitFormat(testFormatGit),
		WithDirFormat(testFormatDir),
	)
}

func testRawRefProcessor(rawRef *RawRef) error {
	// if format option is not set and path is "-", default to bin
	var format string
	var compressionType CompressionType
	if rawRef.Path == "-" || app.IsDevNull(rawRef.Path) || app.IsDevStdin(rawRef.Path) || app.IsDevStdout(rawRef.Path) {
		format = testFormatBin
	} else {
		switch filepath.Ext(rawRef.Path) {
		case ".bin":
			format = testFormatBin
		case ".json":
			format = testFormatJSON
		case ".tar":
			format = testFormatTar
		case ".zip":
			format = testFormatZip
		case ".gz":
			compressionType = CompressionTypeGzip
			switch filepath.Ext(strings.TrimSuffix(rawRef.Path, filepath.Ext(rawRef.Path))) {
			case ".bin":
				format = testFormatBin
			case ".json":
				format = testFormatJSON
			case ".tar":
				format = testFormatTar
			default:
				return fmt.Errorf("path %q had .gz extension with unknown format", rawRef.Path)
			}
		case ".zst":
			compressionType = CompressionTypeZstd
			switch filepath.Ext(strings.TrimSuffix(rawRef.Path, filepath.Ext(rawRef.Path))) {
			case ".bin":
				format = testFormatBin
			case ".json":
				format = testFormatJSON
			case ".tar":
				format = testFormatTar
			default:
				return fmt.Errorf("path %q had .zst extension with unknown format", rawRef.Path)
			}
		case ".tgz":
			format = testFormatTar
			compressionType = CompressionTypeGzip
		case ".git":
			format = testFormatGit
		default:
			format = testFormatDir
		}
	}
	rawRef.Format = format
	rawRef.CompressionType = compressionType
	return nil
}

// Copyright 2020 Buf Technologies Inc.
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
			testFormatTargz,
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
			testFormatTargz,
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
			testFormatTargz,
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
			testFormatTargz,
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
		buildGitRef(
			testFormatGit,
			"path/to/dir.git",
			GitSchemeLocal,
			git.NewBranchRefName("master"),
			false,
		),
		"path/to/dir.git#branch=master",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"/path/to/dir.git",
			GitSchemeLocal,
			git.NewBranchRefName("master"),
			false,
		),
		"file:///path/to/dir.git#branch=master",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"path/to/dir.git",
			GitSchemeLocal,
			git.NewTagRefName("v1.0.0"),
			false,
		),
		"path/to/dir.git#tag=v1.0.0",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"hello.com/path/to/dir.git",
			GitSchemeHTTP,
			git.NewBranchRefName("master"),
			false,
		),
		"http://hello.com/path/to/dir.git#branch=master",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"hello.com/path/to/dir.git",
			GitSchemeHTTPS,
			git.NewBranchRefName("master"),
			false,
		),
		"https://hello.com/path/to/dir.git#branch=master",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"user@hello.com:path/to/dir.git",
			GitSchemeSSH,
			git.NewBranchRefName("master"),
			false,
		),
		"ssh://user@hello.com:path/to/dir.git#branch=master",
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
			testFormatBingz,
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
			testFormatJSONGZ,
			"path/to/file.json.gz",
			FileSchemeLocal,
			CompressionTypeGzip,
		),
		"path/to/file.json.gz",
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
		buildGitRef(
			testFormatGit,
			"/path/to/dir",
			GitSchemeLocal,
			git.NewBranchRefName("master"),
			false,
		),
		"/path/to/dir#branch=master,format=git",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"/path/to/dir",
			GitSchemeLocal,
			git.NewBranchRefName("master/foo"),
			false,
		),
		"/path/to/dir#format=git,branch=master/foo",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"path/to/dir",
			GitSchemeLocal,
			git.NewTagRefName("master/foo"),
			false,
		),
		"path/to/dir#tag=master/foo,format=git",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"path/to/dir",
			GitSchemeLocal,
			git.NewTagRefName("master/foo"),
			false,
		),
		"path/to/dir#format=git,tag=master/foo",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"path/to/dir",
			GitSchemeLocal,
			git.NewTagRefName("master/foo"),
			true,
		),
		"path/to/dir#format=git,tag=master/foo,recurse_submodules=true",
	)
	testGetParsedRefSuccess(
		t,
		buildGitRef(
			testFormatGit,
			"path/to/dir",
			GitSchemeLocal,
			git.NewTagRefName("master/foo"),
			false,
		),
		"path/to/dir#format=git,tag=master/foo,recurse_submodules=false",
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
		newMustSpecifyGitRepositoryRefNameError("path/to/foo.git"),
		"path/to/foo.git",
	)
	testGetParsedRefError(
		t,
		newMustSpecifyGitRepositoryRefNameError("path/to/foo"),
		"path/to/foo#format=git",
	)
	testGetParsedRefError(
		t,
		newCannotSpecifyMultipleGitRepositoryRefNamesError(),
		"path/to/foo#format=git,branch=foo,tag=bar",
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
		newOptionsInvalidKeyError("foo"),
		"path/to/foo.tar.gz#foo=bar",
	)
	testGetParsedRefError(
		t,
		newOptionsInvalidForFormatError(testFormatTargz, "path/to/foo.tar.gz#branch=master"),
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
		WithFormatParser(testFormatParser),
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
		WithGitFormat(testFormatGit),
		WithDirFormat(testFormatDir),
	)
}

func testFormatParser(rawPath string) (string, error) {
	// if format option is not set and path is "-", default to bin
	if rawPath == "-" || rawPath == app.DevNullFilePath {
		return testFormatBin, nil
	}
	switch filepath.Ext(rawPath) {
	case ".bin":
		return testFormatBin, nil
	case ".json":
		return testFormatJSON, nil
	case ".tar":
		return testFormatTar, nil
	case ".gz":
		switch filepath.Ext(strings.TrimSuffix(rawPath, filepath.Ext(rawPath))) {
		case ".bin":
			return testFormatBingz, nil
		case ".json":
			return testFormatJSONGZ, nil
		case ".tar":
			return testFormatTargz, nil
		default:
			return "", fmt.Errorf("path %q had .gz extension with unknown format", rawPath)
		}
	case ".tgz":
		return testFormatTargz, nil
	case ".git":
		return testFormatGit, nil
	default:
		return testFormatDir, nil
	}
}

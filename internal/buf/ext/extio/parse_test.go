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

package extio

import (
	"fmt"
	"testing"

	iov1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/io/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/stretchr/testify/assert"
)

func TestParseInputRefSuccess(t *testing.T) {
	testParseInputRefSuccess(
		t,
		testNewInputBucketRef(
			"path/to/dir",
		),
		"path/to/dir",
	)
	testParseInputRefSuccess(
		t,
		testNewInputArchiveRef(
			iov1beta1.FileScheme_FILE_SCHEME_LOCAL,
			iov1beta1.ArchiveFormat_ARCHIVE_FORMAT_TAR,
			"path/to/file.tar",
			0,
		),
		"path/to/file.tar",
	)
	testParseInputRefSuccess(
		t,
		testNewInputArchiveRef(
			iov1beta1.FileScheme_FILE_SCHEME_LOCAL,
			iov1beta1.ArchiveFormat_ARCHIVE_FORMAT_TAR,
			"/path/to/file.tar",
			0,
		),
		"file:///path/to/file.tar",
	)
	testParseInputRefSuccess(
		t,
		testNewInputArchiveRef(
			iov1beta1.FileScheme_FILE_SCHEME_LOCAL,
			iov1beta1.ArchiveFormat_ARCHIVE_FORMAT_TAR,
			"path/to/file.tar",
			1,
		),
		"path/to/file.tar#strip_components=1",
	)
	testParseInputRefSuccess(
		t,
		testNewInputArchiveRef(
			iov1beta1.FileScheme_FILE_SCHEME_LOCAL,
			iov1beta1.ArchiveFormat_ARCHIVE_FORMAT_TARGZ,
			"path/to/file.tar.gz",
			0,
		),
		"path/to/file.tar.gz",
	)
	testParseInputRefSuccess(
		t,
		testNewInputArchiveRef(
			iov1beta1.FileScheme_FILE_SCHEME_LOCAL,
			iov1beta1.ArchiveFormat_ARCHIVE_FORMAT_TARGZ,
			"path/to/file.tar.gz",
			1,
		),
		"path/to/file.tar.gz#strip_components=1",
	)
	testParseInputRefSuccess(
		t,
		testNewInputArchiveRef(
			iov1beta1.FileScheme_FILE_SCHEME_LOCAL,
			iov1beta1.ArchiveFormat_ARCHIVE_FORMAT_TARGZ,
			"path/to/file.tgz",
			0,
		),
		"path/to/file.tgz",
	)
	testParseInputRefSuccess(
		t,
		testNewInputArchiveRef(
			iov1beta1.FileScheme_FILE_SCHEME_LOCAL,
			iov1beta1.ArchiveFormat_ARCHIVE_FORMAT_TARGZ,
			"path/to/file.tgz",
			1,
		),
		"path/to/file.tgz#strip_components=1",
	)
	testParseInputRefSuccess(
		t,
		testNewInputArchiveRef(
			iov1beta1.FileScheme_FILE_SCHEME_HTTP,
			iov1beta1.ArchiveFormat_ARCHIVE_FORMAT_TAR,
			"path/to/file.tar",
			0,
		),
		"http://path/to/file.tar",
	)
	testParseInputRefSuccess(
		t,
		testNewInputArchiveRef(
			iov1beta1.FileScheme_FILE_SCHEME_HTTPS,
			iov1beta1.ArchiveFormat_ARCHIVE_FORMAT_TAR,
			"path/to/file.tar",
			0,
		),
		"https://path/to/file.tar",
	)
	testParseInputRefSuccess(
		t,
		testNewInputGitRepositoryRef(
			iov1beta1.GitRepositoryScheme_GIT_REPOSITORY_SCHEME_LOCAL,
			"path/to/dir.git",
			"master",
			"",
			iov1beta1.GitRepositorySubmoduleBehavior_GIT_REPOSITORY_SUBMODULE_BEHAVIOR_NONE,
		),
		"path/to/dir.git#branch=master",
	)
	testParseInputRefSuccess(
		t,
		testNewInputGitRepositoryRef(
			iov1beta1.GitRepositoryScheme_GIT_REPOSITORY_SCHEME_LOCAL,
			"/path/to/dir.git",
			"master",
			"",
			iov1beta1.GitRepositorySubmoduleBehavior_GIT_REPOSITORY_SUBMODULE_BEHAVIOR_NONE,
		),
		"file:///path/to/dir.git#branch=master",
	)
	testParseInputRefSuccess(
		t,
		testNewInputGitRepositoryRef(
			iov1beta1.GitRepositoryScheme_GIT_REPOSITORY_SCHEME_LOCAL,
			"path/to/dir.git",
			"",
			"master",
			iov1beta1.GitRepositorySubmoduleBehavior_GIT_REPOSITORY_SUBMODULE_BEHAVIOR_NONE,
		),
		"path/to/dir.git#tag=master",
	)
	testParseInputRefSuccess(
		t,
		testNewInputGitRepositoryRef(
			iov1beta1.GitRepositoryScheme_GIT_REPOSITORY_SCHEME_HTTP,
			"hello.com/path/to/dir.git",
			"master",
			"",
			iov1beta1.GitRepositorySubmoduleBehavior_GIT_REPOSITORY_SUBMODULE_BEHAVIOR_NONE,
		),
		"http://hello.com/path/to/dir.git#branch=master",
	)
	testParseInputRefSuccess(
		t,
		testNewInputGitRepositoryRef(
			iov1beta1.GitRepositoryScheme_GIT_REPOSITORY_SCHEME_HTTPS,
			"hello.com/path/to/dir.git",
			"master",
			"",
			iov1beta1.GitRepositorySubmoduleBehavior_GIT_REPOSITORY_SUBMODULE_BEHAVIOR_NONE,
		),
		"https://hello.com/path/to/dir.git#branch=master",
	)
	testParseInputRefSuccess(
		t,
		testNewInputGitRepositoryRef(
			iov1beta1.GitRepositoryScheme_GIT_REPOSITORY_SCHEME_SSH,
			"user@hello.com:path/to/dir.git",
			"master",
			"",
			iov1beta1.GitRepositorySubmoduleBehavior_GIT_REPOSITORY_SUBMODULE_BEHAVIOR_NONE,
		),
		"ssh://user@hello.com:path/to/dir.git#branch=master",
	)
	testParseInputRefSuccess(
		t,
		testNewInputImageRef(
			iov1beta1.FileScheme_FILE_SCHEME_LOCAL,
			iov1beta1.ImageFormat_IMAGE_FORMAT_BIN,
			"path/to/file.bin",
		),
		"path/to/file.bin",
	)
	testParseInputRefSuccess(
		t,
		testNewInputImageRef(
			iov1beta1.FileScheme_FILE_SCHEME_LOCAL,
			iov1beta1.ImageFormat_IMAGE_FORMAT_BINGZ,
			"path/to/file.bin.gz",
		),
		"path/to/file.bin.gz",
	)
	testParseInputRefSuccess(
		t,
		testNewInputImageRef(
			iov1beta1.FileScheme_FILE_SCHEME_LOCAL,
			iov1beta1.ImageFormat_IMAGE_FORMAT_JSON,
			"path/to/file.json",
		),
		"path/to/file.json",
	)
	testParseInputRefSuccess(
		t,
		testNewInputImageRef(
			iov1beta1.FileScheme_FILE_SCHEME_LOCAL,
			iov1beta1.ImageFormat_IMAGE_FORMAT_JSONGZ,
			"path/to/file.json.gz",
		),
		"path/to/file.json.gz",
	)
	testParseInputRefSuccess(
		t,
		testNewInputImageRef(
			iov1beta1.FileScheme_FILE_SCHEME_STDIO,
			iov1beta1.ImageFormat_IMAGE_FORMAT_BIN,
			"",
		),
		"-",
	)
	testParseInputRefSuccess(
		t,
		testNewInputImageRef(
			iov1beta1.FileScheme_FILE_SCHEME_STDIO,
			iov1beta1.ImageFormat_IMAGE_FORMAT_JSON,
			"",
		),
		"-#format=json",
	)
	testParseInputRefSuccess(
		t,
		testNewInputImageRef(
			iov1beta1.FileScheme_FILE_SCHEME_NULL,
			iov1beta1.ImageFormat_IMAGE_FORMAT_BIN,
			"",
		),
		app.DevNullFilePath,
	)
	testParseInputRefSuccess(
		t,
		testNewInputImageRef(
			iov1beta1.FileScheme_FILE_SCHEME_LOCAL,
			iov1beta1.ImageFormat_IMAGE_FORMAT_BIN,
			"path/to/dir",
		),
		"path/to/dir#format=bin",
	)
	testParseInputRefSuccess(
		t,
		testNewInputGitRepositoryRef(
			iov1beta1.GitRepositoryScheme_GIT_REPOSITORY_SCHEME_LOCAL,
			"/path/to/dir",
			"master",
			"",
			iov1beta1.GitRepositorySubmoduleBehavior_GIT_REPOSITORY_SUBMODULE_BEHAVIOR_NONE,
		),
		"/path/to/dir#branch=master,format=git",
	)
	testParseInputRefSuccess(
		t,
		testNewInputGitRepositoryRef(
			iov1beta1.GitRepositoryScheme_GIT_REPOSITORY_SCHEME_LOCAL,
			"/path/to/dir",
			"master/foo",
			"",
			iov1beta1.GitRepositorySubmoduleBehavior_GIT_REPOSITORY_SUBMODULE_BEHAVIOR_NONE,
		),
		"/path/to/dir#format=git,branch=master/foo",
	)
	testParseInputRefSuccess(
		t,
		testNewInputGitRepositoryRef(
			iov1beta1.GitRepositoryScheme_GIT_REPOSITORY_SCHEME_LOCAL,
			"path/to/dir",
			"",
			"master/foo",
			iov1beta1.GitRepositorySubmoduleBehavior_GIT_REPOSITORY_SUBMODULE_BEHAVIOR_NONE,
		),
		"path/to/dir#tag=master/foo,format=git",
	)
	testParseInputRefSuccess(
		t,
		testNewInputGitRepositoryRef(
			iov1beta1.GitRepositoryScheme_GIT_REPOSITORY_SCHEME_LOCAL,
			"path/to/dir",
			"",
			"master/foo",
			iov1beta1.GitRepositorySubmoduleBehavior_GIT_REPOSITORY_SUBMODULE_BEHAVIOR_NONE,
		),
		"path/to/dir#format=git,tag=master/foo",
	)
	testParseInputRefSuccess(
		t,
		testNewInputGitRepositoryRef(
			iov1beta1.GitRepositoryScheme_GIT_REPOSITORY_SCHEME_LOCAL,
			"path/to/dir",
			"",
			"master/foo",
			iov1beta1.GitRepositorySubmoduleBehavior_GIT_REPOSITORY_SUBMODULE_BEHAVIOR_RECURSIVE,
		),
		"path/to/dir#format=git,tag=master/foo,recurse_submodules=true",
	)
	testParseInputRefSuccess(
		t,
		testNewInputGitRepositoryRef(
			iov1beta1.GitRepositoryScheme_GIT_REPOSITORY_SCHEME_LOCAL,
			"path/to/dir",
			"",
			"master/foo",
			iov1beta1.GitRepositorySubmoduleBehavior_GIT_REPOSITORY_SUBMODULE_BEHAVIOR_NONE,
		),
		"path/to/dir#format=git,tag=master/foo,recurse_submodules=false",
	)
	testParseInputRefSuccess(
		t,
		testNewInputArchiveRef(
			iov1beta1.FileScheme_FILE_SCHEME_LOCAL,
			iov1beta1.ArchiveFormat_ARCHIVE_FORMAT_TARGZ,
			"path/to/file",
			1,
		),
		"path/to/file#format=targz,strip_components=1",
	)
}

func TestParseInputRefError(t *testing.T) {
	testParseInputRefError(
		t,
		newValueEmptyError(),
		"",
	)
	testParseInputRefError(
		t,
		newValueMultipleHashtagsError("foo#format=git#branch=master"),
		"foo#format=git#branch=master",
	)
	testParseInputRefError(
		t,
		newValueStartsWithHashtagError("#path/to/dir"),
		"#path/to/dir",
	)
	testParseInputRefError(
		t,
		newValueEndsWithHashtagError("path/to/dir#"),
		"path/to/dir#",
	)
	testParseInputRefError(
		t,
		newInvalidDirPathError("-"),
		"-#format=dir",
	)
	testParseInputRefError(
		t,
		newInvalidGitPathError("-"),
		"-#format=git,branch=master",
	)
	testParseInputRefError(
		t,
		newMustSpecifyGitRepositoryRefNameError("path/to/foo.git"),
		"path/to/foo.git",
	)
	testParseInputRefError(
		t,
		newMustSpecifyGitRepositoryRefNameError("path/to/foo"),
		"path/to/foo#format=git",
	)
	testParseInputRefError(
		t,
		newCannotSpecifyMultipleGitRepositoryRefNamesError(),
		"path/to/foo#format=git,branch=foo,tag=bar",
	)
	testParseInputRefError(
		t,
		newOptionsDuplicateKeyError("branch"),
		"path/to/foo#format=git,branch=foo,branch=bar",
	)
	testParseInputRefError(
		t,
		newPathUnknownGzError("path/to/foo.gz"),
		"path/to/foo.gz",
	)
	testParseInputRefError(
		t,
		newPathUnknownGzError("path/to/foo.bar.gz"),
		"path/to/foo.bar.gz",
	)
	testParseInputRefError(
		t,
		newOptionsInvalidError("bar"),
		"path/to/foo#bar",
	)
	testParseInputRefError(
		t,
		newOptionsInvalidError("bar="),
		"path/to/foo#bar=",
	)
	testParseInputRefError(
		t,
		newOptionsInvalidError("format=bin,bar="),
		"path/to/foo#format=bin,bar=",
	)
	testParseInputRefError(
		t,
		newOptionsInvalidError("format=bin,=bar"),
		"path/to/foo#format=bin,=bar",
	)
	testParseInputRefError(
		t,
		newFormatOverrideNotAllowedForDevNullError(app.DevNullFilePath),
		fmt.Sprintf("%s#format=bin", app.DevNullFilePath),
	)
	testParseInputRefError(
		t,
		newFormatUnknownError("bar"),
		"path/to/foo#format=bar",
	)
	testParseInputRefError(
		t,
		newOptionsCouldNotParseStripComponentsError("foo"),
		"path/to/foo.tar.gz#strip_components=foo",
	)
	testParseInputRefError(
		t,
		newOptionsInvalidKeyError("foo"),
		"path/to/foo.tar.gz#foo=bar",
	)
	testParseInputRefError(
		t,
		newOptionsInvalidForFormatError(formatTarGz, "path/to/foo.tar.gz#branch=master"),
		"path/to/foo.tar.gz#branch=master",
	)
	testParseInputRefError(
		t,
		newOptionsInvalidForFormatError(formatDir, "path/to/foo#strip_components=1"),
		"path/to/foo#strip_components=1",
	)
	testParseInputRefError(
		t,
		newOptionsDuplicateKeyError("strip_components"),
		"path/to/foo.tar#strip_components=0,strip_components=1",
	)
}

func testParseInputRefSuccess(
	t *testing.T,
	expectedInputRef *iov1beta1.InputRef,
	value string,
) {
	testParseInputRef(
		t,
		expectedInputRef,
		nil,
		value,
	)
}

func testParseInputRefError(
	t *testing.T,
	expectedErr error,
	value string,
) {
	testParseInputRef(
		t,
		nil,
		expectedErr,
		value,
	)
}

func testParseInputRef(
	t *testing.T,
	expectedInputRef *iov1beta1.InputRef,
	expectedErr error,
	value string,
) {
	t.Run(value, func(t *testing.T) {
		t.Parallel()
		inputRef, err := ParseInputRef(value)
		if expectedErr != nil {
			assert.Equal(t, expectedErr, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, expectedInputRef, inputRef)
		}
	})
}

func testNewInputImageRef(
	fileScheme iov1beta1.FileScheme,
	imageFormat iov1beta1.ImageFormat,
	path string,
) *iov1beta1.InputRef {
	return &iov1beta1.InputRef{
		Value: &iov1beta1.InputRef_ImageRef{
			ImageRef: &iov1beta1.ImageRef{
				FileRef: &iov1beta1.FileRef{
					Scheme: fileScheme,
					Path:   path,
				},
				Format: imageFormat,
			},
		},
	}
}

func testNewInputArchiveRef(
	fileScheme iov1beta1.FileScheme,
	archiveFormat iov1beta1.ArchiveFormat,
	path string,
	stripComponents uint32,
) *iov1beta1.InputRef {
	return &iov1beta1.InputRef{
		Value: &iov1beta1.InputRef_SourceRef{
			SourceRef: &iov1beta1.SourceRef{
				Value: &iov1beta1.SourceRef_ArchiveRef{
					ArchiveRef: &iov1beta1.ArchiveRef{
						FileRef: &iov1beta1.FileRef{
							Scheme: fileScheme,
							Path:   path,
						},
						Format:          archiveFormat,
						StripComponents: stripComponents,
					},
				},
			},
		},
	}
}

func testNewInputGitRepositoryRef(
	gitRepositoryScheme iov1beta1.GitRepositoryScheme,
	path string,
	branch string,
	tag string,
	gitSubmoduleBehavior iov1beta1.GitRepositorySubmoduleBehavior,
) *iov1beta1.InputRef {
	gitRepositoryRef := &iov1beta1.GitRepositoryRef{
		Scheme:            gitRepositoryScheme,
		Path:              path,
		SubmoduleBehavior: gitSubmoduleBehavior,
	}
	if branch != "" {
		gitRepositoryRef.Reference = &iov1beta1.GitRepositoryRef_Branch{
			Branch: branch,
		}
	} else if tag != "" {
		gitRepositoryRef.Reference = &iov1beta1.GitRepositoryRef_Tag{
			Tag: tag,
		}
	}
	return &iov1beta1.InputRef{
		Value: &iov1beta1.InputRef_SourceRef{
			SourceRef: &iov1beta1.SourceRef{
				Value: &iov1beta1.SourceRef_GitRepositoryRef{
					GitRepositoryRef: gitRepositoryRef,
				},
			},
		},
	}
}

func testNewInputBucketRef(
	path string,
) *iov1beta1.InputRef {
	return &iov1beta1.InputRef{
		Value: &iov1beta1.InputRef_SourceRef{
			SourceRef: &iov1beta1.SourceRef{
				Value: &iov1beta1.SourceRef_BucketRef{
					BucketRef: &iov1beta1.BucketRef{
						Scheme: iov1beta1.BucketScheme_BUCKET_SCHEME_LOCAL,
						Path:   path,
					},
				},
			},
		},
	}
}

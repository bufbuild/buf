package internal

import (
	"fmt"
	"testing"

	"github.com/bufbuild/buf/internal/pkg/osutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testValueFlagName = "--test-value"

func TestParseInputRefSuccess(t *testing.T) {
	devNull, err := osutil.DevNull()
	require.NoError(t, err)

	testParseInputRefSuccess(
		t,
		&InputRef{
			Format: FormatDir,
			Path:   "path/to/dir",
		},
		"path/to/dir",
	)
	testParseInputRefSuccess(
		t,
		&InputRef{
			Format: FormatTar,
			Path:   "path/to/file.tar",
		},
		"path/to/file.tar",
	)
	testParseInputRefSuccess(
		t,
		&InputRef{
			Format:          FormatTar,
			Path:            "path/to/file.tar",
			StripComponents: 1,
		},
		"path/to/file.tar#strip_components=1",
	)
	testParseInputRefSuccess(
		t,
		&InputRef{
			Format: FormatTarGz,
			Path:   "path/to/file.tar.gz",
		},
		"path/to/file.tar.gz",
	)
	testParseInputRefSuccess(
		t,
		&InputRef{
			Format:          FormatTarGz,
			Path:            "path/to/file.tar.gz",
			StripComponents: 1,
		},
		"path/to/file.tar.gz#strip_components=1",
	)
	testParseInputRefSuccess(
		t,
		&InputRef{
			Format: FormatTarGz,
			Path:   "path/to/file.tgz",
		},
		"path/to/file.tgz",
	)
	testParseInputRefSuccess(
		t,
		&InputRef{
			Format:          FormatTarGz,
			Path:            "path/to/file.tgz",
			StripComponents: 1,
		},
		"path/to/file.tgz#strip_components=1",
	)
	testParseInputRefSuccess(
		t,
		&InputRef{
			Format:    FormatGit,
			Path:      "path/to/dir.git",
			GitBranch: "master",
		},
		"path/to/dir.git#branch=master",
	)
	testParseInputRefSuccess(
		t,
		&InputRef{
			Format: FormatBin,
			Path:   "path/to/file.bin",
		},
		"path/to/file.bin",
	)
	testParseInputRefSuccess(
		t,
		&InputRef{
			Format: FormatBinGz,
			Path:   "path/to/file.bin.gz",
		},
		"path/to/file.bin.gz",
	)
	testParseInputRefSuccess(
		t,
		&InputRef{
			Format: FormatJSON,
			Path:   "path/to/file.json",
		},
		"path/to/file.json",
	)
	testParseInputRefSuccess(
		t,
		&InputRef{
			Format: FormatJSONGz,
			Path:   "path/to/file.json.gz",
		},
		"path/to/file.json.gz",
	)
	testParseInputRefSuccess(
		t,
		&InputRef{
			Format: FormatBin,
			Path:   "-",
		},
		"-",
	)
	testParseInputRefSuccess(
		t,
		&InputRef{
			Format: FormatJSON,
			Path:   "-",
		},
		"-#format=json",
	)
	testParseInputRefSuccess(
		t,
		&InputRef{
			Format: FormatBin,
			Path:   devNull,
		},
		devNull,
	)
	testParseInputRefSuccess(
		t,
		&InputRef{
			Format: FormatBin,
			Path:   "path/to/dir",
		},
		"path/to/dir#format=bin",
	)
	testParseInputRefSuccess(
		t,
		&InputRef{
			Format:    FormatGit,
			Path:      "path/to/dir",
			GitBranch: "master/foo",
		},
		"path/to/dir#branch=master/foo,format=git",
	)
	testParseInputRefSuccess(
		t,
		&InputRef{
			Format:    FormatGit,
			Path:      "path/to/dir",
			GitBranch: "master/foo",
		},
		"path/to/dir#format=git,branch=master/foo",
	)
	testParseInputRefSuccess(
		t,
		&InputRef{
			Format:          FormatTarGz,
			Path:            "path/to/file",
			StripComponents: 1,
		},
		"path/to/file#format=targz,strip_components=1",
	)
}

func TestParseInputRefError(t *testing.T) {
	devNull, err := osutil.DevNull()
	require.NoError(t, err)

	testParseInputRefErrorBasic(
		t,
		newValueEmptyError(testValueFlagName),
		"",
	)
	testParseInputRefErrorBasic(
		t,
		newValueMultipleHashtagsError(testValueFlagName, "foo#format=git#branch=master"),
		"foo#format=git#branch=master",
	)
	testParseInputRefErrorBasic(
		t,
		newValueStartsWithHashtagError(testValueFlagName, "#path/to/dir"),
		"#path/to/dir",
	)
	testParseInputRefErrorBasic(
		t,
		newValueEndsWithHashtagError(testValueFlagName, "path/to/dir#"),
		"path/to/dir#",
	)
	testParseInputRefErrorBasic(
		t,
		newFormatNotFileForDashPathError(testValueFlagName, FormatDir),
		"-#format=dir",
	)
	testParseInputRefErrorBasic(
		t,
		newFormatNotFileForDashPathError(testValueFlagName, FormatGit),
		"-#format=git,branch=master",
	)
	testParseInputRefError(
		t,
		newFormatMustBeSourceError(FormatBin),
		"-",
		true,
		false,
	)
	testParseInputRefError(
		t,
		newFormatMustBeImageError(FormatDir),
		"path/to/dir",
		false,
		true,
	)
	testParseInputRefErrorBasic(
		t,
		newMustSpecifyGitBranchError(testValueFlagName, "path/to/foo.git"),
		"path/to/foo.git",
	)
	testParseInputRefErrorBasic(
		t,
		newMustSpecifyGitBranchError(testValueFlagName, "path/to/foo#format=git"),
		"path/to/foo#format=git",
	)
	testParseInputRefErrorBasic(
		t,
		newPathUnknownGzError(testValueFlagName, "path/to/foo.gz"),
		"path/to/foo.gz",
	)
	testParseInputRefErrorBasic(
		t,
		newPathUnknownGzError(testValueFlagName, "path/to/foo.bar.gz"),
		"path/to/foo.bar.gz",
	)
	testParseInputRefErrorBasic(
		t,
		newOptionsInvalidError(testValueFlagName, "bar"),
		"path/to/foo#bar",
	)
	testParseInputRefErrorBasic(
		t,
		newOptionsInvalidError(testValueFlagName, "bar="),
		"path/to/foo#bar=",
	)
	testParseInputRefErrorBasic(
		t,
		newOptionsInvalidError(testValueFlagName, "format=bin,bar="),
		"path/to/foo#format=bin,bar=",
	)
	testParseInputRefErrorBasic(
		t,
		newOptionsInvalidError(testValueFlagName, "format=bin,=bar"),
		"path/to/foo#format=bin,=bar",
	)
	testParseInputRefErrorBasic(
		t,
		newFormatOverrideNotAllowedForDevNullError(testValueFlagName, devNull),
		fmt.Sprintf("%s#format=bin", devNull),
	)
	testParseInputRefErrorBasic(
		t,
		newFormatOverrideUnknownError(testValueFlagName, "bar"),
		"path/to/foo#format=bar",
	)
	testParseInputRefErrorBasic(
		t,
		newOptionsCouldNotParseStripComponentsError(testValueFlagName, "foo"),
		"path/to/foo.tar.gz#strip_components=foo",
	)
	testParseInputRefErrorBasic(
		t,
		newOptionsInvalidKeyError(testValueFlagName, "foo"),
		"path/to/foo.tar.gz#foo=bar",
	)
	testParseInputRefErrorBasic(
		t,
		newOptionsInvalidForFormatError(testValueFlagName, FormatTarGz, "branch=master"),
		"path/to/foo.tar.gz#branch=master",
	)
	testParseInputRefErrorBasic(
		t,
		newOptionsInvalidForFormatError(testValueFlagName, FormatDir, "strip_components=1"),
		"path/to/foo#strip_components=1",
	)
}

func testParseInputRefSuccess(
	t *testing.T,
	expectedInputRef *InputRef,
	value string,
) {
	testParseInputRef(
		t,
		expectedInputRef,
		nil,
		value,
		false,
		false,
	)
}

func testParseInputRefErrorBasic(
	t *testing.T,
	expectedErr error,
	value string,
) {
	testParseInputRefError(
		t,
		expectedErr,
		value,
		false,
		false,
	)
}

func testParseInputRefError(
	t *testing.T,
	expectedErr error,
	value string,
	onlySources bool,
	onlyImages bool,
) {
	testParseInputRef(
		t,
		nil,
		expectedErr,
		value,
		onlySources,
		onlyImages,
	)
}

func testParseInputRef(
	t *testing.T,
	expectedInputRef *InputRef,
	expectedErr error,
	value string,
	onlySources bool,
	onlyImages bool,
) {
	t.Run(value, func(t *testing.T) {
		t.Parallel()

		inputRef, err := NewInputRefParser(testValueFlagName).ParseInputRef(value, onlySources, onlyImages)
		if expectedErr != nil {
			assert.Equal(t, expectedErr, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, expectedInputRef, inputRef)
		}
	})
}

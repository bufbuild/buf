package buf

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/internal/pkg/cli"
	"github.com/bufbuild/buf/internal/pkg/cli/clicobra"
	"github.com/bufbuild/buf/internal/pkg/osutil"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuccess1(t *testing.T) {
	devNull, err := osutil.DevNull()
	require.NoError(t, err)
	testRun(t, 0, ``, "image", "build", "-o", devNull, "--source", filepath.Join("testdata", "success"))
}

func TestSuccess2(t *testing.T) {
	devNull, err := osutil.DevNull()
	require.NoError(t, err)
	testRun(t, 0, ``, "image", "build", "-o", devNull, "--exclude-imports", "--source", filepath.Join("testdata", "success"))
}

func TestSuccess3(t *testing.T) {
	devNull, err := osutil.DevNull()
	require.NoError(t, err)
	testRun(t, 0, ``, "image", "build", "-o", devNull, "--exclude-source-info", "--source", filepath.Join("testdata", "success"))
}

func TestSuccess4(t *testing.T) {
	devNull, err := osutil.DevNull()
	require.NoError(t, err)
	testRun(t, 0, ``, "image", "build", "-o", devNull, "--exclude-imports", "--exclude-source-info", "--source", filepath.Join("testdata", "success"))
}

func TestSuccess5(t *testing.T) {
	devNull, err := osutil.DevNull()
	require.NoError(t, err)
	testRun(t, 0, ``, "image", "build", "-o", devNull, "--exclude-imports", "--exclude-source-info", "-o", devNull, "--source", filepath.Join("testdata", "success"))
}

func TestSuccess6(t *testing.T) {
	testRun(t, 0, ``, "check", "lint", "--input", filepath.Join("testdata", "success"))
}

func TestSuccessProfile1(t *testing.T) {
	devNull, err := osutil.DevNull()
	require.NoError(t, err)
	testRunProfile(t, 0, ``, "image", "build", "-o", devNull, "--source", filepath.Join("testdata", "success"))
}

func TestFail1(t *testing.T) {
	devNull, err := osutil.DevNull()
	require.NoError(t, err)
	testRun(
		t,
		0,
		``,
		"image", "build", "-o", devNull,
		"--source",
		filepath.Join("testdata", "fail"),
	)
}

func TestFail2(t *testing.T) {
	devNull, err := osutil.DevNull()
	require.NoError(t, err)
	testRun(
		t,
		0,
		``,
		"image", "build", "-o", devNull,
		"--exclude-imports",
		"--source",
		filepath.Join("testdata", "fail"),
	)
}

func TestFail3(t *testing.T) {
	devNull, err := osutil.DevNull()
	require.NoError(t, err)
	testRun(
		t,
		0,
		``,
		"image", "build", "-o", devNull,
		"--exclude-source-info",
		"--source",
		filepath.Join("testdata", "fail"),
	)
}

func TestFail4(t *testing.T) {
	devNull, err := osutil.DevNull()
	require.NoError(t, err)
	testRun(
		t,
		0,
		``,
		"image", "build", "-o", devNull,
		"--exclude-imports",
		"--exclude-source-info",
		"--source",
		filepath.Join("testdata", "fail"),
	)
}

func TestFail5(t *testing.T) {
	testRun(
		t,
		1,
		`testdata/fail/buf/buf.proto:3:1:Files with package "other" must be within a directory "other" relative to root but were in directory "buf".
        testdata/fail/buf/buf.proto:6:9:Field name "oneTwo" should be lower_snake_case, such as "one_two".`,
		"check",
		"lint",
		"--input",
		filepath.Join("testdata", "fail"),
	)
}

func TestFail6(t *testing.T) {
	testRun(
		t,
		1,
		`testdata/fail/buf/buf.proto:3:1:Files with package "other" must be within a directory "other" relative to root but were in directory "buf".
        testdata/fail/buf/buf.proto:6:9:Field name "oneTwo" should be lower_snake_case, such as "one_two".`,
		"check",
		"lint",
		"--input",
		filepath.Join("testdata", "fail"),
		"--file",
		filepath.Join("testdata", "fail", "buf", "buf.proto"),
	)
}

func TestFail7(t *testing.T) {
	testRun(
		t,
		1,
		`testdata/fail/buf/buf.proto:3:1:Files with package "other" must be within a directory "other" relative to root but were in directory "fail/buf".
        testdata/fail/buf/buf.proto:6:9:Field name "oneTwo" should be lower_snake_case, such as "one_two".`,
		"check",
		"lint",
		"--file",
		filepath.Join("testdata", "fail", "buf", "buf.proto"),
		"--input",
		filepath.Join("testdata"),
		"--input-config",
		`{"lint":{"use":["BASIC"]}}`,
	)
}

func TestFail8(t *testing.T) {
	testRun(
		t,
		1,
		`testdata/fail2/buf/buf.proto:6:9:Field name "oneTwo" should be lower_snake_case, such as "one_two".
		testdata/fail2/buf/buf2.proto:9:9:Field name "oneThree" should be lower_snake_case, such as "one_three".`,
		"check",
		"lint",
		"--input",
		filepath.Join("testdata", "fail2"),
	)
}

func TestFail9(t *testing.T) {
	testRun(
		t,
		1,
		`testdata/fail2/buf/buf.proto:6:9:Field name "oneTwo" should be lower_snake_case, such as "one_two".`,
		"check",
		"lint",
		"--input",
		filepath.Join("testdata", "fail2"),
		"--file",
		filepath.Join("testdata", "fail2", "buf", "buf.proto"),
	)
}

func TestFail10(t *testing.T) {
	testRun(
		t,
		0,
		``,
		"check",
		"lint",
		"--input",
		filepath.Join("testdata", "fail2"),
		"--file",
		filepath.Join("testdata", "fail2", "buf", "buf3.proto"),
	)
}

func TestFail11(t *testing.T) {
	testRun(
		t,
		1,
		`testdata/fail2/buf/buf2.proto:5:8:testdata/buf/buf.proto: does not exist`,
		"check",
		"lint",
		"--file",
		filepath.Join("testdata", "fail2", "buf", "buf2.proto"),
		"--input",
		filepath.Join("testdata"),
	)
}

func TestFailCheckBreaking1(t *testing.T) {
	testRun(
		t,
		1,
		`
		../../bufcheck/bufbreaking/testdata/breaking_field_no_delete/1.proto:5:1:Previously present field "3" with name "three" on message "Two" was deleted.
		../../bufcheck/bufbreaking/testdata/breaking_field_no_delete/1.proto:10:1:Previously present field "3" with name "three" on message "Three" was deleted.
		../../bufcheck/bufbreaking/testdata/breaking_field_no_delete/1.proto:12:5:Previously present field "3" with name "three" on message "Five" was deleted.
		../../bufcheck/bufbreaking/testdata/breaking_field_no_delete/1.proto:22:3:Previously present field "3" with name "three" on message "Seven" was deleted.
		../../bufcheck/bufbreaking/testdata/breaking_field_no_delete/2.proto:57:1:Previously present field "3" with name "three" on message "Nine" was deleted.
		`,
		"check",
		"breaking",
		"--input",
		// can't bother right now to filepath.Join this
		"../../bufcheck/bufbreaking/testdata/breaking_field_no_delete",
		"--against-input",
		"../../bufcheck/bufbreaking/testdata_previous/breaking_field_no_delete",
	)
}

func TestCheckLsLintCheckers1(t *testing.T) {
	testRun(
		t,
		0,
		`
		ID                       CATEGORIES  PURPOSE
		RPC_NO_CLIENT_STREAMING  UNARY_RPC   Checks that RPCs are not client streaming.
		RPC_NO_SERVER_STREAMING  UNARY_RPC   Checks that RPCs are not server streaming.
		`,
		"check",
		"ls-lint-checkers",
		"--all",
		"--category",
		"UNARY_RPC",
	)
}

func TestCheckLsLintCheckers2(t *testing.T) {
	testRun(
		t,
		0,
		`
		ID                       CATEGORIES                            PURPOSE
		PACKAGE_DIRECTORY_MATCH  MINIMAL, BASIC, DEFAULT, FILE_LAYOUT  Checks that all files with are in a directory that matches their package name.
		ENUM_NO_ALLOW_ALIAS      MINIMAL, BASIC, DEFAULT, SENSIBLE     Checks that enums do not have the allow_alias option set.
		`,
		"check",
		"ls-lint-checkers",
		"--config",
		filepath.Join("testdata", "small_list_checkers", "buf.yaml"),
	)
}

func TestCheckLsBreakingCheckers1(t *testing.T) {
	testRun(
		t,
		0,
		`
		ID                                           CATEGORIES                      PURPOSE
		FIELD_SAME_JSON_NAME                         FILE, PACKAGE, WIRE_JSON        Checks that fields have the same value for the json_name option.
		FIELD_SAME_NAME                              FILE, PACKAGE, WIRE_JSON        Checks that fields have the same names in a given message.
		ENUM_VALUE_SAME_NUMBER                       FILE, PACKAGE, WIRE_JSON, WIRE  Checks that enum values have the same number.
		FIELD_SAME_LABEL                             FILE, PACKAGE, WIRE_JSON, WIRE  Checks that fields have the same labels in a given message.
		FIELD_SAME_ONEOF                             FILE, PACKAGE, WIRE_JSON, WIRE  Checks that fields have the same oneofs in a given message.
		FIELD_SAME_TYPE                              FILE, PACKAGE, WIRE_JSON, WIRE  Checks that fields have the same types in a given message.
		MESSAGE_SAME_MESSAGE_SET_WIRE_FORMAT         FILE, PACKAGE, WIRE_JSON, WIRE  Checks that messages have the same value for the message_set_wire_format option.
		RESERVED_ENUM_NO_DELETE                      FILE, PACKAGE, WIRE_JSON, WIRE  Checks that reserved ranges and names are not deleted from a given enum.
		RESERVED_MESSAGE_NO_DELETE                   FILE, PACKAGE, WIRE_JSON, WIRE  Checks that reserved ranges and names are not deleted from a given message.
		RPC_SAME_CLIENT_STREAMING                    FILE, PACKAGE, WIRE_JSON, WIRE  Checks that rpcs have the same client streaming value.
		RPC_SAME_IDEMPOTENCY_LEVEL                   FILE, PACKAGE, WIRE_JSON, WIRE  Checks that rpcs have the same value for the idempotency_level option.
		RPC_SAME_REQUEST_TYPE                        FILE, PACKAGE, WIRE_JSON, WIRE  Checks that rpcs are have the same request type.
		RPC_SAME_RESPONSE_TYPE                       FILE, PACKAGE, WIRE_JSON, WIRE  Checks that rpcs are have the same response type.
		RPC_SAME_SERVER_STREAMING                    FILE, PACKAGE, WIRE_JSON, WIRE  Checks that rpcs have the same server streaming value.
		ENUM_VALUE_NO_DELETE_UNLESS_NAME_RESERVED    WIRE_JSON                       Checks that enum values are not deleted from a given enum unless the name is reserved.
		FIELD_NO_DELETE_UNLESS_NAME_RESERVED         WIRE_JSON                       Checks that fields are not deleted from a given message unless the name is reserved.
		ENUM_VALUE_NO_DELETE_UNLESS_NUMBER_RESERVED  WIRE_JSON, WIRE                 Checks that enum values are not deleted from a given enum unless the number is reserved.
		FIELD_NO_DELETE_UNLESS_NUMBER_RESERVED       WIRE_JSON, WIRE                 Checks that fields are not deleted from a given message unless the number is reserved.
		`,
		"check",
		"ls-breaking-checkers",
		"--all",
		"--category",
		"WIRE_JSON",
	)
}

func TestCheckLsBreakingCheckers2(t *testing.T) {
	testRun(
		t,
		0,
		`
		ID                    CATEGORIES     PURPOSE
		ENUM_VALUE_NO_DELETE  FILE, PACKAGE  Checks that enum values are not deleted from a given enum.
		FIELD_SAME_JSTYPE     FILE, PACKAGE  Checks that fields have the same value for the jstype option.
		`,
		"check",
		"ls-breaking-checkers",
		"--config",
		filepath.Join("testdata", "small_list_checkers", "buf.yaml"),
	)
}

func TestLsFiles(t *testing.T) {
	testRun(
		t,
		0,
		`
		testdata/success/buf/buf.proto
		`,
		"ls-files",
		"--input",
		filepath.Join("testdata", "success"),
	)
}

func testRun(t *testing.T, expectedExitCode int, expectedStdout string, args ...string) {
	t.Parallel()
	t.Run("buf", func(t *testing.T) {
		testRunCmd(
			t,
			newRootCommand("test", false),
			expectedExitCode,
			expectedStdout,
			args...,
		)
	})
	t.Run("bufdev", func(t *testing.T) {
		testRunCmd(
			t,
			newRootCommand("test", true),
			expectedExitCode,
			expectedStdout,
			args...,
		)
	})
}

func testRunProfile(t *testing.T, expectedExitCode int, expectedStdout string, args ...string) {
	t.Run("bufdev-profile", func(t *testing.T) {
		profileDirPath, err := ioutil.TempDir("", "")
		require.NoError(t, err)
		defer func() { assert.NoError(t, os.RemoveAll(profileDirPath)) }()
		testRunCmd(
			t,
			newRootCommand("test", true),
			0,
			``,
			append(
				args,
				"--profile",
				fmt.Sprintf("--profile-path=%s", profileDirPath),
				"--profile-loops=1",
				"--profile-type=cpu",
			)...,
		)
	})
}

func testRunCmd(t *testing.T, cmd *clicobra.Command, expectedExitCode int, expectedStdout string, args ...string) {
	t.Parallel()
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	exitCode := clicobra.Run(
		cmd,
		"test",
		&cli.RunEnv{
			Args:   args,
			Stdout: stdout,
			Stderr: stderr,
		},
	)
	assert.Equal(t, expectedExitCode, exitCode, stringutil.TrimLines(stderr.String()))
	if exitCode == expectedExitCode {
		assert.Equal(t, stringutil.TrimLines(expectedStdout), stringutil.TrimLines(stdout.String()), stringutil.TrimLines(stderr.String()))
	}
}

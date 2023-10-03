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

package buf

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/cmd/buf/internal/internaltesting"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appcmd/appcmdtesting"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/storage/storagetesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var convertTestDataDir = filepath.Join("command", "convert", "testdata", "convert")

func TestSuccess1(t *testing.T) {
	t.Parallel()
	testRunStdout(t, nil, 1, ``, "build", "--source", filepath.Join("testdata", "success"))
	testRunStdout(t, nil, 0, ``, "build", filepath.Join("testdata", "success"))
}

func TestSuccess2(t *testing.T) {
	t.Parallel()
	testRunStdout(t, nil, 1, ``, "build", "--exclude-imports", "--source", filepath.Join("testdata", "success"))
	testRunStdout(t, nil, 0, ``, "build", "--exclude-imports", filepath.Join("testdata", "success"))
}

func TestSuccess3(t *testing.T) {
	t.Parallel()
	testRunStdout(t, nil, 1, ``, "build", "--exclude-source-info", "--source", filepath.Join("testdata", "success"))
	testRunStdout(t, nil, 0, ``, "build", "--exclude-source-info", filepath.Join("testdata", "success"))
}

func TestSuccess4(t *testing.T) {
	t.Parallel()
	testRunStdout(t, nil, 1, ``, "build", "--exclude-imports", "--exclude-source-info", "--source", filepath.Join("testdata", "success"))
	testRunStdout(t, nil, 0, ``, "build", "--exclude-imports", "--exclude-source-info", filepath.Join("testdata", "success"))
}

func TestSuccess5(t *testing.T) {
	t.Parallel()
	testRunStdout(t, nil, 1, ``, "build", "--exclude-imports", "--exclude-source-info", "--source", filepath.Join("testdata", "success"))
	testRunStdout(t, nil, 0, ``, "build", "--exclude-imports", "--exclude-source-info", filepath.Join("testdata", "success"))
}

func TestSuccess6(t *testing.T) {
	t.Parallel()
	testRunStdout(t, nil, 0, ``, "lint", filepath.Join("testdata", "success"))
	testRunStdout(t, nil, 1, ``, "lint", "--input", filepath.Join("testdata", "success", "buf", "buf.proto"))
	testRunStdout(t, nil, 0, ``, "lint", filepath.Join("testdata", "success", "buf", "buf.proto"))
}

func TestSuccessProfile1(t *testing.T) {
	t.Parallel()
	testRunStdoutProfile(t, nil, 0, ``, "build", filepath.Join("testdata", "success"))
}

func TestFail1(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		1,
		``,
		"build",
		"--source",
		filepath.Join("testdata", "fail"),
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("testdata", "fail"),
	)
}

func TestFail2(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		1,
		``,
		"build",
		"--exclude-imports",
		"--source",
		filepath.Join("testdata", "fail"),
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		"--exclude-imports",
		filepath.Join("testdata", "fail"),
	)
}

func TestFail3(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		1,
		``,
		"build",
		"--exclude-source-info",
		"--source",
		filepath.Join("testdata", "fail"),
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		"--exclude-source-info",
		filepath.Join("testdata", "fail"),
	)
}

func TestFail4(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		1,
		``,
		"build",
		"--exclude-imports",
		"--exclude-source-info",
		"--source",
		filepath.Join("testdata", "fail"),
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		"--exclude-imports",
		"--exclude-source-info",
		filepath.Join("testdata", "fail"),
	)
}

func TestFail5(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/fail/buf/buf.proto:3:1:Files with package "other" must be within a directory "other" relative to root but were in directory "buf".
        testdata/fail/buf/buf.proto:6:9:Field name "oneTwo" should be lower_snake_case, such as "one_two".`),
		"lint",
		filepath.Join("testdata", "fail"),
	)
	testRunStdout(
		t,
		nil,
		1,
		``,
		"lint",
		"--input",
		filepath.Join("testdata", "fail", "buf", "buf.proto"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/fail/buf/buf.proto:3:1:Files with package "other" must be within a directory "other" relative to root but were in directory "buf".
        testdata/fail/buf/buf.proto:6:9:Field name "oneTwo" should be lower_snake_case, such as "one_two".`),
		"lint",
		filepath.Join("testdata", "fail", "buf", "buf.proto"),
	)
}

func TestFail6(t *testing.T) {
	t.Parallel()
	testRunStdoutStderrNoWarn(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/fail/buf/buf.proto:3:1:Files with package "other" must be within a directory "other" relative to root but were in directory "buf".
        testdata/fail/buf/buf.proto:6:9:Field name "oneTwo" should be lower_snake_case, such as "one_two".`),
		"", // stderr should be empty
		"lint",
		filepath.Join("testdata", "fail"),
		"--path",
		filepath.Join("testdata", "fail", "buf", "buf.proto"),
	)
	testRunStdout(
		t,
		nil,
		1,
		``,
		"lint",
		"--input",
		filepath.Join("testdata", "fail", "buf", "buf.proto"),
		"--path",
		filepath.Join("testdata", "fail", "buf", "buf.proto"),
	)
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		"", // stdout should be empty
		filepath.FromSlash(`Failure: path "." is not contained within any of roots "." - note that specified paths cannot be roots, but must be contained within roots`),
		"lint",
		filepath.Join("testdata", "fail", "buf", "buf.proto"),
		"--path",
		filepath.Join("testdata", "fail", "buf", "buf.proto"),
	)
}

func TestFail7(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		1,
		``,
		"lint",
		filepath.Join("testdata", "fail", "buf", "buf.proto"),
		"--input-config",
		`{"version":"v1","lint":{"use":["BASIC"]}}`,
	)
	testRunStdoutStderrNoWarn(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/fail/buf/buf.proto:3:1:Files with package "other" must be within a directory "other" relative to root but were in directory "fail/buf".
testdata/fail/buf/buf.proto:6:9:Field name "oneTwo" should be lower_snake_case, such as "one_two".`),
		"", // stderr should be empty
		"lint",
		"--path",
		filepath.Join("testdata", "fail", "buf", "buf.proto"),
		filepath.Join("testdata"),
		"--config",
		`{"version":"v1beta1","lint":{"use":["BASIC"]}}`,
	)
	testRunStdoutStderrNoWarn(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/fail/buf/buf.proto:3:1:Files with package "other" must be within a directory "other" relative to root but were in directory "buf".
testdata/fail/buf/buf.proto:6:9:Field name "oneTwo" should be lower_snake_case, such as "one_two".`),
		"", // stderr should be empty
		"lint",
		filepath.Join("testdata", "fail", "buf", "buf.proto"),
		"--config",
		`{"version":"v1","lint":{"use":["BASIC"]}}`,
	)
}

func TestFail8(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/fail2/buf/buf.proto:6:9:Field name "oneTwo" should be lower_snake_case, such as "one_two".
		testdata/fail2/buf/buf2.proto:9:9:Field name "oneThree" should be lower_snake_case, such as "one_three".`),
		"lint",
		filepath.Join("testdata", "fail2"),
	)
}

func TestFail9(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/fail2/buf/buf.proto:6:9:Field name "oneTwo" should be lower_snake_case, such as "one_two".`),
		"lint",
		filepath.Join("testdata", "fail2"),
		"--path",
		filepath.Join("testdata", "fail2", "buf", "buf.proto"),
	)
}

func TestFail10(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"lint",
		filepath.Join("testdata", "fail2"),
		"--path",
		filepath.Join("testdata", "fail2", "buf", "buf3.proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		"",
		"lint",
		filepath.Join("testdata", "fail2", "buf", "buf3.proto"),
	)
}

func TestFail11(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		1,
		``,
		"lint",
		"--path",
		filepath.Join("testdata", "fail2", "buf", "buf2.proto"),
		"--input",
		filepath.Join("testdata"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		fmt.Sprintf("%v:5:8:buf/buf.proto: does not exist", filepath.FromSlash("testdata/fail2/buf/buf2.proto")),
		"lint",
		"--path",
		filepath.Join("testdata", "fail2", "buf", "buf2.proto"),
		filepath.Join("testdata"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/fail2/buf/buf2.proto:9:9:Field name "oneThree" should be lower_snake_case, such as "one_three".`),
		"lint",
		filepath.Join("testdata", "fail2", "buf", "buf2.proto"),
	)
}

func TestFail12(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		`version: v1
lint:
  ignore_only:
    FIELD_LOWER_SNAKE_CASE:
	  - buf/buf.proto
	PACKAGE_DIRECTORY_MATCH:
	  - buf/buf.proto`,
		"lint",
		filepath.Join("testdata", "fail"),
		"--error-format",
		"config-ignore-yaml",
	)
}

func TestFail13(t *testing.T) {
	t.Parallel()
	// this tests that we still use buf.mod if it exists
	// this has an ignore for FIELD_LOWER_SNAKE_CASE to make sure we are actually reading the config
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/fail_buf_mod/buf/buf.proto:3:1:Files with package "other" must be within a directory "other" relative to root but were in directory "buf".`),
		"lint",
		filepath.Join("testdata", "fail_buf_mod"),
	)
}

func TestFailCheckBreaking1(t *testing.T) {
	t.Parallel()
	testRunStdoutStderrNoWarn(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`
		../../../bufpkg/bufcheck/bufbreaking/testdata/breaking_field_no_delete/1.proto:5:1:Previously present field "3" with name "three" on message "Two" was deleted.
		../../../bufpkg/bufcheck/bufbreaking/testdata/breaking_field_no_delete/1.proto:10:1:Previously present field "3" with name "three" on message "Three" was deleted.
		../../../bufpkg/bufcheck/bufbreaking/testdata/breaking_field_no_delete/1.proto:12:5:Previously present field "3" with name "three" on message "Five" was deleted.
		../../../bufpkg/bufcheck/bufbreaking/testdata/breaking_field_no_delete/1.proto:22:3:Previously present field "3" with name "three" on message "Seven" was deleted.
		../../../bufpkg/bufcheck/bufbreaking/testdata/breaking_field_no_delete/2.proto:57:1:Previously present field "3" with name "three" on message "Nine" was deleted.
		`),
		"", // stderr should be empty
		"breaking",
		// can't bother right now to filepath.Join this
		"../../../bufpkg/bufcheck/bufbreaking/testdata/breaking_field_no_delete",
		"--against",
		"../../../bufpkg/bufcheck/bufbreaking/testdata_previous/breaking_field_no_delete",
	)
}

func TestFailCheckBreaking2(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/protofileref/breaking/a/foo.proto:7:3:Field "2" on message "Foo" changed type from "int32" to "string".`),
		"breaking",
		filepath.Join("testdata", "protofileref", "breaking", "a", "foo.proto"),
		"--against",
		filepath.Join("testdata", "protofileref", "breaking", "b", "foo.proto"),
	)
}

func TestFailCheckBreaking3(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`
		<input>:1:1:Previously present file "bar.proto" was deleted.
		testdata/protofileref/breaking/a/foo.proto:7:3:Field "2" on message "Foo" changed type from "int32" to "string".
		`),
		"breaking",
		filepath.Join("testdata", "protofileref", "breaking", "a", "foo.proto"),
		"--against",
		filepath.Join("testdata", "protofileref", "breaking", "b"),
	)
}

func TestFailCheckBreaking4(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`
		testdata/protofileref/breaking/a/bar.proto:5:1:Previously present field "2" with name "value" on message "Bar" was deleted.
		testdata/protofileref/breaking/a/foo.proto:7:3:Field "2" on message "Foo" changed type from "int32" to "string".
		`),
		"breaking",
		fmt.Sprintf("%s#include_package_files=true", filepath.Join("testdata", "protofileref", "breaking", "a", "foo.proto")),
		"--against",
		filepath.Join("testdata", "protofileref", "breaking", "b"),
	)
}

func TestFailCheckBreaking5(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`
    <input>:1:1:Previously present file "bar.proto" was deleted.
		testdata/protofileref/breaking/a/foo.proto:7:3:Field "2" on message "Foo" changed type from "int32" to "string".
		`),
		"breaking",
		filepath.Join("testdata", "protofileref", "breaking", "a", "foo.proto"),
		"--against",
		fmt.Sprintf("%s#include_package_files=true", filepath.Join("testdata", "protofileref", "breaking", "b", "foo.proto")),
	)
}

func TestCheckLsLintRules1(t *testing.T) {
	t.Parallel()
	expectedStdout := `
ID                                CATEGORIES               PURPOSE
DIRECTORY_SAME_PACKAGE            MINIMAL, BASIC, DEFAULT  Checks that all files in a given directory are in the same package.
PACKAGE_DEFINED                   MINIMAL, BASIC, DEFAULT  Checks that all files have a package defined.
PACKAGE_DIRECTORY_MATCH           MINIMAL, BASIC, DEFAULT  Checks that all files are in a directory that matches their package name.
PACKAGE_SAME_DIRECTORY            MINIMAL, BASIC, DEFAULT  Checks that all files with a given package are in the same directory.
ENUM_FIRST_VALUE_ZERO             BASIC, DEFAULT           Checks that all first values of enums have a numeric value of 0.
ENUM_NO_ALLOW_ALIAS               BASIC, DEFAULT           Checks that enums do not have the allow_alias option set.
ENUM_PASCAL_CASE                  BASIC, DEFAULT           Checks that enums are PascalCase.
ENUM_VALUE_UPPER_SNAKE_CASE       BASIC, DEFAULT           Checks that enum values are UPPER_SNAKE_CASE.
FIELD_LOWER_SNAKE_CASE            BASIC, DEFAULT           Checks that field names are lower_snake_case.
IMPORT_NO_PUBLIC                  BASIC, DEFAULT           Checks that imports are not public.
IMPORT_NO_WEAK                    BASIC, DEFAULT           Checks that imports are not weak.
IMPORT_USED                       BASIC, DEFAULT           Checks that imports are used.
MESSAGE_PASCAL_CASE               BASIC, DEFAULT           Checks that messages are PascalCase.
ONEOF_LOWER_SNAKE_CASE            BASIC, DEFAULT           Checks that oneof names are lower_snake_case.
PACKAGE_LOWER_SNAKE_CASE          BASIC, DEFAULT           Checks that packages are lower_snake.case.
PACKAGE_SAME_CSHARP_NAMESPACE     BASIC, DEFAULT           Checks that all files with a given package have the same value for the csharp_namespace option.
PACKAGE_SAME_GO_PACKAGE           BASIC, DEFAULT           Checks that all files with a given package have the same value for the go_package option.
PACKAGE_SAME_JAVA_MULTIPLE_FILES  BASIC, DEFAULT           Checks that all files with a given package have the same value for the java_multiple_files option.
PACKAGE_SAME_JAVA_PACKAGE         BASIC, DEFAULT           Checks that all files with a given package have the same value for the java_package option.
PACKAGE_SAME_PHP_NAMESPACE        BASIC, DEFAULT           Checks that all files with a given package have the same value for the php_namespace option.
PACKAGE_SAME_RUBY_PACKAGE         BASIC, DEFAULT           Checks that all files with a given package have the same value for the ruby_package option.
PACKAGE_SAME_SWIFT_PREFIX         BASIC, DEFAULT           Checks that all files with a given package have the same value for the swift_prefix option.
RPC_PASCAL_CASE                   BASIC, DEFAULT           Checks that RPCs are PascalCase.
SERVICE_PASCAL_CASE               BASIC, DEFAULT           Checks that services are PascalCase.
SYNTAX_SPECIFIED                  BASIC, DEFAULT           Checks that all files have a syntax specified.
ENUM_VALUE_PREFIX                 DEFAULT                  Checks that enum values are prefixed with ENUM_NAME_UPPER_SNAKE_CASE.
ENUM_ZERO_VALUE_SUFFIX            DEFAULT                  Checks that enum zero values are suffixed with _UNSPECIFIED (suffix is configurable).
FILE_LOWER_SNAKE_CASE             DEFAULT                  Checks that filenames are lower_snake_case.
PACKAGE_VERSION_SUFFIX            DEFAULT                  Checks that the last component of all packages is a version of the form v\d+, v\d+test.*, v\d+(alpha|beta)\d+, or v\d+p\d+(alpha|beta)\d+, where numbers are >=1.
PROTOVALIDATE_CEL                 DEFAULT                  Checks that protovalidate CEL expressions compile.
RPC_REQUEST_RESPONSE_UNIQUE       DEFAULT                  Checks that RPC request and response types are only used in one RPC (configurable).
RPC_REQUEST_STANDARD_NAME         DEFAULT                  Checks that RPC request type names are RPCNameRequest or ServiceNameRPCNameRequest (configurable).
RPC_RESPONSE_STANDARD_NAME        DEFAULT                  Checks that RPC response type names are RPCNameResponse or ServiceNameRPCNameResponse (configurable).
SERVICE_SUFFIX                    DEFAULT                  Checks that services are suffixed with Service (suffix is configurable).
COMMENT_ENUM                      COMMENTS                 Checks that enums have non-empty comments.
COMMENT_ENUM_VALUE                COMMENTS                 Checks that enum values have non-empty comments.
COMMENT_FIELD                     COMMENTS                 Checks that fields have non-empty comments.
COMMENT_MESSAGE                   COMMENTS                 Checks that messages have non-empty comments.
COMMENT_ONEOF                     COMMENTS                 Checks that oneof have non-empty comments.
COMMENT_RPC                       COMMENTS                 Checks that RPCs have non-empty comments.
COMMENT_SERVICE                   COMMENTS                 Checks that services have non-empty comments.
RPC_NO_CLIENT_STREAMING           UNARY_RPC                Checks that RPCs are not client streaming.
RPC_NO_SERVER_STREAMING           UNARY_RPC                Checks that RPCs are not server streaming.
PACKAGE_NO_IMPORT_CYCLE                                    Checks that packages do not have import cycles.
		`
	testRunStdout(
		t,
		nil,
		0,
		expectedStdout,
		"mod",
		"ls-lint-rules",
		"--all",
	)
}

func TestCheckLsLintRules2(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		0,
		`
		ID                       CATEGORIES                            PURPOSE
		PACKAGE_DIRECTORY_MATCH  MINIMAL, BASIC, DEFAULT, FILE_LAYOUT  Checks that all files are in a directory that matches their package name.
		ENUM_NO_ALLOW_ALIAS      MINIMAL, BASIC, DEFAULT, SENSIBLE     Checks that enums do not have the allow_alias option set.
		`,
		"mod",
		"ls-lint-rules",
		"--config",
		filepath.Join("testdata", "small_list_rules", bufconfig.ExternalConfigV1FilePath),
	)
}

func TestCheckLsLintRules3(t *testing.T) {
	t.Parallel()
	expectedStdout := `
ID                                CATEGORIES                                  PURPOSE
DIRECTORY_SAME_PACKAGE            MINIMAL, BASIC, DEFAULT, FILE_LAYOUT        Checks that all files in a given directory are in the same package.
PACKAGE_DIRECTORY_MATCH           MINIMAL, BASIC, DEFAULT, FILE_LAYOUT        Checks that all files are in a directory that matches their package name.
PACKAGE_SAME_DIRECTORY            MINIMAL, BASIC, DEFAULT, FILE_LAYOUT        Checks that all files with a given package are in the same directory.
PACKAGE_SAME_CSHARP_NAMESPACE     MINIMAL, BASIC, DEFAULT, PACKAGE_AFFINITY   Checks that all files with a given package have the same value for the csharp_namespace option.
PACKAGE_SAME_GO_PACKAGE           MINIMAL, BASIC, DEFAULT, PACKAGE_AFFINITY   Checks that all files with a given package have the same value for the go_package option.
PACKAGE_SAME_JAVA_MULTIPLE_FILES  MINIMAL, BASIC, DEFAULT, PACKAGE_AFFINITY   Checks that all files with a given package have the same value for the java_multiple_files option.
PACKAGE_SAME_JAVA_PACKAGE         MINIMAL, BASIC, DEFAULT, PACKAGE_AFFINITY   Checks that all files with a given package have the same value for the java_package option.
PACKAGE_SAME_PHP_NAMESPACE        MINIMAL, BASIC, DEFAULT, PACKAGE_AFFINITY   Checks that all files with a given package have the same value for the php_namespace option.
PACKAGE_SAME_RUBY_PACKAGE         MINIMAL, BASIC, DEFAULT, PACKAGE_AFFINITY   Checks that all files with a given package have the same value for the ruby_package option.
PACKAGE_SAME_SWIFT_PREFIX         MINIMAL, BASIC, DEFAULT, PACKAGE_AFFINITY   Checks that all files with a given package have the same value for the swift_prefix option.
ENUM_NO_ALLOW_ALIAS               MINIMAL, BASIC, DEFAULT, SENSIBLE           Checks that enums do not have the allow_alias option set.
FIELD_NO_DESCRIPTOR               MINIMAL, BASIC, DEFAULT, SENSIBLE           Checks that field names are not name capitalization of "descriptor" with any number of prefix or suffix underscores.
IMPORT_NO_PUBLIC                  MINIMAL, BASIC, DEFAULT, SENSIBLE           Checks that imports are not public.
IMPORT_NO_WEAK                    MINIMAL, BASIC, DEFAULT, SENSIBLE           Checks that imports are not weak.
PACKAGE_DEFINED                   MINIMAL, BASIC, DEFAULT, SENSIBLE           Checks that all files have a package defined.
ENUM_PASCAL_CASE                  BASIC, DEFAULT, STYLE_BASIC, STYLE_DEFAULT  Checks that enums are PascalCase.
ENUM_VALUE_UPPER_SNAKE_CASE       BASIC, DEFAULT, STYLE_BASIC, STYLE_DEFAULT  Checks that enum values are UPPER_SNAKE_CASE.
FIELD_LOWER_SNAKE_CASE            BASIC, DEFAULT, STYLE_BASIC, STYLE_DEFAULT  Checks that field names are lower_snake_case.
MESSAGE_PASCAL_CASE               BASIC, DEFAULT, STYLE_BASIC, STYLE_DEFAULT  Checks that messages are PascalCase.
ONEOF_LOWER_SNAKE_CASE            BASIC, DEFAULT, STYLE_BASIC, STYLE_DEFAULT  Checks that oneof names are lower_snake_case.
PACKAGE_LOWER_SNAKE_CASE          BASIC, DEFAULT, STYLE_BASIC, STYLE_DEFAULT  Checks that packages are lower_snake.case.
RPC_PASCAL_CASE                   BASIC, DEFAULT, STYLE_BASIC, STYLE_DEFAULT  Checks that RPCs are PascalCase.
SERVICE_PASCAL_CASE               BASIC, DEFAULT, STYLE_BASIC, STYLE_DEFAULT  Checks that services are PascalCase.
ENUM_VALUE_PREFIX                 DEFAULT, STYLE_DEFAULT                      Checks that enum values are prefixed with ENUM_NAME_UPPER_SNAKE_CASE.
ENUM_ZERO_VALUE_SUFFIX            DEFAULT, STYLE_DEFAULT                      Checks that enum zero values are suffixed with _UNSPECIFIED (suffix is configurable).
FILE_LOWER_SNAKE_CASE             DEFAULT, STYLE_DEFAULT                      Checks that filenames are lower_snake_case.
PACKAGE_VERSION_SUFFIX            DEFAULT, STYLE_DEFAULT                      Checks that the last component of all packages is a version of the form v\d+, v\d+test.*, v\d+(alpha|beta)\d+, or v\d+p\d+(alpha|beta)\d+, where numbers are >=1.
RPC_REQUEST_RESPONSE_UNIQUE       DEFAULT, STYLE_DEFAULT                      Checks that RPC request and response types are only used in one RPC (configurable).
RPC_REQUEST_STANDARD_NAME         DEFAULT, STYLE_DEFAULT                      Checks that RPC request type names are RPCNameRequest or ServiceNameRPCNameRequest (configurable).
RPC_RESPONSE_STANDARD_NAME        DEFAULT, STYLE_DEFAULT                      Checks that RPC response type names are RPCNameResponse or ServiceNameRPCNameResponse (configurable).
SERVICE_SUFFIX                    DEFAULT, STYLE_DEFAULT                      Checks that services are suffixed with Service (suffix is configurable).
COMMENT_ENUM                      COMMENTS                                    Checks that enums have non-empty comments.
COMMENT_ENUM_VALUE                COMMENTS                                    Checks that enum values have non-empty comments.
COMMENT_FIELD                     COMMENTS                                    Checks that fields have non-empty comments.
COMMENT_MESSAGE                   COMMENTS                                    Checks that messages have non-empty comments.
COMMENT_ONEOF                     COMMENTS                                    Checks that oneof have non-empty comments.
COMMENT_RPC                       COMMENTS                                    Checks that RPCs have non-empty comments.
COMMENT_SERVICE                   COMMENTS                                    Checks that services have non-empty comments.
RPC_NO_CLIENT_STREAMING           UNARY_RPC                                   Checks that RPCs are not client streaming.
RPC_NO_SERVER_STREAMING           UNARY_RPC                                   Checks that RPCs are not server streaming.
ENUM_FIRST_VALUE_ZERO             OTHER                                       Checks that all first values of enums have a numeric value of 0.
		`
	testRunStdout(
		t,
		nil,
		0,
		expectedStdout,
		"mod",
		"ls-lint-rules",
		"--version",
		"v1beta1",
	)
}

func TestCheckLsBreakingRules1(t *testing.T) {
	t.Parallel()
	expectedStdout := `
ID                                              CATEGORIES                      PURPOSE
ENUM_NO_DELETE                                  FILE                            Checks that enums are not deleted from a given file.
FILE_NO_DELETE                                  FILE                            Checks that files are not deleted.
MESSAGE_NO_DELETE                               FILE                            Checks that messages are not deleted from a given file.
SERVICE_NO_DELETE                               FILE                            Checks that services are not deleted from a given file.
ENUM_VALUE_NO_DELETE                            FILE, PACKAGE                   Checks that enum values are not deleted from a given enum.
EXTENSION_MESSAGE_NO_DELETE                     FILE, PACKAGE                   Checks that extension ranges are not deleted from a given message.
FIELD_NO_DELETE                                 FILE, PACKAGE                   Checks that fields are not deleted from a given message.
FIELD_SAME_CTYPE                                FILE, PACKAGE                   Checks that fields have the same value for the ctype option.
FIELD_SAME_JSTYPE                               FILE, PACKAGE                   Checks that fields have the same value for the jstype option.
FIELD_SAME_TYPE                                 FILE, PACKAGE                   Checks that fields have the same types in a given message.
FILE_SAME_CC_ENABLE_ARENAS                      FILE, PACKAGE                   Checks that files have the same value for the cc_enable_arenas option.
FILE_SAME_CC_GENERIC_SERVICES                   FILE, PACKAGE                   Checks that files have the same value for the cc_generic_services option.
FILE_SAME_CSHARP_NAMESPACE                      FILE, PACKAGE                   Checks that files have the same value for the csharp_namespace option.
FILE_SAME_GO_PACKAGE                            FILE, PACKAGE                   Checks that files have the same value for the go_package option.
FILE_SAME_JAVA_GENERIC_SERVICES                 FILE, PACKAGE                   Checks that files have the same value for the java_generic_services option.
FILE_SAME_JAVA_MULTIPLE_FILES                   FILE, PACKAGE                   Checks that files have the same value for the java_multiple_files option.
FILE_SAME_JAVA_OUTER_CLASSNAME                  FILE, PACKAGE                   Checks that files have the same value for the java_outer_classname option.
FILE_SAME_JAVA_PACKAGE                          FILE, PACKAGE                   Checks that files have the same value for the java_package option.
FILE_SAME_JAVA_STRING_CHECK_UTF8                FILE, PACKAGE                   Checks that files have the same value for the java_string_check_utf8 option.
FILE_SAME_OBJC_CLASS_PREFIX                     FILE, PACKAGE                   Checks that files have the same value for the objc_class_prefix option.
FILE_SAME_OPTIMIZE_FOR                          FILE, PACKAGE                   Checks that files have the same value for the optimize_for option.
FILE_SAME_PHP_CLASS_PREFIX                      FILE, PACKAGE                   Checks that files have the same value for the php_class_prefix option.
FILE_SAME_PHP_GENERIC_SERVICES                  FILE, PACKAGE                   Checks that files have the same value for the php_generic_services option.
FILE_SAME_PHP_METADATA_NAMESPACE                FILE, PACKAGE                   Checks that files have the same value for the php_metadata_namespace option.
FILE_SAME_PHP_NAMESPACE                         FILE, PACKAGE                   Checks that files have the same value for the php_namespace option.
FILE_SAME_PY_GENERIC_SERVICES                   FILE, PACKAGE                   Checks that files have the same value for the py_generic_services option.
FILE_SAME_RUBY_PACKAGE                          FILE, PACKAGE                   Checks that files have the same value for the ruby_package option.
FILE_SAME_SWIFT_PREFIX                          FILE, PACKAGE                   Checks that files have the same value for the swift_prefix option.
FILE_SAME_SYNTAX                                FILE, PACKAGE                   Checks that files have the same syntax.
MESSAGE_NO_REMOVE_STANDARD_DESCRIPTOR_ACCESSOR  FILE, PACKAGE                   Checks that messages do not change the no_standard_descriptor_accessor option from false or unset to true.
ONEOF_NO_DELETE                                 FILE, PACKAGE                   Checks that oneofs are not deleted from a given message.
RPC_NO_DELETE                                   FILE, PACKAGE                   Checks that rpcs are not deleted from a given service.
ENUM_VALUE_SAME_NAME                            FILE, PACKAGE, WIRE_JSON        Checks that enum values have the same name.
FIELD_SAME_JSON_NAME                            FILE, PACKAGE, WIRE_JSON        Checks that fields have the same value for the json_name option.
FIELD_SAME_NAME                                 FILE, PACKAGE, WIRE_JSON        Checks that fields have the same names in a given message.
FIELD_SAME_LABEL                                FILE, PACKAGE, WIRE_JSON, WIRE  Checks that fields have the same labels in a given message.
FIELD_SAME_ONEOF                                FILE, PACKAGE, WIRE_JSON, WIRE  Checks that fields have the same oneofs in a given message.
FILE_SAME_PACKAGE                               FILE, PACKAGE, WIRE_JSON, WIRE  Checks that files have the same package.
MESSAGE_SAME_MESSAGE_SET_WIRE_FORMAT            FILE, PACKAGE, WIRE_JSON, WIRE  Checks that messages have the same value for the message_set_wire_format option.
MESSAGE_SAME_REQUIRED_FIELDS                    FILE, PACKAGE, WIRE_JSON, WIRE  Checks that messages have no added or deleted required fields.
RESERVED_ENUM_NO_DELETE                         FILE, PACKAGE, WIRE_JSON, WIRE  Checks that reserved ranges and names are not deleted from a given enum.
RESERVED_MESSAGE_NO_DELETE                      FILE, PACKAGE, WIRE_JSON, WIRE  Checks that reserved ranges and names are not deleted from a given message.
RPC_SAME_CLIENT_STREAMING                       FILE, PACKAGE, WIRE_JSON, WIRE  Checks that rpcs have the same client streaming value.
RPC_SAME_IDEMPOTENCY_LEVEL                      FILE, PACKAGE, WIRE_JSON, WIRE  Checks that rpcs have the same value for the idempotency_level option.
RPC_SAME_REQUEST_TYPE                           FILE, PACKAGE, WIRE_JSON, WIRE  Checks that rpcs are have the same request type.
RPC_SAME_RESPONSE_TYPE                          FILE, PACKAGE, WIRE_JSON, WIRE  Checks that rpcs are have the same response type.
RPC_SAME_SERVER_STREAMING                       FILE, PACKAGE, WIRE_JSON, WIRE  Checks that rpcs have the same server streaming value.
PACKAGE_ENUM_NO_DELETE                          PACKAGE                         Checks that enums are not deleted from a given package.
PACKAGE_MESSAGE_NO_DELETE                       PACKAGE                         Checks that messages are not deleted from a given package.
PACKAGE_NO_DELETE                               PACKAGE                         Checks that packages are not deleted.
PACKAGE_SERVICE_NO_DELETE                       PACKAGE                         Checks that services are not deleted from a given package.
ENUM_VALUE_NO_DELETE_UNLESS_NAME_RESERVED       WIRE_JSON                       Checks that enum values are not deleted from a given enum unless the name is reserved.
FIELD_NO_DELETE_UNLESS_NAME_RESERVED            WIRE_JSON                       Checks that fields are not deleted from a given message unless the name is reserved.
FIELD_WIRE_JSON_COMPATIBLE_TYPE                 WIRE_JSON                       Checks that fields have wire and JSON compatible types in a given message.
ENUM_VALUE_NO_DELETE_UNLESS_NUMBER_RESERVED     WIRE_JSON, WIRE                 Checks that enum values are not deleted from a given enum unless the number is reserved.
FIELD_NO_DELETE_UNLESS_NUMBER_RESERVED          WIRE_JSON, WIRE                 Checks that fields are not deleted from a given message unless the number is reserved.
FIELD_WIRE_COMPATIBLE_TYPE                      WIRE                            Checks that fields have wire-compatible types in a given message.
		`
	testRunStdout(
		t,
		nil,
		0,
		expectedStdout,
		"mod",
		"ls-breaking-rules",
		"--version",
		"v1",
	)
}

func TestCheckLsBreakingRules2(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		0,
		`
		ID                    CATEGORIES     PURPOSE
		ENUM_VALUE_NO_DELETE  FILE, PACKAGE  Checks that enum values are not deleted from a given enum.
		FIELD_SAME_JSTYPE     FILE, PACKAGE  Checks that fields have the same value for the jstype option.
		`,
		"mod",
		"ls-breaking-rules",
		"--config",
		filepath.Join("testdata", "small_list_rules", bufconfig.ExternalConfigV1FilePath),
	)
}

func TestCheckLsBreakingRules3(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		0,
		`
		ID                    CATEGORIES     PURPOSE
		ENUM_VALUE_NO_DELETE  FILE, PACKAGE  Checks that enum values are not deleted from a given enum.
		FIELD_SAME_JSTYPE     FILE, PACKAGE  Checks that fields have the same value for the jstype option.
		`,
		"mod",
		"ls-breaking-rules",
		"--config",
		// making sure that .yml works
		filepath.Join("testdata", "small_list_rules_yml", "config.yml"),
	)
}

func TestCheckLsBreakingRules4(t *testing.T) {
	t.Parallel()
	expectedStdout := `
ID                                              CATEGORIES                      PURPOSE
ENUM_NO_DELETE                                  FILE                            Checks that enums are not deleted from a given file.
FILE_NO_DELETE                                  FILE                            Checks that files are not deleted.
FILE_SAME_PACKAGE                               FILE                            Checks that files have the same package.
MESSAGE_NO_DELETE                               FILE                            Checks that messages are not deleted from a given file.
SERVICE_NO_DELETE                               FILE                            Checks that services are not deleted from a given file.
ENUM_VALUE_NO_DELETE                            FILE, PACKAGE                   Checks that enum values are not deleted from a given enum.
EXTENSION_MESSAGE_NO_DELETE                     FILE, PACKAGE                   Checks that extension ranges are not deleted from a given message.
FIELD_NO_DELETE                                 FILE, PACKAGE                   Checks that fields are not deleted from a given message.
FIELD_SAME_CTYPE                                FILE, PACKAGE                   Checks that fields have the same value for the ctype option.
FIELD_SAME_JSTYPE                               FILE, PACKAGE                   Checks that fields have the same value for the jstype option.
FILE_SAME_CC_ENABLE_ARENAS                      FILE, PACKAGE                   Checks that files have the same value for the cc_enable_arenas option.
FILE_SAME_CC_GENERIC_SERVICES                   FILE, PACKAGE                   Checks that files have the same value for the cc_generic_services option.
FILE_SAME_CSHARP_NAMESPACE                      FILE, PACKAGE                   Checks that files have the same value for the csharp_namespace option.
FILE_SAME_GO_PACKAGE                            FILE, PACKAGE                   Checks that files have the same value for the go_package option.
FILE_SAME_JAVA_GENERIC_SERVICES                 FILE, PACKAGE                   Checks that files have the same value for the java_generic_services option.
FILE_SAME_JAVA_MULTIPLE_FILES                   FILE, PACKAGE                   Checks that files have the same value for the java_multiple_files option.
FILE_SAME_JAVA_OUTER_CLASSNAME                  FILE, PACKAGE                   Checks that files have the same value for the java_outer_classname option.
FILE_SAME_JAVA_PACKAGE                          FILE, PACKAGE                   Checks that files have the same value for the java_package option.
FILE_SAME_JAVA_STRING_CHECK_UTF8                FILE, PACKAGE                   Checks that files have the same value for the java_string_check_utf8 option.
FILE_SAME_OBJC_CLASS_PREFIX                     FILE, PACKAGE                   Checks that files have the same value for the objc_class_prefix option.
FILE_SAME_OPTIMIZE_FOR                          FILE, PACKAGE                   Checks that files have the same value for the optimize_for option.
FILE_SAME_PHP_CLASS_PREFIX                      FILE, PACKAGE                   Checks that files have the same value for the php_class_prefix option.
FILE_SAME_PHP_GENERIC_SERVICES                  FILE, PACKAGE                   Checks that files have the same value for the php_generic_services option.
FILE_SAME_PHP_METADATA_NAMESPACE                FILE, PACKAGE                   Checks that files have the same value for the php_metadata_namespace option.
FILE_SAME_PHP_NAMESPACE                         FILE, PACKAGE                   Checks that files have the same value for the php_namespace option.
FILE_SAME_PY_GENERIC_SERVICES                   FILE, PACKAGE                   Checks that files have the same value for the py_generic_services option.
FILE_SAME_RUBY_PACKAGE                          FILE, PACKAGE                   Checks that files have the same value for the ruby_package option.
FILE_SAME_SWIFT_PREFIX                          FILE, PACKAGE                   Checks that files have the same value for the swift_prefix option.
FILE_SAME_SYNTAX                                FILE, PACKAGE                   Checks that files have the same syntax.
MESSAGE_NO_REMOVE_STANDARD_DESCRIPTOR_ACCESSOR  FILE, PACKAGE                   Checks that messages do not change the no_standard_descriptor_accessor option from false or unset to true.
ONEOF_NO_DELETE                                 FILE, PACKAGE                   Checks that oneofs are not deleted from a given message.
RPC_NO_DELETE                                   FILE, PACKAGE                   Checks that rpcs are not deleted from a given service.
ENUM_VALUE_SAME_NAME                            FILE, PACKAGE, WIRE_JSON        Checks that enum values have the same name.
FIELD_SAME_JSON_NAME                            FILE, PACKAGE, WIRE_JSON        Checks that fields have the same value for the json_name option.
FIELD_SAME_NAME                                 FILE, PACKAGE, WIRE_JSON        Checks that fields have the same names in a given message.
FIELD_SAME_LABEL                                FILE, PACKAGE, WIRE_JSON, WIRE  Checks that fields have the same labels in a given message.
FIELD_SAME_ONEOF                                FILE, PACKAGE, WIRE_JSON, WIRE  Checks that fields have the same oneofs in a given message.
FIELD_SAME_TYPE                                 FILE, PACKAGE, WIRE_JSON, WIRE  Checks that fields have the same types in a given message.
MESSAGE_SAME_MESSAGE_SET_WIRE_FORMAT            FILE, PACKAGE, WIRE_JSON, WIRE  Checks that messages have the same value for the message_set_wire_format option.
MESSAGE_SAME_REQUIRED_FIELDS                    FILE, PACKAGE, WIRE_JSON, WIRE  Checks that messages have no added or deleted required fields.
RESERVED_ENUM_NO_DELETE                         FILE, PACKAGE, WIRE_JSON, WIRE  Checks that reserved ranges and names are not deleted from a given enum.
RESERVED_MESSAGE_NO_DELETE                      FILE, PACKAGE, WIRE_JSON, WIRE  Checks that reserved ranges and names are not deleted from a given message.
RPC_SAME_CLIENT_STREAMING                       FILE, PACKAGE, WIRE_JSON, WIRE  Checks that rpcs have the same client streaming value.
RPC_SAME_IDEMPOTENCY_LEVEL                      FILE, PACKAGE, WIRE_JSON, WIRE  Checks that rpcs have the same value for the idempotency_level option.
RPC_SAME_REQUEST_TYPE                           FILE, PACKAGE, WIRE_JSON, WIRE  Checks that rpcs are have the same request type.
RPC_SAME_RESPONSE_TYPE                          FILE, PACKAGE, WIRE_JSON, WIRE  Checks that rpcs are have the same response type.
RPC_SAME_SERVER_STREAMING                       FILE, PACKAGE, WIRE_JSON, WIRE  Checks that rpcs have the same server streaming value.
PACKAGE_ENUM_NO_DELETE                          PACKAGE                         Checks that enums are not deleted from a given package.
PACKAGE_MESSAGE_NO_DELETE                       PACKAGE                         Checks that messages are not deleted from a given package.
PACKAGE_NO_DELETE                               PACKAGE                         Checks that packages are not deleted.
PACKAGE_SERVICE_NO_DELETE                       PACKAGE                         Checks that services are not deleted from a given package.
ENUM_VALUE_NO_DELETE_UNLESS_NAME_RESERVED       WIRE_JSON                       Checks that enum values are not deleted from a given enum unless the name is reserved.
FIELD_NO_DELETE_UNLESS_NAME_RESERVED            WIRE_JSON                       Checks that fields are not deleted from a given message unless the name is reserved.
ENUM_VALUE_NO_DELETE_UNLESS_NUMBER_RESERVED     WIRE_JSON, WIRE                 Checks that enum values are not deleted from a given enum unless the number is reserved.
FIELD_NO_DELETE_UNLESS_NUMBER_RESERVED          WIRE_JSON, WIRE                 Checks that fields are not deleted from a given message unless the number is reserved.
		`
	testRunStdout(
		t,
		nil,
		0,
		expectedStdout,
		"mod",
		"ls-breaking-rules",
		"--version",
		"v1beta1",
	)
}

func TestLsFiles(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`testdata/success/buf/buf.proto`),
		"ls-files",
		filepath.Join("testdata", "success"),
	)
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`testdata/success/buf/buf.proto`),
		"ls-files",
		filepath.Join("testdata", "success", "buf", "buf.proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`testdata/protofileref/success/buf.proto`),
		"ls-files",
		// test single file ref that is not part of the module or workspace
		filepath.Join("testdata", "protofileref", "success", "buf.proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`
			testdata/protofileref/success/buf.proto
			testdata/protofileref/success/other.proto
		`),
		"ls-files",
		// test single file ref that is not part of the module or workspace
		fmt.Sprintf("%s#include_package_files=true", filepath.Join("testdata", "protofileref", "success", "buf.proto")),
	)
}

func TestLsFilesIncludeImports(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		0,
		`google/protobuf/descriptor.proto
`+filepath.FromSlash(`testdata/success/buf/buf.proto`),
		"ls-files",
		"--include-imports",
		filepath.Join("testdata", "success"),
	)
	testRunStdout(
		t,
		nil,
		0,
		`google/protobuf/descriptor.proto
`+filepath.FromSlash(`testdata/protofileref/success/buf.proto`),
		"ls-files",
		"--include-imports",
		filepath.Join("testdata", "protofileref", "success", "buf.proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		`google/protobuf/descriptor.proto`+filepath.FromSlash(`
		testdata/protofileref/success/buf.proto
		testdata/protofileref/success/other.proto
		`),
		"ls-files",
		"--include-imports",
		fmt.Sprintf("%s#include_package_files=true", filepath.Join("testdata", "protofileref", "success", "buf.proto")),
	)
}

func TestLsFilesIncludeImportsAsImportPaths(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		0,
		`buf/buf.proto
google/protobuf/descriptor.proto`,
		"ls-files",
		"--include-imports",
		"--as-import-paths",
		filepath.Join("testdata", "success"),
	)
	testRunStdout(
		t,
		nil,
		0,
		`buf/buf.proto
google/protobuf/descriptor.proto`,
		"ls-files",
		"--include-imports",
		"--as-import-paths",
		filepath.Join("testdata", "success", "buf", "buf.proto"),
	)
}

func TestLsFilesImage1(t *testing.T) {
	t.Parallel()
	stdout := bytes.NewBuffer(nil)
	testRun(
		t,
		0,
		nil,
		stdout,
		"build",
		"-o",
		"-",
		filepath.Join("testdata", "success"),
	)
	testRunStdout(
		t,
		stdout,
		0,
		`
		buf/buf.proto
		`,
		"ls-files",
		"-",
	)
}

func TestLsFilesImage2(t *testing.T) {
	t.Parallel()
	stdout := bytes.NewBuffer(nil)
	testRun(
		t,
		0,
		nil,
		stdout,
		"build",
		"-o",
		"-",
		filepath.Join("testdata", "success"),
	)
	testRunStdout(
		t,
		stdout,
		0,
		`
		buf/buf.proto
		google/protobuf/descriptor.proto
		`,
		"ls-files",
		"-",
		"--include-imports",
	)
}

func TestLsFilesImage3(t *testing.T) {
	t.Parallel()
	stdout := bytes.NewBuffer(nil)
	testRun(
		t,
		0,
		nil,
		stdout,
		"build",
		"--exclude-imports",
		"-o",
		"-",
		filepath.Join("testdata", "success"),
	)
	testRunStdout(
		t,
		stdout,
		0,
		`
		buf/buf.proto
		`,
		"ls-files",
		"-",
	)
}

func TestLsFilesImage4(t *testing.T) {
	t.Parallel()
	stdout := bytes.NewBuffer(nil)
	testRun(
		t,
		0,
		nil,
		stdout,
		"build",
		"-o",
		"-",
		filepath.Join("testdata", "success", "buf", "buf.proto"),
	)
	testRunStdout(
		t,
		stdout,
		0,
		`
		buf/buf.proto
		`,
		"ls-files",
		"-",
	)
}

func TestLsFilesImage5(t *testing.T) {
	t.Parallel()
	stdout := bytes.NewBuffer(nil)
	testRun(
		t,
		0,
		nil,
		stdout,
		"build",
		"-o",
		"-",
		filepath.Join("testdata", "success", "buf", "buf.proto"),
	)
	testRunStdout(
		t,
		stdout,
		0,
		`
		buf/buf.proto
		google/protobuf/descriptor.proto
		`,
		"ls-files",
		"--include-imports",
		"-",
	)
}

func TestBuildFailProtoFileRefWithPathFlag(t *testing.T) {
	t.Parallel()
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		"", // stdout should be empty
		`Failure: path "." is not contained within any of roots "." - note that specified paths cannot be roots, but must be contained within roots`,
		"build",
		filepath.Join("testdata", "success", "buf", "buf.proto"),
		"--path",
		filepath.Join("testdata", "success", "buf", "buf.proto"),
		"-o",
		"-",
	)
}

func TestImageConvertRoundtripBinaryJSONBinary(t *testing.T) {
	t.Parallel()

	stdout := bytes.NewBuffer(nil)
	testRun(
		t,
		0,
		nil,
		stdout,
		"build",
		"-o",
		"-",
		filepath.Join("testdata", "customoptions1"),
	)

	binary1 := stdout.Bytes()
	require.NotEmpty(t, binary1)

	stdin := stdout
	stdout = bytes.NewBuffer(nil)
	testRun(
		t,
		0,
		stdin,
		stdout,
		"build",
		"-",
		"-o",
		"-#format=json",
	)

	stdin = stdout
	stdout = bytes.NewBuffer(nil)
	testRun(
		t,
		0,
		stdin,
		stdout,
		"build",
		"-#format=json",
		"-o",
		"-",
	)

	require.Equal(t, binary1, stdout.Bytes())
}

func TestImageConvertRoundtripJSONBinaryJSON(t *testing.T) {
	t.Parallel()

	stdout := bytes.NewBuffer(nil)
	testRun(
		t,
		0,
		nil,
		stdout,
		"build",
		"-o",
		"-#format=json",
		filepath.Join("testdata", "customoptions1"),
	)

	json1 := stdout.Bytes()
	require.NotEmpty(t, json1)

	stdin := stdout
	stdout = bytes.NewBuffer(nil)
	testRun(
		t,
		0,
		stdin,
		stdout,
		"build",
		"-#format=json",
		"-o",
		"-",
	)

	stdin = stdout
	stdout = bytes.NewBuffer(nil)
	testRun(
		t,
		0,
		stdin,
		stdout,
		"build",
		"-",
		"-o",
		"-#format=json",
	)

	require.Equal(t, json1, stdout.Bytes())
}

func TestModInitBasic(t *testing.T) {
	t.Parallel()
	testModInit(
		t,
		`version: v1
breaking:
  use:
    - FILE
lint:
  use:
    - DEFAULT
`,
		false,
		"",
	)
}

func TestExportProto(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"export",
		"-o",
		tempDir,
		filepath.Join("testdata", "export", "proto"),
	)
	readWriteBucket, err := storageos.NewProvider().NewReadWriteBucket(tempDir)
	require.NoError(t, err)
	// This should NOT include unimported.proto
	storagetesting.AssertPaths(
		t,
		readWriteBucket,
		"",
		"request.proto",
		"rpc.proto",
	)
}

func TestExportOtherProto(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"export",
		"-o",
		tempDir,
		filepath.Join("testdata", "export", "other", "proto"),
	)
	readWriteBucket, err := storageos.NewProvider().NewReadWriteBucket(tempDir)
	require.NoError(t, err)
	storagetesting.AssertPaths(
		t,
		readWriteBucket,
		"",
		"request.proto",
		"unimported.proto",
		"another.proto",
	)
}

func TestExportAll(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"export",
		"-o",
		tempDir,
		filepath.Join("testdata", "export"),
	)
	readWriteBucket, err := storageos.NewProvider().NewReadWriteBucket(tempDir)
	require.NoError(t, err)
	storagetesting.AssertPaths(
		t,
		readWriteBucket,
		"",
		"another.proto",
		"request.proto",
		"rpc.proto",
		"unimported.proto",
	)
}

func TestExportExcludeImports(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"export",
		"--exclude-imports",
		"-o",
		tempDir,
		filepath.Join("testdata", "export", "proto"),
	)
	readWriteBucket, err := storageos.NewProvider().NewReadWriteBucket(tempDir)
	require.NoError(t, err)
	storagetesting.AssertPaths(
		t,
		readWriteBucket,
		"",
		"rpc.proto",
	)
}

func TestExportPaths(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"export",
		"--path",
		filepath.Join("testdata", "export", "other", "proto", "request.proto"),
		"-o",
		tempDir,
		filepath.Join("testdata", "export"),
	)
	readWriteBucket, err := storageos.NewProvider().NewReadWriteBucket(tempDir)
	require.NoError(t, err)
	storagetesting.AssertPaths(
		t,
		readWriteBucket,
		"",
		"request.proto",
	)
}

func TestExportPathsAndExcludes(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"export",
		filepath.Join("testdata", "paths"),
		"--path",
		filepath.Join("testdata", "paths", "a", "v3"),
		"--exclude-path",
		filepath.Join("testdata", "paths", "a", "v3", "foo"),
		"-o",
		tempDir,
	)
	readWriteBucket, err := storageos.NewProvider().NewReadWriteBucket(tempDir)
	require.NoError(t, err)
	storagetesting.AssertPaths(
		t,
		readWriteBucket,
		"",
		"a/v3/a.proto",
	)
	storagetesting.AssertNotExist(
		t,
		readWriteBucket,
		"a/v3/foo/foo.proto",
	)
	storagetesting.AssertNotExist(
		t,
		readWriteBucket,
		"a/v3/foo/bar.proto",
	)
}

func TestExportProtoFileRef(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"export",
		"-o",
		tempDir,
		filepath.Join("testdata", "export", "proto", "rpc.proto"),
	)
	readWriteBucket, err := storageos.NewProvider().NewReadWriteBucket(tempDir)
	require.NoError(t, err)
	storagetesting.AssertPaths(
		t,
		readWriteBucket,
		"",
		"request.proto",
		"rpc.proto",
	)
}

func TestExportProtoFileRefExcludeImports(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"export",
		"--exclude-imports",
		"-o",
		tempDir,
		filepath.Join("testdata", "export", "proto", "rpc.proto"),
	)
	readWriteBucket, err := storageos.NewProvider().NewReadWriteBucket(tempDir)
	require.NoError(t, err)
	storagetesting.AssertPaths(
		t,
		readWriteBucket,
		"",
		"rpc.proto",
	)
}

func TestExportProtoFileRefIncludePackageFiles(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"export",
		"-o",
		tempDir,
		fmt.Sprintf("%s#include_package_files=true", filepath.Join("testdata", "export", "other", "proto", "request.proto")),
	)
	readWriteBucket, err := storageos.NewProvider().NewReadWriteBucket(tempDir)
	require.NoError(t, err)
	storagetesting.AssertPaths(
		t,
		readWriteBucket,
		"",
		"request.proto",
		"unimported.proto",
		"another.proto",
	)
}

func TestExportProtoFileRefIncludePackageFilesExcludeImports(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"export",
		"--exclude-imports",
		"-o",
		tempDir,
		fmt.Sprintf("%s#include_package_files=true", filepath.Join("testdata", "export", "other", "proto", "request.proto")),
	)
	readWriteBucket, err := storageos.NewProvider().NewReadWriteBucket(tempDir)
	require.NoError(t, err)
	storagetesting.AssertPaths(
		t,
		readWriteBucket,
		"",
		"request.proto",
		"unimported.proto",
	)
}

func TestExportProtoFileRefWithPathFlag(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		"", // stdout should be empty
		`Failure: path "." is not contained within any of roots "." - note that specified paths cannot be roots, but must be contained within roots`,
		"export",
		filepath.Join("testdata", "protofileref", "success", "buf.proto"),
		"-o",
		tempDir,
		"--path",
		filepath.Join("testdata", "protofileref", "success", "buf.proto"),
	)
}

func TestBuildWithPaths(t *testing.T) {
	t.Parallel()
	testRunStdout(t, nil, 0, ``, "build", filepath.Join("testdata", "paths"), "--path", filepath.Join("testdata", "paths", "a", "v3"), "--exclude-path", filepath.Join("testdata", "paths", "a", "v3", "foo"))
	testRunStdout(t, nil, 0, ``, "build", filepath.Join("testdata", "paths"), "--path", filepath.Join("testdata", "paths", "a", "v3", "foo"), "--exclude-path", filepath.Join("testdata", "paths", "a", "v3"))
}

func TestLintWithPaths(t *testing.T) {
	t.Parallel()
	testRunStdoutStderrNoWarn(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/paths/a/v3/a.proto:7:10:Field name "Value" should be lower_snake_case, such as "value".`),
		"",
		"lint",
		filepath.Join("testdata", "paths"),
		"--path",
		filepath.Join("testdata", "paths", "a", "v3"),
		"--exclude-path",
		filepath.Join("testdata", "paths", "a", "v3", "foo"),
	)
	testRunStdoutStderrNoWarn(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(
			`testdata/paths/a/v3/foo/bar.proto:3:1:Package name "a.v3.foo" should be suffixed with a correctly formed version, such as "a.v3.foo.v1".
testdata/paths/a/v3/foo/foo.proto:3:1:Package name "a.v3.foo" should be suffixed with a correctly formed version, such as "a.v3.foo.v1".`),
		"",
		"lint",
		filepath.Join("testdata", "paths"),
		"--path",
		filepath.Join("testdata", "paths", "a", "v3", "foo"),
		"--exclude-path",
		filepath.Join("testdata", "paths", "a", "v3"),
	)
}

func TestBreakingWithPaths(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	testRunStdout(t, nil, 0, ``, "build", filepath.Join("command", "generate", "testdata", "paths"), "-o", filepath.Join(tempDir, "previous.binpb"))
	testRunStdout(t, nil, 0, ``, "build", filepath.Join("testdata", "paths"), "-o", filepath.Join(tempDir, "current.binpb"))
	readWriteBucket, err := storageos.NewProvider().NewReadWriteBucket(tempDir)
	require.NoError(t, err)
	storagetesting.AssertPaths(
		t,
		readWriteBucket,
		"",
		"previous.binpb",
		"current.binpb",
	)
	testRunStdoutStderrNoWarn(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		`a/v3/a.proto:6:3:Field "1" on message "Foo" changed type from "string" to "int32".
a/v3/a.proto:7:3:Field "2" with name "Value" on message "Foo" changed option "json_name" from "value" to "Value".
a/v3/a.proto:7:10:Field "2" on message "Foo" changed name from "value" to "Value".`,
		"",
		"breaking",
		filepath.Join(tempDir, "current.binpb"),
		"--against",
		filepath.Join(tempDir, "previous.binpb"),
		"--path",
		filepath.Join("a", "v3"),
		"--exclude-path",
		filepath.Join("a", "v3", "foo"),
	)
}

func TestVersion(t *testing.T) {
	t.Parallel()
	testRunStdout(t, nil, 0, bufcli.Version, "--version")
}

func TestMigrateV1Beta1(t *testing.T) {
	t.Parallel()
	storageosProvider := storageos.NewProvider()
	runner := command.NewRunner()

	// These test cases are ordered alphabetically to align with the folders in testadata.
	t.Run("buf-gen-yaml-without-version", func(t *testing.T) {
		t.Parallel()
		testMigrateV1Beta1Diff(
			t,
			storageosProvider,
			runner,
			"buf-gen-yaml-without-version",
			"Successfully migrated your buf.gen.yaml to v1.",
		)
	})
	t.Run("buf-yaml-without-version", func(t *testing.T) {
		t.Parallel()
		testMigrateV1Beta1Diff(
			t,
			storageosProvider,
			runner,
			"buf-yaml-without-version",
			"Successfully migrated your buf.yaml to v1.",
		)
	})
	t.Run("complex", func(t *testing.T) {
		t.Parallel()
		testMigrateV1Beta1Diff(
			t,
			storageosProvider,
			runner,
			"complex",
			`The ignored file "file3.proto" was not found in any roots and has been removed.
Successfully migrated your buf.yaml and buf.gen.yaml to v1.`,
		)
	})
	t.Run("deps-without-name", func(t *testing.T) {
		t.Parallel()
		testMigrateV1Beta1Diff(
			t,
			storageosProvider,
			runner,
			"deps-without-name",
			"Successfully migrated your buf.yaml to v1.",
		)
	})
	t.Run("flat-deps-without-name", func(t *testing.T) {
		t.Parallel()
		testMigrateV1Beta1Diff(
			t,
			storageosProvider,
			runner,
			"flat-deps-without-name",
			"Successfully migrated your buf.yaml and buf.lock to v1.",
		)
	})
	t.Run("lock-file-without-deps", func(t *testing.T) {
		t.Parallel()
		testMigrateV1Beta1Diff(
			t,
			storageosProvider,
			runner,
			"lock-file-without-deps",
			`Successfully migrated your buf.yaml and buf.lock to v1.`,
		)
	})
	t.Run("nested-folder", func(t *testing.T) {
		t.Parallel()
		testMigrateV1Beta1Diff(
			t,
			storageosProvider,
			runner,
			"nested-folder",
			"Successfully migrated your buf.yaml to v1.",
		)
	})
	t.Run("nested-root", func(t *testing.T) {
		t.Parallel()
		testMigrateV1Beta1Diff(
			t,
			storageosProvider,
			runner,
			"nested-root",
			"Successfully migrated your buf.yaml and buf.gen.yaml to v1.",
		)
	})
	t.Run("no-deps", func(t *testing.T) {
		t.Parallel()
		testMigrateV1Beta1Diff(
			t,
			storageosProvider,
			runner,
			"no-deps",
			"Successfully migrated your buf.yaml and buf.gen.yaml to v1.",
		)
	})
	t.Run("noop", func(t *testing.T) {
		t.Parallel()
		testMigrateV1Beta1Diff(
			t,
			storageosProvider,
			runner,
			"noop",
			"",
		)
	})
	t.Run("only-buf-gen-yaml", func(t *testing.T) {
		t.Parallel()
		testMigrateV1Beta1Diff(
			t,
			storageosProvider,
			runner,
			"only-buf-gen-yaml",
			"Successfully migrated your buf.gen.yaml to v1.",
		)
	})
	t.Run("only-buf-lock", func(t *testing.T) {
		t.Parallel()
		testMigrateV1Beta1Diff(
			t,
			storageosProvider,
			runner,
			"only-buf-lock",
			"Successfully migrated your buf.lock to v1.",
		)
	})
	t.Run("only-buf-yaml", func(t *testing.T) {
		t.Parallel()
		testMigrateV1Beta1Diff(
			t,
			storageosProvider,
			runner,
			"only-buf-yaml",
			"Successfully migrated your buf.yaml to v1.",
		)
	})
	t.Run("only-old-buf-gen-yaml", func(t *testing.T) {
		t.Parallel()
		testMigrateV1Beta1Diff(
			t,
			storageosProvider,
			runner,
			"only-old-buf-gen-yaml",
			"Successfully migrated your buf.gen.yaml to v1.",
		)
	})
	t.Run("only-old-buf-lock", func(t *testing.T) {
		t.Parallel()
		testMigrateV1Beta1Diff(
			t,
			storageosProvider,
			runner,
			"only-old-buf-lock",
			`Successfully migrated your buf.lock to v1.`,
		)
	})
	t.Run("only-old-buf-yaml", func(t *testing.T) {
		t.Parallel()
		testMigrateV1Beta1Diff(
			t,
			storageosProvider,
			runner,
			"only-old-buf-yaml",
			"Successfully migrated your buf.yaml to v1.",
		)
	})
	t.Run("simple", func(t *testing.T) {
		t.Parallel()
		testMigrateV1Beta1Diff(
			t,
			storageosProvider,
			runner,
			"simple",
			"Successfully migrated your buf.yaml, buf.gen.yaml, and buf.lock to v1.",
		)
	})
	t.Run("v1beta1-lock-file", func(t *testing.T) {
		t.Parallel()
		testMigrateV1Beta1Diff(
			t,
			storageosProvider,
			runner,
			"v1beta1-lock-file",
			`Successfully migrated your buf.yaml and buf.lock to v1.`,
		)
	})

	t.Run("fails-on-invalid-version", func(t *testing.T) {
		t.Parallel()
		testMigrateV1Beta1Failure(
			t,
			storageosProvider,
			"invalid-version",
			`failed to migrate config: unknown config file version: spaghetti`,
		)
	})
}

func TestConvertWithImage(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("testdata", "success"),
		"-o",
		filepath.Join(tempDir, "image.binpb"),
	)

	t.Run("stdin input", func(t *testing.T) {
		t.Parallel()
		stdin, err := os.Open(filepath.Join(convertTestDataDir, "descriptor.plain.binpb"))
		require.NoError(t, err)
		defer stdin.Close()
		stdout := bytes.NewBuffer(nil)
		testRun(
			t,
			0,
			stdin,
			stdout,
			"convert",
			filepath.Join(tempDir, "image.binpb"),
			"--type",
			"buf.Foo",
		)
		assert.JSONEq(t, `{"one":"55"}`, stdout.String())
	})

	t.Run("no stdin input", func(t *testing.T) {
		t.Parallel()
		testRunStdoutStderrNoWarn(
			t,
			nil,
			1,
			"",
			"Failure: size of input message must not be zero",
			"convert",
			filepath.Join(tempDir, "image.binpb"),
			"--type",
			"buf.Foo",
		)
	})
}

func TestConvertOutput(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("testdata", "success"),
		"-o",
		filepath.Join(tempDir, "image.binpb"),
	)
	t.Run("json file output", func(t *testing.T) {
		t.Parallel()
		stdin, err := os.Open(filepath.Join(convertTestDataDir, "descriptor.plain.binpb"))
		require.NoError(t, err)
		defer stdin.Close()
		outputTempDir := t.TempDir()
		testRunStdout(
			t,
			stdin,
			0,
			``,
			"convert",
			filepath.Join(tempDir, "image.binpb"),
			"--type",
			"buf.Foo",
			"--to",
			filepath.Join(outputTempDir, "result.json"),
		)
		readWriteBucket, err := storageos.NewProvider().NewReadWriteBucket(outputTempDir)
		require.NoError(t, err)
		storagetesting.AssertPathToContent(
			t,
			readWriteBucket,
			"",
			map[string]string{
				"result.json": `{"one":"55"}`,
			},
		)
	})
	t.Run("txt file output", func(t *testing.T) {
		t.Parallel()
		stdin, err := os.Open(filepath.Join(convertTestDataDir, "descriptor.plain.binpb"))
		require.NoError(t, err)
		defer stdin.Close()
		outputTempDir := t.TempDir()
		testRunStdout(
			t,
			stdin,
			0,
			``,
			"convert",
			filepath.Join(tempDir, "image.binpb"),
			"--type",
			"buf.Foo",
			"--to",
			filepath.Join(outputTempDir, "result.txt"),
		)
		readWriteBucket, err := storageos.NewProvider().NewReadWriteBucket(outputTempDir)
		require.NoError(t, err)
		storagetesting.AssertPathToContent(
			t,
			readWriteBucket,
			"",
			map[string]string{
				"result.txt": `{"one":"55"}`,
			},
		)
	})
	t.Run("stdout with dash", func(t *testing.T) {
		t.Parallel()
		stdin, err := os.Open(filepath.Join(convertTestDataDir, "descriptor.plain.binpb"))
		require.NoError(t, err)
		defer stdin.Close()
		stdout := bytes.NewBuffer(nil)
		testRun(
			t,
			0,
			stdin,
			stdout,
			"convert",
			filepath.Join(tempDir, "image.binpb"),
			"--type",
			"buf.Foo",
			"--to",
			"-",
		)
		assert.JSONEq(t, `{"one":"55"}`, stdout.String())
	})
}

func TestConvertInvalidTypeName(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("testdata", "success"),
		"-o",
		filepath.Join(tempDir, "image.binpb"),
	)
	stdin, err := os.Open(filepath.Join(convertTestDataDir, "descriptor.plain.binpb"))
	require.NoError(t, err)
	defer stdin.Close()
	testRunStdoutStderrNoWarn(
		t,
		stdin,
		1,
		"",
		`Failure: ".foo" is not a valid fully qualified type name`,
		"convert",
		filepath.Join(tempDir, "image.binpb"),
		"--type",
		".foo",
	)
}

func TestConvert(t *testing.T) {
	t.Parallel()
	t.Run("binpb-to-json-file-proto", func(t *testing.T) {
		t.Parallel()
		testRunStdoutFile(t,
			nil,
			0,
			convertTestDataDir+"/bin_json/payload.json",
			"convert",
			"--type=buf.Foo",
			"--from="+convertTestDataDir+"/bin_json/payload.binpb",
			convertTestDataDir+"/bin_json/buf.proto",
		)
	})
	t.Run("json-to-binpb-file-proto", func(t *testing.T) {
		t.Parallel()
		testRunStdoutFile(t,
			nil,
			0,
			convertTestDataDir+"/bin_json/payload.binpb",
			"convert",
			"--type=buf.Foo",
			"--from="+convertTestDataDir+"/bin_json/payload.json",
			convertTestDataDir+"/bin_json/buf.proto",
		)
	})
	t.Run("stdin-json-to-binpb-proto", func(t *testing.T) {
		t.Parallel()
		testRunStdoutFile(t,
			strings.NewReader(`{"one":"55"}`),
			0,
			convertTestDataDir+"/bin_json/payload.binpb",
			"convert",
			"--type=buf.Foo",
			"--from",
			"-#format=json",
			convertTestDataDir+"/bin_json/buf.proto",
		)
	})
	t.Run("stdin-binpb-to-json-proto", func(t *testing.T) {
		t.Parallel()
		file, err := os.Open(convertTestDataDir + "/bin_json/payload.binpb")
		require.NoError(t, err)
		testRunStdoutFile(t, file,
			0,
			convertTestDataDir+"/bin_json/payload.json",
			"convert",
			"--type=buf.Foo",
			"--from",
			"-#format=binpb",
			convertTestDataDir+"/bin_json/buf.proto",
		)
	})
	t.Run("stdin-json-to-json-proto", func(t *testing.T) {
		t.Parallel()
		testRunStdoutFile(t,
			strings.NewReader(`{"one":"55"}`),
			0,
			convertTestDataDir+"/bin_json/payload.json",
			"convert",
			"--type=buf.Foo",
			convertTestDataDir+"/bin_json/buf.proto",
			"--from",
			"-#format=json",
			"--to",
			"-#format=json")
	})
	t.Run("stdin-input-to-json-image", func(t *testing.T) {
		t.Parallel()
		file, err := os.Open(convertTestDataDir + "/bin_json/image.binpb")
		require.NoError(t, err)
		testRunStdoutFile(t, file,
			0,
			convertTestDataDir+"/bin_json/payload.json",
			"convert",
			"--type=buf.Foo",
			"-",
			"--from="+convertTestDataDir+"/bin_json/payload.binpb",
			"--to",
			"-#format=json",
		)
	})
	t.Run("stdin-json-to-json-image", func(t *testing.T) {
		t.Parallel()
		file, err := os.Open(convertTestDataDir + "/bin_json/payload.binpb")
		require.NoError(t, err)
		testRunStdoutFile(t,
			file,
			0,
			convertTestDataDir+"/bin_json/payload.json",
			"convert",
			"--type=buf.Foo",
			convertTestDataDir+"/bin_json/image.binpb",
			"--from",
			"-#format=binpb",
			"--to",
			"-#format=json")
	})
	t.Run("stdin-binpb-payload-to-json-with-image", func(t *testing.T) {
		t.Parallel()
		file, err := os.Open(convertTestDataDir + "/bin_json/payload.binpb")
		require.NoError(t, err)
		testRunStdoutFile(t,
			file,
			0,
			convertTestDataDir+"/bin_json/payload.json",
			"convert",
			"--type=buf.Foo",
			convertTestDataDir+"/bin_json/image.binpb",
			"--to",
			"-#format=json",
		)
	})
	t.Run("stdin-json-payload-to-binpb-with-image", func(t *testing.T) {
		t.Parallel()
		file, err := os.Open(convertTestDataDir + "/bin_json/payload.json")
		require.NoError(t, err)
		testRunStdoutFile(t,
			file,
			0,
			convertTestDataDir+"/bin_json/payload.binpb",
			"convert",
			"--type=buf.Foo",
			convertTestDataDir+"/bin_json/image.binpb",
			"--from",
			"-#format=json",
			"--to",
			"-#format=binpb",
		)
	})
	t.Run("stdin-image-json-to-binpb", func(t *testing.T) {
		t.Parallel()
		file, err := os.Open(convertTestDataDir + "/bin_json/image.json")
		require.NoError(t, err)
		testRunStdoutFile(t,
			file,
			0,
			convertTestDataDir+"/bin_json/payload.binpb",
			"convert",
			"--type=buf.Foo",
			"-#format=json",
			"--from="+convertTestDataDir+"/bin_json/payload.json",
			"--to",
			"-#format=binpb",
		)
	})
	t.Run("stdin-image-txtpb-to-binpb", func(t *testing.T) {
		t.Parallel()
		file, err := os.Open(convertTestDataDir + "/bin_json/image.txtpb")
		require.NoError(t, err)
		testRunStdoutFile(t,
			file,
			0,
			convertTestDataDir+"/bin_json/payload.binpb",
			"convert",
			"--type=buf.Foo",
			"-#format=txtpb",
			"--from="+convertTestDataDir+"/bin_json/payload.txtpb",
			"--to",
			"-#format=binpb",
		)
	})
}

func TestFormat(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		0,
		`
syntax = "proto3";

package simple;

message Object {
  string key = 1;
  bytes value = 2;
}
		`,
		"format",
		filepath.Join("testdata", "format", "simple"),
	)
}

func TestFormatSingleFile(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"format",
		filepath.Join("testdata", "format", "simple"),
		"-o",
		filepath.Join(tempDir, "simple.formatted.proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"format",
		filepath.Join(tempDir, "simple.formatted.proto"),
		"-d",
	)
}

func TestFormatDiff(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	stdout := bytes.NewBuffer(nil)
	testRun(
		t,
		0,
		nil,
		stdout,
		"format",
		filepath.Join("testdata", "format", "diff"),
		"-d",
		"-o",
		filepath.Join(tempDir, "formatted"),
	)
	assert.Contains(
		t,
		stdout.String(),
		`
@@ -1,13 +1,7 @@
-
 syntax = "proto3";
`,
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"format",
		filepath.Join(tempDir, "formatted"),
		"-d",
	)
}

// Tests if the exit code is set for common invocations of buf format
// with the --exit-code flag.
func TestFormatExitCode(t *testing.T) {
	t.Parallel()
	stdout := bytes.NewBuffer(nil)
	testRun(
		t,
		bufcli.ExitCodeFileAnnotation,
		nil,
		stdout,
		"format",
		filepath.Join("testdata", "format", "diff"),
		"--exit-code",
	)
	assert.NotEmpty(t, stdout.String())
	stdout = bytes.NewBuffer(nil)
	testRun(
		t,
		bufcli.ExitCodeFileAnnotation,
		nil,
		stdout,
		"format",
		filepath.Join("testdata", "format", "diff"),
		"-d",
		"--exit-code",
	)
	assert.NotEmpty(t, stdout.String())
}

// Tests if the image produced by the formatted result is
// equivalent to the original result.
func TestFormatEquivalence(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("testdata", "format", "complex"),
		"-o",
		filepath.Join(tempDir, "image.binpb"),
		"--exclude-source-info",
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"format",
		filepath.Join("testdata", "format", "complex"),
		"-o",
		filepath.Join(tempDir, "formatted"),
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join(tempDir, "formatted"),
		"-o",
		filepath.Join(tempDir, "formatted.binpb"),
		"--exclude-source-info",
	)
	originalImageData, err := os.ReadFile(filepath.Join(tempDir, "image.binpb"))
	require.NoError(t, err)
	formattedImageData, err := os.ReadFile(filepath.Join(tempDir, "formatted.binpb"))
	require.NoError(t, err)
	require.Equal(t, originalImageData, formattedImageData)
}

func TestFormatInvalidFlagCombination(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		"",
		`Failure: --output cannot be used with --write`,
		"format",
		filepath.Join("testdata", "format", "diff"),
		"-w",
		"-o",
		filepath.Join(tempDir, "formatted"),
	)
}

func TestFormatInvalidWriteWithModuleReference(t *testing.T) {
	t.Parallel()
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		"",
		`Failure: --write cannot be used with module reference inputs`,
		"format",
		"buf.build/acme/weather",
		"-w",
	)
}

func TestFormatInvalidIncludePackageFiles(t *testing.T) {
	t.Parallel()
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		"",
		`Failure: this command does not support including package files`,
		"format",
		filepath.Join("testdata", "format", "simple", "simple.proto#include_package_files=true"),
	)
}

func TestFormatInvalidInputDoesNotCreateDirectory(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		"",
		filepath.FromSlash(`Failure: testdata/format/invalid/invalid.proto:4:12: syntax error: unexpected '.', expecting '{'`),
		"format",
		filepath.Join("testdata", "format", "invalid"),
		"-o",
		filepath.Join(tempDir, "formatted", "invalid"), // Directory output.
	)
	_, err := os.Stat(filepath.Join(tempDir, "formatted", "invalid"))
	assert.True(t, os.IsNotExist(err))
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		"",
		filepath.FromSlash(`Failure: testdata/format/invalid/invalid.proto:4:12: syntax error: unexpected '.', expecting '{'`),
		"format",
		filepath.Join("testdata", "format", "invalid"),
		"-o",
		filepath.Join(tempDir, "formatted", "invalid", "invalid.proto"), // Single file output.
	)
	_, err = os.Stat(filepath.Join(tempDir, "formatted", "invalid"))
	assert.True(t, os.IsNotExist(err))
}

func TestConvertRoundTrip(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("testdata", "success"),
		"-o",
		filepath.Join(tempDir, "image.binpb"),
	)
	t.Run("stdin and stdout", func(t *testing.T) {
		t.Parallel()
		stdin := bytes.NewBufferString(`{"one":"55"}`)
		encodedMessage := bytes.NewBuffer(nil)
		decodedMessage := bytes.NewBuffer(nil)
		testRun(
			t,
			0,
			stdin,
			encodedMessage,
			"convert",
			filepath.Join(tempDir, "image.binpb"),
			"--type",
			"buf.Foo",
			"--from",
			"-#format=json",
		)
		testRun(
			t,
			0,
			encodedMessage,
			decodedMessage,
			"convert",
			filepath.Join(tempDir, "image.binpb"),
			"--type",
			"buf.Foo",
		)
		assert.JSONEq(t, `{"one":"55"}`, decodedMessage.String())
	})
	t.Run("stdin and stdout with type specified", func(t *testing.T) {
		t.Parallel()
		stdin := bytes.NewBufferString(`{"one":"55"}`)
		encodedMessage := bytes.NewBuffer(nil)
		decodedMessage := bytes.NewBuffer(nil)
		testRun(
			t,
			0,
			stdin,
			encodedMessage,
			"convert",
			filepath.Join(tempDir, "image.binpb"),
			"--type",
			"buf.Foo",
			"--from",
			"-#format=json",
			"--to",
			"-#format=binpb",
		)
		testRun(
			t,
			0,
			encodedMessage,
			decodedMessage,
			"convert",
			filepath.Join(tempDir, "image.binpb"),
			"--type",
			"buf.Foo",
			"--from",
			"-#format=binpb",
			"--to",
			"-#format=json",
		)
		assert.JSONEq(t, `{"one":"55"}`, decodedMessage.String())
	})
	t.Run("file output and input", func(t *testing.T) {
		t.Parallel()
		stdin := bytes.NewBufferString(`{"one":"55"}`)
		decodedMessage := bytes.NewBuffer(nil)
		testRun(
			t,
			0,
			stdin,
			nil,
			"convert",
			filepath.Join(tempDir, "image.binpb"),
			"--type",
			"buf.Foo",
			"--from",
			"-#format=json",
			"--to",
			filepath.Join(tempDir, "decoded_message.binpb"),
		)
		testRun(
			t,
			0,
			nil,
			decodedMessage,
			"convert",
			filepath.Join(tempDir, "image.binpb"),
			"--type",
			"buf.Foo",
			"--from",
			filepath.Join(tempDir, "decoded_message.binpb"),
		)
		assert.JSONEq(t, `{"one":"55"}`, decodedMessage.String())
	})
}

func testMigrateV1Beta1Diff(
	t *testing.T,
	storageosProvider storageos.Provider,
	runner command.Runner,
	scenario string,
	expectedStderr string,
) {
	// Copy test setup to temporary directory to avoid writing to filesystem
	inputBucket, err := storageosProvider.NewReadWriteBucket(filepath.Join("testdata", "migrate-v1beta1", "success", scenario, "input"))
	require.NoError(t, err)
	tempDir, readWriteBucket := internaltesting.CopyReadBucketToTempDir(context.Background(), t, storageosProvider, inputBucket)

	testRunStdoutStderrNoWarn(
		t,
		nil,
		0,
		"",
		expectedStderr,
		"beta",
		"migrate-v1beta1",
		tempDir,
	)

	expectedOutputBucket, err := storageosProvider.NewReadWriteBucket(filepath.Join("testdata", "migrate-v1beta1", "success", scenario, "output"))
	require.NoError(t, err)

	diff, err := storage.DiffBytes(context.Background(), runner, expectedOutputBucket, readWriteBucket)
	require.NoError(t, err)
	require.Empty(t, string(diff))
}

func testMigrateV1Beta1Failure(t *testing.T, storageosProvider storageos.Provider, scenario string, expectedStderr string) {
	// Copy test setup to temporary directory to avoid writing to filesystem
	inputBucket, err := storageosProvider.NewReadWriteBucket(filepath.Join("testdata", "migrate-v1beta1", "failure", scenario))
	require.NoError(t, err)
	tempDir, _ := internaltesting.CopyReadBucketToTempDir(context.Background(), t, storageosProvider, inputBucket)

	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		"",
		expectedStderr,
		"beta",
		"migrate-v1beta1",
		tempDir,
	)
}

func testModInit(t *testing.T, expectedData string, document bool, name string, deps ...string) {
	tempDir := t.TempDir()
	baseArgs := []string{"mod", "init"}
	args := append(baseArgs, "-o", tempDir)
	if document {
		args = append(args, "--doc")
	}
	if name != "" {
		args = append(args, "--name", name)
	}
	testRun(t, 0, nil, nil, args...)
	data, err := os.ReadFile(filepath.Join(tempDir, bufconfig.ExternalConfigV1FilePath))
	require.NoError(t, err)
	require.Equal(t, expectedData, string(data))
}

func testRunStdout(t *testing.T, stdin io.Reader, expectedExitCode int, expectedStdout string, args ...string) {
	appcmdtesting.RunCommandExitCodeStdout(
		t,
		func(use string) *appcmd.Command { return NewRootCommand(use) },
		expectedExitCode,
		expectedStdout,
		internaltesting.NewEnvFunc(t),
		stdin,
		args...,
	)
}

func testRunStdoutStderr(t *testing.T, stdin io.Reader, expectedExitCode int, expectedStderr string, args ...string) {
	appcmdtesting.RunCommandExitCodeStderr(
		t,
		func(use string) *appcmd.Command { return NewRootCommand(use) },
		expectedExitCode,
		expectedStderr,
		internaltesting.NewEnvFunc(t),
		stdin,
		args...,
	)
}

func testRunStdoutStderrNoWarn(t *testing.T, stdin io.Reader, expectedExitCode int, expectedStdout string, expectedStderr string, args ...string) {
	appcmdtesting.RunCommandExitCodeStdoutStderr(
		t,
		func(use string) *appcmd.Command { return NewRootCommand(use) },
		expectedExitCode,
		expectedStdout,
		expectedStderr,
		internaltesting.NewEnvFunc(t),
		stdin,
		// we do not want warnings to be part of our stderr test calculation
		append(
			args,
			"--no-warn",
		)...,
	)
}

func testRunStdoutProfile(t *testing.T, stdin io.Reader, expectedExitCode int, expectedStdout string, args ...string) {
	tempDirPath := t.TempDir()
	testRunStdout(
		t,
		stdin,
		0,
		``,
		append(
			args,
			"--profile",
			fmt.Sprintf("--profile-path=%s", tempDirPath),
			"--profile-loops=1",
			"--profile-type=cpu",
		)...,
	)
}

func testRunStdoutFile(t *testing.T, stdin io.Reader, expectedExitCode int, wantFile string, args ...string) {
	wantReader, err := os.Open(wantFile)
	require.NoError(t, err)
	wantBytes, err := io.ReadAll(wantReader)
	require.NoError(t, err)
	testRunStdout(
		t,
		stdin,
		expectedExitCode,
		string(wantBytes),
		args...,
	)
}

func testRun(
	t *testing.T,
	expectedExitCode int,
	stdin io.Reader,
	stdout io.Writer,
	args ...string,
) {
	stderr := bytes.NewBuffer(nil)
	appcmdtesting.RunCommandExitCode(
		t,
		func(use string) *appcmd.Command { return NewRootCommand(use) },
		expectedExitCode,
		internaltesting.NewEnvFunc(t),
		stdin,
		stdout,
		stderr,
		args...,
	)
}

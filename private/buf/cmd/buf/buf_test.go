// Copyright 2020-2025 Buf Technologies, Inc.
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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"buf.build/go/app/appcmd/appcmdtesting"
	"buf.build/go/bufplugin/check"
	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/buf/cmd/buf/internal/internaltesting"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	imagev1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/image/v1"
	"github.com/bufbuild/buf/private/pkg/osext"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/slogtestext"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/storage/storagetesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	convertTestDataDir = filepath.Join("command", "convert", "testdata", "convert")
	// ordered, contains non-default
	builtinLintRulesV2 = []*outputCheckRule{
		{ID: "DIRECTORY_SAME_PACKAGE", Categories: []string{"MINIMAL", "BASIC", "STANDARD"}, Default: true, Purpose: "Checks that all files in a given directory are in the same package."},
		{ID: "PACKAGE_DEFINED", Categories: []string{"MINIMAL", "BASIC", "STANDARD"}, Default: true, Purpose: "Checks that all files have a package defined."},
		{ID: "PACKAGE_DIRECTORY_MATCH", Categories: []string{"MINIMAL", "BASIC", "STANDARD"}, Default: true, Purpose: "Checks that all files are in a directory that matches their package name."},
		{ID: "PACKAGE_NO_IMPORT_CYCLE", Categories: []string{"MINIMAL", "BASIC", "STANDARD"}, Default: true, Purpose: "Checks that packages do not have import cycles."},
		{ID: "PACKAGE_SAME_DIRECTORY", Categories: []string{"MINIMAL", "BASIC", "STANDARD"}, Default: true, Purpose: "Checks that all files with a given package are in the same directory."},
		{ID: "ENUM_FIRST_VALUE_ZERO", Categories: []string{"BASIC", "STANDARD"}, Default: true, Purpose: "Checks that all first values of enums have a numeric value of 0."},
		{ID: "ENUM_NO_ALLOW_ALIAS", Categories: []string{"BASIC", "STANDARD"}, Default: true, Purpose: "Checks that enums do not have the allow_alias option set."},
		{ID: "ENUM_PASCAL_CASE", Categories: []string{"BASIC", "STANDARD"}, Default: true, Purpose: "Checks that enums are PascalCase."},
		{ID: "ENUM_VALUE_UPPER_SNAKE_CASE", Categories: []string{"BASIC", "STANDARD"}, Default: true, Purpose: "Checks that enum values are UPPER_SNAKE_CASE."},
		{ID: "FIELD_LOWER_SNAKE_CASE", Categories: []string{"BASIC", "STANDARD"}, Default: true, Purpose: "Checks that field names are lower_snake_case."},
		{ID: "FIELD_NOT_REQUIRED", Categories: []string{"BASIC", "STANDARD"}, Default: true, Purpose: "Checks that fields are not configured to be required."},
		{ID: "IMPORT_NO_PUBLIC", Categories: []string{"BASIC", "STANDARD"}, Default: true, Purpose: "Checks that imports are not public."},
		{ID: "IMPORT_USED", Categories: []string{"BASIC", "STANDARD"}, Default: true, Purpose: "Checks that imports are used."},
		{ID: "MESSAGE_PASCAL_CASE", Categories: []string{"BASIC", "STANDARD"}, Default: true, Purpose: "Checks that messages are PascalCase."},
		{ID: "ONEOF_LOWER_SNAKE_CASE", Categories: []string{"BASIC", "STANDARD"}, Default: true, Purpose: "Checks that oneof names are lower_snake_case."},
		{ID: "PACKAGE_LOWER_SNAKE_CASE", Categories: []string{"BASIC", "STANDARD"}, Default: true, Purpose: "Checks that packages are lower_snake.case."},
		{ID: "PACKAGE_SAME_CSHARP_NAMESPACE", Categories: []string{"BASIC", "STANDARD"}, Default: true, Purpose: "Checks that all files with a given package have the same value for the csharp_namespace option."},
		{ID: "PACKAGE_SAME_GO_PACKAGE", Categories: []string{"BASIC", "STANDARD"}, Default: true, Purpose: "Checks that all files with a given package have the same value for the go_package option."},
		{ID: "PACKAGE_SAME_JAVA_MULTIPLE_FILES", Categories: []string{"BASIC", "STANDARD"}, Default: true, Purpose: "Checks that all files with a given package have the same value for the java_multiple_files option."},
		{ID: "PACKAGE_SAME_JAVA_PACKAGE", Categories: []string{"BASIC", "STANDARD"}, Default: true, Purpose: "Checks that all files with a given package have the same value for the java_package option."},
		{ID: "PACKAGE_SAME_PHP_NAMESPACE", Categories: []string{"BASIC", "STANDARD"}, Default: true, Purpose: "Checks that all files with a given package have the same value for the php_namespace option."},
		{ID: "PACKAGE_SAME_RUBY_PACKAGE", Categories: []string{"BASIC", "STANDARD"}, Default: true, Purpose: "Checks that all files with a given package have the same value for the ruby_package option."},
		{ID: "PACKAGE_SAME_SWIFT_PREFIX", Categories: []string{"BASIC", "STANDARD"}, Default: true, Purpose: "Checks that all files with a given package have the same value for the swift_prefix option."},
		{ID: "RPC_PASCAL_CASE", Categories: []string{"BASIC", "STANDARD"}, Default: true, Purpose: "Checks that RPCs are PascalCase."},
		{ID: "SERVICE_PASCAL_CASE", Categories: []string{"BASIC", "STANDARD"}, Default: true, Purpose: "Checks that services are PascalCase."},
		{ID: "SYNTAX_SPECIFIED", Categories: []string{"BASIC", "STANDARD"}, Default: true, Purpose: "Checks that all files have a syntax specified."},
		{ID: "ENUM_VALUE_PREFIX", Categories: []string{"STANDARD"}, Default: true, Purpose: "Checks that enum values are prefixed with ENUM_NAME_UPPER_SNAKE_CASE."},
		{ID: "ENUM_ZERO_VALUE_SUFFIX", Categories: []string{"STANDARD"}, Default: true, Purpose: "Checks that enum zero values have a consistent suffix (configurable, default suffix is \"_UNSPECIFIED\")."},
		{ID: "FILE_LOWER_SNAKE_CASE", Categories: []string{"STANDARD"}, Default: true, Purpose: "Checks that filenames are lower_snake_case."},
		{ID: "PACKAGE_VERSION_SUFFIX", Categories: []string{"STANDARD"}, Default: true, Purpose: "Checks that the last component of all packages is a version of the form v\\d+, v\\d+test.*, v\\d+(alpha|beta)\\d+, or v\\d+p\\d+(alpha|beta)\\d+, where numbers are >=1."},
		{ID: "PROTOVALIDATE", Categories: []string{"STANDARD"}, Default: true, Purpose: "Checks that protovalidate rules are valid and all CEL expressions compile."},
		{ID: "RPC_REQUEST_RESPONSE_UNIQUE", Categories: []string{"STANDARD"}, Default: true, Purpose: "Checks that RPC request and response types are only used in one RPC (configurable)."},
		{ID: "RPC_REQUEST_STANDARD_NAME", Categories: []string{"STANDARD"}, Default: true, Purpose: "Checks that RPC request type names are RPCNameRequest or ServiceNameRPCNameRequest (configurable)."},
		{ID: "RPC_RESPONSE_STANDARD_NAME", Categories: []string{"STANDARD"}, Default: true, Purpose: "Checks that RPC response type names are RPCNameResponse or ServiceNameRPCNameResponse (configurable)."},
		{ID: "SERVICE_SUFFIX", Categories: []string{"STANDARD"}, Default: true, Purpose: "Checks that services have a consistent suffix (configurable, default suffix is \"Service\")."},
		{ID: "COMMENT_ENUM", Categories: []string{"COMMENTS"}, Default: false, Purpose: "Checks that enums have non-empty comments."},
		{ID: "COMMENT_ENUM_VALUE", Categories: []string{"COMMENTS"}, Default: false, Purpose: "Checks that enum values have non-empty comments."},
		{ID: "COMMENT_FIELD", Categories: []string{"COMMENTS"}, Default: false, Purpose: "Checks that fields have non-empty comments."},
		{ID: "COMMENT_MESSAGE", Categories: []string{"COMMENTS"}, Default: false, Purpose: "Checks that messages have non-empty comments."},
		{ID: "COMMENT_ONEOF", Categories: []string{"COMMENTS"}, Default: false, Purpose: "Checks that oneofs have non-empty comments."},
		{ID: "COMMENT_RPC", Categories: []string{"COMMENTS"}, Default: false, Purpose: "Checks that RPCs have non-empty comments."},
		{ID: "COMMENT_SERVICE", Categories: []string{"COMMENTS"}, Default: false, Purpose: "Checks that services have non-empty comments."},
		{ID: "RPC_NO_CLIENT_STREAMING", Categories: []string{"UNARY_RPC"}, Default: false, Purpose: "Checks that RPCs are not client streaming."},
		{ID: "RPC_NO_SERVER_STREAMING", Categories: []string{"UNARY_RPC"}, Default: false, Purpose: "Checks that RPCs are not server streaming."},
		{ID: "STABLE_PACKAGE_NO_IMPORT_UNSTABLE", Categories: []string{}, Default: false, Purpose: "Checks that all files that have stable versioned packages do not import packages with unstable version packages."},
	}
	// ordered, contains non-default
	builtinBreakingRulesV2 = []*outputCheckRule{
		{ID: "EXTENSION_NO_DELETE", Categories: []string{"FILE"}, Default: true, Purpose: "Checks that extensions are not deleted from a given file."},
		{ID: "SERVICE_NO_DELETE", Categories: []string{"FILE"}, Default: true, Purpose: "Checks that services are not deleted from a given file."},
		{ID: "ENUM_NO_DELETE", Categories: []string{"CSR", "FILE"}, Default: true, Purpose: "Checks that enums are not deleted from a given file."},
		{ID: "FILE_NO_DELETE", Categories: []string{"CSR", "FILE"}, Default: true, Purpose: "Checks that files are not deleted."},
		{ID: "MESSAGE_NO_DELETE", Categories: []string{"CSR", "FILE"}, Default: true, Purpose: "Checks that messages are not deleted from a given file."},
		{ID: "ENUM_SAME_TYPE", Categories: []string{"FILE", "PACKAGE"}, Default: true, Purpose: "Checks that enums have the same type (open vs closed)."},
		{ID: "FIELD_SAME_CARDINALITY", Categories: []string{"FILE", "PACKAGE"}, Default: true, Purpose: "Checks that fields have the same cardinalities in a given message."},
		{ID: "FIELD_SAME_CPP_STRING_TYPE", Categories: []string{"FILE", "PACKAGE"}, Default: true, Purpose: "Checks that fields have the same C++ string type, based on ctype field option or (pb.cpp).string_type feature."},
		{ID: "FIELD_SAME_JAVA_UTF8_VALIDATION", Categories: []string{"FILE", "PACKAGE"}, Default: true, Purpose: "Checks that fields have the same Java string UTF8 validation, based on java_string_check_utf8 file option or (pb.java).utf8_validation feature."},
		{ID: "FIELD_SAME_JSTYPE", Categories: []string{"FILE", "PACKAGE"}, Default: true, Purpose: "Checks that fields have the same value for the jstype option."},
		{ID: "FIELD_SAME_UTF8_VALIDATION", Categories: []string{"FILE", "PACKAGE"}, Default: true, Purpose: "Checks that string fields have the same UTF8 validation mode."},
		{ID: "FILE_SAME_CC_ENABLE_ARENAS", Categories: []string{"FILE", "PACKAGE"}, Default: true, Purpose: "Checks that files have the same value for the cc_enable_arenas option."},
		{ID: "FILE_SAME_CC_GENERIC_SERVICES", Categories: []string{"FILE", "PACKAGE"}, Default: true, Purpose: "Checks that files have the same value for the cc_generic_services option."},
		{ID: "FILE_SAME_CSHARP_NAMESPACE", Categories: []string{"FILE", "PACKAGE"}, Default: true, Purpose: "Checks that files have the same value for the csharp_namespace option."},
		{ID: "FILE_SAME_GO_PACKAGE", Categories: []string{"FILE", "PACKAGE"}, Default: true, Purpose: "Checks that files have the same value for the go_package option."},
		{ID: "FILE_SAME_JAVA_GENERIC_SERVICES", Categories: []string{"FILE", "PACKAGE"}, Default: true, Purpose: "Checks that files have the same value for the java_generic_services option."},
		{ID: "FILE_SAME_JAVA_MULTIPLE_FILES", Categories: []string{"FILE", "PACKAGE"}, Default: true, Purpose: "Checks that files have the same value for the java_multiple_files option."},
		{ID: "FILE_SAME_JAVA_OUTER_CLASSNAME", Categories: []string{"FILE", "PACKAGE"}, Default: true, Purpose: "Checks that files have the same value for the java_outer_classname option."},
		{ID: "FILE_SAME_JAVA_PACKAGE", Categories: []string{"FILE", "PACKAGE"}, Default: true, Purpose: "Checks that files have the same value for the java_package option."},
		{ID: "FILE_SAME_OBJC_CLASS_PREFIX", Categories: []string{"FILE", "PACKAGE"}, Default: true, Purpose: "Checks that files have the same value for the objc_class_prefix option."},
		{ID: "FILE_SAME_OPTIMIZE_FOR", Categories: []string{"FILE", "PACKAGE"}, Default: true, Purpose: "Checks that files have the same value for the optimize_for option."},
		{ID: "FILE_SAME_PHP_CLASS_PREFIX", Categories: []string{"FILE", "PACKAGE"}, Default: true, Purpose: "Checks that files have the same value for the php_class_prefix option."},
		{ID: "FILE_SAME_PHP_METADATA_NAMESPACE", Categories: []string{"FILE", "PACKAGE"}, Default: true, Purpose: "Checks that files have the same value for the php_metadata_namespace option."},
		{ID: "FILE_SAME_PHP_NAMESPACE", Categories: []string{"FILE", "PACKAGE"}, Default: true, Purpose: "Checks that files have the same value for the php_namespace option."},
		{ID: "FILE_SAME_PY_GENERIC_SERVICES", Categories: []string{"FILE", "PACKAGE"}, Default: true, Purpose: "Checks that files have the same value for the py_generic_services option."},
		{ID: "FILE_SAME_RUBY_PACKAGE", Categories: []string{"FILE", "PACKAGE"}, Default: true, Purpose: "Checks that files have the same value for the ruby_package option."},
		{ID: "FILE_SAME_SWIFT_PREFIX", Categories: []string{"FILE", "PACKAGE"}, Default: true, Purpose: "Checks that files have the same value for the swift_prefix option."},
		{ID: "RPC_NO_DELETE", Categories: []string{"FILE", "PACKAGE"}, Default: true, Purpose: "Checks that rpcs are not deleted from a given service."},
		{ID: "ENUM_VALUE_NO_DELETE", Categories: []string{"CSR", "FILE", "PACKAGE"}, Default: true, Purpose: "Checks that enum values are not deleted from a given enum."},
		{ID: "EXTENSION_MESSAGE_NO_DELETE", Categories: []string{"CSR", "FILE", "PACKAGE"}, Default: true, Purpose: "Checks that extension ranges are not deleted from a given message."},
		{ID: "FIELD_NO_DELETE", Categories: []string{"CSR", "FILE", "PACKAGE"}, Default: true, Purpose: "Checks that fields are not deleted from a given message."},
		{ID: "FIELD_SAME_TYPE", Categories: []string{"CSR", "FILE", "PACKAGE"}, Default: true, Purpose: "Checks that fields have the same types in a given message."},
		{ID: "FILE_SAME_SYNTAX", Categories: []string{"CSR", "FILE", "PACKAGE"}, Default: true, Purpose: "Checks that files have the same syntax."},
		{ID: "MESSAGE_NO_REMOVE_STANDARD_DESCRIPTOR_ACCESSOR", Categories: []string{"CSR", "FILE", "PACKAGE"}, Default: true, Purpose: "Checks that messages do not change the no_standard_descriptor_accessor option from false or unset to true."},
		{ID: "ONEOF_NO_DELETE", Categories: []string{"CSR", "FILE", "PACKAGE"}, Default: true, Purpose: "Checks that oneofs are not deleted from a given message."},
		{ID: "ENUM_SAME_JSON_FORMAT", Categories: []string{"CSR", "FILE", "PACKAGE", "WIRE_JSON"}, Default: true, Purpose: "Checks that enums have the same JSON format support."},
		{ID: "ENUM_VALUE_SAME_NAME", Categories: []string{"CSR", "FILE", "PACKAGE", "WIRE_JSON"}, Default: true, Purpose: "Checks that enum values have the same name."},
		{ID: "FIELD_SAME_JSON_NAME", Categories: []string{"CSR", "FILE", "PACKAGE", "WIRE_JSON"}, Default: true, Purpose: "Checks that fields have the same value for the json_name option."},
		{ID: "FIELD_SAME_NAME", Categories: []string{"CSR", "FILE", "PACKAGE", "WIRE_JSON"}, Default: true, Purpose: "Checks that fields have the same names in a given message."},
		{ID: "MESSAGE_SAME_JSON_FORMAT", Categories: []string{"CSR", "FILE", "PACKAGE", "WIRE_JSON"}, Default: true, Purpose: "Checks that messages have the same JSON format support."},
		{ID: "FIELD_SAME_DEFAULT", Categories: []string{"CSR", "FILE", "PACKAGE", "WIRE_JSON", "WIRE"}, Default: true, Purpose: "Checks that fields have the same default value, if a default is specified."},
		{ID: "FIELD_SAME_ONEOF", Categories: []string{"CSR", "FILE", "PACKAGE", "WIRE_JSON", "WIRE"}, Default: true, Purpose: "Checks that fields have the same oneofs in a given message."},
		{ID: "FILE_SAME_PACKAGE", Categories: []string{"CSR", "FILE", "PACKAGE", "WIRE_JSON", "WIRE"}, Default: true, Purpose: "Checks that files have the same package."},
		{ID: "MESSAGE_SAME_REQUIRED_FIELDS", Categories: []string{"CSR", "FILE", "PACKAGE", "WIRE_JSON", "WIRE"}, Default: true, Purpose: "Checks that messages have no added or deleted required fields."},
		{ID: "RESERVED_ENUM_NO_DELETE", Categories: []string{"CSR", "FILE", "PACKAGE", "WIRE_JSON", "WIRE"}, Default: true, Purpose: "Checks that reserved ranges and names are not deleted from a given enum."},
		{ID: "RESERVED_MESSAGE_NO_DELETE", Categories: []string{"CSR", "FILE", "PACKAGE", "WIRE_JSON", "WIRE"}, Default: true, Purpose: "Checks that reserved ranges and names are not deleted from a given message."},
		{ID: "RPC_SAME_CLIENT_STREAMING", Categories: []string{"CSR", "FILE", "PACKAGE", "WIRE_JSON", "WIRE"}, Default: true, Purpose: "Checks that rpcs have the same client streaming value."},
		{ID: "RPC_SAME_IDEMPOTENCY_LEVEL", Categories: []string{"CSR", "FILE", "PACKAGE", "WIRE_JSON", "WIRE"}, Default: true, Purpose: "Checks that rpcs have the same value for the idempotency_level option."},
		{ID: "RPC_SAME_REQUEST_TYPE", Categories: []string{"CSR", "FILE", "PACKAGE", "WIRE_JSON", "WIRE"}, Default: true, Purpose: "Checks that rpcs are have the same request type."},
		{ID: "RPC_SAME_RESPONSE_TYPE", Categories: []string{"CSR", "FILE", "PACKAGE", "WIRE_JSON", "WIRE"}, Default: true, Purpose: "Checks that rpcs are have the same response type."},
		{ID: "RPC_SAME_SERVER_STREAMING", Categories: []string{"CSR", "FILE", "PACKAGE", "WIRE_JSON", "WIRE"}, Default: true, Purpose: "Checks that rpcs have the same server streaming value."},
		{ID: "PACKAGE_ENUM_NO_DELETE", Categories: []string{"PACKAGE"}, Default: false, Purpose: "Checks that enums are not deleted from a given package."},
		{ID: "PACKAGE_EXTENSION_NO_DELETE", Categories: []string{"PACKAGE"}, Default: false, Purpose: "Checks that extensions are not deleted from a given package."},
		{ID: "PACKAGE_MESSAGE_NO_DELETE", Categories: []string{"PACKAGE"}, Default: false, Purpose: "Checks that messages are not deleted from a given package."},
		{ID: "PACKAGE_NO_DELETE", Categories: []string{"PACKAGE"}, Default: false, Purpose: "Checks that packages are not deleted."},
		{ID: "PACKAGE_SERVICE_NO_DELETE", Categories: []string{"PACKAGE"}, Default: false, Purpose: "Checks that services are not deleted from a given package."},
		{ID: "ENUM_VALUE_NO_DELETE_UNLESS_NAME_RESERVED", Categories: []string{"CSR", "WIRE_JSON"}, Default: false, Purpose: "Checks that enum values are not deleted from a given enum unless the name is reserved."},
		{ID: "FIELD_NO_DELETE_UNLESS_NAME_RESERVED", Categories: []string{"CSR", "WIRE_JSON"}, Default: false, Purpose: "Checks that fields are not deleted from a given message unless the name is reserved."},
		{ID: "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY", Categories: []string{"CSR", "WIRE_JSON"}, Default: false, Purpose: "Checks that fields have wire and JSON compatible cardinalities in a given message."},
		{ID: "FIELD_WIRE_JSON_COMPATIBLE_TYPE", Categories: []string{"CSR", "WIRE_JSON"}, Default: false, Purpose: "Checks that fields have wire and JSON compatible types in a given message."},
		{ID: "ENUM_VALUE_NO_DELETE_UNLESS_NUMBER_RESERVED", Categories: []string{"CSR", "WIRE_JSON", "WIRE"}, Default: false, Purpose: "Checks that enum values are not deleted from a given enum unless the number is reserved."},
		{ID: "FIELD_NO_DELETE_UNLESS_NUMBER_RESERVED", Categories: []string{"CSR", "WIRE_JSON", "WIRE"}, Default: false, Purpose: "Checks that fields are not deleted from a given message unless the number is reserved."},
		{ID: "FIELD_WIRE_COMPATIBLE_CARDINALITY", Categories: []string{"WIRE"}, Default: false, Purpose: "Checks that fields have wire-compatible cardinalities in a given message."},
		{ID: "FIELD_WIRE_COMPATIBLE_TYPE", Categories: []string{"WIRE"}, Default: false, Purpose: "Checks that fields have wire-compatible types in a given message."},
	}
)

type outputCheckRule struct {
	ID           string   `json:"id"`
	Categories   []string `json:"categories"`
	Default      bool     `json:"default"`
	Purpose      string   `json:"purpose"`
	Plugin       string   `json:"plugin"`
	Deprecated   bool     `json:"deprecated"`
	Replacements []string `json:"replacements"`
}

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

func TestSuccessDir(t *testing.T) {
	t.Parallel()
	testRunStdout(t, nil, 0, ``, "build", filepath.Join("testdata", "successnobufyaml"))
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("testdata", "successnobufyaml"),
		"--path",
		filepath.Join("testdata", "successnobufyaml", "buf", "buf.proto"),
	)
	wd, err := osext.Getwd()
	require.NoError(t, err)
	testRunStdout(t, nil, 0, ``, "build", filepath.Join(wd, "testdata", "successnobufyaml"))
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join(wd, "testdata", "successnobufyaml"),
		"--path",
		filepath.Join(wd, "testdata", "successnobufyaml", "buf", "buf.proto"),
	)
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
		bufctl.ExitCodeFileAnnotation,
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
		bufctl.ExitCodeFileAnnotation,
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
		bufctl.ExitCodeFileAnnotation,
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
		"Failure: --path is not valid for use with .proto file references",
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
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/fail/buf/buf.proto:3:1:Files with package "other" must be within a directory "other" relative to root but were in directory "fail/buf".
testdata/fail/buf/buf.proto:6:9:Field name "oneTwo" should be lower_snake_case, such as "one_two".`),
		// This is new behavior we introduced. When setting a config override, we no longer do
		// a search for the controlling workspace. See bufctl/option.go for additional details.
		// Only the paths specified by "--path" in the command are considered. This avoids build
		// failures from other proto files under testdata. Command "build" succeeds with this path
		// restriction, "lint" should be able to build the image and only fail on lint issue for
		// the specified file.
		"",
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
		bufctl.ExitCodeFileAnnotation,
		// Note: `were in directory "buf"` was changed to `were in directory "testdata/fail/buf"`
		// during the refactor. This is actually more correct - pre-refactor, the CLI was acting
		// as if the buf.yaml at testdata/fail/buf.yaml mattered in some way. In fact, it doesn't -
		// you've said that you have overridden it entirely.
		filepath.FromSlash(`testdata/fail/buf/buf.proto:3:1:Files with package "other" must be within a directory "other" relative to root but were in directory ".".
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
		bufctl.ExitCodeFileAnnotation,
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
		bufctl.ExitCodeFileAnnotation,
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
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/fail2/buf/buf2.proto:9:9:Field name "oneThree" should be lower_snake_case, such as "one_three".`),
		"lint",
		"--path",
		filepath.Join("testdata", "fail2", "buf", "buf2.proto"),
		filepath.Join("testdata"),
	)
	testRunStdout(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
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
		bufctl.ExitCodeFileAnnotation,
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
		bufctl.ExitCodeFileAnnotation,
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
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`
		../../../bufpkg/bufcheck/testdata/breaking/current/breaking_field_no_delete/1.proto:5:1:Previously present field "3" with name "three" on message "Two" was deleted.
		../../../bufpkg/bufcheck/testdata/breaking/current/breaking_field_no_delete/1.proto:10:1:Previously present field "3" with name "three" on message "Three" was deleted.
		../../../bufpkg/bufcheck/testdata/breaking/current/breaking_field_no_delete/1.proto:12:5:Previously present field "3" with name "three" on message "Five" was deleted.
		../../../bufpkg/bufcheck/testdata/breaking/current/breaking_field_no_delete/1.proto:22:3:Previously present field "3" with name "three" on message "Seven" was deleted.
		../../../bufpkg/bufcheck/testdata/breaking/current/breaking_field_no_delete/2.proto:57:1:Previously present field "3" with name "three" on message "Nine" was deleted.
		`),
		"", // stderr should be empty
		"breaking",
		// can't bother right now to filepath.Join this
		"../../../bufpkg/bufcheck/testdata/breaking/current/breaking_field_no_delete",
		"--against",
		"../../../bufpkg/bufcheck/testdata/breaking/previous/breaking_field_no_delete",
	)
}

func TestFailCheckBreaking2(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/protofileref/breaking/a/foo.proto:7:3:Field "2" with name "world" on message "Foo" changed type from "int32" to "string".`),
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
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`
		<input>:1:1:Previously present file "bar.proto" was deleted.
		testdata/protofileref/breaking/a/foo.proto:7:3:Field "2" with name "world" on message "Foo" changed type from "int32" to "string".
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
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`
		testdata/protofileref/breaking/a/bar.proto:5:1:Previously present field "2" with name "value" on message "Bar" was deleted.
		testdata/protofileref/breaking/a/foo.proto:7:3:Field "2" with name "world" on message "Foo" changed type from "int32" to "string".
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
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`
    <input>:1:1:Previously present file "bar.proto" was deleted.
		testdata/protofileref/breaking/a/foo.proto:7:3:Field "2" with name "world" on message "Foo" changed type from "int32" to "string".
		`),
		"breaking",
		filepath.Join("testdata", "protofileref", "breaking", "a", "foo.proto"),
		"--against",
		fmt.Sprintf("%s#include_package_files=true", filepath.Join("testdata", "protofileref", "breaking", "b", "foo.proto")),
	)
}

func TestCheckLsLintRulesModAll(t *testing.T) {
	t.Parallel()
	expectedStdout := `
ID                                CATEGORIES                DEFAULT  PURPOSE
DIRECTORY_SAME_PACKAGE            MINIMAL, BASIC, STANDARD  *        Checks that all files in a given directory are in the same package.
PACKAGE_DEFINED                   MINIMAL, BASIC, STANDARD  *        Checks that all files have a package defined.
PACKAGE_DIRECTORY_MATCH           MINIMAL, BASIC, STANDARD  *        Checks that all files are in a directory that matches their package name.
PACKAGE_SAME_DIRECTORY            MINIMAL, BASIC, STANDARD  *        Checks that all files with a given package are in the same directory.
ENUM_FIRST_VALUE_ZERO             BASIC, STANDARD           *        Checks that all first values of enums have a numeric value of 0.
ENUM_NO_ALLOW_ALIAS               BASIC, STANDARD           *        Checks that enums do not have the allow_alias option set.
ENUM_PASCAL_CASE                  BASIC, STANDARD           *        Checks that enums are PascalCase.
ENUM_VALUE_UPPER_SNAKE_CASE       BASIC, STANDARD           *        Checks that enum values are UPPER_SNAKE_CASE.
FIELD_LOWER_SNAKE_CASE            BASIC, STANDARD           *        Checks that field names are lower_snake_case.
IMPORT_NO_PUBLIC                  BASIC, STANDARD           *        Checks that imports are not public.
IMPORT_USED                       BASIC, STANDARD           *        Checks that imports are used.
MESSAGE_PASCAL_CASE               BASIC, STANDARD           *        Checks that messages are PascalCase.
ONEOF_LOWER_SNAKE_CASE            BASIC, STANDARD           *        Checks that oneof names are lower_snake_case.
PACKAGE_LOWER_SNAKE_CASE          BASIC, STANDARD           *        Checks that packages are lower_snake.case.
PACKAGE_SAME_CSHARP_NAMESPACE     BASIC, STANDARD           *        Checks that all files with a given package have the same value for the csharp_namespace option.
PACKAGE_SAME_GO_PACKAGE           BASIC, STANDARD           *        Checks that all files with a given package have the same value for the go_package option.
PACKAGE_SAME_JAVA_MULTIPLE_FILES  BASIC, STANDARD           *        Checks that all files with a given package have the same value for the java_multiple_files option.
PACKAGE_SAME_JAVA_PACKAGE         BASIC, STANDARD           *        Checks that all files with a given package have the same value for the java_package option.
PACKAGE_SAME_PHP_NAMESPACE        BASIC, STANDARD           *        Checks that all files with a given package have the same value for the php_namespace option.
PACKAGE_SAME_RUBY_PACKAGE         BASIC, STANDARD           *        Checks that all files with a given package have the same value for the ruby_package option.
PACKAGE_SAME_SWIFT_PREFIX         BASIC, STANDARD           *        Checks that all files with a given package have the same value for the swift_prefix option.
RPC_PASCAL_CASE                   BASIC, STANDARD           *        Checks that RPCs are PascalCase.
SERVICE_PASCAL_CASE               BASIC, STANDARD           *        Checks that services are PascalCase.
SYNTAX_SPECIFIED                  BASIC, STANDARD           *        Checks that all files have a syntax specified.
ENUM_VALUE_PREFIX                 STANDARD                  *        Checks that enum values are prefixed with ENUM_NAME_UPPER_SNAKE_CASE.
ENUM_ZERO_VALUE_SUFFIX            STANDARD                  *        Checks that enum zero values have a consistent suffix (configurable, default suffix is "_UNSPECIFIED").
FILE_LOWER_SNAKE_CASE             STANDARD                  *        Checks that filenames are lower_snake_case.
PACKAGE_VERSION_SUFFIX            STANDARD                  *        Checks that the last component of all packages is a version of the form v\d+, v\d+test.*, v\d+(alpha|beta)\d+, or v\d+p\d+(alpha|beta)\d+, where numbers are >=1.
PROTOVALIDATE                     STANDARD                  *        Checks that protovalidate rules are valid and all CEL expressions compile.
RPC_REQUEST_RESPONSE_UNIQUE       STANDARD                  *        Checks that RPC request and response types are only used in one RPC (configurable).
RPC_REQUEST_STANDARD_NAME         STANDARD                  *        Checks that RPC request type names are RPCNameRequest or ServiceNameRPCNameRequest (configurable).
RPC_RESPONSE_STANDARD_NAME        STANDARD                  *        Checks that RPC response type names are RPCNameResponse or ServiceNameRPCNameResponse (configurable).
SERVICE_SUFFIX                    STANDARD                  *        Checks that services have a consistent suffix (configurable, default suffix is "Service").
COMMENT_ENUM                      COMMENTS                           Checks that enums have non-empty comments.
COMMENT_ENUM_VALUE                COMMENTS                           Checks that enum values have non-empty comments.
COMMENT_FIELD                     COMMENTS                           Checks that fields have non-empty comments.
COMMENT_MESSAGE                   COMMENTS                           Checks that messages have non-empty comments.
COMMENT_ONEOF                     COMMENTS                           Checks that oneofs have non-empty comments.
COMMENT_RPC                       COMMENTS                           Checks that RPCs have non-empty comments.
COMMENT_SERVICE                   COMMENTS                           Checks that services have non-empty comments.
RPC_NO_CLIENT_STREAMING           UNARY_RPC                          Checks that RPCs are not client streaming.
RPC_NO_SERVER_STREAMING           UNARY_RPC                          Checks that RPCs are not server streaming.
PACKAGE_NO_IMPORT_CYCLE                                              Checks that packages do not have import cycles.
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

func TestCheckPolicyConfigLsLintRulesConfigured(t *testing.T) {
	t.Parallel()
	expectedConfiguredOnlyStdout, err := os.ReadFile(filepath.Join("testdata", "policy_list_rules", "expected_ls_lint_rules_configured_only.txt"))
	require.NoError(t, err)
	testRunStdout(
		t,
		nil,
		0,
		string(expectedConfiguredOnlyStdout),
		"config",
		"ls-lint-rules",
		"--config",
		filepath.Join("testdata", "policy_list_rules", "buf.yaml"),
		"--configured-only",
	)
}

func TestCheckLsLintRulesFromConfig(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		0,
		`
		ID                       CATEGORIES                             DEFAULT  PURPOSE
		PACKAGE_DIRECTORY_MATCH  MINIMAL, BASIC, STANDARD, FILE_LAYOUT  *        Checks that all files are in a directory that matches their package name.
		ENUM_NO_ALLOW_ALIAS      MINIMAL, BASIC, STANDARD, SENSIBLE     *        Checks that enums do not have the allow_alias option set.
		`,
		"mod",
		"ls-lint-rules",
		"--config",
		filepath.Join("testdata", "small_list_rules", "buf.yaml"),
	)
	// defaults only, built-ins and plugins.
	testLsRuleOutputJSON(
		t,
		check.RuleTypeLint,
		`{
			"version":"v2",
			"plugins":[{"plugin": "buf-plugin-suffix"}]
		}`,
		append(
			xslices.Filter(builtinLintRulesV2, func(lintRule *outputCheckRule) bool {
				return lintRule.Default
			}),
			&outputCheckRule{ID: "RPC_BANNED_SUFFIXES", Categories: []string{"OPERATION_SUFFIXES"}, Default: true, Purpose: "Ensure that there are no RPCs with the list of configured banned suffixes.", Plugin: "buf-plugin-suffix"},
			&outputCheckRule{ID: "SERVICE_BANNED_SUFFIXES", Categories: []string{"OPERATION_SUFFIXES"}, Default: true, Purpose: "Ensure that there are no services with the list of configured banned suffixes.", Plugin: "buf-plugin-suffix"},
		),
	)
	// configure a deprecated category and a non-deprecated built-in category.
	// deprecated category contains some non-deprecated rules.
	testLsRuleOutputJSON(
		t,
		check.RuleTypeLint,
		`{
			"version":"v2",
			"lint": {
				"use": ["MINIMAL", "RESOURCE_SUFFIXES"],
			},
			"plugins":[{"plugin": "buf-plugin-suffix"}]
		}`,
		[]*outputCheckRule{
			{ID: "DIRECTORY_SAME_PACKAGE", Categories: []string{"MINIMAL", "BASIC", "STANDARD"}, Default: true, Purpose: "Checks that all files in a given directory are in the same package."},
			{ID: "PACKAGE_DEFINED", Categories: []string{"MINIMAL", "BASIC", "STANDARD"}, Default: true, Purpose: "Checks that all files have a package defined."},
			{ID: "PACKAGE_DIRECTORY_MATCH", Categories: []string{"MINIMAL", "BASIC", "STANDARD"}, Default: true, Purpose: "Checks that all files are in a directory that matches their package name."},
			{ID: "PACKAGE_NO_IMPORT_CYCLE", Categories: []string{"MINIMAL", "BASIC", "STANDARD"}, Default: true, Purpose: "Checks that packages do not have import cycles."},
			{ID: "PACKAGE_SAME_DIRECTORY", Categories: []string{"MINIMAL", "BASIC", "STANDARD"}, Default: true, Purpose: "Checks that all files with a given package are in the same directory."},
			{ID: "ENUM_VALUE_BANNED_SUFFIXES", Categories: []string{"ATTRIBUTES_SUFFIXES"}, Default: false, Purpose: "Ensure that there are no enum values of top-level enums with the list of configured banned suffixes.", Plugin: "buf-plugin-suffix"},
			{ID: "FIELD_BANNED_SUFFIXES", Categories: []string{"ATTRIBUTES_SUFFIXES"}, Default: false, Purpose: "Ensure that there are no fields with the list of configured banned suffixes.", Plugin: "buf-plugin-suffix"},
		},
	)
	// configure a deprecated category and a non-deprecated category, no built-ins.
	testLsRuleOutputJSON(
		t,
		check.RuleTypeLint,
		`{
			"version":"v2",
			"lint": {
				"use": ["OPERATION_SUFFIXES","ATTRIBUTES_SUFFIXES", "RESOURCE_SUFFIXES"],
			},
			"plugins":[{"plugin": "buf-plugin-suffix"}]
		}`,
		[]*outputCheckRule{
			{ID: "RPC_BANNED_SUFFIXES", Categories: []string{"OPERATION_SUFFIXES"}, Default: true, Purpose: "Ensure that there are no RPCs with the list of configured banned suffixes.", Plugin: "buf-plugin-suffix"},
			{ID: "SERVICE_BANNED_SUFFIXES", Categories: []string{"OPERATION_SUFFIXES"}, Default: true, Purpose: "Ensure that there are no services with the list of configured banned suffixes.", Plugin: "buf-plugin-suffix"},
			{ID: "ENUM_VALUE_BANNED_SUFFIXES", Categories: []string{"ATTRIBUTES_SUFFIXES"}, Default: false, Purpose: "Ensure that there are no enum values of top-level enums with the list of configured banned suffixes.", Plugin: "buf-plugin-suffix"},
			{ID: "FIELD_BANNED_SUFFIXES", Categories: []string{"ATTRIBUTES_SUFFIXES"}, Default: false, Purpose: "Ensure that there are no fields with the list of configured banned suffixes.", Plugin: "buf-plugin-suffix"},
		},
	)
	// configure a deprecated category and a non-deprecated category, no built-ins.
	// note: ATTRIBUTES_SUFFIXES is not in USE, but is the replacement category for
	// RESOURCE_SUFFIXES.
	testLsRuleOutputJSON(
		t,
		check.RuleTypeLint,
		`{
			"version":"v2",
			"lint": {
				"use": ["OPERATION_SUFFIXES", "RESOURCE_SUFFIXES"],
			},
			"plugins":[{"plugin": "buf-plugin-suffix"}]
		}`,
		[]*outputCheckRule{
			{ID: "RPC_BANNED_SUFFIXES", Categories: []string{"OPERATION_SUFFIXES"}, Default: true, Purpose: "Ensure that there are no RPCs with the list of configured banned suffixes.", Plugin: "buf-plugin-suffix"},
			{ID: "SERVICE_BANNED_SUFFIXES", Categories: []string{"OPERATION_SUFFIXES"}, Default: true, Purpose: "Ensure that there are no services with the list of configured banned suffixes.", Plugin: "buf-plugin-suffix"},
			{ID: "ENUM_VALUE_BANNED_SUFFIXES", Categories: []string{"ATTRIBUTES_SUFFIXES"}, Default: false, Purpose: "Ensure that there are no enum values of top-level enums with the list of configured banned suffixes.", Plugin: "buf-plugin-suffix"},
			{ID: "FIELD_BANNED_SUFFIXES", Categories: []string{"ATTRIBUTES_SUFFIXES"}, Default: false, Purpose: "Ensure that there are no fields with the list of configured banned suffixes.", Plugin: "buf-plugin-suffix"},
		},
	)
	// configure a mix of rules from built-ins and plugins. MESSAGE_BANNED_SUFFIXES is a deprecated
	// rule, expect to print its replacement, FIELD_BANNED_SUFFIXES.
	testLsRuleOutputJSON(
		t,
		check.RuleTypeLint,
		`{
			"version":"v2",
			"lint": {
				"use": ["RPC_BANNED_SUFFIXES","SERVICE_SUFFIX", "MESSAGE_BANNED_SUFFIXES"],
			},
			"plugins":[{"plugin": "buf-plugin-suffix"}]
		}`,
		[]*outputCheckRule{
			{ID: "SERVICE_SUFFIX", Categories: []string{"STANDARD"}, Default: true, Purpose: "Checks that services have a consistent suffix (configurable, default suffix is \"Service\")."},
			{ID: "RPC_BANNED_SUFFIXES", Categories: []string{"OPERATION_SUFFIXES"}, Default: true, Purpose: "Ensure that there are no RPCs with the list of configured banned suffixes.", Plugin: "buf-plugin-suffix"},
			{ID: "FIELD_BANNED_SUFFIXES", Categories: []string{"ATTRIBUTES_SUFFIXES"}, Default: false, Purpose: "Ensure that there are no fields with the list of configured banned suffixes.", Plugin: "buf-plugin-suffix"},
		},
	)
	// configure a mix of categories and rules from built-ins and plugins.
	testLsRuleOutputJSON(
		t,
		check.RuleTypeLint,
		`{
			"version":"v2",
			"lint": {
				"use": ["RPC_BANNED_SUFFIXES","SERVICE_SUFFIX", "MINIMAL", "OPERATION_SUFFIXES"],
			},
			"plugins":[{"plugin": "buf-plugin-suffix"}]
		}`,
		[]*outputCheckRule{
			{ID: "DIRECTORY_SAME_PACKAGE", Categories: []string{"MINIMAL", "BASIC", "STANDARD"}, Default: true, Purpose: "Checks that all files in a given directory are in the same package."},
			{ID: "PACKAGE_DEFINED", Categories: []string{"MINIMAL", "BASIC", "STANDARD"}, Default: true, Purpose: "Checks that all files have a package defined."},
			{ID: "PACKAGE_DIRECTORY_MATCH", Categories: []string{"MINIMAL", "BASIC", "STANDARD"}, Default: true, Purpose: "Checks that all files are in a directory that matches their package name."},
			{ID: "PACKAGE_NO_IMPORT_CYCLE", Categories: []string{"MINIMAL", "BASIC", "STANDARD"}, Default: true, Purpose: "Checks that packages do not have import cycles."},
			{ID: "PACKAGE_SAME_DIRECTORY", Categories: []string{"MINIMAL", "BASIC", "STANDARD"}, Default: true, Purpose: "Checks that all files with a given package are in the same directory."},
			{ID: "SERVICE_SUFFIX", Categories: []string{"STANDARD"}, Default: true, Purpose: "Checks that services have a consistent suffix (configurable, default suffix is \"Service\")."},
			{ID: "RPC_BANNED_SUFFIXES", Categories: []string{"OPERATION_SUFFIXES"}, Default: true, Purpose: "Ensure that there are no RPCs with the list of configured banned suffixes.", Plugin: "buf-plugin-suffix"},
			{ID: "SERVICE_BANNED_SUFFIXES", Categories: []string{"OPERATION_SUFFIXES"}, Default: true, Purpose: "Ensure that there are no services with the list of configured banned suffixes.", Plugin: "buf-plugin-suffix"},
		},
	)
}

func TestCheckLsLintRulesV1Beta1(t *testing.T) {
	t.Parallel()
	expectedStdout := `
ID                                CATEGORIES                                    DEFAULT  PURPOSE
DIRECTORY_SAME_PACKAGE            MINIMAL, BASIC, STANDARD, FILE_LAYOUT         *        Checks that all files in a given directory are in the same package.
PACKAGE_DIRECTORY_MATCH           MINIMAL, BASIC, STANDARD, FILE_LAYOUT         *        Checks that all files are in a directory that matches their package name.
PACKAGE_SAME_DIRECTORY            MINIMAL, BASIC, STANDARD, FILE_LAYOUT         *        Checks that all files with a given package are in the same directory.
PACKAGE_SAME_CSHARP_NAMESPACE     MINIMAL, BASIC, STANDARD, PACKAGE_AFFINITY    *        Checks that all files with a given package have the same value for the csharp_namespace option.
PACKAGE_SAME_GO_PACKAGE           MINIMAL, BASIC, STANDARD, PACKAGE_AFFINITY    *        Checks that all files with a given package have the same value for the go_package option.
PACKAGE_SAME_JAVA_MULTIPLE_FILES  MINIMAL, BASIC, STANDARD, PACKAGE_AFFINITY    *        Checks that all files with a given package have the same value for the java_multiple_files option.
PACKAGE_SAME_JAVA_PACKAGE         MINIMAL, BASIC, STANDARD, PACKAGE_AFFINITY    *        Checks that all files with a given package have the same value for the java_package option.
PACKAGE_SAME_PHP_NAMESPACE        MINIMAL, BASIC, STANDARD, PACKAGE_AFFINITY    *        Checks that all files with a given package have the same value for the php_namespace option.
PACKAGE_SAME_RUBY_PACKAGE         MINIMAL, BASIC, STANDARD, PACKAGE_AFFINITY    *        Checks that all files with a given package have the same value for the ruby_package option.
PACKAGE_SAME_SWIFT_PREFIX         MINIMAL, BASIC, STANDARD, PACKAGE_AFFINITY    *        Checks that all files with a given package have the same value for the swift_prefix option.
ENUM_NO_ALLOW_ALIAS               MINIMAL, BASIC, STANDARD, SENSIBLE            *        Checks that enums do not have the allow_alias option set.
FIELD_NO_DESCRIPTOR               MINIMAL, BASIC, STANDARD, SENSIBLE            *        Checks that field names are not any capitalization of "descriptor" with any number of prefix or suffix underscores.
IMPORT_NO_PUBLIC                  MINIMAL, BASIC, STANDARD, SENSIBLE            *        Checks that imports are not public.
PACKAGE_DEFINED                   MINIMAL, BASIC, STANDARD, SENSIBLE            *        Checks that all files have a package defined.
ENUM_PASCAL_CASE                  BASIC, STANDARD, STYLE_BASIC, STYLE_STANDARD  *        Checks that enums are PascalCase.
ENUM_VALUE_UPPER_SNAKE_CASE       BASIC, STANDARD, STYLE_BASIC, STYLE_STANDARD  *        Checks that enum values are UPPER_SNAKE_CASE.
FIELD_LOWER_SNAKE_CASE            BASIC, STANDARD, STYLE_BASIC, STYLE_STANDARD  *        Checks that field names are lower_snake_case.
MESSAGE_PASCAL_CASE               BASIC, STANDARD, STYLE_BASIC, STYLE_STANDARD  *        Checks that messages are PascalCase.
ONEOF_LOWER_SNAKE_CASE            BASIC, STANDARD, STYLE_BASIC, STYLE_STANDARD  *        Checks that oneof names are lower_snake_case.
PACKAGE_LOWER_SNAKE_CASE          BASIC, STANDARD, STYLE_BASIC, STYLE_STANDARD  *        Checks that packages are lower_snake.case.
RPC_PASCAL_CASE                   BASIC, STANDARD, STYLE_BASIC, STYLE_STANDARD  *        Checks that RPCs are PascalCase.
SERVICE_PASCAL_CASE               BASIC, STANDARD, STYLE_BASIC, STYLE_STANDARD  *        Checks that services are PascalCase.
ENUM_VALUE_PREFIX                 STANDARD, STYLE_STANDARD                      *        Checks that enum values are prefixed with ENUM_NAME_UPPER_SNAKE_CASE.
ENUM_ZERO_VALUE_SUFFIX            STANDARD, STYLE_STANDARD                      *        Checks that enum zero values have a consistent suffix (configurable, default suffix is "_UNSPECIFIED").
FILE_LOWER_SNAKE_CASE             STANDARD, STYLE_STANDARD                      *        Checks that filenames are lower_snake_case.
PACKAGE_VERSION_SUFFIX            STANDARD, STYLE_STANDARD                      *        Checks that the last component of all packages is a version of the form v\d+, v\d+test.*, v\d+(alpha|beta)\d+, or v\d+p\d+(alpha|beta)\d+, where numbers are >=1.
RPC_REQUEST_RESPONSE_UNIQUE       STANDARD, STYLE_STANDARD                      *        Checks that RPC request and response types are only used in one RPC (configurable).
RPC_REQUEST_STANDARD_NAME         STANDARD, STYLE_STANDARD                      *        Checks that RPC request type names are RPCNameRequest or ServiceNameRPCNameRequest (configurable).
RPC_RESPONSE_STANDARD_NAME        STANDARD, STYLE_STANDARD                      *        Checks that RPC response type names are RPCNameResponse or ServiceNameRPCNameResponse (configurable).
SERVICE_SUFFIX                    STANDARD, STYLE_STANDARD                      *        Checks that services have a consistent suffix (configurable, default suffix is "Service").
COMMENT_ENUM                      COMMENTS                                               Checks that enums have non-empty comments.
COMMENT_ENUM_VALUE                COMMENTS                                               Checks that enum values have non-empty comments.
COMMENT_FIELD                     COMMENTS                                               Checks that fields have non-empty comments.
COMMENT_MESSAGE                   COMMENTS                                               Checks that messages have non-empty comments.
COMMENT_ONEOF                     COMMENTS                                               Checks that oneofs have non-empty comments.
COMMENT_RPC                       COMMENTS                                               Checks that RPCs have non-empty comments.
COMMENT_SERVICE                   COMMENTS                                               Checks that services have non-empty comments.
RPC_NO_CLIENT_STREAMING           UNARY_RPC                                              Checks that RPCs are not client streaming.
RPC_NO_SERVER_STREAMING           UNARY_RPC                                              Checks that RPCs are not server streaming.
ENUM_FIRST_VALUE_ZERO             OTHER                                                  Checks that all first values of enums have a numeric value of 0.
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

func TestCheckLsLintRulesV2(t *testing.T) {
	t.Parallel()
	expectedStdout := `
ID                                 CATEGORIES                DEFAULT  PURPOSE
DIRECTORY_SAME_PACKAGE             MINIMAL, BASIC, STANDARD  *        Checks that all files in a given directory are in the same package.
PACKAGE_DEFINED                    MINIMAL, BASIC, STANDARD  *        Checks that all files have a package defined.
PACKAGE_DIRECTORY_MATCH            MINIMAL, BASIC, STANDARD  *        Checks that all files are in a directory that matches their package name.
PACKAGE_NO_IMPORT_CYCLE            MINIMAL, BASIC, STANDARD  *        Checks that packages do not have import cycles.
PACKAGE_SAME_DIRECTORY             MINIMAL, BASIC, STANDARD  *        Checks that all files with a given package are in the same directory.
ENUM_FIRST_VALUE_ZERO              BASIC, STANDARD           *        Checks that all first values of enums have a numeric value of 0.
ENUM_NO_ALLOW_ALIAS                BASIC, STANDARD           *        Checks that enums do not have the allow_alias option set.
ENUM_PASCAL_CASE                   BASIC, STANDARD           *        Checks that enums are PascalCase.
ENUM_VALUE_UPPER_SNAKE_CASE        BASIC, STANDARD           *        Checks that enum values are UPPER_SNAKE_CASE.
FIELD_LOWER_SNAKE_CASE             BASIC, STANDARD           *        Checks that field names are lower_snake_case.
FIELD_NOT_REQUIRED                 BASIC, STANDARD           *        Checks that fields are not configured to be required.
IMPORT_NO_PUBLIC                   BASIC, STANDARD           *        Checks that imports are not public.
IMPORT_USED                        BASIC, STANDARD           *        Checks that imports are used.
MESSAGE_PASCAL_CASE                BASIC, STANDARD           *        Checks that messages are PascalCase.
ONEOF_LOWER_SNAKE_CASE             BASIC, STANDARD           *        Checks that oneof names are lower_snake_case.
PACKAGE_LOWER_SNAKE_CASE           BASIC, STANDARD           *        Checks that packages are lower_snake.case.
PACKAGE_SAME_CSHARP_NAMESPACE      BASIC, STANDARD           *        Checks that all files with a given package have the same value for the csharp_namespace option.
PACKAGE_SAME_GO_PACKAGE            BASIC, STANDARD           *        Checks that all files with a given package have the same value for the go_package option.
PACKAGE_SAME_JAVA_MULTIPLE_FILES   BASIC, STANDARD           *        Checks that all files with a given package have the same value for the java_multiple_files option.
PACKAGE_SAME_JAVA_PACKAGE          BASIC, STANDARD           *        Checks that all files with a given package have the same value for the java_package option.
PACKAGE_SAME_PHP_NAMESPACE         BASIC, STANDARD           *        Checks that all files with a given package have the same value for the php_namespace option.
PACKAGE_SAME_RUBY_PACKAGE          BASIC, STANDARD           *        Checks that all files with a given package have the same value for the ruby_package option.
PACKAGE_SAME_SWIFT_PREFIX          BASIC, STANDARD           *        Checks that all files with a given package have the same value for the swift_prefix option.
RPC_PASCAL_CASE                    BASIC, STANDARD           *        Checks that RPCs are PascalCase.
SERVICE_PASCAL_CASE                BASIC, STANDARD           *        Checks that services are PascalCase.
SYNTAX_SPECIFIED                   BASIC, STANDARD           *        Checks that all files have a syntax specified.
ENUM_VALUE_PREFIX                  STANDARD                  *        Checks that enum values are prefixed with ENUM_NAME_UPPER_SNAKE_CASE.
ENUM_ZERO_VALUE_SUFFIX             STANDARD                  *        Checks that enum zero values have a consistent suffix (configurable, default suffix is "_UNSPECIFIED").
FILE_LOWER_SNAKE_CASE              STANDARD                  *        Checks that filenames are lower_snake_case.
PACKAGE_VERSION_SUFFIX             STANDARD                  *        Checks that the last component of all packages is a version of the form v\d+, v\d+test.*, v\d+(alpha|beta)\d+, or v\d+p\d+(alpha|beta)\d+, where numbers are >=1.
PROTOVALIDATE                      STANDARD                  *        Checks that protovalidate rules are valid and all CEL expressions compile.
RPC_REQUEST_RESPONSE_UNIQUE        STANDARD                  *        Checks that RPC request and response types are only used in one RPC (configurable).
RPC_REQUEST_STANDARD_NAME          STANDARD                  *        Checks that RPC request type names are RPCNameRequest or ServiceNameRPCNameRequest (configurable).
RPC_RESPONSE_STANDARD_NAME         STANDARD                  *        Checks that RPC response type names are RPCNameResponse or ServiceNameRPCNameResponse (configurable).
SERVICE_SUFFIX                     STANDARD                  *        Checks that services have a consistent suffix (configurable, default suffix is "Service").
COMMENT_ENUM                       COMMENTS                           Checks that enums have non-empty comments.
COMMENT_ENUM_VALUE                 COMMENTS                           Checks that enum values have non-empty comments.
COMMENT_FIELD                      COMMENTS                           Checks that fields have non-empty comments.
COMMENT_MESSAGE                    COMMENTS                           Checks that messages have non-empty comments.
COMMENT_ONEOF                      COMMENTS                           Checks that oneofs have non-empty comments.
COMMENT_RPC                        COMMENTS                           Checks that RPCs have non-empty comments.
COMMENT_SERVICE                    COMMENTS                           Checks that services have non-empty comments.
RPC_NO_CLIENT_STREAMING            UNARY_RPC                          Checks that RPCs are not client streaming.
RPC_NO_SERVER_STREAMING            UNARY_RPC                          Checks that RPCs are not server streaming.
STABLE_PACKAGE_NO_IMPORT_UNSTABLE                                     Checks that all files that have stable versioned packages do not import packages with unstable version packages.
		`
	testRunStdout(
		t,
		nil,
		0,
		expectedStdout,
		"config",
		"ls-lint-rules",
		"--version",
		"v2",
	)
}

func TestCheckLsBreakingRulesV1(t *testing.T) {
	t.Parallel()
	expectedStdout := `
ID                                              CATEGORIES                      DEFAULT  PURPOSE
ENUM_NO_DELETE                                  FILE                            *        Checks that enums are not deleted from a given file.
FILE_NO_DELETE                                  FILE                            *        Checks that files are not deleted.
MESSAGE_NO_DELETE                               FILE                            *        Checks that messages are not deleted from a given file.
SERVICE_NO_DELETE                               FILE                            *        Checks that services are not deleted from a given file.
ENUM_SAME_TYPE                                  FILE, PACKAGE                   *        Checks that enums have the same type (open vs closed).
ENUM_VALUE_NO_DELETE                            FILE, PACKAGE                   *        Checks that enum values are not deleted from a given enum.
EXTENSION_MESSAGE_NO_DELETE                     FILE, PACKAGE                   *        Checks that extension ranges are not deleted from a given message.
FIELD_NO_DELETE                                 FILE, PACKAGE                   *        Checks that fields are not deleted from a given message.
FIELD_SAME_CARDINALITY                          FILE, PACKAGE                   *        Checks that fields have the same cardinalities in a given message.
FIELD_SAME_CPP_STRING_TYPE                      FILE, PACKAGE                   *        Checks that fields have the same C++ string type, based on ctype field option or (pb.cpp).string_type feature.
FIELD_SAME_JAVA_UTF8_VALIDATION                 FILE, PACKAGE                   *        Checks that fields have the same Java string UTF8 validation, based on java_string_check_utf8 file option or (pb.java).utf8_validation feature.
FIELD_SAME_JSTYPE                               FILE, PACKAGE                   *        Checks that fields have the same value for the jstype option.
FIELD_SAME_TYPE                                 FILE, PACKAGE                   *        Checks that fields have the same types in a given message.
FIELD_SAME_UTF8_VALIDATION                      FILE, PACKAGE                   *        Checks that string fields have the same UTF8 validation mode.
FILE_SAME_CC_ENABLE_ARENAS                      FILE, PACKAGE                   *        Checks that files have the same value for the cc_enable_arenas option.
FILE_SAME_CC_GENERIC_SERVICES                   FILE, PACKAGE                   *        Checks that files have the same value for the cc_generic_services option.
FILE_SAME_CSHARP_NAMESPACE                      FILE, PACKAGE                   *        Checks that files have the same value for the csharp_namespace option.
FILE_SAME_GO_PACKAGE                            FILE, PACKAGE                   *        Checks that files have the same value for the go_package option.
FILE_SAME_JAVA_GENERIC_SERVICES                 FILE, PACKAGE                   *        Checks that files have the same value for the java_generic_services option.
FILE_SAME_JAVA_MULTIPLE_FILES                   FILE, PACKAGE                   *        Checks that files have the same value for the java_multiple_files option.
FILE_SAME_JAVA_OUTER_CLASSNAME                  FILE, PACKAGE                   *        Checks that files have the same value for the java_outer_classname option.
FILE_SAME_JAVA_PACKAGE                          FILE, PACKAGE                   *        Checks that files have the same value for the java_package option.
FILE_SAME_OBJC_CLASS_PREFIX                     FILE, PACKAGE                   *        Checks that files have the same value for the objc_class_prefix option.
FILE_SAME_OPTIMIZE_FOR                          FILE, PACKAGE                   *        Checks that files have the same value for the optimize_for option.
FILE_SAME_PHP_CLASS_PREFIX                      FILE, PACKAGE                   *        Checks that files have the same value for the php_class_prefix option.
FILE_SAME_PHP_METADATA_NAMESPACE                FILE, PACKAGE                   *        Checks that files have the same value for the php_metadata_namespace option.
FILE_SAME_PHP_NAMESPACE                         FILE, PACKAGE                   *        Checks that files have the same value for the php_namespace option.
FILE_SAME_PY_GENERIC_SERVICES                   FILE, PACKAGE                   *        Checks that files have the same value for the py_generic_services option.
FILE_SAME_RUBY_PACKAGE                          FILE, PACKAGE                   *        Checks that files have the same value for the ruby_package option.
FILE_SAME_SWIFT_PREFIX                          FILE, PACKAGE                   *        Checks that files have the same value for the swift_prefix option.
FILE_SAME_SYNTAX                                FILE, PACKAGE                   *        Checks that files have the same syntax.
MESSAGE_NO_REMOVE_STANDARD_DESCRIPTOR_ACCESSOR  FILE, PACKAGE                   *        Checks that messages do not change the no_standard_descriptor_accessor option from false or unset to true.
ONEOF_NO_DELETE                                 FILE, PACKAGE                   *        Checks that oneofs are not deleted from a given message.
RPC_NO_DELETE                                   FILE, PACKAGE                   *        Checks that rpcs are not deleted from a given service.
ENUM_SAME_JSON_FORMAT                           FILE, PACKAGE, WIRE_JSON        *        Checks that enums have the same JSON format support.
ENUM_VALUE_SAME_NAME                            FILE, PACKAGE, WIRE_JSON        *        Checks that enum values have the same name.
FIELD_SAME_JSON_NAME                            FILE, PACKAGE, WIRE_JSON        *        Checks that fields have the same value for the json_name option.
FIELD_SAME_NAME                                 FILE, PACKAGE, WIRE_JSON        *        Checks that fields have the same names in a given message.
MESSAGE_SAME_JSON_FORMAT                        FILE, PACKAGE, WIRE_JSON        *        Checks that messages have the same JSON format support.
FIELD_SAME_ONEOF                                FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that fields have the same oneofs in a given message.
FILE_SAME_PACKAGE                               FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that files have the same package.
MESSAGE_SAME_REQUIRED_FIELDS                    FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that messages have no added or deleted required fields.
RESERVED_ENUM_NO_DELETE                         FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that reserved ranges and names are not deleted from a given enum.
RESERVED_MESSAGE_NO_DELETE                      FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that reserved ranges and names are not deleted from a given message.
RPC_SAME_CLIENT_STREAMING                       FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that rpcs have the same client streaming value.
RPC_SAME_IDEMPOTENCY_LEVEL                      FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that rpcs have the same value for the idempotency_level option.
RPC_SAME_REQUEST_TYPE                           FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that rpcs are have the same request type.
RPC_SAME_RESPONSE_TYPE                          FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that rpcs are have the same response type.
RPC_SAME_SERVER_STREAMING                       FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that rpcs have the same server streaming value.
PACKAGE_ENUM_NO_DELETE                          PACKAGE                                  Checks that enums are not deleted from a given package.
PACKAGE_MESSAGE_NO_DELETE                       PACKAGE                                  Checks that messages are not deleted from a given package.
PACKAGE_NO_DELETE                               PACKAGE                                  Checks that packages are not deleted.
PACKAGE_SERVICE_NO_DELETE                       PACKAGE                                  Checks that services are not deleted from a given package.
ENUM_VALUE_NO_DELETE_UNLESS_NAME_RESERVED       WIRE_JSON                                Checks that enum values are not deleted from a given enum unless the name is reserved.
FIELD_NO_DELETE_UNLESS_NAME_RESERVED            WIRE_JSON                                Checks that fields are not deleted from a given message unless the name is reserved.
FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY          WIRE_JSON                                Checks that fields have wire and JSON compatible cardinalities in a given message.
FIELD_WIRE_JSON_COMPATIBLE_TYPE                 WIRE_JSON                                Checks that fields have wire and JSON compatible types in a given message.
ENUM_VALUE_NO_DELETE_UNLESS_NUMBER_RESERVED     WIRE_JSON, WIRE                          Checks that enum values are not deleted from a given enum unless the number is reserved.
FIELD_NO_DELETE_UNLESS_NUMBER_RESERVED          WIRE_JSON, WIRE                          Checks that fields are not deleted from a given message unless the number is reserved.
FIELD_WIRE_COMPATIBLE_CARDINALITY               WIRE                                     Checks that fields have wire-compatible cardinalities in a given message.
FIELD_WIRE_COMPATIBLE_TYPE                      WIRE                                     Checks that fields have wire-compatible types in a given message.
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

func TestCheckLsBreakingRulesV1Beta1(t *testing.T) {
	t.Parallel()
	expectedStdout := `
ID                                              CATEGORIES                      DEFAULT  PURPOSE
ENUM_NO_DELETE                                  FILE                            *        Checks that enums are not deleted from a given file.
FILE_NO_DELETE                                  FILE                            *        Checks that files are not deleted.
FILE_SAME_PACKAGE                               FILE                            *        Checks that files have the same package.
MESSAGE_NO_DELETE                               FILE                            *        Checks that messages are not deleted from a given file.
SERVICE_NO_DELETE                               FILE                            *        Checks that services are not deleted from a given file.
ENUM_SAME_TYPE                                  FILE, PACKAGE                   *        Checks that enums have the same type (open vs closed).
ENUM_VALUE_NO_DELETE                            FILE, PACKAGE                   *        Checks that enum values are not deleted from a given enum.
EXTENSION_MESSAGE_NO_DELETE                     FILE, PACKAGE                   *        Checks that extension ranges are not deleted from a given message.
FIELD_NO_DELETE                                 FILE, PACKAGE                   *        Checks that fields are not deleted from a given message.
FIELD_SAME_CPP_STRING_TYPE                      FILE, PACKAGE                   *        Checks that fields have the same C++ string type, based on ctype field option or (pb.cpp).string_type feature.
FIELD_SAME_JAVA_UTF8_VALIDATION                 FILE, PACKAGE                   *        Checks that fields have the same Java string UTF8 validation, based on java_string_check_utf8 file option or (pb.java).utf8_validation feature.
FIELD_SAME_JSTYPE                               FILE, PACKAGE                   *        Checks that fields have the same value for the jstype option.
FIELD_SAME_UTF8_VALIDATION                      FILE, PACKAGE                   *        Checks that string fields have the same UTF8 validation mode.
FILE_SAME_CC_ENABLE_ARENAS                      FILE, PACKAGE                   *        Checks that files have the same value for the cc_enable_arenas option.
FILE_SAME_CC_GENERIC_SERVICES                   FILE, PACKAGE                   *        Checks that files have the same value for the cc_generic_services option.
FILE_SAME_CSHARP_NAMESPACE                      FILE, PACKAGE                   *        Checks that files have the same value for the csharp_namespace option.
FILE_SAME_GO_PACKAGE                            FILE, PACKAGE                   *        Checks that files have the same value for the go_package option.
FILE_SAME_JAVA_GENERIC_SERVICES                 FILE, PACKAGE                   *        Checks that files have the same value for the java_generic_services option.
FILE_SAME_JAVA_MULTIPLE_FILES                   FILE, PACKAGE                   *        Checks that files have the same value for the java_multiple_files option.
FILE_SAME_JAVA_OUTER_CLASSNAME                  FILE, PACKAGE                   *        Checks that files have the same value for the java_outer_classname option.
FILE_SAME_JAVA_PACKAGE                          FILE, PACKAGE                   *        Checks that files have the same value for the java_package option.
FILE_SAME_OBJC_CLASS_PREFIX                     FILE, PACKAGE                   *        Checks that files have the same value for the objc_class_prefix option.
FILE_SAME_OPTIMIZE_FOR                          FILE, PACKAGE                   *        Checks that files have the same value for the optimize_for option.
FILE_SAME_PHP_CLASS_PREFIX                      FILE, PACKAGE                   *        Checks that files have the same value for the php_class_prefix option.
FILE_SAME_PHP_METADATA_NAMESPACE                FILE, PACKAGE                   *        Checks that files have the same value for the php_metadata_namespace option.
FILE_SAME_PHP_NAMESPACE                         FILE, PACKAGE                   *        Checks that files have the same value for the php_namespace option.
FILE_SAME_PY_GENERIC_SERVICES                   FILE, PACKAGE                   *        Checks that files have the same value for the py_generic_services option.
FILE_SAME_RUBY_PACKAGE                          FILE, PACKAGE                   *        Checks that files have the same value for the ruby_package option.
FILE_SAME_SWIFT_PREFIX                          FILE, PACKAGE                   *        Checks that files have the same value for the swift_prefix option.
FILE_SAME_SYNTAX                                FILE, PACKAGE                   *        Checks that files have the same syntax.
MESSAGE_NO_REMOVE_STANDARD_DESCRIPTOR_ACCESSOR  FILE, PACKAGE                   *        Checks that messages do not change the no_standard_descriptor_accessor option from false or unset to true.
ONEOF_NO_DELETE                                 FILE, PACKAGE                   *        Checks that oneofs are not deleted from a given message.
RPC_NO_DELETE                                   FILE, PACKAGE                   *        Checks that rpcs are not deleted from a given service.
ENUM_SAME_JSON_FORMAT                           FILE, PACKAGE, WIRE_JSON        *        Checks that enums have the same JSON format support.
ENUM_VALUE_SAME_NAME                            FILE, PACKAGE, WIRE_JSON        *        Checks that enum values have the same name.
FIELD_SAME_JSON_NAME                            FILE, PACKAGE, WIRE_JSON        *        Checks that fields have the same value for the json_name option.
FIELD_SAME_NAME                                 FILE, PACKAGE, WIRE_JSON        *        Checks that fields have the same names in a given message.
MESSAGE_SAME_JSON_FORMAT                        FILE, PACKAGE, WIRE_JSON        *        Checks that messages have the same JSON format support.
FIELD_SAME_CARDINALITY                          FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that fields have the same cardinalities in a given message.
FIELD_SAME_ONEOF                                FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that fields have the same oneofs in a given message.
FIELD_SAME_TYPE                                 FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that fields have the same types in a given message.
MESSAGE_SAME_REQUIRED_FIELDS                    FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that messages have no added or deleted required fields.
RESERVED_ENUM_NO_DELETE                         FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that reserved ranges and names are not deleted from a given enum.
RESERVED_MESSAGE_NO_DELETE                      FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that reserved ranges and names are not deleted from a given message.
RPC_SAME_CLIENT_STREAMING                       FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that rpcs have the same client streaming value.
RPC_SAME_IDEMPOTENCY_LEVEL                      FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that rpcs have the same value for the idempotency_level option.
RPC_SAME_REQUEST_TYPE                           FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that rpcs are have the same request type.
RPC_SAME_RESPONSE_TYPE                          FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that rpcs are have the same response type.
RPC_SAME_SERVER_STREAMING                       FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that rpcs have the same server streaming value.
PACKAGE_ENUM_NO_DELETE                          PACKAGE                                  Checks that enums are not deleted from a given package.
PACKAGE_MESSAGE_NO_DELETE                       PACKAGE                                  Checks that messages are not deleted from a given package.
PACKAGE_NO_DELETE                               PACKAGE                                  Checks that packages are not deleted.
PACKAGE_SERVICE_NO_DELETE                       PACKAGE                                  Checks that services are not deleted from a given package.
ENUM_VALUE_NO_DELETE_UNLESS_NAME_RESERVED       WIRE_JSON                                Checks that enum values are not deleted from a given enum unless the name is reserved.
FIELD_NO_DELETE_UNLESS_NAME_RESERVED            WIRE_JSON                                Checks that fields are not deleted from a given message unless the name is reserved.
FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY          WIRE_JSON                                Checks that fields have wire and JSON compatible cardinalities in a given message.
ENUM_VALUE_NO_DELETE_UNLESS_NUMBER_RESERVED     WIRE_JSON, WIRE                          Checks that enum values are not deleted from a given enum unless the number is reserved.
FIELD_NO_DELETE_UNLESS_NUMBER_RESERVED          WIRE_JSON, WIRE                          Checks that fields are not deleted from a given message unless the number is reserved.
FIELD_WIRE_COMPATIBLE_CARDINALITY               WIRE                                     Checks that fields have wire-compatible cardinalities in a given message.
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

func TestCheckLsBreakingRulesV2(t *testing.T) {
	t.Parallel()
	expectedStdout := `
ID                                              CATEGORIES                           DEFAULT  PURPOSE
EXTENSION_NO_DELETE                             FILE                                 *        Checks that extensions are not deleted from a given file.
SERVICE_NO_DELETE                               FILE                                 *        Checks that services are not deleted from a given file.
ENUM_NO_DELETE                                  CSR, FILE                            *        Checks that enums are not deleted from a given file.
FILE_NO_DELETE                                  CSR, FILE                            *        Checks that files are not deleted.
MESSAGE_NO_DELETE                               CSR, FILE                            *        Checks that messages are not deleted from a given file.
ENUM_SAME_TYPE                                  FILE, PACKAGE                        *        Checks that enums have the same type (open vs closed).
FIELD_SAME_CARDINALITY                          FILE, PACKAGE                        *        Checks that fields have the same cardinalities in a given message.
FIELD_SAME_CPP_STRING_TYPE                      FILE, PACKAGE                        *        Checks that fields have the same C++ string type, based on ctype field option or (pb.cpp).string_type feature.
FIELD_SAME_JAVA_UTF8_VALIDATION                 FILE, PACKAGE                        *        Checks that fields have the same Java string UTF8 validation, based on java_string_check_utf8 file option or (pb.java).utf8_validation feature.
FIELD_SAME_JSTYPE                               FILE, PACKAGE                        *        Checks that fields have the same value for the jstype option.
FIELD_SAME_UTF8_VALIDATION                      FILE, PACKAGE                        *        Checks that string fields have the same UTF8 validation mode.
FILE_SAME_CC_ENABLE_ARENAS                      FILE, PACKAGE                        *        Checks that files have the same value for the cc_enable_arenas option.
FILE_SAME_CC_GENERIC_SERVICES                   FILE, PACKAGE                        *        Checks that files have the same value for the cc_generic_services option.
FILE_SAME_CSHARP_NAMESPACE                      FILE, PACKAGE                        *        Checks that files have the same value for the csharp_namespace option.
FILE_SAME_GO_PACKAGE                            FILE, PACKAGE                        *        Checks that files have the same value for the go_package option.
FILE_SAME_JAVA_GENERIC_SERVICES                 FILE, PACKAGE                        *        Checks that files have the same value for the java_generic_services option.
FILE_SAME_JAVA_MULTIPLE_FILES                   FILE, PACKAGE                        *        Checks that files have the same value for the java_multiple_files option.
FILE_SAME_JAVA_OUTER_CLASSNAME                  FILE, PACKAGE                        *        Checks that files have the same value for the java_outer_classname option.
FILE_SAME_JAVA_PACKAGE                          FILE, PACKAGE                        *        Checks that files have the same value for the java_package option.
FILE_SAME_OBJC_CLASS_PREFIX                     FILE, PACKAGE                        *        Checks that files have the same value for the objc_class_prefix option.
FILE_SAME_OPTIMIZE_FOR                          FILE, PACKAGE                        *        Checks that files have the same value for the optimize_for option.
FILE_SAME_PHP_CLASS_PREFIX                      FILE, PACKAGE                        *        Checks that files have the same value for the php_class_prefix option.
FILE_SAME_PHP_METADATA_NAMESPACE                FILE, PACKAGE                        *        Checks that files have the same value for the php_metadata_namespace option.
FILE_SAME_PHP_NAMESPACE                         FILE, PACKAGE                        *        Checks that files have the same value for the php_namespace option.
FILE_SAME_PY_GENERIC_SERVICES                   FILE, PACKAGE                        *        Checks that files have the same value for the py_generic_services option.
FILE_SAME_RUBY_PACKAGE                          FILE, PACKAGE                        *        Checks that files have the same value for the ruby_package option.
FILE_SAME_SWIFT_PREFIX                          FILE, PACKAGE                        *        Checks that files have the same value for the swift_prefix option.
RPC_NO_DELETE                                   FILE, PACKAGE                        *        Checks that rpcs are not deleted from a given service.
ENUM_VALUE_NO_DELETE                            CSR, FILE, PACKAGE                   *        Checks that enum values are not deleted from a given enum.
EXTENSION_MESSAGE_NO_DELETE                     CSR, FILE, PACKAGE                   *        Checks that extension ranges are not deleted from a given message.
FIELD_NO_DELETE                                 CSR, FILE, PACKAGE                   *        Checks that fields are not deleted from a given message.
FIELD_SAME_TYPE                                 CSR, FILE, PACKAGE                   *        Checks that fields have the same types in a given message.
FILE_SAME_SYNTAX                                CSR, FILE, PACKAGE                   *        Checks that files have the same syntax.
MESSAGE_NO_REMOVE_STANDARD_DESCRIPTOR_ACCESSOR  CSR, FILE, PACKAGE                   *        Checks that messages do not change the no_standard_descriptor_accessor option from false or unset to true.
ONEOF_NO_DELETE                                 CSR, FILE, PACKAGE                   *        Checks that oneofs are not deleted from a given message.
ENUM_SAME_JSON_FORMAT                           CSR, FILE, PACKAGE, WIRE_JSON        *        Checks that enums have the same JSON format support.
ENUM_VALUE_SAME_NAME                            CSR, FILE, PACKAGE, WIRE_JSON        *        Checks that enum values have the same name.
FIELD_SAME_JSON_NAME                            CSR, FILE, PACKAGE, WIRE_JSON        *        Checks that fields have the same value for the json_name option.
FIELD_SAME_NAME                                 CSR, FILE, PACKAGE, WIRE_JSON        *        Checks that fields have the same names in a given message.
MESSAGE_SAME_JSON_FORMAT                        CSR, FILE, PACKAGE, WIRE_JSON        *        Checks that messages have the same JSON format support.
FIELD_SAME_DEFAULT                              CSR, FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that fields have the same default value, if a default is specified.
FIELD_SAME_ONEOF                                CSR, FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that fields have the same oneofs in a given message.
FILE_SAME_PACKAGE                               CSR, FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that files have the same package.
MESSAGE_SAME_REQUIRED_FIELDS                    CSR, FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that messages have no added or deleted required fields.
RESERVED_ENUM_NO_DELETE                         CSR, FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that reserved ranges and names are not deleted from a given enum.
RESERVED_MESSAGE_NO_DELETE                      CSR, FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that reserved ranges and names are not deleted from a given message.
RPC_SAME_CLIENT_STREAMING                       CSR, FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that rpcs have the same client streaming value.
RPC_SAME_IDEMPOTENCY_LEVEL                      CSR, FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that rpcs have the same value for the idempotency_level option.
RPC_SAME_REQUEST_TYPE                           CSR, FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that rpcs are have the same request type.
RPC_SAME_RESPONSE_TYPE                          CSR, FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that rpcs are have the same response type.
RPC_SAME_SERVER_STREAMING                       CSR, FILE, PACKAGE, WIRE_JSON, WIRE  *        Checks that rpcs have the same server streaming value.
PACKAGE_ENUM_NO_DELETE                          PACKAGE                                       Checks that enums are not deleted from a given package.
PACKAGE_EXTENSION_NO_DELETE                     PACKAGE                                       Checks that extensions are not deleted from a given package.
PACKAGE_MESSAGE_NO_DELETE                       PACKAGE                                       Checks that messages are not deleted from a given package.
PACKAGE_NO_DELETE                               PACKAGE                                       Checks that packages are not deleted.
PACKAGE_SERVICE_NO_DELETE                       PACKAGE                                       Checks that services are not deleted from a given package.
ENUM_VALUE_NO_DELETE_UNLESS_NAME_RESERVED       CSR, WIRE_JSON                                Checks that enum values are not deleted from a given enum unless the name is reserved.
FIELD_NO_DELETE_UNLESS_NAME_RESERVED            CSR, WIRE_JSON                                Checks that fields are not deleted from a given message unless the name is reserved.
FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY          CSR, WIRE_JSON                                Checks that fields have wire and JSON compatible cardinalities in a given message.
FIELD_WIRE_JSON_COMPATIBLE_TYPE                 CSR, WIRE_JSON                                Checks that fields have wire and JSON compatible types in a given message.
ENUM_VALUE_NO_DELETE_UNLESS_NUMBER_RESERVED     CSR, WIRE_JSON, WIRE                          Checks that enum values are not deleted from a given enum unless the number is reserved.
FIELD_NO_DELETE_UNLESS_NUMBER_RESERVED          CSR, WIRE_JSON, WIRE                          Checks that fields are not deleted from a given message unless the number is reserved.
FIELD_WIRE_COMPATIBLE_CARDINALITY               WIRE                                          Checks that fields have wire-compatible cardinalities in a given message.
FIELD_WIRE_COMPATIBLE_TYPE                      WIRE                                          Checks that fields have wire-compatible types in a given message.
		`
	testRunStdout(
		t,
		nil,
		0,
		expectedStdout,
		"config",
		"ls-breaking-rules",
		"--version",
		"v2",
	)
}

func TestCheckLsBreakingRulesFromConfig(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		0,
		`
	 	ID                    CATEGORIES     DEFAULT  PURPOSE
	 	ENUM_VALUE_NO_DELETE  FILE, PACKAGE  *        Checks that enum values are not deleted from a given enum.
	 	FIELD_SAME_JSTYPE     FILE, PACKAGE  *        Checks that fields have the same value for the jstype option.
	 	`,
		"mod",
		"ls-breaking-rules",
		"--config",
		filepath.Join("testdata", "small_list_rules", "buf.yaml"),
	)
	// defaults only, built-ins and plugins.
	testLsRuleOutputJSON(
		t,
		check.RuleTypeBreaking,
		`{
			"version":"v2",
			"plugins":[{"plugin": "buf-plugin-suffix"}]
		}`,
		append(
			xslices.Filter(builtinBreakingRulesV2, func(breakingRule *outputCheckRule) bool {
				return breakingRule.Default
			}),
			&outputCheckRule{ID: "SERVICE_SUFFIXES_NO_CHANGE", Categories: []string{"OPERATION_SUFFIXES"}, Default: true, Purpose: "Ensure that services with configured suffixes are not deleted and do not have new RPCs or delete RPCs.", Plugin: "buf-plugin-suffix"},
		),
	)
	// configure a deprecated category and a non-deprecated built-in category.
	// deprecated category contains some non-deprecated rules.
	testLsRuleOutputJSON(
		t,
		check.RuleTypeBreaking,
		`{
			"version":"v2",
			"breaking": {
				"use": ["WIRE", "RESOURCE_SUFFIXES"],
			},
			"plugins":[{"plugin": "buf-plugin-suffix"}]
		}`,
		append(
			xslices.Filter(builtinBreakingRulesV2, func(breakingRule *outputCheckRule) bool {
				return slices.Contains(breakingRule.Categories, "WIRE")
			}),
			&outputCheckRule{ID: "ENUM_SUFFIXES_NO_CHANGE", Categories: []string{"ATTRIBUTES_SUFFIXES"}, Default: false, Purpose: "Ensure that enums with configured suffixes are not deleted and do not have new enum values or delete enum values.", Plugin: "buf-plugin-suffix"},
			&outputCheckRule{ID: "MESSAGE_SUFFIXES_NO_CHANGE", Categories: []string{"ATTRIBUTES_SUFFIXES"}, Default: false, Purpose: "Ensure that messages with configured suffixes are not deleted and do not have new fields or delete fields.", Plugin: "buf-plugin-suffix"},
		),
	)
	// configure a deprecated category and a non-deprecated category, no built-ins.
	testLsRuleOutputJSON(
		t,
		check.RuleTypeBreaking,
		`{
			"version":"v2",
			"breaking": {
				"use": ["OPERATION_SUFFIXES","ATTRIBUTES_SUFFIXES", "RESOURCE_SUFFIXES"],
			},
			"plugins":[{"plugin": "buf-plugin-suffix"}]
		}`,
		[]*outputCheckRule{
			{ID: "SERVICE_SUFFIXES_NO_CHANGE", Categories: []string{"OPERATION_SUFFIXES"}, Default: true, Purpose: "Ensure that services with configured suffixes are not deleted and do not have new RPCs or delete RPCs.", Plugin: "buf-plugin-suffix"},
			{ID: "ENUM_SUFFIXES_NO_CHANGE", Categories: []string{"ATTRIBUTES_SUFFIXES"}, Default: false, Purpose: "Ensure that enums with configured suffixes are not deleted and do not have new enum values or delete enum values.", Plugin: "buf-plugin-suffix"},
			{ID: "MESSAGE_SUFFIXES_NO_CHANGE", Categories: []string{"ATTRIBUTES_SUFFIXES"}, Default: false, Purpose: "Ensure that messages with configured suffixes are not deleted and do not have new fields or delete fields.", Plugin: "buf-plugin-suffix"},
		},
	)
	// configure a mix of rules from built-ins and plugins.
	testLsRuleOutputJSON(
		t,
		check.RuleTypeBreaking,
		`{
			"version":"v2",
			"breaking": {
				"use": ["FIELD_WIRE_COMPATIBLE_TYPE", "PACKAGE", "ENUM_SUFFIXES_NO_CHANGE"],
			},
			"plugins":[{"plugin": "buf-plugin-suffix"}]
		}`,
		append(
			xslices.Filter(builtinBreakingRulesV2, func(breakingRule *outputCheckRule) bool {
				return slices.Contains(breakingRule.Categories, "PACKAGE")
			}),
			&outputCheckRule{ID: "FIELD_WIRE_COMPATIBLE_TYPE", Categories: []string{"WIRE"}, Default: false, Purpose: "Checks that fields have wire-compatible types in a given message."},
			&outputCheckRule{ID: "ENUM_SUFFIXES_NO_CHANGE", Categories: []string{"ATTRIBUTES_SUFFIXES"}, Default: false, Purpose: "Ensure that enums with configured suffixes are not deleted and do not have new enum values or delete enum values.", Plugin: "buf-plugin-suffix"},
		),
	)
	// configure a mix of categories and rules from built-ins and plugins.
	testLsRuleOutputJSON(
		t,
		check.RuleTypeBreaking,
		`{
			"version":"v2",
			"breaking": {
				"use": ["FIELD_WIRE_COMPATIBLE_TYPE", "PACKAGE", "ENUM_SUFFIXES_NO_CHANGE", "OPERATION_SUFFIXES"],
			},
			"plugins":[{"plugin": "buf-plugin-suffix"}]
		}`,
		append(
			xslices.Filter(builtinBreakingRulesV2, func(breakingRule *outputCheckRule) bool {
				return slices.Contains(breakingRule.Categories, "PACKAGE") || breakingRule.ID == "FIELD_WIRE_COMPATIBLE_TYPE"
			}),
			&outputCheckRule{ID: "SERVICE_SUFFIXES_NO_CHANGE", Categories: []string{"OPERATION_SUFFIXES"}, Default: true, Purpose: "Ensure that services with configured suffixes are not deleted and do not have new RPCs or delete RPCs.", Plugin: "buf-plugin-suffix"},
			&outputCheckRule{ID: "ENUM_SUFFIXES_NO_CHANGE", Categories: []string{"ATTRIBUTES_SUFFIXES"}, Default: false, Purpose: "Ensure that enums with configured suffixes are not deleted and do not have new enum values or delete enum values.", Plugin: "buf-plugin-suffix"},
		),
	)
}

func TestCheckLsBreakingRulesFromConfigNotNamedBufYAML(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		0,
		`
		ID                    CATEGORIES     DEFAULT  PURPOSE
		ENUM_VALUE_NO_DELETE  FILE, PACKAGE  *        Checks that enum values are not deleted from a given enum.
		FIELD_SAME_JSTYPE     FILE, PACKAGE  *        Checks that fields have the same value for the jstype option.
		`,
		"mod",
		"ls-breaking-rules",
		"--config",
		// making sure that .yml works
		filepath.Join("testdata", "small_list_rules_yml", "config.yml"),
	)
}

func TestCheckLsBreakingRulesFromConfigExceptDeprecated(t *testing.T) {
	t.Parallel()

	for _, version := range bufconfig.AllFileVersions {
		t.Run(version.String(), func(t *testing.T) {
			t.Parallel()
			// Do not need any custom lint/breaking plugins here.
			client, err := bufcheck.NewClient(slogtestext.NewLogger(t))
			require.NoError(t, err)
			allRules, err := client.AllRules(context.Background(), check.RuleTypeBreaking, version)
			require.NoError(t, err)
			allPackageIDs := make([]string, 0, len(allRules))
			for _, rule := range allRules {
				if rule.Deprecated() {
					// Deprecated rules should not be associated with a category.
					// Instead, their replacements are associated with categories.
					assert.Empty(t, rule.Categories())
					continue
				}
				var found bool
				for _, category := range rule.Categories() {
					if category.ID() == "PACKAGE" {
						found = true
						break
					}
				}
				if found {
					allPackageIDs = append(allPackageIDs, rule.ID())
				}
			}
			sort.Strings(allPackageIDs)
			deprecations, err := bufcheck.GetDeprecatedIDToReplacementIDs(allRules)
			require.NoError(t, err)

			for deprecatedRule := range deprecations {
				t.Run(deprecatedRule, func(t *testing.T) {
					t.Parallel()
					ids := getRuleIDsFromLsBreaking(t, version.String(), []string{"PACKAGE"}, []string{deprecatedRule})
					expectedIDs := make([]string, 0, len(allPackageIDs))
					replacements := deprecations[deprecatedRule]
					for _, id := range allPackageIDs {
						if slices.Contains(replacements, id) {
							continue
						}
						expectedIDs = append(expectedIDs, id)
					}
					require.Equal(t, expectedIDs, ids)
				})
			}
		})
	}
}

func TestLsModulesWorkspaceV1(t *testing.T) {
	// Cannot be parallel since we chdir.
	pwd, err := osext.Getwd()
	require.NoError(t, err)
	defer func() {
		r := recover()
		assert.NoError(t, osext.Chdir(pwd))
		if r != nil {
			panic(r)
		}
	}()

	require.NoError(t, osext.Chdir(filepath.Join(pwd, "testdata", "lsmodules", "workspacev1")))
	testRunStdout(
		t,
		nil,
		0,
		// default format is path
		`
a_v1
b_no_name_v1
c_v1beta1
d_no_file
e_no_file
f_no_name_v1beta1
`,
		"config",
		"ls-modules",
	)
	testRunStdout(
		t,
		nil,
		0,
		// format as path
		`
a_v1
b_no_name_v1
c_v1beta1
d_no_file
e_no_file
f_no_name_v1beta1
`,
		"config",
		"ls-modules",
		"--format",
		"path",
	)
	testRunStdout(
		t,
		nil,
		0,
		// format as name
		`
buf.build/bar/baz
buf.build/foo/bar
`,
		"config",
		"ls-modules",
		"--format",
		"name",
	)
	testRunStdout(
		t,
		nil,
		0,
		// format as json, sort by path
		`
{"path":"a_v1","name":"buf.build/foo/bar"}
{"path":"b_no_name_v1"}
{"path":"c_v1beta1","name":"buf.build/bar/baz"}
{"path":"d_no_file"}
{"path":"e_no_file"}
{"path":"f_no_name_v1beta1"}
`,
		"config",
		"ls-modules",
		"--format",
		"json",
	)
}

func TestLsModulesWorkspaceV2(t *testing.T) {
	// Cannot be parallel since we chdir.
	pwd, err := osext.Getwd()
	require.NoError(t, err)
	defer func() {
		r := recover()
		assert.NoError(t, osext.Chdir(pwd))
		if r != nil {
			panic(r)
		}
	}()

	require.NoError(t, osext.Chdir(filepath.Join(pwd, "testdata", "lsmodules", "workspacev2")))
	testRunStdout(
		t,
		nil,
		0,
		// default format is path
		`
a
b_no_name
c
d_no_name
`,
		"config",
		"ls-modules",
	)
	testRunStdout(
		t,
		nil,
		0,
		// format as path
		`
a
b_no_name
c
d_no_name
`,
		"config",
		"ls-modules",
		"--format",
		"path",
	)
	testRunStdout(
		t,
		nil,
		0,
		// format as name
		`
buf.build/bar/baz
buf.build/foo/bar
`,
		"config",
		"ls-modules",
		"--format",
		"name",
	)
	testRunStdout(
		t,
		nil,
		0,
		// format as json, sort by path
		`
{"path":"a","name":"buf.build/foo/bar"}
{"path":"b_no_name"}
{"path":"c","name":"buf.build/bar/baz"}
{"path":"d_no_name"}
`,
		"config",
		"ls-modules",
		"--format",
		"json",
	)
}

func TestLsModulesModuleV1(t *testing.T) {
	// Cannot be parallel since we chdir.
	pwd, err := osext.Getwd()
	require.NoError(t, err)
	defer func() {
		r := recover()
		assert.NoError(t, osext.Chdir(pwd))
		if r != nil {
			panic(r)
		}
	}()

	// with name
	require.NoError(t, osext.Chdir(filepath.Join(pwd, "testdata", "lsmodules", "workspacev1", "a_v1")))
	testRunStdout(
		t,
		nil,
		0,
		// default format is path
		".",
		"config",
		"ls-modules",
	)
	testRunStdout(
		t,
		nil,
		0,
		".",
		"config",
		"ls-modules",
		"--format",
		"path",
	)
	testRunStdout(
		t,
		nil,
		0,
		`
buf.build/foo/bar
`,
		"config",
		"ls-modules",
		"--format",
		"name",
	)
	testRunStdout(
		t,
		nil,
		0,
		`{"path":".","name":"buf.build/foo/bar"}`,
		"config",
		"ls-modules",
		"--format",
		"json",
	)
	// without name
	require.NoError(t, osext.Chdir(filepath.Join(pwd, "testdata", "lsmodules", "workspacev1", "b_no_name_v1")))
	testRunStdout(
		t,
		nil,
		0,
		// default format is path
		".",
		"config",
		"ls-modules",
	)
	testRunStdout(
		t,
		nil,
		0,
		".",
		"config",
		"ls-modules",
		"--format",
		"path",
	)
	testRunStdout(
		t,
		nil,
		0,
		"", // empty output
		"config",
		"ls-modules",
		"--format",
		"name",
	)
	testRunStdout(
		t,
		nil,
		0,
		`{"path":"."}`,
		"config",
		"ls-modules",
		"--format",
		"json",
	)
}

func TestLsModulesModuleV1Beta1(t *testing.T) {
	// Cannot be parallel since we chdir.
	pwd, err := osext.Getwd()
	require.NoError(t, err)
	defer func() {
		r := recover()
		assert.NoError(t, osext.Chdir(pwd))
		if r != nil {
			panic(r)
		}
	}()

	// with name
	require.NoError(t, osext.Chdir(filepath.Join(pwd, "testdata", "lsmodules", "workspacev1", "c_v1beta1")))
	testRunStdout(
		t,
		nil,
		0,
		// default format is path
		".",
		"config",
		"ls-modules",
	)
	testRunStdout(
		t,
		nil,
		0,
		".",
		"config",
		"ls-modules",
		"--format",
		"path",
	)
	testRunStdout(
		t,
		nil,
		0,
		"buf.build/bar/baz",
		"config",
		"ls-modules",
		"--format",
		"name",
	)
	testRunStdout(
		t,
		nil,
		0,
		`{"path":".","name":"buf.build/bar/baz"}`,
		"config",
		"ls-modules",
		"--format",
		"json",
	)
	// without name
	require.NoError(t, osext.Chdir(filepath.Join(pwd, "testdata", "lsmodules", "workspacev1", "f_no_name_v1beta1")))
	testRunStdout(
		t,
		nil,
		0,
		// default format is path
		".",
		"config",
		"ls-modules",
	)
	testRunStdout(
		t,
		nil,
		0,
		".",
		"config",
		"ls-modules",
		"--format",
		"path",
	)
	testRunStdout(
		t,
		nil,
		0,
		"", // empty output
		"config",
		"ls-modules",
		"--format",
		"name",
	)
	testRunStdout(
		t,
		nil,
		0,
		`{"path":"."}`,
		"config",
		"ls-modules",
		"--format",
		"json",
	)
}

func TestLsModulesNoConfig(t *testing.T) {
	// Cannot be parallel since we chdir.
	pwd, err := osext.Getwd()
	require.NoError(t, err)
	defer func() {
		r := recover()
		assert.NoError(t, osext.Chdir(pwd))
		if r != nil {
			panic(r)
		}
	}()

	// with name
	require.NoError(t, osext.Chdir(t.TempDir()))
	testRunStdout(
		t,
		nil,
		0,
		// default format is path
		".",
		"config",
		"ls-modules",
	)
	testRunStdout(
		t,
		nil,
		0,
		".",
		"config",
		"ls-modules",
		"--format",
		"path",
	)
	testRunStdout(
		t,
		nil,
		0,
		"",
		"config",
		"ls-modules",
		"--format",
		"name",
	)
	testRunStdout(
		t,
		nil,
		0,
		`{"path":"."}`,
		"config",
		"ls-modules",
		"--format",
		"json",
	)
}

func TestLsModulesBothConfig(t *testing.T) {
	// Cannot be parallel since we chdir.
	pwd, err := osext.Getwd()
	require.NoError(t, err)
	defer func() {
		r := recover()
		assert.NoError(t, osext.Chdir(pwd))
		if r != nil {
			panic(r)
		}
	}()

	require.NoError(t, osext.Chdir(filepath.Join(pwd, "testdata", "lsmodules", "extraconfigv1")))
	testRunStderrContainsNoWarn(
		t,
		nil,
		1,
		[]string{"buf.yaml", "buf.work.yaml"},
		"config",
		"ls-modules",
	)

	require.NoError(t, osext.Chdir(filepath.Join(pwd, "testdata", "lsmodules", "extraconfigv2")))
	testRunStderrContainsNoWarn(
		t,
		nil,
		1,
		[]string{"buf.yaml", "buf.work.yaml"},
		"config",
		"ls-modules",
	)
}

func TestLsModulesInvalidVersion(t *testing.T) {
	// Cannot be parallel since we chdir.
	pwd, err := osext.Getwd()
	require.NoError(t, err)
	defer func() {
		r := recover()
		assert.NoError(t, osext.Chdir(pwd))
		if r != nil {
			panic(r)
		}
	}()

	require.NoError(t, osext.Chdir(filepath.Join(pwd, "testdata", "lsmodules", "workspaceinvalid")))
	testRunStderr(
		t,
		nil,
		1,
		`Failure: buf.work.yaml pointed to directory "proto" which has a v2 buf.yaml file`,
		"config",
		"ls-modules",
	)
}

func TestLsModulesConfigFlag(t *testing.T) {
	t.Parallel()

	// v1beta1
	testRunStdout(
		t,
		nil,
		0,
		`{"path":".","name":"buf.build/bar/baz"}`,
		"config",
		"ls-modules",
		"--config",
		filepath.Join("testdata", "lsmodules", "workspacev1", "c_v1beta1", "buf.yaml"),
		"--format",
		"json",
	)

	// v1
	testRunStdout(
		t,
		nil,
		0,
		`{"path":".","name":"buf.build/foo/bar"}`,
		"config",
		"ls-modules",
		"--config",
		filepath.Join("testdata", "lsmodules", "workspacev1", "a_v1", "buf.yaml"),
		"--format",
		"json",
	)

	// v2
	testRunStdout(
		t,
		nil,
		0,
		`
{"path":"a","name":"buf.build/foo/bar"}
{"path":"b_no_name"}
{"path":"c","name":"buf.build/bar/baz"}
{"path":"d_no_name"}
`,
		"config",
		"ls-modules",
		"--config",
		filepath.Join("testdata", "lsmodules", "workspacev2", "buf.yaml"),
		"--format",
		"json",
	)
}

func TestLsModulesConfigFlagTakesPrecedence(t *testing.T) {
	// Cannot be parallel since we chdir.
	pwd, err := osext.Getwd()
	require.NoError(t, err)
	defer func() {
		r := recover()
		assert.NoError(t, osext.Chdir(pwd))
		if r != nil {
			panic(r)
		}
	}()

	require.NoError(t, osext.Chdir(filepath.Join(pwd, "testdata", "lsmodules", "workspacev1")))
	testRunStdout(
		t,
		nil,
		0,
		// default format is path
		`
a
b_no_name
c
d_no_name
`,
		"config",
		"ls-modules",
		"--config",
		filepath.Join(pwd, "testdata", "lsmodules", "workspacev2", "buf.yaml"),
	)
}

func TestLsModulesWorkspaceV2DuplicateDirPath(t *testing.T) {
	// Cannot be parallel since we chdir.
	pwd, err := osext.Getwd()
	require.NoError(t, err)
	defer func() {
		r := recover()
		assert.NoError(t, osext.Chdir(pwd))
		if r != nil {
			panic(r)
		}
	}()

	require.NoError(t, osext.Chdir(filepath.Join(pwd, "testdata", "workspace", "success", "duplicate_dir_path")))
	testRunStdout(
		t,
		nil,
		0,
		// default format is path
		`
proto/shared
proto/shared
proto/shared1
proto/shared1
separate
`,
		"config",
		"ls-modules",
	)
	testRunStdout(
		t,
		nil,
		0,
		// format as path
		`
proto/shared
proto/shared
proto/shared1
proto/shared1
separate
	`,

		"config",
		"ls-modules",
		"--format",
		"path",
	)
	testRunStdout(
		t,
		nil,
		0,
		// format as name
		`
buf.build/shared/one
buf.build/shared/zero
	`,

		"config",
		"ls-modules",
		"--format",
		"name",
	)
	testRunStdout(
		t,
		nil,
		0,
		// format as json, sort by path
		`

	{"path":"proto/shared","excludes":["proto/shared/prefix/foo"]}
	{"path":"proto/shared","excludes":["proto/shared/prefix/bar"],"name":"buf.build/shared/zero"}
	{"path":"proto/shared1","includes":["proto/shared1/prefix/x"],"name":"buf.build/shared/one"}
	{"path":"proto/shared1","excludes":["proto/shared1/prefix/x"]}
	{"path":"separate"}
	`,

		"config",
		"ls-modules",
		"--format",
		"json",
	)
}

func TestLsBreakingRulesDeprecated(t *testing.T) {
	t.Parallel()

	stdout := bytes.NewBuffer(nil)
	testRun(t, 0, nil, stdout, "mod", "ls-breaking-rules", "--version", "v1beta1")
	assert.NotContains(t, stdout.String(), "FIELD_SAME_CTYPE")
	assert.NotContains(t, stdout.String(), "FIELD_SAME_LABEL")
	assert.NotContains(t, stdout.String(), "FILE_SAME_JAVA_STRING_CHECK_UTF8")
	assert.NotContains(t, stdout.String(), "FILE_SAME_PHP_GENERIC_SERVICES")

	stdout = bytes.NewBuffer(nil)
	testRun(t, 0, nil, stdout, "mod", "ls-breaking-rules", "--version", "v1beta1", "--include-deprecated")
	assert.Contains(t, stdout.String(), "FIELD_SAME_CTYPE")
	assert.Contains(t, stdout.String(), "FIELD_SAME_LABEL")
	assert.Contains(t, stdout.String(), "FILE_SAME_JAVA_STRING_CHECK_UTF8")
	assert.Contains(t, stdout.String(), "FILE_SAME_PHP_GENERIC_SERVICES")

	stdout = bytes.NewBuffer(nil)
	testRun(t, 0, nil, stdout, "mod", "ls-breaking-rules", "--version", "v1")
	assert.NotContains(t, stdout.String(), "FIELD_SAME_CTYPE")
	assert.NotContains(t, stdout.String(), "FIELD_SAME_LABEL")
	assert.NotContains(t, stdout.String(), "FILE_SAME_JAVA_STRING_CHECK_UTF8")
	assert.NotContains(t, stdout.String(), "FILE_SAME_PHP_GENERIC_SERVICES")

	stdout = bytes.NewBuffer(nil)
	testRun(t, 0, nil, stdout, "mod", "ls-breaking-rules", "--version", "v1", "--include-deprecated")
	assert.Contains(t, stdout.String(), "FIELD_SAME_CTYPE")
	assert.Contains(t, stdout.String(), "FIELD_SAME_LABEL")
	assert.Contains(t, stdout.String(), "FILE_SAME_JAVA_STRING_CHECK_UTF8")
	assert.Contains(t, stdout.String(), "FILE_SAME_PHP_GENERIC_SERVICES")

	stdout = bytes.NewBuffer(nil)
	testRun(t, 0, nil, stdout, "config", "ls-breaking-rules", "--version", "v1beta1")
	assert.NotContains(t, stdout.String(), "FIELD_SAME_CTYPE")
	assert.NotContains(t, stdout.String(), "FIELD_SAME_LABEL")
	assert.NotContains(t, stdout.String(), "FILE_SAME_JAVA_STRING_CHECK_UTF8")
	assert.NotContains(t, stdout.String(), "FILE_SAME_PHP_GENERIC_SERVICES")

	stdout = bytes.NewBuffer(nil)
	testRun(t, 0, nil, stdout, "config", "ls-breaking-rules", "--version", "v1beta1", "--include-deprecated")
	assert.Contains(t, stdout.String(), "FIELD_SAME_CTYPE")
	assert.Contains(t, stdout.String(), "FIELD_SAME_LABEL")
	assert.Contains(t, stdout.String(), "FILE_SAME_JAVA_STRING_CHECK_UTF8")
	assert.Contains(t, stdout.String(), "FILE_SAME_PHP_GENERIC_SERVICES")

	stdout = bytes.NewBuffer(nil)
	testRun(t, 0, nil, stdout, "config", "ls-breaking-rules", "--version", "v1")
	assert.NotContains(t, stdout.String(), "FIELD_SAME_CTYPE")
	assert.NotContains(t, stdout.String(), "FIELD_SAME_LABEL")
	assert.NotContains(t, stdout.String(), "FILE_SAME_JAVA_STRING_CHECK_UTF8")
	assert.NotContains(t, stdout.String(), "FILE_SAME_PHP_GENERIC_SERVICES")

	stdout = bytes.NewBuffer(nil)
	testRun(t, 0, nil, stdout, "config", "ls-breaking-rules", "--version", "v1", "--include-deprecated")
	assert.Contains(t, stdout.String(), "FIELD_SAME_CTYPE")
	assert.Contains(t, stdout.String(), "FIELD_SAME_LABEL")
	assert.Contains(t, stdout.String(), "FILE_SAME_JAVA_STRING_CHECK_UTF8")
	assert.Contains(t, stdout.String(), "FILE_SAME_PHP_GENERIC_SERVICES")

	stdout = bytes.NewBuffer(nil)
	testRun(t, 0, nil, stdout, "config", "ls-breaking-rules", "--version", "v2")
	assert.NotContains(t, stdout.String(), "FIELD_SAME_CTYPE")
	assert.NotContains(t, stdout.String(), "FIELD_SAME_LABEL")
	assert.NotContains(t, stdout.String(), "FILE_SAME_JAVA_STRING_CHECK_UTF8")
	assert.NotContains(t, stdout.String(), "FILE_SAME_PHP_GENERIC_SERVICES")

	// The deprecated rules are omitted from v2.
	stdout = bytes.NewBuffer(nil)
	testRun(t, 0, nil, stdout, "config", "ls-breaking-rules", "--version", "v2", "--include-deprecated")
	assert.NotContains(t, stdout.String(), "FIELD_SAME_CTYPE")
	assert.NotContains(t, stdout.String(), "FIELD_SAME_LABEL")
	assert.NotContains(t, stdout.String(), "FILE_SAME_JAVA_STRING_CHECK_UTF8")
	assert.NotContains(t, stdout.String(), "FILE_SAME_PHP_GENERIC_SERVICES")

	// Test the non-all version too. Should never have deprecated rules.

	stdout = bytes.NewBuffer(nil)
	testRun(t, 0, nil, stdout, "mod", "ls-breaking-rules")
	assert.NotContains(t, stdout.String(), "FIELD_SAME_CTYPE")
	assert.NotContains(t, stdout.String(), "FIELD_SAME_LABEL")
	assert.NotContains(t, stdout.String(), "FILE_SAME_JAVA_STRING_CHECK_UTF8")
	assert.NotContains(t, stdout.String(), "FILE_SAME_PHP_GENERIC_SERVICES")

	stdout = bytes.NewBuffer(nil)
	testRun(t, 0, nil, stdout, "mod", "ls-breaking-rules", "--include-deprecated")
	assert.NotContains(t, stdout.String(), "FIELD_SAME_CTYPE")
	assert.NotContains(t, stdout.String(), "FIELD_SAME_LABEL")
	assert.NotContains(t, stdout.String(), "FILE_SAME_JAVA_STRING_CHECK_UTF8")
	assert.NotContains(t, stdout.String(), "FILE_SAME_PHP_GENERIC_SERVICES")

	stdout = bytes.NewBuffer(nil)
	testRun(t, 0, nil, stdout, "config", "ls-breaking-rules", "--configured-only")
	assert.NotContains(t, stdout.String(), "FIELD_SAME_CTYPE")
	assert.NotContains(t, stdout.String(), "FIELD_SAME_LABEL")
	assert.NotContains(t, stdout.String(), "FILE_SAME_JAVA_STRING_CHECK_UTF8")
	assert.NotContains(t, stdout.String(), "FILE_SAME_PHP_GENERIC_SERVICES")

	stdout = bytes.NewBuffer(nil)
	testRun(t, 0, nil, stdout, "config", "ls-breaking-rules", "--configured-only", "--include-deprecated")
	assert.NotContains(t, stdout.String(), "FIELD_SAME_CTYPE")
	assert.NotContains(t, stdout.String(), "FIELD_SAME_LABEL")
	assert.NotContains(t, stdout.String(), "FILE_SAME_JAVA_STRING_CHECK_UTF8")
	assert.NotContains(t, stdout.String(), "FILE_SAME_PHP_GENERIC_SERVICES")
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

func TestLsFilesImage1_Yaml(t *testing.T) {
	t.Parallel()
	stdout := bytes.NewBuffer(nil)
	testRun(
		t,
		0,
		nil,
		stdout,
		"build",
		"-o",
		"-#format=yaml",
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
		"-#format=yaml",
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
		`Failure: --path is not valid for use with .proto file references`,
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
		`# For details on buf.yaml configuration, visit https://buf.build/docs/configuration/v2/buf-yaml
version: v2
lint:
  use:
    - STANDARD
breaking:
  use:
    - FILE
`,
		false,
		"",
	)
}

func TestLsFilesOverlappingPaths(t *testing.T) {
	t.Parallel()
	// It should be OK to have paths that overlap, and ls-files (and other commands)
	// should output the union.
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`testdata/paths/a/v3/a.proto
testdata/paths/a/v3/foo/bar.proto
testdata/paths/a/v3/foo/foo.proto`),
		"ls-files",
		filepath.Join("testdata", "paths"),
		"--path",
		filepath.Join("testdata", "paths", "a", "v3"),
		"--path",
		filepath.Join("testdata", "paths", "a", "v3", "foo"),
	)
}

func TestBuildOverlappingPaths(t *testing.T) {
	t.Parallel()
	// This may differ from LsFilesOverlappingPaths as we do a build of an image here.
	// Building of images results in bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles being
	// called, which is the original source of the issue that resulted in this test.
	testBuildLsFilesFormatImport(
		t,
		0,
		[]string{
			`a/v3/a.proto`,
			`a/v3/foo/bar.proto`,
			`a/v3/foo/foo.proto`,
		},
		filepath.Join("testdata", "paths"),
		"--path",
		filepath.Join("testdata", "paths", "a", "v3"),
		"--path",
		filepath.Join("testdata", "paths", "a", "v3", "foo"),
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
		`Failure: --path is not valid for use with .proto file references`,
		"export",
		filepath.Join("testdata", "protofileref", "success", "buf.proto"),
		"-o",
		tempDir,
		"--path",
		filepath.Join("testdata", "protofileref", "success", "buf.proto"),
	)
}

func TestExportAllSourceFilesV1Module(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"export",
		"--all",
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
		"LICENSE",
		"README.md",
		"request.proto",
		"rpc.proto",
	)
}

func TestExportAllSourceFilesV1Workspace(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"export",
		"--all",
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
		"LICENSE.request",
		"LICENSE.rpc",
		"README.another.md",
		"README.rpc.md",
		"another.proto",
		"request.proto",
		"rpc.proto",
		"unimported.proto",
	)
}

func TestExportAllSourceFilesV2Module(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"export",
		"--all",
		"-o",
		tempDir,
		filepath.Join("testdata", "workspace", "success", "v2", "export", "proto"),
	)
	readWriteBucket, err := storageos.NewProvider().NewReadWriteBucket(tempDir)
	require.NoError(t, err)
	storagetesting.AssertPaths(
		t,
		readWriteBucket,
		"",
		"LICENSE",
		"README.md",
		"request.proto",
		"rpc.proto",
	)
}

func TestExportAllSourceFilesV2Workspace(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"export",
		"--all",
		"-o",
		tempDir,
		filepath.Join("testdata", "workspace", "success", "v2", "export"),
	)
	readWriteBucket, err := storageos.NewProvider().NewReadWriteBucket(tempDir)
	require.NoError(t, err)
	storagetesting.AssertPaths(
		t,
		readWriteBucket,
		"",
		"LICENSE.request",
		"LICENSE.rpc",
		"README.another.md",
		"README.rpc.md",
		"another.proto",
		"request.proto",
		"rpc.proto",
		"unimported.proto",
	)
}

func TestBuildWithPaths(t *testing.T) {
	t.Parallel()
	testRunStdout(t, nil, 0, ``, "build", filepath.Join("testdata", "paths"), "--path", filepath.Join("testdata", "paths", "a", "v3"), "--exclude-path", filepath.Join("testdata", "paths", "a", "v3", "foo"))
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		// This is new post-refactor. Before, we gave precedence to --path. While a change,
		// doing --path foo/bar --exclude-path foo seems like a bug rather than expected behavior to maintain.
		filepath.FromSlash(`Failure: excluded path "testdata/paths/a/v3" contains targeted path "testdata/paths/a/v3/foo", which means all paths in "testdata/paths/a/v3/foo" will be excluded`),
		"build",
		filepath.Join("testdata", "paths"),
		"--path",
		filepath.Join("testdata", "paths", "a", "v3", "foo"),
		"--exclude-path",
		filepath.Join("testdata", "paths", "a", "v3"),
	)
}

func TestLintWithPaths(t *testing.T) {
	t.Parallel()
	testRunStdoutStderrNoWarn(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
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
		1,
		"",
		// This is new post-refactor. Before, we gave precedence to --path. While a change,
		// doing --path foo/bar --exclude-path foo seems like a bug rather than expected behavior to maintain.
		filepath.FromSlash(`Failure: excluded path "testdata/paths/a/v3" contains targeted path "testdata/paths/a/v3/foo", which means all paths in "testdata/paths/a/v3/foo" will be excluded`),
		"lint",
		filepath.Join("testdata", "paths"),
		"--path",
		filepath.Join("testdata", "paths", "a", "v3", "foo"),
		"--exclude-path",
		filepath.Join("testdata", "paths", "a", "v3"),
	)
}

func TestLintWithPlugins(t *testing.T) {
	t.Parallel()
	// defaults only, comment ignores on.
	testRunStdout(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`
testdata/check_plugins/current/proto/api/v1/service.proto:11:1:Service name "api.v1.FooServiceMock" has banned suffix "Mock". (buf-plugin-suffix)
testdata/check_plugins/current/proto/api/v1/service.proto:12:14:RPC request type "GetFooMockRequest" should be named "GetFooRequest" or "FooServiceMockGetFooRequest".
testdata/check_plugins/current/proto/api/v1/service.proto:12:42:RPC response type "GetFooMockResponse" should be named "GetFooResponse" or "FooServiceMockGetFooResponse".
testdata/check_plugins/current/proto/api/v1/service.proto:16:9:Service name "FooServiceTest" should be suffixed with "Service".
testdata/check_plugins/current/proto/api/v1/service.proto:17:14:RPC request type "GetFooTestRequest" should be named "GetFooRequest" or "FooServiceTestGetFooRequest".
testdata/check_plugins/current/proto/api/v1/service.proto:17:42:RPC response type "GetFooTestResponse" should be named "GetFooResponse" or "FooServiceTestGetFooResponse".
testdata/check_plugins/current/proto/api/v1/service.proto:26:1:"ListFooResponse" is a pagination response without a page token field named "page_token" (buf-plugin-rpc-ext)
testdata/check_plugins/current/proto/common/v1alpha1/messages.proto:16:5:field "common.v1alpha1.Four.FourTwo.id" does not have rule (buf.validate.field).string.tuuid set (buf-plugin-protovalidate-ext)
testdata/check_plugins/current/vendor/protovalidate/buf/validate/validate.proto:94:3:field "buf.validate.Rule.id" does not have rule (buf.validate.field).string.tuuid set (buf-plugin-protovalidate-ext)
		`),
		"lint",
		filepath.Join("testdata", "check_plugins", "current"),
	)
	// Defaults only, comment ignores off.
	// Always ignore the vendored protovalidate module.
	//
	// There are still lint failures for protovalidate despite being set as an ignore path because
	// proto imports these files, and paths outside of a module cannot be configured as ignore paths
	// for the module -- this is expected behavior for now.
	testRunStdout(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`
testdata/check_plugins/current/proto/api/v1/service.proto:11:1:Service name "api.v1.FooServiceMock" has banned suffix "Mock". (buf-plugin-suffix)
testdata/check_plugins/current/proto/api/v1/service.proto:11:9:Service name "FooServiceMock" should be suffixed with "Service".
testdata/check_plugins/current/proto/api/v1/service.proto:12:14:RPC request type "GetFooMockRequest" should be named "GetFooRequest" or "FooServiceMockGetFooRequest".
testdata/check_plugins/current/proto/api/v1/service.proto:12:42:RPC response type "GetFooMockResponse" should be named "GetFooResponse" or "FooServiceMockGetFooResponse".
testdata/check_plugins/current/proto/api/v1/service.proto:16:1:Service name "api.v1.FooServiceTest" has banned suffix "Test". (buf-plugin-suffix)
testdata/check_plugins/current/proto/api/v1/service.proto:16:9:Service name "FooServiceTest" should be suffixed with "Service".
testdata/check_plugins/current/proto/api/v1/service.proto:17:14:RPC request type "GetFooTestRequest" should be named "GetFooRequest" or "FooServiceTestGetFooRequest".
testdata/check_plugins/current/proto/api/v1/service.proto:17:42:RPC response type "GetFooTestResponse" should be named "GetFooResponse" or "FooServiceTestGetFooResponse".
testdata/check_plugins/current/proto/api/v1/service.proto:26:1:"ListFooResponse" is a pagination response without a page token field named "page_token" (buf-plugin-rpc-ext)
testdata/check_plugins/current/proto/common/v1alpha1/messages.proto:16:5:field "common.v1alpha1.Four.FourTwo.id" does not have rule (buf.validate.field).string.tuuid set (buf-plugin-protovalidate-ext)
testdata/check_plugins/current/vendor/protovalidate/buf/validate/validate.proto:94:3:field "buf.validate.Rule.id" does not have rule (buf.validate.field).string.tuuid set (buf-plugin-protovalidate-ext)
		`),
		"lint",
		filepath.Join("testdata", "check_plugins", "current"),
		"--config",
		`{
			"version":"v2",
			"modules": [
				{"path": "testdata/check_plugins/current/proto"},
				{"path": "testdata/check_plugins/current/vendor/protovalidate"}
			],
			"lint": {
				"disallow_comment_ignores": true,
				"ignore": [
					"testdata/check_plugins/current/vendor/protovalidate",
    				"testdata/check_plugins/current/proto/common/v1/breaking.proto",
    				"testdata/check_plugins/current/proto/common/v1alpha1/breaking.proto"
				]
			},
			"plugins":[
				{
					"plugin": "buf-plugin-suffix",
					"options": {
						"service_banned_suffixes": ["Mock", "Test"],
						"rpc_banned_suffixes": ["Element"],
						"field_banned_suffixes": ["_uuid"],
						"enum_value_banned_suffixes": ["_invalid"],
						"service_no_change_suffixes": ["Service"],
						"message_no_change_suffixes": ["Request", "Response"],
						"enum_no_change_suffixes": ["State"]
					}
				},
				{"plugin": "buf-plugin-protovalidate-ext"},
				{"plugin": "buf-plugin-rpc-ext"}
			]
		}`,
	)
	// With specified use, ignore, and ignore_only configurations.
	// Always ignore the vendored protovalidate module.
	//
	// There are still lint failures for protovalidate despite being set as an ignore path because
	// proto imports these files, and paths outside of a module cannot be configured as ignore paths
	// for the module -- this is expected behavior for now.
	testRunStdout(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`
testdata/check_plugins/current/proto/api/v1/service.proto:11:1:Service name "api.v1.FooServiceMock" has banned suffix "Mock". (buf-plugin-suffix)
testdata/check_plugins/current/proto/common/v1alpha1/messages.proto:16:5:field "common.v1alpha1.Four.FourTwo.id" does not have rule (buf.validate.field).string.tuuid set (buf-plugin-protovalidate-ext)
testdata/check_plugins/current/vendor/protovalidate/buf/validate/validate.proto:94:3:field "buf.validate.Rule.id" does not have rule (buf.validate.field).string.tuuid set (buf-plugin-protovalidate-ext)
		`),
		"lint",
		filepath.Join("testdata", "check_plugins", "current"),
		"--config",
		`{
			"version":"v2",
			"modules": [
				{"path": "testdata/check_plugins/current/proto"},
				{"path": "testdata/check_plugins/current/vendor/protovalidate"}
			],
			"lint": {
				"use": ["PAGE_REQUEST_HAS_TOKEN", "SERVICE_BANNED_SUFFIXES", "VALIDATE_ID_DASHLESS"],
				"ignore": [
					"testdata/check_plugins/current/vendor/protovalidate",
    				"testdata/check_plugins/current/proto/common/v1/breaking.proto",
    				"testdata/check_plugins/current/proto/common/v1alpha1/breaking.proto"
				],
				"ignore_only": {
					"VALIDATE_ID_DASHLESS": ["testdata/check_plugins/current/vendor/protovalidate/buf/validate"],
				}
			},
			"plugins":[
				{
					"plugin": "buf-plugin-suffix",
					"options": {
						"service_banned_suffixes": ["Mock", "Test"],
						"rpc_banned_suffixes": ["Element"],
						"field_banned_suffixes": ["_uuid"],
						"enum_value_banned_suffixes": ["_invalid"],
						"service_no_change_suffixes": ["Service"],
						"message_no_change_suffixes": ["Request", "Response"],
						"enum_no_change_suffixes": ["State"]
					}
				},
				{"plugin": "buf-plugin-protovalidate-ext"},
				{"plugin": "buf-plugin-rpc-ext"}
			]
		}`,
	)
	// Set the same category for lint and breaking, but ensure that only the lint rules run.
	testRunStdout(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`
testdata/check_plugins/current/proto/common/v1/common.proto:8:5:Field name "common.v1.One.Two.foo_uuid" has banned suffix "_uuid". (buf-plugin-suffix)
		`),
		"lint",
		filepath.Join("testdata", "check_plugins", "current"),
		"--config",
		`{
			"version":"v2",
			"modules": [
				{"path": "testdata/check_plugins/current/proto"},
				{"path": "testdata/check_plugins/current/vendor/protovalidate"}
			],
			"lint": {
				"use": ["ATTRIBUTES_SUFFIXES"]
			},
			"breaking": {
				"use": ["ATTRIBUTES_SUFFIXES"]
			},
			"plugins":[
				{
					"plugin": "buf-plugin-suffix",
					"options": {
						"service_banned_suffixes": ["Mock", "Test"],
						"rpc_banned_suffixes": ["Element"],
						"field_banned_suffixes": ["_uuid"],
						"enum_value_banned_suffixes": ["_invalid"],
						"service_no_change_suffixes": ["Service"],
						"message_no_change_suffixes": ["DONT_CHANGE"],
						"enum_no_change_suffixes": ["DO_NOT_CHANGE"]
					}
				},
				{"plugin": "buf-plugin-protovalidate-ext"},
				{"plugin": "buf-plugin-rpc-ext"}
			]
		}`,
	)

	// tests that if a plugin panics, the buf CLI does not panic
	require.NotPanics(
		t,
		func() {
			appcmdtesting.Run(
				t,
				NewRootCommand,
				appcmdtesting.WithEnv(internaltesting.NewEnvFunc(t)),
				appcmdtesting.WithExpectedExitCode(1),
				appcmdtesting.WithExpectedStderrPartials(
					`panic: this panic is intentional`,
					`Failure: plugin "buf-plugin-panic" failed: Exited with code 2: exit status 2`,
				),
				appcmdtesting.WithArgs(
					"lint",
					filepath.Join("testdata", "check_plugins", "current"),
					"--config",
					`{
					"version":"v2",
					"modules": [
						{"path": "testdata/check_plugins/current/proto"},
						{"path": "testdata/check_plugins/current/vendor/protovalidate"}
					],
					"lint": {
						"use": ["LINT_PANIC"]
					},
					"plugins":[
						{"plugin": "buf-plugin-panic"}
					]
				}`,
				),
			)
		},
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
		bufctl.ExitCodeFileAnnotation,
		`a/v3/a.proto:6:3:Field "1" with name "key" on message "Foo" changed type from "string" to "int32".
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
	testRunStdoutStderrNoWarn(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		`a/v3/a.proto:6:3:Field "1" with name "key" on message "Foo" changed type from "string" to "int32". See https://developers.google.com/protocol-buffers/docs/proto3#updating for wire compatibility rules.`,
		"",
		"breaking",
		filepath.Join(tempDir, "current.binpb"),
		"--against",
		filepath.Join(tempDir, "previous.binpb"),
		"--path",
		filepath.Join("a", "v3"),
		"--exclude-path",
		filepath.Join("a", "v3", "foo"),
		"--config",
		`{"version":"v2","breaking":{"use":["WIRE"]}}`,
	)
}

func TestBreakingWithPlugins(t *testing.T) {
	t.Parallel()
	currentConfig := `{
			"version":"v2",
			"modules": [
				{"path": "testdata/check_plugins/current/proto"},
				{"path": "testdata/check_plugins/current/vendor/protovalidate"}
			],
			"breaking": {
				"use": ["STRING_LEN_RANGE_NO_SHRINK"]
			},
			"plugins":[
				{
					"plugin": "buf-plugin-suffix",
					"options": {
						"service_banned_suffixes": ["Mock", "Test"],
						"rpc_banned_suffixes": ["Element"],
						"field_banned_suffixes": ["_uuid"],
						"enum_value_banned_suffixes": ["_invalid"],
						"service_no_change_suffixes": ["Service"],
						"message_no_change_suffixes": ["Request", "Response"],
						"enum_no_change_suffixes": ["State"]
					}
				},
				{"plugin": "buf-plugin-protovalidate-ext"},
				{"plugin": "buf-plugin-rpc-ext"}
			]
		}`
	previousConfig := strings.ReplaceAll(
		currentConfig,
		"current",
		"previous",
	)
	testRunStdout(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`
testdata/check_plugins/current/proto/common/v1/breaking.proto:10:5:max len requirement reduced from 10 to 5 (buf-plugin-protovalidate-ext)
testdata/check_plugins/current/proto/common/v1alpha1/breaking.proto:10:5:max len requirement reduced from 10 to 5 (buf-plugin-protovalidate-ext)
		`),
		"breaking",
		filepath.Join("testdata", "check_plugins", "current", "proto"),
		"--against",
		filepath.Join("testdata", "check_plugins", "previous", "proto"),
		"--config",
		currentConfig,
		"--against-config",
		previousConfig,
	)

	// ignore unstable package
	currentConfig = `{
		"version":"v2",
		"modules": [
			{"path": "testdata/check_plugins/current/proto"},
			{"path": "testdata/check_plugins/current/vendor/protovalidate"}
		],
		"breaking": {
			"use": ["STRING_LEN_RANGE_NO_SHRINK"],
			"ignore_unstable_packages": true
		},
		"plugins":[
			{
				"plugin": "buf-plugin-suffix",
				"options": {
					"service_banned_suffixes": ["Mock", "Test"],
					"rpc_banned_suffixes": ["Element"],
					"field_banned_suffixes": ["_uuid"],
					"enum_value_banned_suffixes": ["_invalid"],
					"service_no_change_suffixes": ["Service"],
					"message_no_change_suffixes": ["Request", "Response"],
					"enum_no_change_suffixes": ["State"]
				}
			},
			{"plugin": "buf-plugin-protovalidate-ext"},
			{"plugin": "buf-plugin-rpc-ext"}
		]
	}`
	previousConfig = strings.ReplaceAll(
		currentConfig,
		"current",
		"previous",
	)
	testRunStdout(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`
testdata/check_plugins/current/proto/common/v1/breaking.proto:10:5:max len requirement reduced from 10 to 5 (buf-plugin-protovalidate-ext)
	`),
		"breaking",
		filepath.Join("testdata", "check_plugins", "current", "proto"),
		"--against",
		filepath.Join("testdata", "check_plugins", "previous", "proto"),
		"--config",
		currentConfig,
		"--against-config",
		previousConfig,
	)

	// use category
	currentConfig = `{
		"version":"v2",
		"modules": [
			{"path": "testdata/check_plugins/current/proto"},
			{"path": "testdata/check_plugins/current/vendor/protovalidate"}
		],
		"breaking": {
			"use": ["ATTRIBUTES_SUFFIXES"],
		},
		"plugins":[
			{
				"plugin": "buf-plugin-suffix",
				"options": {
					"message_no_change_suffixes": ["DONT_CHANGE"],
					"enum_no_change_suffixes": ["DO_NOT_CHANGE"]
				}
			},
		]
	}`
	previousConfig = strings.ReplaceAll(
		currentConfig,
		"current",
		"previous",
	)
	testRunStdout(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`
testdata/check_plugins/current/proto/common/v1/breaking.proto:14:1:Message "common.v1.MSG_DONT_CHANGE" has a suffix configured for no changes has different fields, previously [], currently [common.v1.MSG_DONT_CHANGE.new_field]. (buf-plugin-suffix)
testdata/check_plugins/current/proto/common/v1/breaking.proto:18:1:Enum "common.v1.E_DO_NOT_CHANGE" has a suffix configured for no changes has different enum values, previously [common.v1.ZERO], currently [common.v1.ONE common.v1.ZERO]. (buf-plugin-suffix)
	`),
		"breaking",
		filepath.Join("testdata", "check_plugins", "current", "proto"),
		"--against",
		filepath.Join("testdata", "check_plugins", "previous", "proto"),
		"--config",
		currentConfig,
		"--against-config",
		previousConfig,
	)

	// use deprecated category
	currentConfig = `{
		"version":"v2",
		"modules": [
			{"path": "testdata/check_plugins/current/proto"},
			{"path": "testdata/check_plugins/current/vendor/protovalidate"}
		],
		"breaking": {
			"use": ["RESOURCE_SUFFIXES"],
		},
		"plugins":[
			{
				"plugin": "buf-plugin-suffix",
				"options": {
					"message_no_change_suffixes": ["DONT_CHANGE"],
					"enum_no_change_suffixes": ["DO_NOT_CHANGE"]
				}
			},
		]
	}`
	previousConfig = strings.ReplaceAll(
		currentConfig,
		"current",
		"previous",
	)
	testRunStdout(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`
testdata/check_plugins/current/proto/common/v1/breaking.proto:14:1:Message "common.v1.MSG_DONT_CHANGE" has a suffix configured for no changes has different fields, previously [], currently [common.v1.MSG_DONT_CHANGE.new_field]. (buf-plugin-suffix)
testdata/check_plugins/current/proto/common/v1/breaking.proto:18:1:Enum "common.v1.E_DO_NOT_CHANGE" has a suffix configured for no changes has different enum values, previously [common.v1.ZERO], currently [common.v1.ONE common.v1.ZERO]. (buf-plugin-suffix)
	`),
		"breaking",
		filepath.Join("testdata", "check_plugins", "current", "proto"),
		"--against",
		filepath.Join("testdata", "check_plugins", "previous", "proto"),
		"--config",
		currentConfig,
		"--against-config",
		previousConfig,
	)

	// use except
	currentConfig = `{
		"version":"v2",
		"modules": [
			{"path": "testdata/check_plugins/current/proto"},
			{"path": "testdata/check_plugins/current/vendor/protovalidate"}
		],
		"breaking": {
			"use": ["RESOURCE_SUFFIXES"],
			"except": ["ENUM_SUFFIXES_NO_CHANGE"]
		},
		"plugins":[
			{
				"plugin": "buf-plugin-suffix",
				"options": {
					"message_no_change_suffixes": ["DONT_CHANGE"],
					"enum_no_change_suffixes": ["DO_NOT_CHANGE"]
				}
			},
		]
	}`
	previousConfig = strings.ReplaceAll(
		currentConfig,
		"current",
		"previous",
	)
	testRunStdout(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`
testdata/check_plugins/current/proto/common/v1/breaking.proto:14:1:Message "common.v1.MSG_DONT_CHANGE" has a suffix configured for no changes has different fields, previously [], currently [common.v1.MSG_DONT_CHANGE.new_field]. (buf-plugin-suffix)
	`),
		"breaking",
		filepath.Join("testdata", "check_plugins", "current", "proto"),
		"--against",
		filepath.Join("testdata", "check_plugins", "previous", "proto"),
		"--config",
		currentConfig,
		"--against-config",
		previousConfig,
	)

	// ignore module
	currentConfig = `{
		"version":"v2",
		"modules": [
			{"path": "testdata/check_plugins/current/proto"},
			{"path": "testdata/check_plugins/current/vendor/protovalidate"}
		],
		"breaking": {
			"use": ["RESOURCE_SUFFIXES"],
			"ignore": ["testdata/check_plugins/current/proto"]
		},
		"plugins":[
			{
				"plugin": "buf-plugin-suffix",
				"options": {
					"message_no_change_suffixes": ["DONT_CHANGE"],
					"enum_no_change_suffixes": ["DO_NOT_CHANGE"]
				}
			},
		]
	}`
	previousConfig = strings.ReplaceAll(
		currentConfig,
		"current",
		"previous",
	)
	testRunStdout(
		t,
		nil,
		0,
		"",
		"breaking",
		filepath.Join("testdata", "check_plugins", "current", "proto"),
		"--against",
		filepath.Join("testdata", "check_plugins", "previous", "proto"),
		"--config",
		currentConfig,
		"--against-config",
		previousConfig,
	)
	// ignore path inside module
	currentConfig = `{
		"version":"v2",
		"modules": [
			{"path": "testdata/check_plugins/current/proto"},
			{"path": "testdata/check_plugins/current/vendor/protovalidate"}
		],
		"breaking": {
			"use": ["RESOURCE_SUFFIXES"],
			"ignore": ["testdata/check_plugins/current/proto/common/v1"]
		},
		"plugins":[
			{
				"plugin": "buf-plugin-suffix",
				"options": {
					"message_no_change_suffixes": ["DONT_CHANGE"],
					"enum_no_change_suffixes": ["DO_NOT_CHANGE"]
				}
			},
		]
	}`
	previousConfig = strings.ReplaceAll(
		currentConfig,
		"current",
		"previous",
	)
	testRunStdout(
		t,
		nil,
		0,
		"",
		"breaking",
		filepath.Join("testdata", "check_plugins", "current", "proto"),
		"--against",
		filepath.Join("testdata", "check_plugins", "previous", "proto"),
		"--config",
		currentConfig,
		"--against-config",
		previousConfig,
	)
	// ignore only
	currentConfig = `{
		"version":"v2",
		"modules": [
			{"path": "testdata/check_plugins/current/proto"},
			{"path": "testdata/check_plugins/current/vendor/protovalidate"}
		],
		"breaking": {
			"use": ["RESOURCE_SUFFIXES"],
			"ignore_only": {
				"MESSAGE_SUFFIXES_NO_CHANGE": ["testdata/check_plugins/current/proto/common"],
			},
		},
		"plugins":[
			{
				"plugin": "buf-plugin-suffix",
				"options": {
					"message_no_change_suffixes": ["DONT_CHANGE"],
					"enum_no_change_suffixes": ["DO_NOT_CHANGE"]
				}
			},
		]
	}`
	previousConfig = strings.ReplaceAll(
		currentConfig,
		"current",
		"previous",
	)
	testRunStdout(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`
testdata/check_plugins/current/proto/common/v1/breaking.proto:18:1:Enum "common.v1.E_DO_NOT_CHANGE" has a suffix configured for no changes has different enum values, previously [common.v1.ZERO], currently [common.v1.ONE common.v1.ZERO]. (buf-plugin-suffix)
	`),
		"breaking",
		filepath.Join("testdata", "check_plugins", "current", "proto"),
		"--against",
		filepath.Join("testdata", "check_plugins", "previous", "proto"),
		"--config",
		currentConfig,
		"--against-config",
		previousConfig,
	)
	// Set the same category for lint and breaking but ensure only breaking is run.
	currentConfig = `{
		"version":"v2",
		"modules": [
			{"path": "testdata/check_plugins/current/proto"},
			{"path": "testdata/check_plugins/current/vendor/protovalidate"}
		],
		"lint": {
			"use": ["ATTRIBUTES_SUFFIXES"]
		},
		"breaking": {
			"use": ["ATTRIBUTES_SUFFIXES"]
		},
		"plugins":[
			{
				"plugin": "buf-plugin-suffix",
				"options": {
					"service_banned_suffixes": ["Mock", "Test"],
					"rpc_banned_suffixes": ["Element"],
					"field_banned_suffixes": ["_uuid"],
					"enum_value_banned_suffixes": ["_invalid"],
					"service_no_change_suffixes": ["Service"],
					"message_no_change_suffixes": ["DONT_CHANGE"],
					"enum_no_change_suffixes": ["DO_NOT_CHANGE"]
				}
			},
		]
	}`
	previousConfig = strings.ReplaceAll(
		currentConfig,
		"current",
		"previous",
	)
	testRunStdout(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`
testdata/check_plugins/current/proto/common/v1/breaking.proto:14:1:Message "common.v1.MSG_DONT_CHANGE" has a suffix configured for no changes has different fields, previously [], currently [common.v1.MSG_DONT_CHANGE.new_field]. (buf-plugin-suffix)
testdata/check_plugins/current/proto/common/v1/breaking.proto:18:1:Enum "common.v1.E_DO_NOT_CHANGE" has a suffix configured for no changes has different enum values, previously [common.v1.ZERO], currently [common.v1.ONE common.v1.ZERO]. (buf-plugin-suffix)
	`),
		"breaking",
		filepath.Join("testdata", "check_plugins", "current", "proto"),
		"--against",
		filepath.Join("testdata", "check_plugins", "previous", "proto"),
		"--config",
		currentConfig,
		"--against-config",
		previousConfig,
	)
}

func TestBreakingAgainstRegistryFlag(t *testing.T) {
	t.Parallel()
	testRunStderr(
		t,
		nil,
		1,
		"Failure: Cannot set both --against and --against-registry",
		"breaking",
		filepath.Join("testdata", "check_plugins", "current", "proto"),
		"--against",
		filepath.Join("testdata", "check_plugins", "previous", "proto"),
		"--against-registry",
	)
	testRunStderr(
		t,
		nil,
		1,
		"Failure: cannot use --against-registry with unnamed module, testdata/success",
		"breaking",
		filepath.Join("testdata", "success"),
		"--against-registry",
	)
}

func TestVersion(t *testing.T) {
	t.Parallel()
	testRunStdout(t, nil, 0, bufcli.Version, "--version")
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

	t.Run("no stdin input from binpb", func(t *testing.T) {
		t.Parallel()
		testRun(
			t,
			0,
			nil,
			nil,
			"convert",
			filepath.Join(tempDir, "image.binpb"),
			"--type",
			"buf.Foo",
			"--from=-#format=binpb",
			"--to=-#format=txtpb",
		)
	})
	t.Run("no stdin input from txtpb", func(t *testing.T) {
		t.Parallel()
		testRun(
			t,
			0,
			nil,
			nil,
			"convert",
			filepath.Join(tempDir, "image.binpb"),
			"--type",
			"buf.Foo",
			"--from=-#format=txtpb",
			"--to=-#format=binpb",
		)
	})
	t.Run("no stdin input from yaml", func(t *testing.T) {
		t.Parallel()
		testRun(
			t,
			0,
			nil,
			nil,
			"convert",
			filepath.Join(tempDir, "image.binpb"),
			"--type",
			"buf.Foo",
			"--from=-#format=yaml",
			"--to=-#format=binpb",
		)
	})
	t.Run("no stdin input from json", func(t *testing.T) {
		t.Parallel()
		testRunStderrContainsNoWarn(
			t,
			nil,
			1,
			[]string{
				"Failure: --from:",
				"json unmarshal:",
				"proto:",
				"syntax error (line 1:1): unexpected token",
			},
			"convert",
			filepath.Join(tempDir, "image.binpb"),
			"--type",
			"buf.Foo",
			"--from=-#format=json",
			"--to=-#format=binpb",
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
		`Failure: --from: ".foo" is not a valid fully qualified type name`,
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
	t.Run("stdin-json-payload-to-yaml-with-image", func(t *testing.T) {
		t.Parallel()
		file, err := os.Open(convertTestDataDir + "/bin_json/payload.json")
		require.NoError(t, err)
		testRunStdoutFile(t,
			file,
			0,
			convertTestDataDir+"/bin_json/payload.yaml",
			"convert",
			"--type=buf.Foo",
			convertTestDataDir+"/bin_json/image.yaml",
			"--from",
			"-#format=json",
			"--to",
			"-#format=yaml",
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
	t.Run("stdin-image-json-to-yaml", func(t *testing.T) {
		t.Parallel()
		file, err := os.Open(convertTestDataDir + "/bin_json/image.json")
		require.NoError(t, err)
		testRunStdoutFile(t,
			file,
			0,
			convertTestDataDir+"/bin_json/payload.yaml",
			"convert",
			"--type=buf.Foo",
			"-#format=json",
			"--from="+convertTestDataDir+"/bin_json/payload.json",
			"--to",
			"-#format=yaml",
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
	t.Run("stdin-image-yaml-to-binpb", func(t *testing.T) {
		t.Parallel()
		file, err := os.Open(convertTestDataDir + "/bin_json/image.yaml")
		require.NoError(t, err)
		testRunStdoutFile(t,
			file,
			0,
			convertTestDataDir+"/bin_json/payload.binpb",
			"convert",
			"--type=buf.Foo",
			"-#format=yaml",
			"--from="+convertTestDataDir+"/bin_json/payload.yaml",
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
		filepath.Join(tempDir, "simple.formatted"),
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"format",
		filepath.Join(tempDir, "simple.formatted"),
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
		bufctl.ExitCodeFileAnnotation,
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
		bufctl.ExitCodeFileAnnotation,
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
	testRunStderrContainsNoWarn(
		t,
		nil,
		1,
		[]string{
			`Failure: cannot use --output when using --write`,
		},
		"format",
		filepath.Join("testdata", "format", "diff"),
		"-w",
		"-o",
		filepath.Join(tempDir, "formatted"),
	)
}

func TestFormatInvalidWriteWithModuleReference(t *testing.T) {
	t.Parallel()
	testRunStderrContainsNoWarn(
		t,
		nil,
		1,
		[]string{
			`Failure: invalid input "buf.build/acme/weather" when using --write: must be a directory or proto file`,
		},
		"format",
		"buf.build/acme/weather",
		"-w",
	)
}

func TestFormatInvalidIncludePackageFiles(t *testing.T) {
	t.Parallel()
	testRunStderrContainsNoWarn(
		t,
		nil,
		1,
		[]string{
			"Failure: cannot specify include_package_files=true with format",
		},
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

func TestProtoFileNoWorkspaceOrModule(t *testing.T) {
	t.Parallel()
	// We can build a simple proto file re that does not belong to any workspace or module
	// based on the directory of the input.
	testRunStdout(
		t,
		nil,
		0,
		"",
		"build",
		filepath.Join("testdata", "protofileref", "noworkspaceormodule", "success", "simple.proto"),
	)
	// However, we should fail if there is any complexity (e.g. an import that cannot be
	// resolved) since there is no workspace or module config to base this off of.
	testRunStdoutStderrNoWarn(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		"", // no stdout
		filepath.FromSlash(`testdata/protofileref/noworkspaceormodule/fail/import.proto:3:8:import "`)+`google/type/date.proto": file does not exist`,
		"build",
		filepath.Join("testdata", "protofileref", "noworkspaceormodule", "fail", "import.proto"),
	)
}

func TestModuleArchiveDir(t *testing.T) {
	// Archive that defines module at input path
	t.Parallel()
	zipDir := createZipFromDir(
		t,
		filepath.Join("testdata", "failarchive"),
		"archive.zip",
	)
	testRunStdout(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`fail/buf/buf.proto:3:1:Files with package "other" must be within a directory "other" relative to root but were in directory "buf".
fail/buf/buf.proto:6:9:Field name "oneTwo" should be lower_snake_case, such as "one_two".`),
		"lint",
		filepath.Join(zipDir, "archive.zip#subdir=fail"),
	)
}

func TestLintDisabledForModuleInWorkspace(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/lint_ignore_disabled/proto/a.proto:3:9:Message name "foo" should be PascalCase, such as "Foo".`),
		"lint",
		filepath.Join("testdata", "lint_ignore_disabled"),
	)
}

func TestLintNoSourceCodeInfoIgnores(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	// Build image without source code info
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		"--exclude-source-info",
		filepath.Join("testdata", "fail"),
		"-o",
		filepath.Join(tempDir, "image.binpb"),
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"lint",
		filepath.Join(tempDir, "image.binpb"),
		"--config",
		`{
		"version": "v2",
		"lint": {
			"ignore_only": {
				"FIELD_LOWER_SNAKE_CASE": ["buf/buf.proto"],
				"PACKAGE_DIRECTORY_MATCH": ["buf/buf.proto"],
				"PACKAGE_VERSION_SUFFIX": ["buf/buf.proto"],
				},
			},
		}`,
	)
}

// testBuildLsFilesFormatImport does effectively an ls-files, but via doing a build of an Image, and then
// listing the files from the image as if --format=import was set.
func testBuildLsFilesFormatImport(t *testing.T, expectedExitCode int, expectedFiles []string, buildArgs ...string) {
	buffer := bytes.NewBuffer(nil)
	testRun(t, expectedExitCode, nil, buffer, append([]string{"build", "-o", "-"}, buildArgs...)...)
	protoImage := &imagev1.Image{}
	err := protoencoding.NewWireUnmarshaler(nil).Unmarshal(buffer.Bytes(), protoImage)
	require.NoError(t, err)
	image, err := bufimage.NewImageForProto(protoImage)
	require.NoError(t, err)
	var paths []string
	for _, imageFile := range image.Files() {
		paths = append(paths, imageFile.Path())
	}
	require.Equal(t, expectedFiles, paths)
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
	data, err := os.ReadFile(filepath.Join(tempDir, "buf.yaml"))
	require.NoError(t, err)
	require.Equal(t, expectedData, string(data))
}

func testRunStdout(t *testing.T, stdin io.Reader, expectedExitCode int, expectedStdout string, args ...string) {
	appcmdtesting.Run(
		t,
		NewRootCommand,
		appcmdtesting.WithEnv(internaltesting.NewEnvFunc(t)),
		appcmdtesting.WithStdin(stdin),
		appcmdtesting.WithExpectedExitCode(expectedExitCode),
		appcmdtesting.WithExpectedStdout(expectedStdout),
		appcmdtesting.WithArgs(args...),
	)
}

func testRunStderr(t *testing.T, stdin io.Reader, expectedExitCode int, expectedStderr string, args ...string) {
	appcmdtesting.Run(
		t,
		NewRootCommand,
		appcmdtesting.WithEnv(internaltesting.NewEnvFunc(t)),
		appcmdtesting.WithStdin(stdin),
		appcmdtesting.WithExpectedExitCode(expectedExitCode),
		appcmdtesting.WithExpectedStderr(expectedStderr),
		appcmdtesting.WithArgs(args...),
	)
}

func testRunStdoutStderrNoWarn(t *testing.T, stdin io.Reader, expectedExitCode int, expectedStdout string, expectedStderr string, args ...string) {
	appcmdtesting.Run(
		t,
		NewRootCommand,
		appcmdtesting.WithEnv(internaltesting.NewEnvFunc(t)),
		appcmdtesting.WithStdin(stdin),
		appcmdtesting.WithExpectedExitCode(expectedExitCode),
		appcmdtesting.WithExpectedStdout(expectedStdout),
		appcmdtesting.WithExpectedStderr(expectedStderr),
		appcmdtesting.WithArgs(
			// we do not want warnings to be part of our stderr test calculation
			append(
				args,
				"--no-warn",
			)...,
		),
	)
}

func testRunStderrContainsNoWarn(t *testing.T, stdin io.Reader, expectedExitCode int, expectedStderrPartials []string, args ...string) {
	appcmdtesting.Run(
		t,
		NewRootCommand,
		appcmdtesting.WithEnv(internaltesting.NewEnvFunc(t)),
		appcmdtesting.WithStdin(stdin),
		appcmdtesting.WithExpectedExitCode(expectedExitCode),
		appcmdtesting.WithExpectedStderrPartials(expectedStderrPartials...),
		appcmdtesting.WithArgs(
			// we do not want warnings to be part of our stderr test calculation
			append(
				args,
				"--no-warn",
			)...,
		),
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
	appcmdtesting.Run(
		t,
		NewRootCommand,
		appcmdtesting.WithEnv(internaltesting.NewEnvFunc(t)),
		appcmdtesting.WithStdin(stdin),
		appcmdtesting.WithStdout(stdout),
		appcmdtesting.WithExpectedExitCode(expectedExitCode),
		appcmdtesting.WithArgs(args...),
	)
}

func getRuleIDsFromLsBreaking(t *testing.T, fileVersion string, useIDs []string, exceptIDs []string) []string {
	t.Helper()
	stdout := bytes.NewBuffer(nil)
	appcmdtesting.Run(
		t,
		NewRootCommand,
		appcmdtesting.WithEnv(internaltesting.NewEnvFunc(t)),
		appcmdtesting.WithStdout(stdout),
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithArgs(
			"config",
			"ls-breaking-rules",
			"--format=json",
			"--configured-only",
			"--config",
			fmt.Sprintf(
				`{ "version": %q, "breaking": { "use": %s, "except": %s } }`,
				fileVersion,
				"["+strings.Join(xslices.Map(useIDs, func(s string) string { return strconv.Quote(s) }), ",")+"]",
				"["+strings.Join(xslices.Map(exceptIDs, func(s string) string { return strconv.Quote(s) }), ",")+"]",
			),
		),
	)
	var ids []string
	decoder := json.NewDecoder(stdout)
	type entry struct {
		ID string
	}
	for {
		var entry entry
		err := decoder.Decode(&entry)
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		ids = append(ids, entry.ID)
	}
	sort.Strings(ids)
	return ids
}

func testLsRuleOutputJSON(
	t *testing.T,
	ruleType check.RuleType,
	config string,
	expectedRules []*outputCheckRule,
) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	var command string
	switch ruleType {
	case check.RuleTypeLint:
		command = "ls-lint-rules"
	case check.RuleTypeBreaking:
		command = "ls-breaking-rules"
	default:
		t.Errorf("invalid rule type %v", ruleType)
		t.FailNow()
	}
	appcmdtesting.Run(
		t,
		NewRootCommand,
		appcmdtesting.WithEnv(internaltesting.NewEnvFunc(t)),
		appcmdtesting.WithStdout(stdout),
		appcmdtesting.WithStderr(stderr),
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithArgs(
			"config",
			command,
			"--configured-only",
			"--config",
			config,
			"--format",
			"json",
		),
	)
	outputRules :=
		xslices.Map(
			xslices.Filter(
				bytes.Split(stdout.Bytes(), []byte("\n")),
				func(outputBytes []byte) bool {
					return len(outputBytes) > 0
				},
			),
			func(outputBytes []byte) *outputCheckRule {
				var outputRule outputCheckRule
				require.NoError(t, json.Unmarshal(outputBytes, &outputRule), "unable to unmarshal %s", string(outputBytes))
				return &outputRule
			},
		)
	require.Equal(t, expectedRules, outputRules)
}

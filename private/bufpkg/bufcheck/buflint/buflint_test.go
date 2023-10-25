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

package buflint_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis/bufanalysistesting"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Hint on how to get these:
// 1. cd into the specific directory
// 2. buf lint --error-format=json | jq '[.path, .start_line, .start_column, .end_line, .end_column, .type] | @csv' --raw-output
//      or
//    buf lint --error-format=json | jq -r '"bufanalysistesting.NewFileAnnotation(t, \"\(.path)\", \(.start_line|tostring), \(.start_column|tostring), \(.end_line|tostring), \(.end_column|tostring), \"\(.type)\"),"'

func TestRunComments(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"comments",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 7, 1, 10, 2, "COMMENT_ENUM"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 8, 3, 8, 28, "COMMENT_ENUM_VALUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 9, 3, 9, 20, "COMMENT_ENUM_VALUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 12, 1, 37, 2, "COMMENT_MESSAGE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 13, 3, 28, 4, "COMMENT_MESSAGE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 14, 5, 17, 6, "COMMENT_ENUM"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 15, 7, 15, 27, "COMMENT_ENUM_VALUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 16, 7, 16, 19, "COMMENT_ENUM_VALUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 18, 5, 23, 6, "COMMENT_MESSAGE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 19, 7, 19, 21, "COMMENT_FIELD"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 20, 7, 22, 8, "COMMENT_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 21, 9, 21, 23, "COMMENT_FIELD"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 24, 5, 24, 19, "COMMENT_FIELD"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 25, 5, 27, 6, "COMMENT_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 26, 7, 26, 21, "COMMENT_FIELD"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 29, 3, 32, 4, "COMMENT_ENUM"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 30, 5, 30, 25, "COMMENT_ENUM_VALUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 31, 5, 31, 17, "COMMENT_ENUM_VALUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 33, 3, 33, 17, "COMMENT_FIELD"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 34, 3, 36, 4, "COMMENT_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 35, 5, 35, 19, "COMMENT_FIELD"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 39, 1, 41, 2, "COMMENT_SERVICE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 40, 3, 40, 74, "COMMENT_RPC"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 104, 1, 107, 2, "COMMENT_ENUM"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 105, 3, 105, 29, "COMMENT_ENUM_VALUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 106, 3, 106, 21, "COMMENT_ENUM_VALUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 109, 1, 134, 2, "COMMENT_MESSAGE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 110, 3, 125, 4, "COMMENT_MESSAGE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 111, 5, 114, 6, "COMMENT_ENUM"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 112, 7, 112, 27, "COMMENT_ENUM_VALUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 113, 7, 113, 19, "COMMENT_ENUM_VALUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 115, 5, 120, 6, "COMMENT_MESSAGE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 116, 7, 116, 21, "COMMENT_FIELD"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 117, 7, 119, 8, "COMMENT_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 118, 9, 118, 23, "COMMENT_FIELD"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 121, 5, 121, 19, "COMMENT_FIELD"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 122, 5, 124, 6, "COMMENT_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 123, 7, 123, 21, "COMMENT_FIELD"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 126, 3, 129, 4, "COMMENT_ENUM"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 127, 5, 127, 25, "COMMENT_ENUM_VALUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 128, 5, 128, 17, "COMMENT_ENUM_VALUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 130, 3, 130, 17, "COMMENT_FIELD"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 131, 3, 133, 4, "COMMENT_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 132, 5, 132, 19, "COMMENT_FIELD"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 136, 1, 139, 2, "COMMENT_SERVICE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 137, 3, 137, 74, "COMMENT_RPC"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 138, 3, 138, 72, "COMMENT_RPC"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 142, 1, 147, 2, "COMMENT_ENUM"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 144, 3, 144, 29, "COMMENT_ENUM_VALUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 146, 3, 146, 21, "COMMENT_ENUM_VALUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 150, 1, 192, 2, "COMMENT_MESSAGE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 152, 3, 177, 4, "COMMENT_MESSAGE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 154, 5, 159, 6, "COMMENT_ENUM"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 156, 7, 156, 27, "COMMENT_ENUM_VALUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 158, 7, 158, 19, "COMMENT_ENUM_VALUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 161, 5, 169, 6, "COMMENT_MESSAGE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 163, 7, 163, 21, "COMMENT_FIELD"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 165, 7, 168, 8, "COMMENT_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 167, 9, 167, 23, "COMMENT_FIELD"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 171, 5, 171, 19, "COMMENT_FIELD"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 173, 5, 176, 6, "COMMENT_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 175, 7, 175, 21, "COMMENT_FIELD"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 179, 3, 184, 4, "COMMENT_ENUM"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 181, 5, 181, 25, "COMMENT_ENUM_VALUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 183, 5, 183, 17, "COMMENT_ENUM_VALUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 186, 3, 186, 17, "COMMENT_FIELD"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 188, 3, 191, 4, "COMMENT_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 190, 5, 190, 19, "COMMENT_FIELD"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 195, 1, 200, 2, "COMMENT_SERVICE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 197, 3, 197, 74, "COMMENT_RPC"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 199, 3, 199, 72, "COMMENT_RPC"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 203, 1, 208, 2, "COMMENT_ENUM"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 205, 3, 205, 29, "COMMENT_ENUM_VALUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 207, 3, 207, 21, "COMMENT_ENUM_VALUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 211, 1, 253, 2, "COMMENT_MESSAGE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 213, 3, 238, 4, "COMMENT_MESSAGE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 215, 5, 220, 6, "COMMENT_ENUM"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 217, 7, 217, 27, "COMMENT_ENUM_VALUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 219, 7, 219, 19, "COMMENT_ENUM_VALUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 222, 5, 230, 6, "COMMENT_MESSAGE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 224, 7, 224, 21, "COMMENT_FIELD"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 226, 7, 229, 8, "COMMENT_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 228, 9, 228, 23, "COMMENT_FIELD"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 232, 5, 232, 19, "COMMENT_FIELD"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 234, 5, 237, 6, "COMMENT_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 236, 7, 236, 21, "COMMENT_FIELD"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 240, 3, 245, 4, "COMMENT_ENUM"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 242, 5, 242, 25, "COMMENT_ENUM_VALUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 244, 5, 244, 17, "COMMENT_ENUM_VALUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 247, 3, 247, 17, "COMMENT_FIELD"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 249, 3, 252, 4, "COMMENT_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 251, 5, 251, 19, "COMMENT_FIELD"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 256, 1, 261, 2, "COMMENT_SERVICE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 258, 3, 258, 74, "COMMENT_RPC"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 260, 3, 260, 72, "COMMENT_RPC"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 263, 1, 265, 2, "COMMENT_MESSAGE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 264, 3, 264, 30, "COMMENT_FIELD"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 273, 3, 273, 72, "COMMENT_RPC"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 277, 3, 277, 72, "COMMENT_RPC"),
	)
}

func TestRunDirectorySamePackage(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"directory_same_package",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 3, 1, 3, 11, "DIRECTORY_SAME_PACKAGE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "no_package.proto", "DIRECTORY_SAME_PACKAGE"),
		bufanalysistesting.NewFileAnnotation(t, "one/c.proto", 3, 1, 3, 11, "DIRECTORY_SAME_PACKAGE"),
		bufanalysistesting.NewFileAnnotation(t, "one/d.proto", 3, 1, 3, 11, "DIRECTORY_SAME_PACKAGE"),
	)
}

func TestRunImportNoPublic(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"import_no_public",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 6, 1, 6, 32, "IMPORT_NO_PUBLIC"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 7, 1, 7, 31, "IMPORT_NO_PUBLIC"),
		bufanalysistesting.NewFileAnnotation(t, "one/one.proto", 6, 1, 6, 32, "IMPORT_NO_PUBLIC"),
	)
}

func TestRunImportNoWeak(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"import_no_weak",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 6, 1, 6, 30, "IMPORT_NO_WEAK"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 7, 1, 7, 29, "IMPORT_NO_WEAK"),
		bufanalysistesting.NewFileAnnotation(t, "one/one.proto", 6, 1, 6, 30, "IMPORT_NO_WEAK"),
	)
}

func TestRunImportUsed(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"import_used",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 5, 1, 5, 25, "IMPORT_USED"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 7, 1, 7, 24, "IMPORT_USED"),
		bufanalysistesting.NewFileAnnotation(t, "one/one.proto", 6, 1, 6, 25, "IMPORT_USED"),
	)
}

func TestRunEnumFirstValueZero(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"enum_first_value_zero",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 12, 13, 12, 14, "ENUM_FIRST_VALUE_ZERO"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 18, 13, 18, 14, "ENUM_FIRST_VALUE_ZERO"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 38, 17, 38, 18, "ENUM_FIRST_VALUE_ZERO"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 44, 17, 44, 18, "ENUM_FIRST_VALUE_ZERO"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 62, 15, 62, 16, "ENUM_FIRST_VALUE_ZERO"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 68, 15, 68, 16, "ENUM_FIRST_VALUE_ZERO"),
	)
}

func TestRunEnumNoAllowAlias(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"enum_no_allow_alias",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 12, 3, 12, 29, "ENUM_NO_ALLOW_ALIAS"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 19, 3, 19, 29, "ENUM_NO_ALLOW_ALIAS"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 41, 7, 41, 33, "ENUM_NO_ALLOW_ALIAS"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 48, 7, 48, 33, "ENUM_NO_ALLOW_ALIAS"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 68, 5, 68, 31, "ENUM_NO_ALLOW_ALIAS"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 75, 5, 75, 31, "ENUM_NO_ALLOW_ALIAS"),
	)
}

func TestRunEnumPascalCase(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"enum_pascal_case",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 16, 6, 16, 10, "ENUM_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 19, 6, 19, 13, "ENUM_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 22, 6, 22, 16, "ENUM_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 25, 6, 25, 15, "ENUM_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 42, 10, 42, 14, "ENUM_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 45, 10, 45, 17, "ENUM_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 48, 10, 48, 20, "ENUM_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 51, 10, 51, 19, "ENUM_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 66, 8, 66, 12, "ENUM_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 69, 8, 69, 15, "ENUM_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 72, 8, 72, 18, "ENUM_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 75, 8, 75, 17, "ENUM_PASCAL_CASE"),
	)
}

func TestRunEnumValuePrefix(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"enum_value_prefix",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 10, 3, 10, 12, "ENUM_VALUE_PREFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 11, 3, 11, 14, "ENUM_VALUE_PREFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 12, 3, 12, 15, "ENUM_VALUE_PREFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 22, 7, 22, 17, "ENUM_VALUE_PREFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 23, 7, 23, 19, "ENUM_VALUE_PREFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 24, 7, 24, 20, "ENUM_VALUE_PREFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 25, 7, 25, 20, "ENUM_VALUE_PREFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 33, 5, 33, 15, "ENUM_VALUE_PREFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 34, 5, 34, 17, "ENUM_VALUE_PREFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 35, 5, 35, 18, "ENUM_VALUE_PREFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 36, 5, 36, 18, "ENUM_VALUE_PREFIX"),
	)
}

func TestRunEnumValueUpperSnakeCase(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"enum_value_upper_snake_case",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 10, 3, 10, 12, "ENUM_VALUE_UPPER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 11, 3, 11, 17, "ENUM_VALUE_UPPER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 12, 3, 12, 14, "ENUM_VALUE_UPPER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 23, 7, 23, 16, "ENUM_VALUE_UPPER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 24, 7, 24, 21, "ENUM_VALUE_UPPER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 25, 7, 25, 18, "ENUM_VALUE_UPPER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 34, 5, 34, 14, "ENUM_VALUE_UPPER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 35, 5, 35, 19, "ENUM_VALUE_UPPER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 36, 5, 36, 16, "ENUM_VALUE_UPPER_SNAKE_CASE"),
	)
}

func TestRunEnumZeroValueSuffix(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"enum_zero_value_suffix",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 14, 3, 14, 18, "ENUM_ZERO_VALUE_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 18, 3, 18, 23, "ENUM_ZERO_VALUE_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 22, 3, 22, 19, "ENUM_ZERO_VALUE_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 36, 7, 36, 22, "ENUM_ZERO_VALUE_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 40, 7, 40, 27, "ENUM_ZERO_VALUE_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 44, 7, 44, 23, "ENUM_ZERO_VALUE_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 56, 5, 56, 20, "ENUM_ZERO_VALUE_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 60, 5, 60, 25, "ENUM_ZERO_VALUE_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 64, 5, 64, 21, "ENUM_ZERO_VALUE_SUFFIX"),
	)
}

func TestRunEnumZeroValueSuffixCustom(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"enum_zero_value_suffix_custom",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 18, 3, 18, 16, "ENUM_ZERO_VALUE_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 22, 3, 22, 23, "ENUM_ZERO_VALUE_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 40, 7, 40, 20, "ENUM_ZERO_VALUE_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 44, 7, 44, 27, "ENUM_ZERO_VALUE_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 60, 5, 60, 18, "ENUM_ZERO_VALUE_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 64, 5, 64, 25, "ENUM_ZERO_VALUE_SUFFIX"),
	)
}

func TestRunFieldLowerSnakeCase(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"field_lower_snake_case",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 8, 9, 8, 13, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 9, 9, 9, 16, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 10, 9, 10, 18, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 11, 9, 11, 19, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 12, 9, 12, 19, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 20, 13, 20, 17, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 21, 13, 21, 20, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 22, 13, 22, 22, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 23, 13, 23, 23, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 24, 13, 24, 23, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 28, 11, 28, 15, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 29, 11, 29, 18, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 30, 11, 30, 20, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 31, 11, 31, 21, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 32, 11, 32, 21, "FIELD_LOWER_SNAKE_CASE"),
	)
}

func TestRunFieldNoDescriptor(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"field_no_descriptor",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 6, 19, 6, 29, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 7, 19, 7, 29, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 8, 19, 8, 29, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 9, 19, 9, 30, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 10, 19, 10, 31, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 11, 19, 11, 30, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 12, 19, 12, 31, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 13, 19, 13, 33, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 19, 23, 19, 33, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 20, 23, 20, 33, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 21, 23, 21, 33, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 22, 23, 22, 34, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 23, 23, 23, 35, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 24, 23, 24, 34, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 25, 23, 25, 35, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 26, 23, 26, 37, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 28, 21, 28, 31, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 29, 21, 29, 31, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 30, 21, 30, 31, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 31, 21, 31, 32, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 32, 21, 32, 33, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 33, 21, 33, 32, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 34, 21, 34, 33, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 35, 21, 35, 35, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 37, 19, 37, 29, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 38, 19, 38, 29, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 39, 19, 39, 29, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 40, 19, 40, 30, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 41, 19, 41, 31, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 42, 19, 42, 30, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 43, 19, 43, 31, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 44, 19, 44, 33, "FIELD_NO_DESCRIPTOR"),
	)
}

func TestRunFileLowerSnakeCase(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"file_lower_snake_case",
		bufanalysistesting.NewFileAnnotationNoLocation(t, "B.proto", "FILE_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "Foo.proto", "FILE_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "aBc.proto", "FILE_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "ab_c_.proto", "FILE_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "fooBar.proto", "FILE_LOWER_SNAKE_CASE"),
	)
}

func TestRunMessagePascalCase(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"message_pascal_case",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 8, 11, 8, 15, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 9, 11, 9, 18, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 10, 13, 10, 23, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 14, 9, 14, 13, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 15, 9, 15, 16, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 16, 9, 16, 19, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 17, 9, 17, 18, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 18, 11, 18, 15, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 19, 11, 19, 18, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 20, 13, 20, 23, "MESSAGE_PASCAL_CASE"),
	)
}

func TestRunOneofLowerSnakeCase(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"oneof_lower_snake_case",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 12, 9, 12, 13, "ONEOF_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 15, 9, 15, 16, "ONEOF_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 18, 9, 18, 18, "ONEOF_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 21, 9, 21, 19, "ONEOF_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 24, 9, 24, 19, "ONEOF_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 38, 13, 38, 17, "ONEOF_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 41, 13, 41, 20, "ONEOF_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 44, 13, 44, 22, "ONEOF_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 47, 13, 47, 23, "ONEOF_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 50, 13, 50, 23, "ONEOF_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 60, 11, 60, 15, "ONEOF_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 63, 11, 63, 18, "ONEOF_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 66, 11, 66, 20, "ONEOF_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 69, 11, 69, 21, "ONEOF_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 72, 11, 72, 21, "ONEOF_LOWER_SNAKE_CASE"),
	)
}

func TestRunPackageDefined(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"package_defined",
		bufanalysistesting.NewFileAnnotationNoLocation(t, "a/no_package.proto", "PACKAGE_DEFINED"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "no_package.proto", "PACKAGE_DEFINED"),
	)
}

func TestRunPackageDirectoryMatch(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"package_directory_match",
		bufanalysistesting.NewFileAnnotation(t, "a/b/a_c.proto", 3, 1, 3, 13, "PACKAGE_DIRECTORY_MATCH"),
		bufanalysistesting.NewFileAnnotation(t, "sub/a/b/a_b.proto", 3, 1, 3, 13, "PACKAGE_DIRECTORY_MATCH"),
	)
}

func TestRunPackageLowerSnakeCase(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"package_lower_snake_case",
		bufanalysistesting.NewFileAnnotation(t, "5.proto", 3, 1, 3, 19, "PACKAGE_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "6.proto", 3, 1, 3, 19, "PACKAGE_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "7.proto", 3, 1, 3, 18, "PACKAGE_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "8.proto", 3, 1, 3, 20, "PACKAGE_LOWER_SNAKE_CASE"),
	)
}

func TestRunPackageNoImportCycle(t *testing.T) {
	t.Parallel()
	testLintWithModifiers(
		t,
		"package_no_import_cycle",
		nil,
		func(image bufimage.Image) bufimage.Image {
			// Testing that import cycles are still detected via imports, but are
			// not reported for imports, only for non-imports.
			var newImageFiles []bufimage.ImageFile
			for _, imageFile := range image.Files() {
				if imageFile.FileDescriptor().GetPackage() == "b" {
					newImageFiles = append(newImageFiles, imageFile.ImageFileWithIsImport(true))
				} else {
					require.False(t, imageFile.IsImport())
					newImageFiles = append(newImageFiles, imageFile)
				}
			}
			newImage, err := bufimage.NewImage(newImageFiles)
			require.NoError(t, err)
			return newImage
		},
		"",
		bufanalysistesting.NewFileAnnotation(t, "c1.proto", 5, 1, 5, 19, "PACKAGE_NO_IMPORT_CYCLE"),
		bufanalysistesting.NewFileAnnotation(t, "d1.proto", 5, 1, 5, 19, "PACKAGE_NO_IMPORT_CYCLE"),
	)
}

func TestRunPackageSameDirectory(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"package_same_directory",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 3, 1, 3, 11, "PACKAGE_SAME_DIRECTORY"),
		bufanalysistesting.NewFileAnnotation(t, "one/a.proto", 3, 1, 3, 11, "PACKAGE_SAME_DIRECTORY"),
	)
}

func TestRunPackageSameDirectoryNoPackage(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"package_same_directory_no_package",
		bufanalysistesting.NewFileAnnotationNoLocation(t, "no_package.proto", "PACKAGE_SAME_DIRECTORY"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "one/no_package.proto", "PACKAGE_SAME_DIRECTORY"),
	)
}

func TestRunPackageSameOptionValue(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"package_same_option_value",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 5, 1, 5, 33, "PACKAGE_SAME_CSHARP_NAMESPACE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 6, 1, 6, 35, "PACKAGE_SAME_JAVA_MULTIPLE_FILES"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 7, 1, 7, 29, "PACKAGE_SAME_JAVA_PACKAGE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 8, 1, 8, 27, "PACKAGE_SAME_GO_PACKAGE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 9, 1, 9, 30, "PACKAGE_SAME_PHP_NAMESPACE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 10, 1, 10, 29, "PACKAGE_SAME_RUBY_PACKAGE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 11, 1, 11, 29, "PACKAGE_SAME_SWIFT_PREFIX"),
		bufanalysistesting.NewFileAnnotation(t, "b.proto", 5, 1, 5, 33, "PACKAGE_SAME_CSHARP_NAMESPACE"),
		bufanalysistesting.NewFileAnnotation(t, "b.proto", 6, 1, 6, 36, "PACKAGE_SAME_JAVA_MULTIPLE_FILES"),
		bufanalysistesting.NewFileAnnotation(t, "b.proto", 7, 1, 7, 29, "PACKAGE_SAME_JAVA_PACKAGE"),
		bufanalysistesting.NewFileAnnotation(t, "b.proto", 8, 1, 8, 27, "PACKAGE_SAME_GO_PACKAGE"),
		bufanalysistesting.NewFileAnnotation(t, "b.proto", 9, 1, 9, 30, "PACKAGE_SAME_PHP_NAMESPACE"),
		bufanalysistesting.NewFileAnnotation(t, "b.proto", 10, 1, 10, 29, "PACKAGE_SAME_RUBY_PACKAGE"),
		bufanalysistesting.NewFileAnnotation(t, "b.proto", 11, 1, 11, 29, "PACKAGE_SAME_SWIFT_PREFIX"),
		bufanalysistesting.NewFileAnnotation(t, "sub/a.proto", 6, 1, 6, 33, "PACKAGE_SAME_CSHARP_NAMESPACE"),
		bufanalysistesting.NewFileAnnotation(t, "sub/a.proto", 7, 1, 7, 35, "PACKAGE_SAME_JAVA_MULTIPLE_FILES"),
		bufanalysistesting.NewFileAnnotation(t, "sub/a.proto", 8, 1, 8, 29, "PACKAGE_SAME_JAVA_PACKAGE"),
		bufanalysistesting.NewFileAnnotation(t, "sub/a.proto", 9, 1, 9, 27, "PACKAGE_SAME_GO_PACKAGE"),
		bufanalysistesting.NewFileAnnotation(t, "sub/a.proto", 10, 1, 10, 30, "PACKAGE_SAME_PHP_NAMESPACE"),
		bufanalysistesting.NewFileAnnotation(t, "sub/a.proto", 11, 1, 11, 29, "PACKAGE_SAME_RUBY_PACKAGE"),
		bufanalysistesting.NewFileAnnotation(t, "sub/a.proto", 12, 1, 12, 29, "PACKAGE_SAME_SWIFT_PREFIX"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "sub/b.proto", "PACKAGE_SAME_CSHARP_NAMESPACE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "sub/b.proto", "PACKAGE_SAME_GO_PACKAGE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "sub/b.proto", "PACKAGE_SAME_JAVA_MULTIPLE_FILES"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "sub/b.proto", "PACKAGE_SAME_JAVA_PACKAGE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "sub/b.proto", "PACKAGE_SAME_PHP_NAMESPACE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "sub/b.proto", "PACKAGE_SAME_RUBY_PACKAGE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "sub/b.proto", "PACKAGE_SAME_SWIFT_PREFIX"),
	)
}

func TestRunPackageVersionSuffix(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"package_version_suffix",
		bufanalysistesting.NewFileAnnotation(t, "foo.proto", 3, 1, 3, 13, "PACKAGE_VERSION_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "foo_bar.proto", 3, 1, 3, 17, "PACKAGE_VERSION_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "foo_bar_v0beta1.proto", 3, 1, 3, 25, "PACKAGE_VERSION_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "foo_bar_v1test_foo.proto", 3, 1, 3, 28, "PACKAGE_VERSION_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "foo_bar_v2beta0.proto", 3, 1, 3, 25, "PACKAGE_VERSION_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "foo_bar_vv1beta1.proto", 3, 1, 3, 26, "PACKAGE_VERSION_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "v1.proto", 3, 1, 3, 12, "PACKAGE_VERSION_SUFFIX"),
	)
}

func TestRunProtovalidateRules(t *testing.T) {
	t.Parallel()
	testLintWithValidate(
		t,
		"protovalidate_rules",
		bufanalysistesting.NewFileAnnotation(t, "any.proto", 26, 5, 26, 64, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "bool.proto", 18, 51, 18, 84, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "bool.proto", 19, 31, 19, 69, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "bool.proto", 20, 50, 20, 88, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "bytes.proto", 21, 5, 21, 48, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "bytes.proto", 26, 5, 26, 45, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "bytes.proto", 31, 5, 31, 45, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "cel_field.proto", 10, 37, 14, 4, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "cel_field.proto", 17, 5, 21, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "cel_field.proto", 29, 5, 33, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "cel_field.proto", 35, 39, 39, 4, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "cel_field.proto", 60, 3, 64, 5, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "cel_field.proto", 76, 5, 80, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "cel_field.proto", 88, 5, 92, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "cel_field.proto", 106, 5, 110, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "cel_field.proto", 116, 5, 120, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "cel_message.proto", 22, 3, 26, 5, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "cel_message.proto", 28, 3, 32, 5, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "cel_message.proto", 34, 3, 38, 5, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "cel_message.proto", 40, 3, 44, 5, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "cel_message.proto", 46, 3, 50, 5, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "cel_message.proto", 52, 3, 56, 5, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "cel_message.proto", 58, 3, 62, 5, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "cel_message.proto", 58, 3, 62, 5, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "cel_message.proto", 64, 3, 68, 5, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "cel_message.proto", 70, 3, 74, 5, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "cel_message.proto", 76, 3, 80, 5, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "cel_message.proto", 82, 5, 86, 7, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "duration.proto", 57, 5, 60, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "duration.proto", 61, 5, 64, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "duration.proto", 68, 5, 71, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "duration.proto", 72, 5, 75, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "duration.proto", 79, 5, 82, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "duration.proto", 83, 5, 86, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "duration.proto", 90, 5, 93, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "duration.proto", 94, 5, 97, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "duration.proto", 105, 5, 108, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "duration.proto", 122, 5, 125, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "duration.proto", 127, 5, 130, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "enum.proto", 28, 5, 28, 40, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "enum.proto", 33, 5, 33, 37, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "extension.proto", 25, 7, 25, 43, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "extension.proto", 30, 7, 30, 47, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "extension.proto", 40, 5, 40, 41, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "extension.proto", 45, 5, 45, 45, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "field.proto", 18, 5, 18, 41, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "field.proto", 19, 5, 19, 45, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "map.proto", 24, 38, 24, 76, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "map.proto", 27, 5, 27, 43, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "map.proto", 29, 5, 29, 43, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "map.proto", 34, 5, 34, 53, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "map.proto", 39, 5, 39, 55, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "map.proto", 44, 5, 44, 47, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "map.proto", 46, 5, 46, 47, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "map.proto", 50, 5, 50, 57, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "number.proto", 20, 5, 20, 42, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "number.proto", 26, 5, 26, 39, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "number.proto", 31, 5, 31, 43, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "number.proto", 34, 5, 34, 43, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "number.proto", 38, 5, 38, 43, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "number.proto", 41, 5, 41, 43, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "number.proto", 45, 5, 45, 42, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "number.proto", 48, 5, 48, 42, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "number.proto", 51, 48, 54, 4, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "number.proto", 62, 5, 62, 39, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "number.proto", 71, 5, 71, 41, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "number.proto", 80, 5, 80, 40, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "number.proto", 89, 5, 89, 41, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "number.proto", 98, 5, 98, 39, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "number.proto", 107, 5, 107, 38, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "number.proto", 140, 5, 140, 56, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "number.proto", 145, 5, 145, 50, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "number.proto", 148, 5, 148, 52, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "repeated.proto", 25, 5, 25, 48, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "repeated.proto", 27, 5, 27, 48, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "repeated.proto", 45, 5, 45, 48, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "repeated.proto", 49, 28, 49, 71, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "repeated.proto", 51, 38, 51, 92, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "string.proto", 31, 5, 31, 46, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "string.proto", 36, 5, 36, 43, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "string.proto", 41, 5, 41, 40, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "string.proto", 46, 5, 46, 40, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "string.proto", 50, 5, 50, 44, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "string.proto", 52, 5, 52, 44, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "string.proto", 56, 5, 56, 44, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "string.proto", 58, 5, 58, 44, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "string.proto", 63, 5, 63, 46, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "string.proto", 68, 5, 68, 46, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "string.proto", 72, 5, 72, 46, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "string.proto", 74, 5, 74, 46, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "string.proto", 78, 5, 78, 46, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "string.proto", 80, 5, 80, 46, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "string.proto", 84, 5, 84, 46, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "string.proto", 86, 5, 86, 44, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "string.proto", 90, 5, 90, 46, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "string.proto", 92, 5, 92, 44, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "string.proto", 97, 5, 97, 47, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "string.proto", 102, 5, 102, 46, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "string.proto", 107, 5, 107, 47, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "string.proto", 112, 5, 112, 47, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "string.proto", 117, 5, 117, 49, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "string.proto", 122, 5, 122, 46, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "string.proto", 127, 5, 127, 47, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "string.proto", 135, 5, 135, 43, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "string.proto", 138, 5, 138, 45, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "timestamp.proto", 57, 5, 60, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "timestamp.proto", 61, 5, 64, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "timestamp.proto", 68, 5, 71, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "timestamp.proto", 72, 5, 75, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "timestamp.proto", 79, 5, 82, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "timestamp.proto", 83, 5, 86, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "timestamp.proto", 90, 5, 93, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "timestamp.proto", 94, 5, 97, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "timestamp.proto", 105, 5, 108, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "timestamp.proto", 124, 5, 127, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "timestamp.proto", 129, 5, 132, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "timestamp.proto", 142, 5, 145, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "timestamp.proto", 150, 5, 153, 6, "PROTOVALIDATE"),
		bufanalysistesting.NewFileAnnotation(t, "timestamp.proto", 157, 5, 160, 6, "PROTOVALIDATE"),
	)
}

func TestRunRPCNoStreaming(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"rpc_no_streaming",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 9, 3, 9, 88, "RPC_NO_CLIENT_STREAMING"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 10, 3, 10, 89, "RPC_NO_SERVER_STREAMING"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 11, 3, 11, 92, "RPC_NO_CLIENT_STREAMING"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 11, 3, 11, 92, "RPC_NO_SERVER_STREAMING"),
	)
}

func TestRunRPCPascalCase(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"rpc_pascal_case",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 11, 7, 11, 11, "RPC_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 12, 7, 12, 14, "RPC_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 13, 7, 13, 17, "RPC_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 14, 7, 14, 16, "RPC_PASCAL_CASE"),
	)
}

func TestRunRPCRequestResponseUnique(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"rpc_request_response_unique",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 21, 3, 21, 32, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 22, 3, 22, 36, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 22, 3, 22, 36, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 23, 3, 23, 35, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 24, 3, 24, 34, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 24, 3, 24, 34, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 25, 3, 25, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 27, 3, 27, 52, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 28, 3, 28, 55, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 29, 3, 29, 69, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 29, 3, 29, 69, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 30, 3, 30, 38, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "b.proto", 8, 3, 8, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "b.proto", 8, 3, 8, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
	)
}

func TestRunRPCRequestResponseUniqueAllowEmptyRequests(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"rpc_request_response_unique_allow_empty_requests",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 21, 3, 21, 32, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 22, 3, 22, 36, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 22, 3, 22, 36, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 23, 3, 23, 35, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 24, 3, 24, 34, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 24, 3, 24, 34, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 25, 3, 25, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 28, 3, 28, 55, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 29, 3, 29, 69, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 29, 3, 29, 69, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 30, 3, 30, 38, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "b.proto", 8, 3, 8, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "b.proto", 8, 3, 8, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
	)
}

func TestRunRPCRequestResponseUniqueAllowEmptyResponses(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"rpc_request_response_unique_allow_empty_responses",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 21, 3, 21, 32, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 22, 3, 22, 36, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 22, 3, 22, 36, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 23, 3, 23, 35, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 24, 3, 24, 34, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 24, 3, 24, 34, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 25, 3, 25, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 27, 3, 27, 52, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 29, 3, 29, 69, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 29, 3, 29, 69, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 30, 3, 30, 38, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "b.proto", 8, 3, 8, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "b.proto", 8, 3, 8, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
	)
}

func TestRunRPCRequestResponseUniqueAllowEmptyRequestsAndResponses(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"rpc_request_response_unique_allow_empty_requests_and_responses",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 21, 3, 21, 32, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 22, 3, 22, 36, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 22, 3, 22, 36, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 23, 3, 23, 35, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 24, 3, 24, 34, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 24, 3, 24, 34, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 25, 3, 25, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 30, 3, 30, 38, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "b.proto", 8, 3, 8, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "b.proto", 8, 3, 8, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
	)
}

func TestRunRPCRequestResponseUniqueAllowSame(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"rpc_request_response_unique_allow_same",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 21, 3, 21, 32, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 22, 3, 22, 36, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 23, 3, 23, 35, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 24, 3, 24, 34, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 24, 3, 24, 34, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 25, 3, 25, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 27, 3, 27, 52, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 28, 3, 28, 55, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 29, 3, 29, 69, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "b.proto", 8, 3, 8, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "b.proto", 8, 3, 8, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
	)
}

func TestRunRPCRequestResponseUniqueAllowSameAndEmptyRequestResponses(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"rpc_request_response_unique_allow_same_and_empty_request_responses",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 21, 3, 21, 32, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 22, 3, 22, 36, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 23, 3, 23, 35, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 24, 3, 24, 34, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 24, 3, 24, 34, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 25, 3, 25, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "b.proto", 8, 3, 8, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
		bufanalysistesting.NewFileAnnotation(t, "b.proto", 8, 3, 8, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
	)
}

func TestRunRPCStandardName(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"rpc_standard_name",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 10, 19, 10, 22, "RPC_REQUEST_STANDARD_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 11, 50, 11, 53, "RPC_RESPONSE_STANDARD_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 12, 27, 12, 48, "RPC_REQUEST_STANDARD_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 13, 66, 13, 87, "RPC_RESPONSE_STANDARD_NAME"),
	)
}

func TestRunRPCStandardNameAllowEmpty(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"rpc_standard_name_allow_empty",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 10, 19, 10, 22, "RPC_REQUEST_STANDARD_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 11, 50, 11, 53, "RPC_RESPONSE_STANDARD_NAME"),
	)
}

func TestRunServicePascalCase(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"service_pascal_case",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 8, 9, 8, 13, "SERVICE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 9, 9, 9, 16, "SERVICE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 10, 9, 10, 19, "SERVICE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 11, 9, 11, 18, "SERVICE_PASCAL_CASE"),
	)
}

func TestRunServiceSuffix(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"service_suffix",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 8, 9, 8, 13, "SERVICE_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 9, 9, 9, 16, "SERVICE_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 10, 9, 10, 21, "SERVICE_SUFFIX"),
	)
}

func TestRunServiceSuffixCustom(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"service_suffix_custom",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 8, 9, 8, 13, "SERVICE_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 9, 9, 9, 20, "SERVICE_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 10, 9, 10, 17, "SERVICE_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 11, 9, 11, 17, "SERVICE_SUFFIX"),
	)
}

func TestRunSyntaxSpecified(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"syntax_specified",
		bufanalysistesting.NewFileAnnotationNoLocation(t, "a/a.proto", "SYNTAX_SPECIFIED"),
	)
}

func TestRunIgnores1(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"ignores1",
		bufanalysistesting.NewFileAnnotation(t, "buf/bar/bar.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/bar/bar.proto", 9, 9, 9, 12, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/bar/bar.proto", 13, 6, 13, 9, "ENUM_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/bar/bar2.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/bar/bar2.proto", 9, 9, 9, 13, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/bar/bar2.proto", 13, 6, 13, 10, "ENUM_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/buf.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/buf.proto", 9, 9, 9, 12, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/buf.proto", 13, 6, 13, 9, "ENUM_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/foo/bar/bar.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/foo/bar/bar.proto", 9, 9, 9, 12, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/foo/bar/bar.proto", 13, 6, 13, 9, "ENUM_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/foo/baz/baz.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/foo/baz/baz.proto", 9, 9, 9, 12, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/foo/baz/baz.proto", 13, 6, 13, 9, "ENUM_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/foo/buf.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/foo/buf.proto", 9, 9, 9, 12, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/foo/buf.proto", 13, 6, 13, 9, "ENUM_PASCAL_CASE"),
	)
}

func TestRunIgnores2(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"ignores2",
		bufanalysistesting.NewFileAnnotation(t, "buf/bar/bar.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/bar/bar.proto", 9, 9, 9, 12, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/bar/bar.proto", 13, 6, 13, 9, "ENUM_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/buf.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/buf.proto", 9, 9, 9, 12, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/buf.proto", 13, 6, 13, 9, "ENUM_PASCAL_CASE"),
	)
}

func TestRunIgnores3(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"ignores3",
		bufanalysistesting.NewFileAnnotation(t, "buf/bar/bar.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/bar/bar2.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/bar/bar2.proto", 9, 9, 9, 13, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/bar/bar2.proto", 13, 6, 13, 10, "ENUM_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/buf.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/buf.proto", 9, 9, 9, 12, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/buf.proto", 13, 6, 13, 9, "ENUM_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/foo/baz/baz.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/foo/baz/baz.proto", 9, 9, 9, 12, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/foo/baz/baz.proto", 13, 6, 13, 9, "ENUM_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/foo/buf.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/foo/buf.proto", 9, 9, 9, 12, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/foo/buf.proto", 13, 6, 13, 9, "ENUM_PASCAL_CASE"),
	)
}

func TestRunIgnores4(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"ignores4",
		bufanalysistesting.NewFileAnnotation(t, "buf/bar/bar.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/bar/bar2.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/bar/bar2.proto", 9, 9, 9, 13, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/bar/bar2.proto", 13, 6, 13, 10, "ENUM_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/buf.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/buf.proto", 9, 9, 9, 12, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/buf.proto", 13, 6, 13, 9, "ENUM_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/foo/baz/baz.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/foo/baz/baz.proto", 9, 9, 9, 12, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/foo/baz/baz.proto", 13, 6, 13, 9, "ENUM_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/foo/buf.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/foo/buf.proto", 9, 9, 9, 12, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/foo/buf.proto", 13, 6, 13, 9, "ENUM_PASCAL_CASE"),
	)
}

func TestCommentIgnoresOff(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"comment_ignores",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 9, 1, 9, 11, "PACKAGE_DIRECTORY_MATCH"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 9, 1, 9, 11, "PACKAGE_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 9, 1, 9, 11, "PACKAGE_VERSION_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 12, 1, 12, 45, "IMPORT_NO_PUBLIC"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 15, 6, 15, 13, "ENUM_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 17, 3, 17, 29, "ENUM_NO_ALLOW_ALIAS"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 20, 3, 20, 14, "ENUM_VALUE_UPPER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 20, 3, 20, 14, "ENUM_ZERO_VALUE_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 22, 3, 22, 13, "ENUM_VALUE_UPPER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 24, 3, 24, 13, "ENUM_VALUE_UPPER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 28, 9, 28, 19, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 30, 11, 30, 21, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 32, 13, 32, 23, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 34, 12, 34, 19, "ENUM_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 36, 9, 36, 35, "ENUM_NO_ALLOW_ALIAS"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 39, 9, 39, 20, "ENUM_VALUE_UPPER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 39, 9, 39, 20, "ENUM_ZERO_VALUE_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 41, 9, 41, 19, "ENUM_VALUE_UPPER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 43, 9, 43, 19, "ENUM_VALUE_UPPER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 46, 13, 46, 16, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 48, 13, 48, 16, "ONEOF_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 50, 15, 50, 18, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 54, 11, 54, 14, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 56, 11, 56, 14, "ONEOF_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 58, 13, 58, 16, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 62, 9, 62, 12, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 64, 9, 64, 12, "ONEOF_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 66, 11, 66, 14, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 71, 9, 71, 19, "SERVICE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 75, 7, 75, 16, "RPC_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 75, 17, 75, 38, "RPC_REQUEST_STANDARD_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 75, 49, 75, 70, "RPC_RESPONSE_STANDARD_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "b.proto", 5, 1, 5, 11, "PACKAGE_DIRECTORY_MATCH"),
		bufanalysistesting.NewFileAnnotation(t, "b.proto", 5, 1, 5, 11, "PACKAGE_VERSION_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "b.proto", 9, 26, 9, 28, "ENUM_FIRST_VALUE_ZERO"),
	)
}

func TestCommentIgnoresOn(t *testing.T) {
	t.Parallel()
	testLintWithModifiers(
		t,
		"comment_ignores",
		func(config *bufconfig.Config) {
			config.Lint.AllowCommentIgnores = true
		},
		nil,
		"",
	)
}

func TestCommentIgnoresCascadeOff(t *testing.T) {
	t.Parallel()
	testLint(
		t,
		"comment_ignores_cascade",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 13, 6, 13, 13, "ENUM_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 15, 3, 15, 29, "ENUM_NO_ALLOW_ALIAS"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 16, 3, 16, 14, "ENUM_VALUE_UPPER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 16, 3, 16, 14, "ENUM_ZERO_VALUE_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 17, 3, 17, 13, "ENUM_VALUE_UPPER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 18, 3, 18, 13, "ENUM_VALUE_UPPER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 24, 9, 24, 19, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 28, 11, 28, 21, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 32, 13, 32, 23, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 35, 12, 35, 19, "ENUM_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 37, 9, 37, 35, "ENUM_NO_ALLOW_ALIAS"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 39, 9, 39, 20, "ENUM_VALUE_UPPER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 39, 9, 39, 20, "ENUM_ZERO_VALUE_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 40, 9, 40, 19, "ENUM_VALUE_UPPER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 41, 9, 41, 19, "ENUM_VALUE_UPPER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 43, 13, 43, 16, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 44, 13, 44, 16, "ONEOF_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 45, 15, 45, 18, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 48, 11, 48, 14, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 49, 11, 49, 14, "ONEOF_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 50, 13, 50, 16, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 53, 9, 53, 12, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 54, 9, 54, 12, "ONEOF_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 55, 11, 55, 14, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 63, 9, 63, 19, "SERVICE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 64, 7, 64, 16, "RPC_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 64, 17, 64, 38, "RPC_REQUEST_STANDARD_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 64, 49, 64, 70, "RPC_RESPONSE_STANDARD_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 68, 9, 68, 19, "SERVICE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 72, 7, 72, 16, "RPC_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 72, 17, 72, 38, "RPC_REQUEST_STANDARD_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 72, 49, 72, 70, "RPC_RESPONSE_STANDARD_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "b.proto", 9, 26, 9, 28, "ENUM_FIRST_VALUE_ZERO"),
	)
}

func TestCommentIgnoresCascadeOn(t *testing.T) {
	t.Parallel()
	testLintWithModifiers(
		t,
		"comment_ignores_cascade",
		func(config *bufconfig.Config) {
			config.Lint.AllowCommentIgnores = true
		},
		nil,
		"",
	)
}

func testLint(
	t *testing.T,
	relDirPath string,
	expectedFileAnnotations ...bufanalysis.FileAnnotation,
) {
	testLintWithModifiers(
		t,
		relDirPath,
		nil,
		nil,
		"",
		expectedFileAnnotations...,
	)
}

func testLintWithValidate(
	t *testing.T,
	relDirPath string,
	expectedFileAnnotations ...bufanalysis.FileAnnotation,
) {
	testLintWithModifiers(
		t,
		relDirPath,
		func(config *bufconfig.Config) {
			config.Lint.IgnoreRootPaths = []string{"buf/validate"}
		},
		nil,
		"deps/protovalidate",
		expectedFileAnnotations...,
	)
}

func testLintWithModifiers(
	t *testing.T,
	relDirPath string,
	configModifier func(*bufconfig.Config),
	imageModifier func(bufimage.Image) bufimage.Image,
	dependencyPathPrefix string,
	expectedFileAnnotations ...bufanalysis.FileAnnotation,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	logger := zap.NewNop()

	dirPath := filepath.Join("testdata", relDirPath)

	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(
		dirPath,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	require.NoError(t, err)

	config := testGetConfig(t, readWriteBucket)
	if configModifier != nil {
		configModifier(config)
	}

	var module *bufmodulebuild.BuiltModule
	if dependencyPathPrefix != "" {
		dependencyReadWriteBucket, err := storageosProvider.NewReadWriteBucket(
			filepath.Join("testdata", dependencyPathPrefix),
			storageos.ReadWriteBucketWithSymlinksIfSupported(),
		)
		require.NoError(t, err)
		module, err = bufmodulebuild.NewModuleBucketBuilder().BuildForBucket(
			context.Background(),
			storage.MultiReadBucket(dependencyReadWriteBucket, readWriteBucket),
			config.Build,
		)
		require.NoError(t, err)
	} else {
		module, err = bufmodulebuild.NewModuleBucketBuilder().BuildForBucket(
			context.Background(),
			readWriteBucket,
			config.Build,
		)
		require.NoError(t, err)
	}

	image, fileAnnotations, err := bufimagebuild.NewBuilder(
		zap.NewNop(),
		bufmodule.NewNopModuleReader(),
	).Build(
		ctx,
		module,
	)
	require.NoError(t, err)
	require.Empty(t, fileAnnotations)
	if imageModifier != nil {
		image = imageModifier(image)
	}

	handler := buflint.NewHandler(logger)
	fileAnnotations, err = handler.Check(
		ctx,
		config.Lint,
		image,
	)
	assert.NoError(t, err)
	bufanalysistesting.AssertFileAnnotationsEqual(
		t,
		expectedFileAnnotations,
		fileAnnotations,
	)
}

func testGetConfig(
	t *testing.T,
	readBucket storage.ReadBucket,
) *bufconfig.Config {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	config, err := bufconfig.GetConfigForBucket(ctx, readBucket)
	require.NoError(t, err)
	return config
}

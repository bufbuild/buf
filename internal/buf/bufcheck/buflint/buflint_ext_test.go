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

package buflint_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufanalysis/bufanalysistesting"
	"github.com/bufbuild/buf/internal/buf/bufcheck/buflint"
	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Hint on how to get these:
// 1. cd into the specific diriectory
// 2. buf check lint --error-format=json | jq '[.path, ".", .start_line, .start_column, .end_line, .end_column, .type] | @csv' --raw-output

func TestRunComments(t *testing.T) {
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
	)
}

func TestRunDirectorySamePackage(t *testing.T) {
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
	testLint(
		t,
		"import_no_public",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 6, 1, 6, 32, "IMPORT_NO_PUBLIC"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 7, 1, 7, 31, "IMPORT_NO_PUBLIC"),
		bufanalysistesting.NewFileAnnotation(t, "one/one.proto", 6, 1, 6, 32, "IMPORT_NO_PUBLIC"),
	)
}

func TestRunImportNoWeak(t *testing.T) {
	testLint(
		t,
		"import_no_weak",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 6, 1, 6, 30, "IMPORT_NO_WEAK"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 7, 1, 7, 29, "IMPORT_NO_WEAK"),
		bufanalysistesting.NewFileAnnotation(t, "one/one.proto", 6, 1, 6, 30, "IMPORT_NO_WEAK"),
	)
}

func TestRunEnumFirstValueZero(t *testing.T) {
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
	testLint(
		t,
		"field_no_descriptor",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 6, 10, 6, 20, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 7, 10, 7, 20, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 8, 10, 8, 20, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 9, 10, 9, 21, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 10, 10, 10, 22, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 11, 10, 11, 21, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 12, 10, 12, 22, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 13, 10, 13, 24, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 19, 14, 19, 24, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 20, 14, 20, 24, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 21, 14, 21, 24, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 22, 14, 22, 25, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 23, 14, 23, 26, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 24, 14, 24, 25, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 25, 14, 25, 26, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 26, 14, 26, 28, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 28, 12, 28, 22, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 29, 12, 29, 22, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 30, 12, 30, 22, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 31, 12, 31, 23, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 32, 12, 32, 24, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 33, 12, 33, 23, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 34, 12, 34, 24, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 35, 12, 35, 26, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 37, 10, 37, 20, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 38, 10, 38, 20, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 39, 10, 39, 20, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 40, 10, 40, 21, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 41, 10, 41, 22, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 42, 10, 42, 21, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 43, 10, 43, 22, "FIELD_NO_DESCRIPTOR"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 44, 10, 44, 24, "FIELD_NO_DESCRIPTOR"),
	)
}

func TestRunFileLowerSnakeCase(t *testing.T) {
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
	testLint(
		t,
		"package_defined",
		bufanalysistesting.NewFileAnnotationNoLocation(t, "a/no_package.proto", "PACKAGE_DEFINED"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "no_package.proto", "PACKAGE_DEFINED"),
	)
}

func TestRunPackageDirectoryMatch(t *testing.T) {
	testLint(
		t,
		"package_directory_match",
		bufanalysistesting.NewFileAnnotation(t, "a/b/a_c.proto", 3, 1, 3, 13, "PACKAGE_DIRECTORY_MATCH"),
		bufanalysistesting.NewFileAnnotation(t, "sub/a/b/a_b.proto", 3, 1, 3, 13, "PACKAGE_DIRECTORY_MATCH"),
	)
}

func TestRunPackageLowerSnakeCase(t *testing.T) {
	testLint(
		t,
		"package_lower_snake_case",
		bufanalysistesting.NewFileAnnotation(t, "5.proto", 3, 1, 3, 19, "PACKAGE_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "6.proto", 3, 1, 3, 19, "PACKAGE_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "7.proto", 3, 1, 3, 18, "PACKAGE_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "8.proto", 3, 1, 3, 20, "PACKAGE_LOWER_SNAKE_CASE"),
	)
}

func TestRunPackageSameDirectory(t *testing.T) {
	testLint(
		t,
		"package_same_directory",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 3, 1, 3, 11, "PACKAGE_SAME_DIRECTORY"),
		bufanalysistesting.NewFileAnnotation(t, "one/a.proto", 3, 1, 3, 11, "PACKAGE_SAME_DIRECTORY"),
	)
}

func TestRunPackageSameDirectoryNoPackage(t *testing.T) {
	testLint(
		t,
		"package_same_directory_no_package",
		bufanalysistesting.NewFileAnnotationNoLocation(t, "no_package.proto", "PACKAGE_SAME_DIRECTORY"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "one/no_package.proto", "PACKAGE_SAME_DIRECTORY"),
	)
}

func TestRunPackageSameOptionValue(t *testing.T) {
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

func TestRunRPCNoStreaming(t *testing.T) {
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
	testLint(
		t,
		"rpc_standard_name_allow_empty",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 10, 19, 10, 22, "RPC_REQUEST_STANDARD_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 11, 50, 11, 53, "RPC_RESPONSE_STANDARD_NAME"),
	)
}

func TestRunServicePascalCase(t *testing.T) {
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
	testLint(
		t,
		"service_suffix",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 8, 9, 8, 13, "SERVICE_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 9, 9, 9, 16, "SERVICE_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 10, 9, 10, 21, "SERVICE_SUFFIX"),
	)
}

func TestRunServiceSuffixCustom(t *testing.T) {
	testLint(
		t,
		"service_suffix_custom",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 8, 9, 8, 13, "SERVICE_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 9, 9, 9, 20, "SERVICE_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 10, 9, 10, 17, "SERVICE_SUFFIX"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 11, 9, 11, 17, "SERVICE_SUFFIX"),
	)
}

func TestRunIgnores1(t *testing.T) {
	testLint(
		t,
		"ignores",
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
	testLintExternalConfigModifier(
		t,
		"ignores",
		func(externalConfig *bufconfig.ExternalConfig) {
			externalConfig.Lint.Ignore = []string{
				"buf/bar/bar2.proto",
				"buf/foo",
			}
		},
		bufanalysistesting.NewFileAnnotation(t, "buf/bar/bar.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/bar/bar.proto", 9, 9, 9, 12, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/bar/bar.proto", 13, 6, 13, 9, "ENUM_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/buf.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/buf.proto", 9, 9, 9, 12, "MESSAGE_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "buf/buf.proto", 13, 6, 13, 9, "ENUM_PASCAL_CASE"),
	)
}

func TestRunIgnores3(t *testing.T) {
	testLintExternalConfigModifier(
		t,
		"ignores",
		func(externalConfig *bufconfig.ExternalConfig) {
			externalConfig.Lint.IgnoreOnly = map[string][]string{
				"ENUM_PASCAL_CASE": {
					"buf/bar/bar.proto",
					"buf/foo/bar",
					"buf/foo/bar",
				},
				"MESSAGE_PASCAL_CASE": {
					"buf/bar/bar.proto",
				},
				"STYLE_BASIC": {
					"buf/foo/bar",
				},
			}
		},
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
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 74, 7, 74, 16, "RPC_PASCAL_CASE"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 76, 7, 76, 28, "RPC_REQUEST_STANDARD_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 79, 7, 79, 28, "RPC_RESPONSE_STANDARD_NAME"),
	)
}

func TestCommentIgnoresOn(t *testing.T) {
	testLintExternalConfigModifier(
		t,
		"comment_ignores",
		func(externalConfig *bufconfig.ExternalConfig) {
			externalConfig.Lint.AllowCommentIgnores = true
		},
	)
}

func testLint(
	t *testing.T,
	relDirPath string,
	expectedFileAnnotations ...bufanalysis.FileAnnotation,
) {
	testLintExternalConfigModifier(
		t,
		relDirPath,
		nil,
		expectedFileAnnotations...,
	)
}

func testLintExternalConfigModifier(
	t *testing.T,
	relDirPath string,
	modifier func(*bufconfig.ExternalConfig),
	expectedFileAnnotations ...bufanalysis.FileAnnotation,
) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	logger := zap.NewNop()

	dirPath := filepath.Join("testdata", relDirPath)

	readWriteBucket, err := storageos.NewReadWriteBucket(dirPath)
	require.NoError(t, err)

	var configProviderOptions []bufconfig.ProviderOption
	if modifier != nil {
		configProviderOptions = append(
			configProviderOptions,
			bufconfig.ProviderWithExternalConfigModifier(
				func(externalConfig *bufconfig.ExternalConfig) error {
					modifier(externalConfig)
					return nil
				},
			),
		)
	}
	configProvider := bufconfig.NewProvider(logger, configProviderOptions...)
	config := testGetConfig(t, configProvider, readWriteBucket)

	module, err := bufmodulebuild.NewModuleBucketBuilder(zap.NewNop()).BuildForBucket(
		context.Background(),
		readWriteBucket,
		config.Build,
	)
	require.NoError(t, err)
	moduleFileSet, err := bufmodulebuild.NewModuleFileSetBuilder(
		zap.NewNop(),
		bufmodule.NewNopModuleReader(),
	).Build(
		context.Background(),
		module,
	)
	require.NoError(t, err)
	image, fileAnnotations, err := bufimagebuild.NewBuilder(zap.NewNop()).Build(
		ctx,
		moduleFileSet,
	)
	require.NoError(t, err)
	require.Empty(t, fileAnnotations)
	image = bufimage.ImageWithoutImports(image)

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
	configProvider bufconfig.Provider,
	readBucket storage.ReadBucket,
) *bufconfig.Config {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	config, err := configProvider.GetConfig(ctx, readBucket)
	require.NoError(t, err)
	return config
}

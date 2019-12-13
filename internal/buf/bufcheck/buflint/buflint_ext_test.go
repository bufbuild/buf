package buflint_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/bufbuild/buf/internal/buf/bufbuild"
	"github.com/bufbuild/buf/internal/buf/bufcheck/buflint"
	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/pkg/analysis"
	"github.com/bufbuild/buf/internal/pkg/analysis/analysistesting"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/bufbuild/buf/internal/pkg/storage/storageutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRunComments(t *testing.T) {
	testLint(
		t,
		"comments",
		analysistesting.NewAnnotation("a.proto", 7, 1, 10, 2, "COMMENT_ENUM"),
		analysistesting.NewAnnotation("a.proto", 8, 3, 8, 28, "COMMENT_ENUM_VALUE"),
		analysistesting.NewAnnotation("a.proto", 9, 3, 9, 20, "COMMENT_ENUM_VALUE"),
		analysistesting.NewAnnotation("a.proto", 12, 1, 37, 2, "COMMENT_MESSAGE"),
		analysistesting.NewAnnotation("a.proto", 13, 3, 28, 4, "COMMENT_MESSAGE"),
		analysistesting.NewAnnotation("a.proto", 14, 5, 17, 6, "COMMENT_ENUM"),
		analysistesting.NewAnnotation("a.proto", 15, 7, 15, 27, "COMMENT_ENUM_VALUE"),
		analysistesting.NewAnnotation("a.proto", 16, 7, 16, 19, "COMMENT_ENUM_VALUE"),
		analysistesting.NewAnnotation("a.proto", 18, 5, 23, 6, "COMMENT_MESSAGE"),
		analysistesting.NewAnnotation("a.proto", 19, 7, 19, 21, "COMMENT_FIELD"),
		analysistesting.NewAnnotation("a.proto", 20, 7, 22, 8, "COMMENT_ONEOF"),
		analysistesting.NewAnnotation("a.proto", 21, 9, 21, 23, "COMMENT_FIELD"),
		analysistesting.NewAnnotation("a.proto", 24, 5, 24, 19, "COMMENT_FIELD"),
		analysistesting.NewAnnotation("a.proto", 25, 5, 27, 6, "COMMENT_ONEOF"),
		analysistesting.NewAnnotation("a.proto", 26, 7, 26, 21, "COMMENT_FIELD"),
		analysistesting.NewAnnotation("a.proto", 29, 3, 32, 4, "COMMENT_ENUM"),
		analysistesting.NewAnnotation("a.proto", 30, 5, 30, 25, "COMMENT_ENUM_VALUE"),
		analysistesting.NewAnnotation("a.proto", 31, 5, 31, 17, "COMMENT_ENUM_VALUE"),
		analysistesting.NewAnnotation("a.proto", 33, 3, 33, 17, "COMMENT_FIELD"),
		analysistesting.NewAnnotation("a.proto", 34, 3, 36, 4, "COMMENT_ONEOF"),
		analysistesting.NewAnnotation("a.proto", 35, 5, 35, 19, "COMMENT_FIELD"),
		analysistesting.NewAnnotation("a.proto", 39, 1, 41, 2, "COMMENT_SERVICE"),
		analysistesting.NewAnnotation("a.proto", 40, 3, 40, 74, "COMMENT_RPC"),
		analysistesting.NewAnnotation("a.proto", 104, 1, 107, 2, "COMMENT_ENUM"),
		analysistesting.NewAnnotation("a.proto", 105, 3, 105, 29, "COMMENT_ENUM_VALUE"),
		analysistesting.NewAnnotation("a.proto", 106, 3, 106, 21, "COMMENT_ENUM_VALUE"),
		analysistesting.NewAnnotation("a.proto", 109, 1, 134, 2, "COMMENT_MESSAGE"),
		analysistesting.NewAnnotation("a.proto", 110, 3, 125, 4, "COMMENT_MESSAGE"),
		analysistesting.NewAnnotation("a.proto", 111, 5, 114, 6, "COMMENT_ENUM"),
		analysistesting.NewAnnotation("a.proto", 112, 7, 112, 27, "COMMENT_ENUM_VALUE"),
		analysistesting.NewAnnotation("a.proto", 113, 7, 113, 19, "COMMENT_ENUM_VALUE"),
		analysistesting.NewAnnotation("a.proto", 115, 5, 120, 6, "COMMENT_MESSAGE"),
		analysistesting.NewAnnotation("a.proto", 116, 7, 116, 21, "COMMENT_FIELD"),
		analysistesting.NewAnnotation("a.proto", 117, 7, 119, 8, "COMMENT_ONEOF"),
		analysistesting.NewAnnotation("a.proto", 118, 9, 118, 23, "COMMENT_FIELD"),
		analysistesting.NewAnnotation("a.proto", 121, 5, 121, 19, "COMMENT_FIELD"),
		analysistesting.NewAnnotation("a.proto", 122, 5, 124, 6, "COMMENT_ONEOF"),
		analysistesting.NewAnnotation("a.proto", 123, 7, 123, 21, "COMMENT_FIELD"),
		analysistesting.NewAnnotation("a.proto", 126, 3, 129, 4, "COMMENT_ENUM"),
		analysistesting.NewAnnotation("a.proto", 127, 5, 127, 25, "COMMENT_ENUM_VALUE"),
		analysistesting.NewAnnotation("a.proto", 128, 5, 128, 17, "COMMENT_ENUM_VALUE"),
		analysistesting.NewAnnotation("a.proto", 130, 3, 130, 17, "COMMENT_FIELD"),
		analysistesting.NewAnnotation("a.proto", 131, 3, 133, 4, "COMMENT_ONEOF"),
		analysistesting.NewAnnotation("a.proto", 132, 5, 132, 19, "COMMENT_FIELD"),
		analysistesting.NewAnnotation("a.proto", 136, 1, 139, 2, "COMMENT_SERVICE"),
		analysistesting.NewAnnotation("a.proto", 137, 3, 137, 74, "COMMENT_RPC"),
		analysistesting.NewAnnotation("a.proto", 138, 3, 138, 72, "COMMENT_RPC"),
		analysistesting.NewAnnotation("a.proto", 142, 1, 147, 2, "COMMENT_ENUM"),
		analysistesting.NewAnnotation("a.proto", 144, 3, 144, 29, "COMMENT_ENUM_VALUE"),
		analysistesting.NewAnnotation("a.proto", 146, 3, 146, 21, "COMMENT_ENUM_VALUE"),
		analysistesting.NewAnnotation("a.proto", 150, 1, 192, 2, "COMMENT_MESSAGE"),
		analysistesting.NewAnnotation("a.proto", 152, 3, 177, 4, "COMMENT_MESSAGE"),
		analysistesting.NewAnnotation("a.proto", 154, 5, 159, 6, "COMMENT_ENUM"),
		analysistesting.NewAnnotation("a.proto", 156, 7, 156, 27, "COMMENT_ENUM_VALUE"),
		analysistesting.NewAnnotation("a.proto", 158, 7, 158, 19, "COMMENT_ENUM_VALUE"),
		analysistesting.NewAnnotation("a.proto", 161, 5, 169, 6, "COMMENT_MESSAGE"),
		analysistesting.NewAnnotation("a.proto", 163, 7, 163, 21, "COMMENT_FIELD"),
		analysistesting.NewAnnotation("a.proto", 165, 7, 168, 8, "COMMENT_ONEOF"),
		analysistesting.NewAnnotation("a.proto", 167, 9, 167, 23, "COMMENT_FIELD"),
		analysistesting.NewAnnotation("a.proto", 171, 5, 171, 19, "COMMENT_FIELD"),
		analysistesting.NewAnnotation("a.proto", 173, 5, 176, 6, "COMMENT_ONEOF"),
		analysistesting.NewAnnotation("a.proto", 175, 7, 175, 21, "COMMENT_FIELD"),
		analysistesting.NewAnnotation("a.proto", 179, 3, 184, 4, "COMMENT_ENUM"),
		analysistesting.NewAnnotation("a.proto", 181, 5, 181, 25, "COMMENT_ENUM_VALUE"),
		analysistesting.NewAnnotation("a.proto", 183, 5, 183, 17, "COMMENT_ENUM_VALUE"),
		analysistesting.NewAnnotation("a.proto", 186, 3, 186, 17, "COMMENT_FIELD"),
		analysistesting.NewAnnotation("a.proto", 188, 3, 191, 4, "COMMENT_ONEOF"),
		analysistesting.NewAnnotation("a.proto", 190, 5, 190, 19, "COMMENT_FIELD"),
		analysistesting.NewAnnotation("a.proto", 195, 1, 200, 2, "COMMENT_SERVICE"),
		analysistesting.NewAnnotation("a.proto", 197, 3, 197, 74, "COMMENT_RPC"),
		analysistesting.NewAnnotation("a.proto", 199, 3, 199, 72, "COMMENT_RPC"),
		analysistesting.NewAnnotation("a.proto", 203, 1, 208, 2, "COMMENT_ENUM"),
		analysistesting.NewAnnotation("a.proto", 205, 3, 205, 29, "COMMENT_ENUM_VALUE"),
		analysistesting.NewAnnotation("a.proto", 207, 3, 207, 21, "COMMENT_ENUM_VALUE"),
		analysistesting.NewAnnotation("a.proto", 211, 1, 253, 2, "COMMENT_MESSAGE"),
		analysistesting.NewAnnotation("a.proto", 213, 3, 238, 4, "COMMENT_MESSAGE"),
		analysistesting.NewAnnotation("a.proto", 215, 5, 220, 6, "COMMENT_ENUM"),
		analysistesting.NewAnnotation("a.proto", 217, 7, 217, 27, "COMMENT_ENUM_VALUE"),
		analysistesting.NewAnnotation("a.proto", 219, 7, 219, 19, "COMMENT_ENUM_VALUE"),
		analysistesting.NewAnnotation("a.proto", 222, 5, 230, 6, "COMMENT_MESSAGE"),
		analysistesting.NewAnnotation("a.proto", 224, 7, 224, 21, "COMMENT_FIELD"),
		analysistesting.NewAnnotation("a.proto", 226, 7, 229, 8, "COMMENT_ONEOF"),
		analysistesting.NewAnnotation("a.proto", 228, 9, 228, 23, "COMMENT_FIELD"),
		analysistesting.NewAnnotation("a.proto", 232, 5, 232, 19, "COMMENT_FIELD"),
		analysistesting.NewAnnotation("a.proto", 234, 5, 237, 6, "COMMENT_ONEOF"),
		analysistesting.NewAnnotation("a.proto", 236, 7, 236, 21, "COMMENT_FIELD"),
		analysistesting.NewAnnotation("a.proto", 240, 3, 245, 4, "COMMENT_ENUM"),
		analysistesting.NewAnnotation("a.proto", 242, 5, 242, 25, "COMMENT_ENUM_VALUE"),
		analysistesting.NewAnnotation("a.proto", 244, 5, 244, 17, "COMMENT_ENUM_VALUE"),
		analysistesting.NewAnnotation("a.proto", 247, 3, 247, 17, "COMMENT_FIELD"),
		analysistesting.NewAnnotation("a.proto", 249, 3, 252, 4, "COMMENT_ONEOF"),
		analysistesting.NewAnnotation("a.proto", 251, 5, 251, 19, "COMMENT_FIELD"),
		analysistesting.NewAnnotation("a.proto", 256, 1, 261, 2, "COMMENT_SERVICE"),
		analysistesting.NewAnnotation("a.proto", 258, 3, 258, 74, "COMMENT_RPC"),
		analysistesting.NewAnnotation("a.proto", 260, 3, 260, 72, "COMMENT_RPC"),
		analysistesting.NewAnnotation("a.proto", 263, 1, 265, 2, "COMMENT_MESSAGE"),
		analysistesting.NewAnnotation("a.proto", 264, 3, 264, 30, "COMMENT_FIELD"),
	)
}

func TestRunDirectorySamePackage(t *testing.T) {
	testLint(
		t,
		"directory_same_package",
		analysistesting.NewAnnotation("a.proto", 3, 1, 3, 11, "DIRECTORY_SAME_PACKAGE"),
		analysistesting.NewAnnotationNoLocation("no_package.proto", "DIRECTORY_SAME_PACKAGE"),
		analysistesting.NewAnnotation("one/c.proto", 3, 1, 3, 11, "DIRECTORY_SAME_PACKAGE"),
		analysistesting.NewAnnotation("one/d.proto", 3, 1, 3, 11, "DIRECTORY_SAME_PACKAGE"),
	)
}

func TestRunImportNoPublic(t *testing.T) {
	testLint(
		t,
		"import_no_public",
		analysistesting.NewAnnotation("a.proto", 6, 1, 6, 32, "IMPORT_NO_PUBLIC"),
		analysistesting.NewAnnotation("a.proto", 7, 1, 7, 31, "IMPORT_NO_PUBLIC"),
		analysistesting.NewAnnotation("one/one.proto", 6, 1, 6, 32, "IMPORT_NO_PUBLIC"),
	)
}

func TestRunImportNoWeak(t *testing.T) {
	testLint(
		t,
		"import_no_weak",
		analysistesting.NewAnnotation("a.proto", 6, 1, 6, 30, "IMPORT_NO_WEAK"),
		analysistesting.NewAnnotation("a.proto", 7, 1, 7, 29, "IMPORT_NO_WEAK"),
		analysistesting.NewAnnotation("one/one.proto", 6, 1, 6, 30, "IMPORT_NO_WEAK"),
	)
}

func TestRunEnumNoAllowAlias(t *testing.T) {
	testLint(
		t,
		"enum_no_allow_alias",
		analysistesting.NewAnnotation("a.proto", 12, 3, 12, 29, "ENUM_NO_ALLOW_ALIAS"),
		analysistesting.NewAnnotation("a.proto", 19, 3, 19, 29, "ENUM_NO_ALLOW_ALIAS"),
		analysistesting.NewAnnotation("a.proto", 41, 7, 41, 33, "ENUM_NO_ALLOW_ALIAS"),
		analysistesting.NewAnnotation("a.proto", 48, 7, 48, 33, "ENUM_NO_ALLOW_ALIAS"),
		analysistesting.NewAnnotation("a.proto", 68, 5, 68, 31, "ENUM_NO_ALLOW_ALIAS"),
		analysistesting.NewAnnotation("a.proto", 75, 5, 75, 31, "ENUM_NO_ALLOW_ALIAS"),
	)
}

func TestRunEnumPascalCase(t *testing.T) {
	testLint(
		t,
		"enum_pascal_case",
		analysistesting.NewAnnotation("a.proto", 16, 6, 16, 10, "ENUM_PASCAL_CASE"),
		analysistesting.NewAnnotation("a.proto", 19, 6, 19, 13, "ENUM_PASCAL_CASE"),
		analysistesting.NewAnnotation("a.proto", 22, 6, 22, 16, "ENUM_PASCAL_CASE"),
		analysistesting.NewAnnotation("a.proto", 25, 6, 25, 15, "ENUM_PASCAL_CASE"),
		analysistesting.NewAnnotation("a.proto", 42, 10, 42, 14, "ENUM_PASCAL_CASE"),
		analysistesting.NewAnnotation("a.proto", 45, 10, 45, 17, "ENUM_PASCAL_CASE"),
		analysistesting.NewAnnotation("a.proto", 48, 10, 48, 20, "ENUM_PASCAL_CASE"),
		analysistesting.NewAnnotation("a.proto", 51, 10, 51, 19, "ENUM_PASCAL_CASE"),
		analysistesting.NewAnnotation("a.proto", 66, 8, 66, 12, "ENUM_PASCAL_CASE"),
		analysistesting.NewAnnotation("a.proto", 69, 8, 69, 15, "ENUM_PASCAL_CASE"),
		analysistesting.NewAnnotation("a.proto", 72, 8, 72, 18, "ENUM_PASCAL_CASE"),
		analysistesting.NewAnnotation("a.proto", 75, 8, 75, 17, "ENUM_PASCAL_CASE"),
	)
}

func TestRunEnumValuePrefix(t *testing.T) {
	testLint(
		t,
		"enum_value_prefix",
		analysistesting.NewAnnotation("a.proto", 10, 3, 10, 12, "ENUM_VALUE_PREFIX"),
		analysistesting.NewAnnotation("a.proto", 11, 3, 11, 14, "ENUM_VALUE_PREFIX"),
		analysistesting.NewAnnotation("a.proto", 12, 3, 12, 15, "ENUM_VALUE_PREFIX"),
		analysistesting.NewAnnotation("a.proto", 22, 7, 22, 17, "ENUM_VALUE_PREFIX"),
		analysistesting.NewAnnotation("a.proto", 23, 7, 23, 19, "ENUM_VALUE_PREFIX"),
		analysistesting.NewAnnotation("a.proto", 24, 7, 24, 20, "ENUM_VALUE_PREFIX"),
		analysistesting.NewAnnotation("a.proto", 25, 7, 25, 20, "ENUM_VALUE_PREFIX"),
		analysistesting.NewAnnotation("a.proto", 33, 5, 33, 15, "ENUM_VALUE_PREFIX"),
		analysistesting.NewAnnotation("a.proto", 34, 5, 34, 17, "ENUM_VALUE_PREFIX"),
		analysistesting.NewAnnotation("a.proto", 35, 5, 35, 18, "ENUM_VALUE_PREFIX"),
		analysistesting.NewAnnotation("a.proto", 36, 5, 36, 18, "ENUM_VALUE_PREFIX"),
	)
}

func TestRunEnumValueUpperSnakeCase(t *testing.T) {
	testLint(
		t,
		"enum_value_upper_snake_case",
		analysistesting.NewAnnotation("a.proto", 10, 3, 10, 12, "ENUM_VALUE_UPPER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 11, 3, 11, 17, "ENUM_VALUE_UPPER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 12, 3, 12, 14, "ENUM_VALUE_UPPER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 23, 7, 23, 16, "ENUM_VALUE_UPPER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 24, 7, 24, 21, "ENUM_VALUE_UPPER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 25, 7, 25, 18, "ENUM_VALUE_UPPER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 34, 5, 34, 14, "ENUM_VALUE_UPPER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 35, 5, 35, 19, "ENUM_VALUE_UPPER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 36, 5, 36, 16, "ENUM_VALUE_UPPER_SNAKE_CASE"),
	)
}

func TestRunEnumZeroValueSuffix(t *testing.T) {
	testLint(
		t,
		"enum_zero_value_suffix",
		analysistesting.NewAnnotation("a.proto", 14, 3, 14, 18, "ENUM_ZERO_VALUE_SUFFIX"),
		analysistesting.NewAnnotation("a.proto", 18, 3, 18, 23, "ENUM_ZERO_VALUE_SUFFIX"),
		analysistesting.NewAnnotation("a.proto", 22, 3, 22, 19, "ENUM_ZERO_VALUE_SUFFIX"),
		analysistesting.NewAnnotation("a.proto", 36, 7, 36, 22, "ENUM_ZERO_VALUE_SUFFIX"),
		analysistesting.NewAnnotation("a.proto", 40, 7, 40, 27, "ENUM_ZERO_VALUE_SUFFIX"),
		analysistesting.NewAnnotation("a.proto", 44, 7, 44, 23, "ENUM_ZERO_VALUE_SUFFIX"),
		analysistesting.NewAnnotation("a.proto", 56, 5, 56, 20, "ENUM_ZERO_VALUE_SUFFIX"),
		analysistesting.NewAnnotation("a.proto", 60, 5, 60, 25, "ENUM_ZERO_VALUE_SUFFIX"),
		analysistesting.NewAnnotation("a.proto", 64, 5, 64, 21, "ENUM_ZERO_VALUE_SUFFIX"),
	)
}

func TestRunEnumZeroValueSuffixCustom(t *testing.T) {
	testLint(
		t,
		"enum_zero_value_suffix_custom",
		analysistesting.NewAnnotation("a.proto", 18, 3, 18, 16, "ENUM_ZERO_VALUE_SUFFIX"),
		analysistesting.NewAnnotation("a.proto", 22, 3, 22, 23, "ENUM_ZERO_VALUE_SUFFIX"),
		analysistesting.NewAnnotation("a.proto", 40, 7, 40, 20, "ENUM_ZERO_VALUE_SUFFIX"),
		analysistesting.NewAnnotation("a.proto", 44, 7, 44, 27, "ENUM_ZERO_VALUE_SUFFIX"),
		analysistesting.NewAnnotation("a.proto", 60, 5, 60, 18, "ENUM_ZERO_VALUE_SUFFIX"),
		analysistesting.NewAnnotation("a.proto", 64, 5, 64, 25, "ENUM_ZERO_VALUE_SUFFIX"),
	)
}

func TestRunFieldLowerSnakeCase(t *testing.T) {
	testLint(
		t,
		"field_lower_snake_case",
		analysistesting.NewAnnotation("a.proto", 8, 9, 8, 13, "FIELD_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 9, 9, 9, 16, "FIELD_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 10, 9, 10, 18, "FIELD_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 11, 9, 11, 19, "FIELD_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 12, 9, 12, 19, "FIELD_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 20, 13, 20, 17, "FIELD_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 21, 13, 21, 20, "FIELD_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 22, 13, 22, 22, "FIELD_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 23, 13, 23, 23, "FIELD_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 24, 13, 24, 23, "FIELD_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 28, 11, 28, 15, "FIELD_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 29, 11, 29, 18, "FIELD_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 30, 11, 30, 20, "FIELD_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 31, 11, 31, 21, "FIELD_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 32, 11, 32, 21, "FIELD_LOWER_SNAKE_CASE"),
	)
}

func TestRunFieldNoDescriptor(t *testing.T) {
	testLint(
		t,
		"field_no_descriptor",
		analysistesting.NewAnnotation("a.proto", 6, 10, 6, 20, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 7, 10, 7, 20, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 8, 10, 8, 20, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 9, 10, 9, 21, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 10, 10, 10, 22, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 11, 10, 11, 21, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 12, 10, 12, 22, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 13, 10, 13, 24, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 19, 14, 19, 24, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 20, 14, 20, 24, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 21, 14, 21, 24, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 22, 14, 22, 25, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 23, 14, 23, 26, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 24, 14, 24, 25, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 25, 14, 25, 26, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 26, 14, 26, 28, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 28, 12, 28, 22, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 29, 12, 29, 22, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 30, 12, 30, 22, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 31, 12, 31, 23, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 32, 12, 32, 24, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 33, 12, 33, 23, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 34, 12, 34, 24, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 35, 12, 35, 26, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 37, 10, 37, 20, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 38, 10, 38, 20, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 39, 10, 39, 20, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 40, 10, 40, 21, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 41, 10, 41, 22, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 42, 10, 42, 21, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 43, 10, 43, 22, "FIELD_NO_DESCRIPTOR"),
		analysistesting.NewAnnotation("a.proto", 44, 10, 44, 24, "FIELD_NO_DESCRIPTOR"),
	)
}

func TestRunFileLowerSnakeCase(t *testing.T) {
	testLint(
		t,
		"file_lower_snake_case",
		analysistesting.NewAnnotationNoLocation("B.proto", "FILE_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotationNoLocation("Foo.proto", "FILE_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotationNoLocation("aBc.proto", "FILE_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotationNoLocation("ab_c_.proto", "FILE_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotationNoLocation("fooBar.proto", "FILE_LOWER_SNAKE_CASE"),
	)
}

func TestRunMessagePascalCase(t *testing.T) {
	testLint(
		t,
		"message_pascal_case",
		analysistesting.NewAnnotation("a.proto", 8, 11, 8, 15, "MESSAGE_PASCAL_CASE"),
		analysistesting.NewAnnotation("a.proto", 9, 11, 9, 18, "MESSAGE_PASCAL_CASE"),
		analysistesting.NewAnnotation("a.proto", 10, 13, 10, 23, "MESSAGE_PASCAL_CASE"),
		analysistesting.NewAnnotation("a.proto", 14, 9, 14, 13, "MESSAGE_PASCAL_CASE"),
		analysistesting.NewAnnotation("a.proto", 15, 9, 15, 16, "MESSAGE_PASCAL_CASE"),
		analysistesting.NewAnnotation("a.proto", 16, 9, 16, 19, "MESSAGE_PASCAL_CASE"),
		analysistesting.NewAnnotation("a.proto", 17, 9, 17, 18, "MESSAGE_PASCAL_CASE"),
		analysistesting.NewAnnotation("a.proto", 18, 11, 18, 15, "MESSAGE_PASCAL_CASE"),
		analysistesting.NewAnnotation("a.proto", 19, 11, 19, 18, "MESSAGE_PASCAL_CASE"),
		analysistesting.NewAnnotation("a.proto", 20, 13, 20, 23, "MESSAGE_PASCAL_CASE"),
	)
}

func TestRunOneofLowerSnakeCase(t *testing.T) {
	testLint(
		t,
		"oneof_lower_snake_case",
		analysistesting.NewAnnotation("a.proto", 12, 9, 12, 13, "ONEOF_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 15, 9, 15, 16, "ONEOF_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 18, 9, 18, 18, "ONEOF_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 21, 9, 21, 19, "ONEOF_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 24, 9, 24, 19, "ONEOF_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 38, 13, 38, 17, "ONEOF_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 41, 13, 41, 20, "ONEOF_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 44, 13, 44, 22, "ONEOF_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 47, 13, 47, 23, "ONEOF_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 50, 13, 50, 23, "ONEOF_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 60, 11, 60, 15, "ONEOF_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 63, 11, 63, 18, "ONEOF_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 66, 11, 66, 20, "ONEOF_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 69, 11, 69, 21, "ONEOF_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("a.proto", 72, 11, 72, 21, "ONEOF_LOWER_SNAKE_CASE"),
	)
}

func TestRunPackageDefined(t *testing.T) {
	testLint(
		t,
		"package_defined",
		analysistesting.NewAnnotationNoLocation("a/no_package.proto", "PACKAGE_DEFINED"),
		analysistesting.NewAnnotationNoLocation("no_package.proto", "PACKAGE_DEFINED"),
	)
}

func TestRunPackageDirectoryMatch(t *testing.T) {
	testLint(
		t,
		"package_directory_match",
		analysistesting.NewAnnotation("a/b/a_c.proto", 3, 1, 3, 13, "PACKAGE_DIRECTORY_MATCH"),
		analysistesting.NewAnnotation("sub/a/b/a_b.proto", 3, 1, 3, 13, "PACKAGE_DIRECTORY_MATCH"),
	)
}

func TestRunPackageLowerSnakeCase(t *testing.T) {
	testLint(
		t,
		"package_lower_snake_case",
		analysistesting.NewAnnotation("5.proto", 3, 1, 3, 19, "PACKAGE_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("6.proto", 3, 1, 3, 19, "PACKAGE_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("7.proto", 3, 1, 3, 18, "PACKAGE_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("8.proto", 3, 1, 3, 20, "PACKAGE_LOWER_SNAKE_CASE"),
	)
}

func TestRunPackageSameDirectory(t *testing.T) {
	testLint(
		t,
		"package_same_directory",
		analysistesting.NewAnnotation("a.proto", 3, 1, 3, 11, "PACKAGE_SAME_DIRECTORY"),
		analysistesting.NewAnnotation("one/a.proto", 3, 1, 3, 11, "PACKAGE_SAME_DIRECTORY"),
	)
}

func TestRunPackageSameDirectoryNoPackage(t *testing.T) {
	testLint(
		t,
		"package_same_directory_no_package",
		analysistesting.NewAnnotationNoLocation("no_package.proto", "PACKAGE_SAME_DIRECTORY"),
		analysistesting.NewAnnotationNoLocation("one/no_package.proto", "PACKAGE_SAME_DIRECTORY"),
	)
}

func TestRunPackageSameOptionValue(t *testing.T) {
	testLint(
		t,
		"package_same_option_value",
		analysistesting.NewAnnotation("a.proto", 5, 1, 5, 33, "PACKAGE_SAME_CSHARP_NAMESPACE"),
		analysistesting.NewAnnotation("a.proto", 6, 1, 6, 35, "PACKAGE_SAME_JAVA_MULTIPLE_FILES"),
		analysistesting.NewAnnotation("a.proto", 7, 1, 7, 29, "PACKAGE_SAME_JAVA_PACKAGE"),
		analysistesting.NewAnnotation("a.proto", 8, 1, 8, 27, "PACKAGE_SAME_GO_PACKAGE"),
		analysistesting.NewAnnotation("a.proto", 9, 1, 9, 30, "PACKAGE_SAME_PHP_NAMESPACE"),
		analysistesting.NewAnnotation("a.proto", 10, 1, 10, 29, "PACKAGE_SAME_RUBY_PACKAGE"),
		analysistesting.NewAnnotation("a.proto", 11, 1, 11, 29, "PACKAGE_SAME_SWIFT_PREFIX"),
		analysistesting.NewAnnotation("b.proto", 5, 1, 5, 33, "PACKAGE_SAME_CSHARP_NAMESPACE"),
		analysistesting.NewAnnotation("b.proto", 6, 1, 6, 36, "PACKAGE_SAME_JAVA_MULTIPLE_FILES"),
		analysistesting.NewAnnotation("b.proto", 7, 1, 7, 29, "PACKAGE_SAME_JAVA_PACKAGE"),
		analysistesting.NewAnnotation("b.proto", 8, 1, 8, 27, "PACKAGE_SAME_GO_PACKAGE"),
		analysistesting.NewAnnotation("b.proto", 9, 1, 9, 30, "PACKAGE_SAME_PHP_NAMESPACE"),
		analysistesting.NewAnnotation("b.proto", 10, 1, 10, 29, "PACKAGE_SAME_RUBY_PACKAGE"),
		analysistesting.NewAnnotation("b.proto", 11, 1, 11, 29, "PACKAGE_SAME_SWIFT_PREFIX"),
		analysistesting.NewAnnotation("sub/a.proto", 6, 1, 6, 33, "PACKAGE_SAME_CSHARP_NAMESPACE"),
		analysistesting.NewAnnotation("sub/a.proto", 7, 1, 7, 35, "PACKAGE_SAME_JAVA_MULTIPLE_FILES"),
		analysistesting.NewAnnotation("sub/a.proto", 8, 1, 8, 29, "PACKAGE_SAME_JAVA_PACKAGE"),
		analysistesting.NewAnnotation("sub/a.proto", 9, 1, 9, 27, "PACKAGE_SAME_GO_PACKAGE"),
		analysistesting.NewAnnotation("sub/a.proto", 10, 1, 10, 30, "PACKAGE_SAME_PHP_NAMESPACE"),
		analysistesting.NewAnnotation("sub/a.proto", 11, 1, 11, 29, "PACKAGE_SAME_RUBY_PACKAGE"),
		analysistesting.NewAnnotation("sub/a.proto", 12, 1, 12, 29, "PACKAGE_SAME_SWIFT_PREFIX"),
		analysistesting.NewAnnotationNoLocation("sub/b.proto", "PACKAGE_SAME_CSHARP_NAMESPACE"),
		analysistesting.NewAnnotationNoLocation("sub/b.proto", "PACKAGE_SAME_GO_PACKAGE"),
		analysistesting.NewAnnotationNoLocation("sub/b.proto", "PACKAGE_SAME_JAVA_MULTIPLE_FILES"),
		analysistesting.NewAnnotationNoLocation("sub/b.proto", "PACKAGE_SAME_JAVA_PACKAGE"),
		analysistesting.NewAnnotationNoLocation("sub/b.proto", "PACKAGE_SAME_PHP_NAMESPACE"),
		analysistesting.NewAnnotationNoLocation("sub/b.proto", "PACKAGE_SAME_RUBY_PACKAGE"),
		analysistesting.NewAnnotationNoLocation("sub/b.proto", "PACKAGE_SAME_SWIFT_PREFIX"),
	)
}

func TestRunPackageVersionSuffix(t *testing.T) {
	testLint(
		t,
		"package_version_suffix",
		analysistesting.NewAnnotation("foo.proto", 3, 1, 3, 13, "PACKAGE_VERSION_SUFFIX"),
		analysistesting.NewAnnotation("foo_bar.proto", 3, 1, 3, 17, "PACKAGE_VERSION_SUFFIX"),
		analysistesting.NewAnnotation("foo_bar_v0beta1.proto", 3, 1, 3, 25, "PACKAGE_VERSION_SUFFIX"),
		analysistesting.NewAnnotation("foo_bar_v1test_foo.proto", 3, 1, 3, 28, "PACKAGE_VERSION_SUFFIX"),
		analysistesting.NewAnnotation("foo_bar_v2beta0.proto", 3, 1, 3, 25, "PACKAGE_VERSION_SUFFIX"),
		analysistesting.NewAnnotation("foo_bar_vv1beta1.proto", 3, 1, 3, 26, "PACKAGE_VERSION_SUFFIX"),
		analysistesting.NewAnnotation("v1.proto", 3, 1, 3, 12, "PACKAGE_VERSION_SUFFIX"),
	)
}

func TestRunRPCNoStreaming(t *testing.T) {
	testLint(
		t,
		"rpc_no_streaming",
		analysistesting.NewAnnotation("a.proto", 9, 3, 9, 88, "RPC_NO_CLIENT_STREAMING"),
		analysistesting.NewAnnotation("a.proto", 10, 3, 10, 89, "RPC_NO_SERVER_STREAMING"),
		analysistesting.NewAnnotation("a.proto", 11, 3, 11, 92, "RPC_NO_CLIENT_STREAMING"),
		analysistesting.NewAnnotation("a.proto", 11, 3, 11, 92, "RPC_NO_SERVER_STREAMING"),
	)
}

func TestRunRPCPascalCase(t *testing.T) {
	testLint(
		t,
		"rpc_pascal_case",
		analysistesting.NewAnnotation("a.proto", 11, 7, 11, 11, "RPC_PASCAL_CASE"),
		analysistesting.NewAnnotation("a.proto", 12, 7, 12, 14, "RPC_PASCAL_CASE"),
		analysistesting.NewAnnotation("a.proto", 13, 7, 13, 17, "RPC_PASCAL_CASE"),
		analysistesting.NewAnnotation("a.proto", 14, 7, 14, 16, "RPC_PASCAL_CASE"),
	)
}

func TestRunRPCRequestResponseUnique(t *testing.T) {
	testLint(
		t,
		"rpc_request_response_unique",
		analysistesting.NewAnnotation("a.proto", 21, 3, 21, 32, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 22, 3, 22, 36, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 22, 3, 22, 36, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 23, 3, 23, 35, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 24, 3, 24, 34, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 24, 3, 24, 34, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 25, 3, 25, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 27, 3, 27, 52, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 28, 3, 28, 55, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 29, 3, 29, 69, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 29, 3, 29, 69, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 30, 3, 30, 38, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("b.proto", 8, 3, 8, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("b.proto", 8, 3, 8, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
	)
}

func TestRunRPCRequestResponseUniqueAllowEmptyRequests(t *testing.T) {
	testLint(
		t,
		"rpc_request_response_unique_allow_empty_requests",
		analysistesting.NewAnnotation("a.proto", 21, 3, 21, 32, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 22, 3, 22, 36, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 22, 3, 22, 36, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 23, 3, 23, 35, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 24, 3, 24, 34, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 24, 3, 24, 34, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 25, 3, 25, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 28, 3, 28, 55, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 29, 3, 29, 69, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 29, 3, 29, 69, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 30, 3, 30, 38, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("b.proto", 8, 3, 8, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("b.proto", 8, 3, 8, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
	)
}

func TestRunRPCRequestResponseUniqueAllowEmptyResponses(t *testing.T) {
	testLint(
		t,
		"rpc_request_response_unique_allow_empty_responses",
		analysistesting.NewAnnotation("a.proto", 21, 3, 21, 32, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 22, 3, 22, 36, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 22, 3, 22, 36, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 23, 3, 23, 35, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 24, 3, 24, 34, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 24, 3, 24, 34, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 25, 3, 25, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 27, 3, 27, 52, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 29, 3, 29, 69, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 29, 3, 29, 69, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 30, 3, 30, 38, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("b.proto", 8, 3, 8, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("b.proto", 8, 3, 8, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
	)
}

func TestRunRPCRequestResponseUniqueAllowEmptyRequestsAndResponses(t *testing.T) {
	testLint(
		t,
		"rpc_request_response_unique_allow_empty_requests_and_responses",
		analysistesting.NewAnnotation("a.proto", 21, 3, 21, 32, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 22, 3, 22, 36, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 22, 3, 22, 36, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 23, 3, 23, 35, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 24, 3, 24, 34, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 24, 3, 24, 34, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 25, 3, 25, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 30, 3, 30, 38, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("b.proto", 8, 3, 8, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("b.proto", 8, 3, 8, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
	)
}

func TestRunRPCRequestResponseUniqueAllowSame(t *testing.T) {
	testLint(
		t,
		"rpc_request_response_unique_allow_same",
		analysistesting.NewAnnotation("a.proto", 21, 3, 21, 32, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 22, 3, 22, 36, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 23, 3, 23, 35, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 24, 3, 24, 34, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 24, 3, 24, 34, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 25, 3, 25, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 27, 3, 27, 52, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 28, 3, 28, 55, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 29, 3, 29, 69, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("b.proto", 8, 3, 8, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("b.proto", 8, 3, 8, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
	)
}

func TestRunRPCRequestResponseUniqueAllowSameAndEmptyRequestResponses(t *testing.T) {
	testLint(
		t,
		"rpc_request_response_unique_allow_same_and_empty_request_responses",
		analysistesting.NewAnnotation("a.proto", 21, 3, 21, 32, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 22, 3, 22, 36, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 23, 3, 23, 35, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 24, 3, 24, 34, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 24, 3, 24, 34, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("a.proto", 25, 3, 25, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("b.proto", 8, 3, 8, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
		analysistesting.NewAnnotation("b.proto", 8, 3, 8, 37, "RPC_REQUEST_RESPONSE_UNIQUE"),
	)
}

func TestRunRPCStandardName(t *testing.T) {
	testLint(
		t,
		"rpc_standard_name",
		analysistesting.NewAnnotation("a.proto", 10, 19, 10, 22, "RPC_REQUEST_STANDARD_NAME"),
		analysistesting.NewAnnotation("a.proto", 11, 50, 11, 53, "RPC_RESPONSE_STANDARD_NAME"),
		analysistesting.NewAnnotation("a.proto", 12, 27, 12, 48, "RPC_REQUEST_STANDARD_NAME"),
		analysistesting.NewAnnotation("a.proto", 13, 66, 13, 87, "RPC_RESPONSE_STANDARD_NAME"),
	)
}

func TestRunRPCStandardNameAllowEmpty(t *testing.T) {
	testLint(
		t,
		"rpc_standard_name_allow_empty",
		analysistesting.NewAnnotation("a.proto", 10, 19, 10, 22, "RPC_REQUEST_STANDARD_NAME"),
		analysistesting.NewAnnotation("a.proto", 11, 50, 11, 53, "RPC_RESPONSE_STANDARD_NAME"),
	)
}

func TestRunServicePascalCase(t *testing.T) {
	testLint(
		t,
		"service_pascal_case",
		analysistesting.NewAnnotation("a.proto", 8, 9, 8, 13, "SERVICE_PASCAL_CASE"),
		analysistesting.NewAnnotation("a.proto", 9, 9, 9, 16, "SERVICE_PASCAL_CASE"),
		analysistesting.NewAnnotation("a.proto", 10, 9, 10, 19, "SERVICE_PASCAL_CASE"),
		analysistesting.NewAnnotation("a.proto", 11, 9, 11, 18, "SERVICE_PASCAL_CASE"),
	)
}

func TestRunServiceSuffix(t *testing.T) {
	testLint(
		t,
		"service_suffix",
		analysistesting.NewAnnotation("a.proto", 8, 9, 8, 13, "SERVICE_SUFFIX"),
		analysistesting.NewAnnotation("a.proto", 9, 9, 9, 16, "SERVICE_SUFFIX"),
		analysistesting.NewAnnotation("a.proto", 10, 9, 10, 21, "SERVICE_SUFFIX"),
	)
}

func TestRunServiceSuffixCustom(t *testing.T) {
	testLint(
		t,
		"service_suffix_custom",
		analysistesting.NewAnnotation("a.proto", 8, 9, 8, 13, "SERVICE_SUFFIX"),
		analysistesting.NewAnnotation("a.proto", 9, 9, 9, 20, "SERVICE_SUFFIX"),
		analysistesting.NewAnnotation("a.proto", 10, 9, 10, 17, "SERVICE_SUFFIX"),
		analysistesting.NewAnnotation("a.proto", 11, 9, 11, 17, "SERVICE_SUFFIX"),
	)
}

func TestRunIgnores1(t *testing.T) {
	testLint(
		t,
		"ignores",
		analysistesting.NewAnnotation("buf/bar/bar.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("buf/bar/bar.proto", 9, 9, 9, 12, "MESSAGE_PASCAL_CASE"),
		analysistesting.NewAnnotation("buf/bar/bar.proto", 13, 6, 13, 9, "ENUM_PASCAL_CASE"),
		analysistesting.NewAnnotation("buf/bar/bar2.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("buf/bar/bar2.proto", 9, 9, 9, 13, "MESSAGE_PASCAL_CASE"),
		analysistesting.NewAnnotation("buf/bar/bar2.proto", 13, 6, 13, 10, "ENUM_PASCAL_CASE"),
		analysistesting.NewAnnotation("buf/buf.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("buf/buf.proto", 9, 9, 9, 12, "MESSAGE_PASCAL_CASE"),
		analysistesting.NewAnnotation("buf/buf.proto", 13, 6, 13, 9, "ENUM_PASCAL_CASE"),
		analysistesting.NewAnnotation("buf/foo/bar/bar.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("buf/foo/bar/bar.proto", 9, 9, 9, 12, "MESSAGE_PASCAL_CASE"),
		analysistesting.NewAnnotation("buf/foo/bar/bar.proto", 13, 6, 13, 9, "ENUM_PASCAL_CASE"),
		analysistesting.NewAnnotation("buf/foo/baz/baz.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("buf/foo/baz/baz.proto", 9, 9, 9, 12, "MESSAGE_PASCAL_CASE"),
		analysistesting.NewAnnotation("buf/foo/baz/baz.proto", 13, 6, 13, 9, "ENUM_PASCAL_CASE"),
		analysistesting.NewAnnotation("buf/foo/buf.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("buf/foo/buf.proto", 9, 9, 9, 12, "MESSAGE_PASCAL_CASE"),
		analysistesting.NewAnnotation("buf/foo/buf.proto", 13, 6, 13, 9, "ENUM_PASCAL_CASE"),
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
		analysistesting.NewAnnotation("buf/bar/bar.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("buf/bar/bar.proto", 9, 9, 9, 12, "MESSAGE_PASCAL_CASE"),
		analysistesting.NewAnnotation("buf/bar/bar.proto", 13, 6, 13, 9, "ENUM_PASCAL_CASE"),
		analysistesting.NewAnnotation("buf/buf.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("buf/buf.proto", 9, 9, 9, 12, "MESSAGE_PASCAL_CASE"),
		analysistesting.NewAnnotation("buf/buf.proto", 13, 6, 13, 9, "ENUM_PASCAL_CASE"),
	)
}

func TestRunIgnores3(t *testing.T) {
	testLintExternalConfigModifier(
		t,
		"ignores",
		func(externalConfig *bufconfig.ExternalConfig) {
			externalConfig.Lint.IgnoreOnly = map[string][]string{
				"ENUM_PASCAL_CASE": []string{
					"buf/bar/bar.proto",
					"buf/foo/bar",
					"buf/foo/bar",
				},
				"MESSAGE_PASCAL_CASE": []string{
					"buf/bar/bar.proto",
				},
				"STYLE_BASIC": []string{
					"buf/foo/bar",
				},
			}
		},
		analysistesting.NewAnnotation("buf/bar/bar.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("buf/bar/bar2.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("buf/bar/bar2.proto", 9, 9, 9, 13, "MESSAGE_PASCAL_CASE"),
		analysistesting.NewAnnotation("buf/bar/bar2.proto", 13, 6, 13, 10, "ENUM_PASCAL_CASE"),
		analysistesting.NewAnnotation("buf/buf.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("buf/buf.proto", 9, 9, 9, 12, "MESSAGE_PASCAL_CASE"),
		analysistesting.NewAnnotation("buf/buf.proto", 13, 6, 13, 9, "ENUM_PASCAL_CASE"),
		analysistesting.NewAnnotation("buf/foo/baz/baz.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("buf/foo/baz/baz.proto", 9, 9, 9, 12, "MESSAGE_PASCAL_CASE"),
		analysistesting.NewAnnotation("buf/foo/baz/baz.proto", 13, 6, 13, 9, "ENUM_PASCAL_CASE"),
		analysistesting.NewAnnotation("buf/foo/buf.proto", 6, 9, 6, 15, "FIELD_LOWER_SNAKE_CASE"),
		analysistesting.NewAnnotation("buf/foo/buf.proto", 9, 9, 9, 12, "MESSAGE_PASCAL_CASE"),
		analysistesting.NewAnnotation("buf/foo/buf.proto", 13, 6, 13, 9, "ENUM_PASCAL_CASE"),
	)
}

func testLint(
	t *testing.T,
	dirPath string,
	expectedAnnotations ...*analysis.Annotation,
) {
	testLintExternalConfigModifier(
		t,
		dirPath,
		nil,
		expectedAnnotations...,
	)
}

func testLintExternalConfigModifier(
	t *testing.T,
	dirPath string,
	modifier func(*bufconfig.ExternalConfig),
	expectedAnnotations ...*analysis.Annotation,
) {
	t.Parallel()
	logger := zap.NewNop()

	bucket, err := storageos.NewReadBucket(filepath.Join("testdata", dirPath))
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
	config := testGetConfig(t, configProvider, bucket)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	buildHandler := bufbuild.NewHandler(
		logger,
		bufbuild.NewProvider(logger),
		bufbuild.NewRunner(logger),
	)
	image, resolver, annotations, err := buildHandler.BuildImage(
		ctx,
		bucket,
		config.Build.Roots,
		config.Build.Excludes,
		nil,
		false, // must exist
		true,  // just to make sure this works properly
		true,
	)
	require.NoError(t, err)
	require.Empty(t, annotations)

	handler := buflint.NewHandler(
		logger,
		buflint.NewRunner(logger),
	)
	annotations, err = handler.LintCheck(
		ctx,
		config.Lint,
		image,
	)
	assert.NoError(t, err)
	assert.NoError(t, bufbuild.FixAnnotationFilenames(resolver, annotations))
	analysistesting.AssertAnnotationsEqual(t, expectedAnnotations, annotations)
	assert.NoError(t, bucket.Close())
}

func testGetConfig(
	t *testing.T,
	configProvider bufconfig.Provider,
	bucket storage.ReadBucket,
) *bufconfig.Config {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	data, err := storageutil.ReadPath(ctx, bucket, bufconfig.ConfigFilePath)
	if err != nil && !storage.IsNotExist(err) {
		require.NoError(t, err)
	}
	config, err := configProvider.GetConfigForData(data)
	require.NoError(t, err)
	return config
}

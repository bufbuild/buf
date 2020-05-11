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

package bufbreaking_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/bufbuild/buf/internal/buf/bufbuild"
	"github.com/bufbuild/buf/internal/buf/bufcheck/bufbreaking"
	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/ext/extfile/extfiletesting"
	"github.com/bufbuild/buf/internal/buf/ext/extimage"
	filev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/file/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/bufbuild/buf/internal/pkg/storage/storageutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRunBreakingEnumNoDelete(t *testing.T) {
	testBreaking(
		t,
		"breaking_enum_no_delete",
		extfiletesting.NewFileAnnotationNoLocation("1.proto", "ENUM_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 9, 1, 18, 2, "ENUM_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 10, 3, 14, 4, "ENUM_NO_DELETE"),
	)
}

func TestRunBreakingEnumValueNoDelete(t *testing.T) {
	testBreaking(
		t,
		"breaking_enum_value_no_delete",
		extfiletesting.NewFileAnnotation("1.proto", 5, 1, 8, 2, "ENUM_VALUE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 12, 5, 15, 6, "ENUM_VALUE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 22, 3, 25, 4, "ENUM_VALUE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 40, 1, 43, 2, "ENUM_VALUE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("2.proto", 48, 1, 52, 2, "ENUM_VALUE_NO_DELETE"),
	)
}

func TestRunBreakingEnumValueNoDeleteUnlessNameReserved(t *testing.T) {
	testBreaking(
		t,
		"breaking_enum_value_no_delete_unless_name_reserved",
		extfiletesting.NewFileAnnotation("1.proto", 5, 1, 9, 2, "ENUM_VALUE_NO_DELETE_UNLESS_NAME_RESERVED"),
		extfiletesting.NewFileAnnotation("1.proto", 13, 5, 17, 6, "ENUM_VALUE_NO_DELETE_UNLESS_NAME_RESERVED"),
		extfiletesting.NewFileAnnotation("1.proto", 24, 3, 28, 4, "ENUM_VALUE_NO_DELETE_UNLESS_NAME_RESERVED"),
		extfiletesting.NewFileAnnotation("1.proto", 43, 1, 46, 2, "ENUM_VALUE_NO_DELETE_UNLESS_NAME_RESERVED"),
		extfiletesting.NewFileAnnotation("2.proto", 48, 1, 52, 2, "ENUM_VALUE_NO_DELETE_UNLESS_NAME_RESERVED"),
	)
}

func TestRunBreakingEnumValueNoDeleteUnlessNumberReserved(t *testing.T) {
	testBreaking(
		t,
		"breaking_enum_value_no_delete_unless_number_reserved",
		extfiletesting.NewFileAnnotation("1.proto", 5, 1, 9, 2, "ENUM_VALUE_NO_DELETE_UNLESS_NUMBER_RESERVED"),
		extfiletesting.NewFileAnnotation("1.proto", 13, 5, 17, 6, "ENUM_VALUE_NO_DELETE_UNLESS_NUMBER_RESERVED"),
		extfiletesting.NewFileAnnotation("1.proto", 24, 3, 28, 4, "ENUM_VALUE_NO_DELETE_UNLESS_NUMBER_RESERVED"),
		extfiletesting.NewFileAnnotation("1.proto", 43, 1, 46, 2, "ENUM_VALUE_NO_DELETE_UNLESS_NUMBER_RESERVED"),
		extfiletesting.NewFileAnnotation("2.proto", 48, 1, 52, 2, "ENUM_VALUE_NO_DELETE_UNLESS_NUMBER_RESERVED"),
	)
}

func TestRunBreakingEnumValueSameName(t *testing.T) {
	testBreaking(
		t,
		"breaking_enum_value_same_name",
		extfiletesting.NewFileAnnotation("1.proto", 8, 15, 8, 16, "ENUM_VALUE_SAME_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 16, 20, 16, 21, "ENUM_VALUE_SAME_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 27, 19, 27, 20, "ENUM_VALUE_SAME_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 46, 16, 46, 17, "ENUM_VALUE_SAME_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 47, 18, 47, 19, "ENUM_VALUE_SAME_NAME"),
		extfiletesting.NewFileAnnotation("2.proto", 51, 16, 51, 17, "ENUM_VALUE_SAME_NAME"),
	)
}

func TestRunBreakingExtensionMessageNoDelete(t *testing.T) {
	testBreaking(
		t,
		"breaking_extension_message_no_delete",
		extfiletesting.NewFileAnnotation("1.proto", 5, 1, 11, 2, "EXTENSION_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 5, 1, 11, 2, "EXTENSION_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 5, 1, 11, 2, "EXTENSION_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 15, 5, 21, 6, "EXTENSION_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 15, 5, 21, 6, "EXTENSION_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 15, 5, 21, 6, "EXTENSION_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 30, 3, 36, 4, "EXTENSION_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 30, 3, 36, 4, "EXTENSION_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 30, 3, 36, 4, "EXTENSION_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("2.proto", 73, 1, 79, 2, "EXTENSION_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("2.proto", 73, 1, 79, 2, "EXTENSION_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("2.proto", 73, 1, 79, 2, "EXTENSION_MESSAGE_NO_DELETE"),
	)
}

func TestRunBreakingFieldNoDelete(t *testing.T) {
	testBreaking(
		t,
		"breaking_field_no_delete",
		extfiletesting.NewFileAnnotation("1.proto", 5, 1, 8, 2, "FIELD_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 10, 1, 33, 2, "FIELD_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 12, 5, 15, 6, "FIELD_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 22, 3, 25, 4, "FIELD_NO_DELETE"),
		extfiletesting.NewFileAnnotation("2.proto", 57, 1, 60, 2, "FIELD_NO_DELETE"),
	)
}

func TestRunBreakingFieldNoDeleteUnlessNameReserved(t *testing.T) {
	testBreaking(
		t,
		"breaking_field_no_delete_unless_name_reserved",
		extfiletesting.NewFileAnnotation("1.proto", 5, 1, 9, 2, "FIELD_NO_DELETE_UNLESS_NAME_RESERVED"),
		extfiletesting.NewFileAnnotation("1.proto", 11, 1, 35, 2, "FIELD_NO_DELETE_UNLESS_NAME_RESERVED"),
		extfiletesting.NewFileAnnotation("1.proto", 11, 1, 35, 2, "FIELD_NO_DELETE_UNLESS_NAME_RESERVED"),
		extfiletesting.NewFileAnnotation("1.proto", 13, 5, 17, 6, "FIELD_NO_DELETE_UNLESS_NAME_RESERVED"),
		extfiletesting.NewFileAnnotation("1.proto", 24, 3, 28, 4, "FIELD_NO_DELETE_UNLESS_NAME_RESERVED"),
		extfiletesting.NewFileAnnotation("2.proto", 57, 1, 61, 2, "FIELD_NO_DELETE_UNLESS_NAME_RESERVED"),
	)
}

func TestRunBreakingFieldNoDeleteUnlessNumberReserved(t *testing.T) {
	testBreaking(
		t,
		"breaking_field_no_delete_unless_number_reserved",
		extfiletesting.NewFileAnnotation("1.proto", 5, 1, 9, 2, "FIELD_NO_DELETE_UNLESS_NUMBER_RESERVED"),
		extfiletesting.NewFileAnnotation("1.proto", 11, 1, 35, 2, "FIELD_NO_DELETE_UNLESS_NUMBER_RESERVED"),
		extfiletesting.NewFileAnnotation("1.proto", 11, 1, 35, 2, "FIELD_NO_DELETE_UNLESS_NUMBER_RESERVED"),
		extfiletesting.NewFileAnnotation("1.proto", 13, 5, 17, 6, "FIELD_NO_DELETE_UNLESS_NUMBER_RESERVED"),
		extfiletesting.NewFileAnnotation("1.proto", 24, 3, 28, 4, "FIELD_NO_DELETE_UNLESS_NUMBER_RESERVED"),
		extfiletesting.NewFileAnnotation("2.proto", 57, 1, 61, 2, "FIELD_NO_DELETE_UNLESS_NUMBER_RESERVED"),
	)
}

func TestRunBreakingFieldSameCType(t *testing.T) {
	testBreaking(
		t,
		"breaking_field_same_ctype",
		extfiletesting.NewFileAnnotation("1.proto", 6, 19, 6, 39, "FIELD_SAME_CTYPE"),
		extfiletesting.NewFileAnnotation("1.proto", 7, 3, 7, 18, "FIELD_SAME_CTYPE"),
		extfiletesting.NewFileAnnotation("1.proto", 13, 23, 13, 43, "FIELD_SAME_CTYPE"),
		extfiletesting.NewFileAnnotation("1.proto", 14, 7, 14, 22, "FIELD_SAME_CTYPE"),
		extfiletesting.NewFileAnnotation("1.proto", 23, 21, 23, 33, "FIELD_SAME_CTYPE"),
		extfiletesting.NewFileAnnotation("2.proto", 49, 28, 49, 48, "FIELD_SAME_CTYPE"),
		extfiletesting.NewFileAnnotation("2.proto", 50, 28, 50, 42, "FIELD_SAME_CTYPE"),
	)
}

func TestRunBreakingFieldSameJSONName(t *testing.T) {
	testBreaking(
		t,
		"breaking_field_same_json_name",
		extfiletesting.NewFileAnnotation("1.proto", 6, 3, 6, 17, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 7, 18, 7, 35, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 8, 20, 8, 37, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 9, 3, 9, 27, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 10, 28, 10, 45, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 11, 27, 11, 44, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 12, 3, 12, 31, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 13, 32, 13, 49, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 14, 31, 14, 48, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 20, 7, 20, 21, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 21, 22, 21, 39, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 22, 24, 22, 41, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 23, 7, 23, 31, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 24, 32, 24, 49, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 25, 31, 25, 48, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 26, 7, 26, 35, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 27, 36, 27, 53, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 28, 35, 28, 52, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 43, 5, 43, 19, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 44, 20, 44, 37, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 45, 22, 45, 39, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 46, 5, 46, 29, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 47, 30, 47, 47, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 48, 29, 48, 46, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 49, 5, 49, 33, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 50, 34, 50, 51, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 51, 33, 51, 50, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("2.proto", 92, 5, 92, 19, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("2.proto", 93, 20, 93, 37, "FIELD_SAME_JSON_NAME"),
		extfiletesting.NewFileAnnotation("2.proto", 94, 22, 94, 39, "FIELD_SAME_JSON_NAME"),
	)
}

func TestRunBreakingFieldSameJSType(t *testing.T) {
	testBreaking(
		t,
		"breaking_field_same_jstype",
		extfiletesting.NewFileAnnotation("1.proto", 6, 18, 6, 36, "FIELD_SAME_JSTYPE"),
		extfiletesting.NewFileAnnotation("1.proto", 7, 3, 7, 17, "FIELD_SAME_JSTYPE"),
		extfiletesting.NewFileAnnotation("1.proto", 13, 22, 13, 40, "FIELD_SAME_JSTYPE"),
		extfiletesting.NewFileAnnotation("1.proto", 14, 7, 14, 21, "FIELD_SAME_JSTYPE"),
		extfiletesting.NewFileAnnotation("1.proto", 22, 20, 22, 38, "FIELD_SAME_JSTYPE"),
		extfiletesting.NewFileAnnotation("2.proto", 49, 27, 49, 45, "FIELD_SAME_JSTYPE"),
		extfiletesting.NewFileAnnotation("2.proto", 50, 3, 50, 26, "FIELD_SAME_JSTYPE"),
	)
}

func TestRunBreakingFieldSameLabel(t *testing.T) {
	testBreaking(
		t,
		"breaking_field_same_label",
		extfiletesting.NewFileAnnotation("1.proto", 8, 3, 8, 26, "FIELD_SAME_LABEL"),
		extfiletesting.NewFileAnnotation("1.proto", 9, 3, 9, 24, "FIELD_SAME_LABEL"),
		extfiletesting.NewFileAnnotation("1.proto", 10, 3, 10, 19, "FIELD_SAME_LABEL"),
		extfiletesting.NewFileAnnotation("1.proto", 11, 3, 11, 16, "FIELD_SAME_LABEL"),
		extfiletesting.NewFileAnnotation("1.proto", 12, 3, 12, 18, "FIELD_SAME_LABEL"),
		extfiletesting.NewFileAnnotation("1.proto", 13, 3, 13, 17, "FIELD_SAME_LABEL"),
		extfiletesting.NewFileAnnotation("1.proto", 19, 7, 19, 30, "FIELD_SAME_LABEL"),
		extfiletesting.NewFileAnnotation("1.proto", 20, 7, 20, 28, "FIELD_SAME_LABEL"),
		extfiletesting.NewFileAnnotation("1.proto", 21, 7, 21, 23, "FIELD_SAME_LABEL"),
		extfiletesting.NewFileAnnotation("1.proto", 22, 7, 22, 20, "FIELD_SAME_LABEL"),
		extfiletesting.NewFileAnnotation("1.proto", 23, 7, 23, 22, "FIELD_SAME_LABEL"),
		extfiletesting.NewFileAnnotation("1.proto", 24, 7, 24, 21, "FIELD_SAME_LABEL"),
		extfiletesting.NewFileAnnotation("1.proto", 36, 5, 36, 28, "FIELD_SAME_LABEL"),
		extfiletesting.NewFileAnnotation("1.proto", 37, 5, 37, 26, "FIELD_SAME_LABEL"),
		extfiletesting.NewFileAnnotation("1.proto", 38, 5, 38, 21, "FIELD_SAME_LABEL"),
		extfiletesting.NewFileAnnotation("1.proto", 39, 5, 39, 18, "FIELD_SAME_LABEL"),
		extfiletesting.NewFileAnnotation("1.proto", 40, 5, 40, 20, "FIELD_SAME_LABEL"),
		extfiletesting.NewFileAnnotation("1.proto", 41, 5, 41, 19, "FIELD_SAME_LABEL"),
		extfiletesting.NewFileAnnotation("2.proto", 13, 3, 13, 26, "FIELD_SAME_LABEL"),
		extfiletesting.NewFileAnnotation("2.proto", 14, 3, 14, 24, "FIELD_SAME_LABEL"),
		extfiletesting.NewFileAnnotation("2.proto", 70, 3, 70, 26, "FIELD_SAME_LABEL"),
		extfiletesting.NewFileAnnotation("2.proto", 71, 3, 71, 26, "FIELD_SAME_LABEL"),
	)
}

func TestRunBreakingFieldSameName(t *testing.T) {
	testBreaking(
		t,
		"breaking_field_same_name",
		extfiletesting.NewFileAnnotation("1.proto", 7, 9, 7, 13, "FIELD_SAME_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 15, 13, 15, 17, "FIELD_SAME_NAME"),
		extfiletesting.NewFileAnnotation("1.proto", 26, 11, 26, 15, "FIELD_SAME_NAME"),
		extfiletesting.NewFileAnnotation("2.proto", 60, 11, 60, 15, "FIELD_SAME_NAME"),
	)
}

func TestRunBreakingFieldSameOneof(t *testing.T) {
	testBreaking(
		t,
		"breaking_field_same_oneof",
		extfiletesting.NewFileAnnotation("1.proto", 6, 3, 6, 17, "FIELD_SAME_ONEOF"),
		extfiletesting.NewFileAnnotation("1.proto", 8, 3, 8, 17, "FIELD_SAME_ONEOF"),
		extfiletesting.NewFileAnnotation("1.proto", 11, 3, 11, 19, "FIELD_SAME_ONEOF"),
		extfiletesting.NewFileAnnotation("1.proto", 18, 3, 18, 17, "FIELD_SAME_ONEOF"),
		extfiletesting.NewFileAnnotation("1.proto", 20, 3, 20, 17, "FIELD_SAME_ONEOF"),
		extfiletesting.NewFileAnnotation("1.proto", 23, 3, 23, 19, "FIELD_SAME_ONEOF"),
		extfiletesting.NewFileAnnotation("1.proto", 37, 3, 37, 17, "FIELD_SAME_ONEOF"),
		extfiletesting.NewFileAnnotation("1.proto", 39, 3, 39, 17, "FIELD_SAME_ONEOF"),
		extfiletesting.NewFileAnnotation("1.proto", 42, 3, 42, 19, "FIELD_SAME_ONEOF"),
		extfiletesting.NewFileAnnotation("2.proto", 94, 3, 94, 17, "FIELD_SAME_ONEOF"),
		extfiletesting.NewFileAnnotation("2.proto", 96, 3, 96, 17, "FIELD_SAME_ONEOF"),
		extfiletesting.NewFileAnnotation("2.proto", 99, 3, 99, 19, "FIELD_SAME_ONEOF"),
	)
}

func TestRunBreakingFieldSameType(t *testing.T) {
	// TODO: double check all this
	testBreaking(
		t,
		"breaking_field_same_type",
		extfiletesting.NewFileAnnotationNoLocation("1.proto", "FIELD_SAME_TYPE"),
		extfiletesting.NewFileAnnotationNoLocation("1.proto", "FIELD_SAME_TYPE"),
		extfiletesting.NewFileAnnotationNoLocation("1.proto", "FIELD_SAME_TYPE"),
		extfiletesting.NewFileAnnotation("1.proto", 8, 12, 8, 17, "FIELD_SAME_TYPE"),
		extfiletesting.NewFileAnnotation("1.proto", 9, 12, 9, 15, "FIELD_SAME_TYPE"),
		extfiletesting.NewFileAnnotation("1.proto", 11, 3, 11, 6, "FIELD_SAME_TYPE"),
		extfiletesting.NewFileAnnotation("1.proto", 12, 3, 12, 6, "FIELD_SAME_TYPE"),
		extfiletesting.NewFileAnnotation("1.proto", 19, 16, 19, 21, "FIELD_SAME_TYPE"),
		extfiletesting.NewFileAnnotation("1.proto", 20, 16, 20, 19, "FIELD_SAME_TYPE"),
		extfiletesting.NewFileAnnotation("1.proto", 22, 7, 22, 10, "FIELD_SAME_TYPE"),
		extfiletesting.NewFileAnnotation("1.proto", 23, 7, 23, 10, "FIELD_SAME_TYPE"),
		extfiletesting.NewFileAnnotation("1.proto", 36, 14, 36, 19, "FIELD_SAME_TYPE"),
		extfiletesting.NewFileAnnotation("1.proto", 37, 14, 37, 17, "FIELD_SAME_TYPE"),
		extfiletesting.NewFileAnnotation("1.proto", 39, 5, 39, 8, "FIELD_SAME_TYPE"),
		extfiletesting.NewFileAnnotation("1.proto", 40, 5, 40, 8, "FIELD_SAME_TYPE"),
		extfiletesting.NewFileAnnotation("2.proto", 64, 5, 64, 10, "FIELD_SAME_TYPE"),
		extfiletesting.NewFileAnnotation("2.proto", 65, 5, 65, 9, "FIELD_SAME_TYPE"),
	)
}

func TestRunBreakingFileNoDelete(t *testing.T) {
	testBreaking(
		t,
		"breaking_file_no_delete",
		extfiletesting.NewFileAnnotationNoLocation("", "FILE_NO_DELETE"),
		extfiletesting.NewFileAnnotationNoLocation("", "FILE_NO_DELETE"),
	)
}

func TestRunBreakingFileSamePackage(t *testing.T) {
	testBreaking(
		t,
		"breaking_file_same_package",
		extfiletesting.NewFileAnnotation("a/a.proto", 3, 1, 3, 11, "FILE_SAME_PACKAGE"),
		extfiletesting.NewFileAnnotation("no_package.proto", 3, 1, 3, 11, "FILE_SAME_PACKAGE"),
	)
}

func TestRunBreakingFileSameSyntax(t *testing.T) {
	testBreaking(
		t,
		"breaking_file_same_syntax",
		extfiletesting.NewFileAnnotation("root/no_package.proto", 1, 1, 1, 19, "FILE_SAME_SYNTAX"),
		extfiletesting.NewFileAnnotation("root/sub/a/b/sub_a_b.proto", 2, 1, 2, 19, "FILE_SAME_SYNTAX"),
	)
}

func TestRunBreakingFileSameValues(t *testing.T) {
	testBreaking(
		t,
		"breaking_file_same_values",
		extfiletesting.NewFileAnnotation("1.proto", 5, 1, 5, 29, "FILE_SAME_JAVA_PACKAGE"),
		extfiletesting.NewFileAnnotation("1.proto", 6, 1, 6, 37, "FILE_SAME_JAVA_OUTER_CLASSNAME"),
		extfiletesting.NewFileAnnotation("1.proto", 7, 1, 7, 36, "FILE_SAME_JAVA_MULTIPLE_FILES"),
		extfiletesting.NewFileAnnotation("1.proto", 8, 1, 8, 27, "FILE_SAME_GO_PACKAGE"),
		extfiletesting.NewFileAnnotation("1.proto", 9, 1, 9, 34, "FILE_SAME_OBJC_CLASS_PREFIX"),
		extfiletesting.NewFileAnnotation("1.proto", 10, 1, 10, 33, "FILE_SAME_CSHARP_NAMESPACE"),
		extfiletesting.NewFileAnnotation("1.proto", 11, 1, 11, 29, "FILE_SAME_SWIFT_PREFIX"),
		extfiletesting.NewFileAnnotation("1.proto", 12, 1, 12, 33, "FILE_SAME_PHP_CLASS_PREFIX"),
		extfiletesting.NewFileAnnotation("1.proto", 13, 1, 13, 30, "FILE_SAME_PHP_NAMESPACE"),
		extfiletesting.NewFileAnnotation("1.proto", 14, 1, 14, 39, "FILE_SAME_PHP_METADATA_NAMESPACE"),
		extfiletesting.NewFileAnnotation("1.proto", 15, 1, 15, 29, "FILE_SAME_RUBY_PACKAGE"),
		extfiletesting.NewFileAnnotation("1.proto", 17, 1, 17, 39, "FILE_SAME_JAVA_STRING_CHECK_UTF8"),
		extfiletesting.NewFileAnnotation("1.proto", 18, 1, 18, 29, "FILE_SAME_OPTIMIZE_FOR"),
		extfiletesting.NewFileAnnotation("1.proto", 19, 1, 19, 36, "FILE_SAME_CC_GENERIC_SERVICES"),
		extfiletesting.NewFileAnnotation("1.proto", 20, 1, 20, 38, "FILE_SAME_JAVA_GENERIC_SERVICES"),
		extfiletesting.NewFileAnnotation("1.proto", 21, 1, 21, 36, "FILE_SAME_PY_GENERIC_SERVICES"),
		extfiletesting.NewFileAnnotation("1.proto", 22, 1, 22, 37, "FILE_SAME_PHP_GENERIC_SERVICES"),
		extfiletesting.NewFileAnnotation("1.proto", 23, 1, 23, 33, "FILE_SAME_CC_ENABLE_ARENAS"),
		extfiletesting.NewFileAnnotation("2.proto", 3, 1, 3, 11, "FILE_SAME_PACKAGE"),
		extfiletesting.NewFileAnnotation("2.proto", 5, 1, 5, 29, "FILE_SAME_JAVA_PACKAGE"),
		extfiletesting.NewFileAnnotation("2.proto", 6, 1, 6, 37, "FILE_SAME_JAVA_OUTER_CLASSNAME"),
		extfiletesting.NewFileAnnotation("2.proto", 7, 1, 7, 35, "FILE_SAME_JAVA_MULTIPLE_FILES"),
		extfiletesting.NewFileAnnotation("2.proto", 8, 1, 8, 27, "FILE_SAME_GO_PACKAGE"),
		extfiletesting.NewFileAnnotation("2.proto", 9, 1, 9, 34, "FILE_SAME_OBJC_CLASS_PREFIX"),
		extfiletesting.NewFileAnnotation("2.proto", 10, 1, 10, 33, "FILE_SAME_CSHARP_NAMESPACE"),
		extfiletesting.NewFileAnnotation("2.proto", 11, 1, 11, 29, "FILE_SAME_SWIFT_PREFIX"),
		extfiletesting.NewFileAnnotation("2.proto", 12, 1, 12, 33, "FILE_SAME_PHP_CLASS_PREFIX"),
		extfiletesting.NewFileAnnotation("2.proto", 13, 1, 13, 30, "FILE_SAME_PHP_NAMESPACE"),
		extfiletesting.NewFileAnnotation("2.proto", 14, 1, 14, 39, "FILE_SAME_PHP_METADATA_NAMESPACE"),
		extfiletesting.NewFileAnnotation("2.proto", 15, 1, 15, 29, "FILE_SAME_RUBY_PACKAGE"),
		extfiletesting.NewFileAnnotation("2.proto", 17, 1, 17, 38, "FILE_SAME_JAVA_STRING_CHECK_UTF8"),
		extfiletesting.NewFileAnnotation("2.proto", 18, 1, 18, 33, "FILE_SAME_OPTIMIZE_FOR"),
		extfiletesting.NewFileAnnotation("2.proto", 19, 1, 19, 35, "FILE_SAME_CC_GENERIC_SERVICES"),
		extfiletesting.NewFileAnnotation("2.proto", 20, 1, 20, 37, "FILE_SAME_JAVA_GENERIC_SERVICES"),
		extfiletesting.NewFileAnnotation("2.proto", 21, 1, 21, 35, "FILE_SAME_PY_GENERIC_SERVICES"),
		extfiletesting.NewFileAnnotation("2.proto", 22, 1, 22, 36, "FILE_SAME_PHP_GENERIC_SERVICES"),
		extfiletesting.NewFileAnnotation("2.proto", 23, 1, 23, 33, "FILE_SAME_CC_ENABLE_ARENAS"),
	)
}

func TestRunBreakingMessageNoDelete(t *testing.T) {
	testBreaking(
		t,
		"breaking_message_no_delete",
		extfiletesting.NewFileAnnotationNoLocation("1.proto", "MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 7, 1, 12, 2, "MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 8, 3, 10, 4, "MESSAGE_NO_DELETE"),
	)
}

func TestRunBreakingMessageSameValues(t *testing.T) {
	testBreaking(
		t,
		"breaking_message_same_values",
		extfiletesting.NewFileAnnotationNoLocation("1.proto", "MESSAGE_SAME_MESSAGE_SET_WIRE_FORMAT"),
		extfiletesting.NewFileAnnotationNoLocation("1.proto", "MESSAGE_SAME_MESSAGE_SET_WIRE_FORMAT"),
		extfiletesting.NewFileAnnotation("1.proto", 6, 3, 6, 42, "MESSAGE_SAME_MESSAGE_SET_WIRE_FORMAT"),
		extfiletesting.NewFileAnnotation("1.proto", 7, 3, 7, 49, "MESSAGE_NO_REMOVE_STANDARD_DESCRIPTOR_ACCESSOR"),
		extfiletesting.NewFileAnnotation("1.proto", 13, 7, 13, 53, "MESSAGE_NO_REMOVE_STANDARD_DESCRIPTOR_ACCESSOR"),
		extfiletesting.NewFileAnnotation("1.proto", 16, 7, 16, 45, "MESSAGE_SAME_MESSAGE_SET_WIRE_FORMAT"),
		extfiletesting.NewFileAnnotation("1.proto", 21, 5, 21, 44, "MESSAGE_SAME_MESSAGE_SET_WIRE_FORMAT"),
		extfiletesting.NewFileAnnotation("1.proto", 24, 5, 24, 43, "MESSAGE_SAME_MESSAGE_SET_WIRE_FORMAT"),
		extfiletesting.NewFileAnnotation("1.proto", 27, 3, 27, 49, "MESSAGE_NO_REMOVE_STANDARD_DESCRIPTOR_ACCESSOR"),
		extfiletesting.NewFileAnnotationNoLocation("2.proto", "MESSAGE_SAME_MESSAGE_SET_WIRE_FORMAT"),
		extfiletesting.NewFileAnnotationNoLocation("2.proto", "MESSAGE_SAME_MESSAGE_SET_WIRE_FORMAT"),
		extfiletesting.NewFileAnnotation("2.proto", 6, 3, 6, 49, "MESSAGE_NO_REMOVE_STANDARD_DESCRIPTOR_ACCESSOR"),
		extfiletesting.NewFileAnnotation("2.proto", 10, 3, 10, 49, "MESSAGE_NO_REMOVE_STANDARD_DESCRIPTOR_ACCESSOR"),
	)
}

func TestRunBreakingOneofNoDelete(t *testing.T) {
	testBreaking(
		t,
		"breaking_oneof_no_delete",
		extfiletesting.NewFileAnnotation("1.proto", 5, 1, 9, 2, "ONEOF_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 13, 5, 17, 6, "ONEOF_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 26, 3, 30, 4, "ONEOF_NO_DELETE"),
		extfiletesting.NewFileAnnotation("2.proto", 75, 1, 79, 2, "ONEOF_NO_DELETE"),
	)
}

func TestRunBreakingPackageNoDelete(t *testing.T) {
	testBreaking(
		t,
		"breaking_package_no_delete",
		extfiletesting.NewFileAnnotationNoLocation("", "PACKAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotationNoLocation("a1.proto", "PACKAGE_ENUM_NO_DELETE"),
		extfiletesting.NewFileAnnotationNoLocation("a1.proto", "PACKAGE_ENUM_NO_DELETE"),
		extfiletesting.NewFileAnnotationNoLocation("a1.proto", "PACKAGE_ENUM_NO_DELETE"),
		extfiletesting.NewFileAnnotationNoLocation("a1.proto", "PACKAGE_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotationNoLocation("a1.proto", "PACKAGE_SERVICE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("a2.proto", 11, 1, 16, 2, "PACKAGE_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("a2.proto", 12, 3, 14, 4, "PACKAGE_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotationNoLocation("b1.proto", "PACKAGE_ENUM_NO_DELETE"),
		extfiletesting.NewFileAnnotationNoLocation("b1.proto", "PACKAGE_ENUM_NO_DELETE"),
		extfiletesting.NewFileAnnotationNoLocation("b1.proto", "PACKAGE_SERVICE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("b2.proto", 7, 1, 21, 2, "PACKAGE_MESSAGE_NO_DELETE"),
	)
}

func TestRunBreakingReservedEnumNoDelete(t *testing.T) {
	testBreaking(
		t,
		"breaking_reserved_enum_no_delete",
		extfiletesting.NewFileAnnotation("1.proto", 5, 1, 12, 2, "RESERVED_ENUM_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 5, 1, 12, 2, "RESERVED_ENUM_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 5, 1, 12, 2, "RESERVED_ENUM_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 5, 1, 12, 2, "RESERVED_ENUM_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 5, 1, 12, 2, "RESERVED_ENUM_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 16, 5, 23, 6, "RESERVED_ENUM_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 16, 5, 23, 6, "RESERVED_ENUM_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 16, 5, 23, 6, "RESERVED_ENUM_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 16, 5, 23, 6, "RESERVED_ENUM_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 16, 5, 23, 6, "RESERVED_ENUM_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 34, 3, 41, 4, "RESERVED_ENUM_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 34, 3, 41, 4, "RESERVED_ENUM_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 34, 3, 41, 4, "RESERVED_ENUM_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 34, 3, 41, 4, "RESERVED_ENUM_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 34, 3, 41, 4, "RESERVED_ENUM_NO_DELETE"),
		extfiletesting.NewFileAnnotation("2.proto", 86, 1, 93, 2, "RESERVED_ENUM_NO_DELETE"),
		extfiletesting.NewFileAnnotation("2.proto", 86, 1, 93, 2, "RESERVED_ENUM_NO_DELETE"),
		extfiletesting.NewFileAnnotation("2.proto", 86, 1, 93, 2, "RESERVED_ENUM_NO_DELETE"),
		extfiletesting.NewFileAnnotation("2.proto", 86, 1, 93, 2, "RESERVED_ENUM_NO_DELETE"),
		extfiletesting.NewFileAnnotation("2.proto", 86, 1, 93, 2, "RESERVED_ENUM_NO_DELETE"),
	)
}

func TestRunBreakingReservedMessageNoDelete(t *testing.T) {
	testBreaking(
		t,
		"breaking_reserved_message_no_delete",
		extfiletesting.NewFileAnnotation("1.proto", 5, 1, 12, 2, "RESERVED_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 5, 1, 12, 2, "RESERVED_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 5, 1, 12, 2, "RESERVED_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 5, 1, 12, 2, "RESERVED_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 5, 1, 12, 2, "RESERVED_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 16, 5, 23, 6, "RESERVED_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 16, 5, 23, 6, "RESERVED_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 16, 5, 23, 6, "RESERVED_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 16, 5, 23, 6, "RESERVED_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 16, 5, 23, 6, "RESERVED_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 34, 3, 41, 4, "RESERVED_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 34, 3, 41, 4, "RESERVED_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 34, 3, 41, 4, "RESERVED_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 34, 3, 41, 4, "RESERVED_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("1.proto", 34, 3, 41, 4, "RESERVED_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("2.proto", 89, 1, 96, 2, "RESERVED_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("2.proto", 89, 1, 96, 2, "RESERVED_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("2.proto", 89, 1, 96, 2, "RESERVED_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("2.proto", 89, 1, 96, 2, "RESERVED_MESSAGE_NO_DELETE"),
		extfiletesting.NewFileAnnotation("2.proto", 89, 1, 96, 2, "RESERVED_MESSAGE_NO_DELETE"),
	)
}

func TestRunBreakingRPCNoDelete(t *testing.T) {
	testBreaking(
		t,
		"breaking_rpc_no_delete",
		extfiletesting.NewFileAnnotation("1.proto", 7, 1, 10, 2, "RPC_NO_DELETE"),
		extfiletesting.NewFileAnnotation("2.proto", 31, 1, 34, 2, "RPC_NO_DELETE"),
	)
}

func TestRunBreakingRPCSameValues(t *testing.T) {
	testBreaking(
		t,
		"breaking_rpc_same_values",
		extfiletesting.NewFileAnnotation("1.proto", 9, 3, 9, 71, "RPC_SAME_CLIENT_STREAMING"),
		extfiletesting.NewFileAnnotation("1.proto", 9, 18, 9, 37, "RPC_SAME_REQUEST_TYPE"),
		extfiletesting.NewFileAnnotation("1.proto", 9, 48, 9, 67, "RPC_SAME_RESPONSE_TYPE"),
		extfiletesting.NewFileAnnotation("1.proto", 10, 3, 10, 71, "RPC_SAME_SERVER_STREAMING"),
		extfiletesting.NewFileAnnotation("1.proto", 10, 11, 10, 30, "RPC_SAME_REQUEST_TYPE"),
		extfiletesting.NewFileAnnotation("1.proto", 10, 48, 10, 67, "RPC_SAME_RESPONSE_TYPE"),
		extfiletesting.NewFileAnnotation("1.proto", 11, 3, 11, 68, "RPC_SAME_CLIENT_STREAMING"),
		extfiletesting.NewFileAnnotation("1.proto", 12, 3, 12, 68, "RPC_SAME_SERVER_STREAMING"),
		extfiletesting.NewFileAnnotation("2.proto", 37, 3, 37, 71, "RPC_SAME_CLIENT_STREAMING"),
		extfiletesting.NewFileAnnotation("2.proto", 37, 18, 37, 37, "RPC_SAME_REQUEST_TYPE"),
		extfiletesting.NewFileAnnotation("2.proto", 37, 48, 37, 67, "RPC_SAME_RESPONSE_TYPE"),
		extfiletesting.NewFileAnnotation("2.proto", 38, 3, 38, 71, "RPC_SAME_SERVER_STREAMING"),
		extfiletesting.NewFileAnnotation("2.proto", 38, 11, 38, 30, "RPC_SAME_REQUEST_TYPE"),
		extfiletesting.NewFileAnnotation("2.proto", 38, 48, 38, 67, "RPC_SAME_RESPONSE_TYPE"),
		extfiletesting.NewFileAnnotation("2.proto", 39, 3, 39, 68, "RPC_SAME_CLIENT_STREAMING"),
		extfiletesting.NewFileAnnotation("2.proto", 40, 3, 40, 68, "RPC_SAME_SERVER_STREAMING"),
		extfiletesting.NewFileAnnotation("2.proto", 45, 5, 45, 48, "RPC_SAME_IDEMPOTENCY_LEVEL"),
		extfiletesting.NewFileAnnotation("2.proto", 49, 5, 49, 43, "RPC_SAME_IDEMPOTENCY_LEVEL"),
		extfiletesting.NewFileAnnotation("2.proto", 55, 5, 55, 48, "RPC_SAME_IDEMPOTENCY_LEVEL"),
		extfiletesting.NewFileAnnotation("2.proto", 59, 5, 59, 43, "RPC_SAME_IDEMPOTENCY_LEVEL"),
	)
}

func TestRunBreakingServiceNoDelete(t *testing.T) {
	testBreaking(
		t,
		"breaking_service_no_delete",
		extfiletesting.NewFileAnnotationNoLocation("1.proto", "SERVICE_NO_DELETE"),
		extfiletesting.NewFileAnnotationNoLocation("1.proto", "SERVICE_NO_DELETE"),
	)
}

func testBreaking(
	t *testing.T,
	dirPath string,
	expectedFileAnnotations ...*filev1beta1.FileAnnotation,
) {
	testBreakingExternalConfigModifier(
		t,
		dirPath,
		nil,
		expectedFileAnnotations...,
	)
}

func testBreakingExternalConfigModifier(
	t *testing.T,
	dirPath string,
	modifier func(*bufconfig.ExternalConfig),
	expectedFileAnnotations ...*filev1beta1.FileAnnotation,
) {
	t.Parallel()
	logger := zap.NewNop()

	previousReadWriteBucketCloser, err := storageos.NewReadWriteBucketCloser(filepath.Join("testdata_previous", dirPath))
	require.NoError(t, err)
	readWriteBucketCloser, err := storageos.NewReadWriteBucketCloser(filepath.Join("testdata", dirPath))
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
	previousConfig := testGetConfig(t, configProvider, previousReadWriteBucketCloser)
	config := testGetConfig(t, configProvider, readWriteBucketCloser)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	buildHandler := bufbuild.NewHandler(logger)
	previousProtoFileSet, err := buildHandler.GetProtoFileSet(
		ctx,
		previousReadWriteBucketCloser,
		bufbuild.GetProtoFileSetOptions{
			Roots:    previousConfig.Build.Roots,
			Excludes: previousConfig.Build.Excludes,
		},
	)
	require.NoError(t, err)
	previousBuildResult, previousFileAnnotations, err := buildHandler.Build(
		ctx,
		previousReadWriteBucketCloser,
		previousProtoFileSet,
		bufbuild.BuildOptions{
			IncludeImports:    true, // just to make sure this works properly
			IncludeSourceInfo: false,
		},
	)
	require.NoError(t, err)
	require.Empty(t, previousFileAnnotations)
	previousImage, err := extimage.ImageWithoutImports(previousBuildResult.Image)
	require.NoError(t, err)
	protoFileSet, err := buildHandler.GetProtoFileSet(
		ctx,
		readWriteBucketCloser,
		bufbuild.GetProtoFileSetOptions{
			Roots:    config.Build.Roots,
			Excludes: config.Build.Excludes,
		},
	)
	require.NoError(t, err)
	buildResult, fileAnnotations, err := buildHandler.Build(
		ctx,
		readWriteBucketCloser,
		protoFileSet,
		bufbuild.BuildOptions{
			IncludeImports:    false, // just to make sure this works properly
			IncludeSourceInfo: true,
		},
	)
	require.NoError(t, err)
	require.Empty(t, fileAnnotations)
	image, err := extimage.ImageWithoutImports(buildResult.Image)
	require.NoError(t, err)

	handler := bufbreaking.NewHandler(
		logger,
		bufbreaking.NewRunner(logger),
	)
	fileAnnotations, err = handler.BreakingCheck(
		ctx,
		config.Breaking,
		previousImage,
		image,
	)
	assert.NoError(t, err)
	assert.NoError(t, bufbuild.FixFileAnnotationPaths(protoFileSet, fileAnnotations...))
	extfiletesting.AssertFileAnnotationsEqual(t, expectedFileAnnotations, fileAnnotations)
	assert.NoError(t, previousReadWriteBucketCloser.Close())
	assert.NoError(t, readWriteBucketCloser.Close())
}
func testGetConfig(
	t *testing.T,
	configProvider bufconfig.Provider,
	readBucket storage.ReadBucket,
) *bufconfig.Config {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	data, err := storageutil.ReadPath(ctx, readBucket, bufconfig.ConfigFilePath)
	if err != nil && !storage.IsNotExist(err) {
		require.NoError(t, err)
	}
	config, err := configProvider.GetConfigForData(data)
	require.NoError(t, err)
	return config
}

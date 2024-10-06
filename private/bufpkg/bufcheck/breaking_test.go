// Copyright 2020-2024 Buf Technologies, Inc.
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

package bufcheck_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bufbuild/buf/private/buf/buftarget"
	"github.com/bufbuild/buf/private/buf/bufworkspace"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis/bufanalysistesting"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/slogtestext"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/wasm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunBreakingEnumNoDelete(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_enum_no_delete",
		bufanalysistesting.NewFileAnnotationNoLocation(t, "1.proto", "ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 9, 1, 18, 2, "ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 10, 3, 14, 4, "ENUM_NO_DELETE"),
	)
}

func TestRunBreakingEnumSameJSONFormat(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_enum_same_json_format",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 9, 1, 11, 2, "ENUM_SAME_JSON_FORMAT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 13, 1, 15, 2, "ENUM_SAME_JSON_FORMAT"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 26, 3, 26, 52, "ENUM_SAME_JSON_FORMAT"),
		bufanalysistesting.NewFileAnnotation(t, "4.proto", 11, 1, 13, 2, "ENUM_SAME_JSON_FORMAT"),
	)
}

func TestRunBreakingEnumSameType(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_enum_same_type",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 9, 1, 11, 2, "ENUM_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 13, 1, 15, 2, "ENUM_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 5, 1, 7, 2, "ENUM_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 17, 1, 19, 2, "ENUM_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 5, 1, 7, 2, "ENUM_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 17, 1, 19, 2, "ENUM_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 26, 3, 26, 38, "ENUM_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "4.proto", 11, 1, 13, 2, "ENUM_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "4.proto", 29, 3, 29, 36, "ENUM_SAME_TYPE"),
	)
}

func TestRunBreakingEnumValueNoDelete(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_enum_value_no_delete",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 5, 1, 8, 2, "ENUM_VALUE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 12, 5, 15, 6, "ENUM_VALUE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 22, 3, 25, 4, "ENUM_VALUE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 40, 1, 42, 2, "ENUM_VALUE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 48, 1, 52, 2, "ENUM_VALUE_NO_DELETE"),
	)
}

func TestRunBreakingEnumValueNoDeleteUnlessNameReserved(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_enum_value_no_delete_unless_name_reserved",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 5, 1, 9, 2, "ENUM_VALUE_NO_DELETE_UNLESS_NAME_RESERVED"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 13, 5, 17, 6, "ENUM_VALUE_NO_DELETE_UNLESS_NAME_RESERVED"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 24, 3, 28, 4, "ENUM_VALUE_NO_DELETE_UNLESS_NAME_RESERVED"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 43, 1, 45, 2, "ENUM_VALUE_NO_DELETE_UNLESS_NAME_RESERVED"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 48, 1, 52, 2, "ENUM_VALUE_NO_DELETE_UNLESS_NAME_RESERVED"),
	)
}

func TestRunBreakingEnumValueNoDeleteUnlessNumberReserved(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_enum_value_no_delete_unless_number_reserved",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 5, 1, 9, 2, "ENUM_VALUE_NO_DELETE_UNLESS_NUMBER_RESERVED"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 13, 5, 17, 6, "ENUM_VALUE_NO_DELETE_UNLESS_NUMBER_RESERVED"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 24, 3, 28, 4, "ENUM_VALUE_NO_DELETE_UNLESS_NUMBER_RESERVED"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 43, 1, 45, 2, "ENUM_VALUE_NO_DELETE_UNLESS_NUMBER_RESERVED"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 48, 1, 52, 2, "ENUM_VALUE_NO_DELETE_UNLESS_NUMBER_RESERVED"),
	)
}

func TestRunBreakingEnumValueSameName(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_enum_value_same_name",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 8, 15, 8, 16, "ENUM_VALUE_SAME_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 16, 20, 16, 21, "ENUM_VALUE_SAME_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 27, 19, 27, 20, "ENUM_VALUE_SAME_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 46, 16, 46, 17, "ENUM_VALUE_SAME_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 47, 18, 47, 19, "ENUM_VALUE_SAME_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 51, 16, 51, 17, "ENUM_VALUE_SAME_NAME"),
	)
}

func TestRunBreakingExtensionMessageNoDelete(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_extension_message_no_delete",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 5, 1, 11, 2, "EXTENSION_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 5, 1, 11, 2, "EXTENSION_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 5, 1, 11, 2, "EXTENSION_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 15, 5, 21, 6, "EXTENSION_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 15, 5, 21, 6, "EXTENSION_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 15, 5, 21, 6, "EXTENSION_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 30, 3, 36, 4, "EXTENSION_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 30, 3, 36, 4, "EXTENSION_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 30, 3, 36, 4, "EXTENSION_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 49, 1, 53, 2, "EXTENSION_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 75, 1, 81, 2, "EXTENSION_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 75, 1, 81, 2, "EXTENSION_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 75, 1, 81, 2, "EXTENSION_MESSAGE_NO_DELETE"),
	)
}

func TestRunBreakingExtensionNoDelete(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_extension_no_delete",
		bufanalysistesting.NewFileAnnotationNoLocation(t, "2.proto", "EXTENSION_NO_DELETE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "2.proto", "EXTENSION_NO_DELETE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "2.proto", "EXTENSION_NO_DELETE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "3.proto", "EXTENSION_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 8, 3, 14, 4, "EXTENSION_NO_DELETE"),
	)
}

func TestRunBreakingFieldNoDelete(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_field_no_delete",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 5, 1, 8, 2, "FIELD_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 10, 1, 33, 2, "FIELD_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 12, 5, 15, 6, "FIELD_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 22, 3, 25, 4, "FIELD_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 57, 1, 60, 2, "FIELD_NO_DELETE"),
	)
}

func TestRunBreakingFieldNoDeleteUnlessNameReserved(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_field_no_delete_unless_name_reserved",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 5, 1, 9, 2, "FIELD_NO_DELETE_UNLESS_NAME_RESERVED"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 11, 1, 35, 2, "FIELD_NO_DELETE_UNLESS_NAME_RESERVED"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 11, 1, 35, 2, "FIELD_NO_DELETE_UNLESS_NAME_RESERVED"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 13, 5, 17, 6, "FIELD_NO_DELETE_UNLESS_NAME_RESERVED"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 24, 3, 28, 4, "FIELD_NO_DELETE_UNLESS_NAME_RESERVED"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 57, 1, 61, 2, "FIELD_NO_DELETE_UNLESS_NAME_RESERVED"),
	)
}

func TestRunBreakingFieldNoDeleteUnlessNumberReserved(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_field_no_delete_unless_number_reserved",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 5, 1, 9, 2, "FIELD_NO_DELETE_UNLESS_NUMBER_RESERVED"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 11, 1, 35, 2, "FIELD_NO_DELETE_UNLESS_NUMBER_RESERVED"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 11, 1, 35, 2, "FIELD_NO_DELETE_UNLESS_NUMBER_RESERVED"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 13, 5, 17, 6, "FIELD_NO_DELETE_UNLESS_NUMBER_RESERVED"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 24, 3, 28, 4, "FIELD_NO_DELETE_UNLESS_NUMBER_RESERVED"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 57, 1, 61, 2, "FIELD_NO_DELETE_UNLESS_NUMBER_RESERVED"),
	)
}

func TestRunBreakingFieldSameCardinality(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_field_same_cardinality",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 8, 3, 8, 26, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 9, 3, 9, 24, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 10, 3, 10, 19, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 11, 3, 11, 16, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 12, 3, 12, 18, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 13, 3, 13, 17, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 19, 7, 19, 30, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 20, 7, 20, 28, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 21, 7, 21, 23, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 22, 7, 22, 20, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 23, 7, 23, 22, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 24, 7, 24, 21, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 32, 7, 32, 33, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 40, 5, 40, 28, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 41, 5, 41, 26, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 42, 5, 42, 21, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 43, 5, 43, 18, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 44, 5, 44, 20, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 45, 5, 45, 19, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 48, 5, 48, 19, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 6, 3, 6, 26, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 13, 3, 13, 26, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 14, 3, 14, 24, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 70, 3, 70, 26, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 71, 3, 71, 26, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 17, 7, 17, 32, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 25, 3, 25, 20, "FIELD_SAME_CARDINALITY"),
	)
}

func TestRunBreakingFieldSameCppStringType(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_field_same_cpp_string_type",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 8, 23, 8, 41, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 9, 22, 9, 32, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 12, 23, 12, 57, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 13, 22, 13, 56, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 15, 22, 15, 34, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 16, 21, 16, 39, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 17, 3, 17, 30, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 18, 30, 18, 66, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 19, 22, 19, 56, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 23, 29, 23, 39, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 26, 30, 26, 64, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 27, 29, 27, 63, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 30, 25, 30, 35, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 31, 34, 31, 52, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 33, 25, 33, 59, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 34, 24, 34, 58, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 8, 23, 8, 57, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 9, 22, 9, 56, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 12, 23, 12, 33, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 13, 22, 13, 40, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 15, 22, 15, 58, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 16, 21, 16, 55, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 17, 3, 17, 30, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 18, 30, 18, 42, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 20, 21, 20, 39, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 22, 22, 22, 58, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 23, 21, 23, 55, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 24, 3, 24, 30, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 25, 30, 25, 42, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 26, 22, 26, 32, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 27, 21, 27, 39, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 30, 24, 30, 34, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 31, 34, 31, 52, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 33, 25, 33, 59, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 34, 24, 34, 58, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 43, 7, 43, 37, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 44, 29, 44, 63, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 11, 24, 11, 36, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 12, 23, 12, 33, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 13, 33, 13, 51, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 14, 32, 14, 68, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 16, 23, 16, 57, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 17, 3, 17, 32, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 27, 39, 27, 73, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 28, 7, 28, 29, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 30, 29, 30, 41, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 36, 3, 36, 33, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 40, 3, 40, 34, "FIELD_SAME_CPP_STRING_TYPE"),
	)
}

func TestRunBreakingFieldSameCType(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_field_same_ctype",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 6, 19, 6, 39, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 7, 3, 7, 18, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 13, 23, 13, 43, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 14, 7, 14, 22, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 23, 21, 23, 33, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 49, 28, 49, 48, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 50, 28, 50, 42, "FIELD_SAME_CPP_STRING_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 58, 32, 58, 44, "FIELD_SAME_CPP_STRING_TYPE"),
	)
}

func TestRunBreakingFieldSameJavaUTF8Validation(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_field_same_java_utf8_validation",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 19, 3, 19, 26, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 20, 3, 20, 26, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 21, 3, 21, 22, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 21, 3, 21, 22, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 30, 3, 30, 26, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 31, 3, 31, 26, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 32, 3, 32, 22, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 32, 3, 32, 22, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 42, 3, 42, 26, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 43, 3, 43, 26, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 44, 3, 44, 22, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 44, 3, 44, 22, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 54, 3, 54, 26, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 55, 3, 55, 26, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 56, 3, 56, 22, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 56, 3, 56, 22, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 66, 3, 66, 26, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 67, 3, 67, 26, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 68, 3, 68, 22, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 68, 3, 68, 22, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 5, 1, 5, 38, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 5, 1, 5, 38, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 5, 1, 5, 38, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 5, 1, 5, 38, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 6, 3, 6, 17, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 7, 3, 7, 26, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 8, 3, 8, 22, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 8, 3, 8, 22, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "4.proto", 8, 3, 8, 17, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "4.proto", 9, 3, 9, 26, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "4.proto", 10, 3, 10, 22, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "4.proto", 10, 3, 10, 22, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "4.proto", 85, 18, 85, 60, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "4.proto", 87, 3, 87, 22, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "4.proto", 87, 3, 87, 22, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "5.proto", 10, 3, 10, 17, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "5.proto", 11, 3, 11, 26, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "5.proto", 12, 3, 12, 22, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "5.proto", 12, 3, 12, 22, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "5.proto", 87, 18, 87, 59, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "5.proto", 89, 3, 89, 22, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "5.proto", 89, 3, 89, 22, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "6.proto", 11, 3, 11, 17, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "6.proto", 12, 3, 12, 26, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "6.proto", 13, 3, 13, 22, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "6.proto", 13, 3, 13, 22, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "6.proto", 23, 5, 23, 24, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "6.proto", 24, 5, 24, 33, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "6.proto", 106, 5, 106, 24, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "6.proto", 114, 3, 114, 22, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "6.proto", 114, 3, 114, 22, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "6.proto", 125, 3, 125, 22, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "6.proto", 125, 3, 125, 22, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "6.proto", 134, 3, 134, 22, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "6.proto", 135, 3, 135, 31, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "6.proto", 154, 3, 154, 22, "FIELD_SAME_JAVA_UTF8_VALIDATION"),
	)
}

func TestRunBreakingFieldSameDefault(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_field_same_default",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 7, 3, 7, 26, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 9, 27, 9, 42, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 12, 3, 12, 25, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 14, 26, 14, 41, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 18, 5, 18, 27, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 20, 28, 20, 41, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 25, 5, 25, 28, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 27, 29, 27, 42, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 32, 5, 32, 28, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 34, 29, 34, 42, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 39, 5, 39, 29, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 41, 30, 41, 43, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 46, 5, 46, 30, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 48, 31, 48, 44, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 53, 5, 53, 27, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 55, 28, 55, 41, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 60, 5, 60, 28, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 62, 29, 62, 42, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 67, 5, 67, 28, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 69, 29, 69, 42, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 74, 5, 74, 29, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 76, 30, 76, 43, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 81, 5, 81, 30, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 83, 31, 83, 44, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 88, 5, 88, 27, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 90, 28, 90, 41, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 95, 5, 95, 28, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 97, 29, 97, 42, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 102, 5, 102, 26, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 104, 27, 104, 42, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 109, 5, 109, 26, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 111, 27, 111, 40, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 140, 30, 140, 45, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 141, 5, 141, 27, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 148, 29, 148, 42, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 150, 28, 150, 43, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 153, 29, 153, 42, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 6, 18, 6, 33, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 8, 17, 8, 30, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 10, 17, 10, 32, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 12, 16, 12, 30, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 15, 19, 15, 32, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 17, 20, 17, 35, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 21, 20, 21, 33, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 23, 21, 23, 34, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 27, 20, 27, 33, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 29, 19, 29, 34, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 33, 21, 33, 34, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 35, 20, 35, 33, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 39, 22, 39, 35, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 41, 19, 41, 31, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 45, 19, 45, 32, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 47, 20, 47, 33, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 51, 20, 51, 33, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 53, 20, 53, 35, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 57, 20, 57, 33, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 59, 20, 59, 33, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 63, 21, 63, 34, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 65, 19, 65, 32, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 69, 22, 69, 35, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 71, 19, 71, 34, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 75, 19, 75, 36, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 77, 20, 77, 33, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 81, 20, 81, 37, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 83, 20, 83, 35, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 87, 18, 87, 32, "FIELD_SAME_DEFAULT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 89, 20, 89, 33, "FIELD_SAME_DEFAULT"),
	)
}

func TestRunBreakingFieldSameJSONName(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_field_same_json_name",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 6, 3, 6, 17, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 7, 18, 7, 35, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 8, 20, 8, 37, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 9, 3, 9, 27, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 10, 28, 10, 46, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 11, 27, 11, 45, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 12, 3, 12, 31, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 13, 32, 13, 50, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 14, 31, 14, 49, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 20, 7, 20, 21, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 21, 22, 21, 39, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 22, 24, 22, 41, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 23, 7, 23, 31, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 24, 32, 24, 50, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 25, 31, 25, 49, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 26, 7, 26, 35, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 27, 36, 27, 54, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 28, 35, 28, 53, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 43, 5, 43, 19, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 44, 20, 44, 37, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 45, 22, 45, 39, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 46, 5, 46, 29, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 47, 30, 47, 48, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 48, 29, 48, 47, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 49, 5, 49, 33, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 50, 34, 50, 52, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 51, 33, 51, 51, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 92, 5, 92, 19, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 93, 20, 93, 37, "FIELD_SAME_JSON_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 94, 22, 94, 39, "FIELD_SAME_JSON_NAME"),
	)
}

func TestRunBreakingFieldSameJSType(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_field_same_jstype",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 6, 18, 6, 36, "FIELD_SAME_JSTYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 7, 3, 7, 17, "FIELD_SAME_JSTYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 13, 22, 13, 40, "FIELD_SAME_JSTYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 14, 7, 14, 21, "FIELD_SAME_JSTYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 22, 20, 22, 38, "FIELD_SAME_JSTYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 51, 27, 51, 45, "FIELD_SAME_JSTYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 52, 3, 52, 26, "FIELD_SAME_JSTYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 57, 32, 57, 50, "FIELD_SAME_JSTYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 58, 34, 58, 52, "FIELD_SAME_JSTYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 59, 5, 59, 34, "FIELD_SAME_JSTYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 68, 29, 68, 47, "FIELD_SAME_JSTYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 69, 29, 69, 47, "FIELD_SAME_JSTYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 70, 3, 70, 28, "FIELD_SAME_JSTYPE"),
	)
}

func TestRunBreakingFieldSameLabel(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_field_same_label",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 8, 3, 8, 26, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 8, 3, 8, 26, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 8, 3, 8, 26, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 9, 3, 9, 24, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 9, 3, 9, 24, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 9, 3, 9, 24, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 10, 3, 10, 19, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 10, 3, 10, 19, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 10, 3, 10, 19, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 11, 3, 11, 16, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 11, 3, 11, 16, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 11, 3, 11, 16, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 12, 3, 12, 18, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 12, 3, 12, 18, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 12, 3, 12, 18, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 13, 3, 13, 17, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 13, 3, 13, 17, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 13, 3, 13, 17, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 19, 7, 19, 30, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 19, 7, 19, 30, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 19, 7, 19, 30, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 20, 7, 20, 28, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 20, 7, 20, 28, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 20, 7, 20, 28, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 21, 7, 21, 23, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 21, 7, 21, 23, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 21, 7, 21, 23, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 22, 7, 22, 20, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 22, 7, 22, 20, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 22, 7, 22, 20, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 23, 7, 23, 22, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 23, 7, 23, 22, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 23, 7, 23, 22, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 24, 7, 24, 21, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 24, 7, 24, 21, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 24, 7, 24, 21, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 32, 7, 32, 33, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 32, 7, 32, 33, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 40, 5, 40, 28, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 40, 5, 40, 28, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 40, 5, 40, 28, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 41, 5, 41, 26, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 41, 5, 41, 26, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 41, 5, 41, 26, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 42, 5, 42, 21, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 42, 5, 42, 21, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 42, 5, 42, 21, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 43, 5, 43, 18, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 43, 5, 43, 18, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 43, 5, 43, 18, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 44, 5, 44, 20, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 44, 5, 44, 20, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 44, 5, 44, 20, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 45, 5, 45, 19, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 45, 5, 45, 19, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 45, 5, 45, 19, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 48, 5, 48, 19, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 6, 3, 6, 26, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 13, 3, 13, 26, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 13, 3, 13, 26, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 13, 3, 13, 26, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 14, 3, 14, 24, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 14, 3, 14, 24, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 14, 3, 14, 24, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 70, 3, 70, 26, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 70, 3, 70, 26, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 70, 3, 70, 26, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 71, 3, 71, 26, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 71, 3, 71, 26, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 71, 3, 71, 26, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
	)
}

func TestRunBreakingFieldSameName(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_field_same_name",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 7, 9, 7, 13, "FIELD_SAME_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 15, 13, 15, 17, "FIELD_SAME_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 26, 11, 26, 15, "FIELD_SAME_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 35, 14, 35, 25, "FIELD_SAME_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 48, 23, 48, 33, "FIELD_SAME_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 64, 19, 64, 27, "FIELD_SAME_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 72, 19, 72, 29, "FIELD_SAME_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 78, 11, 78, 15, "FIELD_SAME_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 82, 19, 82, 27, "FIELD_SAME_NAME"),
	)
}

func TestRunBreakingFieldSameOneof(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_field_same_oneof",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 6, 3, 6, 17, "FIELD_SAME_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 8, 5, 8, 19, "FIELD_SAME_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 11, 5, 11, 21, "FIELD_SAME_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 18, 7, 18, 21, "FIELD_SAME_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 20, 9, 20, 23, "FIELD_SAME_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 23, 9, 23, 25, "FIELD_SAME_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 37, 5, 37, 19, "FIELD_SAME_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 39, 7, 39, 21, "FIELD_SAME_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 42, 7, 42, 23, "FIELD_SAME_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 95, 3, 95, 17, "FIELD_SAME_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 97, 5, 97, 19, "FIELD_SAME_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 100, 5, 100, 21, "FIELD_SAME_ONEOF"),
	)
}

func TestRunBreakingFieldSameType(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_field_same_type",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 8, 12, 8, 17, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 9, 12, 9, 15, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 11, 3, 11, 6, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 12, 3, 12, 6, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 13, 3, 13, 18, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 19, 16, 19, 21, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 20, 16, 20, 19, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 22, 7, 22, 10, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 23, 7, 23, 10, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 24, 7, 24, 22, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 36, 14, 36, 19, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 37, 14, 37, 17, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 39, 5, 39, 8, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 40, 5, 40, 8, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 41, 5, 41, 20, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 57, 5, 57, 10, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 58, 5, 58, 9, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 68, 12, 68, 16, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 8, 3, 8, 7, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 9, 3, 9, 7, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 31, 5, 31, 9, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 33, 5, 33, 10, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 38, 3, 38, 8, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 39, 3, 39, 7, "FIELD_SAME_TYPE"),
	)
}

func TestRunBreakingFieldSameUTF8Validation(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_field_same_utf8_validation",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 16, 3, 16, 27, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 17, 3, 17, 28, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 18, 3, 18, 21, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 19, 3, 19, 21, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 20, 3, 20, 22, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 20, 3, 20, 22, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 26, 3, 26, 27, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 27, 3, 27, 28, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 28, 3, 28, 21, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 29, 3, 29, 21, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 30, 3, 30, 22, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 30, 3, 30, 22, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 6, 3, 6, 18, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 7, 3, 7, 28, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 8, 3, 8, 21, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 9, 3, 9, 21, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 10, 3, 10, 22, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 10, 3, 10, 22, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 36, 3, 36, 18, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 37, 3, 37, 28, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 38, 3, 38, 21, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 39, 3, 39, 21, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 40, 3, 40, 22, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 40, 3, 40, 22, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 6, 3, 6, 18, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 7, 3, 7, 28, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 8, 3, 8, 21, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 9, 3, 9, 21, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 10, 3, 10, 22, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 10, 3, 10, 22, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 36, 3, 36, 18, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 37, 3, 37, 28, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 38, 3, 38, 21, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 39, 3, 39, 21, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 40, 3, 40, 22, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 40, 3, 40, 22, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 56, 19, 56, 50, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 57, 29, 57, 60, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 58, 3, 58, 21, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 59, 3, 59, 21, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 60, 3, 60, 22, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 60, 3, 60, 22, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "4.proto", 18, 3, 18, 18, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "4.proto", 19, 3, 19, 28, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "4.proto", 20, 3, 20, 21, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "4.proto", 21, 3, 21, 21, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "4.proto", 22, 3, 22, 22, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "4.proto", 22, 3, 22, 22, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "4.proto", 58, 19, 58, 52, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "4.proto", 59, 29, 59, 62, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "4.proto", 60, 3, 60, 21, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "4.proto", 61, 3, 61, 21, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "4.proto", 62, 3, 62, 22, "FIELD_SAME_UTF8_VALIDATION"),
		bufanalysistesting.NewFileAnnotation(t, "4.proto", 62, 3, 62, 22, "FIELD_SAME_UTF8_VALIDATION"),
	)
}

func TestRunBreakingFieldWireCompatibleCardinality(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_field_wire_compatible_cardinality",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 8, 3, 8, 26, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 9, 3, 9, 24, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 10, 3, 10, 19, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 11, 3, 11, 16, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 12, 3, 12, 18, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 13, 3, 13, 17, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 19, 7, 19, 30, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 20, 7, 20, 28, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 21, 7, 21, 23, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 22, 7, 22, 20, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 23, 7, 23, 22, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 24, 7, 24, 21, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 40, 5, 40, 28, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 41, 5, 41, 26, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 42, 5, 42, 21, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 43, 5, 43, 18, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 44, 5, 44, 20, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 45, 5, 45, 19, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 13, 3, 13, 26, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 14, 3, 14, 24, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 70, 3, 70, 26, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 71, 3, 71, 26, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 17, 7, 17, 32, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 25, 3, 25, 20, "FIELD_WIRE_COMPATIBLE_CARDINALITY"),
	)
}

func TestRunBreakingFieldWireCompatibleType(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_field_wire_compatible_type",
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 31, 3, 31, 11, "FIELD_WIRE_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 32, 3, 32, 9, "FIELD_WIRE_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 33, 3, 33, 8, "FIELD_WIRE_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 34, 3, 34, 8, "FIELD_WIRE_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 35, 3, 35, 9, "FIELD_WIRE_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 36, 3, 36, 10, "FIELD_WIRE_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 37, 3, 37, 11, "FIELD_WIRE_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 38, 3, 38, 9, "FIELD_WIRE_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 39, 3, 39, 9, "FIELD_WIRE_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 40, 3, 40, 9, "FIELD_WIRE_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 41, 3, 41, 8, "FIELD_WIRE_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 42, 3, 42, 9, "FIELD_WIRE_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 43, 3, 43, 8, "FIELD_WIRE_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 44, 3, 44, 9, "FIELD_WIRE_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 45, 3, 45, 7, "FIELD_WIRE_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 78, 3, 78, 6, "FIELD_WIRE_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 79, 3, 79, 6, "FIELD_WIRE_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 80, 3, 80, 6, "FIELD_WIRE_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 85, 3, 85, 9, "FIELD_WIRE_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 89, 12, 89, 15, "FIELD_WIRE_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 6, 3, 6, 10, "FIELD_WIRE_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 7, 3, 7, 10, "FIELD_WIRE_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 15, 5, 15, 12, "FIELD_WIRE_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 23, 3, 23, 9, "FIELD_WIRE_COMPATIBLE_TYPE"),
	)
}

func TestRunBreakingFieldWireJSONCompatibleCardinality(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_field_wire_json_compatible_cardinality",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 8, 3, 8, 26, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 9, 3, 9, 24, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 10, 3, 10, 19, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 11, 3, 11, 16, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 12, 3, 12, 18, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 13, 3, 13, 17, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 19, 7, 19, 30, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 20, 7, 20, 28, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 21, 7, 21, 23, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 22, 7, 22, 20, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 23, 7, 23, 22, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 24, 7, 24, 21, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 32, 7, 32, 33, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 40, 5, 40, 28, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 41, 5, 41, 26, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 42, 5, 42, 21, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 43, 5, 43, 18, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 44, 5, 44, 20, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 45, 5, 45, 19, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 13, 3, 13, 26, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 14, 3, 14, 24, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 70, 3, 70, 26, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 71, 3, 71, 26, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 17, 7, 17, 32, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 25, 3, 25, 20, "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY"),
	)
}

func TestRunBreakingFieldWireJSONCompatibleType(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_field_wire_json_compatible_type",
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 34, 3, 34, 11, "FIELD_WIRE_JSON_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 35, 3, 35, 9, "FIELD_WIRE_JSON_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 36, 3, 36, 8, "FIELD_WIRE_JSON_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 37, 3, 37, 8, "FIELD_WIRE_JSON_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 38, 3, 38, 9, "FIELD_WIRE_JSON_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 39, 3, 39, 10, "FIELD_WIRE_JSON_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 40, 3, 40, 11, "FIELD_WIRE_JSON_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 41, 3, 41, 9, "FIELD_WIRE_JSON_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 42, 3, 42, 9, "FIELD_WIRE_JSON_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 43, 3, 43, 9, "FIELD_WIRE_JSON_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 44, 3, 44, 8, "FIELD_WIRE_JSON_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 45, 3, 45, 9, "FIELD_WIRE_JSON_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 46, 3, 46, 8, "FIELD_WIRE_JSON_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 47, 3, 47, 9, "FIELD_WIRE_JSON_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 48, 3, 48, 7, "FIELD_WIRE_JSON_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 81, 3, 81, 6, "FIELD_WIRE_JSON_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 82, 3, 82, 6, "FIELD_WIRE_JSON_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 83, 3, 83, 6, "FIELD_WIRE_JSON_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 87, 3, 87, 8, "FIELD_WIRE_JSON_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 88, 3, 88, 9, "FIELD_WIRE_JSON_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 92, 12, 92, 15, "FIELD_WIRE_JSON_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 93, 3, 93, 8, "FIELD_WIRE_JSON_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 6, 3, 6, 10, "FIELD_WIRE_JSON_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 7, 3, 7, 10, "FIELD_WIRE_JSON_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 14, 5, 14, 10, "FIELD_WIRE_JSON_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 15, 5, 15, 12, "FIELD_WIRE_JSON_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 16, 5, 16, 11, "FIELD_WIRE_JSON_COMPATIBLE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 23, 3, 23, 9, "FIELD_WIRE_JSON_COMPATIBLE_TYPE"),
	)
}

func TestRunBreakingFileNoDelete(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_file_no_delete",
		bufanalysistesting.NewFileAnnotationNoLocationOrPath(t, "FILE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotationNoLocationOrPath(t, "FILE_NO_DELETE"),
	)
}

func TestRunBreakingFileNoDeleteUnstable(t *testing.T) {
	t.Parallel()
	// https://github.com/bufbuild/buf/issues/211
	testBreaking(
		t,
		"breaking_file_no_delete_unstable",
	)
}

func TestRunBreakingFileNoDeleteIgnores(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_file_no_delete_ignores",
		// a/a.proto deleted but not ignored
		bufanalysistesting.NewFileAnnotationNoLocationOrPath(t, "FILE_NO_DELETE"),
	)
}

func TestRunBreakingFileSamePackage(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_file_same_package",
		bufanalysistesting.NewFileAnnotation(t, "a/a.proto", 3, 1, 3, 11, "FILE_SAME_PACKAGE"),
		bufanalysistesting.NewFileAnnotation(t, "no_package.proto", 3, 1, 3, 11, "FILE_SAME_PACKAGE"),
	)
}

func TestRunBreakingFileSameSyntax(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_file_same_syntax",
		bufanalysistesting.NewFileAnnotation(t, "no_package.proto", 1, 1, 1, 19, "FILE_SAME_SYNTAX"),
		bufanalysistesting.NewFileAnnotation(t, "sub/a/b/sub_a_b.proto", 2, 1, 2, 19, "FILE_SAME_SYNTAX"),
	)
}

func TestRunBreakingFileSameValues(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_file_same_values",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 5, 1, 5, 29, "FILE_SAME_JAVA_PACKAGE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 6, 1, 6, 37, "FILE_SAME_JAVA_OUTER_CLASSNAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 7, 1, 7, 36, "FILE_SAME_JAVA_MULTIPLE_FILES"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 8, 1, 8, 27, "FILE_SAME_GO_PACKAGE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 9, 1, 9, 34, "FILE_SAME_OBJC_CLASS_PREFIX"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 10, 1, 10, 33, "FILE_SAME_CSHARP_NAMESPACE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 11, 1, 11, 29, "FILE_SAME_SWIFT_PREFIX"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 12, 1, 12, 33, "FILE_SAME_PHP_CLASS_PREFIX"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 13, 1, 13, 30, "FILE_SAME_PHP_NAMESPACE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 14, 1, 14, 39, "FILE_SAME_PHP_METADATA_NAMESPACE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 15, 1, 15, 29, "FILE_SAME_RUBY_PACKAGE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 18, 1, 18, 29, "FILE_SAME_OPTIMIZE_FOR"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 19, 1, 19, 36, "FILE_SAME_CC_GENERIC_SERVICES"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 20, 1, 20, 38, "FILE_SAME_JAVA_GENERIC_SERVICES"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 21, 1, 21, 36, "FILE_SAME_PY_GENERIC_SERVICES"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 23, 1, 23, 33, "FILE_SAME_CC_ENABLE_ARENAS"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 3, 1, 3, 11, "FILE_SAME_PACKAGE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 5, 1, 5, 29, "FILE_SAME_JAVA_PACKAGE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 6, 1, 6, 37, "FILE_SAME_JAVA_OUTER_CLASSNAME"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 7, 1, 7, 35, "FILE_SAME_JAVA_MULTIPLE_FILES"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 8, 1, 8, 27, "FILE_SAME_GO_PACKAGE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 9, 1, 9, 34, "FILE_SAME_OBJC_CLASS_PREFIX"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 10, 1, 10, 33, "FILE_SAME_CSHARP_NAMESPACE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 11, 1, 11, 29, "FILE_SAME_SWIFT_PREFIX"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 12, 1, 12, 33, "FILE_SAME_PHP_CLASS_PREFIX"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 13, 1, 13, 30, "FILE_SAME_PHP_NAMESPACE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 14, 1, 14, 39, "FILE_SAME_PHP_METADATA_NAMESPACE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 15, 1, 15, 29, "FILE_SAME_RUBY_PACKAGE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 18, 1, 18, 33, "FILE_SAME_OPTIMIZE_FOR"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 19, 1, 19, 35, "FILE_SAME_CC_GENERIC_SERVICES"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 20, 1, 20, 37, "FILE_SAME_JAVA_GENERIC_SERVICES"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 21, 1, 21, 35, "FILE_SAME_PY_GENERIC_SERVICES"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 23, 1, 23, 33, "FILE_SAME_CC_ENABLE_ARENAS"),
	)
}

func TestRunBreakingMessageNoDelete(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_message_no_delete",
		bufanalysistesting.NewFileAnnotationNoLocation(t, "1.proto", "MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 7, 1, 12, 2, "MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 8, 3, 10, 4, "MESSAGE_NO_DELETE"),
	)
}

func TestRunBreakingMessageSameJSONFormat(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_message_same_json_format",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 9, 1, 11, 2, "MESSAGE_SAME_JSON_FORMAT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 13, 1, 15, 2, "MESSAGE_SAME_JSON_FORMAT"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 26, 3, 26, 52, "MESSAGE_SAME_JSON_FORMAT"),
		bufanalysistesting.NewFileAnnotation(t, "4.proto", 11, 1, 13, 2, "MESSAGE_SAME_JSON_FORMAT"),
	)
}

func TestRunBreakingMessageSameValues(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_message_same_values",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 6, 3, 6, 49, "MESSAGE_NO_REMOVE_STANDARD_DESCRIPTOR_ACCESSOR"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 12, 7, 12, 53, "MESSAGE_NO_REMOVE_STANDARD_DESCRIPTOR_ACCESSOR"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 23, 3, 23, 49, "MESSAGE_NO_REMOVE_STANDARD_DESCRIPTOR_ACCESSOR"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 6, 3, 6, 49, "MESSAGE_NO_REMOVE_STANDARD_DESCRIPTOR_ACCESSOR"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 10, 3, 10, 49, "MESSAGE_NO_REMOVE_STANDARD_DESCRIPTOR_ACCESSOR"),
	)
}

func TestRunBreakingMessageSameRequiredFields(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_message_same_required_fields",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 5, 1, 7, 2, "MESSAGE_SAME_REQUIRED_FIELDS"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 6, 3, 6, 27, "MESSAGE_SAME_REQUIRED_FIELDS"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 9, 1, 11, 2, "MESSAGE_SAME_REQUIRED_FIELDS"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 10, 3, 10, 27, "MESSAGE_SAME_REQUIRED_FIELDS"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 13, 1, 30, 2, "MESSAGE_SAME_REQUIRED_FIELDS"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 15, 5, 17, 6, "MESSAGE_SAME_REQUIRED_FIELDS"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 16, 7, 16, 31, "MESSAGE_SAME_REQUIRED_FIELDS"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 18, 5, 20, 6, "MESSAGE_SAME_REQUIRED_FIELDS"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 22, 3, 24, 4, "MESSAGE_SAME_REQUIRED_FIELDS"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 23, 5, 23, 29, "MESSAGE_SAME_REQUIRED_FIELDS"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 27, 5, 27, 29, "MESSAGE_SAME_REQUIRED_FIELDS"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 29, 3, 29, 27, "MESSAGE_SAME_REQUIRED_FIELDS"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 5, 1, 7, 2, "MESSAGE_SAME_REQUIRED_FIELDS"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 6, 3, 6, 27, "MESSAGE_SAME_REQUIRED_FIELDS"),
	)
}

func TestRunBreakingOneofNoDelete(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_oneof_no_delete",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 5, 1, 9, 2, "ONEOF_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 13, 5, 17, 6, "ONEOF_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 26, 3, 30, 4, "ONEOF_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 76, 1, 80, 2, "ONEOF_NO_DELETE"),
	)
}

func TestRunBreakingPackageExtensionNoDelete(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_package_extension_no_delete",
		bufanalysistesting.NewFileAnnotationNoLocation(t, "2.proto", "PACKAGE_EXTENSION_NO_DELETE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "3.proto", "PACKAGE_EXTENSION_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "3.proto", 8, 3, 14, 4, "PACKAGE_EXTENSION_NO_DELETE"),
	)
}

func TestRunBreakingPackageNoDelete(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_package_no_delete",
		bufanalysistesting.NewFileAnnotationNoLocationOrPath(t, "PACKAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "a1.proto", "PACKAGE_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "a1.proto", "PACKAGE_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "a1.proto", "PACKAGE_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "a1.proto", "PACKAGE_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "a1.proto", "PACKAGE_SERVICE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "a2.proto", 11, 1, 16, 2, "PACKAGE_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "a2.proto", 12, 3, 14, 4, "PACKAGE_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "b1.proto", "PACKAGE_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "b1.proto", "PACKAGE_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "b1.proto", "PACKAGE_SERVICE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "b2.proto", 7, 1, 21, 2, "PACKAGE_MESSAGE_NO_DELETE"),
	)
}

func TestRunBreakingPackageServiceNoDelete(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_package_service_no_delete",
		bufanalysistesting.NewFileAnnotationNoLocation(t, "1.proto", "PACKAGE_SERVICE_NO_DELETE"),
	)
}

func TestRunBreakingReservedEnumNoDelete(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_reserved_enum_no_delete",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 5, 1, 12, 2, "RESERVED_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 5, 1, 12, 2, "RESERVED_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 5, 1, 12, 2, "RESERVED_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 5, 1, 12, 2, "RESERVED_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 5, 1, 12, 2, "RESERVED_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 16, 5, 23, 6, "RESERVED_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 16, 5, 23, 6, "RESERVED_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 16, 5, 23, 6, "RESERVED_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 16, 5, 23, 6, "RESERVED_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 16, 5, 23, 6, "RESERVED_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 34, 3, 41, 4, "RESERVED_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 34, 3, 41, 4, "RESERVED_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 34, 3, 41, 4, "RESERVED_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 34, 3, 41, 4, "RESERVED_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 34, 3, 41, 4, "RESERVED_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 53, 1, 58, 2, "RESERVED_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 88, 1, 95, 2, "RESERVED_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 88, 1, 95, 2, "RESERVED_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 88, 1, 95, 2, "RESERVED_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 88, 1, 95, 2, "RESERVED_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 88, 1, 95, 2, "RESERVED_ENUM_NO_DELETE"),
	)
}

func TestRunBreakingReservedMessageNoDelete(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_reserved_message_no_delete",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 5, 1, 12, 2, "RESERVED_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 5, 1, 12, 2, "RESERVED_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 5, 1, 12, 2, "RESERVED_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 5, 1, 12, 2, "RESERVED_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 5, 1, 12, 2, "RESERVED_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 16, 5, 23, 6, "RESERVED_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 16, 5, 23, 6, "RESERVED_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 16, 5, 23, 6, "RESERVED_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 16, 5, 23, 6, "RESERVED_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 16, 5, 23, 6, "RESERVED_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 34, 3, 41, 4, "RESERVED_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 34, 3, 41, 4, "RESERVED_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 34, 3, 41, 4, "RESERVED_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 34, 3, 41, 4, "RESERVED_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 34, 3, 41, 4, "RESERVED_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 56, 1, 60, 2, "RESERVED_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 91, 1, 98, 2, "RESERVED_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 91, 1, 98, 2, "RESERVED_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 91, 1, 98, 2, "RESERVED_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 91, 1, 98, 2, "RESERVED_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 91, 1, 98, 2, "RESERVED_MESSAGE_NO_DELETE"),
	)
}

func TestRunBreakingRPCNoDelete(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_rpc_no_delete",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 7, 1, 10, 2, "RPC_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 31, 1, 34, 2, "RPC_NO_DELETE"),
	)
}

func TestRunBreakingRPCSameValues(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_rpc_same_values",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 9, 3, 9, 71, "RPC_SAME_CLIENT_STREAMING"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 9, 18, 9, 37, "RPC_SAME_REQUEST_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 9, 48, 9, 67, "RPC_SAME_RESPONSE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 10, 3, 10, 71, "RPC_SAME_SERVER_STREAMING"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 10, 11, 10, 30, "RPC_SAME_REQUEST_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 10, 48, 10, 67, "RPC_SAME_RESPONSE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 11, 3, 11, 68, "RPC_SAME_CLIENT_STREAMING"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 12, 3, 12, 68, "RPC_SAME_SERVER_STREAMING"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 37, 3, 37, 71, "RPC_SAME_CLIENT_STREAMING"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 37, 18, 37, 37, "RPC_SAME_REQUEST_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 37, 48, 37, 67, "RPC_SAME_RESPONSE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 38, 3, 38, 71, "RPC_SAME_SERVER_STREAMING"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 38, 11, 38, 30, "RPC_SAME_REQUEST_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 38, 48, 38, 67, "RPC_SAME_RESPONSE_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 39, 3, 39, 68, "RPC_SAME_CLIENT_STREAMING"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 40, 3, 40, 68, "RPC_SAME_SERVER_STREAMING"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 45, 5, 45, 48, "RPC_SAME_IDEMPOTENCY_LEVEL"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 49, 5, 49, 43, "RPC_SAME_IDEMPOTENCY_LEVEL"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 55, 5, 55, 48, "RPC_SAME_IDEMPOTENCY_LEVEL"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 59, 5, 59, 43, "RPC_SAME_IDEMPOTENCY_LEVEL"),
	)
}

func TestRunBreakingServiceNoDelete(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_service_no_delete",
		bufanalysistesting.NewFileAnnotationNoLocation(t, "1.proto", "SERVICE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "1.proto", "SERVICE_NO_DELETE"),
	)
}

func TestRunBreakingIgnoreUnstablePackagesTrue(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_ignore_unstable_packages_true",
		bufanalysistesting.NewFileAnnotationNoLocation(t, "1.proto", "ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 7, 1, 16, 2, "ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 8, 3, 12, 4, "ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "a/v1/1.proto", "ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "a/v1/1.proto", 9, 1, 18, 2, "ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "a/v1/1.proto", 10, 3, 14, 4, "ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "b/1.proto", "ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "b/1.proto", 9, 1, 18, 2, "ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "b/1.proto", 10, 3, 14, 4, "ENUM_NO_DELETE"),
	)
}

func TestRunBreakingIgnoreUnstablePackagesFalse(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_ignore_unstable_packages_false",
		bufanalysistesting.NewFileAnnotationNoLocation(t, "1.proto", "ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 7, 1, 16, 2, "ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 8, 3, 12, 4, "ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "a/v1/1.proto", "ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "a/v1/1.proto", 9, 1, 18, 2, "ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "a/v1/1.proto", 10, 3, 14, 4, "ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "a/v1beta1/1.proto", "ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "a/v1beta1/1.proto", 9, 1, 18, 2, "ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "a/v1beta1/1.proto", 10, 3, 14, 4, "ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "b/1.proto", "ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "b/1.proto", 9, 1, 18, 2, "ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "b/1.proto", 10, 3, 14, 4, "ENUM_NO_DELETE"),
	)
}

func TestRunBreakingIgnoreUnstablePackagesDeleteFile(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_ignore_unstable_packages_delete_file",
	)
}

func TestRunBreakingIntEnum(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_int_enum",
		bufanalysistesting.NewFileAnnotationNoLocation(t, "1.proto", "ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 3, 3, 3, 8, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 3, 3, 3, 7, "FIELD_SAME_TYPE"),
	)
}

func TestRunBreakingMessageEnum(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_message_enum",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 3, 3, 3, 15, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 3, 3, 3, 6, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "2.proto", "ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 3, 3, 3, 16, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 3, 3, 3, 7, "FIELD_SAME_TYPE"),
	)
}

func TestRunBreakingMessageInt(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_message_int",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 3, 3, 3, 15, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 3, 3, 3, 6, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 3, 3, 3, 17, "FIELD_SAME_CARDINALITY"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 3, 3, 3, 8, "FIELD_SAME_TYPE"),
	)
}

func TestRunBreakingMessageMessage(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_message_message",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 3, 3, 3, 8, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "2.proto", "MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 3, 3, 3, 7, "FIELD_SAME_TYPE"),
	)
}

func TestRunBreakingWithCustomPlugins(t *testing.T) {
	t.Parallel()
	testBreaking(
		t,
		"breaking_custom_plugins",
		bufanalysistesting.NewFileAnnotation(t, "a.proto", 3, 1, 6, 2, "SERVICE_SUFFIXES_NO_CHANGE"),
		bufanalysistesting.NewFileAnnotation(t, "b.proto", 11, 3, 14, 4, "ENUM_SUFFIXES_NO_CHANGE"),
		bufanalysistesting.NewFileAnnotation(t, "b.proto", 15, 3, 19, 4, "MESSAGE_SUFFIXES_NO_CHANGE"),
	)
}

func testBreaking(
	t *testing.T,
	relDirPath string,
	expectedFileAnnotations ...bufanalysis.FileAnnotation,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	logger := slogtestext.NewLogger(t)

	baseDirPath := filepath.Join("testdata", "breaking", "current")
	basePreviousDirPath := filepath.Join("testdata", "breaking", "previous")
	previousDirPath := filepath.Join(basePreviousDirPath, relDirPath)
	dirPath := filepath.Join(baseDirPath, relDirPath)

	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	previousReadWriteBucket, err := storageosProvider.NewReadWriteBucket(
		previousDirPath,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	require.NoError(t, err)
	previousBucketTargeting, err := buftarget.NewBucketTargeting(
		ctx,
		logger,
		previousReadWriteBucket,
		".", // the bucket is rooted at the input
		nil,
		nil,
		buftarget.TerminateAtControllingWorkspace,
	)
	require.NoError(t, err)
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(
		dirPath,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	require.NoError(t, err)
	bucketTargeting, err := buftarget.NewBucketTargeting(
		ctx,
		logger,
		readWriteBucket,
		".", // the bucket is rooted at the input
		nil,
		nil,
		buftarget.TerminateAtControllingWorkspace,
	)
	require.NoError(t, err)

	workspaceProvider := bufworkspace.NewWorkspaceProvider(
		logger,
		bufmodule.NopGraphProvider,
		bufmodule.NopModuleDataProvider,
		bufmodule.NopCommitProvider,
	)
	previousWorkspace, err := workspaceProvider.GetWorkspaceForBucket(
		ctx,
		previousReadWriteBucket,
		previousBucketTargeting,
	)
	require.NoError(t, err)
	workspace, err := workspaceProvider.GetWorkspaceForBucket(
		ctx,
		readWriteBucket,
		bucketTargeting,
	)
	require.NoError(t, err)

	previousImage, err := bufimage.BuildImage(
		ctx,
		logger,
		bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(previousWorkspace),
	)
	require.NoError(t, err)

	image, err := bufimage.BuildImage(
		ctx,
		logger,
		bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(workspace),
	)
	require.NoError(t, err)

	opaqueID, err := testGetRootOpaqueID(workspace, ".")
	require.NoError(t, err)
	breakingConfig := workspace.GetBreakingConfigForOpaqueID(opaqueID)
	require.NotNil(t, breakingConfig)
	client, err := bufcheck.NewClient(logger, bufcheck.NewRunnerProvider(command.NewRunner(), wasm.UnimplementedRuntime))
	require.NoError(t, err)
	err = client.Breaking(
		ctx,
		breakingConfig,
		image,
		previousImage,
		bufcheck.BreakingWithExcludeImports(),
		bufcheck.WithPluginConfigs(workspace.PluginConfigs()...),
	)
	if len(expectedFileAnnotations) == 0 {
		assert.NoError(t, err)
	} else {
		var fileAnnotationSet bufanalysis.FileAnnotationSet
		require.ErrorAs(t, err, &fileAnnotationSet)
		bufanalysistesting.AssertFileAnnotationsEqual(
			t,
			expectedFileAnnotations,
			fileAnnotationSet.FileAnnotations(),
		)
	}
}

func testGetRootOpaqueID(workspace bufworkspace.Workspace, prefix string) (string, error) {
	var rootModules []bufmodule.Module
	for _, module := range workspace.Modules() {
		if strings.HasPrefix(module.OpaqueID(), prefix) {
			rootModules = append(rootModules, module)
		}
	}
	switch len(rootModules) {
	case 0:
		return "", errors.New("no module with opaque ID starting with \".\"")
	case 1:
		return rootModules[0].OpaqueID(), nil
	default:
		return "", fmt.Errorf("got %d modules with opaque ID starting with \".\"", len(rootModules))
	}
}

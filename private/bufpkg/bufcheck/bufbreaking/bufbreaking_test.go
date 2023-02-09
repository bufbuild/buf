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

package bufbreaking_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis/bufanalysistesting"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking"
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

func TestRunBreakingEnumNoDelete(t *testing.T) {
	testBreaking(
		t,
		"breaking_enum_no_delete",
		bufanalysistesting.NewFileAnnotationNoLocation(t, "1.proto", "ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 9, 1, 18, 2, "ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 10, 3, 14, 4, "ENUM_NO_DELETE"),
	)
}

func TestRunBreakingEnumValueNoDelete(t *testing.T) {
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
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 73, 1, 79, 2, "EXTENSION_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 73, 1, 79, 2, "EXTENSION_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 73, 1, 79, 2, "EXTENSION_MESSAGE_NO_DELETE"),
	)
}

func TestRunBreakingFieldNoDelete(t *testing.T) {
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

func TestRunBreakingFieldSameCType(t *testing.T) {
	testBreaking(
		t,
		"breaking_field_same_ctype",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 6, 19, 6, 39, "FIELD_SAME_CTYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 7, 3, 7, 18, "FIELD_SAME_CTYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 13, 23, 13, 43, "FIELD_SAME_CTYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 14, 7, 14, 22, "FIELD_SAME_CTYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 23, 21, 23, 33, "FIELD_SAME_CTYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 49, 28, 49, 48, "FIELD_SAME_CTYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 50, 28, 50, 42, "FIELD_SAME_CTYPE"),
	)
}

func TestRunBreakingFieldSameJSONName(t *testing.T) {
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
	testBreaking(
		t,
		"breaking_field_same_jstype",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 6, 18, 6, 36, "FIELD_SAME_JSTYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 7, 3, 7, 17, "FIELD_SAME_JSTYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 13, 22, 13, 40, "FIELD_SAME_JSTYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 14, 7, 14, 21, "FIELD_SAME_JSTYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 22, 20, 22, 38, "FIELD_SAME_JSTYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 49, 27, 49, 45, "FIELD_SAME_JSTYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 50, 3, 50, 26, "FIELD_SAME_JSTYPE"),
	)
}

func TestRunBreakingFieldSameLabel(t *testing.T) {
	testBreaking(
		t,
		"breaking_field_same_label",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 8, 3, 8, 26, "FIELD_SAME_LABEL"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 9, 3, 9, 24, "FIELD_SAME_LABEL"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 10, 3, 10, 19, "FIELD_SAME_LABEL"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 11, 3, 11, 16, "FIELD_SAME_LABEL"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 12, 3, 12, 18, "FIELD_SAME_LABEL"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 13, 3, 13, 17, "FIELD_SAME_LABEL"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 19, 7, 19, 30, "FIELD_SAME_LABEL"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 20, 7, 20, 28, "FIELD_SAME_LABEL"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 21, 7, 21, 23, "FIELD_SAME_LABEL"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 22, 7, 22, 20, "FIELD_SAME_LABEL"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 23, 7, 23, 22, "FIELD_SAME_LABEL"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 24, 7, 24, 21, "FIELD_SAME_LABEL"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 36, 5, 36, 28, "FIELD_SAME_LABEL"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 37, 5, 37, 26, "FIELD_SAME_LABEL"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 38, 5, 38, 21, "FIELD_SAME_LABEL"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 39, 5, 39, 18, "FIELD_SAME_LABEL"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 40, 5, 40, 20, "FIELD_SAME_LABEL"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 41, 5, 41, 19, "FIELD_SAME_LABEL"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 13, 3, 13, 26, "FIELD_SAME_LABEL"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 14, 3, 14, 24, "FIELD_SAME_LABEL"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 70, 3, 70, 26, "FIELD_SAME_LABEL"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 71, 3, 71, 26, "FIELD_SAME_LABEL"),
	)
}

func TestRunBreakingFieldSameName(t *testing.T) {
	testBreaking(
		t,
		"breaking_field_same_name",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 7, 9, 7, 13, "FIELD_SAME_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 15, 13, 15, 17, "FIELD_SAME_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 26, 11, 26, 15, "FIELD_SAME_NAME"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 60, 11, 60, 15, "FIELD_SAME_NAME"),
	)
}

func TestRunBreakingFieldSameOneof(t *testing.T) {
	testBreaking(
		t,
		"breaking_field_same_oneof",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 6, 3, 6, 17, "FIELD_SAME_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 8, 3, 8, 17, "FIELD_SAME_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 11, 3, 11, 19, "FIELD_SAME_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 18, 3, 18, 17, "FIELD_SAME_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 20, 3, 20, 17, "FIELD_SAME_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 23, 3, 23, 19, "FIELD_SAME_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 37, 3, 37, 17, "FIELD_SAME_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 39, 3, 39, 17, "FIELD_SAME_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 42, 3, 42, 19, "FIELD_SAME_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 94, 3, 94, 17, "FIELD_SAME_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 96, 3, 96, 17, "FIELD_SAME_ONEOF"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 99, 3, 99, 19, "FIELD_SAME_ONEOF"),
	)
}

func TestRunBreakingFieldSameType(t *testing.T) {
	// TODO: double check all this
	testBreaking(
		t,
		"breaking_field_same_type",
		bufanalysistesting.NewFileAnnotationNoLocation(t, "1.proto", "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "1.proto", "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "1.proto", "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 8, 12, 8, 17, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 9, 12, 9, 15, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 11, 3, 11, 6, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 12, 3, 12, 6, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 19, 16, 19, 21, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 20, 16, 20, 19, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 22, 7, 22, 10, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 23, 7, 23, 10, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 36, 14, 36, 19, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 37, 14, 37, 17, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 39, 5, 39, 8, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 40, 5, 40, 8, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 64, 5, 64, 10, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 65, 5, 65, 9, "FIELD_SAME_TYPE"),
	)
}

func TestRunBreakingFieldWireCompatibleType(t *testing.T) {
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
	)
}
func TestRunBreakingFieldWireJSONCompatibleType(t *testing.T) {
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
	)
}

func TestRunBreakingFileNoDelete(t *testing.T) {
	testBreaking(
		t,
		"breaking_file_no_delete",
		bufanalysistesting.NewFileAnnotationNoLocationOrPath(t, "FILE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotationNoLocationOrPath(t, "FILE_NO_DELETE"),
	)
}

func TestRunBreakingFileNoDeleteUnstable(t *testing.T) {
	// https://github.com/bufbuild/buf/issues/211
	testBreaking(
		t,
		"breaking_file_no_delete_unstable",
	)
}

func TestRunBreakingFileNoDeleteIgnores(t *testing.T) {
	testBreaking(
		t,
		"breaking_file_no_delete_ignores",
		// a/a.proto deleted but not ignored
		bufanalysistesting.NewFileAnnotationNoLocationOrPath(t, "FILE_NO_DELETE"),
	)
}

func TestRunBreakingFileSamePackage(t *testing.T) {
	testBreaking(
		t,
		"breaking_file_same_package",
		bufanalysistesting.NewFileAnnotation(t, "a/a.proto", 3, 1, 3, 11, "FILE_SAME_PACKAGE"),
		bufanalysistesting.NewFileAnnotation(t, "no_package.proto", 3, 1, 3, 11, "FILE_SAME_PACKAGE"),
	)
}

func TestRunBreakingFileSameSyntax(t *testing.T) {
	testBreaking(
		t,
		"breaking_file_same_syntax",
		bufanalysistesting.NewFileAnnotation(t, "no_package.proto", 1, 1, 1, 19, "FILE_SAME_SYNTAX"),
		bufanalysistesting.NewFileAnnotation(t, "sub/a/b/sub_a_b.proto", 2, 1, 2, 19, "FILE_SAME_SYNTAX"),
	)
}

func TestRunBreakingFileSameValues(t *testing.T) {
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
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 17, 1, 17, 39, "FILE_SAME_JAVA_STRING_CHECK_UTF8"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 18, 1, 18, 29, "FILE_SAME_OPTIMIZE_FOR"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 19, 1, 19, 36, "FILE_SAME_CC_GENERIC_SERVICES"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 20, 1, 20, 38, "FILE_SAME_JAVA_GENERIC_SERVICES"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 21, 1, 21, 36, "FILE_SAME_PY_GENERIC_SERVICES"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 22, 1, 22, 37, "FILE_SAME_PHP_GENERIC_SERVICES"),
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
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 17, 1, 17, 38, "FILE_SAME_JAVA_STRING_CHECK_UTF8"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 18, 1, 18, 33, "FILE_SAME_OPTIMIZE_FOR"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 19, 1, 19, 35, "FILE_SAME_CC_GENERIC_SERVICES"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 20, 1, 20, 37, "FILE_SAME_JAVA_GENERIC_SERVICES"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 21, 1, 21, 35, "FILE_SAME_PY_GENERIC_SERVICES"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 22, 1, 22, 36, "FILE_SAME_PHP_GENERIC_SERVICES"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 23, 1, 23, 33, "FILE_SAME_CC_ENABLE_ARENAS"),
	)
}

func TestRunBreakingMessageNoDelete(t *testing.T) {
	testBreaking(
		t,
		"breaking_message_no_delete",
		bufanalysistesting.NewFileAnnotationNoLocation(t, "1.proto", "MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 7, 1, 12, 2, "MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 8, 3, 10, 4, "MESSAGE_NO_DELETE"),
	)
}

func TestRunBreakingMessageSameValues(t *testing.T) {
	testBreaking(
		t,
		"breaking_message_same_values",
		bufanalysistesting.NewFileAnnotationNoLocation(t, "1.proto", "MESSAGE_SAME_MESSAGE_SET_WIRE_FORMAT"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "1.proto", "MESSAGE_SAME_MESSAGE_SET_WIRE_FORMAT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 6, 3, 6, 42, "MESSAGE_SAME_MESSAGE_SET_WIRE_FORMAT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 7, 3, 7, 49, "MESSAGE_NO_REMOVE_STANDARD_DESCRIPTOR_ACCESSOR"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 13, 7, 13, 53, "MESSAGE_NO_REMOVE_STANDARD_DESCRIPTOR_ACCESSOR"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 16, 7, 16, 45, "MESSAGE_SAME_MESSAGE_SET_WIRE_FORMAT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 21, 5, 21, 44, "MESSAGE_SAME_MESSAGE_SET_WIRE_FORMAT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 24, 5, 24, 43, "MESSAGE_SAME_MESSAGE_SET_WIRE_FORMAT"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 27, 3, 27, 49, "MESSAGE_NO_REMOVE_STANDARD_DESCRIPTOR_ACCESSOR"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "2.proto", "MESSAGE_SAME_MESSAGE_SET_WIRE_FORMAT"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "2.proto", "MESSAGE_SAME_MESSAGE_SET_WIRE_FORMAT"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 6, 3, 6, 49, "MESSAGE_NO_REMOVE_STANDARD_DESCRIPTOR_ACCESSOR"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 10, 3, 10, 49, "MESSAGE_NO_REMOVE_STANDARD_DESCRIPTOR_ACCESSOR"),
	)
}

func TestRunBreakingMessageSameRequiredFields(t *testing.T) {
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
	testBreaking(
		t,
		"breaking_oneof_no_delete",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 5, 1, 9, 2, "ONEOF_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 13, 5, 17, 6, "ONEOF_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 26, 3, 30, 4, "ONEOF_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 75, 1, 79, 2, "ONEOF_NO_DELETE"),
	)
}

func TestRunBreakingPackageNoDelete(t *testing.T) {
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

func TestRunBreakingReservedEnumNoDelete(t *testing.T) {
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
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 86, 1, 93, 2, "RESERVED_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 86, 1, 93, 2, "RESERVED_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 86, 1, 93, 2, "RESERVED_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 86, 1, 93, 2, "RESERVED_ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 86, 1, 93, 2, "RESERVED_ENUM_NO_DELETE"),
	)
}

func TestRunBreakingReservedMessageNoDelete(t *testing.T) {
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
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 89, 1, 96, 2, "RESERVED_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 89, 1, 96, 2, "RESERVED_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 89, 1, 96, 2, "RESERVED_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 89, 1, 96, 2, "RESERVED_MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 89, 1, 96, 2, "RESERVED_MESSAGE_NO_DELETE"),
	)
}

func TestRunBreakingRPCNoDelete(t *testing.T) {
	testBreaking(
		t,
		"breaking_rpc_no_delete",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 7, 1, 10, 2, "RPC_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 31, 1, 34, 2, "RPC_NO_DELETE"),
	)
}

func TestRunBreakingRPCSameValues(t *testing.T) {
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
	testBreaking(
		t,
		"breaking_service_no_delete",
		bufanalysistesting.NewFileAnnotationNoLocation(t, "1.proto", "SERVICE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotationNoLocation(t, "1.proto", "SERVICE_NO_DELETE"),
	)
}

func TestRunBreakingPackageServiceNoDelete(t *testing.T) {
	testBreaking(
		t,
		"breaking_package_service_no_delete",
		bufanalysistesting.NewFileAnnotationNoLocation(t, "1.proto", "PACKAGE_SERVICE_NO_DELETE"),
	)
}

func TestRunBreakingIgnoreUnstablePackagesTrue(t *testing.T) {
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
	testBreaking(
		t,
		"breaking_ignore_unstable_packages_delete_file",
	)
}

func TestRunBreakingIntEnum(t *testing.T) {
	testBreaking(
		t,
		"breaking_int_enum",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 0, 0, 0, 0, "ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 3, 3, 3, 8, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 3, 3, 3, 7, "FIELD_SAME_TYPE"),
	)
}

func TestRunBreakingMessageEnum(t *testing.T) {
	testBreaking(
		t,
		"breaking_message_enum",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 3, 3, 3, 6, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 0, 0, 0, 0, "ENUM_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 3, 3, 3, 7, "FIELD_SAME_TYPE"),
	)
}

func TestRunBreakingMessageInt(t *testing.T) {
	testBreaking(
		t,
		"breaking_message_int",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 3, 3, 3, 6, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 3, 3, 3, 8, "FIELD_SAME_TYPE"),
	)
}

func TestRunBreakingMessageMessage(t *testing.T) {
	testBreaking(
		t,
		"breaking_message_message",
		bufanalysistesting.NewFileAnnotation(t, "1.proto", 3, 3, 3, 8, "FIELD_SAME_TYPE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 0, 0, 0, 0, "MESSAGE_NO_DELETE"),
		bufanalysistesting.NewFileAnnotation(t, "2.proto", 3, 3, 3, 7, "FIELD_SAME_TYPE"),
	)
}

func testBreaking(
	t *testing.T,
	relDirPath string,
	expectedFileAnnotations ...bufanalysis.FileAnnotation,
) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	logger := zap.NewNop()

	previousDirPath := filepath.Join("testdata_previous", relDirPath)
	dirPath := filepath.Join("testdata", relDirPath)

	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	previousReadWriteBucket, err := storageosProvider.NewReadWriteBucket(
		previousDirPath,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	require.NoError(t, err)
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(
		dirPath,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	require.NoError(t, err)

	previousConfig := testGetConfig(t, previousReadWriteBucket)
	config := testGetConfig(t, readWriteBucket)

	previousModule, err := bufmodulebuild.BuildForBucket(
		context.Background(),
		previousReadWriteBucket,
		previousConfig.Build,
	)
	require.NoError(t, err)
	previousModuleFileSet, err := bufmodulebuild.NewModuleFileSetBuilder(
		zap.NewNop(),
		bufmodule.NewNopModuleReader(),
	).Build(
		context.Background(),
		previousModule,
	)
	require.NoError(t, err)
	previousImage, previousFileAnnotations, err := bufimagebuild.NewBuilder(zap.NewNop()).Build(
		ctx,
		previousModuleFileSet,
		bufimagebuild.WithExcludeSourceCodeInfo(),
	)
	require.NoError(t, err)
	require.Empty(t, previousFileAnnotations)
	previousImage = bufimage.ImageWithoutImports(previousImage)

	module, err := bufmodulebuild.BuildForBucket(
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

	handler := bufbreaking.NewHandler(logger)
	fileAnnotations, err = handler.Check(
		ctx,
		config.Breaking,
		previousImage,
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

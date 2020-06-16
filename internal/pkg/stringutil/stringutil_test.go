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

package stringutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToLowerSnakeCase(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", ToLowerSnakeCase(""))
	assert.Equal(t, "", ToLowerSnakeCase("  "))
	assert.Equal(t, "", ToLowerSnakeCase("_"))
	assert.Equal(t, "", ToLowerSnakeCase("__"))
	assert.Equal(t, "pascal_case", ToLowerSnakeCase("PascalCase"))
	assert.Equal(t, "pascal_case", ToLowerSnakeCase("  PascalCase"))
	assert.Equal(t, "pascal_case", ToLowerSnakeCase("PascalCase  "))
	assert.Equal(t, "pascal_case", ToLowerSnakeCase("pascalCase"))
	assert.Equal(t, "pascal_case", ToLowerSnakeCase("PascalCase_"))
	assert.Equal(t, "pascal_case", ToLowerSnakeCase("_PascalCase"))
	assert.Equal(t, "pascal_case_hello", ToLowerSnakeCase("PascalCase__Hello"))
	assert.Equal(t, "json_pascal", ToLowerSnakeCase("JSONPascal"))
	assert.Equal(t, "foo_json_pascal", ToLowerSnakeCase("FooJSONPascal"))
	assert.Equal(t, "json_pascal_json", ToLowerSnakeCase("JSONPascalJSON"))
	assert.Equal(t, "v1", ToLowerSnakeCase("v1"))
	assert.Equal(t, "v1beta1", ToLowerSnakeCase("v1beta1"))
	assert.Equal(t, "v1beta_1", ToLowerSnakeCase("v1beta_1"))
	assert.Equal(t, "v_1", ToLowerSnakeCase("v1", SnakeCaseWithNewWordOnDigits()))
	assert.Equal(t, "v_1beta_1", ToLowerSnakeCase("v1beta1", SnakeCaseWithNewWordOnDigits()))
	assert.Equal(t, "pascal_case1", ToLowerSnakeCase("PascalCase1"))
	assert.Equal(t, "pascal_case_1", ToLowerSnakeCase("PascalCase_1"))
	assert.Equal(t, "pascal_case_1", ToLowerSnakeCase("PascalCase1", SnakeCaseWithNewWordOnDigits()))
	assert.Equal(t, "foo_json1_pascal", ToLowerSnakeCase("FooJSON1Pascal"))
	assert.Equal(t, "foo_json_1_pascal", ToLowerSnakeCase("FooJSON_1Pascal"))
	assert.Equal(t, "foo_json_1_pascal", ToLowerSnakeCase("FooJSON1Pascal", SnakeCaseWithNewWordOnDigits()))
	assert.Equal(t, "pascal_case1", ToLowerSnakeCase("pascal_case1"))
	assert.Equal(t, "pascal_case_1", ToLowerSnakeCase("pascal_case_1"))
	assert.Equal(t, "pascal_case_1", ToLowerSnakeCase("pascal_case1", SnakeCaseWithNewWordOnDigits()))
	assert.Equal(t, "pascal_case_1", ToLowerSnakeCase("pascal_case_1", SnakeCaseWithNewWordOnDigits()))
	assert.Equal(t, "foo_bar_baz", ToLowerSnakeCase("foo_bar_baz"))
	assert.Equal(t, "foo_bar_baz", ToLowerSnakeCase("_foo_bar_baz_"))
	assert.Equal(t, "foo_bar_baz", ToLowerSnakeCase("foo_bar__baz"))
	assert.Equal(t, "pascal_case_hello", ToLowerSnakeCase("PascalCase--Hello"))
	assert.Equal(t, "foo_bar_baz", ToLowerSnakeCase("_foo-bar-baz_"))
	assert.Equal(t, "foo_bar_baz", ToLowerSnakeCase("  Foo  Bar  _Baz"))
	assert.Equal(t, "pascal_case", ToLowerSnakeCase("pascal_case"))
	assert.Equal(t, "pascal_case", ToLowerSnakeCase("pascalCase"))
	assert.Equal(t, "pascal_case", ToLowerSnakeCase("  pascal_case"))
	assert.Equal(t, "pascal_case", ToLowerSnakeCase("  pascal_case  "))
	assert.Equal(t, "pascal_case", ToLowerSnakeCase("pascal_case  "))
	assert.Equal(t, "pascal_case", ToLowerSnakeCase("Pascal_case"))
	assert.Equal(t, "pascal_case", ToLowerSnakeCase("__Pascal___case"))
	assert.Equal(t, "pascal_case", ToLowerSnakeCase("__Pascal___case__"))
	assert.Equal(t, "pascal_case", ToLowerSnakeCase("Pascal___case__"))
	assert.Equal(t, "pascal_case", ToLowerSnakeCase("Pascal-case"))
	assert.Equal(t, "pascal_case", ToLowerSnakeCase("Pascal case"))
	assert.Equal(t, "pascal_case", ToLowerSnakeCase("  Pascal case"))
	assert.Equal(t, "pascal_case", ToLowerSnakeCase("PASCAL_case"))
	assert.Equal(t, "pascal_case", ToLowerSnakeCase("__PASCAL___case"))
	assert.Equal(t, "pascal_case", ToLowerSnakeCase("__PASCAL___case__"))
	assert.Equal(t, "pascal_case", ToLowerSnakeCase("PASCAL___case__"))
	assert.Equal(t, "pascal_case", ToLowerSnakeCase("PASCAL-case"))
	assert.Equal(t, "pascal_case", ToLowerSnakeCase("PASCAL case"))
	assert.Equal(t, "pascal_case", ToLowerSnakeCase("  PASCAL case"))
}

func TestToUpperSnakeCase(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", ToUpperSnakeCase(""))
	assert.Equal(t, "PASCAL_CASE", ToUpperSnakeCase("PascalCase"))
	assert.Equal(t, "PASCAL_CASE", ToUpperSnakeCase("pascalCase"))
	assert.Equal(t, "PASCAL_CASE", ToUpperSnakeCase("PascalCase_"))
	assert.Equal(t, "PASCAL_CASE", ToUpperSnakeCase("_PascalCase"))
	assert.Equal(t, "PASCAL_CASE_HELLO", ToUpperSnakeCase("PascalCase__Hello"))
	assert.Equal(t, "JSON_PASCAL", ToUpperSnakeCase("JSONPascal"))
	assert.Equal(t, "FOO_JSON_PASCAL", ToUpperSnakeCase("FooJSONPascal"))
	assert.Equal(t, "PASCAL_CASE1", ToUpperSnakeCase("PascalCase1"))
	assert.Equal(t, "PASCAL_CASE_1", ToUpperSnakeCase("PascalCase_1"))
	assert.Equal(t, "PASCAL_CASE_1", ToUpperSnakeCase("PascalCase1", SnakeCaseWithNewWordOnDigits()))
	assert.Equal(t, "FOO_JSON1_PASCAL", ToUpperSnakeCase("FooJSON1Pascal"))
	assert.Equal(t, "FOO_JSON_1_PASCAL", ToUpperSnakeCase("FooJSON_1Pascal"))
	assert.Equal(t, "FOO_JSON_1_PASCAL", ToUpperSnakeCase("FooJSON1Pascal", SnakeCaseWithNewWordOnDigits()))
	assert.Equal(t, "PASCAL_CASE1", ToUpperSnakeCase("pascal_case1"))
	assert.Equal(t, "PASCAL_CASE_1", ToUpperSnakeCase("pascal_case_1"))
	assert.Equal(t, "PASCAL_CASE_1", ToUpperSnakeCase("pascal_case1", SnakeCaseWithNewWordOnDigits()))
	assert.Equal(t, "PASCAL_CASE_1", ToUpperSnakeCase("pascal_case_1", SnakeCaseWithNewWordOnDigits()))
	assert.Equal(t, "PASCAL_CASE1", ToUpperSnakeCase("PASCAL_CASE1"))
	assert.Equal(t, "PASCAL_CASE_1", ToUpperSnakeCase("PASCAL_CASE_1"))
	assert.Equal(t, "PASCAL_CASE_1", ToUpperSnakeCase("PASCAL_CASE1", SnakeCaseWithNewWordOnDigits()))
	assert.Equal(t, "PASCAL_CASE_1", ToUpperSnakeCase("PASCAL_CASE_1", SnakeCaseWithNewWordOnDigits()))
	assert.Equal(t, "V1", ToUpperSnakeCase("v1"))
	assert.Equal(t, "V1BETA1", ToUpperSnakeCase("v1beta1"))
	assert.Equal(t, "V_1", ToUpperSnakeCase("v1", SnakeCaseWithNewWordOnDigits()))
	assert.Equal(t, "V_1BETA_1", ToUpperSnakeCase("v1beta1", SnakeCaseWithNewWordOnDigits()))
	assert.Equal(t, "FOO_BAR_BAZ", ToUpperSnakeCase("foo_bar_baz"))
	assert.Equal(t, "FOO_BAR_BAZ", ToUpperSnakeCase("_foo_bar_baz_"))
	assert.Equal(t, "FOO_BAR_BAZ", ToUpperSnakeCase("foo_bar__baz"))
	assert.Equal(t, "PASCAL_CASE_HELLO", ToUpperSnakeCase("PascalCase--Hello"))
	assert.Equal(t, "FOO_BAR_BAZ", ToUpperSnakeCase("_foo-bar-baz_"))
	assert.Equal(t, "FOO_BAR_BAZ", ToUpperSnakeCase("  Foo  Bar  _Baz"))
	assert.Equal(t, "PASCAL_CASE", ToUpperSnakeCase("pascal_case"))
	assert.Equal(t, "PASCAL_CASE", ToUpperSnakeCase("pascalCase"))
	assert.Equal(t, "PASCAL_CASE", ToUpperSnakeCase("  pascal_case"))
	assert.Equal(t, "PASCAL_CASE", ToUpperSnakeCase("  pascal_case  "))
	assert.Equal(t, "PASCAL_CASE", ToUpperSnakeCase("pascal_case  "))
	assert.Equal(t, "PASCAL_CASE", ToUpperSnakeCase("Pascal_case"))
	assert.Equal(t, "PASCAL_CASE", ToUpperSnakeCase("__Pascal___case"))
	assert.Equal(t, "PASCAL_CASE", ToUpperSnakeCase("__Pascal___case__"))
	assert.Equal(t, "PASCAL_CASE", ToUpperSnakeCase("Pascal___case__"))
	assert.Equal(t, "PASCAL_CASE", ToUpperSnakeCase("Pascal-case"))
	assert.Equal(t, "PASCAL_CASE", ToUpperSnakeCase("Pascal case"))
	assert.Equal(t, "PASCAL_CASE", ToUpperSnakeCase("  Pascal case"))
	assert.Equal(t, "PASCAL_CASE", ToUpperSnakeCase("PASCAL_case"))
	assert.Equal(t, "PASCAL_CASE", ToUpperSnakeCase("__PASCAL___case"))
	assert.Equal(t, "PASCAL_CASE", ToUpperSnakeCase("__PASCAL___case__"))
	assert.Equal(t, "PASCAL_CASE", ToUpperSnakeCase("PASCAL___case__"))
	assert.Equal(t, "PASCAL_CASE", ToUpperSnakeCase("PASCAL-case"))
	assert.Equal(t, "PASCAL_CASE", ToUpperSnakeCase("PASCAL case"))
	assert.Equal(t, "PASCAL_CASE", ToUpperSnakeCase("  PASCAL case"))
	assert.Equal(t, "PASCAL_CASE", ToUpperSnakeCase("PASCAL_case"))
}

func TestToPascalCase(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", ToPascalCase(""))
	assert.Equal(t, "", ToPascalCase("  "))
	assert.Equal(t, "PascalCase", ToPascalCase("pascal_case"))
	assert.Equal(t, "PascalCase", ToPascalCase("pascalCase"))
	assert.Equal(t, "PascalCase", ToPascalCase("  pascal_case"))
	assert.Equal(t, "PascalCase", ToPascalCase("  pascal_case  "))
	assert.Equal(t, "PascalCase", ToPascalCase("pascal_case  "))
	assert.Equal(t, "PascalCase", ToPascalCase("Pascal_case"))
	assert.Equal(t, "PascalCase", ToPascalCase("__Pascal___case"))
	assert.Equal(t, "PascalCase", ToPascalCase("__Pascal___case__"))
	assert.Equal(t, "PascalCase", ToPascalCase("Pascal___case__"))
	assert.Equal(t, "PascalCase", ToPascalCase("Pascal-case"))
	assert.Equal(t, "PascalCase", ToPascalCase("Pascal case"))
	assert.Equal(t, "PascalCase", ToPascalCase("  Pascal case"))
	assert.Equal(t, "PASCALCase", ToPascalCase("PASCAL_case"))
	assert.Equal(t, "PASCALCase", ToPascalCase("__PASCAL___case"))
	assert.Equal(t, "PASCALCase", ToPascalCase("__PASCAL___case__"))
	assert.Equal(t, "PASCALCase", ToPascalCase("PASCAL___case__"))
	assert.Equal(t, "PASCALCase", ToPascalCase("PASCAL-case"))
	assert.Equal(t, "PASCALCase", ToPascalCase("PASCAL case"))
	assert.Equal(t, "PASCALCase", ToPascalCase("  PASCAL case"))
	assert.Equal(t, "PascalCase", ToPascalCase("PascalCase"))
	assert.Equal(t, "PascalCase", ToPascalCase("pascalCase"))
	assert.Equal(t, "PascalCase", ToPascalCase("PascalCase_"))
	assert.Equal(t, "PascalCase", ToPascalCase("_PascalCase"))
	assert.Equal(t, "PascalCaseHello", ToPascalCase("PascalCase__Hello"))
	assert.Equal(t, "JSONPascal", ToPascalCase("JSONPascal"))
	assert.Equal(t, "FooJSONPascal", ToPascalCase("FooJSONPascal"))
	assert.Equal(t, "PascalCase1", ToPascalCase("PascalCase1"))
	assert.Equal(t, "FooJSON1Pascal", ToPascalCase("FooJSON1Pascal"))
	assert.Equal(t, "FooBarBaz", ToPascalCase("foo_bar_baz"))
	assert.Equal(t, "FooBarBaz", ToPascalCase("_foo_bar_baz_"))
	assert.Equal(t, "FooBarBaz", ToPascalCase("foo_bar__baz"))
	assert.Equal(t, "PascalCaseHello", ToPascalCase("PascalCase--Hello"))
	assert.Equal(t, "FooBarBaz", ToPascalCase("_foo-bar-baz_"))
	assert.Equal(t, "FooBarBaz", ToPascalCase("  Foo  Bar  _Baz"))
}

func TestJoinSliceQuoted(t *testing.T) {
	t.Parallel()
	assert.Equal(t, ``, JoinSliceQuoted(nil, ", "))
	assert.Equal(t, ``, JoinSliceQuoted([]string{}, ", "))
	assert.Equal(t, `"a"`, JoinSliceQuoted([]string{"a"}, ", "))
	assert.Equal(t, `"a", "b"`, JoinSliceQuoted([]string{"a", "b"}, ", "))
	assert.Equal(t, `"a", "b", "c"`, JoinSliceQuoted([]string{"a", "b", "c"}, ", "))
}

func TestSliceToUniqueSortedSlice(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{}, SliceToUniqueSortedSlice(nil))
	assert.Equal(t, []string{}, SliceToUniqueSortedSlice([]string{}))
	assert.Equal(t, []string{"Are", "bats", "cats"}, SliceToUniqueSortedSlice([]string{"bats", "Are", "cats"}))
	assert.Equal(t, []string{"Are", "are", "bats", "cats"}, SliceToUniqueSortedSlice([]string{"bats", "Are", "cats", "are"}))
	assert.Equal(t, []string{"Are", "Bats", "bats", "cats"}, SliceToUniqueSortedSlice([]string{"bats", "Are", "cats", "Are", "Bats"}))
	assert.Equal(t, []string{"", "Are", "Bats", "bats", "cats"}, SliceToUniqueSortedSlice([]string{"bats", "Are", "cats", "", "Are", "Bats", ""}))
	assert.Equal(t, []string{"", "  ", "Are", "Bats", "bats", "cats"}, SliceToUniqueSortedSlice([]string{"bats", "Are", "cats", "", "Are", "Bats", "", "  "}))
	assert.Equal(t, []string{""}, SliceToUniqueSortedSlice([]string{"", ""}))
	assert.Equal(t, []string{""}, SliceToUniqueSortedSlice([]string{""}))
}

func TestSliceToUniqueSortedSliceFilterEmptyStrings(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{}, SliceToUniqueSortedSliceFilterEmptyStrings(nil))
	assert.Equal(t, []string{}, SliceToUniqueSortedSliceFilterEmptyStrings([]string{}))
	assert.Equal(t, []string{"Are", "bats", "cats"}, SliceToUniqueSortedSliceFilterEmptyStrings([]string{"bats", "Are", "cats"}))
	assert.Equal(t, []string{"Are", "are", "bats", "cats"}, SliceToUniqueSortedSliceFilterEmptyStrings([]string{"bats", "Are", "cats", "are"}))
	assert.Equal(t, []string{"Are", "Bats", "bats", "cats"}, SliceToUniqueSortedSliceFilterEmptyStrings([]string{"bats", "Are", "cats", "Are", "Bats"}))
	assert.Equal(t, []string{"Are", "Bats", "bats", "cats"}, SliceToUniqueSortedSliceFilterEmptyStrings([]string{"bats", "Are", "cats", "", "Are", "Bats", ""}))
	assert.Equal(t, []string{}, SliceToUniqueSortedSliceFilterEmptyStrings([]string{"", "", "  "}))
	assert.Equal(t, []string{}, SliceToUniqueSortedSliceFilterEmptyStrings([]string{""}))
}

func TestSliceToChunks(t *testing.T) {
	t.Parallel()
	testSliceToChunks(
		t,
		[]string{"are"},
		1,
		[]string{"are"},
	)
	testSliceToChunks(
		t,
		[]string{"are", "bats", "cats", "do", "eagle"},
		1,
		[]string{"are"},
		[]string{"bats"},
		[]string{"cats"},
		[]string{"do"},
		[]string{"eagle"},
	)
	testSliceToChunks(
		t,
		[]string{"are", "bats", "cats", "do", "eagle"},
		2,
		[]string{"are", "bats"},
		[]string{"cats", "do"},
		[]string{"eagle"},
	)
	testSliceToChunks(
		t,
		[]string{"are", "bats", "cats", "do", "eagle"},
		3,
		[]string{"are", "bats", "cats"},
		[]string{"do", "eagle"},
	)
	testSliceToChunks(
		t,
		[]string{"are", "bats", "cats", "do", "eagle"},
		6,
		[]string{"are", "bats", "cats", "do", "eagle"},
	)
	testSliceToChunks(
		t,
		nil,
		0,
	)
}

func testSliceToChunks(t *testing.T, input []string, chunkSize int, expected ...[]string) {
	assert.Equal(t, expected, SliceToChunks(input, chunkSize))
}

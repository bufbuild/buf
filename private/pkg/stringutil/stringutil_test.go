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

package stringutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.Equal(t, "FooBarBaz", ToPascalCase("  Foo.Bar.Baz"))
	assert.Equal(t, "FooBarBaz", ToPascalCase("foo_bar.baz"))
}

func TestJoinSliceQuoted(t *testing.T) {
	t.Parallel()
	assert.Equal(t, ``, JoinSliceQuoted(nil, ", "))
	assert.Equal(t, ``, JoinSliceQuoted([]string{}, ", "))
	assert.Equal(t, `"a"`, JoinSliceQuoted([]string{"a"}, ", "))
	assert.Equal(t, `"a", "b"`, JoinSliceQuoted([]string{"a", "b"}, ", "))
	assert.Equal(t, `"a", "b", "c"`, JoinSliceQuoted([]string{"a", "b", "c"}, ", "))
}

func TestSliceToHumanString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, ``, SliceToHumanString(nil))
	assert.Equal(t, ``, SliceToHumanString([]string{}))
	assert.Equal(t, `a`, SliceToHumanString([]string{"a"}))
	assert.Equal(t, `a and b`, SliceToHumanString([]string{"a", "b"}))
	assert.Equal(t, `a, b, and c`, SliceToHumanString([]string{"a", "b", "c"}))
}

func TestSliceToHumanStringQuoted(t *testing.T) {
	t.Parallel()
	assert.Equal(t, ``, SliceToHumanStringQuoted(nil))
	assert.Equal(t, ``, SliceToHumanStringQuoted([]string{}))
	assert.Equal(t, `"a"`, SliceToHumanStringQuoted([]string{"a"}))
	assert.Equal(t, `"a" and "b"`, SliceToHumanStringQuoted([]string{"a", "b"}))
	assert.Equal(t, `"a", "b", and "c"`, SliceToHumanStringQuoted([]string{"a", "b", "c"}))
}

func TestSliceToHumanStringOr(t *testing.T) {
	t.Parallel()
	assert.Equal(t, ``, SliceToHumanStringOr(nil))
	assert.Equal(t, ``, SliceToHumanStringOr([]string{}))
	assert.Equal(t, `a`, SliceToHumanStringOr([]string{"a"}))
	assert.Equal(t, `a or b`, SliceToHumanStringOr([]string{"a", "b"}))
	assert.Equal(t, `a, b, or c`, SliceToHumanStringOr([]string{"a", "b", "c"}))
}

func TestSliceToHumanStringOrQuoted(t *testing.T) {
	t.Parallel()
	assert.Equal(t, ``, SliceToHumanStringOrQuoted(nil))
	assert.Equal(t, ``, SliceToHumanStringOrQuoted([]string{}))
	assert.Equal(t, `"a"`, SliceToHumanStringOrQuoted([]string{"a"}))
	assert.Equal(t, `"a" or "b"`, SliceToHumanStringOrQuoted([]string{"a", "b"}))
	assert.Equal(t, `"a", "b", or "c"`, SliceToHumanStringOrQuoted([]string{"a", "b", "c"}))
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

func TestAlphanumeric(t *testing.T) {
	t.Parallel()
	assert.True(t, IsAlphanumeric('0'))
	assert.False(t, IsAlphanumeric('.'))
}

func TestIsAlpha(t *testing.T) {
	t.Parallel()
	assert.True(t, IsAlpha('r'))
	assert.True(t, IsAlpha('A'))
	assert.True(t, IsAlpha('Z'))
	assert.True(t, IsAlpha('a'))
	assert.True(t, IsAlpha('z'))
	assert.False(t, IsAlpha('.'))
	assert.False(t, IsAlpha('0'))
	assert.False(t, IsAlpha('9'))
	assert.False(t, IsAlpha('!'))
}

func TestIsLowerAlpha(t *testing.T) {
	t.Parallel()
	assert.True(t, IsLowerAlpha('r'))
	assert.False(t, IsLowerAlpha('R'))
}

func TestIsUpperAlpha(t *testing.T) {
	t.Parallel()
	assert.True(t, IsUpperAlpha('R'))
	assert.False(t, IsUpperAlpha('r'))
}

func TestIsNumeric(t *testing.T) {
	t.Parallel()
	assert.True(t, IsNumeric('0'))
	assert.False(t, IsNumeric('r'))
}

func TestIsLowerAlphanumeric(t *testing.T) {
	t.Parallel()
	assert.True(t, IsLowerAlphanumeric('0'))
	assert.True(t, IsLowerAlphanumeric('r'))
	assert.True(t, IsLowerAlphanumeric('a'))
	assert.True(t, IsLowerAlphanumeric('z'))
	assert.True(t, IsLowerAlphanumeric('9'))
	assert.False(t, IsLowerAlphanumeric('R'))
	assert.False(t, IsLowerAlphanumeric('A'))
	assert.False(t, IsLowerAlphanumeric('Z'))
	assert.False(t, IsLowerAlphanumeric('!'))
}

func TestIsAlphanumeric(t *testing.T) {
	t.Parallel()
	require.True(t, IsAlphanumeric('A'))
	require.True(t, IsAlphanumeric('Z'))
	require.True(t, IsAlphanumeric('a'))
	require.True(t, IsAlphanumeric('z'))
	require.True(t, IsAlphanumeric('0'))
	require.True(t, IsAlphanumeric('9'))
	require.False(t, IsAlphanumeric('!'))
}

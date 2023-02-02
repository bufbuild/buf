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

package internal

import (
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAndCheckPathsRelSuccess1(t *testing.T) {
	// https://github.com/bufbuild/buf/issues/56
	t.Parallel()
	testNormalizeAndCheckPathsRelSuccess(
		t,
		[]string{
			"proto",
			"proto-vendor",
		},
	)
}

func TestNormalizeAndCheckPathsRelError1(t *testing.T) {
	t.Parallel()
	testNormalizeAndCheckPathsRelError(
		t,
		[]string{
			"/a/b",
		},
	)
}

func TestNormalizeAndCheckPathsRelError2(t *testing.T) {
	t.Parallel()
	testNormalizeAndCheckPathsRelError(
		t,
		[]string{
			"a/b",
			"a/b",
		},
	)
}

func TestNormalizeAndCheckPathsRelError3(t *testing.T) {
	t.Parallel()
	testNormalizeAndCheckPathsRelError(
		t,
		[]string{
			"a/b",
			"a/b/c",
		},
	)
}

func TestNormalizeAndCheckPathsRelError4(t *testing.T) {
	t.Parallel()
	testNormalizeAndCheckPathsRelError(
		t,
		[]string{
			".",
			"a",
		},
	)
}

func TestNormalizeAndCheckPathsRelError5(t *testing.T) {
	t.Parallel()
	testNormalizeAndCheckPathsRelError(
		t,
		[]string{
			"",
		},
	)
}

func TestNormalizeAndCheckPathsRelError6(t *testing.T) {
	t.Parallel()
	testNormalizeAndCheckPathsRelError(
		t,
		[]string{
			"a/b",
			"",
		},
	)
}

func TestNormalizeAndCheckPathsRelEqual1(t *testing.T) {
	t.Parallel()
	testNormalizeAndCheckPathsRelEqual(
		t,
		[]string{
			"b",
			"a/../a",
		},
		[]string{
			"a",
			"b",
		},
	)
}

func TestNormalizeAndCheckPathsAbsSuccess1(t *testing.T) {
	// https://github.com/bufbuild/buf/issues/56
	t.Parallel()
	testNormalizeAndCheckPathsAbsSuccess(
		t,
		[]string{
			"proto",
			"proto-vendor",
		},
	)
}

func TestNormalizeAndCheckPathsAbsError1(t *testing.T) {
	t.Parallel()
	testNormalizeAndCheckPathsAbsError(
		t,
		[]string{
			"a/b",
			"a/b",
		},
	)
}

func TestNormalizeAndCheckPathsAbsError2(t *testing.T) {
	t.Parallel()
	testNormalizeAndCheckPathsAbsError(
		t,
		[]string{
			"a/b",
			"a/b/c",
		},
	)
}

func TestNormalizeAndCheckPathsAbsError3(t *testing.T) {
	t.Parallel()
	testNormalizeAndCheckPathsAbsError(
		t,
		[]string{
			".",
			"a",
		},
	)
}

func TestNormalizeAndCheckPathsAbsError4(t *testing.T) {
	t.Parallel()
	testNormalizeAndCheckPathsAbsError(
		t,
		[]string{
			"",
		},
	)
}

func TestNormalizeAndCheckPathsAbsError5(t *testing.T) {
	t.Parallel()
	testNormalizeAndCheckPathsAbsError(
		t,
		[]string{
			"a/b",
			"",
		},
	)
}

func TestNormalizeAndCheckPathsAbsEqual1(t *testing.T) {
	t.Parallel()
	absA, err := filepath.Abs("a")
	require.NoError(t, err)
	absA = filepath.ToSlash(absA)
	absB, err := filepath.Abs("b")
	require.NoError(t, err)
	absB = filepath.ToSlash(absB)
	testNormalizeAndCheckPathsAbsEqual(
		t,
		[]string{
			"b",
			"a/../a",
		},
		[]string{
			absA,
			absB,
		},
	)
}

func TestNormalizeAndCheckPathsAbsSuccessAbs1(t *testing.T) {
	// https://github.com/bufbuild/buf/issues/56
	t.Parallel()
	testNormalizeAndCheckPathsAbsSuccess(
		t,
		[]string{
			"/proto",
			"/proto-vendor",
		},
	)
}

func TestNormalizeAndCheckPathsAbsErrorAbs1(t *testing.T) {
	t.Parallel()
	testNormalizeAndCheckPathsAbsError(
		t,
		[]string{
			"/a/b",
			"/a/b",
		},
	)
}

func TestNormalizeAndCheckPathsAbsErrorAbs2(t *testing.T) {
	t.Parallel()
	testNormalizeAndCheckPathsAbsError(
		t,
		[]string{
			"/a/b",
			"/a/b/c",
		},
	)
}

func TestNormalizeAndCheckPathsAbsErrorAbs3(t *testing.T) {
	t.Parallel()
	testNormalizeAndCheckPathsAbsError(
		t,
		[]string{
			"/",
			"/a",
		},
	)
}

func TestNormalizeAndCheckPathsAbsEqualAbs1(t *testing.T) {
	t.Parallel()
	absA, err := filepath.Abs("/a")
	require.NoError(t, err)
	absA = filepath.ToSlash(absA)
	absB, err := filepath.Abs("/b")
	require.NoError(t, err)
	absB = filepath.ToSlash(absB)
	testNormalizeAndCheckPathsAbsEqual(
		t,
		[]string{
			"/b",
			"/a/../a",
		},
		[]string{
			absA,
			absB,
		},
	)
}

func testNormalizeAndCheckPathsRelSuccess(t *testing.T, paths []string) {
	_, err := NormalizeAndCheckPaths(paths, "test", normalpath.Relative, true)
	assert.NoError(t, err, paths)
}

func testNormalizeAndCheckPathsRelError(t *testing.T, paths []string) {
	_, err := NormalizeAndCheckPaths(paths, "test", normalpath.Relative, true)
	assert.Error(t, err, paths)
}

func testNormalizeAndCheckPathsRelEqual(
	t *testing.T,
	paths []string,
	expected []string,
) {
	actual, err := NormalizeAndCheckPaths(paths, "test", normalpath.Relative, true)
	assert.NoError(t, err, paths)
	assert.Equal(t, expected, actual)
}

func testNormalizeAndCheckPathsAbsSuccess(t *testing.T, paths []string) {
	_, err := NormalizeAndCheckPaths(paths, "test", normalpath.Absolute, true)
	assert.NoError(t, err, paths)
}

func testNormalizeAndCheckPathsAbsError(t *testing.T, paths []string) {
	_, err := NormalizeAndCheckPaths(paths, "test", normalpath.Absolute, true)
	assert.Error(t, err, paths)
}

func testNormalizeAndCheckPathsAbsEqual(
	t *testing.T,
	paths []string,
	expected []string,
) {
	actual, err := NormalizeAndCheckPaths(paths, "test", normalpath.Absolute, true)
	assert.NoError(t, err, paths)
	assert.Equal(t, expected, actual)
}

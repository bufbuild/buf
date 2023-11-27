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

package bufconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

func testNormalizeAndCheckPathsRelSuccess(t *testing.T, paths []string) {
	_, err := normalizeAndCheckPaths(paths, "test")
	assert.NoError(t, err, paths)
}

func testNormalizeAndCheckPathsRelError(t *testing.T, paths []string) {
	_, err := normalizeAndCheckPaths(paths, "test")
	assert.Error(t, err, paths)
}

func testNormalizeAndCheckPathsRelEqual(
	t *testing.T,
	paths []string,
	expected []string,
) {
	actual, err := normalizeAndCheckPaths(paths, "test")
	assert.NoError(t, err, paths)
	assert.Equal(t, expected, actual)
}

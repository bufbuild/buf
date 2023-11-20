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

//go:build !windows
// +build !windows

package normalpath

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/bufbuild/buf/private/pkg/slicesextended"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeAndValidate(t *testing.T) {
	t.Parallel()
	path, err := NormalizeAndValidate("")
	assert.NoError(t, err)
	assert.Equal(t, ".", path)
	path, err = NormalizeAndValidate(".")
	assert.NoError(t, err)
	assert.Equal(t, ".", path)
	path, err = NormalizeAndValidate("./.")
	assert.NoError(t, err)
	assert.Equal(t, ".", path)
	path, err = NormalizeAndValidate("./foo")
	assert.NoError(t, err)
	assert.Equal(t, "foo", path)

	_, err = NormalizeAndValidate("/foo")
	assert.Error(t, err)

	_, err = NormalizeAndValidate("../foo")
	assert.Error(t, err)
}

func TestNormalize(t *testing.T) {
	t.Parallel()
	assert.Equal(t, ".", Normalize(""))
	assert.Equal(t, ".", Normalize("."))
	assert.Equal(t, ".", Normalize("./."))
	assert.Equal(t, "foo", Normalize("./foo"))
	assert.Equal(t, "../foo", Normalize("../foo"))
	assert.Equal(t, "../foo", Normalize("../foo"))
	assert.Equal(t, "foo", Normalize("foo/"))
	assert.Equal(t, "foo", Normalize("./foo/"))
	assert.Equal(t, "/foo", Normalize("/foo"))
	assert.Equal(t, "/foo", Normalize("/foo/"))
	assert.Equal(t, "/foo/bar", Normalize("/foo/../foo/bar"))
}

func TestUnnormalize(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", Unnormalize(""))
	assert.Equal(t, ".", Unnormalize("."))
	assert.Equal(t, "/foo", Unnormalize("/foo"))
}

func TestBase(t *testing.T) {
	t.Parallel()
	testBase(t, ".", ".")
	testBase(t, ".", ".")
	testBase(t, ".", "./.")
	testBase(t, "foo", "./foo")
	testBase(t, "bar", "./foo/bar")
	testBase(t, "bar", "../foo/bar")
	testBase(t, "foo", "/foo")
	testBase(t, "bar", "/foo/bar")
}

func testBase(t *testing.T, expected string, input string) {
	if os.PathSeparator == '/' {
		assert.Equal(t, expected, filepath.Base(input))
	}
	assert.Equal(t, expected, Base(input))
}

func TestDir(t *testing.T) {
	t.Parallel()
	testDir(t, ".", "")
	testDir(t, ".", ".")
	testDir(t, "/", "/")
	testDir(t, ".", "./")
	testDir(t, ".", "./.")
	testDir(t, ".", "foo")
	testDir(t, ".", "./foo")
	testDir(t, "foo", "./foo/bar")
	testDir(t, "../foo", "../foo/bar")
	testDir(t, "../foo", "../foo/bar/../..")
	testDir(t, "/foo", "/foo/bar")
	testDir(t, "/", "/foo")
}

func testDir(t *testing.T, expected string, input string) {
	assert.Equal(t, expected, Dir(input))
}

func TestExt(t *testing.T) {
	t.Parallel()
	testExt(t, "", "")
	testExt(t, ".", ".")
	testExt(t, ".txt", ".txt")
	testExt(t, ".txt", ".js.txt")
	testExt(t, "", "foo")
	testExt(t, ".txt", "foo.txt")
	testExt(t, ".txt", "foo.js.txt")
	testExt(t, "", "./foo")
	testExt(t, ".txt", "./foo.txt")
	testExt(t, ".txt", "./foo.js.txt")
	testExt(t, "", "./foo/bar")
	testExt(t, ".txt", "./foo/bar.txt")
	testExt(t, ".txt", "./foo/bar.txt")
	testExt(t, ".txt", "./foo/bar.js.txt")
	testExt(t, "", "../foo/bar")
	testExt(t, ".txt", "../foo/bar.txt")
	testExt(t, ".txt", "../foo/bar.js.txt")
}

func testExt(t *testing.T, expected string, input string) {
	if os.PathSeparator == '/' {
		assert.Equal(t, expected, filepath.Ext(input))
	}
	assert.Equal(t, expected, Ext(input))
}

func TestJoin(t *testing.T) {
	t.Parallel()
	testJoin(t, "", "")
	testJoin(t, "", "", "")
	testJoin(t, ".", ".", ".")
	testJoin(t, ".", "", ".", "")
	testJoin(t, "foo/bar", "foo", "./bar")
	testJoin(t, "foo", "foo", "./bar", "..")
	testJoin(t, "/foo/bar", "/foo", "./bar")
	testJoin(t, "/foo", "/foo", "./bar", "..")
	testJoin(t, "bar", ".", "bar")
}

func testJoin(t *testing.T, expected string, input ...string) {
	if os.PathSeparator == '/' {
		assert.Equal(t, expected, filepath.Join(input...))
	}
	assert.Equal(t, expected, Join(input...))
}

func TestRel(t *testing.T) {
	t.Parallel()
	testRel(t, ".", "", "")
	testRel(t, ".", "", ".")
	testRel(t, ".", ".", "")
	testRel(t, ".", ".", ".")
	testRel(t, ".", "foo", "foo")
	testRel(t, "foo", ".", "foo")
	testRel(t, "foo", ".", "./foo")
	testRel(t, "foo/bar", ".", "foo/bar")
	testRel(t, "bar", "/foo", "/foo/bar")
	testRel(t, "bar", "foo", "foo/bar")
	testRel(t, "baz", "/foo/./bar", "/foo/bar/baz")
	testRel(t, "baz", "foo/./bar", "foo/bar/baz")
	testRelError(t, "", "..", "foo/bar")
}

func testRel(t *testing.T, expected string, basepath string, targpath string) {
	if os.PathSeparator == '/' {
		rel, err := filepath.Rel(basepath, targpath)
		assert.NoError(t, err)
		assert.Equal(t, expected, rel)
	}
	rel, err := Rel(basepath, targpath)
	assert.NoError(t, err)
	assert.Equal(t, expected, rel)
}

func testRelError(t *testing.T, expected string, basepath string, targpath string) {
	if os.PathSeparator == '/' {
		rel, err := filepath.Rel(basepath, targpath)
		assert.Error(t, err)
		assert.Equal(t, expected, rel)
	}
	rel, err := Rel(basepath, targpath)
	assert.Error(t, err)
	assert.Equal(t, expected, rel)
}

func TestComponents(t *testing.T) {
	t.Parallel()
	testComponents(t, "", ".")
	testComponents(t, ".", ".")
	testComponents(t, "foo", "foo")
	testComponents(t, "foo/bar", "foo", "bar")
	testComponents(t, "foo/bar/../baz", "foo", "bar", "..", "baz")
	testComponents(t, "/foo/bar", "/", "foo", "bar")
	testComponents(t, "./foo/bar", ".", "foo", "bar")
	testComponents(t, "../foo/bar", "..", "foo", "bar")
}

func testComponents(t *testing.T, path string, expected ...string) {
	assert.Equal(t, expected, Components(path))
}

func TestStripComponents(t *testing.T) {
	t.Parallel()
	testStripComponents(t, 0, "", true, "")
	testStripComponents(t, 0, "foo", true, "foo")
	testStripComponents(t, 0, "foo", true, "foo")
	testStripComponents(t, 1, "", false, "foo")
	testStripComponents(t, 1, "bar", true, "foo/bar")
	testStripComponents(t, 1, "bar/baz", true, "foo/bar/baz")
	testStripComponents(t, 2, "baz", true, "foo/bar/baz")
	testStripComponents(t, 1, "bar/baz/bat", true, "foo/bar/baz/bat")
	testStripComponents(t, 2, "baz/bat", true, "foo/bar/baz/bat")
	testStripComponents(t, 3, "bat", true, "foo/bar/baz/bat")
	testStripComponents(t, 4, "", false, "foo/bar/baz/bat")
	testStripComponents(t, 5, "", false, "foo/bar/baz/bat")
}

func testStripComponents(t *testing.T, count int, expected string, expectedOK bool, path string) {
	actual, ok := StripComponents(path, uint32(count))
	assert.Equal(t, expectedOK, ok)
	assert.Equal(t, expected, actual)
}

func TestByDir(t *testing.T) {
	t.Parallel()
	assert.Equal(
		t,
		map[string][]string{
			"one": {
				"one/1.txt",
				"one/2.txt",
				"one/3.txt",
			},
			"two": {
				"two/1.txt",
				"two/2.txt",
				"two/3.txt",
			},
		},
		ByDir(
			"one/2.txt",
			"one/1.txt",
			"two/2.txt",
			"one/3.txt",
			"two/1.txt",
			"two/3.txt",
		),
	)
	assert.Equal(
		t,
		map[string][]string{
			".": {
				"1.txt",
				"2.txt",
				"3.txt",
			},
			"two": {
				"two/1.txt",
				"two/2.txt",
				"two/3.txt",
			},
		},
		ByDir(
			"2.txt",
			"1.txt",
			"3.txt",
			"two/3.txt",
			"two/2.txt",
			"two/1.txt",
		),
	)
}

func TestContainsPath(t *testing.T) {
	t.Parallel()
	testContainsPath(t, false, "a.proto", "a.proto")
	testContainsPath(t, true, ".", "a.proto")
	testContainsPath(t, false, "a.proto", ".")
	testContainsPath(t, false, ".", ".")
	testContainsPath(t, true, ".", "a/b.proto")
	testContainsPath(t, true, ".", "a/b")
	testContainsPath(t, false, "a", "ab/c")
	testContainsPath(t, true, "a", "a/b/c")
	testContainsPath(t, false, "b", "a/b/c")
	testContainsPath(t, true, "b", "b/b/c")
	testContainsPath(t, true, "b", "b/a/c")
}

func testContainsPath(t *testing.T, expected bool, value string, path string) {
	assert.Equal(t, expected, ContainsPath(value, path, Relative), fmt.Sprintf("%s %s", value, path))
}

func TestEqualsOrContainsPath(t *testing.T) {
	t.Parallel()
	testEqualsOrContainsPath(t, true, "a.proto", "a.proto")
	testEqualsOrContainsPath(t, true, ".", "a.proto")
	testEqualsOrContainsPath(t, false, "a.proto", ".")
	testEqualsOrContainsPath(t, true, ".", "a/b.proto")
	testEqualsOrContainsPath(t, true, ".", "a/b")
	testEqualsOrContainsPath(t, false, "a", "ab/c")
	testEqualsOrContainsPath(t, true, "a", "a/b/c")
	testEqualsOrContainsPath(t, false, "b", "a/b/c")
	testEqualsOrContainsPath(t, true, "b", "b/b/c")
	testEqualsOrContainsPath(t, true, "b", "b/a/c")
}

func testEqualsOrContainsPath(t *testing.T, expected bool, value string, path string) {
	assert.Equal(t, expected, EqualsOrContainsPath(value, path, Relative), fmt.Sprintf("%s %s", value, path))
}

func TestMapHasEqualOrContainingPath(t *testing.T) {
	t.Parallel()
	testMapHasEqualOrContainingPath(t, true, "a.proto", "a.proto")
	testMapHasEqualOrContainingPath(t, false, ".", "a.proto")
	testMapHasEqualOrContainingPath(t, true, "a.proto", ".")
	testMapHasEqualOrContainingPath(t, true, "a/b.proto", ".")
	testMapHasEqualOrContainingPath(t, true, "a/b", ".")
	testMapHasEqualOrContainingPath(t, false, "ab/c", "a", "b")
	testMapHasEqualOrContainingPath(t, true, "a/b/c", "a", "b")
	testMapHasEqualOrContainingPath(t, false, "a/b/c", "b")
	testMapHasEqualOrContainingPath(t, true, "b/b/c", "b")
	testMapHasEqualOrContainingPath(t, true, "b/a/c", "b")
	testMapHasEqualOrContainingPath(t, true, "b/b/c", "b", ".")
}

func testMapHasEqualOrContainingPath(t *testing.T, expected bool, path string, keys ...string) {
	keyMap := slicesextended.ToMap(keys)
	assert.Equal(t, expected, MapHasEqualOrContainingPath(keyMap, path, Relative), fmt.Sprintf("%s %v", path, keys))
}

func TestMapAllEqualOrContainingPaths(t *testing.T) {
	t.Parallel()
	testMapAllEqualOrContainingPaths(t, []string{"a.proto"}, "a.proto", "a.proto")
	testMapAllEqualOrContainingPaths(t, nil, ".", "a.proto")
	testMapAllEqualOrContainingPaths(t, []string{"."}, "a.proto", ".")
	testMapAllEqualOrContainingPaths(t, []string{"."}, "a/b.proto", ".")
	testMapAllEqualOrContainingPaths(t, []string{"."}, "a/b", ".")
	testMapAllEqualOrContainingPaths(t, nil, "ab/c", "a", "b")
	testMapAllEqualOrContainingPaths(t, []string{"a"}, "a/b/c", "a", "b")
	testMapAllEqualOrContainingPaths(t, nil, "a/b/c", "b")
	testMapAllEqualOrContainingPaths(t, []string{"b"}, "b/b/c", "b")
	testMapAllEqualOrContainingPaths(t, []string{"b"}, "b/a/c", "b")
	testMapAllEqualOrContainingPaths(t, []string{"b", "."}, "b/b/c", "b", ".")
	testMapAllEqualOrContainingPaths(t, []string{"b", "b/b", "."}, "b/b/c", "b", "b/b", ".")
}

func testMapAllEqualOrContainingPaths(t *testing.T, expected []string, path string, keys ...string) {
	if expected == nil {
		expected = make([]string, 0)
	}
	sort.Strings(expected)
	keyMap := slicesextended.ToMap(keys)
	assert.Equal(t, expected, MapAllEqualOrContainingPaths(keyMap, path, Relative), fmt.Sprintf("%s %v", path, keys))
}

func TestContainsPathAbs(t *testing.T) {
	t.Parallel()
	testContainsPathAbs(t, false, "/a.proto", "/a.proto")
	testContainsPathAbs(t, true, "/", "/a.proto")
	testContainsPathAbs(t, false, "/a.proto", "/")
	testContainsPathAbs(t, false, "/", "/")
	testContainsPathAbs(t, true, "/", "/a/b.proto")
	testContainsPathAbs(t, true, "/", "/a/b")
	testContainsPathAbs(t, false, "/a", "/ab/c")
	testContainsPathAbs(t, true, "/a", "/a/b/c")
	testContainsPathAbs(t, false, "/b", "/a/b/c")
	testContainsPathAbs(t, true, "/b", "/b/b/c")
	testContainsPathAbs(t, true, "/b", "/b/a/c")
}

func testContainsPathAbs(t *testing.T, expected bool, value string, path string) {
	assert.Equal(t, expected, ContainsPath(value, path, Absolute), fmt.Sprintf("%s %s", value, path))
}

func TestEqualsOrContainsPathAbs(t *testing.T) {
	t.Parallel()
	testEqualsOrContainsPathAbs(t, true, "/a.proto", "/a.proto")
	testEqualsOrContainsPathAbs(t, true, "/", "/a.proto")
	testEqualsOrContainsPathAbs(t, false, "a.proto", "/")
	testEqualsOrContainsPathAbs(t, true, "/", "/a/b.proto")
	testEqualsOrContainsPathAbs(t, true, "/", "/a/b")
	testEqualsOrContainsPathAbs(t, false, "/a", "/ab/c")
	testEqualsOrContainsPathAbs(t, true, "/a", "/a/b/c")
	testEqualsOrContainsPathAbs(t, false, "/b", "/a/b/c")
	testEqualsOrContainsPathAbs(t, true, "/b", "/b/b/c")
	testEqualsOrContainsPathAbs(t, true, "/b", "/b/a/c")
}

func testEqualsOrContainsPathAbs(t *testing.T, expected bool, value string, path string) {
	assert.Equal(t, expected, EqualsOrContainsPath(value, path, Absolute), fmt.Sprintf("%s %s", value, path))
}

func TestMapHasEqualOrContainingPathAbs(t *testing.T) {
	t.Parallel()
	testMapHasEqualOrContainingPathAbs(t, true, "/a.proto", "/a.proto")
	testMapHasEqualOrContainingPathAbs(t, false, "/", "/a.proto")
	testMapHasEqualOrContainingPathAbs(t, true, "/a.proto", "/")
	testMapHasEqualOrContainingPathAbs(t, true, "/a/b.proto", "/")
	testMapHasEqualOrContainingPathAbs(t, true, "/a/b", "/")
	testMapHasEqualOrContainingPathAbs(t, false, "/ab/c", "/a", "/b")
	testMapHasEqualOrContainingPathAbs(t, true, "/a/b/c", "/a", "/b")
	testMapHasEqualOrContainingPathAbs(t, false, "/a/b/c", "/b")
	testMapHasEqualOrContainingPathAbs(t, true, "/b/b/c", "/b")
	testMapHasEqualOrContainingPathAbs(t, true, "/b/a/c", "/b")
	testMapHasEqualOrContainingPathAbs(t, true, "/b/b/c", "/b", "/")
}

func testMapHasEqualOrContainingPathAbs(t *testing.T, expected bool, path string, keys ...string) {
	keyMap := slicesextended.ToMap(keys)
	assert.Equal(t, expected, MapHasEqualOrContainingPath(keyMap, path, Absolute), fmt.Sprintf("%s %v", path, keys))
}

func TestMapAllEqualOrContainingPathsAbs(t *testing.T) {
	t.Parallel()
	testMapAllEqualOrContainingPathsAbs(t, []string{"/a.proto"}, "/a.proto", "/a.proto")
	testMapAllEqualOrContainingPathsAbs(t, nil, "/", "/a.proto")
	testMapAllEqualOrContainingPathsAbs(t, []string{"/"}, "/a.proto", "/")
	testMapAllEqualOrContainingPathsAbs(t, []string{"/"}, "/a/b.proto", "/")
	testMapAllEqualOrContainingPathsAbs(t, []string{"/"}, "/a/b", "/")
	testMapAllEqualOrContainingPathsAbs(t, nil, "/ab/c", "/a", "/b")
	testMapAllEqualOrContainingPathsAbs(t, []string{"/a"}, "/a/b/c", "/a", "/b")
	testMapAllEqualOrContainingPathsAbs(t, nil, "/a/b/c", "/b")
	testMapAllEqualOrContainingPathsAbs(t, []string{"/b"}, "/b/b/c", "/b")
	testMapAllEqualOrContainingPathsAbs(t, []string{"/b"}, "/b/a/c", "/b")
	testMapAllEqualOrContainingPathsAbs(t, []string{"/b", "/"}, "/b/b/c", "/b", "/")
	testMapAllEqualOrContainingPathsAbs(t, []string{"/b", "/b/b", "/"}, "/b/b/c", "/b", "/b/b", "/")
}

func testMapAllEqualOrContainingPathsAbs(t *testing.T, expected []string, path string, keys ...string) {
	if expected == nil {
		expected = make([]string, 0)
	}
	sort.Strings(expected)
	keyMap := slicesextended.ToMap(keys)
	assert.Equal(t, expected, MapAllEqualOrContainingPaths(keyMap, path, Absolute), fmt.Sprintf("%s %v", path, keys))
}

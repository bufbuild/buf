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

//go:build windows
// +build windows

package normalpath

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/bufbuild/buf/private/pkg/slicesext"
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

	// Drive letter path
	_, err = NormalizeAndValidate("c:\\foo")
	assert.Error(t, err)

	// Network Drive UNC path
	_, err = NormalizeAndValidate("\\\\127.0.0.1\\$c\\")
	assert.Error(t, err)

	// Absolute path on current drive
	_, err = NormalizeAndValidate("\\root\\path")
	assert.Error(t, err)

	_, err = NormalizeAndValidate("../foo")
	assert.Error(t, err)
}

func TestNormalize(t *testing.T) {
	t.Parallel()
	assert.Equal(t, ".", Normalize(""))
	assert.Equal(t, ".", Normalize("."))
	assert.Equal(t, ".", Normalize(".\\."))
	assert.Equal(t, "foo", Normalize(".\\foo"))
	assert.Equal(t, "../foo", Normalize("..\\foo"))
	assert.Equal(t, "../Foo", Normalize("..\\Foo"))
	assert.Equal(t, "foo", Normalize("foo\\"))
	assert.Equal(t, "foo", Normalize(".\\foo\\"))
	assert.Equal(t, "c:/foo", Normalize("c:\\foo"))
	assert.Equal(t, "C:/foo", Normalize("C:\\foo\\"))
	assert.Equal(t, "c:/foo/bar", Normalize("c:\\foo\\..\\foo\\bar\\"))
	assert.Equal(t, "//127.0.0.1/$c/foo/bar", Normalize("\\\\127.0.0.1\\$c\\foo\\bar\\"))
	assert.Equal(t, "c:/", Normalize("c:\\"))
	assert.Equal(t, "//127.0.0.1/$c/", Normalize("\\\\127.0.0.1\\$c\\"))
}

func TestUnnormalize(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", Unnormalize(""))
	assert.Equal(t, ".", Unnormalize("."))
	assert.Equal(t, ".\\foo\\bar", Unnormalize("./foo/bar"))
	assert.Equal(t, "c:\\foo\\bar", Unnormalize("c:/foo/bar"))
	assert.Equal(t, "\\foo\\bar", Unnormalize("/foo/bar"))
	assert.Equal(t, "\\\\127.0.0.1\\$c\\foo\\bar", Unnormalize("//127.0.0.1/$c/foo/bar"))
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
	testBase(t, "baz", "c:/foo/bar/baz")
	testBase(t, "baz", "//127.0.0.1/$D/foo/bar/baz")

	// Base doesn't specify the input is normalized.
	testBase(t, "bar", "c:\\foo\\bar")
	testBase(t, "baz", "\\\\127.0.0.1\\$c\\foo\\bar\\baz")
}

func testBase(t *testing.T, expected string, input string) {
	assert.Equal(t, expected, Base(input))
}

func TestDir(t *testing.T) {
	t.Parallel()
	testDir(t, ".", "")
	testDir(t, ".", ".")
	testDir(t, "c:/", "c:\\")
	testDir(t, ".", ".\\")
	testDir(t, ".", ".\\.")
	testDir(t, ".", "foo")
	testDir(t, ".", ".\\foo")
	testDir(t, "foo", ".\\foo\\bar")
	testDir(t, "../foo", "..\\foo\\bar")
	testDir(t, "../foo", "..\\foo\\bar\\..\\..")
	testDir(t, "c:/foo", "c:\\foo\\bar")
	testDir(t, "//127.0.0.1/$c/", "\\\\127.0.0.1\\$c\\foo")
	testDir(t, "/foo", "\\foo\\bar")
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
	testExt(t, "", ".\\foo")
	testExt(t, ".txt", ".\\foo.txt")
	testExt(t, ".txt", ".\\foo.js.txt")
	testExt(t, "", ".\\foo\\bar")
	testExt(t, ".txt", ".\\foo\\bar.txt")
	testExt(t, ".txt", ".\\foo\\bar.txt")
	testExt(t, ".txt", ".\\foo\\bar.js.txt")
	testExt(t, "", "..\\foo\\bar")
	testExt(t, ".txt", "..\\foo\\bar.txt")
	testExt(t, ".txt", "..\\foo\\bar.js.txt")
	testExt(t, ".txt", "\\\\127.0.0.1\\$d\\foo.txt")
	testExt(t, ".txt", "c:\\foo.txt")
}

func testExt(t *testing.T, expected string, input string) {
	assert.Equal(t, expected, Ext(input))
}

func TestJoin(t *testing.T) {
	t.Parallel()
	testJoin(t, "", "")
	testJoin(t, "", "", "")
	testJoin(t, ".", ".", ".")
	testJoin(t, ".", "", ".", "")
	testJoin(t, "foo/bar", "foo", ".\\bar")
	testJoin(t, "foo", "foo", ".\\bar", "..")
	testJoin(t, "c:/foo/bar", "c:\\", "foo", ".\\bar")
	testJoin(t, "//127.0.0.1/$c/foo", "\\\\127.0.0.1\\$c\\", "foo", ".\\bar", "..")
	testJoin(t, "bar", ".", "bar")
	testJoin(t, "/foo/bar", "\\foo", ".", ".", ".", "baz", "..", "bar")
}

func testJoin(t *testing.T, expected string, input ...string) {
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
	testRel(t, "bar", "c:\\foo", "c:\\foo\\bar")
	testRel(t, "bar", "foo", "foo/bar")
	testRel(t, "baz", "c:\\foo/./bar", "c:\\foo\\bar\\baz")
	testRel(t, "baz", "foo/./bar", "foo/bar/baz")

	// This would require querying the current directory to know what
	// Drive `\` refers to.
	testRelError(t, "", "\\foo\\bar", "c:\\foo\\bar")

	testRelError(t, "", "..", "foo/bar")
	testRelError(t, "", "c:\\foo", "d:\\bar")
	testRelError(t, "", "\\\\127.0.0.1\\$c\\foo", "\\\\127.0.0.1\\$d\\foo")
}

func testRel(t *testing.T, expected string, basepath string, targpath string) {
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
	testComponents(t, "c:/foo/bar", "c:/", "foo", "bar")
	testComponents(t, "//127.0.0.1/$c/foo/bar", "//127.0.0.1/$c/", "foo", "bar")
	testComponents(t, "./foo/bar", ".", "foo", "bar")
	testComponents(t, "../foo/bar", "..", "foo", "bar")
	testComponents(t, "/foo/bar", "/", "foo", "bar")
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
	testStripComponents(t, 1, "bar/baz/BAT", true, "foo/bar/baz/BAT")
	testStripComponents(t, 2, "baz/BAT", true, "foo/bar/baz/BAT")
	testStripComponents(t, 3, "BAT", true, "foo/bar/baz/BAT")
	testStripComponents(t, 4, "", false, "foo/bar/baz/BAT")
	testStripComponents(t, 5, "", false, "foo/bar/baz/BAT")

	// Volume names contain separators but they are a single component
	testStripComponents(t, 1, "foo", true, "//127.0.0.1/$c/foo")
	testStripComponents(t, 2, "", false, "//127.0.0.1/$c/foo")
	testStripComponents(t, 1, "foo", true, "c:/foo")
	testStripComponents(t, 2, "", false, "c:/foo")
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
	keyMap := slicesext.ToStructMap(keys)
	assert.Equal(t, expected, MapHasEqualOrContainingPath(keyMap, path, Relative), fmt.Sprintf("%s %v", path, keys))
}

func TestMapAllEqualOrContainingPaths(t *testing.T) {
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
	keyMap := slicesext.ToStructMap(keys)
	assert.Equal(t, expected, MapAllEqualOrContainingPaths(keyMap, path, Relative), fmt.Sprintf("%s %v", path, keys))
}

func TestContainsPathAbs(t *testing.T) {
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

	testContainsPathAbs(t, true, "c:/", "c:/a.proto")
	testContainsPathAbs(t, false, "c:/", "c:/")
	testContainsPathAbs(t, false, "c:/", "d:/")
	// Can't be known without knowing current directory to detect drive
	testContainsPathAbs(t, false, "c:/", "/")

	testContainsPathAbs(t, true, "//127.0.0.1/$c/", "//127.0.0.1/$c/a.proto")
	testContainsPathAbs(t, false, "//127.0.0.1/$c/", "//127.0.0.1/$c/")
	testContainsPathAbs(t, false, "//127.0.0.1/$c/", "//127.0.0.1/$d/a.proto")
	// Can't be known without knowing current directory to detect drive
	testContainsPathAbs(t, false, "//127.0.0.1/$c/", "/")
}

func testContainsPathAbs(t *testing.T, expected bool, value string, path string) {
	assert.Equal(t, expected, ContainsPath(value, path, Absolute), fmt.Sprintf("%s %s", value, path))
}

func TestEqualsOrContainsPathAbs(t *testing.T) {
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

	testEqualsOrContainsPathAbs(t, true, "c:/", "c:/a.proto")
	testEqualsOrContainsPathAbs(t, true, "c:/", "c:/")
	testEqualsOrContainsPathAbs(t, false, "c:/", "d:/")
	// Can't be known without knowing current directory to detect drive
	testEqualsOrContainsPathAbs(t, false, "c:/", "/")

	testEqualsOrContainsPathAbs(t, true, "//127.0.0.1/$c/", "//127.0.0.1/$c/a.proto")
	testEqualsOrContainsPathAbs(t, true, "//127.0.0.1/$c/", "//127.0.0.1/$c/")
	testEqualsOrContainsPathAbs(t, false, "//127.0.0.1/$c/", "//127.0.0.1/$d/a.proto")
	// Can't be known without knowing current directory to detect drive
	testEqualsOrContainsPathAbs(t, false, "//127.0.0.1/$c/", "/")

	// Case Folding tests
	// c.f. https://www.unicode.org/versions/Unicode13.0.0/ch05.pdf#page=44
	// We are not trying to test all code point folding, only a few specific
	// cases. We defer to the golang implementation of case folding and retesting
	// that is not a goal.

	// \u212a is the kelvin symbol (K) which looks like a capital K
	// The capital K is the kelvin symbol U+212A
	testEqualsOrContainsPathAbs(t, true, "c:/k", "c:/\u212a")

	// In Turkish, a lower case i maps to a capital I with a dot (İ) U+0130
	// a lower case i with no dot ı U+0131 maps to a capital I
	// In all other languages a lower case i maps to a capital I
	// We explicitly do not support this special case (its 2 codepoints total) as
	// it is locale dependent.
	testEqualsOrContainsPathAbs(t, true, "c:/i", "c:/I")
	// TODO: The Go stdlib unicode tables seem to fold \u0131 to \u0131 which
	// doesn't seem spec compliant, but not important enough right now to warrant
	// further research, or further complicating this module.
	testEqualsOrContainsPathAbs(t, false, "c:/\u0131", "c:/I")
	testEqualsOrContainsPathAbs(t, false, "c:/i", "c:/\u0130")

}

func testEqualsOrContainsPathAbs(t *testing.T, expected bool, value string, path string) {
	assert.Equal(t, expected, EqualsOrContainsPath(value, path, Absolute), fmt.Sprintf("%s %s", value, path))
}

func TestMapHasEqualOrContainingPathAbs(t *testing.T) {
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

	testMapHasEqualOrContainingPathAbs(t, true, "c:/a.proto", "d:/", "c:/")
	testMapHasEqualOrContainingPathAbs(t, false, "c:/", "d:/", "c:/a.proto")
	// Can't be known without knowing current directory
	testMapHasEqualOrContainingPathAbs(t, false, "c:/", "/")

	testMapHasEqualOrContainingPathAbs(t, true, "//127.0.0.1/$c/a.proto", "d:/", "//127.0.0.1/$c/")
	testMapHasEqualOrContainingPathAbs(t, false, "//127.0.0.1/$c/", "d:/", "//127.0.0.1/$c/a.proto")
	// Can't be known without knowing current directory
	testMapHasEqualOrContainingPathAbs(t, false, "//127.0.0.1/$c/", "/")
}

func testMapHasEqualOrContainingPathAbs(t *testing.T, expected bool, path string, keys ...string) {
	keyMap := slicesext.ToStructMap(keys)
	assert.Equal(t, expected, MapHasEqualOrContainingPath(keyMap, path, Absolute), fmt.Sprintf("%s %v", path, keys))
}

func TestMapAllEqualOrContainingPathsAbs(t *testing.T) {
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

	testMapAllEqualOrContainingPathsAbs(t, []string{"c:/"}, "c:/a.proto", "d:/", "c:/", "/")
	testMapAllEqualOrContainingPathsAbs(t, nil, "c:/", "d:/", "c:/a.proto")

	testMapAllEqualOrContainingPathsAbs(t, []string{"//127.0.0.1/$c/"}, "//127.0.0.1/$c/a.proto", "d:/", "//127.0.0.1/$c/", "//127.0.0.1/$d/", "/")
	testMapAllEqualOrContainingPathsAbs(t, nil, "//127.0.0.1/$c/", "d:/", "//127.0.0.1/$c/a.proto")
}

func testMapAllEqualOrContainingPathsAbs(t *testing.T, expected []string, path string, keys ...string) {
	if expected == nil {
		expected = make([]string, 0)
	}
	sort.Strings(expected)
	keyMap := slicesext.ToStructMap(keys)
	assert.Equal(t, expected, MapAllEqualOrContainingPaths(keyMap, path, Absolute), fmt.Sprintf("%s %v", path, keys))
}

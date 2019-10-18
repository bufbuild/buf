package storagepath

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/internal/pkg/stringutil"
	"github.com/stretchr/testify/assert"
)

func TestWarnPathSeparator(t *testing.T) {
	fmt.Fprintf(os.Stderr, "WARN: os.PathSeparator is %q\n", string(os.PathSeparator))
}

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
}

func TestUnnormalize(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", Unnormalize(""))
	assert.Equal(t, ".", Unnormalize("."))
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
	if os.PathSeparator == '/' {
		assert.Equal(t, expected, filepath.Dir(input))
	}
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
	testJoin(t, ".", "", ".", "")
	testJoin(t, "foo/bar", "foo", "./bar")
	testJoin(t, "foo", "foo", "./bar", "..")
	testJoin(t, "/foo/bar", "/foo", "./bar")
	testJoin(t, "/foo", "/foo", "./bar", "..")
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
	actual, ok := stripComponents(path, count)
	assert.Equal(t, expectedOK, ok)
	assert.Equal(t, expected, actual)
}

func TestTransformer(t *testing.T) {
	t.Parallel()
	testTransformer(
		t,
		"bar",
		true,
		"foo/bar",
		WithStripComponents(1),
	)
	testTransformer(
		t,
		"bar",
		true,
		"foo/bar",
		WithStripComponents(1),
		WithExactPath("bar"),
	)
	testTransformer(
		t,
		"",
		false,
		"foo/bar",
		WithStripComponents(1),
		WithExactPath("foo/bar"),
	)
	testTransformer(
		t,
		"foo/bar",
		true,
		"foo/bar",
		WithExactPath("foo/bar"),
	)
}

func testTransformer(t *testing.T, expected string, expectedOK bool, path string, options ...TransformerOption) {
	transformed, ok := NewTransformer(options...).Transform(path)
	assert.Equal(t, expectedOK, ok)
	assert.Equal(t, expected, transformed)
}

func TestByDir(t *testing.T) {
	t.Parallel()
	assert.Equal(
		t,
		map[string][]string{
			"one": []string{
				"one/1.txt",
				"one/2.txt",
				"one/3.txt",
			},
			"two": []string{
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
			".": []string{
				"1.txt",
				"2.txt",
				"3.txt",
			},
			"two": []string{
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

func TestMapContainsMatch(t *testing.T) {
	testMapContainsMatch(t, true, "a.proto", "a.proto")
	testMapContainsMatch(t, false, ".", "a.proto")
	testMapContainsMatch(t, true, "a.proto", ".")
	testMapContainsMatch(t, true, "a/b.proto", ".")
	testMapContainsMatch(t, true, "a/b", ".")
	testMapContainsMatch(t, false, "ab/c", "a", "b")
	testMapContainsMatch(t, true, "a/b/c", "a", "b")
	testMapContainsMatch(t, false, "a/b/c", "b")
	testMapContainsMatch(t, true, "b/b/c", "b")
	testMapContainsMatch(t, true, "b/a/c", "b")
	testMapContainsMatch(t, true, "b/b/c", "b", ".")
}

func testMapContainsMatch(t *testing.T, expected bool, path string, keys ...string) {
	keyMap := stringutil.SliceToMap(keys)
	assert.Equal(t, expected, MapContainsMatch(keyMap, path), fmt.Sprintf("%s %v", path, keys))
}

func TestMapMatches(t *testing.T) {
	testMapMatches(t, []string{"a.proto"}, "a.proto", "a.proto")
	testMapMatches(t, nil, ".", "a.proto")
	testMapMatches(t, []string{"."}, "a.proto", ".")
	testMapMatches(t, []string{"."}, "a/b.proto", ".")
	testMapMatches(t, []string{"."}, "a/b", ".")
	testMapMatches(t, nil, "ab/c", "a", "b")
	testMapMatches(t, []string{"a"}, "a/b/c", "a", "b")
	testMapMatches(t, nil, "a/b/c", "b")
	testMapMatches(t, []string{"b"}, "b/b/c", "b")
	testMapMatches(t, []string{"b"}, "b/a/c", "b")
	testMapMatches(t, []string{"b", "."}, "b/b/c", "b", ".")
	testMapMatches(t, []string{"b", "b/b", "."}, "b/b/c", "b", "b/b", ".")
}

func testMapMatches(t *testing.T, expected []string, path string, keys ...string) {
	expectedMap := stringutil.SliceToMap(expected)
	keyMap := stringutil.SliceToMap(keys)
	assert.Equal(t, expectedMap, MapMatches(keyMap, path), fmt.Sprintf("%s %v", path, keys))
}

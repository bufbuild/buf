package bufbuild

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeAndValidateRootsExcludesError1(t *testing.T) {
	testNormalizeAndValidateRootsExcludesError(
		t,
		[]string{
			"/a/b",
		},
		[]string{},
	)
}

func TestNormalizeAndValidateRootsExcludesError2(t *testing.T) {
	testNormalizeAndValidateRootsExcludesError(
		t,
		[]string{},
		[]string{
			"/a/b",
		},
	)
}

func TestNormalizeAndValidateRootsExcludesError3(t *testing.T) {
	testNormalizeAndValidateRootsExcludesError(
		t,
		[]string{
			"a/b",
			"a/b",
		},
		[]string{},
	)
}

func TestNormalizeAndValidateRootsExcludesError4(t *testing.T) {
	testNormalizeAndValidateRootsExcludesError(
		t,
		[]string{
			"/a/b",
			"/a/b/c",
		},
		[]string{},
	)
}

func TestNormalizeAndValidateRootsExcludesError5(t *testing.T) {
	testNormalizeAndValidateRootsExcludesError(
		t,
		[]string{
			"a/b",
		},
		[]string{
			"a/c",
		},
	)
}

func TestNormalizeAndValidateRootsExcludesError6(t *testing.T) {
	testNormalizeAndValidateRootsExcludesError(
		t,
		[]string{
			".",
			"a",
		},
		[]string{},
	)
}

func testNormalizeAndValidateRootsExcludesError(t *testing.T, roots []string, excludes []string) {
	t.Parallel()
	_, _, err := normalizeAndValidateRootsExcludes(roots, excludes)
	assert.Error(t, err, fmt.Sprintf("%v %v", roots, excludes))
}

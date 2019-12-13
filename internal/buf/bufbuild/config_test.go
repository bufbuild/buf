package bufbuild

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfigError1(t *testing.T) {
	testNewConfigError(
		t,
		[]string{
			"/a/b",
		},
		[]string{},
	)
}

func TestNewConfigError2(t *testing.T) {
	testNewConfigError(
		t,
		[]string{},
		[]string{
			"/a/b",
		},
	)
}

func TestNewConfigError3(t *testing.T) {
	testNewConfigError(
		t,
		[]string{
			"a/b",
			"a/b",
		},
		[]string{},
	)
}

func TestNewConfigError4(t *testing.T) {
	testNewConfigError(
		t,
		[]string{
			"/a/b",
			"/a/b/c",
		},
		[]string{},
	)
}

func TestNewConfigError5(t *testing.T) {
	testNewConfigError(
		t,
		[]string{
			"a/b",
		},
		[]string{
			"a/c",
		},
	)
}

func TestNewConfigError6(t *testing.T) {
	testNewConfigError(
		t,
		[]string{
			".",
			"a",
		},
		[]string{},
	)
}

func testNewConfigError(t *testing.T, roots []string, excludes []string) {
	t.Parallel()
	_, err := newConfig(roots, excludes)
	assert.Error(t, err, fmt.Sprintf("%v %v", roots, excludes))
}

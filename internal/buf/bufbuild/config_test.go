package bufbuild

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func testNewConfig(
	t *testing.T,
	relRoots []string,
	relExcludes []string,
) *Config {
	config, err := ConfigBuilder{
		Roots:    relRoots,
		Excludes: relExcludes,
	}.NewConfig()
	require.NoError(t, err)
	return config
}

func testNewConfigError(t *testing.T, roots []string, excludes []string) {
	t.Parallel()
	_, err := ConfigBuilder{
		Roots:    roots,
		Excludes: excludes,
	}.NewConfig()
	assert.Error(t, err, fmt.Sprintf("%v %v", roots, excludes))
}

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

package bufmodulebuild

import (
	"fmt"
	"testing"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmoduletesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigSuccess1(t *testing.T) {
	// https://github.com/bufbuild/buf/issues/56
	t.Parallel()
	testNewConfigSuccess(
		t,
		[]string{
			"proto",
			"proto-vendor",
		},
		[]string{},
		[]string{},
	)
}

func TestNewConfigError1(t *testing.T) {
	t.Parallel()
	testNewConfigError(
		t,
		[]string{
			"/a/b",
		},
		[]string{},
		[]string{},
	)
}

func TestNewConfigError2(t *testing.T) {
	t.Parallel()
	testNewConfigError(
		t,
		[]string{},
		[]string{
			"/a/b",
		},
		[]string{},
	)
}

func TestNewConfigError3(t *testing.T) {
	t.Parallel()
	testNewConfigError(
		t,
		[]string{
			"a/b",
			"a/b",
		},
		[]string{},
		[]string{},
	)
}

func TestNewConfigError4(t *testing.T) {
	t.Parallel()
	testNewConfigError(
		t,
		[]string{
			"a/b",
			"a/b/c",
		},
		[]string{},
		[]string{},
	)
}

func TestNewConfigError5(t *testing.T) {
	t.Parallel()
	testNewConfigError(
		t,
		[]string{
			"a/b",
		},
		[]string{
			"a/c",
		},
		[]string{},
	)
}

func TestNewConfigError6(t *testing.T) {
	t.Parallel()
	testNewConfigError(
		t,
		[]string{
			".",
			"a",
		},
		[]string{},
		[]string{},
	)
}

func TestNewConfigError7(t *testing.T) {
	t.Parallel()
	testNewConfigError(
		t,
		[]string{
			"proto",
		},
		[]string{
			"proto/a/c",
			// error since not a directory
			"proto/d/1.proto",
		},
		[]string{},
	)
}

func TestNewConfigError8(t *testing.T) {
	t.Parallel()
	testNewConfigError(
		t,
		[]string{
			"proto",
		},
		[]string{},
		[]string{
			// Duplicate dependency
			"buf.build/foo/bar/v1",
			"buf.build/foo/bar/v1",
		},
	)
}

func TestNewConfigError9(t *testing.T) {
	t.Parallel()
	testNewConfigError(
		t,
		[]string{
			"proto",
		},
		[]string{},
		[]string{
			// Duplicate dependency with and without digest
			"buf.build/foo/bar/v1",
			"buf.build/foo/bar/v1:" + bufmoduletesting.TestDigest,
		},
	)
}

func TestNewConfigEqual1(t *testing.T) {
	t.Parallel()
	testNewConfigEqual(
		t,
		[]string{
			"a",
			"b",
		},
		[]string{
			"a/foo",
		},
		[]string{
			"buf.build/foo/bar/v1:" + bufmoduletesting.TestDigest,
			"buf.build/baz/qux/v2",
		},
		&Config{
			RootToExcludes: map[string][]string{
				"a": {
					"foo",
				},
				"b": {},
			},
			Deps: testParseDependencies(
				t,
				"buf.build/foo/bar/v1:"+bufmoduletesting.TestDigest,
				"buf.build/baz/qux/v2",
			),
		},
	)
}

func TestNewConfigEqual2(t *testing.T) {
	t.Parallel()
	testNewConfigEqual(
		t,
		[]string{
			"a",
			"b",
		},
		[]string{
			"a/foo",
			"b/foo",
			"b/bar",
		},
		[]string{},
		&Config{
			RootToExcludes: map[string][]string{
				"a": {
					"foo",
				},
				"b": {
					"bar",
					"foo",
				},
			},
		},
	)
}

func testNewConfigSuccess(t *testing.T, roots []string, excludes []string, deps []string) {
	_, err := NewConfig(ExternalConfig{Roots: roots, Excludes: excludes}, deps...)
	assert.NoError(t, err, fmt.Sprintf("%v %v %v", roots, excludes, deps))
}

func testNewConfigError(t *testing.T, roots []string, excludes []string, deps []string) {
	_, err := NewConfig(ExternalConfig{Roots: roots, Excludes: excludes}, deps...)
	assert.Error(t, err, fmt.Sprintf("%v %v %v", roots, excludes, deps))
}

func testNewConfigEqual(
	t *testing.T,
	roots []string,
	excludes []string,
	deps []string,
	expectedConfig *Config,
) {
	config, err := NewConfig(ExternalConfig{Roots: roots, Excludes: excludes}, deps...)
	assert.NoError(t, err, fmt.Sprintf("%v %v %v", roots, excludes, deps))
	assert.Equal(t, expectedConfig, config)
}

func testParseDependencies(t *testing.T, deps ...string) []bufmodule.ModuleName {
	moduleNames, err := parseDependencies(deps...)
	require.NoError(t, err)
	return moduleNames
}

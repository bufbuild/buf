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

package bufmod

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
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
		map[string][]string{
			"a": {
				"foo",
			},
			"b": {},
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
		map[string][]string{
			"a": {
				"foo",
			},
			"b": {
				"bar",
				"foo",
			},
		},
	)
}

func testNewConfigSuccess(t *testing.T, roots []string, excludes []string) {
	_, err := NewConfig(ExternalConfig{Roots: roots, Excludes: excludes})
	assert.NoError(t, err, fmt.Sprintf("%v %v", roots, excludes))
}

func testNewConfigError(t *testing.T, roots []string, excludes []string) {
	_, err := NewConfig(ExternalConfig{Roots: roots, Excludes: excludes})
	assert.Error(t, err, fmt.Sprintf("%v %v", roots, excludes))
}

func testNewConfigEqual(
	t *testing.T,
	roots []string,
	excludes []string,
	expectedRootToExcludes map[string][]string,
) {
	config, err := NewConfig(ExternalConfig{Roots: roots, Excludes: excludes})
	assert.NoError(t, err, fmt.Sprintf("%v %v", roots, excludes))
	assert.Equal(t, expectedRootToExcludes, config.RootToExcludes)
}

// Copyright 2020 Buf Technologies Inc.
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

package bufbuild

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeAndValidateRootsExcludesSuccess1(t *testing.T) {
	// https://github.com/bufbuild/buf/issues/56
	testNormalizeAndValidateRootsExcludesSuccess(
		t,
		[]string{
			"proto",
			"proto-vendor",
		},
		[]string{},
	)
}

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

func testNormalizeAndValidateRootsExcludesSuccess(t *testing.T, roots []string, excludes []string) {
	t.Parallel()
	_, _, err := normalizeAndValidateRootsExcludes(roots, excludes)
	assert.NoError(t, err, fmt.Sprintf("%v %v", roots, excludes))
}

func testNormalizeAndValidateRootsExcludesError(t *testing.T, roots []string, excludes []string) {
	t.Parallel()
	_, _, err := normalizeAndValidateRootsExcludes(roots, excludes)
	assert.Error(t, err, fmt.Sprintf("%v %v", roots, excludes))
}

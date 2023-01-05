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

package normalpath

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChunkByDir(t *testing.T) {
	t.Parallel()
	testChunkByDir(
		t,
		nil,
		0,
	)
	testChunkByDir(
		t,
		nil,
		5,
	)
	testChunkByDir(
		t,
		[]string{},
		0,
	)
	testChunkByDir(
		t,
		[]string{},
		5,
	)
	testChunkByDir(
		t,
		[]string{"a/a.proto"},
		1,
		[]string{"a/a.proto"},
	)
	testChunkByDir(
		t,
		[]string{"a/a.proto"},
		2,
		[]string{"a/a.proto"},
	)
	testChunkByDir(
		t,
		[]string{
			"a/a.proto",
			"b/b.proto",
		},
		1,
		[]string{"a/a.proto"},
		[]string{"b/b.proto"},
	)
	testChunkByDir(
		t,
		[]string{
			"a/a.proto",
			"a/b.proto",
			"b/b.proto",
		},
		1,
		[]string{
			"a/a.proto",
			"a/b.proto",
		},
		[]string{"b/b.proto"},
	)
	testChunkByDir(
		t,
		[]string{
			"a/a.proto",
			"a/b.proto",
			"b/b.proto",
		},
		2,
		[]string{
			"a/a.proto",
			"a/b.proto",
		},
		[]string{"b/b.proto"},
	)
	testChunkByDir(
		t,
		[]string{
			"a/a.proto",
			"a/b.proto",
			"b/b.proto",
		},
		3,
		[]string{
			"a/a.proto",
			"a/b.proto",
			"b/b.proto",
		},
	)
	testChunkByDir(
		t,
		[]string{
			"a/a.proto",
			"a/b.proto",
			"a/c.proto",
			"a/d.proto",
			"a/e.proto",
			"b/a.proto",
			"b/b.proto",
			"b/c.proto",
			"b/d.proto",
			"c/a.proto",
			"c/b.proto",
			"c/c.proto",
			"d/a.proto",
			"d/b.proto",
			"e/a.proto",
		},
		5,
		[]string{
			"a/a.proto",
			"a/b.proto",
			"a/c.proto",
			"a/d.proto",
			"a/e.proto",
		},
		[]string{
			"b/a.proto",
			"b/b.proto",
			"b/c.proto",
			"b/d.proto",
			"e/a.proto",
		},
		[]string{
			"c/a.proto",
			"c/b.proto",
			"c/c.proto",
			"d/a.proto",
			"d/b.proto",
		},
	)
	testChunkByDir(
		t,
		[]string{
			"a/a.proto",
			"a/b.proto",
			"a/c.proto",
			"a/d.proto",
			"a/e.proto",
			"b/a.proto",
			"b/b.proto",
			"b/c.proto",
			"b/d.proto",
			"c/a.proto",
			"c/b.proto",
			"c/c.proto",
			"d/a.proto",
			"d/b.proto",
			"e/a.proto",
		},
		6,
		[]string{
			"a/a.proto",
			"a/b.proto",
			"a/c.proto",
			"a/d.proto",
			"a/e.proto",
			"e/a.proto",
		},
		[]string{
			"b/a.proto",
			"b/b.proto",
			"b/c.proto",
			"b/d.proto",
			"d/a.proto",
			"d/b.proto",
		},
		[]string{
			"c/a.proto",
			"c/b.proto",
			"c/c.proto",
		},
	)
}

func testChunkByDir(t *testing.T, paths []string, suggestedChunkSize int, expected ...[]string) {
	// This is testing the implementation unfortunately, so if we change to a different
	// algorithm, our expectations will change.
	assert.Equal(t, expected, ChunkByDir(paths, suggestedChunkSize))
}

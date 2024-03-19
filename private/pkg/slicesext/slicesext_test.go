// Copyright 2020-2024 Buf Technologies, Inc.
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

package slicesext

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToUniqueSorted(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{}, ToUniqueSorted([]string{}))
	assert.Equal(t, []string{"Are", "bats", "cats"}, ToUniqueSorted([]string{"bats", "Are", "cats"}))
	assert.Equal(t, []string{"Are", "are", "bats", "cats"}, ToUniqueSorted([]string{"bats", "Are", "cats", "are"}))
	assert.Equal(t, []string{"Are", "Bats", "bats", "cats"}, ToUniqueSorted([]string{"bats", "Are", "cats", "Are", "Bats"}))
	assert.Equal(t, []string{"", "Are", "Bats", "bats", "cats"}, ToUniqueSorted([]string{"bats", "Are", "cats", "", "Are", "Bats", ""}))
	assert.Equal(t, []string{"", "  ", "Are", "Bats", "bats", "cats"}, ToUniqueSorted([]string{"bats", "Are", "cats", "", "Are", "Bats", "", "  "}))
	assert.Equal(t, []string{""}, ToUniqueSorted([]string{"", ""}))
	assert.Equal(t, []string{""}, ToUniqueSorted([]string{""}))
}

func TestElementsContained(t *testing.T) {
	t.Parallel()
	assert.True(t, ElementsContained([]string{}, []string{}))
	assert.True(t, ElementsContained(nil, []string{}))
	assert.True(t, ElementsContained([]string{}, nil))
	assert.True(t, ElementsContained([]string{"one"}, []string{"one"}))
	assert.True(t, ElementsContained([]string{"one", "two"}, []string{"one"}))
	assert.True(t, ElementsContained([]string{"one", "two"}, []string{"two"}))
	assert.True(t, ElementsContained([]string{"one", "two"}, []string{"one", "two"}))
	assert.True(t, ElementsContained([]string{"one", "two"}, []string{"two", "one"}))
	assert.False(t, ElementsContained([]string{"one", "two"}, []string{"three"}))
	assert.False(t, ElementsContained([]string{}, []string{"three"}))
	assert.False(t, ElementsContained([]string{"one"}, []string{"one", "two"}))
	assert.False(t, ElementsContained([]string{"two"}, []string{"one", "two"}))
}

func TestDuplicates(t *testing.T) {
	t.Parallel()
	assert.Equal(
		t,
		[]string{},
		Duplicates([]string{"a", "b", "c", "d", "e"}),
	)
	assert.Equal(
		t,
		[]string{"a"},
		Duplicates([]string{"a", "b", "c", "a", "e"}),
	)
	assert.Equal(
		t,
		[]string{"a"},
		Duplicates([]string{"a", "a", "c", "a", "e"}),
	)
	assert.Equal(
		t,
		[]string{"b", "a"},
		Duplicates([]string{"a", "b", "b", "a", "e"}),
	)
	assert.Equal(
		t,
		[]string{"b", "a"},
		Duplicates([]string{"a", "b", "b", "a", "b"}),
	)
}

func TestDeduplicate(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{}, Deduplicate([]string{}))
	assert.Equal(t, []string{"a"}, Deduplicate([]string{"a"}))
	assert.Equal(t, []string{"b", "a"}, Deduplicate([]string{"b", "a"}))
	assert.Equal(t, []string{"b", "a"}, Deduplicate([]string{"b", "a", "b"}))
}

func TestDeduplicateAny(t *testing.T) {
	t.Parallel()
	f := func(i int) int { return i / 2 }
	assert.Equal(t, []int{}, DeduplicateAny([]int{}, f))
	assert.Equal(t, []int{1}, DeduplicateAny([]int{1}, f))
	assert.Equal(t, []int{2, 1}, DeduplicateAny([]int{2, 1}, f))
	assert.Equal(t, []int{2, 1}, DeduplicateAny([]int{2, 1, 3}, f))
}

func TestToChunks(t *testing.T) {
	t.Parallel()
	testToChunks(
		t,
		[]string{"are"},
		1,
		[]string{"are"},
	)
	testToChunks(
		t,
		[]string{"are", "bats", "cats", "do", "eagle"},
		1,
		[]string{"are"},
		[]string{"bats"},
		[]string{"cats"},
		[]string{"do"},
		[]string{"eagle"},
	)
	testToChunks(
		t,
		[]string{"are", "bats", "cats", "do", "eagle"},
		2,
		[]string{"are", "bats"},
		[]string{"cats", "do"},
		[]string{"eagle"},
	)
	testToChunks(
		t,
		[]string{"are", "bats", "cats", "do", "eagle"},
		3,
		[]string{"are", "bats", "cats"},
		[]string{"do", "eagle"},
	)
	testToChunks(
		t,
		[]string{"are", "bats", "cats", "do", "eagle"},
		6,
		[]string{"are", "bats", "cats", "do", "eagle"},
	)
	testToChunks(
		t,
		nil,
		0,
	)
}

func testToChunks(t *testing.T, input []string, chunkSize int, expected ...[]string) {
	assert.Equal(t, expected, ToChunks(input, chunkSize))
}

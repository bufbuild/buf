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

package slicesextended

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
	assert.True(t, ElementsContained(nil, nil))
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

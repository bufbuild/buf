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

package bufcheckserverhandle

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollapseRanges(t *testing.T) {
	t.Parallel()
	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		doCollapseRanges(
			t,
			[]simpleTagRange{},
			nil,
		)
	})
	t.Run("single", func(t *testing.T) {
		t.Parallel()
		doCollapseRanges(
			t,
			[]simpleTagRange{
				{1, 100},
			},
			[]simpleTagRange{
				{1, 100},
			},
		)
	})
	t.Run("many", func(t *testing.T) {
		t.Parallel()
		doCollapseRanges(
			t,
			[]simpleTagRange{
				{1, 1},
				{2, 5},
				{7, 20},
				{21, 21},
				{22, 23},
				{99, 99},
				{100, 100},
				{101, 101},
				{110, 120},
			},
			[]simpleTagRange{
				{1, 5},
				{7, 23},
				{99, 101},
				{110, 120},
			},
		)
	})
	t.Run("overlaps", func(t *testing.T) {
		t.Parallel()
		doCollapseRanges(
			t,
			[]simpleTagRange{
				{1, 5},
				{2, 3},
				{7, 20},
				{15, 23},
				{99, 99},
				{100, 100},
				{101, 101},
				{110, 118},
				{116, 120},
				{120, 120},
			},
			[]simpleTagRange{
				{1, 5},
				{7, 23},
				{99, 101},
				{110, 120},
			},
		)
	})
}

func TestFindMissing(t *testing.T) {
	t.Parallel()
	ranges := []simpleTagRange{
		{10, 100},
		{200, 200},
		{300, 302},
		{305, 310},
		{312, 320},
		{330, 350},
	}
	missing := findMissing(10, 100, ranges)
	assert.Empty(t, missing)
	missing = findMissing(1, 5, ranges)
	assert.Equal(t, []simpleTagRange{{1, 5}}, missing)
	missing = findMissing(101, 110, ranges)
	assert.Equal(t, []simpleTagRange{{101, 110}}, missing)
	missing = findMissing(150, 200, ranges)
	assert.Equal(t, []simpleTagRange{{150, 199}}, missing)
	missing = findMissing(199, 201, ranges)
	assert.Equal(t, []simpleTagRange{{199, 199}, {201, 201}}, missing)
	missing = findMissing(200, 200, ranges)
	assert.Empty(t, missing)
	missing = findMissing(300, 300, ranges)
	assert.Empty(t, missing)
	missing = findMissing(300, 350, ranges)
	assert.Equal(t, []simpleTagRange{{303, 304}, {311, 311}, {321, 329}}, missing)
	missing = findMissing(335, 360, ranges)
	assert.Equal(t, []simpleTagRange{{351, 360}}, missing)
	missing = findMissing(400, 400, ranges)
	assert.Equal(t, []simpleTagRange{{400, 400}}, missing)
	missing = findMissing(1, 400, ranges)
	assert.Equal(t, []simpleTagRange{{1, 9}, {101, 199}, {201, 299}, {303, 304}, {311, 311}, {321, 329}, {351, 400}}, missing)
}

func doCollapseRanges(t *testing.T, input, expected []simpleTagRange) {
	t.Helper()
	collapsed := collapseRanges(input)
	assert.Equal(t, expected, collapsed)
	// Try some random permutations of the inputs and make sure
	// they resolve in the same way.
	for i := 0; i < 10; i++ {
		rand.Shuffle(len(input), func(i, j int) {
			input[i], input[j] = input[j], input[i]
		})
		collapsed := collapseRanges(input)
		assert.Equal(t, expected, collapsed)
	}
}

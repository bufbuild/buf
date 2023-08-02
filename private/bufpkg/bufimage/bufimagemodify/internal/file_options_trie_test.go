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

package internal

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileOptionsTrieInsert(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		description   string
		pathsToInsert [][]int32
	}{
		{
			description: "single path",
			pathsToInsert: [][]int32{
				{8, 4, 9, 5},
			},
		},
		{
			description: "insert ancestor after descendant",
			pathsToInsert: [][]int32{
				{8, 4, 9, 5},
				{8, 4},
			},
		},
		{
			description: "insert multiple ancestors after descendant",
			pathsToInsert: [][]int32{
				{20, 15, 10, 5}, // 0
				{20},            // 1
				{20, 10},        // 2
				{20, 15},        // 3
			},
		},
		{
			description: "insert descendants",
			pathsToInsert: [][]int32{
				{20},                         // 0
				{20, 50, 100},                // 1
				{20, 50, 100, 150},           // 2
				{20, 50, 100, 150, 300, 500}, // 3
				{20, 50, 100, 150, 300},      // 4
			},
		},
		{
			description: "insert last sibling",
			pathsToInsert: [][]int32{
				{20, 30, 50}, // 0
				{20, 50, 70}, // 1
				{20, 50, 80}, // 2
				{30, 10},     // 3
			},
		},
		{
			description: "insert first sibling",
			pathsToInsert: [][]int32{
				{20, 30, 50},    // 0
				{20, 0, 70},     // 1
				{20, 0, 20},     // 2
				{10, 10},        // 3
				{5, 10, 15, 20}, // 4
				{0},             // 5
			},
		},
		{
			description: "insert middle sibling",
			pathsToInsert: [][]int32{
				{20, 30, 50}, // 0
				{20, 30, 70}, // 1
				{20, 30, 60}, // 2
				{20, 30, 65}, // 3
				{20, 30, 55}, // 4
				{20, 0, 0},   // 5
				{20, 15, 0},  // 6
				{20, 6, 20},  // 7
				{20, 22, 20}, // 8
				{0, 50, 50},  // 9
				{10, 50, 50}, // 10
				{15, 50, 50}, // 11
				{5, 50, 50},  // 12
				{2, 50, 50},  // 13
				{1, 50, 50},  // 14
			},
		},
	}
	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()
			trie := fieldOptionsTrie{}
			for i, path := range testcase.pathsToInsert {
				// i is good enough for testing purposes since it's unique.
				trie.insert(path, i)
			}
			indicesInTrie := trie.indicesWithoutDescendant()
			sort.Ints(indicesInTrie)
			require.Equal(
				t,
				firstNNaturalNumbers(len(testcase.pathsToInsert)),
				indicesInTrie,
			)
		})
	}
}

func TestRegisterDescendant(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		description                      string
		pathsToInsert                    [][]int32
		pathsToRegister                  [][]int32
		expectedIndicesWithoutDescendant []int
	}{
		{
			description: "register none",
			pathsToInsert: [][]int32{
				{4, 0, 2, 0, 8},       // 0
				{4, 0, 2, 1, 8},       // 1
				{4, 0, 3, 0, 2, 1, 8}, // 2
			},
			expectedIndicesWithoutDescendant: []int{
				0,
				1,
				2,
			},
		},
		{
			description: "register non-child",
			pathsToInsert: [][]int32{
				{4, 0, 2, 0, 8},       // 0
				{4, 0, 2, 1, 8},       // 1
				{4, 0, 3, 0, 2, 1, 8}, // 2
			},
			pathsToRegister: [][]int32{
				{30},
			},
			expectedIndicesWithoutDescendant: []int{
				0,
				1,
				2,
			},
		},
		{
			description: "register sibling",
			pathsToInsert: [][]int32{
				{4, 0, 2, 0, 8},       // 0
				{4, 0, 2, 1, 8},       // 1
				{4, 0, 3, 0, 2, 1, 8}, // 2
			},
			pathsToRegister: [][]int32{
				{40, 0, 2, 0, 1},
			},
			expectedIndicesWithoutDescendant: []int{
				0,
				1,
				2,
			},
		},
		{
			description: "register child",
			pathsToInsert: [][]int32{
				{4, 0, 2, 3, 8},       // 0
				{4, 0, 2, 2, 8},       // 1
				{4, 0, 3, 0, 2, 1, 8}, // 2
			},
			pathsToRegister: [][]int32{
				{4, 0, 2, 2, 8, 6}, // descendant of 1
			},
			expectedIndicesWithoutDescendant: []int{
				0,
				2,
			},
		},
		{
			description: "register descendant",
			pathsToInsert: [][]int32{
				{4, 0, 2, 0, 8},       // 0
				{4, 0, 2, 2, 8},       // 1
				{4, 0, 3, 0, 2, 1, 8}, // 2
			},
			pathsToRegister: [][]int32{
				{4, 0, 2, 2, 8, 0, 1139}, // descendant of 1
			},
			expectedIndicesWithoutDescendant: []int{
				0,
				2,
			},
		},
		{
			description: "register multiple for the same parent",
			pathsToInsert: [][]int32{
				{4, 0, 2, 0, 8},       // 0
				{4, 0, 2, 1, 8},       // 1
				{4, 0, 3, 0, 2, 1, 8}, // 2
			},
			pathsToRegister: [][]int32{
				{4, 0, 2, 1, 8, 0, 1139}, // descendant of 1
				{4, 0, 2, 1, 8, 6},       // descendant of 1
			},
			expectedIndicesWithoutDescendant: []int{
				0,
				2,
			},
		},
		{
			description: "register for multiple parents",
			pathsToInsert: [][]int32{
				{4, 0, 2, 0, 8},             // 0, no register
				{4, 0, 2, 1, 8},             // 1
				{4, 0, 2, 2, 8},             // 2
				{4, 0, 3, 0, 2, 1, 8},       // 3, no register
				{4, 0, 3, 0, 3, 1, 2, 3, 8}, // 4
				{7, 0, 8},                   // 5
				{7, 1, 8},                   // 6, no register
			},
			pathsToRegister: [][]int32{
				{4, 0, 3, 0, 3, 1, 2, 3, 8, 1},
				{4, 0, 2, 1, 8, 0, 1139},
				{4, 0, 2, 1, 8, 6},
				{7, 0, 8, 50003, 0},
				{4, 0, 2, 2, 8, 5},
			},
			expectedIndicesWithoutDescendant: []int{
				0,
				3,
				6,
			},
		},
		{
			description: "register all non-field option",
			pathsToInsert: [][]int32{
				{4, 0, 2, 0, 8},             // 0
				{4, 0, 2, 1, 8},             // 1
				{4, 0, 2, 2, 8},             // 2
				{4, 0, 3, 0, 2, 1, 8},       // 3
				{4, 0, 3, 0, 3, 1, 2, 3, 8}, // 4
				{7, 0, 8},                   // 5
				{7, 1, 8},                   // 6
			},
			pathsToRegister: [][]int32{
				{4, 0, 3, 0, 3, 1, 2, 3, 8, 1}, // descendant of 4
				{4, 0, 2, 1, 8, 0, 1139},       // descendant of 1
				{4, 0, 2, 1, 8, 6},             // descendant of 1
				{7, 0, 8, 50003, 0},            // descendant of 5
				{4, 0, 2, 2, 8, 5},             // descendant of 2
				{7, 1, 8},                      // descendant of none
				{7, 1},                         // descendant of none
				{4, 0, 2, 0, 8},                // descendant of none
				{4, 0, 2},                      // descendant of none
			},
			expectedIndicesWithoutDescendant: []int{
				0,
				3,
				6,
			},
		},
		{
			description: "register for all ancestors",
			pathsToInsert: [][]int32{
				{4, 0, 2, 0, 8},             // 0
				{4, 0, 2, 1, 8},             // 1
				{4, 0, 2, 2, 8},             // 2
				{4, 0, 3, 0, 2, 1, 8},       // 3
				{4, 0, 3, 0, 3, 1, 2, 3, 8}, // 4
				{7, 0, 8},                   // 5
				{7, 1, 8},                   // 6
			},
			pathsToRegister: [][]int32{
				{4, 0, 3, 0, 3, 1, 2, 3, 8, 1}, // descendant of 4
				{4, 0, 2, 1, 8, 0, 1139},       // descendant of 1
				{4, 0, 2, 1, 8, 6},             // descendant of 1
				{7, 0, 8, 50003, 0},            // descendant of 5
				{4, 0, 2, 2, 8, 5},             // descendant of 2
				{7, 1, 8},                      // descendant of none
				{7, 1},                         // descendant of none
				{4, 0, 2, 0, 8},                // descendant of none
				{4, 0, 2},                      // descendant of none
				{7, 1, 8, 1},                   // descendant of 6
				{4, 0, 2, 0, 8, 2000, 1},       // descendant of 0
				{4, 0, 3, 0, 2, 1, 8, 6},       // descendant of 3
			},
			expectedIndicesWithoutDescendant: []int{}, // require.Equal does not consider nil the same as []int{}
		},
	}
	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()
			trie := &fieldOptionsTrie{}
			for i, path := range testcase.pathsToInsert {
				// i is good enough for testing purposes since it's unique
				trie.insert(path, i)
			}
			for _, path := range testcase.pathsToRegister {
				trie.registerDescendant(path)
			}
			remainingIndices := trie.indicesWithoutDescendant()
			sort.Ints(remainingIndices)
			require.Equal(
				t,
				testcase.expectedIndicesWithoutDescendant,
				remainingIndices,
			)
		})
	}
}

func firstNNaturalNumbers(n int) []int {
	numbers := make([]int, n)
	for i := 0; i < n; i++ {
		numbers[i] = i
	}
	return numbers
}

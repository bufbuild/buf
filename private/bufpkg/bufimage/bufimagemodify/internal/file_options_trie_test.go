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
	"golang.org/x/exp/slices"
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
			description: "insert ancestor after descendent",
			pathsToInsert: [][]int32{
				{8, 4, 9, 5},
				{8, 4},
			},
		},
		{
			description: "insert multiple ancestors after descendent",
			pathsToInsert: [][]int32{
				{20, 15, 10, 5},
				{20},
				{20, 10},
				{20, 15},
			},
		},
		{
			description: "insert descendents",
			pathsToInsert: [][]int32{
				{20},
				{20, 50, 100},
				{20, 50, 100, 150},
				{20, 50, 100, 150, 300, 500},
				{20, 50, 100, 150, 300},
			},
		},
		{
			description: "insert last sibling",
			pathsToInsert: [][]int32{
				{20, 30, 50},
				{20, 50, 70},
				{20, 50, 80},
				{30, 10},
			},
		},
		{
			description: "insert first sibling",
			pathsToInsert: [][]int32{
				{20, 30, 50},
				{20, 0, 70},
				{20, 0, 20},
				{10, 10},
				{5, 10, 15, 20},
				{0},
			},
		},
		{
			description: "insert middle sibling",
			pathsToInsert: [][]int32{
				{20, 30, 50},
				{20, 30, 70},
				{20, 0, 0},
				{20, 15, 0},
				{20, 6, 20},
				{20, 22, 20},
				{20, 30, 60},
				{20, 30, 65},
				{0, 50, 50},
				{10, 50, 50},
				{15, 50, 50},
				{5, 50, 50},
				{2, 50, 50},
				{1, 50, 50},
				{20, 30, 55},
				{20, 30, 52},
			},
		},
	}
	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.description, func(t *testing.T) {
			trie := &fieldOptionsTrie{}
			for _, path := range testcase.pathsToInsert {
				trie.insert(path)
			}
			sort.Slice(testcase.pathsToInsert, func(i, j int) bool {
				return slices.Compare(testcase.pathsToInsert[i], testcase.pathsToInsert[j]) < 0
			})
			// pathsWithoutChildren returns all paths in sorted order because it does a preorder traversal
			require.Equal(
				t,
				testcase.pathsToInsert,
				trie.pathsWithoutChildren(),
			)
		})
	}
}

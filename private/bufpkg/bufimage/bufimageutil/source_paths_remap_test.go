package bufimageutil

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSourcePathsRemapTrie_Insert(t *testing.T) {
	t.Parallel()
	expectedSlices := [][]string{
		{"4"},
		{"4", "1 -> -1"},
		{"4", "2 -> 1"},
		{"4", "2 -> 1", "2"},
		{"4", "2 -> 1", "2", "0 -> -1"},
		{"4", "2 -> 1", "2", "1 -> 0"},
		{"4", "2 -> 1", "2", "1 -> 0", "8"},
		{"4", "2 -> 1", "2", "1 -> 0", "8", "3 -> -1"},
		{"4", "2 -> 1", "2", "1 -> 0", "8", "4 -> -1"},
		{"4", "2 -> 1", "2", "1 -> 0", "8", "5 -> 3"},
		{"4", "2 -> 1", "2", "1 -> 0", "8", "6 -> 4"},
		{"4", "2 -> 1", "2", "1 -> 0", "8", "7 -> 5"},
		{"4", "3 -> 2"},
		{"4", "4 -> 3"},
		{"4", "5 -> -1"},
	}
	t.Run("in order", func(t *testing.T) {
		t.Parallel()
		trie := createTrie(nil)
		slices := asSlices(trie)
		require.Equal(t, expectedSlices, slices)
	})
	// shuffle a few times and make sure the trie is always constructed correctly
	for i := 0; i < 5; i++ {
		rnd := rand.New(rand.NewSource(int64(i)))
		t.Run(fmt.Sprintf("random order %d", i), func(t *testing.T) {
			t.Parallel()
			trie := createTrie(func(ops []insertionOp) []insertionOp {
				shuffle(rnd, ops)
				return ops
			})
			slices := asSlices(trie)
			require.Equal(t, expectedSlices, slices)
		})
	}
}

func TestSourcePathsRemapTrie_NewPath(t *testing.T) {
	t.Parallel()
	trie := createTrie(nil)
	// make sure the items in the trie construct correct new path
	path, noComment := trie.newPath([]int32{4, 1})
	require.Nil(t, path)
	require.False(t, noComment)
	path, noComment = trie.newPath([]int32{4, 2})
	require.Equal(t, []int32{4, 1}, path)
	require.True(t, noComment)
	path, noComment = trie.newPath([]int32{4, 2, 2, 0})
	require.Nil(t, path)
	require.False(t, noComment)
	path, noComment = trie.newPath([]int32{4, 2, 2, 1})
	require.Equal(t, []int32{4, 1, 2, 0}, path)
	require.False(t, noComment)
	path, noComment = trie.newPath([]int32{4, 2, 2, 1, 8, 3})
	require.Nil(t, path)
	require.False(t, noComment)
	path, noComment = trie.newPath([]int32{4, 2, 2, 1, 8, 4})
	require.Nil(t, path)
	require.False(t, noComment)
	path, noComment = trie.newPath([]int32{4, 2, 2, 1, 8, 5})
	require.Equal(t, []int32{4, 1, 2, 0, 8, 3}, path)
	require.False(t, noComment)
	path, noComment = trie.newPath([]int32{4, 2, 2, 1, 8, 6})
	require.Equal(t, []int32{4, 1, 2, 0, 8, 4}, path)
	require.False(t, noComment)
	path, noComment = trie.newPath([]int32{4, 2, 2, 1, 8, 7})
	require.Equal(t, []int32{4, 1, 2, 0, 8, 5}, path)
	require.False(t, noComment)
	path, noComment = trie.newPath([]int32{4, 3})
	require.Equal(t, []int32{4, 2}, path)
	require.False(t, noComment)
	path, noComment = trie.newPath([]int32{4, 4})
	require.Equal(t, []int32{4, 3}, path)
	require.False(t, noComment)
	path, noComment = trie.newPath([]int32{4, 5})
	require.Nil(t, path)
	require.False(t, noComment)

	// items not in the trie or not re-written remain unchanged
	path, noComment = trie.newPath([]int32{0, 1, 2, 3})
	require.Equal(t, []int32{0, 1, 2, 3}, path)
	require.False(t, noComment)
	path, noComment = trie.newPath([]int32{4, 0, 3, 2, 8, 5})
	require.Equal(t, []int32{4, 0, 3, 2, 8, 5}, path)
	require.False(t, noComment)
	path, noComment = trie.newPath([]int32{4, 6})
	require.Equal(t, []int32{4, 6}, path)
	require.False(t, noComment)
	// items that are descendants of re-written items are also changed
	path, noComment = trie.newPath([]int32{4, 2, 2, 1, 8, 5, 5, 9, 4, 4})
	require.Equal(t, []int32{4, 1, 2, 0, 8, 3, 5, 9, 4, 4}, path)
	require.False(t, noComment)
	path, noComment = trie.newPath([]int32{4, 4, 9, 4, 3, 5})
	require.Equal(t, []int32{4, 3, 9, 4, 3, 5}, path)
	require.False(t, noComment)
	// items that are descendants of deleted items are also deleted
	path, noComment = trie.newPath([]int32{4, 5, 1, 3, 4, 5})
	require.Nil(t, path)
	require.False(t, noComment)
}

type insertionOp struct {
	oldPath  []int32
	newIndex int32
}

func createTrie(permutation func([]insertionOp) []insertionOp) *sourcePathsRemapTrie {
	// Test data has the following source info path changes
	// 		4,1 -> deleted
	// 		4,2 -> 4,1 *no comment
	// 		4,2,2,0 -> deleted
	// 		4,2,2,1 -> 4,1,2,0
	// 		4,2,2,1,8,3 -> deleted
	// 		4,2,2,1,8,4 -> deleted
	// 		4,2,2,1,8,5 -> 4,1,2,0,8,3
	// 		4,2,2,1,8,6 -> 4,1,2,0,8,4
	// 		4,2,2,1,8,7 -> 4,1,2,0,8,5
	// 		4,3 -> 4,2
	// 		4,4 -> 4,3
	// 		4,5 -> deleted
	// Test data is sorted (unless permutation function rearranges)
	ops := []insertionOp{
		{[]int32{4, 1}, -1},
		{[]int32{4, 2}, 1},
		{[]int32{4, 2}, -2},
		{[]int32{4, 2, 2, 0}, -1},
		{[]int32{4, 2, 2, 1}, 0},
		{[]int32{4, 2, 2, 1, 8, 3}, -1},
		{[]int32{4, 2, 2, 1, 8, 4}, -1},
		{[]int32{4, 2, 2, 1, 8, 5}, 3},
		{[]int32{4, 2, 2, 1, 8, 6}, 4},
		{[]int32{4, 2, 2, 1, 8, 7}, 5},
		{[]int32{4, 3}, 2},
		{[]int32{4, 4}, 3},
		{[]int32{4, 5}, -1},
	}
	if permutation != nil {
		ops = permutation(ops)
	}
	trie := &sourcePathsRemapTrie{}
	for _, op := range ops {
		if op.newIndex == -2 {
			trie.noComment(op.oldPath)
			continue
		}
		trie.insert(op.oldPath, op.newIndex)
	}
	return trie
}

func shuffle[T any](rnd *rand.Rand, slice []T) {
	for i := range slice {
		pick := rnd.Intn(len(slice)-i) + i
		if i != pick {
			slice[i], slice[pick] = slice[pick], slice[i]
		}
	}
}

func asSlices(t *sourcePathsRemapTrie) [][]string {
	var result [][]string
	for _, child := range *t {
		toSlices(child, nil, &result)
	}
	return result
}

func toSlices(t *sourcePathsRemapTrieNode, soFar []string, result *[][]string) {
	if t.oldIndex == t.newIndex {
		soFar = append(soFar, fmt.Sprintf("%d", t.oldIndex))
	} else {
		soFar = append(soFar, fmt.Sprintf("%d -> %d", t.oldIndex, t.newIndex))
	}
	clone := make([]string, len(soFar))
	copy(clone, soFar)
	*result = append(*result, clone)
	if len(t.children) == 0 {
		return
	}
	for _, child := range t.children {
		toSlices(child, soFar, result)
	}
}

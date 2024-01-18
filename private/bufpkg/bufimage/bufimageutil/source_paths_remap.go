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

package bufimageutil

import "sort"

// sourcePathsRemapTrieNode is a node in a trie. Each node represents the
// path of a source code location.
type sourcePathsRemapTrieNode struct {
	oldIndex int32
	// If == -1, the item at this path is being deleted. Otherwise,
	// if != oldIndex, the item at this path is being moved, as well as
	// all its descendents in the trie.
	newIndex int32
	// If true, the item at this point has its comments omitted. This is
	// used to omit comments for messages that, after filtering, are only
	// present as a namespace (so the comments likely no longer apply).
	noComment bool
	// This node's children. These represent paths for which the current
	// node is a path prefix (aka ancestor).
	children sourcePathsRemapTrie
}

// sourcePathsRemapTrie is a trie (aka prefix tree) whose children are a
// sorted slice (more efficient than a map, mainly due to not having to
// sort it with every addition, in practice, since source code info is
// mostly sorted).
//
// Each node in the trie represents some path of a source code location.
// This is used to track renumbering and deletions of paths.
type sourcePathsRemapTrie []*sourcePathsRemapTrieNode

// markMoved inserts the given path into the trie and marks the last element
// of oldPath to be replaced with newIndex.
func (t *sourcePathsRemapTrie) markMoved(oldPath []int32, newIndex int32) {
	t.doTrieInsert(oldPath, newIndex, false)
}

// markDeleted marks the given path for deletion.
func (t *sourcePathsRemapTrie) markDeleted(oldPath []int32) {
	t.doTrieInsert(oldPath, -1, false)
}

// markNoComment inserts the given path into the trie and marks the element so
// its comments will be dropped.
func (t *sourcePathsRemapTrie) markNoComment(oldPath []int32) {
	t.doTrieInsert(oldPath, oldPath[len(oldPath)-1], true)
}

func (t *sourcePathsRemapTrie) doTrieInsert(oldPath []int32, newIndex int32, noComment bool) {
	if t == nil {
		return
	}
	items := *t
	searchIndex := oldPath[0]
	idx, found := sort.Find(len(items), func(i int) int {
		return int(searchIndex - items[i].oldIndex)
	})
	if !found {
		// shouldn't usually need to sort because incoming items are often in order
		needSort := len(items) > 0 && searchIndex < items[len(items)-1].oldIndex
		idx = len(items)
		items = append(items, &sourcePathsRemapTrieNode{
			oldIndex: searchIndex,
			newIndex: searchIndex,
		})
		if needSort {
			sort.Slice(items, func(i, j int) bool {
				return items[i].oldIndex < items[j].oldIndex
			})
			// find the index of the thing we just added
			idx, _ = sort.Find(len(items), func(i int) int {
				return int(searchIndex - items[i].oldIndex)
			})
		}
		*t = items
	}
	if len(oldPath) > 1 {
		items[idx].children.doTrieInsert(oldPath[1:], newIndex, noComment)
		return
	}
	if noComment {
		items[idx].noComment = noComment
	} else {
		items[idx].newIndex = newIndex
	}
}

// newPath returns the corrected path of oldPath, given any moves and
// deletions inserted into t. If the item at the given oldPath was deleted
// then nil is returned. Otherwise, the corrected path is returned. If the
// item at oldPath was not moved or deleted, the returned path has the
// same values as oldPath.
func (t *sourcePathsRemapTrie) newPath(oldPath []int32) (path []int32, noComment bool) {
	if len(oldPath) == 0 {
		// make sure return value is non-nil, so response doesn't
		// get confused for "delete this entry"
		return []int32{}, false
	}
	if t == nil {
		return oldPath, false
	}
	newPath := make([]int32, len(oldPath))
	keep, noComment := t.fix(oldPath, newPath)
	if !keep {
		return nil, false
	}
	return newPath, noComment
}

func (t *sourcePathsRemapTrie) fix(oldPath, newPath []int32) (keep, noComment bool) {
	items := *t
	searchIndex := oldPath[0]
	idx, found := sort.Find(len(items), func(i int) int {
		return int(searchIndex - items[i].oldIndex)
	})
	if !found {
		copy(newPath, oldPath)
		return true, false
	}
	item := items[idx]
	if item.newIndex == -1 {
		return false, false
	}
	newPath[0] = item.newIndex
	if len(oldPath) > 1 {
		if item.newIndex == -1 {
			newPath[0] = item.oldIndex
		}
		return item.children.fix(oldPath[1:], newPath[1:])
	}
	return true, item.noComment
}

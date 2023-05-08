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

package bufimageutil

import "sort"

type sourcePathsRemapTrieNode struct {
	oldIndex  int32
	newIndex  int32
	noComment bool
	children  sourcePathsRemapTrie
}

type sourcePathsRemapTrie []*sourcePathsRemapTrieNode

// insert inserts the given path into the trie and marks the last element
// of oldPath to be replaced with newIndex. If newIndex is -1 then it means
// the element at oldPath was deleted, not moved.
func (t *sourcePathsRemapTrie) insert(oldPath []int32, newIndex int32) {
	t.doInsert(oldPath, newIndex, false)
}

// noComment inserts the given path into the trie and marks the element so
// its comments will be dropped.
func (t *sourcePathsRemapTrie) noComment(oldPath []int32) {
	t.doInsert(oldPath, oldPath[len(oldPath)-1], true)
}

func (t *sourcePathsRemapTrie) doInsert(oldPath []int32, newIndex int32, noComment bool) {
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
		items[idx].children.doInsert(oldPath[1:], newIndex, noComment)
		return
	}
	if noComment {
		items[idx].noComment = noComment
	} else {
		items[idx].newIndex = newIndex
	}
}

// newPath returns the corrected path of oldPath, given any moves and
// deletions insert into t. If the item at the given oldPath was deleted
// nil is returned. Otherwise, the corrected path is returned. If the
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

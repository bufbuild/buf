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

	"golang.org/x/exp/slices"
)

type fieldOptionsTrie []*fieldOptionsTrieNode

type fieldOptionsTrieNode struct {
	value    int32
	path     []int32
	count    int
	children fieldOptionsTrie
}

func (p *fieldOptionsTrie) insert(path []int32) {
	trie := p
	for index, element := range path {
		isLastElement := index == len(path)-1
		nodes := *trie
		pos, found := sort.Find(len(nodes), func(i int) int {
			return int(element - nodes[i].value)
		})
		if found {
			if isLastElement {
				nodes[pos].path = path
				return
			}
			trie = &nodes[pos].children
			continue
		}
		newNode := &fieldOptionsTrieNode{
			value:    element,
			path:     nil, // not a path end
			children: fieldOptionsTrie{},
		}
		if isLastElement {
			newNode.path = path
		}
		nodes = slices.Insert(nodes, pos, newNode)
		*trie = nodes
		trie = &nodes[pos].children
	}
}

func (p *fieldOptionsTrie) registerChild(childPath []int32) {
}

func (p *fieldOptionsTrie) pathsWithoutChildren() [][]int32 {
	paths := [][]int32{}
	walkTrie(*p, func(node *fieldOptionsTrieNode) {
		if len(node.path) > 0 && node.count == 0 {
			paths = append(paths, node.path)
		}
	})
	return paths
}

func walkTrie(trie fieldOptionsTrie, f func(node *fieldOptionsTrieNode)) {
	for _, node := range trie {
		walkTrieNode(node, f)
	}
}

func walkTrieNode(node *fieldOptionsTrieNode, f func(node *fieldOptionsTrieNode)) {
	f(node)
	walkTrie(node.children, f)
}

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

package internal

import (
	"sort"

	"golang.org/x/exp/slices"
)

// fieldOptionsTrie stores paths to FieldOptions (tag 8 of a FieldDescriptorProto).
type fieldOptionsTrie []*fieldOptionsTrieNode

type fieldOptionsTrieNode struct {
	value     int32
	isPathEnd bool
	children  fieldOptionsTrie
	// locationIndex is the data attached that we are interested in retrieving.
	// This field is irrelevant to traversal or the trie structure.
	locationIndex int
	// registeredDescendantCount records how many times a descendant of this node
	// is registered. This field is irrelevant to traversal or the trie structure.
	registeredDescendantCount int
}

// insert inserts a path into the trie. The caller should
// ensure that the path is for a FieldOptions.
func (p *fieldOptionsTrie) insert(path []int32, locationIndex int) {
	trie := p
	for index, element := range path {
		isLastElement := index == len(path)-1
		nodes := *trie
		pos, found := sort.Find(len(nodes), func(i int) int {
			return int(element - nodes[i].value)
		})
		if found {
			if isLastElement {
				nodes[pos].isPathEnd = true
				nodes[pos].locationIndex = locationIndex
				return
			}
			trie = &nodes[pos].children
			continue
		}
		newNode := &fieldOptionsTrieNode{
			value:    element,
			children: fieldOptionsTrie{},
		}
		if isLastElement {
			newNode.isPathEnd = true
			newNode.locationIndex = locationIndex
		}
		nodes = slices.Insert(nodes, pos, newNode)
		*trie = nodes
		trie = &nodes[pos].children
	}
}

// registerDescendant finds if there is an ancestor of the provided
// path and increments this ancestor's counter if it exists.
func (p *fieldOptionsTrie) registerDescendant(descendant []int32) {
	trie := p
	for i, element := range descendant {
		nodes := *trie
		pos, found := sort.Find(len(nodes), func(i int) int {
			return int(element - nodes[i].value)
		})
		if !found {
			return
		}
		ancestor := nodes[pos]
		descendantContinues := i != len(descendant)-1
		if ancestor.isPathEnd && descendantContinues {
			ancestor.registeredDescendantCount += 1
			return
		}
		trie = &ancestor.children
	}
}

// indicesWithoutDescendant returns the location indices of
func (p *fieldOptionsTrie) indicesWithoutDescendant() []int {
	locationIndices := []int{}
	walkTrie(*p, func(node *fieldOptionsTrieNode) {
		if node.isPathEnd && node.registeredDescendantCount == 0 {
			locationIndices = append(locationIndices, node.locationIndex)
		}
	})
	return locationIndices
}

func walkTrie(trie fieldOptionsTrie, enter func(node *fieldOptionsTrieNode)) {
	for _, node := range trie {
		walkTrieNode(node, enter)
	}
}

func walkTrieNode(node *fieldOptionsTrieNode, enter func(node *fieldOptionsTrieNode)) {
	enter(node)
	walkTrie(node.children, enter)
}

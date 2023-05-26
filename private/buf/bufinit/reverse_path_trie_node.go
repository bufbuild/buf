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

package bufinit

import (
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/stringutil"
)

type reversePathTrieNode struct {
	componentToChild map[string]*reversePathTrieNode
}

func newReversePathTrieNode() *reversePathTrieNode {
	return &reversePathTrieNode{
		componentToChild: make(map[string]*reversePathTrieNode),
	}
}

// Insert inserts paths in reverse to a trie.
//
// Example: a/b/c.proto is inserted as c.proto -> b -> a.
//
// path assumed to be normalized, validated, and non-empty.
func (r *reversePathTrieNode) Insert(path string) {
	r.insert(reverseComponents(path))
}

// Get will get the directories that contain a given path, if present.
//
// present is true if this is a path contained in the trie.
// If present is true, directories is the list of directories that contain this path.
//
// Example: Trie contains paths a/b/c.proto, a/d/c.proto -> Get for "c.proto"
// will return (["a/b", "a/d"], true), however "e/c.proto" will return (nil, false).
//
// path assumed to be normalized, validated, and non-empty.
func (r *reversePathTrieNode) Get(path string) (directories []string, present bool) {
	return r.get(reverseComponents(path))
}

func (r *reversePathTrieNode) insert(reverseComponents []string) {
	if len(reverseComponents) == 0 {
		return
	}
	node, ok := r.componentToChild[reverseComponents[0]]
	if !ok {
		node = newReversePathTrieNode()
		r.componentToChild[reverseComponents[0]] = node
	}
	node.insert(reverseComponents[1:])
}

func (r *reversePathTrieNode) get(reverseComponents []string) ([]string, bool) {
	if len(reverseComponents) == 0 {
		return r.getAllDirectories("."), true
	}
	child, ok := r.componentToChild[reverseComponents[0]]
	if !ok {
		return nil, false
	}
	return child.get(reverseComponents[1:])
}

func (r *reversePathTrieNode) getAllDirectories(base string) []string {
	if len(r.componentToChild) == 0 {
		return []string{base}
	}
	directoryMap := make(map[string]struct{})
	for component, child := range r.componentToChild {
		// component comes first, since this is reverse
		for _, childDirectory := range child.getAllDirectories(normalpath.Join(component, base)) {
			directoryMap[childDirectory] = struct{}{}
		}
	}
	return stringutil.MapToSortedSlice(directoryMap)
}

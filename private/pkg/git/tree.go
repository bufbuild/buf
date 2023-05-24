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

package git

import (
	"bytes"
	"errors"
	"fmt"
)

type tree struct {
	hash  Hash
	nodes []Node
}

func parseTree(hash Hash, data []byte) (*tree, error) {
	t := &tree{
		hash: hash,
	}
	for len(data) > 0 {
		i := bytes.Index(data, []byte{0})
		if i == -1 {
			return nil, errors.New("malformed tree")
		}
		length := i + 1 + hashLength
		node, err := parseTreeNode(data[:length])
		if err != nil {
			return nil, fmt.Errorf("malformed tree: %w", err)
		}
		t.nodes = append(t.nodes, node)
		data = data[length:]
	}
	return t, nil
}

func (t *tree) Hash() Hash {
	return t.hash
}

func (t *tree) Nodes() []Node {
	return t.nodes
}

func (t *tree) Traverse(objectReader ObjectReader, names ...string) (Node, error) {
	return traverse(objectReader, t, names...)
}

func traverse(
	objectReader ObjectReader,
	root Tree,
	names ...string,
) (Node, error) {
	name := names[0]
	names = names[1:]
	// Find name in this tree.
	var found Node
	for _, node := range root.Nodes() {
		if node.Name() == name {
			found = node
			break
		}
	}
	if found == nil {
		// No name in this tree.
		return nil, ErrSubTreeNotFound
	}
	if len(names) == 0 {
		// We found it.
		return found, nil
	}
	if found.Mode() != ModeDir {
		// Part of the path is not a directory.
		return nil, ErrSubTreeNotFound
	}
	// TODO: support symlinks (on intermediate dirs) with traverse option
	// Walk down the tree.
	tree, err := objectReader.Tree(found.Hash())
	if err != nil {
		return nil, err
	}
	return traverse(objectReader, tree, names...)
}

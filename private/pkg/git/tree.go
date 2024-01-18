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

package git

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/pkg/normalpath"
)

type tree struct {
	hash  Hash
	nodes []TreeNode
}

func parseTree(hash Hash, data []byte) (*tree, error) {
	t := &tree{
		hash: hash,
	}
	/*
		data is in the format
			<mode><space><name>\0<hash>
		repeated
	*/
	for len(data) > 0 {
		// We can find the \0 character before the <hash>
		// and slice to the index of \0 + the length of a hash.
		// That gives us a single node.
		i := bytes.Index(data, []byte{0})
		if i == -1 {
			return nil, errors.New("parse tree")
		}
		length := i + 1 + hashLength
		node, err := parseTreeNode(data[:length])
		if err != nil {
			return nil, fmt.Errorf("parse tree: %w", err)
		}
		t.nodes = append(t.nodes, node)
		data = data[length:]
	}
	return t, nil
}

func (t *tree) Hash() Hash {
	return t.hash
}

func (t *tree) Nodes() []TreeNode {
	return t.nodes
}

func (t *tree) Descendant(path string, objectReader ObjectReader) (TreeNode, error) {
	if path == "" {
		return nil, errors.New("empty path")
	}
	return descendant(objectReader, t, normalpath.Components(path))
}

func descendant(
	objectReader ObjectReader,
	root Tree,
	names []string,
) (TreeNode, error) {
	// split by the name of the next node we're looking for
	// and the names of the descendant nodes
	name := names[0]
	if len(names) >= 2 {
		names = names[1:]
	} else {
		names = nil
	}
	// Find node with that name in this tree.
	var found TreeNode
	for _, node := range root.Nodes() {
		if node.Name() == name {
			found = node
			break
		}
	}
	if found == nil {
		// No node with that name in this tree.
		return nil, ErrTreeNodeNotFound
	}
	if len(names) == 0 {
		// No more descendants, we've found our terminal node.
		return found, nil
	}
	if found.Mode() != ModeDir {
		// This is an intermediate (non-terminal) node, which are expected to be
		// directories. This is node is not a directory, so we fail with a non-found
		// errror.
		return nil, ErrTreeNodeNotFound
	}
	// TODO: support symlinks (on intermediate dirs) with descendant option
	// Descend down and traverse.
	tree, err := objectReader.Tree(found.Hash())
	if err != nil {
		return nil, err
	}
	return descendant(objectReader, tree, names)
}

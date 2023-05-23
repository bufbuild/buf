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
	hash    Hash
	entries []TreeEntry
}

func (t *tree) Hash() Hash {
	return t.hash
}

func (t *tree) Entries() []TreeEntry {
	return t.entries
}

func (t *tree) Traverse(objectReader ObjectReader, names ...string) (TreeEntry, error) {
	return traverse(objectReader, t, names...)
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
		entry, err := parseTreeEntry(data[:length])
		if err != nil {
			return nil, fmt.Errorf("malformed tree: %w", err)
		}
		t.entries = append(t.entries, entry)
		data = data[length:]
	}
	return t, nil
}

func traverse(
	objectReader ObjectReader,
	root Tree,
	names ...string,
) (TreeEntry, error) {
	name := names[0]
	names = names[1:]
	// Find name in this tree.
	var found TreeEntry
	for _, entry := range root.Entries() {
		if entry.Name() == name {
			found = entry
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
	// Walk down the tree.
	tree, err := objectReader.Tree(found.Hash())
	if err != nil {
		return nil, err
	}
	return traverse(objectReader, tree, names...)
}

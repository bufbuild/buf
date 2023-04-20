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

package gittest

import (
	"errors"
	"hash"
	"hash/fnv"

	"github.com/bufbuild/buf/private/pkg/git/object"
)

// MemObjectStore implements a trivial ObjectService useful for testing.
type MemObjectStore struct {
	hasher hash.Hash
	trees  map[string]*object.Tree
	blobs  map[string][]byte
}

// NewMemObjectStoreFromMap builds a MemObjectStore from a map representing
// files.
func NewMemObjectStoreFromMap(files map[string]string) (*MemObjectStore, object.ID, error) {
	// Project files into a more useful tree structure.
	tree, err := newTree(files)
	if err != nil {
		return nil, nil, err
	}
	// Put it into in-memory go-git storage.
	store := NewMemObjectStore()
	root, err := tree.WriteStore(store)
	if err != nil {
		return nil, nil, err
	}
	return store, root, nil
}

// NewMemObjectStore creates an empty store.
func NewMemObjectStore() *MemObjectStore {
	return &MemObjectStore{
		hasher: fnv.New128(),
		trees:  make(map[string]*object.Tree),
		blobs:  make(map[string][]byte),
	}
}

func (m *MemObjectStore) Commit(id object.ID) (*object.Commit, error) {
	return nil, errors.New("unimplemented")
}

func (m *MemObjectStore) Tree(id object.ID) (*object.Tree, error) {
	tree, ok := m.trees[id.String()]
	if !ok {
		return nil, errors.New("not found")
	}
	return tree, nil
}

func (m *MemObjectStore) Blob(id object.ID) ([]byte, error) {
	blob, ok := m.blobs[id.String()]
	if !ok {
		return nil, errors.New("not found")
	}
	return blob, nil
}

func (m *MemObjectStore) Close() error {
	return nil
}

// PutTree adds tree to the store.
func (m *MemObjectStore) PutTree(tree *object.Tree) (object.ID, error) {
	m.hasher.Reset()
	for _, entry := range tree.Entries {
		if _, err := m.hasher.Write([]byte(entry.ID)); err != nil {
			return nil, err
		}
	}
	id := object.ID(m.hasher.Sum(nil))
	m.trees[id.String()] = tree
	return id, nil
}

// PutBlob adds blob to the store.
func (m *MemObjectStore) PutBlob(blob []byte) (object.ID, error) {
	m.hasher.Reset()
	id := object.ID(m.hasher.Sum(blob))
	m.blobs[id.String()] = blob
	return id, nil
}

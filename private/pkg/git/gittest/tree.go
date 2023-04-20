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
	"path"
	"strings"

	"github.com/bufbuild/buf/private/pkg/git/object"
)

type file struct {
	name    string
	content string
}

type tree struct {
	files []file
	dirs  map[string]*tree
}

// newTree builds a tree from a map representing files.
func newTree(files map[string]string) (*tree, error) {
	root := &tree{
		files: nil,
		dirs:  make(map[string]*tree),
	}
	for filename, content := range files {
		if err := root.Add(path.Clean(filename), content); err != nil {
			return nil, err
		}
	}
	return root, nil
}

// WriteStore encodes tree into an object store. The store and ID of the root
// is returned.
func (t *tree) WriteStore(store *MemObjectStore) (object.ID, error) {
	// Store directories.
	var entries []object.TreeEntry
	for dirname, dir := range t.dirs {
		id, err := dir.WriteStore(store)
		if err != nil {
			return nil, err
		}
		entries = append(entries, object.TreeEntry{
			Name: dirname,
			Mode: object.ModeDir,
			ID:   id,
		})
	}
	// Store files.
	for _, f := range t.files {
		content := []byte(f.content)
		id, err := store.PutBlob(content)
		if err != nil {
			return nil, err
		}
		entries = append(entries, object.TreeEntry{
			Name: f.name,
			Mode: object.ModeFile,
			ID:   id,
		})
	}
	// Store the tree.
	return store.PutTree(&object.Tree{Entries: entries})
}

// Add places a file into the given tree.
func (t *tree) Add(name, content string) error {
	dirname, basename := path.Split(name)
	dirname = path.Clean(dirname)
	if dirname == ".." {
		return errors.New("escaping beyond root is unsupported")
	}
	if dirname == "." {
		t.files = append(t.files, file{name: name, content: content})
		return nil
	}
	treename, remaining, _ := strings.Cut(dirname, "/")
	subtree, ok := t.dirs[treename]
	if !ok {
		subtree = &tree{dirs: make(map[string]*tree)}
		t.dirs[treename] = subtree
	}
	return subtree.Add(path.Join(remaining, basename), content)
}

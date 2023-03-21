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
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreeReader(t *testing.T) {
	t.Run("file exists", func(t *testing.T) {
		t.Parallel()
		tree := makeObjectTree(t, map[string]string{
			"foo": "bar",
		})
		treeReader := NewTreeReader(tree)
		info, err := treeReader.Stat(context.Background(), "foo")
		require.NoError(t, err)
		assert.Equal(t, "foo", info.Path())
	})
	t.Run("file doesn't exist", func(t *testing.T) {
		t.Parallel()
		tree := makeObjectTree(t, map[string]string{})
		treeReader := NewTreeReader(tree)
		_, err := treeReader.Stat(context.Background(), "foo")
		assert.True(t, storage.IsNotExist(err))
	})
	t.Run("read foo", func(t *testing.T) {
		t.Parallel()
		tree := makeObjectTree(t, map[string]string{
			"foo": "bar",
		})
		treeReader := NewTreeReader(tree)
		file, err := treeReader.Get(context.Background(), "foo")
		require.NoError(t, err)
		bytes, err := io.ReadAll(file)
		require.NoError(t, err)
		assert.Equal(t, "foo", file.Path())
		assert.Equal(t, "bar", string(bytes))
	})
	t.Run("walk", func(t *testing.T) {
		t.Parallel()
		tree := makeObjectTree(t, map[string]string{
			"foo": "bar",
			"baz": "qux",
		})
		treeReader := NewTreeReader(tree)
		count := 0
		err := treeReader.Walk(
			context.Background(),
			"",
			func(info storage.ObjectInfo) error {
				count++
				switch info.Path() {
				case "foo":
				case "baz":
				default:
					return fmt.Errorf("unknown file: %q", info.Path())
				}
				return nil
			},
		)
		require.NoError(t, err)
		assert.Equal(t, 2, count, "unexpected number of callbacks")
	})
	t.Run("walk with not found prefix", func(t *testing.T) {
		t.Parallel()
		tree := makeObjectTree(t, map[string]string{
			"foo": "bar",
		})
		treeReader := NewTreeReader(tree)
		count := 0
		err := treeReader.Walk(
			context.Background(),
			"dir",
			func(info storage.ObjectInfo) error {
				count++
				return nil
			},
		)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "unexpected number of callbacks")
	})
	t.Run("walk with found prefix", func(t *testing.T) {
		t.Parallel()
		tree := makeObjectTree(t, map[string]string{
			"foo":     "bar",
			"dir/baz": "qux",
		})
		treeReader := NewTreeReader(tree)
		count := 0
		err := treeReader.Walk(
			context.Background(),
			"dir",
			func(info storage.ObjectInfo) error {
				count++
				if info.Path() != "dir/baz" {
					return fmt.Errorf("unknown file: %q", info.Path())
				}
				return nil
			},
		)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "unexpected number of callbacks")
	})
	t.Run("walk callback error", func(t *testing.T) {
		t.Parallel()
		tree := makeObjectTree(t, map[string]string{
			"foo": "bar",
		})
		treeReader := NewTreeReader(tree)
		expectedErr := errors.New("it was not to be")
		err := treeReader.Walk(
			context.Background(),
			"",
			func(info storage.ObjectInfo) error {
				return expectedErr
			},
		)
		assert.Equal(t, expectedErr, err)
	})
}

type file struct {
	name    string
	content string
}

type tree struct {
	files []file
	dirs  map[string]*tree
}

// putFile places a file into the given tree
func putFile(root *tree, name, content string) error {
	dirname, basename := path.Split(name)
	dirname = path.Clean(dirname)
	if dirname == ".." {
		return errors.New("escaping beyond root is unsupported")
	}
	if dirname == "." {
		root.files = append(root.files, file{name: name, content: content})
		return nil
	}
	treename, remaining, _ := strings.Cut(dirname, "/")
	subtree, ok := root.dirs[treename]
	if !ok {
		subtree = &tree{dirs: make(map[string]*tree)}
		root.dirs[treename] = subtree
	}
	return putFile(subtree, path.Join(remaining, basename), content)
}

// makeTree builds a tree from a map representing files.
func makeTree(files map[string]string) (*tree, error) {
	root := &tree{
		files: nil,
		dirs:  make(map[string]*tree),
	}
	for filename, content := range files {
		if err := putFile(root, path.Clean(filename), content); err != nil {
			return nil, err
		}
	}
	return root, nil
}

// storeTree encodes tree as a tree object into store. The hash of the tree is
// returned.
func storeTree(
	store storer.EncodedObjectStorer,
	t *tree,
) (plumbing.Hash, error) {
	// Store directories.
	var entries []object.TreeEntry
	for dirname, dir := range t.dirs {
		hash, err := storeTree(store, dir)
		if err != nil {
			return plumbing.ZeroHash, err
		}
		entries = append(entries, object.TreeEntry{
			Name: dirname,
			Mode: filemode.Dir,
			Hash: hash,
		})
	}

	// Store files.
	for _, f := range t.files {
		var obj plumbing.MemoryObject
		content := []byte(f.content)
		obj.SetType(plumbing.BlobObject)
		obj.SetSize(int64(len(content)))
		n, err := obj.Write(content)
		if err != nil {
			return plumbing.ZeroHash, err
		}
		if n != len(content) {
			err = fmt.Errorf("short write: %d of %d bytes", n, len(content))
			return plumbing.ZeroHash, err
		}
		hash, err := store.SetEncodedObject(&obj)
		if err != nil {
			return plumbing.ZeroHash, err
		}
		entries = append(entries, object.TreeEntry{
			Name: f.name,
			Mode: filemode.Regular,
			Hash: hash,
		})
	}

	// Create the tree object and store it.
	tree := &object.Tree{Entries: entries}
	treeObject := store.NewEncodedObject()
	err := tree.Encode(treeObject)
	if err != nil {
		return plumbing.ZeroHash, err
	}
	hash, err := store.SetEncodedObject(treeObject)
	if err != nil {
		return plumbing.ZeroHash, err
	}
	return hash, nil
}

// makeObjectTree builds a go-git tree object in memory backed storage from a
// map representing files.
func makeObjectTree(t *testing.T, files map[string]string) *object.Tree {
	// Project files into a more useful tree structure.
	tree, err := makeTree(files)
	require.NoError(t, err)
	// Put it into in-memory go-git storage.
	store := memory.NewStorage()
	hash, err := storeTree(store, tree)
	require.NoError(t, err)
	// Read the tree from storage. This ties a tree to it's underlying storage.
	treeObj, err := object.GetTree(store, hash)
	require.NoError(t, err)
	return treeObj
}

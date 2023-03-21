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

	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// pathinfo provides a storage.ObjectInfo for a single path
type pathinfo string

func (path pathinfo) Path() string {
	return string(path)
}

func (path pathinfo) ExternalPath() string {
	return string(path)
}

var _ storage.ObjectInfo = pathinfo("")

// namedReader provides a storage.ReadCloser from an io.ReadCloser with an
// associated storage.ObjectInfo.
type namedReader struct {
	info   storage.ObjectInfo
	reader io.ReadCloser
}

var _ storage.ReadObjectCloser = (*namedReader)(nil)

func (br *namedReader) Path() string {
	return br.info.Path()
}
func (br *namedReader) ExternalPath() string {
	return br.info.ExternalPath()
}

func (br *namedReader) Read(p []byte) (n int, err error) {
	return br.reader.Read(p)
}

func (br *namedReader) Close() error {
	return br.reader.Close()
}

// TreeReader exposes a go-git Tree as a storage.ReadBucket.
type TreeReader struct {
	tree *object.Tree
}

var _ storage.ReadBucket = (*TreeReader)(nil)

// NewTreeReader constructs a TreeReader from a go-git Tree.
func NewTreeReader(tree *object.Tree) *TreeReader {
	return &TreeReader{
		tree: tree,
	}
}

// entry locates an entry for the given path.
// storage.NewErrorNotExist is returned if the path is not found.
func (tr *TreeReader) entry(path string) (*object.TreeEntry, error) {
	treeEntry, err := tr.tree.FindEntry(path)
	if err != nil {
		if errors.Is(err, object.ErrEntryNotFound) {
			return nil, storage.NewErrNotExist(path)
		}
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return treeEntry, nil
}

// open returns a named reader for a path.
// storage.NewErrorNotExist is returned if the path is not found.
func (tr *TreeReader) open(path string) (*namedReader, error) {
	entry, err := tr.entry(path)
	if err != nil {
		return nil, err
	}
	file, err := tr.tree.TreeEntryFile(entry)
	if err != nil {
		return nil, err
	}
	reader, err := file.Blob.Reader()
	if err != nil {
		return nil, err
	}
	return &namedReader{
		info:   pathinfo(entry.Name),
		reader: reader,
	}, nil
}

func (tr *TreeReader) Get(
	_ context.Context,
	path string,
) (storage.ReadObjectCloser, error) {
	return tr.open(path)
}

func (tr *TreeReader) Stat(
	_ context.Context,
	path string,
) (storage.ObjectInfo, error) {
	entry, err := tr.entry(path)
	if err != nil {
		return nil, err
	}
	return pathinfo(entry.Name), nil
}

func (tr *TreeReader) Walk(
	_ context.Context,
	prefix string,
	callback func(storage.ObjectInfo) error,
) error {
	prefix = path.Clean(prefix)
	root, err := tr.subdir(prefix)
	if err != nil {
		// No directory means no files to walk.
		if errors.Is(err, object.ErrDirectoryNotFound) {
			return nil
		}
		return err
	}
	walker := object.NewTreeWalker(
		root,
		false,
		make(map[plumbing.Hash]bool),
	)
	defer walker.Close()
	for {
		_, treeEntry, err := walker.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		switch treeEntry.Mode {
		case filemode.Empty:
			// Zero value shouldn't occur.
		case filemode.Dir:
			// Directories are not processed with Walk.
		case filemode.Submodule:
			// Not supported.
		case filemode.Symlink:
			// Not supported.
		case filemode.Regular, filemode.Deprecated, filemode.Executable:
			err = callback(pathinfo(path.Join(prefix, treeEntry.Name)))
			if err != nil {
				return err
			}
		}
	}
}

func (tr *TreeReader) subdir(dir string) (*object.Tree, error) {
	if dir == "." {
		return tr.tree, nil
	}
	subtree, err := tr.tree.Tree(dir)
	if err != nil {
		return nil, err
	}
	return subtree, nil
}

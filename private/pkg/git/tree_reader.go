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
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/bufbuild/buf/private/pkg/git/object"
	"github.com/bufbuild/buf/private/pkg/storage"
)

// TreeReader exposes a go-git Tree as a storage.ReadBucket.
type TreeReader struct {
	objects ObjectService
	finder  *TreeFinder
}

var _ storage.ReadBucket = (*TreeReader)(nil)

// NewTreeReader constructs a TreeReader from a git Tree.
func NewTreeReader(objects ObjectService, root object.ID) *TreeReader {
	return &TreeReader{
		objects: objects,
		finder:  NewTreeFinder(objects, root),
	}
}

// entry locates an entry for the given path.
// storage.NewErrorNotExist is returned if the path is not found.
func (tr *TreeReader) entry(path string) (*object.TreeEntry, error) {
	treeEntry, err := tr.finder.FindEntry(path)
	if err != nil {
		if errors.Is(err, ErrEntryNotFound) {
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
	blob, err := tr.objects.Blob(entry.ID)
	if err != nil {
		return nil, err
	}
	return &namedReader{
		info:   pathinfo(entry.Name),
		reader: bytes.NewReader(blob),
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
	_, err := tr.entry(path)
	if err != nil {
		return nil, err
	}
	return pathinfo(path), nil
}

func (tr *TreeReader) Walk(
	_ context.Context,
	prefix string,
	callback func(storage.ObjectInfo) error,
) error {
	err := tr.finder.Range(prefix,
		func(entryPath string, entry *object.TreeEntry) error {
			switch entry.Mode {
			case object.ModeUnknown:
				return fmt.Errorf("unknown entry at %q", entryPath)
			case object.ModeSubmodule:
				// Submodules are not supported.
			case object.ModeSymlink:
				// Buckets do not support symlinks.
			case object.ModeDir:
				// Skip directories.
			case object.ModeFile, object.ModeExe:
				return callback(pathinfo(entryPath))
			}
			return nil
		},
	)
	if err != nil && !errors.Is(err, ErrEntryNotFound) {
		// No directory means no files to walk.
		return err
	}
	return nil
}

// pathinfo provides a storage.ObjectInfo for a single path
type pathinfo string

func (path pathinfo) Path() string {
	return string(path)
}

func (path pathinfo) ExternalPath() string {
	return string(path)
}

var _ storage.ObjectInfo = pathinfo("")

// namedReader provides a storage.ReadCloser from an io.Reader with an
// associated storage.ObjectInfo.
type namedReader struct {
	info   storage.ObjectInfo
	reader io.Reader
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
	// No operation because object storage doesn't stream (yet).
	return nil
}

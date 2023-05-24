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

package storagegit

import (
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageutil"
)

type bucket struct {
	objectReader git.ObjectReader
	symlinks     bool
	root         git.Tree
}

func newBucket(
	objectReader git.ObjectReader,
	symlinksIfSupported bool,
	root git.Tree,
) (storage.ReadBucket, error) {
	return &bucket{
		objectReader: objectReader,
		symlinks:     symlinksIfSupported,
		root:         root,
	}, nil
}

func (b *bucket) Get(ctx context.Context, path string) (storage.ReadObjectCloser, error) {
	node, err := b.root.Traverse(b.objectReader, normalpath.Components(path)...)
	if err != nil {
		if errors.Is(err, git.ErrSubTreeNotFound) {
			return nil, storage.NewErrNotExist(path)
		}
		return nil, err
	}
	switch node.Mode() {
	case git.ModeFile, git.ModeExe:
		data, err := b.objectReader.Blob(node.Hash())
		if err != nil {
			return nil, err
		}
		return &namedReader{
			info:   b.newObjectInfo(path),
			reader: bytes.NewReader(data),
		}, nil
	case git.ModeSymlink:
		if !b.symlinks {
			return nil, storage.NewErrNotExist(path)
		}
		// Symlinks are stored as blobs that reference the target path as a relative
		// path. We can follow this symlink trivially.
		data, err := b.objectReader.Blob(node.Hash())
		if err != nil {
			return nil, err
		}
		path = normalpath.Join(
			normalpath.Base(path),
			normalpath.Normalize(string(data)),
		)
		return b.Get(ctx, path)
	default:
		return nil, storage.NewErrNotExist(path)
	}
}

func (b *bucket) Stat(ctx context.Context, path string) (storage.ObjectInfo, error) {
	node, err := b.root.Traverse(b.objectReader, normalpath.Components(path)...)
	if err != nil {
		if errors.Is(err, git.ErrSubTreeNotFound) {
			return nil, storage.NewErrNotExist(path)
		}
		return nil, err
	}
	switch node.Mode() {
	case git.ModeFile, git.ModeExe:
		return b.newObjectInfo(path), nil
	case git.ModeSymlink:
		if !b.symlinks {
			return nil, storage.NewErrNotExist(path)
		}
		return b.newObjectInfo(path), nil
	default:
		// TODO: should this be an error? What kind of error can we throw here?
		return nil, storage.NewErrNotExist(path)
	}
}

func (b *bucket) Walk(ctx context.Context, prefix string, f func(storage.ObjectInfo) error) error {
	walkChecker := storageutil.NewWalkChecker()
	return b.walk(b.root, b.objectReader, prefix, func(path string, te git.Node) error {
		if err := walkChecker.Check(ctx); err != nil {
			return err
		}
		return f(b.newObjectInfo(path))
	})
}

func (b *bucket) walk(
	root git.Tree,
	objectReader git.ObjectReader,
	prefix string,
	walkFn func(string, git.Node) error,
) error {
	prefix = normalpath.Normalize(prefix)
	if prefix != "." {
		node, err := b.root.Traverse(b.objectReader, normalpath.Components(prefix)...)
		if err != nil {
			if errors.Is(err, git.ErrSubTreeNotFound) {
				return storage.NewErrNotExist(prefix)
			}
			return err
		}
		subTree, err := b.objectReader.Tree(node.Hash())
		if err != nil {
			return err
		}
		root = subTree
	}
	return b.walkTree(root, objectReader, prefix, walkFn)
}

func (b *bucket) walkTree(
	root git.Tree,
	objectReader git.ObjectReader,
	prefix string,
	walkFn func(string, git.Node) error,
) error {
	for _, node := range root.Nodes() {
		path := normalpath.Join(prefix, node.Name())
		switch node.Mode() {
		case git.ModeFile, git.ModeExe:
			if err := walkFn(path, node); err != nil {
				return err
			}
		case git.ModeSymlink:
			if b.symlinks {
				if err := walkFn(path, node); err != nil {
					return err
				}
			}
		case git.ModeDir:
			subTree, err := objectReader.Tree(node.Hash())
			if err != nil {
				return err
			}
			if err := b.walkTree(subTree, objectReader, path, walkFn); err != nil {
				return err
			}
		case git.ModeSubmodule, git.ModeUnknown:
			// ignored
		}
	}
	return nil
}

func (b *bucket) newObjectInfo(path string) storage.ObjectInfo {
	return storageutil.NewObjectInfo(
		normalpath.Normalize(path),
		normalpath.Unnormalize(path),
	)
}

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
	return nil
}

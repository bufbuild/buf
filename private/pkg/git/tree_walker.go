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
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/bufbuild/buf/private/pkg/git/object"
)

// ErrEntryNotFound is returned when an entry of any type is not found.
var ErrEntryNotFound = errors.New("not found")

// ErrDirEntryNotFound is returned when an ModeDir entry is not found. It
// wraps ErrEntryNotFound.
var ErrDirEntryNotFound = fmt.Errorf("dir %w", ErrEntryNotFound)

// TreeFinder finds or ranges over entries in a tree.
type TreeFinder struct {
	objects ObjectService
	root    object.ID
}

// NewTreeFinder binds storage with a tree identifier to create a TreeWalker.
func NewTreeFinder(objects ObjectService, tree object.ID) *TreeFinder {
	return &TreeFinder{objects: objects, root: tree}
}

// FindEntry locates an object from a given path name. For example, "foo/bar"
// locates a tree named "foo" and then locates an entry named "bar" in foo's
// tree.
func (tw *TreeFinder) FindEntry(name string) (*object.TreeEntry, error) {
	name = path.Clean(name)
	if name == "/" || name == "." {
		// name is a path to the root
		return &object.TreeEntry{
			Name: name,
			Mode: object.ModeDir,
			ID:   tw.root,
		}, nil
	}
	tree, err := tw.objects.Tree(tw.root)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(name, "/")
	entry, err := tw.find(tree, parts)
	if err != nil {
		return nil, fmt.Errorf("%w: %q", err, name)
	}
	return entry, nil
}

// Range iterates from the name path in root and through all its decendends,
// calling callback with each reachable TreeEntry. name may be "", ".", or "/"
// to range strating from the root. [ErrDirNotFound] is returned if name points
// to a valid entry but is not a directory.
//
// Iteration stops and returns if the the callback returns an error.
func (tw *TreeFinder) Range(
	name string,
	callback func(filepath string, entry *object.TreeEntry) error,
) error {
	dir, err := tw.FindEntry(name)
	if err != nil {
		return err
	}
	if dir.Mode != object.ModeDir {
		return ErrDirEntryNotFound
	}
	return tw.ranger(dir, dir.Name, callback)
}

func (tw *TreeFinder) ranger(
	dir *object.TreeEntry,
	dirname string,
	callback func(filepath string, entry *object.TreeEntry) error,
) error {
	// This directory's entry.
	if err := callback(dirname, dir); err != nil {
		return err
	}
	// Iterate over non-dir objects
	dirents, err := tw.objects.Tree(dir.ID)
	if err != nil {
		return err
	}
	dirstack := make([]*object.TreeEntry, 0, len(dirents.Entries))
	for i := range dirents.Entries {
		entry := &dirents.Entries[i]
		if entry.Mode == object.ModeDir {
			dirstack = append(dirstack, entry)
		} else {
			filePath := path.Join(dirname, entry.Name)
			if err := callback(filePath, entry); err != nil {
				return err
			}
		}
	}
	// Dive into the dirs.
	for _, subdir := range dirstack {
		subdirPath := path.Join(dirname, subdir.Name)
		if err := tw.ranger(subdir, subdirPath, callback); err != nil {
			return err
		}
	}
	return nil
}

func (tw *TreeFinder) find(
	tree *object.Tree,
	names []string,
) (*object.TreeEntry, error) {
	name := names[0]
	names = names[1:]
	// Find name in this tree.
	var entry *object.TreeEntry
	for i := range tree.Entries {
		ent := &tree.Entries[i]
		if ent.Name == name {
			entry = ent
			break
		}
	}
	if entry == nil {
		// No name in this tree.
		return nil, ErrEntryNotFound
	}
	if len(names) == 0 {
		// We found it.
		return entry, nil
	}
	if entry.Mode != object.ModeDir {
		// Part of the path is not a directory.
		return nil, ErrEntryNotFound
	}
	// Walk down the tree.
	tree, err := tw.objects.Tree(entry.ID)
	if err != nil {
		return nil, err
	}
	return tw.find(tree, names)
}

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
	"testing"

	"github.com/bufbuild/buf/private/pkg/git/gittest"
	"github.com/bufbuild/buf/private/pkg/git/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testStore(t *testing.T) (
	memstore ObjectService,
	root, ponies, dashie object.ID,
) {
	memStore := gittest.NewMemObjectStore()
	dashie, err := memStore.PutBlob([]byte("best"))
	require.NoError(t, err)
	ponies, err = memStore.PutTree(&object.Tree{
		Entries: []object.TreeEntry{
			{
				Name: "dashie",
				Mode: object.ModeFile,
				ID:   dashie,
			},
		},
	})
	require.NoError(t, err)
	root, err = memStore.PutTree(&object.Tree{
		Entries: []object.TreeEntry{
			{
				Name: "ponies",
				Mode: object.ModeDir,
				ID:   ponies,
			},
		},
	})
	require.NoError(t, err)
	return memStore, root, ponies, dashie
}

func TestFindEntry(t *testing.T) {
	store, root, ponies, dashie := testStore(t)
	testFindEntry(t, store, root, "found file",
		"ponies/dashie",
		false,
		&object.TreeEntry{
			Name: "dashie",
			Mode: object.ModeFile,
			ID:   dashie,
		},
	)
	testFindEntry(t, store, root, "file not found",
		"ponies/pikachu",
		true,
		nil,
	)
	testFindEntry(t, store, root, "found tree",
		"ponies",
		false,
		&object.TreeEntry{
			Name: "ponies",
			Mode: object.ModeDir,
			ID:   ponies,
		},
	)
	testFindEntry(t, store, root, "root entry",
		".",
		false,
		&object.TreeEntry{
			Name: ".",
			Mode: object.ModeDir,
			ID:   root,
		},
	)
	testFindEntry(t, store, root, "absolute path to root",
		"/",
		false,
		&object.TreeEntry{
			Name: "/",
			Mode: object.ModeDir,
			ID:   root,
		},
	)
	testFindEntry(t, store, root, "convoluted path to root",
		"./ponies/..",
		false,
		&object.TreeEntry{
			Name: ".",
			Mode: object.ModeDir,
			ID:   root,
		},
	)
	testFindEntry(t, store, root, "path through a file",
		"ponies/dashie/rainbow",
		true,
		nil,
	)
}

func TestRange(t *testing.T) {
	t.Parallel()
	store, root, _, _ := testStore(t)
	tw := NewTreeFinder(store, root)
	err := tw.Range("", func(path string, entry *object.TreeEntry) error {
		switch path {
		case "ponies":
		case "ponies/dashie":
		case ".":
		default:
			return fmt.Errorf("unexpected path: %q", path)
		}
		switch entry.Name {
		case "ponies":
		case "dashie":
		case ".":
		default:
			return fmt.Errorf("unexpected basename: %q", entry.Name)
		}
		return nil
	})
	assert.NoError(t, err)
	err = tw.Range("", func(_ string, entry *object.TreeEntry) error {
		if entry.Mode == object.ModeDir {
			return errors.New("explode in a dir")
		}
		return nil
	})
	assert.Error(t, err)
	err = tw.Range("", func(_ string, entry *object.TreeEntry) error {
		if entry.Mode == object.ModeFile {
			return errors.New("explode in a file")
		}
		return nil
	})
	assert.Error(t, err)
}

func testFindEntry(
	t *testing.T,
	store ObjectService,
	root object.ID,
	desc string,
	path string,
	expectErr bool,
	expectedEntry *object.TreeEntry,
) {
	t.Run(desc, func(t *testing.T) {
		t.Parallel()
		tw := NewTreeFinder(store, root)
		entry, err := tw.FindEntry(path)
		if expectErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
		assert.Equal(t, expectedEntry, entry)
	})
}

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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/git/gittest"
	"github.com/bufbuild/buf/private/pkg/git/object"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreeReader(t *testing.T) {
	t.Run("file exists", func(t *testing.T) {
		t.Parallel()
		objects, tree, err := gittest.NewMemObjectStoreFromMap(map[string]string{
			"foo": "bar",
		})
		require.NoError(t, err)
		treeReader := NewTreeReader(objects, tree)
		info, err := treeReader.Stat(context.Background(), "foo")
		require.NoError(t, err)
		assert.Equal(t, "foo", info.Path())
	})
	t.Run("file doesn't exist", func(t *testing.T) {
		t.Parallel()
		objects, tree, err := gittest.NewMemObjectStoreFromMap(nil)
		require.NoError(t, err)
		treeReader := NewTreeReader(objects, tree)
		_, err = treeReader.Stat(context.Background(), "foo")
		assert.True(t, storage.IsNotExist(err))
	})
	t.Run("read foo", func(t *testing.T) {
		t.Parallel()
		objects, tree, err := gittest.NewMemObjectStoreFromMap(map[string]string{
			"foo": "bar",
		})
		require.NoError(t, err)
		treeReader := NewTreeReader(objects, tree)
		file, err := treeReader.Get(context.Background(), "foo")
		require.NoError(t, err)
		bytes, err := io.ReadAll(file)
		require.NoError(t, err)
		assert.Equal(t, "foo", file.Path())
		assert.Equal(t, "bar", string(bytes))
	})
	t.Run("walk", func(t *testing.T) {
		t.Parallel()
		objects, tree, err := gittest.NewMemObjectStoreFromMap(map[string]string{
			"foo": "bar",
			"baz": "qux",
		})
		require.NoError(t, err)
		treeReader := NewTreeReader(objects, tree)
		count := 0
		err = treeReader.Walk(
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
		objects, tree, err := gittest.NewMemObjectStoreFromMap(map[string]string{
			"foo": "bar",
		})
		require.NoError(t, err)
		treeReader := NewTreeReader(objects, tree)
		count := 0
		err = treeReader.Walk(
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
		objects, tree, err := gittest.NewMemObjectStoreFromMap(map[string]string{
			"foo":     "bar",
			"dir/baz": "qux",
		})
		require.NoError(t, err)
		treeReader := NewTreeReader(objects, tree)
		count := 0
		err = treeReader.Walk(
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
		objects, tree, err := gittest.NewMemObjectStoreFromMap(map[string]string{
			"foo": "bar",
		})
		require.NoError(t, err)
		treeReader := NewTreeReader(objects, tree)
		expectedErr := errors.New("it was not to be")
		err = treeReader.Walk(
			context.Background(),
			"",
			func(info storage.ObjectInfo) error {
				return expectedErr
			},
		)
		assert.Equal(t, expectedErr, err)
	})
	t.Run("walk objectinfo has correct paths", func(t *testing.T) {
		t.Parallel()
		objects, tree, err := gittest.NewMemObjectStoreFromMap(map[string]string{
			"foo":     "bar",
			"dir/baz": "qux",
		})
		require.NoError(t, err)
		treeReader := NewTreeReader(objects, tree)
		ctx := context.Background()
		var paths []string
		err = treeReader.Walk(ctx, "", func(info storage.ObjectInfo) error {
			paths = append(paths, info.Path())
			return nil
		})
		assert.NoError(t, err)
		assert.ElementsMatch(t, []string{"foo", "dir/baz"}, paths)
	})
}

func TestStorageIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("git storage integration is slow")
	}
	t.Parallel()
	// Construct a git repository.
	dir := t.TempDir()
	runner := command.NewRunner()
	git := gittest.NewGitCmd(t, runner, gittest.GitCmdInit(dir))
	git.Cmd("config", "--local", "user.name", "buftest")
	git.Cmd("config", "--local", "user.email", "buftest@example.com")
	// Produce a commit with a file.
	path := "image.proto"
	testProto, err := os.Create(filepath.Join(dir, path))
	require.NoError(t, err)
	testProto.Close()
	git.Cmd("add", path)
	git.Cmd("commit", "-m", "msg")
	rootHash := git.Cmd("rev-parse", "HEAD")
	rootHash = strings.TrimRight(rootHash, "\n")
	var commitref object.ID
	err = commitref.UnmarshalText([]byte(rootHash))
	require.NoError(t, err)
	// Start an object service.
	catfile, err := NewCatFile(
		runner,
		CatFileGitDir(filepath.Join(dir, ".git")),
	)
	require.NoError(t, err)
	objects, err := catfile.Connect()
	require.NoError(t, err)
	defer func() { assert.NoError(t, objects.Close()) }()
	require.NoError(t, err)
	// Load the commit and find our file.
	commit, err := objects.Commit(commitref)
	require.NoError(t, err)
	treereader := NewTreeReader(objects, commit.Tree)
	ctx := context.Background()
	info, err := treereader.Stat(ctx, path)
	require.NoError(t, err)
	assert.Equal(t, path, info.Path())
}

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
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/git/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gitCmd wraps calling out to the host's "git"
type gitCmd struct {
	t       *testing.T
	runner  command.Runner
	gitdir  string
	timeout time.Duration
	env     map[string]string
}

func newGitCmd(t *testing.T, runner command.Runner, gitdir string) *gitCmd {
	return &gitCmd{
		t:       t,
		runner:  runner,
		gitdir:  gitdir,
		timeout: 5 * time.Second,
	}
}

func (g *gitCmd) Env(env map[string]string) *gitCmd {
	git := *g
	git.env = env
	return &git
}

func (g *gitCmd) Cmd(args ...string) string {
	cmdEnv := map[string]string{
		"GIT_DIR": g.gitdir,
	}
	for k, v := range g.env {
		cmdEnv[k] = v
	}
	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	defer cancel()
	var stdout strings.Builder
	err := g.runner.Run(ctx,
		"git",
		command.RunWithArgs(args...),
		command.RunWithEnv(cmdEnv),
		command.RunWithStdout(io.MultiWriter(&stdout, os.Stdout)),
		command.RunWithStderr(os.Stderr),
	)
	if err != nil {
		argStr := strings.Join(args, " ")
		g.t.Fatalf("`git %s`: %s", argStr, err)
	}
	return stdout.String()
}

func TestCatFileGitDir(t *testing.T) {
	dir := t.TempDir()        // probing is only on a plain dir
	var runner command.Runner // runner shouldn't be touched in construction
	_, err := NewCatFile(runner, CatFileGitDir(filepath.Join(dir, "none")))
	assert.Error(t, err)
	_, err = NewCatFile(runner, CatFileGitDir(dir))
	assert.NoError(t, err)
}

func TestCatFileIntegration(t *testing.T) {
	if testing.Short() {
		// This test builds a git repo and spawns a live git-cat-file process.
		t.Skip("skipping git-cat-file integration test")
	}
	// Construct a git repository.
	dir := t.TempDir()
	runner := command.NewRunner()
	git := newGitCmd(t, runner, dir)
	git.Cmd("init", "--bare")
	git.Cmd("config", "--local", "user.name", "buftest")
	git.Cmd("config", "--local", "user.email", "buftest@example.com")
	// produces a root commit
	rootHash := git.Env(map[string]string{
		"GIT_AUTHOR_DATE":    "2000-01-01T00:00:00",
		"GIT_COMMITTER_DATE": "2000-01-01T00:00:00",
	}).Cmd(
		"commit-tree",
		"-m", "msg",
		"4b825dc642cb6eb9a060e54bf8d69288fbee4904", // zero tree
	)
	rootHash = strings.TrimRight(rootHash, "\n")
	// produces a descendent from the root
	secondHash := git.Env(map[string]string{
		"GIT_AUTHOR_DATE":    "2000-01-01T00:00:00",
		"GIT_COMMITTER_DATE": "2000-01-01T00:00:00",
	}).Cmd(
		"commit-tree",
		"-m", "different msg",
		"-p", rootHash,
		"4b825dc642cb6eb9a060e54bf8d69288fbee4904", // zero tree
	)
	secondHash = strings.TrimRight(secondHash, "\n")
	catfile, err := NewCatFile(runner, CatFileGitDir(dir))
	require.NoError(t, err)
	objects, err := catfile.Connect()
	require.NoError(t, err)
	// root commit
	firstCommit := mustID(t, rootHash)
	commit, err := objects.Commit(firstCommit)
	require.NoError(t, err)
	assert.Equal(t,
		mustID(t, "4b825dc642cb6eb9a060e54bf8d69288fbee4904"),
		commit.Tree,
	)
	assert.Nil(t, commit.Parents)
	assert.Equal(t, "buftest", commit.Author.Name)
	assert.Equal(t, "buftest@example.com", commit.Author.Email)
	assert.Equal(t, "msg\n", commit.Message)
	// second commit
	secondCommit := mustID(t, secondHash)
	commit, err = objects.Commit(secondCommit)
	require.NoError(t, err)
	assert.Equal(t,
		mustID(t, "4b825dc642cb6eb9a060e54bf8d69288fbee4904"),
		commit.Tree,
	)
	assert.Equal(t, []object.ID{firstCommit}, commit.Parents)
	assert.Equal(t, "buftest", commit.Author.Name)
	assert.Equal(t, "buftest@example.com", commit.Author.Email)
	assert.Equal(t, "different msg\n", commit.Message)
	assert.NoError(t, objects.Close())
}

func mustID(t *testing.T, hexid string) (objID object.ID) {
	err := objID.UnmarshalText([]byte(hexid))
	require.NoError(t, err)
	return objID
}

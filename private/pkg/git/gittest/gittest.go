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
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/stretchr/testify/require"
)

const (
	DefaultBranch = "master"
	DefaultRemote = "origin"
)

type Repository interface {
	git.Repository
	Commit(t *testing.T, msg string, files map[string]string, opts ...CommitOption)
	Checkout(t *testing.T, branch string)
	CheckoutB(t *testing.T, branch string)
	Tag(t *testing.T, name string, msg string)
	Push(t *testing.T)
	Merge(t *testing.T, branch string)
	PackRefs(t *testing.T)
}

type commitOpts struct {
	executablePaths []string
}

type CommitOption func(*commitOpts)

func CommitWithExecutableFile(path string) CommitOption {
	return func(opts *commitOpts) {
		opts.executablePaths = append(opts.executablePaths, path)
	}
}

func ScaffoldGitRepository(t *testing.T) Repository {
	runner := command.NewRunner()
	dir := scaffoldGitRepository(t, runner, DefaultBranch)
	dotGitPath := filepath.Join(dir, git.DotGitDir)
	repo, err := git.OpenRepository(
		context.Background(),
		dotGitPath,
		runner,
		git.OpenRepositoryWithDefaultBranch(DefaultBranch),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, repo.Close())
	})
	testRepo := newRepository(
		repo,
		dir,
		runner,
	)
	return testRepo
}

func scaffoldGitRepository(t *testing.T, runner command.Runner, defaultBranch string) string {
	dir := t.TempDir()

	// (0) setup local and remote
	runInDir(t, runner, dir, "mkdir", "local", "remote")
	remoteDir := filepath.Join(dir, "remote")
	runInDir(t, runner, remoteDir, "git", "init", "--bare")
	runInDir(t, runner, remoteDir, "git", "config", "user.name", "Buf TestBot")
	runInDir(t, runner, remoteDir, "git", "config", "user.email", "testbot@buf.build")
	localDir := filepath.Join(dir, "local")
	runInDir(t, runner, localDir, "git", "init")
	runInDir(t, runner, localDir, "git", "config", "user.name", "Buf TestBot")
	runInDir(t, runner, localDir, "git", "config", "user.email", "testbot@buf.build")
	runInDir(t, runner, localDir, "git", "remote", "add", DefaultRemote, remoteDir)

	// (1) initial commit and push
	writeFiles(t, localDir, map[string]string{"README.md": "This is a scaffold repository.\n"})
	runInDir(t, runner, localDir, "git", "add", ".")
	runInDir(t, runner, localDir, "git", "commit", "-m", "initial commit")
	runInDir(t, runner, localDir, "git", "push", "-u", "-f", "origin", defaultBranch)

	return localDir
}

func runInDir(t *testing.T, runner command.Runner, dir string, cmd string, args ...string) {
	stderr := bytes.NewBuffer(nil)
	err := runner.Run(
		context.Background(),
		cmd,
		command.RunWithArgs(args...),
		command.RunWithDir(dir),
		command.RunWithStderr(stderr),
	)
	if err != nil {
		require.FailNow(
			t,
			fmt.Sprintf(
				"git command failed: %q: %s",
				strings.Join(append([]string{cmd}, args...), " "),
				err.Error(),
			),
			stderr,
		)
	}
}

func writeFiles(t *testing.T, dir string, files map[string]string) {
	for path, contents := range files {
		path := normalpath.Unnormalize(path)
		require.NoError(t, os.MkdirAll(filepath.Join(dir, filepath.Dir(path)), 0700))
		require.NoError(t, os.WriteFile(filepath.Join(dir, path), []byte(contents), 0600))
	}
}

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

package bufsync

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/stretchr/testify/require"
)

// scaffoldGitRepository returns a repo with the following commits:
// | o-o----------o-----------------o (master)
// |   └o-o (foo) └o--------o (bar)
// |               └o (baz)
func scaffoldGitRepository(t *testing.T, moduleIdentity bufmoduleref.ModuleIdentity) git.Repository {
	runner := command.NewRunner()
	dir := scaffoldGitRepositoryDir(t, runner, moduleIdentity)
	dotGitPath := path.Join(dir, git.DotGitDir)
	repo, err := git.OpenRepository(
		context.Background(),
		dotGitPath,
		runner,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, repo.Close())
	})
	return repo
}

func scaffoldGitRepositoryDir(t *testing.T, runner command.Runner, moduleIdentity bufmoduleref.ModuleIdentity) string {
	dir := t.TempDir()

	// setup local and remote
	runInDir(t, runner, dir, "mkdir", "local", "remote")
	remoteDir := path.Join(dir, "remote")
	runInDir(t, runner, remoteDir, "git", "init", "--bare")
	runInDir(t, runner, remoteDir, "git", "config", "user.name", "Buf TestBot")
	runInDir(t, runner, remoteDir, "git", "config", "user.email", "testbot@buf.build")
	localDir := path.Join(dir, "local")
	const defaultBranch = "main"
	runInDir(t, runner, localDir, "git", "init", "--initial-branch", defaultBranch)
	runInDir(t, runner, localDir, "git", "config", "user.name", "Buf TestBot")
	runInDir(t, runner, localDir, "git", "config", "user.email", "testbot@buf.build")

	var allBranches = []string{defaultBranch, "foo", "bar", "baz"}

	var commitsCounter int
	doEmptyCommit := func(numOfCommits int) {
		for i := 0; i < numOfCommits; i++ {
			commitsCounter++
			runInDir(
				t, runner, localDir,
				"git", "commit", "--allow-empty",
				"-m", fmt.Sprintf("commit %d", commitsCounter),
			)
		}
	}

	// write the base module in the root
	writeFiles(t, localDir, map[string]string{
		"buf.yaml": fmt.Sprintf("version: v1\nname: %s\n", moduleIdentity.IdentityString()),
	})
	runInDir(t, runner, localDir, "git", "add", ".")
	runInDir(t, runner, localDir, "git", "commit", "-m", "commit 0")

	doEmptyCommit(1)
	runInDir(t, runner, localDir, "git", "checkout", "-b", allBranches[1])
	doEmptyCommit(2)
	runInDir(t, runner, localDir, "git", "checkout", defaultBranch)
	doEmptyCommit(1)
	runInDir(t, runner, localDir, "git", "checkout", "-b", allBranches[2])
	doEmptyCommit(1)
	runInDir(t, runner, localDir, "git", "checkout", "-b", allBranches[3])
	doEmptyCommit(1)
	runInDir(t, runner, localDir, "git", "checkout", allBranches[2])
	doEmptyCommit(1)
	runInDir(t, runner, localDir, "git", "checkout", defaultBranch)
	doEmptyCommit(1)

	// push them all
	const remoteName = "origin"
	runInDir(t, runner, localDir, "git", "remote", "add", remoteName, remoteDir)
	for _, branch := range allBranches {
		runInDir(t, runner, localDir, "git", "checkout", branch)
		runInDir(t, runner, localDir, "git", "push", "-u", "-f", remoteName, branch)
	}

	// set a default remote branch
	runInDir(t, runner, localDir, "git", "remote", "set-head", remoteName, defaultBranch)

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
		t.Logf("run %q", strings.Join(append([]string{cmd}, args...), " "))
		_, err := io.Copy(os.Stderr, stderr)
		require.NoError(t, err)
	}
	require.NoError(t, err)
}

func writeFiles(t *testing.T, dir string, files map[string]string) {
	for path, contents := range files {
		require.NoError(t, os.MkdirAll(filepath.Join(dir, filepath.Dir(path)), 0700))
		require.NoError(t, os.WriteFile(filepath.Join(dir, path), []byte(contents), 0600))
	}
}

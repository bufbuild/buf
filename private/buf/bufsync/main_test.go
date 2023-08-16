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
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/stretchr/testify/require"
)

const (
	defaultBranch = "main"
	remoteName    = "origin"
)

// scaffoldGitRepository returns an initialized git repository with a single commit, and returns the
// repository and its directory.
func scaffoldGitRepository(t *testing.T) (git.Repository, string) {
	runner := command.NewRunner()
	repoDir := scaffoldGitRepositoryDir(t, runner)
	dotGitPath := path.Join(repoDir, git.DotGitDir)
	repo, err := git.OpenRepository(
		context.Background(),
		dotGitPath,
		runner,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, repo.Close())
	})
	return repo, repoDir
}

// scaffoldGitRepositoryDir prepares a git repository with an initial README, and a single commit
// pushed to a remote named origin. It returns the directory where the local git repo is.
func scaffoldGitRepositoryDir(t *testing.T, runner command.Runner) string {
	dir := t.TempDir()

	// setup local and remote
	runInDir(t, runner, dir, "mkdir", "local", "remote")
	remoteDir := path.Join(dir, "remote")
	runInDir(t, runner, remoteDir, "git", "init", "--bare")
	runInDir(t, runner, remoteDir, "git", "config", "user.name", "Buf TestBot")
	runInDir(t, runner, remoteDir, "git", "config", "user.email", "testbot@buf.build")
	localDir := path.Join(dir, "local")
	runInDir(t, runner, localDir, "git", "init", "--initial-branch", defaultBranch)
	runInDir(t, runner, localDir, "git", "config", "user.name", "Buf TestBot")
	runInDir(t, runner, localDir, "git", "config", "user.email", "testbot@buf.build")

	// write and commit a README file
	writeFiles(t, localDir, map[string]string{"README.md": "This is a scaffold repository.\n"})
	runInDir(t, runner, localDir, "git", "add", ".")
	runInDir(t, runner, localDir, "git", "commit", "--allow-empty", "-m", "Write README")

	// push to origin
	runInDir(t, runner, localDir, "git", "remote", "add", remoteName, remoteDir)
	runInDir(t, runner, localDir, "git", "push", "-u", "-f", remoteName, defaultBranch)

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

func writeFiles(t *testing.T, directoryPath string, pathToContents map[string]string) {
	for path, contents := range pathToContents {
		require.NoError(t, os.MkdirAll(filepath.Join(directoryPath, filepath.Dir(path)), 0700))
		require.NoError(t, os.WriteFile(filepath.Join(directoryPath, path), []byte(contents), 0600))
	}
}

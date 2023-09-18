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
	"net/http/cgi"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGitCloner(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container, err := app.NewContainerForOS()
	require.NoError(t, err)
	runner := command.NewRunner()
	originDir, workDir := createGitDirs(ctx, t, container, runner)

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, runner, workDir, 1, nil, false)

		content, err := storage.ReadPath(ctx, readBucket, "test.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 2", string(content), "expected the commit on local-branch to be checked out")
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.True(t, storage.IsNotExist(err))
		_, err = storage.ReadPath(ctx, readBucket, "submodule/test.proto")
		assert.True(t, storage.IsNotExist(err))
	})

	t.Run("default_submodule", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, runner, workDir, 1, nil, true)

		content, err := storage.ReadPath(ctx, readBucket, "test.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 2", string(content), "expected the commit on local-branch to be checked out")
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.True(t, storage.IsNotExist(err))
		content, err = storage.ReadPath(ctx, readBucket, "submodule/test.proto")
		require.NoError(t, err)
		assert.Equal(t, "// submodule", string(content))
	})

	t.Run("main", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, runner, workDir, 1, NewBranchName("main"), false)

		content, err := storage.ReadPath(ctx, readBucket, "test.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 1", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.True(t, storage.IsNotExist(err))
	})

	t.Run("origin/main", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, runner, workDir, 1, NewBranchName("origin/main"), false)

		content, err := storage.ReadPath(ctx, readBucket, "test.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 3", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.True(t, storage.IsNotExist(err))
	})

	t.Run("origin/remote-branch", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, runner, workDir, 1, NewBranchName("origin/remote-branch"), false)

		content, err := storage.ReadPath(ctx, readBucket, "test.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 4", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.True(t, storage.IsNotExist(err))
	})

	t.Run("remote-tag", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, runner, workDir, 1, NewTagName("remote-tag"), false)

		content, err := storage.ReadPath(ctx, readBucket, "test.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 4", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.True(t, storage.IsNotExist(err))
	})

	t.Run("branch_and_main_ref", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, runner, workDir, 2, NewRefNameWithBranch("HEAD~", "main"), false)

		content, err := storage.ReadPath(ctx, readBucket, "test.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 0", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.True(t, storage.IsNotExist(err))
	})

	t.Run("branch_and_ref", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, runner, workDir, 2, NewRefNameWithBranch("local-branch~", "local-branch"), false)

		content, err := storage.ReadPath(ctx, readBucket, "test.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 1", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.True(t, storage.IsNotExist(err))
	})

	t.Run("HEAD", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, runner, workDir, 1, NewRefName("HEAD"), false)

		content, err := storage.ReadPath(ctx, readBucket, "test.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 2", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.True(t, storage.IsNotExist(err))
	})

	t.Run("commit-local", func(t *testing.T) {
		t.Parallel()
		revParseBytes, err := command.RunStdout(ctx, container, runner, "git", "-C", workDir, "rev-parse", "HEAD~")
		require.NoError(t, err)
		readBucket := readBucketForName(ctx, t, runner, workDir, 2, NewRefName(strings.TrimSpace(string(revParseBytes))), false)

		content, err := storage.ReadPath(ctx, readBucket, "test.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 1", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.True(t, storage.IsNotExist(err))
	})

	t.Run("commit-remote", func(t *testing.T) {
		t.Parallel()
		revParseBytes, err := command.RunStdout(ctx, container, runner, "git", "-C", originDir, "rev-parse", "remote-branch~")
		require.NoError(t, err)
		readBucket := readBucketForName(ctx, t, runner, workDir, 2, NewRefNameWithBranch(strings.TrimSpace(string(revParseBytes)), "origin/remote-branch"), false)

		content, err := storage.ReadPath(ctx, readBucket, "test.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 3", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.True(t, storage.IsNotExist(err))
	})
}

func readBucketForName(ctx context.Context, t *testing.T, runner command.Runner, path string, depth uint32, name Name, recurseSubmodules bool) storage.ReadBucket {
	t.Helper()
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	cloner := NewCloner(zap.NewNop(), storageosProvider, runner, ClonerOptions{})
	envContainer, err := app.NewEnvContainerForOS()
	require.NoError(t, err)

	readWriteBucket := storagemem.NewReadWriteBucket()
	err = cloner.CloneToBucket(
		ctx,
		envContainer,
		"file://"+normalpath.Join(path, ".git"),
		depth,
		readWriteBucket,
		CloneToBucketOptions{
			Mapper:            storage.MatchPathExt(".proto"),
			Name:              name,
			RecurseSubmodules: recurseSubmodules,
		},
	)
	require.NoError(t, err)
	return readWriteBucket
}

func createGitDirs(
	ctx context.Context,
	t *testing.T,
	container app.EnvStdioContainer,
	runner command.Runner,
) (string, string) {
	tmpDir := t.TempDir()

	submodulePath := normalpath.Join(tmpDir, "submodule")
	require.NoError(t, os.MkdirAll(submodulePath, os.ModePerm))
	runCommand(ctx, t, container, runner, "git", "-C", submodulePath, "init")
	runCommand(ctx, t, container, runner, "git", "-C", submodulePath, "config", "user.email", "tests@buf.build")
	runCommand(ctx, t, container, runner, "git", "-C", submodulePath, "config", "user.name", "Buf go tests")
	runCommand(ctx, t, container, runner, "git", "-C", submodulePath, "checkout", "-b", "main")
	require.NoError(t, os.WriteFile(normalpath.Join(submodulePath, "test.proto"), []byte("// submodule"), 0600))
	runCommand(ctx, t, container, runner, "git", "-C", submodulePath, "add", "test.proto")
	runCommand(ctx, t, container, runner, "git", "-C", submodulePath, "commit", "-m", "commit 0")

	gitExecPath, err := command.RunStdout(ctx, container, runner, "git", "--exec-path")
	require.NoError(t, err)
	t.Log(normalpath.Join(string(gitExecPath), "git-http-backend"))
	// https://git-scm.com/docs/git-http-backend#_description
	f, err := os.Create(normalpath.Join(submodulePath, ".git", "git-daemon-export-ok"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	server := httptest.NewServer(&cgi.Handler{
		Path: normalpath.Join(strings.TrimSpace(string(gitExecPath)), "git-http-backend"),
		Dir:  submodulePath,
		Env:  []string{"GIT_PROJECT_ROOT=" + submodulePath},
	})
	t.Cleanup(server.Close)
	submodulePath = server.URL

	originPath := normalpath.Join(tmpDir, "origin")
	require.NoError(t, os.MkdirAll(originPath, 0777))
	runCommand(ctx, t, container, runner, "git", "-C", originPath, "init")
	runCommand(ctx, t, container, runner, "git", "-C", originPath, "config", "user.email", "tests@buf.build")
	runCommand(ctx, t, container, runner, "git", "-C", originPath, "config", "user.name", "Buf go tests")
	runCommand(ctx, t, container, runner, "git", "-C", originPath, "checkout", "-b", "main")
	require.NoError(t, os.WriteFile(normalpath.Join(originPath, "test.proto"), []byte("// commit 0"), 0600))
	runCommand(ctx, t, container, runner, "git", "-C", originPath, "add", "test.proto")
	runCommand(ctx, t, container, runner, "git", "-C", originPath, "commit", "-m", "commit 0")
	runCommand(ctx, t, container, runner, "git", "-C", originPath, "submodule", "add", submodulePath, "submodule")
	require.NoError(t, os.WriteFile(normalpath.Join(originPath, "test.proto"), []byte("// commit 1"), 0600))
	runCommand(ctx, t, container, runner, "git", "-C", originPath, "add", "test.proto")
	runCommand(ctx, t, container, runner, "git", "-C", originPath, "commit", "-m", "commit 1")

	workPath := normalpath.Join(tmpDir, "workdir")
	runCommand(ctx, t, container, runner, "git", "clone", originPath, workPath)
	runCommand(ctx, t, container, runner, "git", "-C", workPath, "config", "user.email", "tests@buf.build")
	runCommand(ctx, t, container, runner, "git", "-C", workPath, "config", "user.name", "Buf go tests")
	runCommand(ctx, t, container, runner, "git", "-C", workPath, "checkout", "-b", "local-branch")
	require.NoError(t, os.WriteFile(normalpath.Join(workPath, "test.proto"), []byte("// commit 2"), 0600))
	runCommand(ctx, t, container, runner, "git", "-C", workPath, "commit", "-a", "-m", "commit 2")

	require.NoError(t, os.WriteFile(normalpath.Join(originPath, "test.proto"), []byte("// commit 3"), 0600))
	runCommand(ctx, t, container, runner, "git", "-C", originPath, "add", "test.proto")
	runCommand(ctx, t, container, runner, "git", "-C", originPath, "commit", "-m", "commit 3")

	runCommand(ctx, t, container, runner, "git", "-C", originPath, "checkout", "-b", "remote-branch")
	require.NoError(t, os.WriteFile(normalpath.Join(originPath, "test.proto"), []byte("// commit 4"), 0600))
	runCommand(ctx, t, container, runner, "git", "-C", originPath, "add", "test.proto")
	runCommand(ctx, t, container, runner, "git", "-C", originPath, "commit", "-m", "commit 4")
	runCommand(ctx, t, container, runner, "git", "-C", originPath, "tag", "remote-tag")

	runCommand(ctx, t, container, runner, "git", "-C", workPath, "fetch", "origin")
	return originPath, workPath
}

func runCommand(
	ctx context.Context,
	t *testing.T,
	container app.EnvStdioContainer,
	runner command.Runner,
	name string,
	args ...string,
) {
	t.Helper()
	output, err := command.RunStdout(ctx, container, runner, name, args...)
	if err != nil {
		var exitErr *exec.ExitError
		var stdErr []byte
		if errors.As(err, &exitErr) {
			stdErr = exitErr.Stderr
		}
		assert.FailNow(t, err.Error(), "stdout: %s\nstderr: %s", output, stdErr)
	}
}

// Copyright 2020-2021 Buf Technologies, Inc.
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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGitCloner(t *testing.T) {
	t.Parallel()

	// Add git version to log for easier context of ci failures.
	gitVersion, err := exec.Command("git", "--version").CombinedOutput()
	require.NoError(t, err)
	t.Log(string(gitVersion))

	originDir, workDir := createGitDirs(t)

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(t, workDir, 1, nil, false)

		content, err := storage.ReadPath(context.Background(), readBucket, "test.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 2", string(content), "expected the commit on local-branch to be checked out")
		_, err = readBucket.Stat(context.Background(), "nonexistent")
		assert.True(t, storage.IsNotExist(err))
		_, err = storage.ReadPath(context.Background(), readBucket, "submodule/test.proto")
		assert.True(t, storage.IsNotExist(err))
	})

	t.Run("default_submodule", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(t, workDir, 1, nil, true)

		content, err := storage.ReadPath(context.Background(), readBucket, "test.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 2", string(content), "expected the commit on local-branch to be checked out")
		_, err = readBucket.Stat(context.Background(), "nonexistent")
		assert.True(t, storage.IsNotExist(err))
		content, err = storage.ReadPath(context.Background(), readBucket, "submodule/test.proto")
		require.NoError(t, err)
		assert.Equal(t, "// submodule", string(content))
	})

	t.Run("main", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(t, workDir, 1, NewBranchName("main"), false)

		content, err := storage.ReadPath(context.Background(), readBucket, "test.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 1", string(content))
		_, err = readBucket.Stat(context.Background(), "nonexistent")
		assert.True(t, storage.IsNotExist(err))
	})

	t.Run("origin/main", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(t, workDir, 1, NewBranchName("origin/main"), false)

		content, err := storage.ReadPath(context.Background(), readBucket, "test.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 3", string(content))
		_, err = readBucket.Stat(context.Background(), "nonexistent")
		assert.True(t, storage.IsNotExist(err))
	})

	t.Run("origin/remote-branch", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(t, workDir, 1, NewBranchName("origin/remote-branch"), false)

		content, err := storage.ReadPath(context.Background(), readBucket, "test.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 4", string(content))
		_, err = readBucket.Stat(context.Background(), "nonexistent")
		assert.True(t, storage.IsNotExist(err))
	})

	t.Run("remote-tag", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(t, workDir, 1, NewTagName("remote-tag"), false)

		content, err := storage.ReadPath(context.Background(), readBucket, "test.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 4", string(content))
		_, err = readBucket.Stat(context.Background(), "nonexistent")
		assert.True(t, storage.IsNotExist(err))
	})

	t.Run("branch_and_main_ref", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(t, workDir, 2, NewRefNameWithBranch("HEAD~", "main"), false)

		content, err := storage.ReadPath(context.Background(), readBucket, "test.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 0", string(content))
		_, err = readBucket.Stat(context.Background(), "nonexistent")
		assert.True(t, storage.IsNotExist(err))
	})

	t.Run("branch_and_ref", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(t, workDir, 2, NewRefNameWithBranch("local-branch~", "local-branch"), false)

		content, err := storage.ReadPath(context.Background(), readBucket, "test.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 1", string(content))
		_, err = readBucket.Stat(context.Background(), "nonexistent")
		assert.True(t, storage.IsNotExist(err))
	})

	t.Run("HEAD", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(t, workDir, 1, NewRefName("HEAD"), false)

		content, err := storage.ReadPath(context.Background(), readBucket, "test.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 2", string(content))
		_, err = readBucket.Stat(context.Background(), "nonexistent")
		assert.True(t, storage.IsNotExist(err))
	})

	t.Run("commit-local", func(t *testing.T) {
		t.Parallel()
		revParseBytes, err := exec.Command("git", "-C", workDir, "rev-parse", "HEAD~").Output()
		require.NoError(t, err)
		readBucket := readBucketForName(t, workDir, 2, NewRefName(strings.TrimSpace(string(revParseBytes))), false)

		content, err := storage.ReadPath(context.Background(), readBucket, "test.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 1", string(content))
		_, err = readBucket.Stat(context.Background(), "nonexistent")
		assert.True(t, storage.IsNotExist(err))
	})

	t.Run("commit-remote", func(t *testing.T) {
		t.Parallel()
		revParseBytes, err := exec.Command("git", "-C", originDir, "rev-parse", "remote-branch~").Output()
		require.NoError(t, err)
		readBucket := readBucketForName(t, workDir, 2, NewRefNameWithBranch(strings.TrimSpace(string(revParseBytes)), "origin/remote-branch"), false)

		content, err := storage.ReadPath(context.Background(), readBucket, "test.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 3", string(content))
		_, err = readBucket.Stat(context.Background(), "nonexistent")
		assert.True(t, storage.IsNotExist(err))
	})
}

func readBucketForName(t *testing.T, path string, depth uint32, name Name, recurseSubmodules bool) storage.ReadBucket {
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	cloner := NewCloner(zap.NewNop(), storageosProvider, ClonerOptions{})
	envContainer, err := app.NewEnvContainerForOS()
	require.NoError(t, err)

	readBucketBuilder := storagemem.NewReadBucketBuilder()
	err = cloner.CloneToBucket(
		context.Background(),
		envContainer,
		"file://"+filepath.Join(path, ".git"),
		depth,
		readBucketBuilder,
		CloneToBucketOptions{
			Mapper:            storage.MatchPathExt(".proto"),
			Name:              name,
			RecurseSubmodules: recurseSubmodules,
		},
	)
	require.NoError(t, err)
	readBucket, err := readBucketBuilder.ToReadBucket()
	require.NoError(t, err)
	return readBucket
}

func createGitDirs(t *testing.T) (string, string) {
	tmpDir := t.TempDir()

	submodulePath := filepath.Join(tmpDir, "submodule")
	require.NoError(t, os.MkdirAll(submodulePath, os.ModePerm))
	runCommand(t, "git", "-C", submodulePath, "init")
	runCommand(t, "git", "-C", submodulePath, "config", "user.email", "tests@buf.build")
	runCommand(t, "git", "-C", submodulePath, "config", "user.name", "Buf go tests")
	runCommand(t, "git", "-C", submodulePath, "checkout", "-b", "main")
	require.NoError(t, os.WriteFile(filepath.Join(submodulePath, "test.proto"), []byte("// submodule"), 0600))
	runCommand(t, "git", "-C", submodulePath, "add", "test.proto")
	runCommand(t, "git", "-C", submodulePath, "commit", "-m", "commit 0")

	originPath := filepath.Join(tmpDir, "origin")
	require.NoError(t, os.MkdirAll(originPath, 0777))
	runCommand(t, "git", "-C", originPath, "init")
	runCommand(t, "git", "-C", originPath, "config", "user.email", "tests@buf.build")
	runCommand(t, "git", "-C", originPath, "config", "user.name", "Buf go tests")
	runCommand(t, "git", "-C", originPath, "checkout", "-b", "main")
	require.NoError(t, os.WriteFile(filepath.Join(originPath, "test.proto"), []byte("// commit 0"), 0600))
	runCommand(t, "git", "-C", originPath, "add", "test.proto")
	runCommand(t, "git", "-C", originPath, "commit", "-m", "commit 0")
	runCommand(t, "git", "-C", originPath, "submodule", "add", submodulePath, "submodule")
	require.NoError(t, os.WriteFile(filepath.Join(originPath, "test.proto"), []byte("// commit 1"), 0600))
	runCommand(t, "git", "-C", originPath, "add", "test.proto")
	runCommand(t, "git", "-C", originPath, "commit", "-m", "commit 1")

	workPath := filepath.Join(tmpDir, "workdir")
	runCommand(t, "git", "clone", originPath, workPath)
	runCommand(t, "git", "-C", workPath, "config", "user.email", "tests@buf.build")
	runCommand(t, "git", "-C", workPath, "config", "user.name", "Buf go tests")
	runCommand(t, "git", "-C", workPath, "checkout", "-b", "local-branch")
	require.NoError(t, os.WriteFile(filepath.Join(workPath, "test.proto"), []byte("// commit 2"), 0600))
	runCommand(t, "git", "-C", workPath, "commit", "-a", "-m", "commit 2")

	require.NoError(t, os.WriteFile(filepath.Join(originPath, "test.proto"), []byte("// commit 3"), 0600))
	runCommand(t, "git", "-C", originPath, "add", "test.proto")
	runCommand(t, "git", "-C", originPath, "commit", "-m", "commit 3")

	runCommand(t, "git", "-C", originPath, "checkout", "-b", "remote-branch")
	require.NoError(t, os.WriteFile(filepath.Join(originPath, "test.proto"), []byte("// commit 4"), 0600))
	runCommand(t, "git", "-C", originPath, "add", "test.proto")
	runCommand(t, "git", "-C", originPath, "commit", "-m", "commit 4")
	runCommand(t, "git", "-C", originPath, "tag", "remote-tag")

	runCommand(t, "git", "-C", workPath, "fetch", "origin")
	return originPath, workPath
}

func runCommand(t *testing.T, name string, args ...string) {
	output, err := exec.Command(name, args...).Output()
	if err != nil {
		var exitErr *exec.ExitError
		var stdErr []byte
		if errors.As(err, &exitErr) {
			stdErr = exitErr.Stderr
		}
		assert.FailNow(t, err.Error(), "stdout: %s\nstderr: %s", output, stdErr)
	}
}

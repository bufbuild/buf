// Copyright 2020-2025 Buf Technologies, Inc.
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
	"io/fs"
	"net/http/cgi"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"buf.build/go/app"
	"github.com/bufbuild/buf/private/pkg/slogtestext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitCloner(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container, err := app.NewContainerForOS()
	require.NoError(t, err)
	originDir, workDir := createGitDirs(ctx, t, container)

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, workDir, readBucketForNameOptions{})

		content, err := storage.ReadPath(ctx, readBucket, "a.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 2", string(content), "expected the commit on local-branch to be checked out")
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.True(t, errors.Is(err, fs.ErrNotExist))
		_, err = storage.ReadPath(ctx, readBucket, "submodule/sub.proto")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("default_submodule", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, workDir, readBucketForNameOptions{recurseSubmodules: true})

		content, err := storage.ReadPath(ctx, readBucket, "a.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 2", string(content), "expected the commit on local-branch to be checked out")
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.True(t, errors.Is(err, fs.ErrNotExist))
		content, err = storage.ReadPath(ctx, readBucket, "submodule/sub.proto")
		require.NoError(t, err)
		assert.Equal(t, "// submodule", string(content))
	})

	t.Run("main", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, workDir, readBucketForNameOptions{name: NewBranchName("main")})

		content, err := storage.ReadPath(ctx, readBucket, "a.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 1", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})
	t.Run("ref=main", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, workDir, readBucketForNameOptions{name: NewRefName("main")})

		content, err := storage.ReadPath(ctx, readBucket, "a.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 1", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("origin/main", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, workDir, readBucketForNameOptions{name: NewBranchName("origin/main")})

		content, err := storage.ReadPath(ctx, readBucket, "a.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 3", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})
	t.Run("ref=origin/main", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, workDir, readBucketForNameOptions{name: NewRefName("origin/main")})

		content, err := storage.ReadPath(ctx, readBucket, "a.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 3", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("origin/remote-branch", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, workDir, readBucketForNameOptions{name: NewBranchName("origin/remote-branch")})

		content, err := storage.ReadPath(ctx, readBucket, "a.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 4", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("remote-tag", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, workDir, readBucketForNameOptions{name: NewTagName("remote-tag")})

		content, err := storage.ReadPath(ctx, readBucket, "a.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 4", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})
	t.Run("ref=remote-tag", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, workDir, readBucketForNameOptions{name: NewRefName("remote-tag")})

		content, err := storage.ReadPath(ctx, readBucket, "a.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 4", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})
	t.Run("tag=remote-annotated-tag", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, workDir, readBucketForNameOptions{name: NewTagName("remote-annotated-tag")})

		content, err := storage.ReadPath(ctx, readBucket, "a.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 4", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})
	t.Run("ref=remote-annotated-tag", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, workDir, readBucketForNameOptions{name: NewRefName("remote-annotated-tag")})

		content, err := storage.ReadPath(ctx, readBucket, "a.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 4", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("branch_and_main_ref", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, workDir, readBucketForNameOptions{depth: 2, name: NewRefNameWithBranch("HEAD~", "main")})

		content, err := storage.ReadPath(ctx, readBucket, "a.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 0", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})
	t.Run("branch=main,ref=main~1", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, workDir, readBucketForNameOptions{depth: 2, name: NewRefNameWithBranch("main~", "main")})

		content, err := storage.ReadPath(ctx, readBucket, "a.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 0", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("branch_and_ref", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, workDir, readBucketForNameOptions{depth: 2, name: NewRefNameWithBranch("local-branch~", "local-branch")})

		content, err := storage.ReadPath(ctx, readBucket, "a.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 1", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("ref=HEAD", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, workDir, readBucketForNameOptions{name: NewRefName("HEAD")})

		content, err := storage.ReadPath(ctx, readBucket, "a.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 2", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.True(t, errors.Is(err, fs.ErrNotExist))
	})
	t.Run("ref=HEAD~", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, workDir, readBucketForNameOptions{depth: 2, name: NewRefName("HEAD~")})

		content, err := storage.ReadPath(ctx, readBucket, "a.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 1", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})
	t.Run("ref=HEAD~1", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, workDir, readBucketForNameOptions{depth: 2, name: NewRefName("HEAD~1")})

		content, err := storage.ReadPath(ctx, readBucket, "a.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 1", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})
	t.Run("ref=HEAD^", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, workDir, readBucketForNameOptions{depth: 2, name: NewRefName("HEAD^")})

		content, err := storage.ReadPath(ctx, readBucket, "a.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 1", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.True(t, errors.Is(err, fs.ErrNotExist))
	})
	t.Run("ref=HEAD^1", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, workDir, readBucketForNameOptions{depth: 2, name: NewRefName("HEAD^1")})

		content, err := storage.ReadPath(ctx, readBucket, "a.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 1", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("ref=<commit>", func(t *testing.T) {
		t.Parallel()
		revParseBytes, err := runStdout(ctx, container, "git", "-C", workDir, "rev-parse", "HEAD~")
		require.NoError(t, err)
		readBucket := readBucketForName(ctx, t, workDir, readBucketForNameOptions{depth: 2, name: NewRefName(strings.TrimSpace(string(revParseBytes)))})

		content, err := storage.ReadPath(ctx, readBucket, "a.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 1", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})
	t.Run("ref=<partial-commit>", func(t *testing.T) {
		t.Parallel()
		revParseBytes, err := runStdout(ctx, container, "git", "-C", workDir, "rev-parse", "HEAD~")
		require.NoError(t, err)
		partialRef := NewRefName(strings.TrimSpace(string(revParseBytes))[:8])
		readBucket := readBucketForName(ctx, t, workDir, readBucketForNameOptions{depth: 8, name: partialRef})

		content, err := storage.ReadPath(ctx, readBucket, "a.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 1", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("ref=<commit>,branch=origin/remote-branch", func(t *testing.T) {
		t.Parallel()
		revParseBytes, err := runStdout(ctx, container, "git", "-C", originDir, "rev-parse", "remote-branch~")
		require.NoError(t, err)
		readBucket := readBucketForName(ctx, t, workDir, readBucketForNameOptions{depth: 2, name: NewRefNameWithBranch(strings.TrimSpace(string(revParseBytes)), "origin/remote-branch")})

		content, err := storage.ReadPath(ctx, readBucket, "a.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 3", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})
	t.Run("ref=<partial-commit>,branch=origin/remote-branch", func(t *testing.T) {
		t.Parallel()
		revParseBytes, err := runStdout(ctx, container, "git", "-C", originDir, "rev-parse", "remote-branch~")
		require.NoError(t, err)
		partialRef := strings.TrimSpace(string(revParseBytes))[:8]
		readBucket := readBucketForName(ctx, t, workDir, readBucketForNameOptions{depth: 2, name: NewRefNameWithBranch(partialRef, "origin/remote-branch")})

		content, err := storage.ReadPath(ctx, readBucket, "a.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 3", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})
	t.Run("filter=tree:0", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, workDir, readBucketForNameOptions{name: NewBranchName("main"), filter: "tree:0"})

		_, err := readBucket.Stat(ctx, "a.proto")
		assert.NoError(t, err)
		_, err = readBucket.Stat(ctx, "c/c.proto")
		assert.NoError(t, err)
		content, err := storage.ReadPath(ctx, readBucket, "b/b.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 0", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})
	t.Run("filter=blob:none", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, workDir, readBucketForNameOptions{name: NewBranchName("main"), filter: "tree:0"})

		_, err := readBucket.Stat(ctx, "a.proto")
		assert.NoError(t, err)
		_, err = readBucket.Stat(ctx, "c/c.proto")
		assert.NoError(t, err)
		content, err := storage.ReadPath(ctx, readBucket, "b/b.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 0", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})
	t.Run("filter=tree:0,subdir=b", func(t *testing.T) {
		t.Parallel()
		readBucket := readBucketForName(ctx, t, workDir, readBucketForNameOptions{name: NewBranchName("main"), filter: "tree:0", subDir: "b"})

		// Only root and the b directory should be present. It is a sparse checkout.
		_, err := readBucket.Stat(ctx, "a.proto")
		assert.NoError(t, err)
		_, err = readBucket.Stat(ctx, "c/c.proto")
		assert.ErrorIs(t, err, fs.ErrNotExist)
		content, err := storage.ReadPath(ctx, readBucket, "b/b.proto")
		require.NoError(t, err)
		assert.Equal(t, "// commit 0", string(content))
		_, err = readBucket.Stat(ctx, "nonexistent")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})
}

type readBucketForNameOptions struct {
	depth             uint32
	name              Name
	recurseSubmodules bool
	subDir            string
	filter            string
}

func readBucketForName(ctx context.Context, t *testing.T, path string, options readBucketForNameOptions) storage.ReadBucket {
	t.Helper()
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	cloner := NewCloner(slogtestext.NewLogger(t), storageosProvider, ClonerOptions{})
	envContainer, err := app.NewEnvContainerForOS()
	require.NoError(t, err)

	depth := options.depth
	if depth == 0 {
		depth = 1
	}

	readWriteBucket := storagemem.NewReadWriteBucket()
	err = cloner.CloneToBucket(
		ctx,
		envContainer,
		"file://"+filepath.Join(path, ".git"),
		depth,
		readWriteBucket,
		CloneToBucketOptions{
			Matcher:           storage.MatchPathExt(".proto"),
			Name:              options.name,
			RecurseSubmodules: options.recurseSubmodules,
			SubDir:            options.subDir,
			Filter:            options.filter,
		},
	)
	require.NoError(t, err)
	return readWriteBucket
}

func createGitDirs(
	ctx context.Context,
	t *testing.T,
	container app.EnvStdioContainer,
) (string, string) {
	tmpDir := t.TempDir()

	submodulePath := filepath.Join(tmpDir, "submodule")
	require.NoError(t, os.MkdirAll(submodulePath, os.ModePerm))
	runCommand(ctx, t, container, "git", "-C", submodulePath, "init")
	runCommand(ctx, t, container, "git", "-C", submodulePath, "config", "user.email", "tests@buf.build")
	runCommand(ctx, t, container, "git", "-C", submodulePath, "config", "user.name", "Buf go tests")
	runCommand(ctx, t, container, "git", "-C", submodulePath, "checkout", "-b", "main")
	require.NoError(t, os.WriteFile(filepath.Join(submodulePath, "sub.proto"), []byte("// submodule"), 0600))
	runCommand(ctx, t, container, "git", "-C", submodulePath, "add", "sub.proto")
	runCommand(ctx, t, container, "git", "-C", submodulePath, "commit", "-m", "commit 0")

	gitExecPathBytes, err := runStdout(ctx, container, "git", "--exec-path")
	require.NoError(t, err)
	gitExecPath := strings.TrimSpace(string(gitExecPathBytes))
	// In Golang 1.22, the behavior of "os/exec" was changed so that LookPath is no longer called
	// in some cases. This preserves the behavior.
	// https://cs.opensource.google/go/go/+/f7f266c88598398dcf32b448bcea2100e1702630:src/os/exec/exec.go;dlc=07d4de9312aef72d1bd7427316a2ac21b83e4a20
	// https://tip.golang.org/doc/go1.22 (search for "LookPath")
	gitHTTPBackendPath, err := exec.LookPath(filepath.Join(gitExecPath, "git-http-backend"))
	require.NoError(t, err)
	t.Logf("gitHttpBackendPath=%q submodulePath=%q", gitHTTPBackendPath, submodulePath)
	// https://git-scm.com/docs/git-http-backend#_description
	f, err := os.Create(filepath.Join(submodulePath, ".git", "git-daemon-export-ok"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	server := httptest.NewServer(
		&cgi.Handler{
			Path: gitHTTPBackendPath,
			Dir:  submodulePath,
			Env: append(
				app.Environ(container),
				fmt.Sprintf("GIT_PROJECT_ROOT=%s", submodulePath),
			),
			Stderr: container.Stderr(),
		},
	)
	t.Cleanup(server.Close)
	submodulePath = server.URL

	originPath := filepath.Join(tmpDir, "origin")
	require.NoError(t, os.MkdirAll(originPath, 0777))
	runCommand(ctx, t, container, "git", "-C", originPath, "init")
	runCommand(ctx, t, container, "git", "-C", originPath, "config", "user.email", "tests@buf.build")
	runCommand(ctx, t, container, "git", "-C", originPath, "config", "user.name", "Buf go tests")
	runCommand(ctx, t, container, "git", "-C", originPath, "checkout", "-b", "main")
	require.NoError(t, os.WriteFile(filepath.Join(originPath, "a.proto"), []byte("// commit 0"), 0600))
	require.NoError(t, os.MkdirAll(filepath.Join(originPath, "b"), 0777))
	require.NoError(t, os.WriteFile(filepath.Join(originPath, "b", "b.proto"), []byte("// commit 0"), 0600))
	require.NoError(t, os.MkdirAll(filepath.Join(originPath, "c"), 0777))
	require.NoError(t, os.WriteFile(filepath.Join(originPath, "c", "c.proto"), []byte("// commit 0"), 0600))
	runCommand(ctx, t, container, "git", "-C", originPath, "add", "a.proto")
	runCommand(ctx, t, container, "git", "-C", originPath, "add", "b/b.proto")
	runCommand(ctx, t, container, "git", "-C", originPath, "add", "c/c.proto")
	runCommand(ctx, t, container, "git", "-C", originPath, "commit", "-m", "commit 0")
	runCommand(ctx, t, container, "git", "-C", originPath, "submodule", "add", submodulePath, "submodule")
	require.NoError(t, os.WriteFile(filepath.Join(originPath, "a.proto"), []byte("// commit 1"), 0600))
	runCommand(ctx, t, container, "git", "-C", originPath, "add", "a.proto")
	runCommand(ctx, t, container, "git", "-C", originPath, "commit", "-m", "commit 1")

	workPath := filepath.Join(tmpDir, "workdir")
	runCommand(ctx, t, container, "git", "clone", originPath, workPath)
	runCommand(ctx, t, container, "git", "-C", workPath, "config", "user.email", "tests@buf.build")
	runCommand(ctx, t, container, "git", "-C", workPath, "config", "user.name", "Buf go tests")
	runCommand(ctx, t, container, "git", "-C", workPath, "checkout", "-b", "local-branch")
	require.NoError(t, os.WriteFile(filepath.Join(workPath, "a.proto"), []byte("// commit 2"), 0600))
	runCommand(ctx, t, container, "git", "-C", workPath, "commit", "-a", "-m", "commit 2")

	require.NoError(t, os.WriteFile(filepath.Join(originPath, "a.proto"), []byte("// commit 3"), 0600))
	runCommand(ctx, t, container, "git", "-C", originPath, "add", "a.proto")
	runCommand(ctx, t, container, "git", "-C", originPath, "commit", "-m", "commit 3")

	runCommand(ctx, t, container, "git", "-C", originPath, "checkout", "-b", "remote-branch")
	require.NoError(t, os.WriteFile(filepath.Join(originPath, "a.proto"), []byte("// commit 4"), 0600))
	runCommand(ctx, t, container, "git", "-C", originPath, "add", "a.proto")
	runCommand(ctx, t, container, "git", "-C", originPath, "commit", "-m", "commit 4")
	runCommand(ctx, t, container, "git", "-C", originPath, "tag", "remote-tag")
	runCommand(ctx, t, container, "git", "-C", originPath, "tag", "-a", "remote-annotated-tag", "-m", "annotated tag")

	runCommand(ctx, t, container, "git", "-C", workPath, "fetch", "origin")
	return originPath, workPath
}

func runCommand(
	ctx context.Context,
	t *testing.T,
	container app.EnvStdioContainer,
	name string,
	args ...string,
) {
	t.Helper()
	output, err := runStdout(ctx, container, name, args...)
	if err != nil {
		var exitErr *exec.ExitError
		var stdErr []byte
		if errors.As(err, &exitErr) {
			stdErr = exitErr.Stderr
		}
		assert.FailNow(t, err.Error(), "stdout: %s\nstderr: %s", output, stdErr)
	}
}

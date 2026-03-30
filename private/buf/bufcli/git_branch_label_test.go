// Copyright 2020-2026 Buf Technologies, Inc.
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

package bufcli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"buf.build/go/app"
	"buf.build/go/standard/xos/xexec"
	"github.com/bufbuild/buf/private/pkg/slogtestext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetGitBranchLabelForModule(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	logger := slogtestext.NewLogger(t)

	t.Run("disabled_when_list_is_empty", func(t *testing.T) {
		t.Parallel()
		envContainer, err := app.NewEnvContainerForOS()
		require.NoError(t, err)
		branch, enabled, err := GetGitBranchLabelForModule(
			ctx, logger, envContainer, t.TempDir(),
			"buf.build/acme/weather",
			nil,
			nil,
		)
		require.NoError(t, err)
		assert.False(t, enabled)
		assert.Empty(t, branch)
	})

	t.Run("disabled_when_module_not_in_list", func(t *testing.T) {
		t.Parallel()
		envContainer, err := app.NewEnvContainerForOS()
		require.NoError(t, err)
		branch, enabled, err := GetGitBranchLabelForModule(
			ctx, logger, envContainer, t.TempDir(),
			"buf.build/acme/other",
			[]string{"buf.build/acme/weather"},
			[]string{"main"},
		)
		require.NoError(t, err)
		assert.False(t, enabled)
		assert.Empty(t, branch)
	})

	t.Run("returns_branch_when_module_matches", func(t *testing.T) {
		t.Parallel()
		envContainer, err := app.NewEnvContainerForOS()
		require.NoError(t, err)
		repoDir := createTestGitRepo(ctx, t, envContainer, "feature/new-api")
		label, enabled, err := GetGitBranchLabelForModule(
			ctx, logger, envContainer, repoDir,
			"buf.build/acme/weather",
			[]string{"buf.build/acme/weather"},
			[]string{"main", "master"},
		)
		require.NoError(t, err)
		assert.True(t, enabled)
		// "/" in branch names is converted to "_" for BSR label compatibility.
		assert.Equal(t, "feature_new-api", label)
	})

	t.Run("branch_without_slash_unchanged", func(t *testing.T) {
		t.Parallel()
		envContainer, err := app.NewEnvContainerForOS()
		require.NoError(t, err)
		repoDir := createTestGitRepo(ctx, t, envContainer, "my-feature")
		label, enabled, err := GetGitBranchLabelForModule(
			ctx, logger, envContainer, repoDir,
			"buf.build/acme/weather",
			[]string{"buf.build/acme/weather"},
			[]string{"main", "master"},
		)
		require.NoError(t, err)
		assert.True(t, enabled)
		assert.Equal(t, "my-feature", label)
	})

	t.Run("disabled_when_on_disabled_branch", func(t *testing.T) {
		t.Parallel()
		envContainer, err := app.NewEnvContainerForOS()
		require.NoError(t, err)
		repoDir := createTestGitRepo(ctx, t, envContainer, "main")
		branch, enabled, err := GetGitBranchLabelForModule(
			ctx, logger, envContainer, repoDir,
			"buf.build/acme/weather",
			[]string{"buf.build/acme/weather"},
			[]string{"main", "master"},
		)
		require.NoError(t, err)
		assert.False(t, enabled)
		assert.Empty(t, branch)
	})

	t.Run("disabled_when_not_git_repo", func(t *testing.T) {
		t.Parallel()
		envContainer, err := app.NewEnvContainerForOS()
		require.NoError(t, err)
		branch, enabled, err := GetGitBranchLabelForModule(
			ctx, logger, envContainer, t.TempDir(),
			"buf.build/acme/weather",
			[]string{"buf.build/acme/weather"},
			[]string{"main"},
		)
		require.NoError(t, err)
		assert.False(t, enabled)
		assert.Empty(t, branch)
	})

	t.Run("disabled_when_env_var_off", func(t *testing.T) {
		t.Parallel()
		envContainer, err := app.NewEnvContainerForOS()
		require.NoError(t, err)
		envContainer = app.NewEnvContainerWithOverrides(
			envContainer,
			map[string]string{"BUF_USE_GIT_BRANCH_AS_LABEL": "OFF"},
		)
		repoDir := createTestGitRepo(ctx, t, envContainer, "feature/test")
		branch, enabled, err := GetGitBranchLabelForModule(
			ctx, logger, envContainer, repoDir,
			"buf.build/acme/weather",
			[]string{"buf.build/acme/weather"},
			[]string{"main"},
		)
		require.NoError(t, err)
		assert.False(t, enabled)
		assert.Empty(t, branch)
	})

	t.Run("env_var_off_case_insensitive", func(t *testing.T) {
		t.Parallel()
		envContainer, err := app.NewEnvContainerForOS()
		require.NoError(t, err)
		envContainer = app.NewEnvContainerWithOverrides(
			envContainer,
			map[string]string{"BUF_USE_GIT_BRANCH_AS_LABEL": "off"},
		)
		repoDir := createTestGitRepo(ctx, t, envContainer, "feature/test")
		branch, enabled, err := GetGitBranchLabelForModule(
			ctx, logger, envContainer, repoDir,
			"buf.build/acme/weather",
			[]string{"buf.build/acme/weather"},
			[]string{"main"},
		)
		require.NoError(t, err)
		assert.False(t, enabled)
		assert.Empty(t, branch)
	})
}

// createTestGitRepo creates a temporary git repo on the given branch with an
// initial commit. Returns the repo directory path.
func createTestGitRepo(
	ctx context.Context,
	t *testing.T,
	envContainer app.EnvContainer,
	branchName string,
) string {
	t.Helper()
	repoDir := t.TempDir()
	environ := app.Environ(envContainer)
	runGit(ctx, t, repoDir, environ, "init")
	runGit(ctx, t, repoDir, environ, "config", "user.email", "tests@buf.build")
	runGit(ctx, t, repoDir, environ, "config", "user.name", "Buf go tests")
	runGit(ctx, t, repoDir, environ, "checkout", "-b", "main")
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "test.txt"), []byte("hello"), 0600))
	runGit(ctx, t, repoDir, environ, "add", "test.txt")
	runGit(ctx, t, repoDir, environ, "commit", "-m", "initial commit")
	if branchName != "main" {
		runGit(ctx, t, repoDir, environ, "checkout", "-b", branchName)
	}
	return repoDir
}

func runGit(ctx context.Context, t *testing.T, dir string, environ []string, args ...string) {
	t.Helper()
	stderr := bytes.NewBuffer(nil)
	err := xexec.Run(
		ctx,
		"git",
		xexec.WithArgs(args...),
		xexec.WithDir(dir),
		xexec.WithEnv(environ),
		xexec.WithStderr(stderr),
	)
	require.NoError(t, err, "git %v failed: %s", args, stderr.String())
}

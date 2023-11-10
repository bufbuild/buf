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

package bufsync_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bufbuild/buf/private/buf/bufsync"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/git/gittest"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
)

// scaffoldGitRepository returns an initialized git repository with a single commit, and returns the
// repository and its directory.
func scaffoldGitRepository(t *testing.T, defaultBranchName string) (git.Repository, string) {
	runner := command.NewRunner()
	repoDir := scaffoldGitRepositoryDir(t, runner, defaultBranchName)
	dotGitPath := path.Join(repoDir, git.DotGitDir)
	repo, err := git.OpenRepository(
		context.Background(),
		dotGitPath,
		runner,
		git.OpenRepositoryWithDefaultBranch(defaultBranchName),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, repo.Close())
	})
	return repo, repoDir
}

// scaffoldGitRepositoryDir prepares a git repository with an initial README, and a single commit.
// It returns the directory where the local git repo is.
func scaffoldGitRepositoryDir(t *testing.T, runner command.Runner, defaultBranchName string) string {
	repoDir := t.TempDir()

	// setup repo
	runInDir(t, runner, repoDir, "git", "init", "--initial-branch", defaultBranchName)
	runInDir(t, runner, repoDir, "git", "config", "user.name", "Buf TestBot")
	runInDir(t, runner, repoDir, "git", "config", "user.email", "testbot@buf.build")

	// write and commit a README file
	writeFiles(t, repoDir, map[string]string{"README.md": "This is a scaffold repository.\n"})
	runInDir(t, runner, repoDir, "git", "add", ".")
	runInDir(t, runner, repoDir, "git", "commit", "-m", "Write README")

	return repoDir
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

type mockClock struct {
	now time.Time
}

func (c *mockClock) Now() time.Time { return c.now }

type mockSyncHandler struct {
	syncedCommitsSHAs               map[string]struct{}
	commitsByBranch                 map[string][]bufsync.ModuleCommit
	hashByTag                       map[string]git.Hash
	tagsByHash                      map[string][]string
	manualSyncPointByModuleByBranch map[string]map[string]git.Hash
}

func newMockSyncHandler() *mockSyncHandler {
	return &mockSyncHandler{
		syncedCommitsSHAs:               make(map[string]struct{}),
		commitsByBranch:                 make(map[string][]bufsync.ModuleCommit),
		hashByTag:                       make(map[string]git.Hash),
		tagsByHash:                      make(map[string][]string),
		manualSyncPointByModuleByBranch: make(map[string]map[string]git.Hash),
	}
}

func (c *mockSyncHandler) setSyncPoint(branch string, hash git.Hash, identity bufmoduleref.ModuleIdentity) {
	branchSyncpoints, ok := c.manualSyncPointByModuleByBranch[branch]
	if !ok {
		branchSyncpoints = make(map[string]git.Hash)
		c.manualSyncPointByModuleByBranch[branch] = branchSyncpoints
	}
	branchSyncpoints[identity.IdentityString()] = hash
	c.syncedCommitsSHAs[hash.Hex()] = struct{}{}
}

func (c *mockSyncHandler) HandleReadModuleError(
	readErr *bufsync.ReadModuleError,
) bufsync.LookbackDecisionCode {
	if readErr.Code() == bufsync.ReadModuleErrorCodeUnexpectedName {
		return bufsync.LookbackDecisionCodeOverride
	}
	return bufsync.LookbackDecisionCodeSkip
}

func (c *mockSyncHandler) InvalidBSRSyncPoint(
	identity bufmoduleref.ModuleIdentity,
	branch string,
	gitHash git.Hash,
	isDefaultBranch bool,
	err error,
) error {
	return errors.New("unimplemented")
}

func (c *mockSyncHandler) BackfillTags(
	ctx context.Context,
	module bufmoduleref.ModuleIdentity,
	alreadySyncedHash git.Hash,
	author git.Ident,
	committer git.Ident,
	tags []string,
) (string, error) {
	for _, tag := range tags {
		if previousHash, ok := c.hashByTag[tag]; ok {
			// clear previous tag
			c.tagsByHash[previousHash.Hex()] = slices.DeleteFunc(
				c.tagsByHash[previousHash.Hex()],
				func(previousTag string) bool {
					return previousTag == tag
				},
			)
		}
		c.hashByTag[tag] = alreadySyncedHash
	}
	c.tagsByHash[alreadySyncedHash.Hex()] = tags
	return "some-BSR-commit-name", nil
}

func (c *mockSyncHandler) ResolveSyncPoint(
	ctx context.Context,
	module bufmoduleref.ModuleIdentity,
	branch string,
) (git.Hash, error) {
	// if we have commits from SyncModuleCommit, prefer that over
	// manually set sync point
	if branch, ok := c.commitsByBranch[branch]; ok && len(branch) > 0 {
		// everything here is synced; return tip of branch
		return branch[len(branch)-1].Commit().Hash(), nil
	}
	if branch, ok := c.manualSyncPointByModuleByBranch[branch]; ok {
		if syncPoint, ok := branch[module.IdentityString()]; ok {
			return syncPoint, nil
		}
	}
	return nil, nil
}

func (c *mockSyncHandler) SyncModuleCommit(
	ctx context.Context,
	commit bufsync.ModuleCommit,
) error {
	c.setSyncPoint(
		commit.Branch(),
		commit.Commit().Hash(),
		commit.Identity(),
	)
	// append-only, no backfill; good enough for now!
	c.commitsByBranch[commit.Branch()] = append(c.commitsByBranch[commit.Branch()], commit)
	_, err := c.BackfillTags(
		ctx,
		commit.Identity(),
		commit.Commit().Hash(),
		commit.Commit().Author(),
		commit.Commit().Committer(),
		commit.Tags(),
	)
	return err
}

func (c *mockSyncHandler) IsGitCommitSynced(
	ctx context.Context,
	module bufmoduleref.ModuleIdentity,
	hash git.Hash,
) (bool, error) {
	_, isSynced := c.syncedCommitsSHAs[hash.Hex()]
	return isSynced, nil
}

func (c *mockSyncHandler) IsProtectedBranch(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branch string,
) (bool, error) {
	return branch == gittest.DefaultBranch || branch == bufmoduleref.Main, nil
}

var _ bufsync.Handler = (*mockSyncHandler)(nil)

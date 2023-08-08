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
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestCommitsToSyncWithNoPreviousSyncPoints(t *testing.T) {
	t.Parallel()
	// share git repo, bsr checker and modules to sync for all the test scenarios, as a regular `buf
	// sync` run would do.
	mockBSRChecker := newMockSyncGitChecker()
	moduleIdentityInHEAD, err := bufmoduleref.NewModuleIdentity("buf.build", "acme", "foo")
	require.NoError(t, err)
	// scaffoldGitRepository returns a repo with the following commits:
	// | o-o----------o-----------------o (main)
	// |   └o-o (foo) └o--------o (bar)
	// |               └o (baz)
	repo := scaffoldGitRepository(t, moduleIdentityInHEAD)
	testSyncer := syncer{
		repo:                                  repo,
		storageGitProvider:                    storagegit.NewProvider(repo.Objects()),
		logger:                                zaptest.NewLogger(t),
		modulesDirsToIdentityOverrideForSync:  map[string]struct{}{".": {}},
		sortedModulesDirsForSync:              []string{"."},
		syncAllBranches:                       true,
		syncedGitCommitChecker:                mockBSRChecker.checkFunc(),
		commitsToTags:                         make(map[string][]string),
		branchesToModulesForSync:              make(map[string]map[string]bufmoduleref.ModuleIdentity),
		modulesToBranchesLastSyncPoints:       make(map[string]map[string]string),
		modulesIdentitiesToCommitsSyncedCache: make(map[string]map[string]struct{}),
	}
	require.NoError(t, testSyncer.prepareSync(context.Background()))

	type testCase struct {
		name            string
		branch          string
		expectedCommits int
	}
	testCases := []testCase{
		{
			name:            "when_main",
			branch:          "main",
			expectedCommits: 4,
		},
		{
			name:            "when_foo",
			branch:          "foo",
			expectedCommits: 3, // counting the commit that branches off main
		},
		{
			name:            "when_bar",
			branch:          "bar",
			expectedCommits: 3, // counting the commit that branches off main
		},
		{
			name:            "when_baz",
			branch:          "baz",
			expectedCommits: 2, // counting the commit that branches off bar
		},
	}
	for _, tc := range testCases {
		func(tc testCase) {
			t.Run(tc.name, func(t *testing.T) {
				syncableCommits, err := testSyncer.branchSyncableCommits(
					context.Background(),
					tc.branch,
				)
				// uncomment for debug purposes
				// s.printCommitsToSync(tc.branch, syncableCommits)
				require.NoError(t, err)
				require.Len(t, syncableCommits, tc.expectedCommits)
				for i, syncableCommit := range syncableCommits {
					assert.NotEmpty(t, syncableCommit.commit.Hash().Hex())
					mockBSRChecker.markSynced(syncableCommit.commit.Hash().Hex())
					if tc.branch != "main" && i == 0 {
						// first commit in non-default branches will come with no modules to sync, because it's
						// the commit in which it branches off the parent branch.
						assert.Empty(t, syncableCommit.modules)
					} else {
						assert.Len(t, syncableCommit.modules, 1)
						for moduleDir, builtModule := range syncableCommit.modules {
							assert.Equal(t, ".", moduleDir)
							assert.Equal(t, moduleIdentityInHEAD.IdentityString(), builtModule.ModuleIdentity().IdentityString())
						}
					}
				}
			})
		}(tc)
	}
}

type mockSyncedGitChecker struct {
	syncedCommitsSHAs map[string]struct{}
}

func newMockSyncGitChecker() mockSyncedGitChecker {
	return mockSyncedGitChecker{syncedCommitsSHAs: make(map[string]struct{})}
}

func (c *mockSyncedGitChecker) markSynced(gitHash string) {
	c.syncedCommitsSHAs[gitHash] = struct{}{}
}

func (c *mockSyncedGitChecker) checkFunc() SyncedGitCommitChecker {
	return func(
		_ context.Context,
		_ bufmoduleref.ModuleIdentity,
		commitHashes map[string]struct{},
	) (map[string]struct{}, error) {
		syncedHashes := make(map[string]struct{})
		for hash := range commitHashes {
			if _, isSynced := c.syncedCommitsSHAs[hash]; isSynced {
				syncedHashes[hash] = struct{}{}
			}
		}
		return syncedHashes, nil
	}
}

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

func writeFiles(t *testing.T, directoryPath string, pathToContents map[string]string) {
	for path, contents := range pathToContents {
		require.NoError(t, os.MkdirAll(filepath.Join(directoryPath, filepath.Dir(path)), 0700))
		require.NoError(t, os.WriteFile(filepath.Join(directoryPath, path), []byte(contents), 0600))
	}
}

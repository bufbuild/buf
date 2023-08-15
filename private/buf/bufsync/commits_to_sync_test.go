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
	"context"
	"errors"
	"fmt"
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
	moduleIdentityInHEAD, err := bufmoduleref.NewModuleIdentity("buf.build", "acme", "foo")
	require.NoError(t, err)
	moduleIdentityOverride, err := bufmoduleref.NewModuleIdentity("buf.build", "acme", "bar")
	require.NoError(t, err)
	repo, repoDir := scaffoldGitRepository(t)
	prepareGitRepoSyncWithNoPreviousSyncPoints(t, repoDir, moduleIdentityInHEAD)
	type testCase struct {
		name            string
		branch          string
		expectedCommits int
	}
	testCases := []testCase{
		{
			name:            "when_main",
			branch:          "main",
			expectedCommits: 5, // including the initial commit when scaffolding the test repo
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
	for _, withOverride := range []bool{false, true} {
		mockBSRChecker := newMockSyncGitChecker()
		for _, tc := range testCases {
			func(tc testCase) {
				t.Run(fmt.Sprintf("%s/override_%t", tc.name, withOverride), func(t *testing.T) {
					moduleDirsForSync := []string{"."}
					moduleDirsToIdentity := make(map[string]bufmoduleref.ModuleIdentity)
					for _, moduleDir := range moduleDirsForSync {
						if withOverride {
							moduleDirsToIdentity[moduleDir] = moduleIdentityOverride
						} else {
							moduleDirsToIdentity[moduleDir] = nil
						}
					}
					testSyncer := syncer{
						repo:                                  repo,
						storageGitProvider:                    storagegit.NewProvider(repo.Objects()),
						logger:                                zaptest.NewLogger(t),
						errorHandler:                          &mockErrorHandler{},
						modulesDirsToIdentityOverrideForSync:  moduleDirsToIdentity,
						sortedModulesDirsForSync:              moduleDirsForSync,
						syncAllBranches:                       true,
						syncedGitCommitChecker:                mockBSRChecker.checkFunc(),
						commitsToTags:                         make(map[string][]string),
						branchesToModulesForSync:              make(map[string]map[string]bufmoduleref.ModuleIdentity),
						modulesToBranchesLastSyncPoints:       make(map[string]map[string]string),
						modulesIdentitiesToCommitsSyncedCache: make(map[string]map[string]struct{}),
					}
					require.NoError(t, testSyncer.prepareSync(context.Background()))
					syncableCommits, err := testSyncer.branchSyncableCommits(
						context.Background(),
						tc.branch,
					)
					// uncomment for debug purposes
					// testSyncer.printCommitsForSync(tc.branch, syncableCommits)
					require.NoError(t, err)
					require.Len(t, syncableCommits, tc.expectedCommits)
					for i, syncableCommit := range syncableCommits {
						assert.NotEmpty(t, syncableCommit.commit.Hash().Hex())
						mockBSRChecker.markSynced(syncableCommit.commit.Hash().Hex())
						if i == 0 {
							// First commit in the default branch has no module. Also, first commit in non-default
							// branches will come with no modules to sync, because it's the commit in which it
							// branches off the parent branch.
							assert.Empty(t, syncableCommit.modules)
						} else {
							assert.Len(t, syncableCommit.modules, 1)
							for moduleDir, builtModule := range syncableCommit.modules {
								assert.Equal(t, ".", moduleDir)
								if withOverride {
									assert.Equal(t, moduleIdentityOverride.IdentityString(), builtModule.ModuleIdentity().IdentityString())
								} else {
									assert.Equal(t, moduleIdentityInHEAD.IdentityString(), builtModule.ModuleIdentity().IdentityString())
								}
							}
						}
					}
				})
			}(tc)
		}
	}
}

type mockErrorHandler struct{}

func (*mockErrorHandler) HandleReadModuleError(readErr *ReadModuleError) LookbackDecisionCode {
	if readErr.code == ReadModuleErrorCodeUnexpectedName {
		return LookbackDecisionCodeOverride
	}
	return LookbackDecisionCodeSkip
}

func (*mockErrorHandler) InvalidRemoteSyncPoint(bufmoduleref.ModuleIdentity, string, git.Hash, bool, error) error {
	return errors.New("unimplemented")
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

// prepareGitRepoSyncWithNoPreviousSyncPoints writes and pushes commits in the repo with the
// following commits:
//
// | o-o----------o-----------------o (master)
// |   └o-o (foo) └o--------o (bar)
// |               └o (baz)
func prepareGitRepoSyncWithNoPreviousSyncPoints(t *testing.T, repoDir string, moduleIdentity bufmoduleref.ModuleIdentity) {
	runner := command.NewRunner()
	var allBranches = []string{defaultBranch, "foo", "bar", "baz"}

	var commitsCounter int
	doEmptyCommit := func(numOfCommits int) {
		for i := 0; i < numOfCommits; i++ {
			commitsCounter++
			runInDir(
				t, runner, repoDir,
				"git", "commit", "--allow-empty",
				"-m", fmt.Sprintf("commit %d", commitsCounter),
			)
		}
	}

	// write the base module in the root
	writeFiles(t, repoDir, map[string]string{
		"buf.yaml": fmt.Sprintf("version: v1\nname: %s\n", moduleIdentity.IdentityString()),
	})
	runInDir(t, runner, repoDir, "git", "add", ".")
	runInDir(t, runner, repoDir, "git", "commit", "-m", "commit 0")

	doEmptyCommit(1)
	runInDir(t, runner, repoDir, "git", "checkout", "-b", allBranches[1])
	doEmptyCommit(2)
	runInDir(t, runner, repoDir, "git", "checkout", defaultBranch)
	doEmptyCommit(1)
	runInDir(t, runner, repoDir, "git", "checkout", "-b", allBranches[2])
	doEmptyCommit(1)
	runInDir(t, runner, repoDir, "git", "checkout", "-b", allBranches[3])
	doEmptyCommit(1)
	runInDir(t, runner, repoDir, "git", "checkout", allBranches[2])
	doEmptyCommit(1)
	runInDir(t, runner, repoDir, "git", "checkout", defaultBranch)
	doEmptyCommit(1)

	// push them all
	for _, branch := range allBranches {
		runInDir(t, runner, repoDir, "git", "checkout", branch)
		runInDir(t, runner, repoDir, "git", "push", "-u", "-f", remoteName, branch)
	}
}

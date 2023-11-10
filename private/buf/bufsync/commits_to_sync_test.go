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
	"context"
	"fmt"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufsync"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestCommitsToSyncWithNoPreviousSyncPoints(t *testing.T) {
	t.Parallel()
	moduleIdentityInHEAD, err := bufmoduleref.NewModuleIdentity("buf.build", "acme", "foo")
	require.NoError(t, err)
	moduleIdentityOverride, err := bufmoduleref.NewModuleIdentity("buf.build", "acme", "bar")
	require.NoError(t, err)
	const defaultBranchName = "main"
	repo, repoDir := scaffoldGitRepository(t, defaultBranchName)
	prepareGitRepoSyncWithNoPreviousSyncPoints(t, repoDir, moduleIdentityInHEAD, defaultBranchName)
	type testCase struct {
		name            string
		branch          string
		expectedCommits int
	}
	testCases := []testCase{
		{
			name:            "when_main",
			branch:          "main",
			expectedCommits: 4, // doesn't include initial scaffolding empty commit
		},
		{
			name:            "when_foo",
			branch:          "foo",
			expectedCommits: 2,
		},
		{
			name:            "when_bar",
			branch:          "bar",
			expectedCommits: 2,
		},
		{
			name:            "when_baz",
			branch:          "baz",
			expectedCommits: 1,
		},
	}
	for _, withOverride := range []bool{false, true} {
		for _, tc := range testCases {
			func(tc testCase) {
				t.Run(fmt.Sprintf("%s/override_%t", tc.name, withOverride), func(t *testing.T) {
					const moduleDir = "."
					opts := []bufsync.SyncerOption{
						bufsync.SyncerWithAllBranches(),
					}
					if withOverride {
						opts = append(opts, bufsync.SyncerWithModule(moduleDir, moduleIdentityOverride))
					} else {
						opts = append(opts, bufsync.SyncerWithModule(moduleDir, nil))
					}
					handler := newMockSyncHandler()
					syncer, err := bufsync.NewSyncer(
						zaptest.NewLogger(t),
						bufsync.NewRealClock(),
						repo,
						storagegit.NewProvider(repo.Objects()),
						handler,
						opts...,
					)
					require.NoError(t, err)
					require.NoError(t, syncer.Sync(context.Background()))
					syncedCommits := handler.commitsByBranch[tc.branch]
					require.Len(t, syncedCommits, tc.expectedCommits)
				})
			}(tc)
		}
	}
}

// prepareGitRepoSyncWithNoPreviousSyncPoints writes and pushes commits in the repo with the
// following commits:
//
// | o-o----------o-----------------o (master)
// |   └o-o (foo) └o--------o (bar)
// |               └o (baz)
func prepareGitRepoSyncWithNoPreviousSyncPoints(t *testing.T, repoDir string, moduleIdentity bufmoduleref.ModuleIdentity, defaultBranchName string) {
	runner := command.NewRunner()
	var allBranches = []string{defaultBranchName, "foo", "bar", "baz"}

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
	runInDir(t, runner, repoDir, "git", "checkout", defaultBranchName)
	doEmptyCommit(1)
	runInDir(t, runner, repoDir, "git", "checkout", "-b", allBranches[2])
	doEmptyCommit(1)
	runInDir(t, runner, repoDir, "git", "checkout", "-b", allBranches[3])
	doEmptyCommit(1)
	runInDir(t, runner, repoDir, "git", "checkout", allBranches[2])
	doEmptyCommit(1)
	runInDir(t, runner, repoDir, "git", "checkout", defaultBranchName)
	doEmptyCommit(1)
}

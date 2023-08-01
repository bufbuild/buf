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
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommitsToSyncWithNoPreviousSyncPoints(t *testing.T) {
	t.Parallel()
	// share git repo, bsr checker and modules to sync for all the test scenarios, as a regular `buf
	// sync` run would do.
	mockBSRChecker := newMockSyncGitChecker()
	someModule, err := bufmoduleref.NewModuleIdentity("buf.test", "owner", "repo")
	require.NoError(t, err)
	moduleToSync, err := newSyncableModule(".", someModule)
	require.NoError(t, err)
	// scaffoldGitRepository returns a repo with the following commits:
	// | o-o----------o-----------------o (main)
	// |   └o-o (foo) └o--------o (bar)
	// |               └o (baz)
	repo := scaffoldGitRepository(t)
	s := syncer{
		repo:                   repo,
		modulesDirsToSync:      []Module{moduleToSync},
		syncedGitCommitChecker: mockBSRChecker.checkFunc(),
	}

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
	for _, tc := range testCases {
		func(tc testCase) {
			t.Run(tc.name, func(t *testing.T) {
				syncableCommits, err := s.commitsToSync(
					context.Background(),
					tc.branch,
					nil,
				)
				require.NoError(t, err)
				require.Len(t, syncableCommits, tc.expectedCommits)
				for _, syncableCommit := range syncableCommits {
					assert.NotEmpty(t, syncableCommit.commit.Hash().Hex())
					mockBSRChecker.markSynced(syncableCommit.commit.Hash().Hex())
					assert.Equal(t, len(syncableCommit.modules), 1)
					for module := range syncableCommit.modules {
						assert.Equal(t, someModule.IdentityString(), module.RemoteIdentity().IdentityString())
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

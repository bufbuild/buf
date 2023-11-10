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
	"time"

	"github.com/bufbuild/buf/private/buf/bufsync"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/git/gittest"
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestBackfilltags(t *testing.T) {
	t.Parallel()
	const defaultBranchName = "main"
	repo := gittest.ScaffoldGitRepository(t, gittest.ScaffoldGitRepositoryWithOnlyInitialCommit())
	moduleIdentityInHEAD, err := bufmoduleref.NewModuleIdentity("buf.build", "acme", "foo")
	require.NoError(t, err)
	prepareGitRepoBackfillTags(t, repo, moduleIdentityInHEAD)
	mockHandler := newMockSyncHandler()
	// prepare the top 5 commits as syncable commits, mark the rest as if they were already synced
	var (
		commitCount            int
		allCommitsHashes       []string
		fakeNowCommitLimitTime time.Time // to be sent as a fake clock and discard "old" commits
	)
	require.NoError(t, repo.ForEachCommit(func(commit git.Commit) error {
		allCommitsHashes = append(allCommitsHashes, commit.Hash().Hex())
		commitCount++
		if commitCount == 6 {
			// mark this commit as synced; nothing after this needs to be marked because syncer
			// won't travel past this
			mockHandler.setSyncPoint(
				defaultBranchName,
				commit.Hash(),
				moduleIdentityInHEAD,
			)
		}
		if commitCount == 15 {
			// have the time limit at the commit 15 looking back
			fakeNowCommitLimitTime = commit.Committer().Timestamp()
		}
		return nil
	}))
	const moduleDir = "." // module is at the git root repo
	syncer, err := bufsync.NewSyncer(
		zaptest.NewLogger(t),
		&mockClock{now: fakeNowCommitLimitTime.Add(bufsync.LookbackTimeLimit)},
		repo,
		storagegit.NewProvider(repo.Objects()),
		mockHandler,
		bufsync.SyncerWithModule(moduleDir, nil),
	)
	require.NoError(t, err)
	require.NoError(t, syncer.Sync(context.Background()))
	// in total the repo has at least 20 commits, we expect to backfill 11 of them
	// and sync the next 4 commits
	assert.GreaterOrEqual(t, len(allCommitsHashes), 20)
	assert.Len(t, mockHandler.tagsByHash, 15)
	// as follows:
	for i, commitHash := range allCommitsHashes {
		if i < 15 {
			// Between 0-4, the tags should be synced.
			// Between 5-15 the tags should be backfilled.
			//
			// The func it's backfilling more than 5 commits, because it needs to backfill until both
			// conditions are met, at least 5 commits and at least 24 hours.
			assert.Contains(t, mockHandler.tagsByHash, commitHash)
		} else {
			// past the #15 the commits are too old, we don't backfill back there
			assert.NotContains(t, mockHandler.tagsByHash, commitHash)
		}
	}
}

// prepareGitRepoBackfillTags adds 20 commits and tags in the default branch, one tag per commit. It
// waits 1s between commit 5 and 6 to be easily used as the lookback commit limit time.
func prepareGitRepoBackfillTags(t *testing.T, repo gittest.Repository, moduleIdentity bufmoduleref.ModuleIdentity) {
	var commitsCounter int
	doEmptyCommitAndTag := func(numOfCommits int) {
		for i := 0; i < numOfCommits; i++ {
			commitsCounter++
			repo.Commit(context.Background(), t, fmt.Sprintf("commit %d", commitsCounter), nil)
			repo.Tag(context.Background(), t, fmt.Sprintf("tag-%d", commitsCounter))
		}
	}
	// write the base module in the root
	repo.Commit(context.Background(), t, "commit 0", map[string]string{
		"buf.yaml":         fmt.Sprintf("version: v1\nname: %s\n", moduleIdentity.IdentityString()),
		"foo/v1/foo.proto": "syntax = \"proto3\";\n\npackage foo.v1;\n\nmessage Foo {}\n",
	})
	// commit and tag
	doEmptyCommitAndTag(5)
	time.Sleep(1 * time.Second)
	doEmptyCommitAndTag(15)
}

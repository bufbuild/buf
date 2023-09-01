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
	"fmt"
	"testing"
	"time"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestBackfilltags(t *testing.T) {
	t.Parallel()
	repo, repoDir := scaffoldGitRepository(t)
	moduleIdentityInHEAD, err := bufmoduleref.NewModuleIdentity("buf.build", "acme", "foo")
	require.NoError(t, err)
	prepareGitRepoBackfillTags(t, repoDir, moduleIdentityInHEAD)
	mockBSRChecker := newMockSyncGitChecker()
	// prepare the top 5 commits as syncable commits, mark the rest as if they were already synced
	var (
		commitCount            int
		allCommitsHashes       []string
		syncableCommits        []*syncableCommit
		fakeNowCommitLimitTime time.Time // to be sent as a fake clock and discard "old" commits
	)
	require.NoError(t, repo.ForEachCommit(func(commit git.Commit) error {
		allCommitsHashes = append(allCommitsHashes, commit.Hash().Hex())
		commitCount++
		if commitCount <= 5 {
			syncableCommits = append(syncableCommits, &syncableCommit{
				commit: commit,
				modules: map[string]*bufmodulebuild.BuiltModule{
					".": nil, // no need for built modules when backfilling tags
				},
			})
		} else {
			if commitCount == 15 {
				// have the time limit at the commit 15 looking back
				fakeNowCommitLimitTime = commit.Committer().Timestamp()
			}
			mockBSRChecker.markSynced(commit.Hash().Hex())
		}
		return nil
	}))
	// reverse syncable commits, to leave them in time order parent -> children
	// https://github.com/golang/go/wiki/SliceTricks#reversing
	for i := len(syncableCommits)/2 - 1; i >= 0; i-- {
		opp := len(syncableCommits) - 1 - i
		syncableCommits[i], syncableCommits[opp] = syncableCommits[opp], syncableCommits[i]
	}
	mockTagsBackfiller := newMockTagsBackfiller()
	mockClock := &mockClock{now: fakeNowCommitLimitTime.Add(lookbackTimeLimit)}
	testSyncer := syncer{
		repo:                                  repo,
		storageGitProvider:                    storagegit.NewProvider(repo.Objects()),
		logger:                                zaptest.NewLogger(t),
		sortedModulesDirsForSync:              []string{"."},
		modulesDirsToIdentityOverrideForSync:  map[string]bufmoduleref.ModuleIdentity{".": nil},
		syncedGitCommitChecker:                mockBSRChecker.checkFunc(),
		commitsToTags:                         make(map[string][]string),
		modulesDirsToBranchesToIdentities:     make(map[string]map[string]bufmoduleref.ModuleIdentity),
		modulesToBranchesExpectedSyncPoints:   make(map[string]map[string]string),
		modulesIdentitiesToCommitsSyncedCache: make(map[string]map[string]struct{}),
		tagsBackfiller:                        mockTagsBackfiller.backfillFunc(),
	}
	require.NoError(t, testSyncer.prepareSync(context.Background()))
	require.NoError(t, testSyncer.backfillTags(context.Background(), defaultBranch, syncableCommits, mockClock))
	// in total the repo has at least 20 commits, we expect to backfill 11 of them...
	assert.GreaterOrEqual(t, len(allCommitsHashes), 20)
	assert.Len(t, mockTagsBackfiller.backfilledCommitsToTags, 11)
	// as follows:
	for i, commitHash := range allCommitsHashes {
		if i < 4 {
			// the 4 most recent should not be backfilling anything, those are unsynced commits that will
			// be synced by another func.
			assert.NotContains(t, mockTagsBackfiller.backfilledCommitsToTags, commitHash)
		} else if i < 15 {
			// Between 5-15 the tags should be backfilled.
			//
			// The commit #5 is the git start sync point, which will also be handled by sync because it's
			// sometimes already synced and sometimes not. It's handled by both sync and backfill tags.
			//
			// The func it's backfilling more than 5 commits, because it needs to backfill until both
			// conditions are met, at least 5 commits and at least 24 hours.
			assert.Contains(t, mockTagsBackfiller.backfilledCommitsToTags, commitHash)
		} else {
			// past the #15 the commits are too old, we don't backfill back there
			assert.NotContains(t, mockTagsBackfiller.backfilledCommitsToTags, commitHash)
		}
	}
}

// prepareGitRepoBackfillTags adds 20 commits and tags in the default branch, one tag per commit. It
// waits 1s between commit 5 and 6 to be easily used as the lookback commit limit time.
func prepareGitRepoBackfillTags(t *testing.T, repoDir string, moduleIdentity bufmoduleref.ModuleIdentity) {
	runner := command.NewRunner()
	var commitsCounter int
	doEmptyCommitAndTag := func(numOfCommits int) {
		for i := 0; i < numOfCommits; i++ {
			commitsCounter++
			runInDir(
				t, runner, repoDir,
				"git", "commit", "--allow-empty",
				"-m", fmt.Sprintf("commit %d", commitsCounter),
			)
			runInDir(
				t, runner, repoDir,
				"git", "tag", fmt.Sprintf("tag-%d", commitsCounter),
			)
		}
	}
	// write the base module in the root
	writeFiles(t, repoDir, map[string]string{
		"buf.yaml":         fmt.Sprintf("version: v1\nname: %s\n", moduleIdentity.IdentityString()),
		"foo/v1/foo.proto": `syntax = "proto3";\n\npackage foo.v1;\n\nmessage Foo {}\n`,
	})
	runInDir(t, runner, repoDir, "git", "add", ".")
	runInDir(t, runner, repoDir, "git", "commit", "-m", "commit 0")
	// commit and tag
	doEmptyCommitAndTag(5)
	time.Sleep(1 * time.Second)
	doEmptyCommitAndTag(15)
	// push both commits and tags
	runInDir(t, runner, repoDir, "git", "push", "-u", "-f", remoteName, defaultBranch)
	runInDir(t, runner, repoDir, "git", "push", "--tags")
}

type mockTagsBackfiller struct {
	backfilledCommitsToTags map[string]struct{}
}

func newMockTagsBackfiller() mockTagsBackfiller {
	return mockTagsBackfiller{backfilledCommitsToTags: make(map[string]struct{})}
}

func (b *mockTagsBackfiller) backfillFunc() TagsBackfiller {
	return func(_ context.Context, _ bufmoduleref.ModuleIdentity, alreadySyncedHash git.Hash, _, _ git.Ident, _ []string) (string, error) {
		// we don't really test which tags were backfilled, only which commits had its tags backfilled
		b.backfilledCommitsToTags[alreadySyncedHash.Hex()] = struct{}{}
		return "some-BSR-commit-name", nil
	}
}

type mockClock struct {
	now time.Time
}

func (c *mockClock) Now() time.Time { return c.now }

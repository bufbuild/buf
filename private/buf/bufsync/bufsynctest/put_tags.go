package bufsynctest

import (
	"context"
	"fmt"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufsync"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/git/gittest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testPutTags(t *testing.T, handler TestHandler, run runFunc) {
	repo := gittest.ScaffoldGitRepository(t)
	moduleIdentityInHEAD, err := bufmoduleref.NewModuleIdentity("buf.build", "acme", "foo")
	require.NoError(t, err)
	prepareGitRepoWithTags(t, repo, moduleIdentityInHEAD)
	// prepare the top 5 commits as syncable commits, mark the rest as if they were already synced
	var (
		previousHeadIndex = 6
		commitCount       int
		allCommitsHashes  []string
	)
	require.NoError(t, repo.ForEachCommit(func(commit git.Commit) error {
		allCommitsHashes = append(allCommitsHashes, commit.Hash().Hex())
		if commitCount == previousHeadIndex {
			// mark this commit as synced; nothing after this needs to be marked because syncer
			// won't travel past this
			handler.SetSyncPoint(
				context.Background(),
				t,
				moduleIdentityInHEAD,
				repo.DefaultBranch(),
				commit.Hash(),
			)
		}
		commitCount++
		return nil
	}))
	plan, err := run(t, repo, bufsync.SyncerWithModule(".", nil))
	require.NoError(t, err)
	require.Len(t, plan.ModuleTagsToSync(), 1)
	moduleTags := plan.ModuleTagsToSync()[0]
	// In total the repo has at least 20 commits; we manually marked index 6 as the synced point,
	// so we expect to sync commits where index < 6. At the end, we should have 6+1 commits synced.
	// For those 6+1 commits, we expect their tags to be put. All other tags are not put because
	// they point to unsynced commits.
	assert.GreaterOrEqual(t, len(allCommitsHashes), 20)
	require.Len(t, moduleTags.TaggedCommitsToSync(), previousHeadIndex+1)
	var syncedCommits []string
	for _, taggedCommit := range moduleTags.TaggedCommitsToSync() {
		syncedCommits = append(syncedCommits, taggedCommit.Commit().Hash().Hex())
	}
	for i, commitHash := range allCommitsHashes {
		if i < previousHeadIndex+1 {
			assert.Contains(t, syncedCommits, commitHash)
		} else {
			assert.NotContains(t, syncedCommits, commitHash)
		}
	}
}

// prepareGitRepoWithTags adds 20 commits and tags in the default branch, one tag per commit.
func prepareGitRepoWithTags(t *testing.T, repo gittest.Repository, moduleIdentity bufmoduleref.ModuleIdentity) {
	var commitsCounter int
	doEmptyCommitAndTag := func(numOfCommits int) {
		for i := 0; i < numOfCommits; i++ {
			commitsCounter++
			repo.Commit(t, fmt.Sprintf("commit %d", commitsCounter), nil)
			repo.Tag(t, fmt.Sprintf("tag-%d", commitsCounter), "")
		}
	}
	// write the base module in the root
	repo.Commit(t, "commit 0", map[string]string{
		"buf.yaml":         fmt.Sprintf("version: v1\nname: %s\n", moduleIdentity.IdentityString()),
		"foo/v1/foo.proto": "syntax = \"proto3\";\n\npackage foo.v1;\n\nmessage Foo {}\n",
	})
	// commit and tag
	doEmptyCommitAndTag(20)
}

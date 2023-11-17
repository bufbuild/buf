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

package bufsynctest

import (
	"fmt"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufsync"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/git/gittest"
	"github.com/stretchr/testify/require"
)

func testResumeBranchOverlapWithSyncedBranches(t *testing.T, handler TestHandler, run runFunc) {
	repo := gittest.ScaffoldGitRepository(t)
	moduleIdentityInHEAD, err := bufmoduleref.NewModuleIdentity("buf.build", "acme", "foo")
	require.NoError(t, err)
	var commitsCounter int
	doEmptyCommitAndTag := func(numOfCommits int) {
		for i := 0; i < numOfCommits; i++ {
			commitsCounter++
			repo.Commit(t, fmt.Sprintf("commit %d", commitsCounter), nil)
			repo.Tag(t, fmt.Sprintf("tag-%d", commitsCounter), "")
		}
	}
	const (
		originalBranch = "originalBranch"
		forkedBranch   = "forkedBranch"
	)

	// (1) checkout new branch, commit module + 5 commits, _don't_ sync;
	// 6 commits would have been be synced if we synced this branch
	repo.CheckoutB(t, originalBranch)
	repo.Commit(t, "commit 0", map[string]string{
		"buf.yaml":         fmt.Sprintf("version: v1\nname: %s\n", moduleIdentityInHEAD.IdentityString()),
		"foo/v1/foo.proto": "syntax = \"proto3\";\n\npackage foo.v1;\n\nmessage Foo {}\n",
	})
	doEmptyCommitAndTag(5)

	// (2) fork branch, commit twice, sync;
	// 6 commits from default branch + 2 commits here are synced
	repo.CheckoutB(t, forkedBranch)
	doEmptyCommitAndTag(2)
	plan, err := run(t, repo, bufsync.SyncerWithModule(".", nil))
	require.NoError(t, err)
	require.Len(t, plan.ModuleBranchesToSync(), 1)
	assertCommitsSynced(
		t,
		plan.ModuleBranchesToSync()[0],
		"commit 0",
		"commit 1",
		"commit 2",
		"commit 3",
		"commit 4",
		"commit 5",
		"commit 6",
		"commit 7",
	)
	require.Len(t, plan.ModuleTagsToSync(), 1)
	assertTagsSynced(t,
		plan.ModuleTagsToSync()[0],
		"tag-1",
		"tag-2",
		"tag-3",
		"tag-4",
		"tag-5",
		"tag-6",
		"tag-7",
	)

	// (3) go back to original branch, commit 4 more times and finally sync this branch;
	// 5 commits should be synced
	repo.Checkout(t, originalBranch)
	doEmptyCommitAndTag(4)
	plan, err = run(t, repo, bufsync.SyncerWithModule(".", nil))
	require.NoError(t, err)
	require.Len(t, plan.ModuleBranchesToSync(), 1)
	assertCommitsSynced(t,
		plan.ModuleBranchesToSync()[0],
		"commit 5",
		"commit 8",
		"commit 9",
		"commit 10",
		"commit 11",
	)
	require.Len(t, plan.ModuleTagsToSync(), 1)
	assertTagsSynced(t,
		plan.ModuleTagsToSync()[0],
		"tag-1",
		"tag-2",
		"tag-3",
		"tag-4",
		"tag-5",
		"tag-8",
		"tag-9",
		"tag-10",
		"tag-11",
	)
}

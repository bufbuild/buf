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
	"testing"

	"github.com/bufbuild/buf/private/buf/bufsync"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/git/gittest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testNewRemoteBranchUnprotectedOverlapWithAnotherSyncedBranch(t *testing.T, run runFunc) {
	gitRepo := gittest.ScaffoldGitRepository(t)
	module1 := doCommitRandomModule(t, gitRepo, ".", nil)
	opts := []bufsync.SyncerOption{
		bufsync.SyncerWithModule(".", module1),
	}
	var counter int
	doEmptyCommits(t, gitRepo, 2, &counter)
	_, err := run(t, gitRepo, opts...)
	require.NoError(t, err)
	gitRepo.CheckoutB(t, "otherbranch")
	doEmptyCommits(t, gitRepo, 5, &counter)

	plan, err := run(t, gitRepo, opts...)

	require.NoError(t, err)
	assert.False(t, plan.Nop())
	assertPlanForModuleBranch(
		t, plan, module1, "otherbranch",
		"commit-2", // last synced commit from main branch
		"commit-3", // the rest of the commits on this branch
		"commit-4",
		"commit-5",
		"commit-6",
		"commit-7",
	)
	assertPlanForModuleTags(
		t, plan, module1,
		"module-"+module1.IdentityString(), // previously synced commits for this module
		"commit-1",
		"commit-2", // last synced commit from main branch
		"commit-3", // the rest of the commits on this branch
		"commit-4",
		"commit-5",
		"commit-6",
		"commit-7",
	)
}

func testNewRemoteBranchUnprotectedNoOverlapWithAnySyncedBranch(t *testing.T, run runFunc) {
	gitRepo := gittest.ScaffoldGitRepository(t)
	module1 := doCommitRandomModule(t, gitRepo, ".", nil)
	opts := []bufsync.SyncerOption{
		bufsync.SyncerWithModule(".", module1),
	}
	var counter int
	doEmptyCommits(t, gitRepo, 2, &counter)
	gitRepo.CheckoutB(t, "otherbranch")
	doEmptyCommits(t, gitRepo, 5, &counter)

	plan, err := run(t, gitRepo, opts...)

	require.NoError(t, err)
	assert.False(t, plan.Nop())
	assertPlanForModuleBranch(
		t, plan, module1, "otherbranch",
		"module-"+module1.IdentityString(), // all commits all the way back
		"commit-1",
		"commit-2",
		"commit-3",
		"commit-4",
		"commit-5",
		"commit-6",
		"commit-7",
	)
	assertPlanForModuleTags(
		t, plan, module1,
		"module-"+module1.IdentityString(), // all synced commits all the way back
		"commit-1",
		"commit-2", // last synced commit from main branch
		"commit-3", // the rest of the commits on this branch
		"commit-4",
		"commit-5",
		"commit-6",
		"commit-7",
	)
}

func testNewRemoteBranchProtectedNotReleaseBranch(t *testing.T, run runFunc) {
	gitRepo := gittest.ScaffoldGitRepository(t)
	module1 := doCommitRandomModule(t, gitRepo, ".", nil)
	opts := []bufsync.SyncerOption{
		bufsync.SyncerWithModule(".", module1),
	}
	var counter int
	doEmptyCommits(t, gitRepo, 2, &counter)
	gitRepo.CheckoutB(t, OtherProtectedBranchName)
	doEmptyCommits(t, gitRepo, 5, &counter)

	plan, err := run(t, gitRepo, opts...)

	require.NoError(t, err)
	assert.False(t, plan.Nop())
	assertPlanForModuleBranch(
		t, plan, module1, OtherProtectedBranchName,
		"module-"+module1.IdentityString(), // all commits all the way back
		"commit-1",
		"commit-2",
		"commit-3",
		"commit-4",
		"commit-5",
		"commit-6",
		"commit-7",
	)
	assertPlanForModuleTags(
		t, plan, module1,
		"module-"+module1.IdentityString(), // all synced commits all the way back
		"commit-1",
		"commit-2", // last synced commit from main branch
		"commit-3", // the rest of the commits on this branch
		"commit-4",
		"commit-5",
		"commit-6",
		"commit-7",
	)
}

func testNewRemoteBranchProtectedReleaseBranchEmpty(t *testing.T, run runFunc) {
	gitRepo := gittest.ScaffoldGitRepository(t)
	module1 := doCommitRandomModule(t, gitRepo, ".", nil)
	opts := []bufsync.SyncerOption{
		bufsync.SyncerWithModule(".", module1),
	}
	var counter int
	doEmptyCommits(t, gitRepo, 2, &counter)
	gitRepo.CheckoutB(t, ReleaseBranchName)
	doEmptyCommits(t, gitRepo, 5, &counter)

	plan, err := run(t, gitRepo, opts...)

	require.NoError(t, err)
	assert.False(t, plan.Nop())
	assertPlanForModuleBranch(
		t, plan, module1, ReleaseBranchName,
		"module-"+module1.IdentityString(), // all commits all the way back
		"commit-1",
		"commit-2",
		"commit-3",
		"commit-4",
		"commit-5",
		"commit-6",
		"commit-7",
	)
	assertPlanForModuleTags(
		t, plan, module1,
		"module-"+module1.IdentityString(), // all synced commits all the way back
		"commit-1",
		"commit-2", // last synced commit from main branch
		"commit-3", // the rest of the commits on this branch
		"commit-4",
		"commit-5",
		"commit-6",
		"commit-7",
	)
}

func testNewRemoteBranchProtectedReleaseBranchNotEmptyContentMatch(t *testing.T, handler TestHandler, run runFunc) {
	gitRepo := gittest.ScaffoldGitRepository(t)
	module1 := doCommitRandomModule(t, gitRepo, ".", nil)
	opts := []bufsync.SyncerOption{
		bufsync.SyncerWithModule(".", module1),
	}
	var counter int
	doEmptyCommits(t, gitRepo, 2, &counter)
	gitRepo.CheckoutB(t, ReleaseBranchName)
	doEmptyCommits(t, gitRepo, 5, &counter)
	headCommit, err := gitRepo.HEADCommit(git.HEADCommitWithBranch(ReleaseBranchName))
	require.NoError(t, err)
	doManualPushCommit(t, handler, gitRepo, module1, ".", "", headCommit)
	doRandomUpdateToModule(t, gitRepo, ".", &counter)

	plan, err := run(t, gitRepo, opts...)

	require.NoError(t, err)
	assert.False(t, plan.Nop())
	assertPlanForModuleBranch(
		t, plan, module1, ReleaseBranchName,
		"commit-7",        // content matched to second last commit
		"change-module-8", // latest head, different content
	)
	assertPlanForModuleTags(
		t, plan, module1,
		"commit-7",
		"change-module-8",
	)
}

func testNewRemoteBranchProtectedReleaseBranchNotEmptyNoContentMatch(t *testing.T, handler TestHandler, run runFunc) {
	gitRepo := gittest.ScaffoldGitRepository(t)
	module1 := doCommitRandomModule(t, gitRepo, ".", nil)
	opts := []bufsync.SyncerOption{
		bufsync.SyncerWithModule(".", module1),
	}
	var counter int
	doEmptyCommits(t, gitRepo, 2, &counter)
	gitRepo.CheckoutB(t, ReleaseBranchName)
	doEmptyCommits(t, gitRepo, 5, &counter)
	// checkout other branch and manual push from there
	gitRepo.CheckoutB(t, "foo")
	doRandomUpdateToModule(t, gitRepo, ".", &counter)
	headCommit, err := gitRepo.HEADCommit(git.HEADCommitWithBranch(ReleaseBranchName))
	doManualPushCommit(t, handler, gitRepo, module1, ".", "", headCommit)
	require.NoError(t, err)
	// go back to release branch
	gitRepo.Checkout(t, ReleaseBranchName)

	plan, err := run(t, gitRepo, opts...)

	require.NoError(t, err)
	assert.False(t, plan.Nop())
	assertPlanForModuleBranch(
		t, plan, module1, ReleaseBranchName,
		"commit-7", // no content match, HEAD only
	)
	assertPlanForModuleTags(
		t, plan, module1,
		"commit-7",
	)
}

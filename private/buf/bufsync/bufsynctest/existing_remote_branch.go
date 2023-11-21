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

func testExistingRemoteBranchNotPreviouslySyncedContentMatch(t *testing.T, handler TestHandler, run runFunc) {
	repo := gittest.ScaffoldGitRepository(t)
	repo.CheckoutB(t, "randombranch")
	module1 := doCommitRandomModule(t, repo, ".", nil)
	opts := []bufsync.SyncerOption{
		bufsync.SyncerWithModule(".", module1),
	}
	var counter int
	doEmptyCommits(t, repo, 2, &counter)
	doRandomUpdateToModule(t, repo, ".", &counter)
	headCommit, err := repo.HEADCommit(git.HEADCommitWithBranch("randombranch"))
	require.NoError(t, err)
	doManualPushCommit(t, handler, repo, module1, ".", "randombranch", headCommit)
	doRandomUpdateToModule(t, repo, ".", &counter)
	doRandomUpdateToModule(t, repo, ".", &counter)

	plan, err := run(t, repo, opts...)

	require.NoError(t, err)
	assert.False(t, plan.Nop())
	assertPlanForModuleBranch(
		t, plan, module1, "randombranch",
		"change-module-3", // content matched commit
		"change-module-4", // all commits after that
		"change-module-5",
	)
	assertPlanForModuleTags(
		t, plan, module1,
		"change-module-3",
		"change-module-4",
		"change-module-5",
	)
}

func testExistingRemoteBranchNotPreviouslySyncedNoContentMatch(t *testing.T, handler TestHandler, run runFunc) {
	repo := gittest.ScaffoldGitRepository(t)
	repo.CheckoutB(t, "randombranch")
	module1 := doCommitRandomModule(t, repo, ".", nil)
	opts := []bufsync.SyncerOption{
		bufsync.SyncerWithModule(".", module1),
	}
	var counter int
	doEmptyCommits(t, repo, 3, &counter)
	doRandomUpdateToModule(t, repo, ".", &counter)
	doRandomUpdateToModule(t, repo, ".", &counter)
	doManualPushRandomModule(t, handler, module1, "randombranch", &counter)

	plan, err := run(t, repo, opts...)

	require.NoError(t, err)
	assert.False(t, plan.Nop())
	assertPlanForModuleBranch(
		t, plan, module1, "randombranch",
		"change-module-5", // head only
	)
	assertPlanForModuleTags(
		t, plan, module1,
		"change-module-5", // head only
	)
}

func testExistingRemoteBranchPreviouslySyncedProtectedFailsProtection(t *testing.T, handler TestHandler, run runFunc) {
	repo := gittest.ScaffoldGitRepository(t)
	module1 := doCommitRandomModule(t, repo, ".", nil)
	opts := []bufsync.SyncerOption{
		bufsync.SyncerWithModule(".", module1),
	}
	var counter int
	doEmptyCommits(t, repo, 3, &counter)
	_, err := run(t, repo, opts...)
	require.NoError(t, err)
	repo.ResetHard(t, "HEAD~1")
	doRandomUpdateToModule(t, repo, ".", &counter)

	_, err = run(t, repo, opts...)

	require.Error(t, err)
	assert.Contains(t, err.Error(), `history on protected branch "master" has diverged`)
}

func testExistingRemoteBranchPreviouslySyncedProtectedPassesProtection(t *testing.T, run runFunc) {
	repo := gittest.ScaffoldGitRepository(t)
	module1 := doCommitRandomModule(t, repo, ".", nil)
	opts := []bufsync.SyncerOption{
		bufsync.SyncerWithModule(".", module1),
	}
	var counter int
	doEmptyCommits(t, repo, 3, &counter)
	_, err := run(t, repo, opts...)
	require.NoError(t, err)
	doRandomUpdateToModule(t, repo, ".", &counter)

	plan, err := run(t, repo, opts...)

	require.NoError(t, err)
	assert.False(t, plan.Nop())
	assertPlanForModuleBranch(
		t, plan, module1, gittest.DefaultBranch,
		"commit-3",        // last synced commit
		"change-module-4", // unsynced but will be synced commit
	)
	assertPlanForModuleTags(
		t, plan, module1,
		"module-"+module1.IdentityString(), // previously synced commits
		"commit-1",
		"commit-2",
		"commit-3",        // last synced commit
		"change-module-4", // unsynced but will be synced commit
	)
}

func testExistingRemoteBranchPreviouslySyncedUnprotectedOverlapWithAnotherSyncedBranch(t *testing.T, run runFunc) {
	repo := gittest.ScaffoldGitRepository(t)
	repo.CheckoutB(t, "basebranch")
	module1 := doCommitRandomModule(t, repo, ".", nil)
	opts := []bufsync.SyncerOption{
		bufsync.SyncerWithModule(".", module1),
	}
	var counter int
	doEmptyCommits(t, repo, 3, &counter)
	_, err := run(t, repo, opts...)
	require.NoError(t, err)
	doEmptyCommits(t, repo, 3, &counter)
	repo.CheckoutB(t, "otherbranch")
	doRandomUpdateToModule(t, repo, ".", &counter)
	_, err = run(t, repo, opts...)
	require.NoError(t, err)
	repo.Checkout(t, "basebranch")

	plan, err := run(t, repo, opts...)

	require.NoError(t, err)
	assert.False(t, plan.Nop())
	assertPlanForModuleBranch(
		t, plan, module1, "basebranch",
		"commit-6", // last synced commit, synced by otherbranch
	)
	assertPlanForModuleTags(
		t, plan, module1,
		"module-"+module1.IdentityString(), // commits previously synced by basebranch
		"commit-1",
		"commit-2",
		"commit-3",
		"commit-4", // commits previously synced by otherbranch
		"commit-5",
		"commit-6",
	)
}

func testExistingRemoteBranchPreviouslySyncedUnprotectedNoOverlapWithAnySyncedBranchContentMatch(t *testing.T, run runFunc) {
	repo := gittest.ScaffoldGitRepository(t)
	repo.CheckoutB(t, "basebranch")
	module1 := doCommitRandomModule(t, repo, ".", nil)
	opts := []bufsync.SyncerOption{
		bufsync.SyncerWithModule(".", module1),
	}
	var counter int
	doEmptyCommits(t, repo, 3, &counter)
	_, err := run(t, repo, opts...)
	require.NoError(t, err)
	// remove all commits and recreate again
	repo.ResetHard(t, "HEAD~4")
	doCommitRandomModule(t, repo, ".", module1) // put back original module
	doEmptyCommits(t, repo, 3, &counter)

	plan, err := run(t, repo, opts...)

	require.NoError(t, err)
	assert.False(t, plan.Nop())
	assertPlanForModuleBranch(
		t, plan, module1, "basebranch",
		"module-"+module1.IdentityString(), // content-matched commit
		"commit-4",
		"commit-5",
		"commit-6",
	)
	assertPlanForModuleTags(
		t, plan, module1,
		"module-"+module1.IdentityString(),
		"commit-4",
		"commit-5",
		"commit-6",
	)
}

func testExistingRemoteBranchPreviouslySyncedUnprotectedNoOverlapWithAnySyncedBranchNoContentMatch(t *testing.T, run runFunc) {
	repo := gittest.ScaffoldGitRepository(t)
	repo.CheckoutB(t, "basebranch")
	module1 := doCommitRandomModule(t, repo, ".", nil)
	opts := []bufsync.SyncerOption{
		bufsync.SyncerWithModule(".", module1),
	}
	var counter int
	doEmptyCommits(t, repo, 3, &counter)
	_, err := run(t, repo, opts...)
	require.NoError(t, err)
	// remove all commits and recreate again
	repo.ResetHard(t, "HEAD~4")
	doRandomUpdateToModule(t, repo, ".", &counter) // put some module files first
	doCommitRandomModule(t, repo, ".", module1)    // then put back original module, but content won't match
	doEmptyCommits(t, repo, 3, &counter)

	plan, err := run(t, repo, opts...)

	require.NoError(t, err)
	assert.False(t, plan.Nop())
	assertPlanForModuleBranch(
		t, plan, module1, "basebranch",
		"commit-7", // content-matched commit
	)
	assertPlanForModuleTags(
		t, plan, module1,
		"commit-7", // content-matched commit
	)
}

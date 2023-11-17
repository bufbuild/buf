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
	"context"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufsync"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestHandler is a bufsync.Handler with a few helpful utilities for tests to set up
// and assert some state.
type TestHandler interface {
	bufsync.Handler
	SetSyncPoint(
		ctx context.Context,
		t *testing.T,
		targetModuleIdentity bufmoduleref.ModuleIdentity,
		branchName string,
		gitHash git.Hash,
	)
}

// RunTestSuite runs a set of test cases using Syncer with the provided TestHandler. Use
// this test suite to ensure compliance with Sync behaviour.
//
// The following behaviour is asserted:
//
// # Local validation
//
//   - Prevent duplicate module identities across module dirs on a branch.
//
// # Syncing commits
//
//	CONDITION							RESUME FROM (-> means fallback)
//	new remote branch:
//		unprotected:					any synced commit from any branch -> START of branch
//		protected:
//			not release branch:			START of branch
//			release lineage:
//				empty:					START of branch
//				not empty:				content match(HEAD of Release) -> HEAD of branch
//	existing remote branch:
//		not previously synced:			content match -> HEAD of branch
//		previously synced:
//			protected:					protect branch && any synced commit from branch -> error
//			unprotected:				any synced commit from any branch -> content match -> HEAD of branch
func RunTestSuite(t *testing.T, handlerProvider func() TestHandler) {
	t.Run("duplicate_identities", func(t *testing.T) {
		t.Parallel()
		handler := handlerProvider()
		testDuplicateIdentities(t, handler, makeRunFunc(handler))
	})
	// TODO: implement test cases
	t.Run("new_remote_branch", func(t *testing.T) {
		t.Run("unprotected", func(t *testing.T) {
			t.Run("overlap_with_another_synced_branch", func(t *testing.T) {})
			t.Run("no_overlap_with_any_synced_branch", func(t *testing.T) {})
		})
		t.Run("protected", func(t *testing.T) {
			t.Run("not_release_branch", func(t *testing.T) {})
			t.Run("release_branch", func(t *testing.T) {
				t.Run("empty", func(t *testing.T) {})
				t.Run("not_empty", func(t *testing.T) {
					t.Run("content_match", func(t *testing.T) {})
					t.Run("no_content_match", func(t *testing.T) {})
				})
			})
		})
	})
	t.Run("existing_remote_branch", func(t *testing.T) {
		t.Run("not_previously_synced", func(t *testing.T) {
			t.Run("content_match", func(t *testing.T) {})
			t.Run("no_content_match", func(t *testing.T) {})
		})
		t.Run("previously_synced", func(t *testing.T) {
			t.Run("protected", func(t *testing.T) {
				t.Run("fails_protection", func(t *testing.T) {})
				t.Run("passes_protection", func(t *testing.T) {})
			})
			t.Run("unprotected", func(t *testing.T) {
				t.Run("overlap_with_another_synced_branch", func(t *testing.T) {})
				t.Run("content_match", func(t *testing.T) {})
				t.Run("no_content_match", func(t *testing.T) {})
			})
		})
	})
	t.Run("new_branches_forking_off_of_synced_branches", func(t *testing.T) {
		t.Parallel()
		handler := handlerProvider()
		testNewBranchesForkingOffOfSyncedBranches(t, handler, makeRunFunc(handler))
	})
	t.Run("resume_branch_no_overlap", func(t *testing.T) {
		t.Parallel()
		handler := handlerProvider()
		testResumeBranchNoOverlapWithSyncedBranches(t, handler, makeRunFunc(handler))
	})
	t.Run("resume_branch_overlaps_sync_branch", func(t *testing.T) {
		t.Parallel()
		handler := handlerProvider()
		testResumeBranchOverlapWithSyncedBranches(t, handler, makeRunFunc(handler))
	})
	t.Run("resume_protected_branch_overlaps_sync_branch", func(t *testing.T) {
		t.Parallel()
		handler := handlerProvider()
		testResumeProtectedBranchOverlapWithSyncedBranches(t, handler, makeRunFunc(handler))
	})
}

// runFunc runs Plan and Sync on the provided Repository with the provided options, returning any error that occurred along the way.
// If Plan errors, Sync is not invoked.
type runFunc func(t *testing.T, repo git.Repository, options ...bufsync.SyncerOption) (bufsync.ExecutionPlan, error)

func makeRunFunc(handler bufsync.Handler) runFunc {
	return func(t *testing.T, repo git.Repository, options ...bufsync.SyncerOption) (bufsync.ExecutionPlan, error) {
		syncer, err := bufsync.NewSyncer(
			zaptest.NewLogger(t),
			repo,
			storagegit.NewProvider(repo.Objects()),
			handler,
			options...,
		)
		require.NoError(t, err)
		plan, err := syncer.Plan(context.Background())
		if err != nil {
			return plan, err
		}
		return plan, syncer.Sync(context.Background())
	}
}

func assertTagsSynced(t *testing.T, moduleTags bufsync.ModuleTags, expectedTags ...string) {
	t.Helper()
	var syncedTags []string
	for _, taggedCommit := range moduleTags.TaggedCommitsToSync() {
		syncedTags = append(syncedTags, taggedCommit.Tags()...)
	}
	assert.Len(t, syncedTags, len(expectedTags))
	assert.ElementsMatch(t, expectedTags, syncedTags)
}
func assertCommitsSynced(t *testing.T, moduleBranch bufsync.ModuleBranch, expectedMessages ...string) {
	t.Helper()
	var syncedCommitMessages []string
	for _, syncedCommit := range moduleBranch.CommitsToSync() {
		syncedCommitMessages = append(syncedCommitMessages, syncedCommit.Commit().Message())
	}
	assert.Len(t, syncedCommitMessages, len(expectedMessages))
	assert.ElementsMatch(t, expectedMessages, syncedCommitMessages)
}

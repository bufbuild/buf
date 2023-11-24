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
	"fmt"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufsync"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufcas/bufcasalpha"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/git/gittest"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

const (
	ReleaseBranchName        = "__release__"
	OtherProtectedBranchName = "__protected__"
)

// TestHandler is a bufsync.Handler with a few helpful utilities for tests to set up
// and assert some state.
type TestHandler interface {
	bufsync.Handler
	ManuallyPushModule(
		ctx context.Context,
		t *testing.T,
		targetModuleFullName bufmodule.ModuleFullName,
		branchName string,
		manifest *modulev1alpha1.Blob,
		blobs []*modulev1alpha1.Blob,
	)
}

// RunTestSuite runs a set of test cases using Syncer with the provided TestHandler. Use
// this test suite to ensure compliance with Sync behavior.
//
// The following behavior is asserted:
//
// # Local validation
//
//   - Prevent duplicate module identities across module dirs on a branch.
//
// # Syncing commits
//
//	CONDITION                        RESUME FROM (-> means fallback)
//	new remote branch:
//	  unprotected:                   any synced commit from any branch -> START of branch
//	  protected:
//	    not release branch:          START of branch
//	    release lineage:
//	      empty:                     START of branch
//	      not empty:                 content match(HEAD of Release) -> HEAD of branch
//	existing remote branch:
//	  not previously synced:         content match -> HEAD of branch
//	  previously synced:
//	    protected:                   protect branch && any synced commit from branch -> error
//	    unprotected:                 any synced commit from any branch -> content match -> HEAD of branch
func RunTestSuite(t *testing.T, handlerProvider func() TestHandler) {
	t.Run("duplicate_identities", func(t *testing.T) {
		t.Parallel()
		handler := handlerProvider()
		testDuplicateIdentities(t, handler, makeRunFunc(handler))
	})
	t.Run("new_remote_branch", func(t *testing.T) {
		t.Parallel()
		t.Run("unprotected", func(t *testing.T) {
			t.Parallel()
			t.Run("overlap_with_another_synced_branch", func(t *testing.T) {
				t.Parallel()
				testNewRemoteBranchUnprotectedOverlapWithAnotherSyncedBranch(t, makeRunFunc(handlerProvider()))
			})
			t.Run("no_overlap_with_any_synced_branch", func(t *testing.T) {
				t.Parallel()
				testNewRemoteBranchUnprotectedNoOverlapWithAnySyncedBranch(t, makeRunFunc(handlerProvider()))
			})
		})
		t.Run("protected", func(t *testing.T) {
			t.Parallel()
			t.Run("not_release_branch", func(t *testing.T) {
				t.Parallel()
				testNewRemoteBranchProtectedNotReleaseBranch(t, makeRunFunc(handlerProvider()))
			})
			t.Run("release_branch", func(t *testing.T) {
				t.Parallel()
				t.Run("empty", func(t *testing.T) {
					t.Parallel()
					testNewRemoteBranchProtectedReleaseBranchEmpty(t, makeRunFunc(handlerProvider()))
				})
				t.Run("not_empty", func(t *testing.T) {
					t.Parallel()
					t.Run("content_match", func(t *testing.T) {
						t.Parallel()
						handler := handlerProvider()
						testNewRemoteBranchProtectedReleaseBranchNotEmptyContentMatch(t, handler, makeRunFunc(handler))
					})
					t.Run("no_content_match", func(t *testing.T) {
						t.Parallel()
						handler := handlerProvider()
						testNewRemoteBranchProtectedReleaseBranchNotEmptyNoContentMatch(t, handler, makeRunFunc(handler))
					})
				})
			})
		})
	})
	t.Run("existing_remote_branch", func(t *testing.T) {
		t.Parallel()
		t.Run("not_previously_synced", func(t *testing.T) {
			t.Parallel()
			t.Run("content_match", func(t *testing.T) {
				t.Parallel()
				handler := handlerProvider()
				testExistingRemoteBranchNotPreviouslySyncedContentMatch(t, handler, makeRunFunc(handler))
			})
			t.Run("no_content_match", func(t *testing.T) {
				t.Parallel()
				handler := handlerProvider()
				testExistingRemoteBranchNotPreviouslySyncedNoContentMatch(t, handler, makeRunFunc(handler))
			})
		})
		t.Run("previously_synced", func(t *testing.T) {
			t.Parallel()
			t.Run("protected", func(t *testing.T) {
				t.Parallel()
				t.Run("fails_protection", func(t *testing.T) {
					t.Parallel()
					handler := handlerProvider()
					testExistingRemoteBranchPreviouslySyncedProtectedFailsProtection(t, handler, makeRunFunc(handler))
				})
				t.Run("passes_protection", func(t *testing.T) {
					t.Parallel()
					testExistingRemoteBranchPreviouslySyncedProtectedPassesProtection(t, makeRunFunc(handlerProvider()))
				})
			})
			t.Run("unprotected", func(t *testing.T) {
				t.Parallel()
				t.Run("overlap_with_another_synced_branch", func(t *testing.T) {
					t.Parallel()
					testExistingRemoteBranchPreviouslySyncedUnprotectedOverlapWithAnotherSyncedBranch(t, makeRunFunc(handlerProvider()))
				})
				t.Run("no_overlap_with_any_synced_branch", func(t *testing.T) {
					t.Parallel()
					t.Run("content_match", func(t *testing.T) {
						t.Parallel()
						testExistingRemoteBranchPreviouslySyncedUnprotectedNoOverlapWithAnySyncedBranchContentMatch(t, makeRunFunc(handlerProvider()))
					})
					t.Run("no_content_match", func(t *testing.T) {
						t.Parallel()
						testExistingRemoteBranchPreviouslySyncedUnprotectedNoOverlapWithAnySyncedBranchNoContentMatch(t, makeRunFunc(handlerProvider()))
					})
				})
			})
		})
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

func doCommitRandomModule(
	t *testing.T,
	repo gittest.Repository,
	dir string,
	moduleFullName bufmodule.ModuleFullName,
) bufmodule.ModuleFullName {
	if moduleFullName == nil {
		moduleName, err := uuidutil.New()
		require.NoError(t, err)
		moduleFullName, err = bufmodule.NewModuleFullName("buf.build", "acme", moduleName.String())
		require.NoError(t, err)
	}
	repo.Commit(t, "module-"+moduleFullName.String(), map[string]string{
		filepath.Join(dir, "buf.yaml"):  fmt.Sprintf("version: v1\nname: %s\n", moduleFullName.String()),
		filepath.Join(dir, "foo.proto"): `syntax="proto3"; package buf;`,
	})
	repo.Tag(t, "module/"+moduleFullName.String(), "")
	return moduleFullName
}

func doRandomUpdateToModule(t *testing.T, repo gittest.Repository, dir string, counter *int) {
	*counter++
	repo.Commit(t, fmt.Sprintf("change-module-%d", *counter), map[string]string{
		filepath.Join(dir, fmt.Sprintf("foo_%d.proto", *counter)): fmt.Sprintf(`syntax="proto3"; package buf_%d;`, *counter),
	})
	repo.Tag(t, fmt.Sprintf("tag-%d", *counter), "")
}

func doManualPushCommit(
	t *testing.T,
	handler TestHandler,
	repo gittest.Repository,
	targetModuleFullName bufmodule.ModuleFullName,
	moduleDir string,
	branch string,
	commit git.Commit,
) {
	commitBucket, err := storagegit.NewProvider(repo.Objects()).NewReadBucket(commit.Tree())
	require.NoError(t, err)
	moduleBucket := storage.MapReadBucket(commitBucket, storage.MapOnPrefix(moduleDir))
	sourceConfig, err := bufconfig.GetConfigForBucket(context.Background(), moduleBucket)
	require.NoError(t, err)
	builtModule, err := bufmodulebuild.NewModuleBucketBuilder().BuildForBucket(
		context.Background(),
		moduleBucket,
		sourceConfig.Build,
		bufmodulebuild.WithModuleFullName(sourceConfig.ModuleFullName),
	)
	require.NoError(t, err)
	fileSet, err := bufcas.NewFileSetForBucket(context.Background(), builtModule.Bucket)
	require.NoError(t, err)
	protoManifestBlob, protoBlobs, err := bufcas.FileSetToProtoManifestBlobAndBlobs(fileSet)
	require.NoError(t, err)
	handler.ManuallyPushModule(
		context.Background(),
		t,
		targetModuleFullName,
		branch,
		bufcasalpha.BlobToAlpha(protoManifestBlob),
		bufcasalpha.BlobsToAlpha(protoBlobs),
	)
}
func doManualPushRandomModule(
	t *testing.T,
	handler TestHandler,
	targetModuleFullName bufmodule.ModuleFullName,
	branch string,
	counter *int,
) {
	*counter++
	bucket, err := storagemem.NewReadBucket(map[string][]byte{
		"buf.yaml":                            []byte(fmt.Sprintf("version: v1\nname: %s\n", targetModuleFullName.String())),
		fmt.Sprintf("foo_%d.proto", *counter): []byte(fmt.Sprintf(`syntax="proto3"; package buf_%d;`, *counter)),
	})
	require.NoError(t, err)
	fileSet, err := bufcas.NewFileSetForBucket(context.Background(), bucket)
	require.NoError(t, err)
	protoManifestBlob, protoBlobs, err := bufcas.FileSetToProtoManifestBlobAndBlobs(fileSet)
	require.NoError(t, err)
	handler.ManuallyPushModule(
		context.Background(),
		t,
		targetModuleFullName,
		branch,
		bufcasalpha.BlobToAlpha(protoManifestBlob),
		bufcasalpha.BlobsToAlpha(protoBlobs),
	)
}

func doEmptyCommits(t *testing.T, repo gittest.Repository, numOfCommits int, counter *int) {
	for i := 0; i < numOfCommits; i++ {
		*counter++
		randomContent, err := uuidutil.New()
		require.NoError(t, err)
		repo.Commit(t, fmt.Sprintf("commit-%d", *counter), map[string]string{
			"randomfile.txt": randomContent.String(),
		})
		repo.Tag(t, fmt.Sprintf("tag-%d", *counter), "")
	}
}

func assertPlanForModuleBranch(
	t *testing.T,
	plan bufsync.ExecutionPlan,
	identity bufmodule.ModuleFullName,
	branch string,
	expectedMessagesOfCommitsToSync ...string,
) {
	t.Helper()
	var found = false
	for _, moduleBranch := range plan.ModuleBranchesToSync() {
		if moduleBranch.BranchName() != branch {
			continue
		}
		if moduleBranch.TargetModuleFullName().IdentityString() != identity.IdentityString() {
			continue
		}
		found = true
		var actualMessagesOfCommitsToSync []string
		for _, commitToSync := range moduleBranch.CommitsToSync() {
			actualMessagesOfCommitsToSync = append(actualMessagesOfCommitsToSync, commitToSync.Commit().Message())
		}
		assert.Equal(t, expectedMessagesOfCommitsToSync, actualMessagesOfCommitsToSync)
	}
	assert.True(t, found, "no plan for module branch")
}

func assertPlanForModuleTags(
	t *testing.T,
	plan bufsync.ExecutionPlan,
	identity bufmodule.ModuleFullName,
	expectedMessagesOfTaggedCommitsToSync ...string,
) {
	t.Helper()
	var found = false
	for _, moduleBranch := range plan.ModuleTagsToSync() {
		if moduleBranch.TargetModuleFullName().IdentityString() != identity.IdentityString() {
			continue
		}
		found = true
		var actualMessagesOfTaggedCommitsToSync []string
		for _, commitToSync := range moduleBranch.TaggedCommitsToSync() {
			actualMessagesOfTaggedCommitsToSync = append(actualMessagesOfTaggedCommitsToSync, commitToSync.Commit().Message())
		}
		assert.ElementsMatch(t, expectedMessagesOfTaggedCommitsToSync, actualMessagesOfTaggedCommitsToSync)
	}
	assert.True(t, found, "no plan for module tags")
}

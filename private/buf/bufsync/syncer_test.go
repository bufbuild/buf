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
	"testing"

	"github.com/bufbuild/buf/private/buf/bufsync"
	"github.com/bufbuild/buf/private/buf/bufsync/bufsynctest"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/git/gittest"
	"golang.org/x/exp/slices"
)

func TestSyncer(t *testing.T) {
	t.Parallel()
	bufsynctest.RunTestSuite(t, func() bufsynctest.TestHandler {
		return newTestSyncHandler()
	})
}

type testRepo struct {
	syncedGitHashes map[string]struct{}
	branches        map[string]*testBranch
	tagsByName      map[string]git.Hash
	tagsForHash     map[string][]string
}

type testBranch struct {
	manualSyncPoint git.Hash
	commits         []*testCommit
}

type testCommit struct {
	hash       git.Hash
	fromDigest string
	fromSync   bufsync.ModuleCommit
}

type testSyncHandler struct {
	repos map[string]*testRepo
}

func newTestSyncHandler() *testSyncHandler {
	return &testSyncHandler{
		repos: make(map[string]*testRepo),
	}
}

func (c *testSyncHandler) getRepo(identity bufmoduleref.ModuleIdentity) *testRepo {
	fullName := identity.IdentityString()
	if _, ok := c.repos[fullName]; !ok {
		c.repos[fullName] = &testRepo{
			syncedGitHashes: make(map[string]struct{}),
			branches:        make(map[string]*testBranch),
			tagsByName:      make(map[string]git.Hash),
			tagsForHash:     make(map[string][]string),
		}
	}
	return c.repos[fullName]
}

func (c *testSyncHandler) getRepoBranch(moduleIdentity bufmoduleref.ModuleIdentity, branchName string) (*testRepo, *testBranch) {
	repo := c.getRepo(moduleIdentity)
	if _, ok := repo.branches[branchName]; !ok {
		repo.branches[branchName] = &testBranch{}
	}
	return repo, repo.branches[branchName]
}

func (c *testSyncHandler) SetSyncPoint(ctx context.Context, t *testing.T, moduleIdentity bufmoduleref.ModuleIdentity, branchName string, hash git.Hash) {
	repo, branch := c.getRepoBranch(moduleIdentity, branchName)
	repo.syncedGitHashes[hash.Hex()] = struct{}{}
	branch.manualSyncPoint = hash
}

func (c *testSyncHandler) putTags(repo *testRepo, commitHash git.Hash, tags []string) {
	for _, tag := range tags {
		if previousHash, ok := repo.tagsByName[tag]; ok {
			// clear previous tag
			repo.tagsForHash[previousHash.Hex()] = slices.DeleteFunc(
				repo.tagsForHash[previousHash.Hex()],
				func(previousTag string) bool {
					return previousTag == tag
				},
			)
		}
		repo.tagsByName[tag] = commitHash
	}
	repo.tagsForHash[commitHash.Hex()] = tags
}

func (c *testSyncHandler) SyncModuleTags(
	ctx context.Context,
	moduleTags bufsync.ModuleTags,
) error {
	for _, commit := range moduleTags.TaggedCommitsToSync() {
		repo := c.getRepo(moduleTags.TargetModuleIdentity())
		c.putTags(repo, commit.Commit().Hash(), commit.Tags())
	}
	return nil
}

func (c *testSyncHandler) ResolveSyncPoint(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branchName string,
) (git.Hash, error) {
	_, branch := c.getRepoBranch(moduleIdentity, branchName)
	// if we have commits from SyncModuleCommit, prefer that over
	// manually set sync point
	if len(branch.commits) > 0 {
		for i := len(branch.commits) - 1; i >= 0; i-- {
			commit := branch.commits[i]
			if commit.fromSync != nil {
				// the latest synced commit
				return commit.hash, nil
			}
		}
		return nil, nil
	}
	if branch.manualSyncPoint != nil {
		return branch.manualSyncPoint, nil
	}
	return nil, nil
}

func (c *testSyncHandler) SyncModuleBranch(
	ctx context.Context,
	moduleBranch bufsync.ModuleBranch,
) error {
	repo, branch := c.getRepoBranch(moduleBranch.TargetModuleIdentity(), moduleBranch.BranchName())
	branch.manualSyncPoint = nil // clear manual sync point
	for _, commit := range moduleBranch.CommitsToSync() {
		repo.syncedGitHashes[commit.Commit().Hash().Hex()] = struct{}{}
		branch.commits = append(branch.commits, &testCommit{
			hash:     commit.Commit().Hash(),
			fromSync: commit,
		})
		c.putTags(repo, commit.Commit().Hash(), commit.Tags())
	}
	return nil
}

func (c *testSyncHandler) IsGitCommitSynced(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	hash git.Hash,
) (bool, error) {
	repo := c.getRepo(moduleIdentity)
	_, isSynced := repo.syncedGitHashes[hash.Hex()]
	return isSynced, nil
}

func (c *testSyncHandler) IsReleaseBranch(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branchName string,
) (bool, error) {
	return branchName == bufmoduleref.Main, nil
}

func (c *testSyncHandler) IsProtectedBranch(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branchName string,
) (bool, error) {
	return branchName == gittest.DefaultBranch || branchName == bufmoduleref.Main, nil
}

func (c *testSyncHandler) GetBranchHead(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branchName string,
) (*registryv1alpha1.RepositoryCommit, error) {
	_, branch := c.getRepoBranch(moduleIdentity, branchName)
	for i := len(branch.commits) - 1; i >= 0; i-- {
		commit := branch.commits[i]
		if commit.fromDigest != "" {
			// the latest repository commit commit
			return &registryv1alpha1.RepositoryCommit{
				// The only thing that matters here is the digest.
				// We give it a useless name.
				Name:           "manual",
				ManifestDigest: commit.fromDigest,
			}, nil
		}
		if commit.fromSync != nil {
			// we want to "fake" a repository commit here
			return &registryv1alpha1.RepositoryCommit{
				// We manually give it a gibberish digest, this will not content match
				// to any module.
				ManifestDigest: "gibberish",
			}, nil
		}
	}
	return nil, nil
}

func (c *testSyncHandler) GetReleaseHead(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
) (*registryv1alpha1.RepositoryCommit, error) {
	return nil, nil
}

func (c *testSyncHandler) IsBranchSynced(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branchName string,
) (bool, error) {
	_, branch := c.getRepoBranch(moduleIdentity, branchName)
	for i := len(branch.commits) - 1; i >= 0; i-- {
		commit := branch.commits[i]
		if commit.fromSync != nil {
			return true, nil
		}
	}
	return false, nil
}

func (c *testSyncHandler) IsGitCommitSyncedToBranch(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branchName string,
	hash git.Hash,
) (bool, error) {
	_, branch := c.getRepoBranch(moduleIdentity, branchName)
	for i := len(branch.commits) - 1; i >= 0; i-- {
		commit := branch.commits[i]
		if commit.fromSync != nil && commit.fromSync.Commit().Hash().String() == hash.String() {
			return true, nil
		}
	}
	return false, nil
}

var _ bufsync.Handler = (*testSyncHandler)(nil)

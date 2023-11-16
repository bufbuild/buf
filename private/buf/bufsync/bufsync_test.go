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

	"github.com/bufbuild/buf/private/buf/bufsync"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/git/gittest"
	"golang.org/x/exp/slices"
)

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
	hash                 git.Hash
	fromRepositoryCommit *registryv1alpha1.RepositoryCommit
	fromSync             bufsync.ModuleCommit
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

func (c *testSyncHandler) getRepoBranch(moduleIdentity bufmoduleref.ModuleIdentity, branchName string) *testBranch {
	repo := c.getRepo(moduleIdentity)
	if _, ok := repo.branches[branchName]; !ok {
		repo.branches[branchName] = &testBranch{}
	}
	return repo.branches[branchName]
}

func (c *testSyncHandler) setSyncPoint(branchName string, hash git.Hash, moduleIdentity bufmoduleref.ModuleIdentity) {
	repo := c.getRepo(moduleIdentity)
	repo.syncedGitHashes[hash.Hex()] = struct{}{}
	branch := c.getRepoBranch(moduleIdentity, branchName)
	branch.manualSyncPoint = hash
}

func (c *testSyncHandler) SyncModuleTaggedCommits(
	ctx context.Context,
	taggedCommits []bufsync.ModuleCommit,
) error {
	for _, commit := range taggedCommits {
		repo := c.getRepo(commit.ModuleIdentity())
		for _, tag := range commit.Tags() {
			if previousHash, ok := repo.tagsByName[tag]; ok {
				// clear previous tag
				repo.tagsForHash[previousHash.Hex()] = slices.DeleteFunc(
					repo.tagsForHash[previousHash.Hex()],
					func(previousTag string) bool {
						return previousTag == tag
					},
				)
			}
			repo.tagsByName[tag] = commit.Commit().Hash()
		}
		repo.tagsForHash[commit.Commit().Hash().Hex()] = commit.Tags()
	}
	return nil
}

func (c *testSyncHandler) ResolveSyncPoint(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branchName string,
) (git.Hash, error) {
	branch := c.getRepoBranch(moduleIdentity, branchName)
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

func (c *testSyncHandler) SyncModuleBranchCommit(
	ctx context.Context,
	commit bufsync.ModuleBranchCommit,
) error {
	c.setSyncPoint(
		commit.Branch(),
		commit.Commit().Hash(),
		commit.ModuleIdentity(),
	)
	branch := c.getRepoBranch(commit.ModuleIdentity(), commit.Branch())
	// append-only, no backfill; good enough for now!
	branch.commits = append(branch.commits, &testCommit{
		hash:     commit.Commit().Hash(),
		fromSync: commit,
	})
	err := c.SyncModuleTaggedCommits(ctx, []bufsync.ModuleCommit{commit})
	return err
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
	branch := c.getRepoBranch(moduleIdentity, branchName)
	for i := len(branch.commits) - 1; i >= 0; i-- {
		commit := branch.commits[i]
		if commit.fromRepositoryCommit != nil {
			// the latest repository commit commit
			return commit.fromRepositoryCommit, nil
		}
	}
	return nil, nil
}

func (c *testSyncHandler) IsBranchSynced(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branchName string,
) (bool, error) {
	branch := c.getRepoBranch(moduleIdentity, branchName)
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
	branch := c.getRepoBranch(moduleIdentity, branchName)
	for i := len(branch.commits) - 1; i >= 0; i-- {
		commit := branch.commits[i]
		if commit.fromSync != nil {
			return true, nil
		}
	}
	return false, nil
}

var _ bufsync.Handler = (*testSyncHandler)(nil)

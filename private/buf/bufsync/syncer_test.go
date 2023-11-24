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
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufcas/bufcasalpha"
	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/git/gittest"
	"github.com/stretchr/testify/require"
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
	releasedCommits []*testCommit
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

func (c *testSyncHandler) getRepo(identity bufmodule.ModuleFullName) *testRepo {
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

func (c *testSyncHandler) getRepoBranch(moduleFullName bufmodule.ModuleFullName, branchName string) (*testRepo, *testBranch) {
	repo := c.getRepo(moduleFullName)
	if _, ok := repo.branches[branchName]; !ok {
		repo.branches[branchName] = &testBranch{}
	}
	return repo, repo.branches[branchName]
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
		repo := c.getRepo(moduleTags.TargetModuleFullName())
		c.putTags(repo, commit.Commit().Hash(), commit.Tags())
	}
	return nil
}

func (c *testSyncHandler) ResolveSyncPoint(
	ctx context.Context,
	moduleFullName bufmodule.ModuleFullName,
	branchName string,
) (git.Hash, error) {
	_, branch := c.getRepoBranch(moduleFullName, branchName)
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
	repo, branch := c.getRepoBranch(moduleBranch.TargetModuleFullName(), moduleBranch.BranchName())
	branch.manualSyncPoint = nil // clear manual sync point
	for _, commit := range moduleBranch.CommitsToSync() {
		repo.syncedGitHashes[commit.Commit().Hash().Hex()] = struct{}{}
		if moduleBranch.BranchName() == bufsynctest.ReleaseBranchName {
			repo.releasedCommits = append(repo.releasedCommits, &testCommit{
				hash:     commit.Commit().Hash(),
				fromSync: commit,
			})
		}
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
	moduleFullName bufmodule.ModuleFullName,
	hash git.Hash,
) (bool, error) {
	repo := c.getRepo(moduleFullName)
	_, isSynced := repo.syncedGitHashes[hash.Hex()]
	return isSynced, nil
}

func (c *testSyncHandler) IsReleaseBranch(
	ctx context.Context,
	moduleFullName bufmodule.ModuleFullName,
	branchName string,
) (bool, error) {
	return branchName == bufsynctest.ReleaseBranchName, nil
}

func (c *testSyncHandler) IsProtectedBranch(
	ctx context.Context,
	moduleFullName bufmodule.ModuleFullName,
	branchName string,
) (bool, error) {
	return branchName == gittest.DefaultBranch ||
		branchName == bufsynctest.ReleaseBranchName ||
		branchName == bufsynctest.OtherProtectedBranchName, nil
}

func (c *testSyncHandler) GetBranchHead(
	ctx context.Context,
	moduleFullName bufmodule.ModuleFullName,
	branchName string,
) (*registryv1alpha1.RepositoryCommit, error) {
	_, branch := c.getRepoBranch(moduleFullName, branchName)
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
				ManifestDigest: "shake256:ab534c69595477fefb2eeceb5ea6239b5ba5b8308e4b4b8009a3b7a4a53dd8272899e3d59704ebc41c51f1dcd3d931f56ee2abe2e53a00839ea19decb6de06dc",
			}, nil
		}
	}
	return nil, nil
}

func (c *testSyncHandler) GetReleaseHead(
	ctx context.Context,
	moduleFullName bufmodule.ModuleFullName,
) (*registryv1alpha1.RepositoryCommit, error) {
	repo := c.getRepo(moduleFullName)
	for i := len(repo.releasedCommits) - 1; i >= 0; i-- {
		commit := repo.releasedCommits[i]
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

func (c *testSyncHandler) IsBranchSynced(
	ctx context.Context,
	moduleFullName bufmodule.ModuleFullName,
	branchName string,
) (bool, error) {
	_, branch := c.getRepoBranch(moduleFullName, branchName)
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
	moduleFullName bufmodule.ModuleFullName,
	branchName string,
	hash git.Hash,
) (bool, error) {
	_, branch := c.getRepoBranch(moduleFullName, branchName)
	for i := len(branch.commits) - 1; i >= 0; i-- {
		commit := branch.commits[i]
		if commit.fromSync != nil && commit.fromSync.Commit().Hash().String() == hash.String() {
			return true, nil
		}
	}
	return false, nil
}

func (c *testSyncHandler) ManuallyPushModule(
	ctx context.Context,
	t *testing.T,
	targetModuleFullName bufmodule.ModuleFullName,
	branchName string,
	manifest *modulev1alpha1.Blob,
	blobs []*modulev1alpha1.Blob,
) {
	digest, err := bufcas.ProtoToDigest(bufcasalpha.AlphaToDigest(manifest.Digest))
	require.NoError(t, err)
	if branchName == "" {
		// release commit
		repo := c.getRepo(targetModuleFullName)
		repo.releasedCommits = append(repo.releasedCommits, &testCommit{
			fromDigest: digest.String(),
		})
	} else {
		_, branch := c.getRepoBranch(targetModuleFullName, branchName)
		branch.commits = append(branch.commits, &testCommit{
			fromDigest: digest.String(),
		})
	}
}

var _ bufsync.Handler = (*testSyncHandler)(nil)

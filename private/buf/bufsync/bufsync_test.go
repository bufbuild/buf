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
	"errors"
	"time"

	"github.com/bufbuild/buf/private/buf/bufsync"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/git/gittest"
	"golang.org/x/exp/slices"
)

type mockClock struct {
	now time.Time
}

func (c *mockClock) Now() time.Time { return c.now }

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
	hash     git.Hash
	fromSync bufsync.ModuleCommit
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

func (c *testSyncHandler) getRepoBranch(identity bufmoduleref.ModuleIdentity, branch string) *testBranch {
	repo := c.getRepo(identity)
	if _, ok := repo.branches[branch]; !ok {
		repo.branches[branch] = &testBranch{}
	}
	return repo.branches[branch]
}

func (c *testSyncHandler) setSyncPoint(branchName string, hash git.Hash, identity bufmoduleref.ModuleIdentity) {
	repo := c.getRepo(identity)
	repo.syncedGitHashes[hash.Hex()] = struct{}{}
	branch := c.getRepoBranch(identity, branchName)
	branch.manualSyncPoint = hash
}

func (c *testSyncHandler) InvalidBSRSyncPoint(
	identity bufmoduleref.ModuleIdentity,
	branch string,
	gitHash git.Hash,
	isDefaultBranch bool,
	err error,
) error {
	return errors.New("unimplemented")
}

func (c *testSyncHandler) BackfillTags(
	ctx context.Context,
	module bufmoduleref.ModuleIdentity,
	alreadySyncedHash git.Hash,
	author git.Ident,
	committer git.Ident,
	tags []string,
) (string, error) {
	repo := c.getRepo(module)
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
		repo.tagsByName[tag] = alreadySyncedHash
	}
	repo.tagsForHash[alreadySyncedHash.Hex()] = tags
	return "some-BSR-commit-name", nil
}

func (c *testSyncHandler) ResolveSyncPoint(
	ctx context.Context,
	module bufmoduleref.ModuleIdentity,
	branchName string,
) (git.Hash, error) {
	branch := c.getRepoBranch(module, branchName)
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

func (c *testSyncHandler) SyncModuleCommit(
	ctx context.Context,
	commit bufsync.ModuleCommit,
) error {
	c.setSyncPoint(
		commit.Branch(),
		commit.Commit().Hash(),
		commit.Identity(),
	)
	branch := c.getRepoBranch(commit.Identity(), commit.Branch())
	// append-only, no backfill; good enough for now!
	branch.commits = append(branch.commits, &testCommit{
		hash:     commit.Commit().Hash(),
		fromSync: commit,
	})
	_, err := c.BackfillTags(
		ctx,
		commit.Identity(),
		commit.Commit().Hash(),
		commit.Commit().Author(),
		commit.Commit().Committer(),
		commit.Tags(),
	)
	return err
}

func (c *testSyncHandler) IsGitCommitSynced(
	ctx context.Context,
	module bufmoduleref.ModuleIdentity,
	hash git.Hash,
) (bool, error) {
	repo := c.getRepo(module)
	_, isSynced := repo.syncedGitHashes[hash.Hex()]
	return isSynced, nil
}

func (c *testSyncHandler) IsProtectedBranch(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branch string,
) (bool, error) {
	return branch == gittest.DefaultBranch || branch == bufmoduleref.Main, nil
}

var _ bufsync.Handler = (*testSyncHandler)(nil)

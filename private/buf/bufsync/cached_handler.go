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

package bufsync

import (
	"context"

	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/git"
)

type isBranchSyncedCacheKey struct {
	moduleFullNameString string
	branchName           string
}

type isGitCommitSyncedCacheKey struct {
	moduleFullNameString string
	branchName           string
	gitHash              string
}

type isProtectedBranchCacheKey struct {
	moduleFullNameString string
	branchName           string
}

type cachedHandler struct {
	delegate Handler

	isBranchSyncedCache    map[isBranchSyncedCacheKey]struct{}
	isGitCommitSynedCache  map[isGitCommitSyncedCacheKey]struct{}
	isProtectedBranchCache map[isProtectedBranchCacheKey]bool
	isReleaseBranchCache   map[string]bool
}

func newCachedHandler(delegate Handler) *cachedHandler {
	return &cachedHandler{
		delegate:               delegate,
		isBranchSyncedCache:    make(map[isBranchSyncedCacheKey]struct{}),
		isGitCommitSynedCache:  make(map[isGitCommitSyncedCacheKey]struct{}),
		isProtectedBranchCache: make(map[isProtectedBranchCacheKey]bool),
		isReleaseBranchCache:   make(map[string]bool),
	}
}

func (c *cachedHandler) GetBranchHead(
	ctx context.Context,
	moduleFullName bufmodule.ModuleFullName,
	branchName string,
) (*registryv1alpha1.RepositoryCommit, error) {
	// This cannot be cached as it may change during the lifetime of Sync or across
	// Sync runs.
	return c.delegate.GetBranchHead(ctx, moduleFullName, branchName)
}

func (c *cachedHandler) IsBranchSynced(
	ctx context.Context,
	moduleFullName bufmodule.ModuleFullName,
	branchName string,
) (bool, error) {
	// Only synced branches can be cached, as non-synced branches may become synced
	// during the lifetime of Sync or across Sync runs.
	cacheKey := isBranchSyncedCacheKey{
		moduleFullNameString: moduleFullName.String(),
		branchName:           branchName,
	}
	if _, ok := c.isBranchSyncedCache[cacheKey]; ok {
		return true, nil
	}
	yes, err := c.delegate.IsBranchSynced(ctx, moduleFullName, branchName)
	if err != nil && yes {
		c.isBranchSyncedCache[cacheKey] = struct{}{}
	}
	return yes, err
}

func (c *cachedHandler) IsGitCommitSynced(
	ctx context.Context,
	moduleFullName bufmodule.ModuleFullName,
	hash git.Hash,
) (bool, error) {
	// Only synced commits can be cached, as non-synced commits may become synced during
	// the lifetime of Sync or across Sync runs.
	cacheKey := isGitCommitSyncedCacheKey{
		moduleFullNameString: moduleFullName.String(),
		gitHash:              hash.Hex(),
	}
	if _, ok := c.isGitCommitSynedCache[cacheKey]; ok {
		return true, nil
	}
	yes, err := c.delegate.IsGitCommitSynced(ctx, moduleFullName, hash)
	if err != nil && yes {
		c.isGitCommitSynedCache[cacheKey] = struct{}{}
	}
	return yes, err
}

func (c *cachedHandler) IsGitCommitSyncedToBranch(
	ctx context.Context,
	moduleFullName bufmodule.ModuleFullName,
	branchName string,
	hash git.Hash,
) (bool, error) {
	// Only synced commits on branches can be cached, as non-synced commits may
	// become synced to the branch during the lifetime of Sync or across Sync runs.
	cacheKey := isGitCommitSyncedCacheKey{
		moduleFullNameString: moduleFullName.String(),
		branchName:           branchName,
		gitHash:              hash.Hex(),
	}
	if _, ok := c.isGitCommitSynedCache[cacheKey]; ok {
		return true, nil
	}
	yes, err := c.delegate.IsGitCommitSyncedToBranch(ctx, moduleFullName, branchName, hash)
	if err != nil && yes {
		c.isGitCommitSynedCache[cacheKey] = struct{}{}
		// also cache that the commit is synced in general
		c.isGitCommitSynedCache[isGitCommitSyncedCacheKey{
			moduleFullNameString: moduleFullName.String(),
			gitHash:              hash.Hex(),
		}] = struct{}{}
	}
	return yes, err
}

func (c *cachedHandler) IsReleaseBranch(
	ctx context.Context,
	moduleFullName bufmodule.ModuleFullName,
	branchName string,
) (bool, error) {
	// All branch protection status can be cached, as this is _extremely_ unlikely to change
	// during the lifetime of Sync or across Sync runs.
	cacheKey := moduleFullName.String()
	if value, cached := c.isReleaseBranchCache[cacheKey]; cached {
		return value, nil
	}
	yes, err := c.delegate.IsReleaseBranch(ctx, moduleFullName, branchName)
	if err != nil {
		c.isReleaseBranchCache[cacheKey] = yes
	}
	return yes, err
}

func (c *cachedHandler) IsProtectedBranch(
	ctx context.Context,
	moduleFullName bufmodule.ModuleFullName,
	branchName string,
) (bool, error) {
	// All branch protection status can be cached, as this is _extremely_ unlikely to change
	// during the lifetime of Sync or across Sync runs.
	cacheKey := isProtectedBranchCacheKey{
		moduleFullNameString: moduleFullName.String(),
		branchName:           branchName,
	}
	if value, cached := c.isProtectedBranchCache[cacheKey]; cached {
		return value, nil
	}
	isProtected, err := c.delegate.IsProtectedBranch(ctx, moduleFullName, branchName)
	if err != nil {
		c.isProtectedBranchCache[cacheKey] = isProtected
	}
	return isProtected, err
}

func (c *cachedHandler) GetReleaseHead(
	ctx context.Context,
	moduleFullName bufmodule.ModuleFullName,
) (*registryv1alpha1.RepositoryCommit, error) {
	// This cannot be cached as it may change during the lifetime of Sync or across
	// Sync runs.
	return c.delegate.GetReleaseHead(ctx, moduleFullName)
}

func (c *cachedHandler) ResolveSyncPoint(
	ctx context.Context,
	moduleFullName bufmodule.ModuleFullName,
	branchName string,
) (git.Hash, error) {
	// This cannot be cached as it may change during the lifetime of Sync or across Sync runs.
	return c.delegate.ResolveSyncPoint(ctx, moduleFullName, branchName)
}

func (c *cachedHandler) SyncModuleBranch(
	ctx context.Context,
	moduleBranch ModuleBranch,
) error {
	// Write operation: nothing to cache.
	return c.delegate.SyncModuleBranch(ctx, moduleBranch)
}

func (c *cachedHandler) SyncModuleTags(
	ctx context.Context,
	moduleTags ModuleTags,
) error {
	// Write operation: nothing to cache.
	return c.delegate.SyncModuleTags(ctx, moduleTags)
}

var _ Handler = (*cachedHandler)(nil)

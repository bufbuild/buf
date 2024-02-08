// Copyright 2020-2024 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/git"
)

type isBranchSyncedCacheKey struct {
	moduleIdentityString string
	branchName           string
}

type isGitCommitSyncedCacheKey struct {
	moduleIdentityString string
	branchName           string
	gitHash              string
}

type isProtectedBranchCacheKey struct {
	moduleIdentityString string
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
	moduleIdentity bufmoduleref.ModuleIdentity,
	branchName string,
) (*registryv1alpha1.RepositoryCommit, error) {
	// This cannot be cached as it may change during the lifetime of Sync or across
	// Sync runs.
	return c.delegate.GetBranchHead(ctx, moduleIdentity, branchName)
}

func (c *cachedHandler) IsBranchSynced(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branchName string,
) (bool, error) {
	// Only synced branches can be cached, as non-synced branches may become synced
	// during the lifetime of Sync or across Sync runs.
	cacheKey := isBranchSyncedCacheKey{
		moduleIdentityString: moduleIdentity.IdentityString(),
		branchName:           branchName,
	}
	if _, ok := c.isBranchSyncedCache[cacheKey]; ok {
		return true, nil
	}
	yes, err := c.delegate.IsBranchSynced(ctx, moduleIdentity, branchName)
	if err != nil && yes {
		c.isBranchSyncedCache[cacheKey] = struct{}{}
	}
	return yes, err
}

func (c *cachedHandler) IsGitCommitSynced(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	hash git.Hash,
) (bool, error) {
	// Only synced commits can be cached, as non-synced commits may become synced during
	// the lifetime of Sync or across Sync runs.
	cacheKey := isGitCommitSyncedCacheKey{
		moduleIdentityString: moduleIdentity.IdentityString(),
		gitHash:              hash.Hex(),
	}
	if _, ok := c.isGitCommitSynedCache[cacheKey]; ok {
		return true, nil
	}
	yes, err := c.delegate.IsGitCommitSynced(ctx, moduleIdentity, hash)
	if err != nil && yes {
		c.isGitCommitSynedCache[cacheKey] = struct{}{}
	}
	return yes, err
}

func (c *cachedHandler) IsGitCommitSyncedToBranch(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branchName string,
	hash git.Hash,
) (bool, error) {
	// Only synced commits on branches can be cached, as non-synced commits may
	// become synced to the branch during the lifetime of Sync or across Sync runs.
	cacheKey := isGitCommitSyncedCacheKey{
		moduleIdentityString: moduleIdentity.IdentityString(),
		branchName:           branchName,
		gitHash:              hash.Hex(),
	}
	if _, ok := c.isGitCommitSynedCache[cacheKey]; ok {
		return true, nil
	}
	yes, err := c.delegate.IsGitCommitSyncedToBranch(ctx, moduleIdentity, branchName, hash)
	if err != nil && yes {
		c.isGitCommitSynedCache[cacheKey] = struct{}{}
		// also cache that the commit is synced in general
		c.isGitCommitSynedCache[isGitCommitSyncedCacheKey{
			moduleIdentityString: moduleIdentity.IdentityString(),
			gitHash:              hash.Hex(),
		}] = struct{}{}
	}
	return yes, err
}

func (c *cachedHandler) IsReleaseBranch(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branchName string,
) (bool, error) {
	// All branch protection status can be cached, as this is _extremely_ unlikely to change
	// during the lifetime of Sync or across Sync runs.
	cacheKey := moduleIdentity.IdentityString()
	if value, cached := c.isReleaseBranchCache[cacheKey]; cached {
		return value, nil
	}
	yes, err := c.delegate.IsReleaseBranch(ctx, moduleIdentity, branchName)
	if err != nil {
		c.isReleaseBranchCache[cacheKey] = yes
	}
	return yes, err
}

func (c *cachedHandler) IsProtectedBranch(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branchName string,
) (bool, error) {
	// All branch protection status can be cached, as this is _extremely_ unlikely to change
	// during the lifetime of Sync or across Sync runs.
	cacheKey := isProtectedBranchCacheKey{
		moduleIdentityString: moduleIdentity.IdentityString(),
		branchName:           branchName,
	}
	if value, cached := c.isProtectedBranchCache[cacheKey]; cached {
		return value, nil
	}
	isProtected, err := c.delegate.IsProtectedBranch(ctx, moduleIdentity, branchName)
	if err != nil {
		c.isProtectedBranchCache[cacheKey] = isProtected
	}
	return isProtected, err
}

func (c *cachedHandler) GetReleaseHead(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
) (*registryv1alpha1.RepositoryCommit, error) {
	// This cannot be cached as it may change during the lifetime of Sync or across
	// Sync runs.
	return c.delegate.GetReleaseHead(ctx, moduleIdentity)
}

func (c *cachedHandler) ResolveSyncPoint(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branchName string,
) (git.Hash, error) {
	// This cannot be cached as it may change during the lifetime of Sync or across Sync runs.
	return c.delegate.ResolveSyncPoint(ctx, moduleIdentity, branchName)
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

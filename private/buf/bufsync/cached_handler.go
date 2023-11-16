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

	isBranchSyncedCache    map[isBranchSyncedCacheKey]bool
	isGitCommitSynedCache  map[isGitCommitSyncedCacheKey]bool
	isProtectedBranchCache map[isProtectedBranchCacheKey]bool
}

func newCachedHandler(delegate Handler) *cachedHandler {
	return &cachedHandler{
		delegate:               delegate,
		isBranchSyncedCache:    make(map[isBranchSyncedCacheKey]bool),
		isGitCommitSynedCache:  make(map[isGitCommitSyncedCacheKey]bool),
		isProtectedBranchCache: make(map[isProtectedBranchCacheKey]bool),
	}
}

func (c *cachedHandler) GetBranchHead(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branchName string,
) (*registryv1alpha1.RepositoryCommit, error) {
	// This cannot be cached as it may change during the lifetime of Sync
	// or across Sync runs.
	return c.delegate.GetBranchHead(ctx, moduleIdentity, branchName)
}

func (c *cachedHandler) IsBranchSynced(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branchName string,
) (bool, error) {
	// Only synced branches can be cached, as non-synced branches may
	// become synced during the lifetime of Sync or across Sync runs.
	cacheKey := isBranchSyncedCacheKey{
		moduleIdentityString: moduleIdentity.IdentityString(),
		branchName:           branchName,
	}
	if c.isBranchSyncedCache[cacheKey] {
		return true, nil
	}
	yes, err := c.delegate.IsBranchSynced(ctx, moduleIdentity, branchName)
	if err != nil && yes {
		c.isBranchSyncedCache[cacheKey] = yes
	}
	return yes, err
}

func (c *cachedHandler) IsGitCommitSynced(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	hash git.Hash,
) (bool, error) {
	// Only synced commits can be cached, as non-synced commits may
	// become synced during the lifetime of Sync or across Sync runs.
	// Note: we want this to overlap with IsGitCommitSyncedToBranch, hence the foreach
	for cached := range c.isGitCommitSynedCache {
		if cached.moduleIdentityString == moduleIdentity.IdentityString() && cached.gitHash == hash.Hex() {
			return true, nil
		}
	}
	yes, err := c.delegate.IsGitCommitSynced(ctx, moduleIdentity, hash)
	if err != nil && yes {
		cacheKey := isGitCommitSyncedCacheKey{
			moduleIdentityString: moduleIdentity.IdentityString(),
			gitHash:              hash.Hex(),
		}
		c.isGitCommitSynedCache[cacheKey] = yes
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
	if c.isGitCommitSynedCache[cacheKey] {
		return true, nil
	}
	yes, err := c.delegate.IsGitCommitSyncedToBranch(ctx, moduleIdentity, branchName, hash)
	if err != nil && yes {
		cacheKey := isGitCommitSyncedCacheKey{
			moduleIdentityString: moduleIdentity.IdentityString(),
			branchName:           branchName,
			gitHash:              hash.Hex(),
		}
		c.isGitCommitSynedCache[cacheKey] = yes
	}
	return yes, err
}

func (c *cachedHandler) IsProtectedBranch(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branchName string,
) (bool, error) {
	// All branch protection status can be synced, as this does not change
	// during the lifetime of Sync or across Sync runs.
	cacheKey := isProtectedBranchCacheKey{
		moduleIdentityString: moduleIdentity.IdentityString(),
		branchName:           branchName,
	}
	if value, cached := c.isProtectedBranchCache[cacheKey]; cached {
		return value, nil
	}
	yes, err := c.delegate.IsProtectedBranch(ctx, moduleIdentity, branchName)
	if err != nil {
		c.isProtectedBranchCache[cacheKey] = yes
	}
	return yes, err
}

func (c *cachedHandler) ResolveSyncPoint(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branchName string,
) (git.Hash, error) {
	// This cannot be cached as it may change during the lifetime of Sync
	// or across Sync runs.
	return c.delegate.ResolveSyncPoint(ctx, moduleIdentity, branchName)
}

func (c *cachedHandler) SyncModuleBranchCommit(
	ctx context.Context,
	moduleCommit ModuleBranchCommit,
) error {
	// Write operation: nothing to cache.
	return c.delegate.SyncModuleBranchCommit(ctx, moduleCommit)
}

func (c *cachedHandler) SyncModuleTaggedCommits(
	ctx context.Context,
	taggedCommits []ModuleCommit,
) error {
	// Write operation: nothing to cache.
	return c.delegate.SyncModuleTaggedCommits(ctx, taggedCommits)
}

var _ Handler = (*cachedHandler)(nil)

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
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
	"go.uber.org/zap"
)

// Handler is a handler for Syncer. It provides any information the Syncer needs to Sync commits,
// and receives ModuleCommits and ModuleBranchCommits that should be synced.
//
// Handler implementations should be safe to use across multiple Syncer#Sync invocations.
type Handler interface {
	// SyncModuleBranch is invoked to sync a set of commits on a branch. If an error is returned, sync
	// will abort.
	//
	// Syncer guarantees that for all commits, either the commit's parent is synced, or none of the
	// commit's ancestors are synced. A commit may be synced _more than once_, in the case where some
	// metadata about the commit has changed (e.g., branch).
	SyncModuleBranch(
		ctx context.Context,
		moduleBranch ModuleBranch,
	) error

	// SyncModuleTags is invoked to sync a set of tagged commits. If an error is returned, sync will abort.
	//
	// Syncer guarantees that this is the complete set of tags for a module identity, and that commits in
	// this set are synced.
	SyncModuleTags(
		ctx context.Context,
		moduleTags ModuleTags,
	) error

	// ResolveSyncPoint is invoked to resolve a syncpoint for a particular module at a particular branch.
	// If no syncpoint is found, this function returns nil. If an error is returned, sync will abort.
	ResolveSyncPoint(
		ctx context.Context,
		moduleIdentity bufmoduleref.ModuleIdentity,
		branchName string,
	) (git.Hash, error)

	// GetBranchHead is invoked by Syncer to resolve the latest commit on a branch. If an error is returned,
	// sync will abort.
	//
	// If a branch does not exist or is empty, implementations must return (nil, nil).
	GetBranchHead(
		ctx context.Context,
		moduleIdentity bufmoduleref.ModuleIdentity,
		branchName string,
	) (*registryv1alpha1.RepositoryCommit, error)

	// GetBranchHead is invoked by Syncer to resolve the latest released commit for the module. If an error is
	// returned, sync will abort.
	//
	// If a branch does not exist or is empty, implementations must return (nil, nil).
	GetReleaseHead(
		ctx context.Context,
		moduleIdentity bufmoduleref.ModuleIdentity,
	) (*registryv1alpha1.RepositoryCommit, error)

	// IsBranchSynced is invoked by Syncer to determine if a particular branch for a module is synced. If
	// an error is returned, sync will abort.
	IsBranchSynced(
		ctx context.Context,
		moduleIdentity bufmoduleref.ModuleIdentity,
		branchName string,
	) (bool, error)

	// IsGitCommitSynced is invoked when syncing branches to know if a Git commit is already synced.
	// If an error is returned, sync will abort.
	IsGitCommitSynced(
		ctx context.Context,
		moduleIdentity bufmoduleref.ModuleIdentity,
		hash git.Hash,
	) (bool, error)

	// IsGitCommitSyncedToBranch is invoked when syncing branches to know if a Git commit is already synced
	// to a particular branch. If an error is returned, sync will abort.
	IsGitCommitSyncedToBranch(
		ctx context.Context,
		moduleIdentity bufmoduleref.ModuleIdentity,
		branchName string,
		hash git.Hash,
	) (bool, error)

	// IsReleaseBranch is invoked when syncing branches to know if a branch's history is the release
	// and must not diverge since the last sync. If an error is returned, sync will abort.
	IsReleaseBranch(
		ctx context.Context,
		moduleIdentity bufmoduleref.ModuleIdentity,
		branchName string,
	) (bool, error)

	// IsProtectedBranch is invoked when syncing branches to know if a branch's history is protected
	// and must not diverge since the last sync. If an error is returned, sync will abort.
	IsProtectedBranch(
		ctx context.Context,
		moduleIdentity bufmoduleref.ModuleIdentity,
		branchName string,
	) (bool, error)
}

// Syncer syncs modules in a git.Repository.
type Syncer interface {
	// Plan generates the ExecutionPlan that Syncer will follow when syncing the Repository.
	//
	// It not necessary to invoke Plan before Sync.
	Plan(context.Context) (ExecutionPlan, error)
	// Sync syncs the repository. It processes commits in reverse topological order, loads any
	// configured named modules, extracts any Git metadata for that commit, and invokes
	// Handler#SyncModuleCommit with a ModuleCommit.
	Sync(context.Context) error
}

// NewSyncer creates a new Syncer.
func NewSyncer(
	logger *zap.Logger,
	repo git.Repository,
	storageGitProvider storagegit.Provider,
	handler Handler,
	options ...SyncerOption,
) (Syncer, error) {
	return newSyncer(
		logger,
		repo,
		storageGitProvider,
		handler,
		options...,
	)
}

// SyncerOption configures the creation of a new Syncer.
type SyncerOption func(*syncer) error

// SyncerWithGitRemote configures a Syncer to sync commits from particular Git remote.
func SyncerWithGitRemote(gitRemoteName string) SyncerOption {
	return func(s *syncer) error {
		s.gitRemoteName = gitRemoteName
		return nil
	}
}

// SyncerWithModule configures a Syncer to sync a module in the specified module directory, with an
// optional module override.
//
// If a not-nil module identity is passed, it will be used as the expected module target for the
// module directory. On the other hand, if a nil module identity is passed, then the module identity
// target for that module directory is read from the HEAD commit on each git branch.
//
// This option can be provided multiple times to sync multiple distinct modules. The order in which
// the module are passed is preserved, and those modules are synced in the same order. If the same
// module directory is passed multiple times this option errors, since the order cannot be preserved
// anymore.
func SyncerWithModule(moduleDir string, identityOverride bufmoduleref.ModuleIdentity) SyncerOption {
	return func(s *syncer) error {
		moduleDir = normalpath.Normalize(moduleDir)
		if _, alreadyAdded := s.modulesDirsToIdentityOverrideForSync[moduleDir]; alreadyAdded {
			return fmt.Errorf("module directory %s already added", moduleDir)
		}
		s.modulesDirsToIdentityOverrideForSync[moduleDir] = identityOverride
		s.sortedModulesDirsForSync = append(s.sortedModulesDirsForSync, moduleDir)
		return nil
	}
}

// SyncerWithAllBranches sets the syncer to sync all branches. Be default the syncer only processes
// commits in the current checked out branch.
func SyncerWithAllBranches() SyncerOption {
	return func(s *syncer) error {
		s.syncAllBranches = true
		return nil
	}
}

// ModuleCommit is a commit with a module that will be synced.
type ModuleCommit interface {
	// Commit is the commit that the module is sourced from.
	Commit() git.Commit
	// Tags are the git tags associated with Commit.
	Tags() []string
	// Bucket is the bucket for the module.
	Bucket(ctx context.Context) (storage.ReadBucket, error)
}

// ModuleBranch is a branch that contains a module at a particular directory,
// along with a set of commits to sync for the branch to the module's module identity.
type ModuleBranch interface {
	// BranchName is the name of git branch that this module is sourced from.
	BranchName() string
	// Directory is the directory relative to the root of the git repository that this module is
	// sourced from.
	Directory() string
	// ModuleIdentity is the identity of the module located in Directory, or an override if one
	// was specified for Directory. This does not necessarily match the identity in each commit
	// in source, but overrides their identity.
	TargetModuleIdentity() bufmoduleref.ModuleIdentity
	// CommitsToSync is the set of commits that will be synced, in the order in which they will
	// be synced.
	CommitsToSync() []ModuleCommit
}

type ModuleTags interface {
	// ModuleIdentity is the identity of the module located in Directory, or an override if one
	// was specified for Directory. This does not necessarily match the identity in each commit
	// in source, but overrides their identity.
	TargetModuleIdentity() bufmoduleref.ModuleIdentity
	TaggedCommitsToSync() []TaggedCommit
}

type TaggedCommit interface {
	// Commit is the git commit that is tagged with Tags.
	Commit() git.Commit
	// Tags are the git tags associated with Commit.
	Tags() []string
}

type ExecutionPlan interface {
	// ModuleBranchesToSync is the set of module branches that Syncer will sync.
	ModuleBranchesToSync() []ModuleBranch
	// TaggedCommitsToSync
	ModuleTagsToSync() []ModuleTags
	// Nop returns true if there is nothing to sync.
	Nop() bool
	// Log logs the plan to the logger
	log(logger *zap.Logger)
}

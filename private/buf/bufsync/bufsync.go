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
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
	"go.uber.org/zap"
)

// ModuleDoesNotExistErr is an error returned when looking for a remote module.
var ModuleDoesNotExistErr = errors.New("BSR module does not exist")

// ErrorHandler handles errors reported by the Syncer. If a non-nil
// error is returned by the handler, sync will abort in a partially-synced
// state.
type ErrorHandler interface {
	// InvalidModuleConfig is invoked by Syncer upon encountering a module
	// with an invalid module config.
	//
	// Returning an error will abort sync.
	InvalidModuleConfig(
		module Module,
		commit git.Commit,
		err error,
	) error
	// BuildFailure is invoked by Syncer upon encountering a module that fails
	// build.
	//
	// Returning an error will abort sync.
	BuildFailure(
		module Module,
		commit git.Commit,
		err error,
	) error
	// InvalidSyncPoint is invoked by Syncer upon encountering a module's branch
	// sync point that is invalid. A typical example is either a sync point that
	// point to a commit that cannot be found anymore, or the commit itself has
	// been corrupted.
	//
	// Returning an error will abort sync.
	InvalidSyncPoint(
		module Module,
		branch string,
		syncPoint git.Hash,
		err error,
	) error
}

// Module is a module that will be synced by Syncer.
type Module interface {
	// Dir is the path to the module relative to the repository root.
	Dir() string
	// RemoteIdentity is the identity of the remote module that the
	// local module is synced to.
	RemoteIdentity() bufmoduleref.ModuleIdentity
	// String is the string representation of this module.
	String() string
}

// NewModule constructs a new module that can be synced with a Syncer.
func NewModule(dir string, identityOverride bufmoduleref.ModuleIdentity) (Module, error) {
	return newSyncableModule(
		dir,
		identityOverride,
	)
}

// Syncer syncs a modules in a git.Repository.
type Syncer interface {
	// Sync syncs the repository using the provided SyncFunc. It processes
	// commits in reverse topological order, loads any configured named
	// modules, extracts any Git metadata for that commit, and invokes
	// SyncFunc with a ModuleCommit.
	//
	// Only commits/branches belonging to the remote named 'origin' are
	// processed. All tags are processed.
	Sync(context.Context, SyncFunc) error
}

// NewSyncer creates a new Syncer.
func NewSyncer(
	logger *zap.Logger,
	repo git.Repository,
	storageGitProvider storagegit.Provider,
	errorHandler ErrorHandler,
	options ...SyncerOption,
) (Syncer, error) {
	return newSyncer(
		logger,
		repo,
		storageGitProvider,
		errorHandler,
		options...,
	)
}

// SyncerOption configures the creation of a new Syncer.
type SyncerOption func(*syncer) error

// SyncerWithModule configures a Syncer to sync the specified module.
//
// This option can be provided multiple times to sync multiple distinct modules.
func SyncerWithModule(module Module) SyncerOption {
	return func(s *syncer) error {
		for _, existingModule := range s.modulesToSync {
			if existingModule.String() == module.String() {
				return fmt.Errorf("duplicate module %s", module)
			}
		}
		s.modulesToSync = append(s.modulesToSync, module)
		return nil
	}
}

// SyncerWithResumption configures a Syncer with a resumption using a SyncPointResolver.
func SyncerWithResumption(resolver SyncPointResolver) SyncerOption {
	return func(s *syncer) error {
		s.syncPointResolver = resolver
		return nil
	}
}

// SyncerWithGitCommitChecker configures a git commit checker, to know if a module has a given git
// hash alrady synced in a BSR instance.
func SyncerWithGitCommitChecker(checker SyncedGitCommitChecker) SyncerOption {
	return func(s *syncer) error {
		s.syncedGitCommitChecker = checker
		return nil
	}
}

// SyncerWithModuleDefaultBranchGetter configures a getter for modules' default branch, to contrast
// a BSR repository default branch vs the local git repository branch. If left empty, the syncer
// skips this validation step.
func SyncerWithModuleDefaultBranchGetter(getter ModuleDefaultBranchGetter) SyncerOption {
	return func(s *syncer) error {
		s.moduleDefaultBranchGetter = getter
		return nil
	}
}

// SyncFunc is invoked by Syncer to process a sync point. If an error is returned,
// sync will abort.
type SyncFunc func(ctx context.Context, commit ModuleCommit) error

// SyncPointResolver is invoked by Syncer to resolve a syncpoint for a particular module
// at a particular branch. If no syncpoint is found, this function returns nil. If an error
// is returned, sync will abort.
type SyncPointResolver func(
	ctx context.Context,
	module bufmoduleref.ModuleIdentity,
	branch string,
) (git.Hash, error)

// SyncedGitCommitChecker is invoked when syncing branches to know which commits hashes from a set
// are already synced inthe BSR. It expects to receive the commit hashes that are synced already. If
// an error is returned, sync will abort.
type SyncedGitCommitChecker func(
	ctx context.Context,
	module bufmoduleref.ModuleIdentity,
	commitHashes map[string]struct{},
) (map[string]struct{}, error)

// ModuleDefaultBranchGetter is invoked before syncing, to make sure all modules that are about to
// be synced have a BSR default branch that matches the local git repo. If the BSR remote module
// does not exist, the implementation should return `ModuleDoesNotExistErr` error.
type ModuleDefaultBranchGetter func(
	ctx context.Context,
	module bufmoduleref.ModuleIdentity,
) (string, error)

// ModuleCommit is a module at a particular commit.
type ModuleCommit interface {
	// Identity is the identity of the module, accounting for any configured override.
	Identity() bufmoduleref.ModuleIdentity
	// Bucket is the bucket for the module.
	Bucket() storage.ReadBucket
	// Commit is the commit that the module is sourced from.
	Commit() git.Commit
	// Branch is the git branch that this module is sourced from.
	Branch() string
	// Tags are the git tags associated with Commit.
	Tags() []string
}

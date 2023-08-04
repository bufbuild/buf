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
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
	"go.uber.org/zap"
)

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

// SyncerWithModuleDirectory configures a Syncer to sync a module in the specified module directory.
//
// This option can be provided multiple times to sync multiple distinct modules. The order in which
// the module directories are passed is preserved, and those modules are synced in the same order.
// If the same module directory is passed multiple times this option errors, since the order cannot
// be preserved anymore.
func SyncerWithModuleDirectory(moduleDir string) SyncerOption {
	return func(s *syncer) error {
		moduleDir = normalpath.Normalize(moduleDir)
		if _, alreadyAdded := s.modulesDirsToSync[moduleDir]; alreadyAdded {
			return fmt.Errorf("module directory %s already added", moduleDir)
		}
		s.modulesDirsToSync[moduleDir] = struct{}{}
		s.sortedModulesDirsToSync = append(s.sortedModulesDirsToSync, moduleDir)
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

// SyncerWithAllBranches sets the syncer to sync all branches. Be default the syncer only processes
// commits in the current checked out branch.
func SyncerWithAllBranches() SyncerOption {
	return func(s *syncer) error {
		s.syncAllBranches = true
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
	// Branch is the git branch that this module is sourced from.
	Branch() string
	// Commit is the commit that the module is sourced from.
	Commit() git.Commit
	// Tags are the git tags associated with Commit.
	Tags() []string
	// Directory is the directory relative to the root of the git repository that this module is
	// sourced from.
	Directory() string
	// Identity is the identity of the module.
	Identity() bufmoduleref.ModuleIdentity
	// Bucket is the bucket for the module.
	Bucket() storage.ReadBucket
}

// ErrModuleDoesNotExist is an error returned when looking for a remote module.
var ErrModuleDoesNotExist = errors.New("BSR module does not exist")

const (
	// ReadModuleErrorCodeUnknown is any unknown or unexpected error while reading a module.
	ReadModuleErrorCodeUnknown = iota
	// ReadModuleErrorCodeModuleNotFound happens when the passed module directory does not have any
	// module.
	ReadModuleErrorCodeModuleNotFound
	// ReadModuleErrorCodeUnnamedModule happens when the read module does not have a name.
	ReadModuleErrorCodeUnnamedModule
	// ReadModuleErrorCodeInvalidModuleConfig happens when the module directory has an invalid module
	// configuration.
	ReadModuleErrorCodeInvalidModuleConfig
	// ReadModuleErrorCodeBuildModule happens when the read module errors building.
	ReadModuleErrorCodeBuildModule
	// ReadModuleErrorCodeUnexpectedName happens when the read module has a different name than
	// expected, usually the one in the branch HEAD commit.
	ReadModuleErrorCodeUnexpectedName
)

// ReadModuleErrorCode is the type of errors that can be thrown by the syncer when reading a module
// from a passed module directory.
type ReadModuleErrorCode int

// ReadModuleError is an error that happens when trying to read a module from a module directory in
// a git commit.
type ReadModuleError struct {
	err       error
	code      ReadModuleErrorCode
	branch    string
	commit    string
	moduleDir string
}

func (e *ReadModuleError) Error() string {
	return fmt.Sprintf(
		"read module in branch %s, commit %s, directory %s: %s",
		e.branch, e.commit, e.moduleDir, e.err.Error(),
	)
}

// ErrorHandler handles errors reported by the Syncer before or during the sync process.
type ErrorHandler interface {
	// StopLookback is invoked when deciding on a git start sync point.
	//
	// For each branch to be synced, the Syncer travels back from HEAD looking for modules in the
	// given module directories, until finding a commit that is already synced to the BSR, or the
	// beginning of the Git repository.
	//
	// The syncer might find errors trying to read a module in that directory. Those errors are sent
	// to this function to decide if the Syncer should stop looking back or not, and choose the
	// previous one (if any) as the start sync point.
	//
	// e.g.: The git commits in topological order are: `a -> ... -> z (HEAD)`, and the modules on a
	// given module directory are:
	//
	// commit | module name or failure | could be synced? | why?
	// ----------------------------------------------------------------------------------------
	// z      | buf.build/acme/foo     | Y                | HEAD
	// y      | buf.build/acme/foo     | Y                | same as HEAD
	// x      | buf.build/acme/bar     | N                | different than HEAD
	// w      | unnamed module         | N                | no module name
	// v      | unbuildable module     | N                | module does not build
	// u      | module not found       | N                | no module name, no 'buf.yaml' file
	// t      | buf.build/acme/foo     | Y                | same as HEAD
	// s      | buf.build/acme/foo     | Y                | same as HEAD
	// r      | buf.build/acme/foo     | N                | already synced to the BSR
	//
	// If this func returns 'true' for any `ReadModuleErrorCode`, then the syncer will stop looking
	// when reaching the commit `r` because it already exists in the BSR, select `s` as the start sync
	// point, and the synced commits into the BSR will be [s, t, x, y, z].
	//
	// On the other hand, if this func returns true for `ReadModuleErrorCodeModuleNotFound`, the
	// syncer will stop looking when reaching the commit `u`, will select `v` as the start sync point,
	// and the synced commits into the BSR will be [x, y, z].
	StopLookback(err *ReadModuleError) bool
	// InvalidRemoteSyncPoint is invoked by Syncer upon encountering a module's branch sync point that
	// is invalid locally. A typical example is either a sync point that points to a commit that
	// cannot be found anymore, or the commit itself has been corrupted.
	//
	// Returning an error will abort sync.
	InvalidRemoteSyncPoint(
		module bufmoduleref.ModuleIdentity,
		branch string,
		syncPoint git.Hash,
		isGitDefaultBranch bool,
		err error,
	) error
}

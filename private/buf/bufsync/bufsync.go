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
	"time"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
	"go.uber.org/zap"
)

// ErrModuleDoesNotExist is an error returned when looking for a BSR module.
var ErrModuleDoesNotExist = errors.New("BSR module does not exist")

const (
	// ReadModuleErrorCodeModuleNotFound happens when the passed module directory does not have any
	// module.
	ReadModuleErrorCodeModuleNotFound = iota + 1
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

// Code returns the error code for this read module error.
func (e *ReadModuleError) Code() ReadModuleErrorCode {
	return e.code
}

// Code returns the module directory in which this error code was thrown.
func (e *ReadModuleError) ModuleDir() string {
	return e.moduleDir
}

func (e *ReadModuleError) Error() string {
	return fmt.Sprintf(
		"read module in branch %s, commit %s, directory %s: %s",
		e.branch, e.commit, e.moduleDir, e.err.Error(),
	)
}

const (
	// LookbackDecisionCodeSkip instructs the syncer to skip the commit that threw the read module
	// error, and keep looking back.
	LookbackDecisionCodeSkip = iota + 1
	// LookbackDecisionCodeOverride instructs the syncer to use the read module and override its
	// identity with the target module identity for that directory, read either from the branch's HEAD
	// commit, or the passed module identity override in the command.
	LookbackDecisionCodeOverride
	// LookbackDecisionCodeStop instructs the syncer to stop looking back when finding the read module
	// error, and use the previous commit (if any) as the start sync point.
	LookbackDecisionCodeStop
	// LookbackDecisionCodeFail instructs the syncer to fail the lookback process for the branch,
	// effectively failing the sync process.
	LookbackDecisionCodeFail
)

// LookbackDecisionCode is the decision made by the ErrorHandler when finding a commit that throws
// an error reading a module.
type LookbackDecisionCode int

// ErrorHandler handles errors reported by the Syncer before or during the sync process.
type ErrorHandler interface {
	// HandleReadModuleError is invoked when navigating a branch from HEAD and seeing an error reading
	// a module.
	//
	// For each branch to be synced, the Syncer travels back from HEAD looking for modules in the
	// given module directories, until finding a commit that is already synced to the BSR, or the
	// beginning of the Git repository.
	//
	// The syncer might find errors trying to read a module in that directory. Those errors are sent
	// to this function to know what to do on those commits.
	//
	// decide if the Syncer should stop looking back or not, and choose the previous one (if any) as
	// the start sync point.
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
	// If this func returns `LookbackDecisionCodeSkip` for any `ReadModuleErrorCode`, then the syncer
	// will stop looking when reaching the commit `r` because it already exists in the BSR, select `s`
	// as the start sync point, and the synced commits into the BSR will be [s, t, x, y, z].
	//
	// If this func returns `LookbackDecisionCodeStop` for `ReadModuleErrorCodeModuleNotFound`, the
	// syncer will stop looking when reaching the commit `u`, will select `v` as the start sync point,
	// and the synced commits into the BSR will be [x, y, z].
	HandleReadModuleError(err *ReadModuleError) LookbackDecisionCode
	// InvalidBSRSyncPoint is invoked by Syncer upon encountering a module's branch sync point that is
	// invalid locally. A typical example is either a sync point that points to a commit that cannot
	// be found anymore, or the commit itself has been corrupted.
	//
	// Returning an error will abort sync.
	InvalidBSRSyncPoint(
		module bufmoduleref.ModuleIdentity,
		branch string,
		syncPoint git.Hash,
		isGitDefaultBranch bool,
		err error,
	) error
}

// Handler is a handler for Syncer. It controls the way in which Syncer handles errors, provides
// any information the Syncer needs to Sync commits, and receives ModuleCommits that should be
// synced.
type Handler interface {
	ErrorHandler

	// SyncModuleCommit is invoked to process a sync point. If an error is returned, sync will abort.
	SyncModuleCommit(ctx context.Context, commit ModuleCommit) error

	// ResolveSyncPoint is invoked to resolve a syncpoint for a particular module at a particular branch.
	// If no syncpoint is found, this function returns nil. If an error is returned, sync will abort.
	ResolveSyncPoint(
		ctx context.Context,
		module bufmoduleref.ModuleIdentity,
		branch string,
	) (git.Hash, error)

	// IsGitCommitSynced is invoked when syncing branches to know if a Git commit is already synced.
	// If an error is returned, sync will abort.
	IsGitCommitSynced(
		ctx context.Context,
		module bufmoduleref.ModuleIdentity,
		hash git.Hash,
	) (bool, error)

	// IsProtectedBranch is invoked when syncing branches to know if a branch's history is protected
	// and must not diverge since the last sync. If an error is returned, sync will abort.
	IsProtectedBranch(
		ctx context.Context,
		moduleIdentity bufmoduleref.ModuleIdentity,
		branch string,
	) (bool, error)

	// BackfillTags is invoked when a commit with valid modules is found within a lookback threshold
	// past the start sync point for such module. The Syncer assumes that the "old" commit is already
	// synced, so it will attempt to backfill existing tags using that git hash, in case they were
	// recently created or moved there.
	//
	// A common scenario is SemVer releases: a commit is pushed to the default Git branch, the sync
	// process triggers and completes, and some minutes later that commit is tagged "v1.2.3". The next
	// time the sync command runs, this backfiller would pick such tag and backfill it to the correct
	// BSR commit.
	//
	// It's expected to return the BSR commit name to which the tags were backfilled.
	BackfillTags(
		ctx context.Context,
		module bufmoduleref.ModuleIdentity,
		alreadySyncedHash git.Hash,
		author git.Ident,
		committer git.Ident,
		tags []string,
	) (string, error)
}

// Syncer syncs modules in a git.Repository.
type Syncer interface {
	// Sync syncs the repository. It processes commits in reverse topological order, loads any
	// configured named modules, extracts any Git metadata for that commit, and invokes
	// Handler#SyncModuleCommit with a ModuleCommit.
	Sync(context.Context) error
}

// NewSyncer creates a new Syncer.
func NewSyncer(
	logger *zap.Logger,
	clock Clock,
	repo git.Repository,
	storageGitProvider storagegit.Provider,
	handler Handler,
	options ...SyncerOption,
) (Syncer, error) {
	return newSyncer(
		logger,
		clock,
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

// Clock provides the current time.
type Clock interface {
	// Now provides the current time.
	Now() time.Time
}

// NewRealClock returns a Clock that returns the current time using time#Now().
func NewRealClock() Clock {
	return newClock()
}

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

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type syncer struct {
	logger                    *zap.Logger
	repo                      git.Repository
	storageGitProvider        storagegit.Provider
	errorHandler              ErrorHandler
	modulesToSync             []Module
	syncPointResolver         SyncPointResolver
	syncedGitCommitChecker    SyncedGitCommitChecker
	moduleDefaultBranchGetter ModuleDefaultBranchGetter
	allBranches               bool

	// scanned information from the repo on sync start
	tagsByCommitHash map[string][]string
	branchesToSync   map[string]struct{}
}

func newSyncer(
	logger *zap.Logger,
	repo git.Repository,
	storageGitProvider storagegit.Provider,
	errorHandler ErrorHandler,
	options ...SyncerOption,
) (Syncer, error) {
	s := &syncer{
		logger:             logger,
		repo:               repo,
		storageGitProvider: storageGitProvider,
	}
	for _, opt := range options {
		if err := opt(s); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// resolveSyncPoints resolves sync points for all known modules for the specified branch,
// returning all modules for which sync points were found, along with their sync points.
//
// If a SyncPointResolver is not configured, this returns an empty map immediately.
func (s *syncer) resolveSyncPoints(ctx context.Context, branch string) (map[Module]git.Hash, error) {
	syncPoints := map[Module]git.Hash{}
	// If resumption is not enabled, we can bail early.
	if s.syncPointResolver == nil {
		return syncPoints, nil
	}
	for _, module := range s.modulesToSync {
		syncPoint, err := s.resolveSyncPoint(ctx, module, branch)
		if err != nil {
			return nil, fmt.Errorf("resolve sync point for module %q in branch %q: %w", module.String(), branch, err)
		}
		if syncPoint != nil {
			s.logger.Debug(
				"resolved sync point, will sync after this commit",
				zap.String("branch", branch),
				zap.Stringer("module", module),
				zap.Stringer("syncPoint", syncPoint),
			)
			syncPoints[module] = syncPoint
		} else {
			s.logger.Debug(
				"no sync point, syncing all branch",
				zap.String("branch", branch),
				zap.Stringer("module", module),
			)
		}
	}
	return syncPoints, nil
}

// resolveSyncPoint resolves a sync point for a particular module and branch. It assumes
// that a SyncPointResolver is configured.
func (s *syncer) resolveSyncPoint(ctx context.Context, module Module, branch string) (git.Hash, error) {
	syncPoint, err := s.syncPointResolver(ctx, module.RemoteIdentity(), branch)
	if err != nil {
		return nil, fmt.Errorf("resolve syncPoint for module %s: %w", module.RemoteIdentity().IdentityString(), err)
	}
	if syncPoint == nil {
		return nil, nil
	}
	// Validate that the commit pointed to by the sync point exists.
	if _, err := s.repo.Objects().Commit(syncPoint); err != nil {
		return nil, s.errorHandler.InvalidSyncPoint(module, branch, syncPoint, err)
	}
	return syncPoint, nil
}

func (s *syncer) Sync(ctx context.Context, syncFunc SyncFunc) error {
	if err := s.scanRepo(); err != nil {
		return fmt.Errorf("scan repo: %w", err)
	}
	if err := s.validateDefaultBranches(ctx); err != nil {
		return err
	}
	branchesSyncPoints := make(map[string]map[Module]git.Hash)
	for branch := range s.branchesToSync {
		syncPoints, err := s.resolveSyncPoints(ctx, branch)
		if err != nil {
			return fmt.Errorf("resolve sync points for branch %q: %w", branch, err)
		}
		branchesSyncPoints[branch] = syncPoints
	}
	// first, default branch, if present
	defaultBranch := s.repo.DefaultBranch()
	if _, shouldSyncDefaultBranch := s.branchesToSync[defaultBranch]; shouldSyncDefaultBranch {
		if err := s.syncBranch(ctx, defaultBranch, branchesSyncPoints[defaultBranch], syncFunc); err != nil {
			return fmt.Errorf("sync default branch %q: %w", defaultBranch, err)
		}
	}
	// then the rest of the branches, in a deterministic order
	sortedBranchesToSync := stringutil.MapToSortedSlice(s.branchesToSync)
	for _, branch := range sortedBranchesToSync {
		if branch == defaultBranch {
			continue // default branch already synced
		}
		if err := s.syncBranch(ctx, branch, branchesSyncPoints[branch], syncFunc); err != nil {
			return fmt.Errorf("sync branch %q: %w", branch, err)
		}
	}
	return nil
}

// validateDefaultBranches checks that all modules to sync, are being synced to BSR repositories
// that have the same default git branch as this repo.
func (s *syncer) validateDefaultBranches(ctx context.Context) error {
	expectedDefaultGitBranch := s.repo.DefaultBranch()
	if s.moduleDefaultBranchGetter == nil {
		s.logger.Warn(
			"default branch validation skipped for all modules",
			zap.String("expected_default_branch", expectedDefaultGitBranch),
		)
		return nil
	}
	var validationErr error
	for _, module := range s.modulesToSync {
		bsrDefaultBranch, err := s.moduleDefaultBranchGetter(ctx, module.RemoteIdentity())
		if err != nil {
			if errors.Is(err, ErrModuleDoesNotExist) {
				s.logger.Warn(
					"default branch validation skipped",
					zap.String("expected_default_branch", expectedDefaultGitBranch),
					zap.String("module", module.RemoteIdentity().IdentityString()),
					zap.Error(err),
				)
				continue
			}
			validationErr = multierr.Append(validationErr, fmt.Errorf("getting bsr module %q default branch: %w", module.RemoteIdentity().IdentityString(), err))
			continue
		}
		if bsrDefaultBranch != expectedDefaultGitBranch {
			validationErr = multierr.Append(
				validationErr,
				fmt.Errorf(
					"remote module %q with default branch %q does not match the git repository's default branch %q, aborting sync",
					module.RemoteIdentity().IdentityString(), bsrDefaultBranch, expectedDefaultGitBranch,
				),
			)
		}
	}
	return validationErr
}

// syncBranch syncs all modules in a branch.
func (s *syncer) syncBranch(
	ctx context.Context,
	branch string,
	modulesSyncPoints map[Module]git.Hash,
	syncFunc SyncFunc,
) error {
	commitsToSync, err := s.commitsToSync(ctx, branch, modulesSyncPoints)
	if err != nil {
		return fmt.Errorf("finding commits to sync: %w", err)
	}
	if len(commitsToSync) == 0 {
		s.logger.Debug(
			"modules already up to date in branch",
			zap.String("branch", branch),
		)
		return nil
	}
	for _, commitToSync := range commitsToSync {
		for _, module := range s.modulesToSync { // looping over the original sort order of modules
			if _, shouldSyncModule := commitToSync.modules[module]; !shouldSyncModule {
				continue
			}
			if err := s.syncModule(ctx, branch, commitToSync.commit, module, syncFunc); err != nil {
				return fmt.Errorf("sync module %q in commit %q: %w", module.String(), commitToSync.commit.Hash().Hex(), err)
			}
		}
	}
	return nil
}

// syncableCommit holds the git commit and modules in that commit that need to be synced.
type syncableCommit struct {
	commit  git.Commit
	modules map[Module]struct{}
}

// commitsToSync returns a sorted commit+modules tuples array that are pending to sync for a branch.
func (s *syncer) commitsToSync(
	ctx context.Context,
	branch string,
	modulesSyncPoints map[Module]git.Hash,
) ([]syncableCommit, error) {
	// First, mark all modules as pending, until its starting sync point is reached. They'll be
	// removed from this list as its initial sync point is found.
	pendingModules := make(map[Module]struct{}, len(s.modulesToSync))
	for _, module := range s.modulesToSync {
		pendingModules[module] = struct{}{}
	}
	var commitsToSync []syncableCommit
	// travel branch commits from HEAD and check if they're already synced, until finding a synced git
	// commit, or adding them all to be synced
	stopLoopErr := errors.New("stop loop")
	if err := s.repo.ForEachCommit(branch, func(commit git.Commit) error {
		if len(pendingModules) == 0 {
			// no more pending modules to sync, no need to keep navigating the branch
			return stopLoopErr
		}
		commitHash := commit.Hash().Hex()
		modulesToSyncInThisCommit := make(map[Module]struct{})
		modulesFoundSyncPointInThisCommit := make(map[Module]struct{})
		for module := range pendingModules {
			// TODO do this in a paginated fashion
			isSynced, err := s.isGitCommitSynced(ctx, module, commitHash)
			if err != nil {
				return fmt.Errorf("check if module %q already synced git commit %q: %w", module.String(), commitHash, err)
			}
			if !isSynced {
				modulesToSyncInThisCommit[module] = struct{}{}
				continue
			}
			// reached a commit that is already synced for this module
			modulesFoundSyncPointInThisCommit[module] = struct{}{}
			expectedSyncPoint, ok := modulesSyncPoints[module]
			if !ok {
				// this module did not have an expected sync point, we probably reached the beginning of the
				// branch off another branch that is already synced.
				continue
			}
			if commitHash != expectedSyncPoint.Hex() {
				if s.repo.DefaultBranch() == branch {
					// TODO: add details to error message saying: "run again with --force-branch-sync <branch
					// name>" when we support a flag like that.
					return fmt.Errorf(
						"found synced git commit %q for default branch %q, but expected sync point was %q, did you rebase or reset your default branch?",
						commitHash,
						branch,
						expectedSyncPoint,
					)
				}
				// syncing non-default branches from an unexpected sync point can be a common scenario in PRs,
				// we can just WARN and continue
				s.logger.Warn(
					"unexpected_sync_point",
					zap.String("expected_sync_point", expectedSyncPoint.Hex()),
					zap.String("found_sync_point", commitHash),
					zap.String("branch", branch),
					zap.String("module", module.String()),
				)
			}
		}
		// clear modules that already found its sync point
		for module := range modulesFoundSyncPointInThisCommit {
			delete(pendingModules, module)
		}
		if len(modulesToSyncInThisCommit) > 0 {
			commitsToSync = append(commitsToSync, syncableCommit{
				commit:  commit,
				modules: modulesToSyncInThisCommit,
			})
		} else {
			// no modules to sync in this commit, we should not have any pending modules
			if len(pendingModules) > 0 {
				return fmt.Errorf(
					"commit %q has no modules to sync, but still has pending modules %v",
					commitHash,
					pendingModules,
				)
			}
		}
		return nil
	}); err != nil && !errors.Is(err, stopLoopErr) {
		return nil, err
	}
	if len(commitsToSync) == 0 {
		return nil, nil
	}
	// https://github.com/golang/go/wiki/SliceTricks#reversing
	for i := len(commitsToSync)/2 - 1; i >= 0; i-- {
		opp := len(commitsToSync) - 1 - i
		commitsToSync[i], commitsToSync[opp] = commitsToSync[opp], commitsToSync[i]
	}
	return commitsToSync, nil
}

func (s *syncer) isGitCommitSynced(ctx context.Context, module Module, commitHash string) (bool, error) {
	if s.syncedGitCommitChecker == nil {
		return false, nil
	}
	syncedCommits, err := s.syncedGitCommitChecker(ctx, module.RemoteIdentity(), map[string]struct{}{commitHash: {}})
	if err != nil {
		return false, err
	}
	_, synced := syncedCommits[commitHash]
	return synced, nil
}

// scanRepo gathers repo information and stores it in the syncer, like tags and branches to sync.
func (s *syncer) scanRepo() error {
	s.tagsByCommitHash = make(map[string][]string)
	if err := s.repo.ForEachTag(func(tag string, commitHash git.Hash) error {
		s.tagsByCommitHash[commitHash.Hex()] = append(s.tagsByCommitHash[commitHash.Hex()], tag)
		return nil
	}); err != nil {
		return fmt.Errorf("load tags: %w", err)
	}
	allRemoteBranches := make(map[string]struct{})
	if err := s.repo.ForEachBranch(func(branch string, _ git.Hash) error {
		allRemoteBranches[branch] = struct{}{}
		return nil
	}); err != nil {
		return fmt.Errorf("looping over repo remote branches: %w", err)
	}
	if s.allBranches {
		s.branchesToSync = allRemoteBranches
		// make sure the default branch is present in the branches to sync
		defaultBranch := s.repo.DefaultBranch()
		if _, isDefaultBranchPushedInRemote := s.branchesToSync[defaultBranch]; !isDefaultBranchPushedInRemote {
			return fmt.Errorf(`repo default branch %q is not present in "origin" remote`, defaultBranch)
		}
	} else {
		// only sync current branch
		currentBranch := s.repo.CurrentBranch()
		if _, isCurrentBranchPushedInRemote := s.branchesToSync[currentBranch]; !isCurrentBranchPushedInRemote {
			return fmt.Errorf(`current branch %q is not present in "origin" remote`, currentBranch)
		}
		s.branchesToSync = map[string]struct{}{currentBranch: {}}
	}
	return nil
}

// syncModule looks for the module in the commit, and if found tries to validate it. If it is valid,
// it invokes `syncFunc`.
//
// It does not return errors on invalid modules, but it will return any errors from `syncFunc` as
// those may be transient.
func (s *syncer) syncModule(
	ctx context.Context,
	branch string,
	commit git.Commit,
	module Module,
	syncFunc SyncFunc,
) error {
	logger := s.logger.With(
		zap.Stringer("commit", commit.Hash()),
		zap.Stringer("module", module),
	)
	sourceBucket, err := s.storageGitProvider.NewReadBucket(
		commit.Tree(),
		storagegit.ReadBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return err
	}
	sourceBucket = storage.MapReadBucket(sourceBucket, storage.MapOnPrefix(module.Dir()))
	foundModule, err := bufconfig.ExistingConfigFilePath(ctx, sourceBucket)
	if err != nil {
		return err
	}
	if foundModule == "" {
		logger.Debug("module not found, skipping commit")
		return nil
	}
	sourceConfig, err := bufconfig.GetConfigForBucket(ctx, sourceBucket)
	if err != nil {
		return s.errorHandler.InvalidModuleConfig(module, commit, err)
	}
	if sourceConfig.ModuleIdentity == nil {
		logger.Debug("unnamed module, skipping commit")
		return nil
	}
	builtModule, err := bufmodulebuild.NewModuleBucketBuilder().BuildForBucket(
		ctx,
		sourceBucket,
		sourceConfig.Build,
	)
	if err != nil {
		return s.errorHandler.BuildFailure(module, commit, err)
	}
	return syncFunc(
		ctx,
		newModuleCommit(
			module.RemoteIdentity(),
			builtModule.Bucket,
			commit,
			branch,
			s.tagsByCommitHash[commit.Hash().Hex()],
		),
	)
}

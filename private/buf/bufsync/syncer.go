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
	"go.uber.org/zap"
)

type syncer struct {
	logger                 *zap.Logger
	repo                   git.Repository
	storageGitProvider     storagegit.Provider
	errorHandler           ErrorHandler
	modulesToSync          []Module
	syncPointResolver      SyncPointResolver
	syncedGitCommitChecker SyncedGitCommitChecker

	// scanned information from the repo on sync start
	tagsByCommitHash map[string][]string
	remoteBranches   map[string]struct{}
	// local cache to know which git hashes are synced and which ones aren't, to avoid requesting the
	// BSR more than once per git hash.
	cacheSyncedGitHashes   map[string]struct{}
	cacheUnsyncedGitHashes map[string]struct{}
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
			return nil, err
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
				"no sync point, syncing from the beginning",
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
	allBranchesSyncPoints := make(map[string]map[Module]git.Hash)
	for branch := range s.remoteBranches {
		syncPoints, err := s.resolveSyncPoints(ctx, branch)
		if err != nil {
			return fmt.Errorf("resolve sync points for branch %q: %w", branch, err)
		}
		allBranchesSyncPoints[branch] = syncPoints
	}
	s.syncBranch(ctx, s.repo.BaseBranch(), allBranchesSyncPoints[s.repo.BaseBranch()])
	// first, default branch
	baseBranch := s.repo.BaseBranch()
	if err := s.repo.ForEachCommit(baseBranch, func(commit git.Commit) error {
		// sync default branch
		return nil
	}); err != nil {
		return fmt.Errorf("sync base branch %q: %w", baseBranch, err)
	}
	// TODO: then the rest of the branches...
	// If we have any sync points left, they were not encountered during sync, which is unexpected
	// behavior.
	for branch, modulesSyncPoints := range allBranchesSyncPoints {
		for module, syncPoint := range modulesSyncPoints {
			if err := s.errorHandler.SyncPointNotEncountered(module, branch, syncPoint); err != nil {
				return err
			}
		}
	}
	return nil
}

// commitsToSync finds the commits that are pending to sync for a branch+module tuple.
func (s *syncer) commitsToSync(
	ctx context.Context,
	branch string,
	module Module,
	expectedSyncPoint string,
) ([]git.Commit, error) {
	const maxCommitsPageLen = 10
	var (
		commitsPageToCheck []git.Commit
		commitsToSync      []git.Commit
	)
	// travel branch commits from HEAD, and check in a paginated fashion if they're already synced,
	// until finding a synced git commit, or adding them all to be synced
	stopLoopErr := errors.New("stop loop")
	checkCommitsPage := func() error {
		syncedCommits, err := s.checkSyncedGitHashes(ctx, commitsPageToCheck)
		if err != nil {
			return fmt.Errorf("check synced git commits: %w", err)
		}
		if len(syncedCommits) == 0 {
			// this page didn't include any git commit already synced, add it to the commits to sync, and
			// start a new page
			commitsToSync = append(commitsToSync, commitsPageToCheck...)
			commitsPageToCheck = []git.Commit{}
			return nil
		}
		// there was at least one git commit that is already synced in this page
		for _, commit := range commitsPageToCheck {
			commitHash := commit.Hash().Hex()
			if _, synced := syncedCommits[commitHash]; !synced {
				commitsToSync = append(commitsToSync, commit)
				continue
			}
			// sync point found
			if expectedSyncPoint != commitHash {
				if s.repo.BaseBranch() == branch {
					// TODO: add details to error message saying: "run again with --force-branch-sync <branch
					// name>" when we support a flag like that.
					return fmt.Errorf(
						"found synced git commit %q for default branch %q, but expected sync point was %q, did you rebase or reset your default branch?",
						commitHash,
						branch,
						expectedSyncPoint,
					)
				}
				s.logger.Warn(
					"unexpected_sync_point",
					zap.String("expected_sync_point", expectedSyncPoint),
					zap.String("found_sync_point", commitHash),
					zap.String("branch", branch),
					zap.String("module", module.String()),
				)
			}
			return stopLoopErr
		}
		return fmt.Errorf("synced commits %v are not present in requested page %v", syncedCommits, commitsPageToCheck)
	}
	var pendingPage bool
	if err := s.repo.ForEachCommit(branch, func(commit git.Commit) error {
		commitsPageToCheck = append(commitsPageToCheck, commit)
		pendingPage = true
		if len(commitsPageToCheck) < maxCommitsPageLen {
			// haven't reached page size, keep appending
			return nil
		}
		pendingPage = false
		return checkCommitsPage()
	}); err != nil && !errors.Is(err, stopLoopErr) {
		return nil, err
	}
	if pendingPage {
		// we reached the end of commits for the branch, with a pending commits page to check
		if err := checkCommitsPage(); err != nil {
			return nil, err
		}
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

// checkSyncedGitHashes checks which of the passed git hashes are synced already, and returns them.
// It first checks the local cache before using the remote checker.
func (s *syncer) checkSyncedGitHashes(
	ctx context.Context,
	commits []git.Commit,
) (map[string]struct{} /* synced */, error) {
	syncedGitHashes := make(map[string]struct{})
	gitHashesToCheck := make(map[string]struct{})
	// check local cache
	for _, commit := range commits {
		commitHash := commit.Hash().Hex()
		if _, cachedSynced := s.cacheSyncedGitHashes[commitHash]; cachedSynced {
			syncedGitHashes[commitHash] = struct{}{} // add it to the return
			continue                                 // already know it's synced, no need to check again
		} else if _, cachedUnsynced := s.cacheUnsyncedGitHashes[commitHash]; cachedUnsynced {
			continue // already know it's not synced, no need to check again
		}
		gitHashesToCheck[commitHash] = struct{}{}
	}
	if len(gitHashesToCheck) == 0 {
		// all of the requested git hashes were cached (synced or unsynced), no need to check remotely
		return syncedGitHashes, nil
	}
	// check remotely
	checkedSyncedGitHashes, err := s.syncedGitCommitChecker(ctx, gitHashesToCheck)
	if err != nil {
		return nil, fmt.Errorf("check synced git commits: %w", err)
	}
	// cache new response
	for gitHash := range gitHashesToCheck {
		if _, synced := checkedSyncedGitHashes[gitHash]; synced {
			s.cacheSyncedGitHashes[gitHash] = struct{}{} // cache as synced
		} else {
			s.cacheUnsyncedGitHashes[gitHash] = struct{}{} // cache as unsynced
		}
	}
	// include checked response in the return
	for syncedGitHash := range checkedSyncedGitHashes {
		syncedGitHashes[syncedGitHash] = struct{}{}
	}
	return syncedGitHashes, nil
}

// scanRepo gathers repo information and stores it in the syncer: gathers all tags, all
// remote/origin branches, and visits each commit on them, starting from the base branch, to assign
// it to a unique branch and sort them by aithor timestamp.
func (s *syncer) scanRepo() error {
	s.tagsByCommitHash = make(map[string][]string)
	if err := s.repo.ForEachTag(func(tag string, commitHash git.Hash) error {
		s.tagsByCommitHash[commitHash.Hex()] = append(s.tagsByCommitHash[commitHash.Hex()], tag)
		return nil
	}); err != nil {
		return fmt.Errorf("load tags: %w", err)
	}
	s.remoteBranches = make(map[string]struct{})
	if err := s.repo.ForEachBranch(func(branch string, _ git.Hash) error {
		s.remoteBranches[branch] = struct{}{}
		return nil
	}); err != nil {
		return fmt.Errorf("looping over repo branches: %w", err)
	}
	baseBranch := s.repo.BaseBranch()
	if _, baseBranchPushedInRemote := s.remoteBranches[baseBranch]; !baseBranchPushedInRemote {
		return fmt.Errorf(`repo base branch %q is not present in "origin" remote`, baseBranch)
	}
	return nil
}

// visitCommit looks for the module in the commit, and if found tries to validate it.
// If it is valid, it invokes `syncFunc`.
//
// It does not return errors on invalid modules, but it will return any errors from
// `syncFunc` as those may be transient.
func (s *syncer) visitCommit(
	ctx context.Context,
	module Module,
	branch string,
	commit git.Commit,
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

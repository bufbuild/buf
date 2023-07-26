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
	// first, default branch
	baseBranch := s.repo.BaseBranch()
	if err := s.syncBranch(ctx, baseBranch, allBranchesSyncPoints[baseBranch]); err != nil {
		return fmt.Errorf("sync base branch %q: %w", baseBranch, err)
	}
	for branch := range s.remoteBranches {
		if branch == baseBranch {
			// already synced
			continue
		}
		if err := s.syncBranch(ctx, branch, allBranchesSyncPoints[branch]); err != nil {
			return fmt.Errorf("sync branch %q: %w", branch, err)
		}
	}
	// If we have any sync points left, they were not encountered during sync, which is unexpected
	// behavior.
	for branch, modulesSyncPoints := range allBranchesSyncPoints {
		for module, syncPoint := range modulesSyncPoints {
			// TODO: check that this is just WARNing
			if err := s.errorHandler.SyncPointNotEncountered(module, branch, syncPoint); err != nil {
				return err
			}
		}
	}
	return nil
}

// syncBranch syncs all modules in a branch.
func (s *syncer) syncBranch(
	ctx context.Context,
	branch string,
	modulesSyncPoints map[Module]git.Hash,
) error {
	for module, expectedSyncPoint := range modulesSyncPoints {
		_, err := s.commitsToSync(ctx, branch, module, expectedSyncPoint.Hex())
		if err != nil {
			return fmt.Errorf("finding commits to sync for branch %q and module %q: %w", branch, module.String(), err)
		}
		// TODO: sync commits
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
	var commitsToSync []git.Commit
	// travel branch commits from HEAD and check if they're already synced, until finding a synced git
	// commit, or adding them all to be synced
	stopLoopErr := errors.New("stop loop")
	if err := s.repo.ForEachCommit(branch, func(commit git.Commit) error {
		commitHash := commit.Hash().Hex()
		isSynced, err := s.isGitCommitSynced(ctx, commitHash)
		if err != nil {
			return fmt.Errorf("check if git commit is synced: %w", err)
		}
		if !isSynced {
			commitsToSync = append(commitsToSync, commit)
			return nil
		}
		if commitHash != expectedSyncPoint {
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
			// syncing non-default branches from an unexpected sync point can be a common scenario in PRs,
			// we can just WARN and continue
			s.logger.Warn(
				"unexpected_sync_point",
				zap.String("expected_sync_point", expectedSyncPoint),
				zap.String("found_sync_point", commitHash),
				zap.String("branch", branch),
				zap.String("module", module.String()),
			)
		}
		return stopLoopErr
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

// TODO: remove, checking that git hashes are synced should happen in a paginated fashion
func (s *syncer) isGitCommitSynced(
	ctx context.Context,
	commitHash string,
) (bool, error) {
	syncedCommits, err := s.checkSyncedGitHashes(ctx, []string{commitHash})
	if err != nil {
		return false, err
	}
	_, synced := syncedCommits[commitHash]
	return synced, nil
}

// checkSyncedGitHashes checks which of the passed git hashes are synced already, and returns them.
// It first checks the local cache before using the remote checker.
func (s *syncer) checkSyncedGitHashes(
	ctx context.Context,
	commitHashes []string,
) (map[string]struct{} /* synced */, error) {
	syncedGitHashes := make(map[string]struct{})
	gitHashesToCheck := make(map[string]struct{})
	// check local cache
	for _, commitHash := range commitHashes {
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

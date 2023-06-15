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
	"fmt"
	"sort"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"go.uber.org/zap"
)

type syncer struct {
	logger             *zap.Logger
	repo               git.Repository
	storageGitProvider storagegit.Provider
	errorHandler       ErrorHandler
	modulesToSync      []Module
	syncPointResolver  SyncPointResolver

	// scanned information from the repo on sync start
	knownTagsByCommitHash map[string][]string
	remoteBranches        map[string]struct{}
	commitHashToBranch    map[string]string
	sortedCommits         []git.Commit
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
	for _, commit := range s.sortedCommits {
		branch, ok := s.commitHashToBranch[commit.Hash().Hex()]
		if !ok {
			return fmt.Errorf("commit %q has no associated branch", commit.Hash().Hex())
		}
		logger := s.logger.With(
			zap.String("branch", branch),
			zap.String("commit", commit.Hash().Hex()),
		)
		for _, module := range s.modulesToSync {
			logger := logger.With(zap.String("module", module.String()))
			var syncPoint git.Hash
			if allBranchesSyncPoints != nil &&
				allBranchesSyncPoints[branch] != nil &&
				allBranchesSyncPoints[branch][module] != nil {
				syncPoint = allBranchesSyncPoints[branch][module]
			}
			if syncPoint != nil {
				logger := logger.With(zap.Stringer("syncPoint", syncPoint))
				// This module has a sync point for this branch. We need to check if we've encountered the
				// sync point.
				if syncPoint.Hex() == commit.Hash().Hex() {
					delete(allBranchesSyncPoints[branch], module)
					logger.Debug("syncPoint encountered, will resume syncing next commit")
				} else {
					logger.Debug("syncPoint not encountered yet, skipping commit")
				}
				continue
			}
			logger.Debug("sync")
			if err := s.visitCommit(ctx, module, branch, commit, syncFunc); err != nil {
				return fmt.Errorf("process commit %s (%s): %w", commit.Hash().Hex(), branch, err)
			}
		}
	}
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

// scanRepo gathers repo information and stores it in the syncer: gathers all tags, all
// remote/origin branches, and visits each commit on them, starting from the base branch, to assign
// it to a unique branch and sort them by aithor timestamp.
func (s *syncer) scanRepo() error {
	s.knownTagsByCommitHash = make(map[string][]string)
	if err := s.repo.ForEachTag(func(tag string, commitHash git.Hash) error {
		s.knownTagsByCommitHash[commitHash.Hex()] = append(s.knownTagsByCommitHash[commitHash.Hex()], tag)
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
	s.commitHashToBranch = make(map[string]string)
	s.sortedCommits = make([]git.Commit, 0)
	loopOverBranchCommits := func(branch string) error {
		if err := s.repo.ForEachCommit(branch, func(commit git.Commit) error {
			if _, alreadyVisited := s.commitHashToBranch[commit.Hash().Hex()]; !alreadyVisited {
				s.commitHashToBranch[commit.Hash().Hex()] = branch
				s.sortedCommits = append(s.sortedCommits, commit)
			}
			return nil
		}); err != nil {
			return fmt.Errorf("looping over commits in branch %q: %w", baseBranch, err)
		}
		return nil
	}
	// first, assign all commits in base branch, then the remaining ones in a deterministic order.
	if err := loopOverBranchCommits(baseBranch); err != nil {
		return err
	}
	sortedBranches := stringutil.MapToSortedSlice(s.remoteBranches)
	for _, branch := range sortedBranches {
		if branch == baseBranch {
			continue // this one was already visited
		}
		if err := loopOverBranchCommits(branch); err != nil {
			return err
		}
	}
	// sort all commits by author timestamp
	sort.Slice(s.sortedCommits, func(i, j int) bool {
		return s.sortedCommits[i].Author().Timestamp().Before(s.sortedCommits[j].Author().Timestamp())
	})
	// TODO remove, this will be extra verbose
	s.logger.Debug(
		"repo scan",
		zap.Any("tags", s.knownTagsByCommitHash),
		zap.Any("branches", s.remoteBranches),
		zap.Any("commit to branch", s.commitHashToBranch),
		zap.Stringers("sorted commits", s.sortedCommits),
	)
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
	builtModule, err := bufmodulebuild.BuildForBucket(
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
			s.knownTagsByCommitHash[commit.Hash().Hex()],
		),
	)
}

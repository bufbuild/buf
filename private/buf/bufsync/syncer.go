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

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
	"go.uber.org/zap"
)

type syncer struct {
	logger             *zap.Logger
	repo               git.Repository
	storageGitProvider storagegit.Provider
	errorHandler       ErrorHandler
	modulesToSync      []Module
	syncPointResolver  SyncPointResolver

	knownTagsByCommitHash map[string][]string
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
	s.knownTagsByCommitHash = map[string][]string{}
	if err := s.repo.ForEachTag(func(tag string, commitHash git.Hash) error {
		s.knownTagsByCommitHash[commitHash.Hex()] = append(s.knownTagsByCommitHash[commitHash.Hex()], tag)
		return nil
	}); err != nil {
		return fmt.Errorf("load tags: %w", err)
	}
	allRepoBranches := make(map[string]struct{})
	if err := s.repo.ForEachBranch(func(branch string, _ git.Hash) error {
		allRepoBranches[branch] = struct{}{}
		return nil
	}); err != nil {
		return fmt.Errorf("looping over repo branches: %w", err)
	}
	s.logger.Debug("all branches", zap.Any("branches", allRepoBranches))
	// TODO: sync other branches
	for _, branch := range []string{s.repo.BaseBranch()} {
		logger := s.logger.With(zap.String("branch", branch))
		syncPoints, err := s.resolveSyncPoints(ctx, branch)
		if err != nil {
			return fmt.Errorf("resolve sync points: %w", err)
		}
		// We sync all modules in a commit before advancing to the next commit so that
		// inter-module dependencies across commits can be resolved.
		if err := s.repo.ForEachCommit(branch, func(commit git.Commit) error {
			logger := logger.With(zap.String("commit", commit.Hash().String()))
			for _, module := range s.modulesToSync {
				logger.Debug("sync", zap.String("module", module.String()))
				if syncPoint := syncPoints[module]; syncPoint != nil {
					// This module has a sync point. We need to check if we've encountered the sync point.
					if syncPoint.Hex() == commit.Hash().Hex() {
						// We have found the syncPoint! We can resume syncing _after_ this point.
						delete(syncPoints, module)
						s.logger.Debug(
							"syncPoint encountered, skipping commit",
							zap.Stringer("commit", commit.Hash()),
							zap.Stringer("module", module),
							zap.Stringer("syncPoint", syncPoint),
						)
					} else {
						// We have not encountered the syncPoint yet. Skip this commit and keep looking
						// for the syncPoint.
						s.logger.Debug(
							"syncPoint not encountered, skipping commit",
							zap.Stringer("commit", commit.Hash()),
							zap.Stringer("module", module),
							zap.Stringer("syncPoint", syncPoint),
						)
					}
					continue
				}
				if err := s.visitCommit(ctx, module, branch, commit, syncFunc); err != nil {
					return fmt.Errorf("process commit %s (%s): %w", commit.Hash().Hex(), branch, err)
				}
			}
			return nil
		}); err != nil {
			return fmt.Errorf("process commits: %w", err)
		}
		// If we have any sync points left, they were not encountered during sync, which is unexpected behavior.
		for module, syncPoint := range syncPoints {
			if err := s.errorHandler.SyncPointNotEncountered(module, branch, syncPoint); err != nil {
				return err
			}
		}
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

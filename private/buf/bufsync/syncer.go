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
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"go.uber.org/zap"
)

type syncer struct {
	logger                    *zap.Logger
	repo                      git.Repository
	storageGitProvider        storagegit.Provider
	errorHandler              ErrorHandler
	syncedGitCommitChecker    SyncedGitCommitChecker
	moduleDefaultBranchGetter ModuleDefaultBranchGetter
	syncPointResolver         SyncPointResolver

	// flags received on creation
	sortedModulesDirsToSync []string
	modulesDirsToSync       map[string]struct{}
	syncAllBranches         bool

	// populated before and during sync
	tagsByCommitHash  map[string][]string
	branchesToSync    map[string]struct{}
	modulesSyncPoints map[bufmoduleref.ModuleIdentity]map[string]git.Hash // moduleIdentity:branch:gitSyncPoint
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
		modulesDirsToSync:  make(map[string]struct{}),
		tagsByCommitHash:   make(map[string][]string),
		branchesToSync:     make(map[string]struct{}),
		modulesSyncPoints:  make(map[bufmoduleref.ModuleIdentity]map[string]git.Hash),
	}
	for _, opt := range options {
		if err := opt(s); err != nil {
			return nil, err
		}
	}
	if s.moduleDefaultBranchGetter == nil {
		s.logger.Warn(
			"default branch validation skipped",
			zap.String("expected_default_branch", s.repo.DefaultBranch()),
		)
	}
	if s.syncedGitCommitChecker == nil {
		s.logger.Warn("no sync git commit checker, branches will attempt to sync from the start")
	}
	return s, nil
}

func (s *syncer) Sync(ctx context.Context, syncFunc SyncFunc) error {
	if err := s.prepareSync(ctx); err != nil {
		return fmt.Errorf("scan repo: %w", err)
	}
	// first, default branch, if present
	defaultBranch := s.repo.DefaultBranch()
	if _, shouldSyncDefaultBranch := s.branchesToSync[defaultBranch]; shouldSyncDefaultBranch {
		if err := s.syncBranch(ctx, defaultBranch, syncFunc); err != nil {
			return fmt.Errorf("sync default branch %q: %w", defaultBranch, err)
		}
	}
	// then the rest of the branches, in a deterministic order
	sortedBranchesToSync := stringutil.MapToSortedSlice(s.branchesToSync)
	for _, branch := range sortedBranchesToSync {
		if branch == defaultBranch {
			continue // default branch already synced
		}
		if err := s.syncBranch(ctx, branch, syncFunc); err != nil {
			return fmt.Errorf("sync branch %q: %w", branch, err)
		}
	}
	return nil
}

// syncBranch syncs all modules in a branch.
func (s *syncer) syncBranch(
	ctx context.Context,
	branch string,
	syncFunc SyncFunc,
) error {
	commitsToSync, err := s.commitsToSync(ctx, branch)
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
		for _, module := range s.modulesDirsToSync { // looping over the original sort order of modules
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
	builtNamedModule, err := s.builtNamedModuleAt(ctx, commit, module.Dir())
	if err != nil {
		if errors.Is(err, errModuleNotFound) {
			logger.Debug("module not found, skipping commit")
			return nil
		}
		if invalidConfigErr := (&invalidModuleConfigError{}); errors.As(err, &invalidConfigErr) {
			return s.errorHandler.InvalidModuleConfig(module, commit, err)
		}
		if errors.Is(err, errUnnamedModule) {
			logger.Debug("unnamed module, skipping commit")
			return nil
		}
		if buildModuleErr := (&buildModuleError{}); errors.As(err, &buildModuleErr) {
			return s.errorHandler.BuildFailure(module, commit, err)
		}
		return err
	}
	return syncFunc(
		ctx,
		newModuleCommitToSync(
			builtNamedModule.ModuleIdentity(), // TODO make sure it's the same name
			builtNamedModule.Bucket,
			commit,
			branch,
			s.tagsByCommitHash[commit.Hash().Hex()],
		),
	)
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
	for _, module := range s.modulesDirsToSync {
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

// validateDefaultBranch checks that the passed module identity has the same default branch as the
// syncer Git repo's default branch.
func (s *syncer) validateDefaultBranch(ctx context.Context, moduleIdentity bufmoduleref.ModuleIdentity) error {
	if s.moduleDefaultBranchGetter == nil {
		return nil
	}
	expectedDefaultGitBranch := s.repo.DefaultBranch()
	bsrDefaultBranch, err := s.moduleDefaultBranchGetter(ctx, moduleIdentity)
	if err != nil {
		if errors.Is(err, ErrModuleDoesNotExist) {
			s.logger.Warn(
				"default branch validation skipped",
				zap.String("expected_default_branch", expectedDefaultGitBranch),
				zap.String("module", moduleIdentity.IdentityString()),
				zap.Error(err),
			)
			return nil
		}
		return fmt.Errorf("getting bsr module %q default branch: %w", moduleIdentity.IdentityString(), err)
	}
	if bsrDefaultBranch != expectedDefaultGitBranch {
		return fmt.Errorf(
			"remote module %q with default branch %q does not match the git repository's default branch %q, aborting sync",
			moduleIdentity.IdentityString(), bsrDefaultBranch, expectedDefaultGitBranch,
		)
	}
	return nil
}

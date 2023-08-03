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

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
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

	commitsTags                   map[string][]string                               // commits:[]tags
	branchesModulesToSync         map[string]map[string]bufmoduleref.ModuleIdentity // branch:moduleDir:moduleIdentityInHEAD
	modulesBranchesLastSyncPoints map[string]map[string]string                      // moduleIdentity:branch:lastSyncPointGitHash

	// syncedModulesCommitsCache (moduleIdentity:commit) caches commits already synced to a given BSR
	// module, so we don't ask twice the same module:commit when we already know it's already synced.
	// We don't cache "unsynced" git commits, since during the sync process we will be syncing new git
	// commits, which will be added to this cache when they are.
	syncedModulesCommitsCache map[string]map[string]struct{}
}

func newSyncer(
	logger *zap.Logger,
	repo git.Repository,
	storageGitProvider storagegit.Provider,
	errorHandler ErrorHandler,
	options ...SyncerOption,
) (Syncer, error) {
	s := &syncer{
		logger:                        logger,
		repo:                          repo,
		storageGitProvider:            storageGitProvider,
		errorHandler:                  errorHandler,
		modulesDirsToSync:             make(map[string]struct{}),
		commitsTags:                   make(map[string][]string),
		branchesModulesToSync:         make(map[string]map[string]bufmoduleref.ModuleIdentity),
		modulesBranchesLastSyncPoints: make(map[string]map[string]string),
		syncedModulesCommitsCache:     make(map[string]map[string]struct{}),
	}
	for _, opt := range options {
		if err := opt(s); err != nil {
			return nil, err
		}
	}
	if s.moduleDefaultBranchGetter == nil {
		s.logger.Warn(
			"no module default branch getter, the default branch validation will be skipped for all modules and branches",
			zap.String("default_git_branch", s.repo.DefaultBranch()),
		)
	}
	if s.syncedGitCommitChecker == nil {
		s.logger.Warn("no sync git commit checker, all branches will attempt to sync from the start")
	}
	return s, nil
}

func (s *syncer) Sync(ctx context.Context, syncFunc SyncFunc) error {
	if err := s.prepareSync(ctx); err != nil {
		return fmt.Errorf("scan repo: %w", err)
	}
	s.printSyncPreparation()
	// first, default branch, if present
	defaultBranch := s.repo.DefaultBranch()
	if _, shouldSyncDefaultBranch := s.branchesModulesToSync[defaultBranch]; shouldSyncDefaultBranch {
		if err := s.syncBranch(ctx, defaultBranch, syncFunc); err != nil {
			return fmt.Errorf("sync default branch %s: %w", defaultBranch, err)
		}
	}
	// then the rest of the branches, in a deterministic order
	var sortedBranchesToSync []string
	for branch := range s.branchesModulesToSync {
		if branch == defaultBranch {
			continue // default branch was already synced
		}
		sortedBranchesToSync = append(sortedBranchesToSync, branch)
	}
	sort.Strings(sortedBranchesToSync)
	for _, branch := range sortedBranchesToSync {
		if err := s.syncBranch(ctx, branch, syncFunc); err != nil {
			return fmt.Errorf("sync branch %s: %w", branch, err)
		}
	}
	return nil
}

// syncBranch modules from a branch.
//
// It first navigates the branch calculating the commits+modules that are to be synced. Once all
// modules have their initial git sync point, then we loop over those commits and invoke the sync
// function.
//
// Any error from the sync func aborts the sync process and leaves it partially complete, presumably
// safe to resume.
func (s *syncer) syncBranch(ctx context.Context, branch string, syncFunc SyncFunc) error {
	commitsToSync, err := s.branchCommitsToSync(ctx, branch)
	if err != nil {
		return fmt.Errorf("finding commits to sync: %w", err)
	}
	if len(commitsToSync) == 0 {
		s.logger.Debug(
			"no modules to sync in branch",
			zap.String("branch", branch),
		)
		return nil
	}
	s.printCommitsToSync(branch, commitsToSync)
	for _, commitToSync := range commitsToSync {
		commitHash := commitToSync.commit.Hash().Hex()
		if len(commitToSync.modules) == 0 {
			s.logger.Debug(
				"branch commit with no modules to sync, skipping commit",
				zap.String("branch", branch),
				zap.String("commit", commitHash),
			)
			continue
		}
		for _, moduleDir := range s.sortedModulesDirsToSync { // looping over the original sort order of modules
			builtModule, shouldSyncModule := commitToSync.modules[moduleDir]
			if !shouldSyncModule {
				s.logger.Debug(
					"module directory not present as a module to sync, skipping module in commit",
					zap.String("branch", branch),
					zap.String("commit", commitHash),
					zap.String("module directory", moduleDir),
				)
				continue
			}
			modIdentity := builtModule.ModuleIdentity().IdentityString()
			if err := syncFunc(
				ctx,
				newModuleCommitToSync(
					branch,
					commitToSync.commit,
					s.commitsTags[commitHash],
					moduleDir,
					builtModule.ModuleIdentity(),
					builtModule.Bucket,
				),
			); err != nil {
				return fmt.Errorf("sync module %s:%s in commit %s: %w", moduleDir, modIdentity, commitHash, err)
			}
			// module was synced successfully, add it to the cache
			if s.syncedModulesCommitsCache[modIdentity] == nil {
				s.syncedModulesCommitsCache[modIdentity] = make(map[string]struct{})
			}
			s.syncedModulesCommitsCache[modIdentity][commitHash] = struct{}{}
		}
	}
	return nil
}

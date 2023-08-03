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

	commitsTags               map[string][]string                                 // commits:[]tags
	branchesModulesToSync     map[string]map[string]bufmoduleref.ModuleIdentity   // branch:moduleDir:moduleIdentity
	modulesBranchesSyncPoints map[bufmoduleref.ModuleIdentity]map[string]git.Hash // moduleIdentity:branch:gitSyncPoint
}

func newSyncer(
	logger *zap.Logger,
	repo git.Repository,
	storageGitProvider storagegit.Provider,
	errorHandler ErrorHandler,
	options ...SyncerOption,
) (Syncer, error) {
	s := &syncer{
		logger:                    logger,
		repo:                      repo,
		storageGitProvider:        storageGitProvider,
		modulesDirsToSync:         make(map[string]struct{}),
		commitsTags:               make(map[string][]string),
		branchesModulesToSync:     make(map[string]map[string]bufmoduleref.ModuleIdentity),
		modulesBranchesSyncPoints: make(map[bufmoduleref.ModuleIdentity]map[string]git.Hash),
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
	s.printValidation()
	// first, default branch, if present
	defaultBranch := s.repo.DefaultBranch()
	if _, shouldSyncDefaultBranch := s.branchesModulesToSync[defaultBranch]; shouldSyncDefaultBranch {
		if err := s.syncBranch(ctx, defaultBranch, syncFunc); err != nil {
			return fmt.Errorf("sync default branch %q: %w", defaultBranch, err)
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
	// commitsToSync, err := s.commitsToSync(ctx, branch)
	// if err != nil {
	// 	return fmt.Errorf("finding commits to sync: %w", err)
	// }
	// if len(commitsToSync) == 0 {
	// 	s.logger.Debug(
	// 		"modules already up to date in branch",
	// 		zap.String("branch", branch),
	// 	)
	// 	return nil
	// }
	// for _, commitToSync := range commitsToSync {
	// 	for _, module := range s.modulesDirsToSync { // looping over the original sort order of modules
	// 		if _, shouldSyncModule := commitToSync.modules[module]; !shouldSyncModule {
	// 			continue
	// 		}
	// 		if err := s.syncModule(ctx, branch, commitToSync.commit, module, syncFunc); err != nil {
	// 			return fmt.Errorf("sync module %q in commit %q: %w", module.String(), commitToSync.commit.Hash().Hex(), err)
	// 		}
	// 	}
	// }
	return nil
}

// // syncModule looks for the module in the commit, and if found tries to validate it. If it is valid,
// // it invokes `syncFunc`.
// //
// // It does not return errors on invalid modules, but it will return any errors from `syncFunc` as
// // those may be transient.
// func (s *syncer) syncModule(
// 	ctx context.Context,
// 	branch string,
// 	commit git.Commit,
// 	module Module,
// 	syncFunc SyncFunc,
// ) error {
// 	logger := s.logger.With(
// 		zap.Stringer("commit", commit.Hash()),
// 		zap.Stringer("module", module),
// 	)
// 	builtNamedModule, err := s.builtNamedModuleAt(ctx, commit, module.Dir())
// 	if err != nil {
// 		if errors.Is(err, errModuleNotFound) {
// 			logger.Debug("module not found, skipping commit")
// 			return nil
// 		}
// 		if invalidConfigErr := (&invalidModuleConfigError{}); errors.As(err, &invalidConfigErr) {
// 			return s.errorHandler.InvalidModuleConfig(module, commit, err)
// 		}
// 		if errors.Is(err, errUnnamedModule) {
// 			logger.Debug("unnamed module, skipping commit")
// 			return nil
// 		}
// 		if buildModuleErr := (&buildModuleError{}); errors.As(err, &buildModuleErr) {
// 			return s.errorHandler.BuildFailure(module, commit, err)
// 		}
// 		return err
// 	}
// 	return syncFunc(
// 		ctx,
// 		newModuleCommitToSync(
// 			builtNamedModule.ModuleIdentity(), // TODO make sure it's the same name
// 			builtNamedModule.Bucket,
// 			commit,
// 			branch,
// 			s.commitsTags[commit.Hash().Hex()],
// 		),
// 	)
// }

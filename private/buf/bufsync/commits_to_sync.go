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

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/git"
	"go.uber.org/zap"
)

// syncableCommit holds the git commit and modules in that commit that need to be synced.
type syncableCommit struct {
	commit  git.Commit
	modules map[string]bufmodulebuild.BuiltModule // moduleDir:builtModule
}

// branchCommitsToSync returns a sorted commit+modules tuples array that are pending to sync for a
// branch. A commit in the array might have no modules to sync, in case those are skipped by the
// Syncer error handler.
func (s *syncer) branchCommitsToSync(ctx context.Context, branch string) ([]syncableCommit, error) {
	modulesToSync, ok := s.branchesModulesToSync[branch]
	if !ok || len(modulesToSync) == 0 {
		// branch should not be synced, or no modules to sync in that branch
		return nil, nil
	}
	// Copy all branch modules to sync and mark them as pending, until its starting sync point is
	// reached. They'll be removed from this list as its initial sync point is found.
	type moduleTarget struct {
		moduleIdentityInHEAD string
		expectedSyncPoint    *string
	}
	pendingModules := make(map[string]moduleTarget, len(modulesToSync))
	for moduleDir, moduleIdentityInHEAD := range modulesToSync {
		var expectedSyncPoint *string
		if moduleSyncPoints, ok := s.modulesBranchesSyncPoints[moduleIdentityInHEAD]; ok {
			if moduleBranchSyncPoint, ok := moduleSyncPoints[branch]; ok {
				expectedSyncPoint = &moduleBranchSyncPoint
			}
		}
		pendingModules[moduleDir] = moduleTarget{
			moduleIdentityInHEAD: moduleIdentityInHEAD,
			expectedSyncPoint:    expectedSyncPoint,
		}
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
		modulesDirsToSyncInThisCommit := make(map[string]bufmodulebuild.BuiltModule)
		modulesDirsFoundSyncPointInThisCommit := make(map[string]struct{})
		for moduleDir, pendingModule := range pendingModules {
			logger := s.logger.With(
				zap.String("branch", branch),
				zap.String("commit", commit.Hash().Hex()),
				zap.String("module directory", moduleDir),
				zap.String("module identity in HEAD", pendingModule.moduleIdentityInHEAD),
				zap.Stringp("expected sync point", pendingModule.expectedSyncPoint),
			)
			builtModule, readErr := s.readModuleAt(ctx, branch, commit, moduleDir)
			if readErr != nil {
				if err := s.errorHandler.ReadModule(readErr); err != nil {
					logger.Warn("error reading module, stop looking back", zap.Error(readErr))
					// receiving a not-nil error from the error handler means stop looking further back for
					// this module in this branch.
					modulesDirsFoundSyncPointInThisCommit[moduleDir] = struct{}{}
					continue
				}
				logger.Warn("error reading module, skipping commit", zap.Error(readErr))
				continue
			}
			isSynced, err := s.isGitCommitSynced(ctx, builtModule.ModuleIdentity(), commitHash)
			if err != nil {
				return fmt.Errorf(
					"check if module %s already synced git commit %s: %w",
					builtModule.ModuleIdentity().IdentityString(), commitHash, err,
				)
			}
			if !isSynced {
				modulesDirsToSyncInThisCommit[moduleDir] = *builtModule
				continue
			}
			// reached a commit that is already synced for this module
			modulesDirsFoundSyncPointInThisCommit[moduleDir] = struct{}{}
			if pendingModule.expectedSyncPoint == nil {
				// this module did not have an expected sync point for this branch, we probably reached the
				// beginning of the branch off another branch that is already synced.
				continue
			}
			if commitHash != *pendingModule.expectedSyncPoint {
				if s.repo.DefaultBranch() == branch {
					// TODO: add details to error message saying: "run again with --force-branch-sync <branch
					// name>" when we support a flag like that.
					return fmt.Errorf(
						"found synced git commit %s for default branch %s, but expected sync point was %s, did you rebase or reset your default branch?",
						commitHash,
						branch,
						*pendingModule.expectedSyncPoint,
					)
				}
				// syncing non-default branches from an unexpected sync point can be a common scenario in PRs,
				// we can just WARN and continue
				logger.Warn("unexpected_sync_point", zap.String("found_sync_point", commitHash))
			}
		}
		// clear modules that already found its sync point
		for moduleDir := range modulesDirsFoundSyncPointInThisCommit {
			delete(pendingModules, moduleDir)
		}
		commitsToSync = append(commitsToSync, syncableCommit{
			commit:  commit,
			modules: modulesDirsToSyncInThisCommit,
		})
		return nil
	}); err != nil && !errors.Is(err, stopLoopErr) {
		return nil, err
	}
	if len(commitsToSync) == 0 {
		return nil, nil
	}
	for moduleDir, pendingModule := range pendingModules {
		s.logger.Debug(
			"module did not find any synced git commit in the branch, will sync all the way from the beginning of the branch",
			zap.String("branch", branch),
			zap.String("module dir", moduleDir),
			zap.String("module identity in HEAD", pendingModule.moduleIdentityInHEAD),
			zap.Stringp("expected sync point", pendingModule.expectedSyncPoint),
		)
	}
	// https://github.com/golang/go/wiki/SliceTricks#reversing
	for i := len(commitsToSync)/2 - 1; i >= 0; i-- {
		opp := len(commitsToSync) - 1 - i
		commitsToSync[i], commitsToSync[opp] = commitsToSync[opp], commitsToSync[i]
	}
	return commitsToSync, nil
}

func (s *syncer) isGitCommitSynced(ctx context.Context, moduleIdentity bufmoduleref.ModuleIdentity, commitHash string) (bool, error) {
	// TODO: cache moduleIdentities and commits
	if s.syncedGitCommitChecker == nil {
		return false, nil
	}
	syncedCommits, err := s.syncedGitCommitChecker(ctx, moduleIdentity, map[string]struct{}{commitHash: {}})
	if err != nil {
		return false, err
	}
	_, synced := syncedCommits[commitHash]
	return synced, nil
}

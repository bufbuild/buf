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

// syncableCommit holds the modules that need to be synced in a git commit.
type syncableCommit struct {
	commit  git.Commit
	modules map[string]bufmodulebuild.BuiltModule // moduleDir:builtModule
}

// branchCommitsToSync returns a sorted commit+modules tuples array that are pending to sync for a
// branch. A commit in the array might have no modules to sync if those are skipped by the
// Syncer error handler, or are a found sync point.
func (s *syncer) branchCommitsToSync(ctx context.Context, branch string) ([]syncableCommit, error) {
	modulesToSync, ok := s.branchesModulesToSync[branch]
	if !ok || len(modulesToSync) == 0 {
		// branch should not be synced, or no modules to sync in that branch
		return nil, nil
	}
	// Copy all branch modules to sync and mark them as pending, until its starting sync point is
	// reached. They'll be removed from this list as its initial sync point is found.
	type moduleTarget struct {
		moduleIdentityInHEAD bufmoduleref.ModuleIdentity
		expectedSyncPoint    *string
	}
	pendingModules := make(map[string]moduleTarget, len(modulesToSync))
	for moduleDir, moduleIdentityInHEAD := range modulesToSync {
		var expectedSyncPoint *string
		if moduleSyncPoints, ok := s.modulesBranchesLastSyncPoints[moduleIdentityInHEAD.IdentityString()]; ok {
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
		modulesDirsToStopInThisCommit := make(map[string]struct{})
		for moduleDir, pendingModule := range pendingModules {
			moduleIdentityInHEAD := pendingModule.moduleIdentityInHEAD.IdentityString()
			logger := s.logger.With(
				zap.String("branch", branch),
				zap.String("commit", commit.Hash().Hex()),
				zap.String("module directory", moduleDir),
				zap.String("module identity in branch HEAD", moduleIdentityInHEAD),
				zap.Stringp("expected sync point", pendingModule.expectedSyncPoint),
			)
			// check if the module identity already synced this commit
			isSynced, err := s.isGitCommitSynced(ctx, pendingModule.moduleIdentityInHEAD, commitHash)
			if err != nil {
				return fmt.Errorf(
					"checking if module %s already synced git commit %s: %w",
					moduleIdentityInHEAD, commitHash, err,
				)
			}
			if isSynced {
				// reached a commit that is already synced for this module, we can stop looking for this
				// module dir
				modulesDirsToStopInThisCommit[moduleDir] = struct{}{}
				if pendingModule.expectedSyncPoint == nil {
					// this module did not have an expected sync point for this branch, we probably reached
					// the beginning of the branch off another branch that is already synced.
					logger.Debug("git commit already synced, stop looking back in branch")
					continue
				}
				if commitHash != *pendingModule.expectedSyncPoint {
					if s.repo.DefaultBranch() == branch {
						// TODO: add details to error message saying: "run again with --force-branch-sync <branch
						// name>" when we support a flag like that.
						return fmt.Errorf(
							"found synced git commit %s for default branch %s, but expected sync point was %s, "+
								"did you rebase or reset your default branch?",
							commitHash,
							branch,
							*pendingModule.expectedSyncPoint,
						)
					}
					// syncing non-default branches from an unexpected sync point can be a common scenario in
					// PRs, we can just WARN and continue
					logger.Warn(
						"unexpected sync point reached, stop looking back in branch",
						zap.String("found_sync_point", commitHash),
					)
				} else {
					logger.Debug("expected sync point reached, stop looking back in branch")
				}
				continue
			}
			// git commit is not synced, attempt to read module in the commit:moduleDir
			builtModule, readErr := s.readModuleAt(ctx, branch, commit, moduleDir, &moduleIdentityInHEAD)
			if readErr != nil {
				if s.errorHandler.StopLookback(readErr) {
					logger.Warn("read module at commit failed, stop looking back in branch", zap.Error(readErr))
					modulesDirsToStopInThisCommit[moduleDir] = struct{}{}
					continue
				}
				logger.Warn("read module at commit failed, skipping commit", zap.Error(readErr))
				continue
			}
			// add the read module to sync
			modulesDirsToSyncInThisCommit[moduleDir] = *builtModule
		}
		// clear modules that are set to stop in this commit
		for moduleDir := range modulesDirsToStopInThisCommit {
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
	// if we have no commits to sync, no need to make more checks, bail early
	if len(commitsToSync) == 0 {
		return nil, nil
	}
	// we reached the branch starting point, do we still have pending modules?
	for moduleDir, pendingModule := range pendingModules {
		moduleIdentityInHEAD := pendingModule.moduleIdentityInHEAD.IdentityString()
		logger := s.logger.With(
			zap.String("branch", branch),
			zap.String("module directory", moduleDir),
			zap.String("module identity in branch HEAD", moduleIdentityInHEAD),
		)
		if pendingModule.expectedSyncPoint != nil {
			if branch == s.repo.DefaultBranch() {
				return nil, fmt.Errorf(
					"module %s in directory %s in the default branch %s did not find its expected sync point %s, aborting sync",
					moduleIdentityInHEAD,
					moduleDir,
					branch,
					*pendingModule.expectedSyncPoint,
				)
			}
			logger.Warn(
				"module did not find its expected sync point, or any other synced git commit, "+
					"will sync all the way from the beginning of the branch",
				zap.String("expected sync point", *pendingModule.expectedSyncPoint),
			)
		}
		logger.Debug(
			"module without expected sync point did not find any synced git commit, " +
				"will sync all the way from the beginning of the branch",
		)
	}
	// reverse commits to sync, to leave them in time order parent -> children
	// https://github.com/golang/go/wiki/SliceTricks#reversing
	for i := len(commitsToSync)/2 - 1; i >= 0; i-- {
		opp := len(commitsToSync) - 1 - i
		commitsToSync[i], commitsToSync[opp] = commitsToSync[opp], commitsToSync[i]
	}
	return commitsToSync, nil
}

// isGitCommitSynced checks if a commit hash is already synced to a remote BSR module.
func (s *syncer) isGitCommitSynced(ctx context.Context, moduleIdentity bufmoduleref.ModuleIdentity, commitHash string) (bool, error) {
	if s.syncedGitCommitChecker == nil {
		return false, nil
	}
	modIdentity := moduleIdentity.IdentityString()
	// check local cache first
	if syncedModuleCommits, ok := s.syncedModulesCommitsCache[modIdentity]; ok {
		if _, commitSynced := syncedModuleCommits[commitHash]; commitSynced {
			return true, nil
		}
	}
	// not in the cache, request remote check
	syncedModuleCommits, err := s.syncedGitCommitChecker(ctx, moduleIdentity, map[string]struct{}{commitHash: {}})
	if err != nil {
		return false, err
	}
	_, commitSynced := syncedModuleCommits[commitHash]
	if commitSynced {
		// populate local cache
		if s.syncedModulesCommitsCache[modIdentity] == nil {
			s.syncedModulesCommitsCache[modIdentity] = make(map[string]struct{})
		}
		s.syncedModulesCommitsCache[modIdentity][commitHash] = struct{}{}
	}
	return commitSynced, nil
}

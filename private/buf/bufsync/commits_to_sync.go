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

	"github.com/bufbuild/buf/private/pkg/git"
	"go.uber.org/zap"
)

// syncableCommit holds the git commit and module directories in that commit that need to be synced.
type syncableCommit struct {
	commit      git.Commit
	modulesDirs map[string]struct{}
}

// commitsToSync returns a sorted commit+modules tuples array that are pending to sync for a branch.
func (s *syncer) commitsToSync(
	ctx context.Context,
	branch string,
) ([]syncableCommit, error) {
	// First, copy all modules dirs to sync and mark them as pending, until its starting sync point is
	// reached. They'll be removed from this list as its initial sync point is found.
	pendingModulesDirs := make(map[string]struct{}, len(s.modulesDirsToSync))
	for module := range s.modulesDirsToSync {
		pendingModulesDirs[module] = struct{}{}
	}
	var commitsToSync []syncableCommit
	// travel branch commits from HEAD and check if they're already synced, until finding a synced git
	// commit, or adding them all to be synced
	stopLoopErr := errors.New("stop loop")
	if err := s.repo.ForEachCommit(branch, func(commit git.Commit) error {
		if len(pendingModulesDirs) == 0 {
			// no more pending modules to sync, no need to keep navigating the branch
			return stopLoopErr
		}
		commitHash := commit.Hash().Hex()
		modulesToSyncInThisCommit := make(map[Module]struct{})
		modulesFoundSyncPointInThisCommit := make(map[Module]struct{})
		for moduleDir := range pendingModulesDirs {
			logger := s.logger.With(
				zap.String("branch", branch),
				zap.String("commit", commit.Hash().Hex()),
				zap.String("module directory", moduleDir),
			)
			builtModule, err := s.builtNamedModuleAt(ctx, commit, moduleDir)
			if err != nil {
				if errors.Is(err, errModuleNotFound) {
					logger.Debug("module not found, module in commit won't be synced")
				}
				if invalidConfigErr := (&invalidModuleConfigError{}); errors.As(err, &invalidConfigErr) {
					s.errorHandler.InvalidModuleConfig(module, commit, err)
				}
				if errors.Is(err, errUnnamedModule) {
					logger.Debug("unnamed module, skipping commit")
					return nil
				}
				if buildModuleErr := (&buildModuleError{}); errors.As(err, &buildModuleErr) {
					return s.errorHandler.BuildFailure(module, commit, err)
				}
				continue
			}
			isSynced, err := s.isGitCommitSynced(ctx, moduleDir, commitHash)
			if err != nil {
				return fmt.Errorf("check if module %q already synced git commit %q: %w", moduleDir.String(), commitHash, err)
			}
			if !isSynced {
				modulesToSyncInThisCommit[moduleDir] = struct{}{}
				continue
			}
			// reached a commit that is already synced for this module
			modulesFoundSyncPointInThisCommit[moduleDir] = struct{}{}
			expectedSyncPoint, ok := modulesSyncPoints[moduleDir]
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
					zap.String("module", moduleDir.String()),
				)
			}
		}
		// clear modules that already found its sync point
		for module := range modulesFoundSyncPointInThisCommit {
			delete(pendingModulesDirs, module)
		}
		if len(modulesToSyncInThisCommit) > 0 {
			commitsToSync = append(commitsToSync, syncableCommit{
				commit:  commit,
				modules: modulesToSyncInThisCommit,
			})
		} else {
			// no modules to sync in this commit, we should not have any pending modules
			if len(pendingModulesDirs) > 0 {
				return fmt.Errorf(
					"commit %q has no modules to sync, but still has pending modules %v",
					commitHash,
					pendingModulesDirs,
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

func (s *syncer) isGitCommitSynced(ctx context.Context, moduleDir string, commitHash string) (bool, error) {
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

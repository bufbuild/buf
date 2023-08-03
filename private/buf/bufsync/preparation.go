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
	"go.uber.org/zap"
)

func (s *syncer) prepareSync(ctx context.Context) error {
	// Populate all tags locations.
	if err := s.repo.ForEachTag(func(tag string, commitHash git.Hash) error {
		s.commitsTags[commitHash.Hex()] = append(s.commitsTags[commitHash.Hex()], tag)
		return nil
	}); err != nil {
		return fmt.Errorf("load tags: %w", err)
	}
	// Populate all branches to be synced.
	allRemoteBranches := make(map[string]struct{})
	if err := s.repo.ForEachBranch(func(branch string, _ git.Hash) error {
		allRemoteBranches[branch] = struct{}{}
		return nil
	}); err != nil {
		return fmt.Errorf("looping over repo remote branches: %w", err)
	}
	if s.syncAllBranches {
		// make sure the default branch is present in the branches to sync
		defaultBranch := s.repo.DefaultBranch()
		if _, isDefaultBranchPushedInRemote := allRemoteBranches[defaultBranch]; !isDefaultBranchPushedInRemote {
			return fmt.Errorf("default branch %s is not present in 'origin' remote", defaultBranch)
		}
		for remoteBranch := range allRemoteBranches {
			s.branchesModulesToSync[remoteBranch] = make(map[string]bufmoduleref.ModuleIdentity)
		}
	} else {
		// only sync current branch, make sure it's present in the remote
		currentBranch := s.repo.CurrentBranch()
		if _, isCurrentBranchPushedInRemote := allRemoteBranches[currentBranch]; !isCurrentBranchPushedInRemote {
			return fmt.Errorf("current branch %s is not present in 'origin' remote", currentBranch)
		}
		s.branchesModulesToSync[currentBranch] = make(map[string]bufmoduleref.ModuleIdentity)
		s.logger.Debug("current branch", zap.String("name", currentBranch))
	}
	// Populate module identities from HEAD, and its sync points if any
	allModulesIdentitiesToSync := make(map[bufmoduleref.ModuleIdentity]struct{})
	for branch := range s.branchesModulesToSync {
		headCommit, err := s.repo.HEADCommit(branch)
		if err != nil {
			return fmt.Errorf("reading head commit for branch %s: %w", branch, err)
		}
		for moduleDir := range s.modulesDirsToSync {
			builtModule, readErr := s.readModuleAt(ctx, branch, headCommit, moduleDir)
			if readErr != nil {
				// any error reading module in HEAD, skip syncing that module in that branch
				s.logger.Warn(
					"read module from HEAD failed, module won't be synced for this branch",
					zap.Error(readErr),
				)
				continue
			}
			if builtModule == nil || builtModule.ModuleIdentity() == nil {
				return fmt.Errorf("nil built module or built module identity for dir %s in branch %s HEAD", moduleDir, branch)
			}
			// there is a valid module in the module dir at the HEAD of this branch, enqueue it for sync
			s.branchesModulesToSync[branch][moduleDir] = builtModule.ModuleIdentity()
			// do we have a remote git sync point for this module+branch?
			moduleBranchSyncpoint, err := s.resolveSyncPoint(ctx, builtModule.ModuleIdentity(), branch)
			if err != nil {
				return fmt.Errorf(
					"resolve sync point for module %s in branch %s: %w",
					branch, builtModule.ModuleIdentity().IdentityString(), err,
				)
			}
			allModulesIdentitiesToSync[builtModule.ModuleIdentity()] = struct{}{}
			if s.modulesBranchesSyncPoints[builtModule.ModuleIdentity()] == nil {
				s.modulesBranchesSyncPoints[builtModule.ModuleIdentity()] = make(map[string]git.Hash)
			}
			if moduleBranchSyncpoint != nil {
				s.modulesBranchesSyncPoints[builtModule.ModuleIdentity()][branch] = moduleBranchSyncpoint
			}
		}
	}
	// make sure all module identities we are about to sync in all branches have the same BSR default
	// branch as the local git default branch.
	for moduleIdentity := range allModulesIdentitiesToSync {
		if err := s.validateDefaultBranch(ctx, moduleIdentity); err != nil {
			return fmt.Errorf("validate default branch for module %s: %w", moduleIdentity.IdentityString(), err)
		}
	}
	return nil
}

// resolveSyncPoint resolves a sync point for a particular module identity and branch.
func (s *syncer) resolveSyncPoint(ctx context.Context, module bufmoduleref.ModuleIdentity, branch string) (git.Hash, error) {
	// If resumption is not enabled, we can bail early.
	if s.syncPointResolver == nil {
		return nil, nil
	}
	syncPoint, err := s.syncPointResolver(ctx, module, branch)
	if err != nil {
		return nil, fmt.Errorf("resolve sync point for module %s: %w", module.IdentityString(), err)
	}
	if syncPoint == nil {
		// no sync point for that module in that branch
		return nil, nil
	}
	// Validate that the commit pointed to by the sync point exists in the git repo.
	if _, err := s.repo.Objects().Commit(syncPoint); err != nil {
		isDefaultBranch := branch == s.repo.DefaultBranch()
		return nil, s.errorHandler.InvalidSyncPoint(module, branch, syncPoint, isDefaultBranch, err)
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
		return fmt.Errorf("getting bsr module: %w", err)
	}
	if bsrDefaultBranch != expectedDefaultGitBranch {
		return fmt.Errorf(
			"remote module default branch %s does not match the git repository's default branch %s, aborting sync",
			bsrDefaultBranch, expectedDefaultGitBranch,
		)
	}
	return nil
}

// TODO: remove
func (s *syncer) printValidation() {
	branchesModulesToSync := make(map[string]map[string]string)
	for branch, modules := range s.branchesModulesToSync {
		m := make(map[string]string)
		for moduleDir, moduleIdentity := range modules {
			m[moduleDir] = moduleIdentity.IdentityString()
		}
		branchesModulesToSync[branch] = m
	}
	modulesBranchesSyncPoints := make(map[string]map[string]string)
	for moduleIdentity, branches := range s.modulesBranchesSyncPoints {
		b := make(map[string]string)
		for branch, syncPoint := range branches {
			var syncPointHash string
			if syncPoint != nil {
				syncPointHash = syncPoint.Hex()
			}
			b[branch] = syncPointHash
		}
		modulesBranchesSyncPoints[moduleIdentity.IdentityString()] = b
	}
	s.logger.Debug(
		"sync prepared",
		zap.Any("modulesDirsToSync", s.modulesDirsToSync),
		zap.Any("commitsTags", s.commitsTags),
		zap.Any("branchesModulesToSync", branchesModulesToSync),
		zap.Any("modulesBranchesSyncPoints", modulesBranchesSyncPoints),
	)
}

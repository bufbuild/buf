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

// branchModuleTarget holds the target information for a single module in a branch.
type branchModuleTarget struct {
	// identityInHEAD is the module identity read from the HEAD commit in the branch.
	identityInHEAD bufmoduleref.ModuleIdentity
	// syncPoint is the latest git commit in this branch that is already synced in this identity.
	syncPoint git.Hash
}

// prepareSync gathers repo, modules, and target information and stores it in the syncer, before the
// actual sync process.
func (s *syncer) prepareSync(ctx context.Context) error {
	// Gather all tags locations. TODO: Only take into account remote tags.
	s.tagsByCommitHash = make(map[string][]string)
	if err := s.repo.ForEachTag(func(tag string, commitHash git.Hash) error {
		s.tagsByCommitHash[commitHash.Hex()] = append(s.tagsByCommitHash[commitHash.Hex()], tag)
		return nil
	}); err != nil {
		return fmt.Errorf("load tags: %w", err)
	}
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
			return fmt.Errorf(`default branch %q is not present in "origin" remote`, defaultBranch)
		}
		for remoteBranch := range allRemoteBranches {
			s.branchesModulesToSync[remoteBranch] = make(map[string]branchModuleTarget)
		}
	} else {
		// only sync current branch, make sure it's present in remote
		currentBranch := s.repo.CurrentBranch()
		if _, isCurrentBranchPushedInRemote := allRemoteBranches[currentBranch]; !isCurrentBranchPushedInRemote {
			return fmt.Errorf(`current branch %q is not present in "origin" remote`, currentBranch)
		}
		s.branchesModulesToSync = map[string]map[string]branchModuleTarget{
			currentBranch: make(map[string]branchModuleTarget),
		}
		s.logger.Debug("current branch", zap.String("name", currentBranch))
	}
	// populate each branch with its module dirs and expected module identities from HEAD
	s.allModulesToSync = make(map[bufmoduleref.ModuleIdentity]struct{})
	for branch := range s.branchesModulesToSync {
		headCommit, err := s.repo.HEADCommit(branch)
		if err != nil {
			return fmt.Errorf("get HEAD commit at branch %q: %w", branch, err)
		}
		for moduleDir := range s.modulesDirsToSync {
			module, err := s.builtNamedModuleAt(ctx, headCommit, moduleDir)
			if err != nil {
				s.logger.Warn(
					"cannot determine remote module identity in head commit for branch, won't sync this module for this branch",
					zap.String("branch", branch),
					zap.String("head commit", headCommit.Hash().Hex()),
					zap.String("module dir", moduleDir),
					zap.Error(err),
				)
				continue
			}
			s.branchesModulesToSync[branch][moduleDir] = branchModuleTarget{identityInHEAD: module.ModuleIdentity()}
			s.allModulesToSync[module.ModuleIdentity()] = struct{}{}
		}
	}
	if len(s.allModulesToSync) == 0 {
		return errors.New("no modules to sync in any branch, aborting sync")
	}
	return nil
}

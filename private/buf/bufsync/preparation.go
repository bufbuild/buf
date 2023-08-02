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

	"github.com/bufbuild/buf/private/pkg/git"
	"go.uber.org/zap"
)

func (s *syncer) prepareSync(ctx context.Context) error {
	// Gather all tags locations. TODO: Only take into account remote tags.
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
			s.branchesToSync[remoteBranch] = struct{}{}
		}
	} else {
		// only sync current branch, make sure it's present in the remote
		currentBranch := s.repo.CurrentBranch()
		if _, isCurrentBranchPushedInRemote := allRemoteBranches[currentBranch]; !isCurrentBranchPushedInRemote {
			return fmt.Errorf(`current branch %q is not present in "origin" remote`, currentBranch)
		}
		s.branchesToSync = map[string]struct{}{currentBranch: {}}
		s.logger.Debug("current branch", zap.String("name", currentBranch))
	}
	return nil
}

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
	"fmt"
	"strings"

	"go.uber.org/zap"
)

// printSyncPreparation prints information gathered at the sync preparation step.
func (s *syncer) printSyncPreparation() {
	s.logger.Debug(
		"sync preparation",
		zap.Any("modulesDirsToSync", s.modulesDirsForSync),
		zap.Any("commitsTags", s.commitsToTags),
		zap.Any("branchesModulesToSync", s.branchesToModulesForSync),
		zap.Any("modulesBranchesSyncPoints", s.modulesToBranchesLastSyncPoints),
	)
}

// printCommitsForSync prints syncable commits for a given branch.
func (s *syncer) printCommitsForSync(branch string, syncableCommits []*syncableCommit) {
	printableCommits := make([]map[string]string, 0)
	for _, sCommit := range syncableCommits {
		var commitModules []string
		for moduleDir, builtModule := range sCommit.modules {
			commitModules = append(commitModules, moduleDir+":"+builtModule.ModuleIdentity().IdentityString())
		}
		printableCommits = append(printableCommits, map[string]string{
			sCommit.commit.Hash().Hex(): fmt.Sprintf("(%d)[%s]", len(commitModules), strings.Join(commitModules, ", ")),
		})
	}
	s.logger.Debug(
		"branch commits to sync",
		zap.String("branch", branch),
		zap.Any("commits", printableCommits),
	)
}

// Copyright 2020-2024 Buf Technologies, Inc.
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

	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

type executionPlan struct {
	moduleBranchesToSync []ModuleBranch
	moduleTagsToSync     []ModuleTags
}

func newExecutionPlan(
	sortedModuleDirs []string,
	moduleBranchesToSync []ModuleBranch,
	moduleTagsToSync []ModuleTags,
) (*executionPlan, error) {
	sortedModuleDirIndexes := make(map[string]int)
	for i, dir := range sortedModuleDirs {
		sortedModuleDirIndexes[dir] = i
	}
	for _, moduleBranch := range moduleBranchesToSync {
		if _, ok := sortedModuleDirIndexes[moduleBranch.Directory()]; !ok {
			return nil, fmt.Errorf("sort index for moduleDir %q is unknown", moduleBranch.Directory())
		}
	}
	sortedBranchesToSync := make([]ModuleBranch, len(moduleBranchesToSync))
	copy(sortedBranchesToSync, moduleBranchesToSync)
	// ModuleBranches are sorted by moduleDir, then branch name.
	slices.SortFunc(sortedBranchesToSync, func(a, b ModuleBranch) int {
		//  We retain the order of moduleDirs passed in.
		if sortedModuleDirIndexes[a.Directory()] != sortedModuleDirIndexes[b.Directory()] {
			return sortedModuleDirIndexes[a.Directory()] - sortedModuleDirIndexes[b.Directory()]
		}
		// NOTE: Protected branches should ideally be synced prior to non-protected branches,
		// because they are preferred backfill sources for all branches that follow. But this is
		// not strictly required.
		if a.BranchName() > b.BranchName() {
			return 1
		}
		if a.BranchName() < b.BranchName() {
			return -1
		}
		return 0
	})
	return &executionPlan{
		moduleBranchesToSync: sortedBranchesToSync,
		moduleTagsToSync:     moduleTagsToSync,
	}, nil
}

func (p *executionPlan) ModuleBranchesToSync() []ModuleBranch {
	return p.moduleBranchesToSync
}

func (p *executionPlan) ModuleTagsToSync() []ModuleTags {
	return p.moduleTagsToSync
}

func (p *executionPlan) Nop() bool {
	return len(p.moduleBranchesToSync) == 0 && len(p.moduleTagsToSync) == 0
}

func (p *executionPlan) log(logger *zap.Logger) {
	if !logger.Level().Enabled(zap.DebugLevel) {
		return
	}
	for _, branch := range p.ModuleBranchesToSync() {
		var commitSHAs []string
		for _, commit := range branch.CommitsToSync() {
			commitSHAs = append(commitSHAs, commit.Commit().Hash().Hex())
		}
		logger.Debug(
			"sync plan for module branch",
			zap.String("branch", branch.BranchName()),
			zap.String("moduleDir", branch.Directory()),
			zap.String("moduleIdentity", branch.TargetModuleIdentity().IdentityString()),
			zap.Strings("commitsToSync", commitSHAs),
		)
	}
	for _, moduleTags := range p.ModuleTagsToSync() {
		for _, commitTags := range moduleTags.TaggedCommitsToSync() {
			logger.Debug(
				"sync plan for tags for module commit",
				zap.Stringer("commit", commitTags.Commit()),
				zap.String("moduleIdentity", moduleTags.TargetModuleIdentity().IdentityString()),
				zap.Strings("tagsToSync", commitTags.Tags()),
			)
		}
	}
}

var _ ExecutionPlan = (*executionPlan)(nil)

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
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

type executionPlan struct {
	moduleBranchesToSync []ModuleBranch
	taggedCommitsToSync  []ModuleCommit
}

func newExecutionPlan(
	moduleBranchesToSync []ModuleBranch,
	taggedCommitsToSync []ModuleCommit,
) *executionPlan {
	sortedBranchesToSync := make([]ModuleBranch, len(moduleBranchesToSync))
	copy(sortedBranchesToSync, moduleBranchesToSync)
	slices.SortFunc(sortedBranchesToSync, func(a, b ModuleBranch) int {
		if a.Directory() > b.Directory() {
			return 1
		}
		if a.Directory() < b.Directory() {
			return -1
		}
		if a.Name() > b.Name() {
			return 1
		}
		if a.Name() < b.Name() {
			return -1
		}
		return 0
	})
	return &executionPlan{
		moduleBranchesToSync: sortedBranchesToSync,
		taggedCommitsToSync:  taggedCommitsToSync,
	}
}

func (p *executionPlan) ModuleBranchesToSync() []ModuleBranch {
	return p.moduleBranchesToSync
}

func (p *executionPlan) TaggedCommitsToSync() []ModuleCommit {
	return p.taggedCommitsToSync
}

func (p *executionPlan) Nop() bool {
	return len(p.moduleBranchesToSync) == 0 && len(p.taggedCommitsToSync) == 0
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
			"branch plan for module",
			zap.String("branch", branch.Name()),
			zap.String("moduleDir", branch.Directory()),
			zap.String("moduleIdentity", branch.ModuleIdentity().IdentityString()),
			zap.Strings("commitsToSync", commitSHAs),
		)
	}
	for _, commitTags := range p.TaggedCommitsToSync() {
		logger.Debug(
			"tag plan for module",
			zap.Stringer("commit", commitTags.Commit()),
			zap.String("moduleIdentity", commitTags.ModuleIdentity().IdentityString()),
			zap.Strings("tagsToSync", commitTags.Tags()),
		)
	}
}

var _ ExecutionPlan = (*executionPlan)(nil)

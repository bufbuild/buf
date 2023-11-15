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
	"golang.org/x/exp/slices"
)

type executionPlan struct {
	// Branches are sorted by moduleDir, then by branch.
	branchesToSync []*syncableBranch
	tagsToSync     []*syncableCommitTags
}

func newExecutionPlan(
	branchesToSync []*syncableBranch,
	tagsToSync []*syncableCommitTags,
) *executionPlan {
	sortedBranchesToSync := make([]*syncableBranch, len(branchesToSync))
	copy(sortedBranchesToSync, branchesToSync)
	slices.SortFunc(sortedBranchesToSync, func(a, b *syncableBranch) int {
		if a.moduleDir > b.moduleDir {
			return 1
		}
		if a.moduleDir < b.moduleDir {
			return -1
		}
		if a.name > b.name {
			return 1
		}
		if a.name < b.name {
			return -1
		}
		return 0
	})
	return &executionPlan{
		branchesToSync: sortedBranchesToSync,
		tagsToSync:     tagsToSync,
	}
}

func (p *executionPlan) hasAnythingToSync() bool {
	return len(p.branchesToSync) > 0 || len(p.tagsToSync) > 0
}

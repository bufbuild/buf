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
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/git"
)

// The set of all commits for a module identity that are tagged, along with the tags pointing
// to those commits.
//
// Note that these commits may not have valid modules on them and therefore should be
// synced after all/the branch(es) is/are synced, with tags referencing commits that don't exist
// in the remote pruned from this list.
type syncableCommitTags struct {
	moduleIdentity bufmoduleref.ModuleIdentity
	commit         git.Hash
	tagsToSync     []string
}

func newSyncableCommitTags(
	moduleIdentity bufmoduleref.ModuleIdentity,
	commit git.Hash,
	tagsToSync []string,
) *syncableCommitTags {
	return &syncableCommitTags{
		moduleIdentity: moduleIdentity,
		commit:         commit,
		tagsToSync:     tagsToSync,
	}
}

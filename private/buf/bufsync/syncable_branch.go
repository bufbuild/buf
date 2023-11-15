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
)

// syncableBranch is a branch that has a module on it at a particular dir along with it's
// commits that should be synced
type syncableBranch struct {
	// The name of the branch.
	name string
	// The identity of the module to use when syncing from this branch. This is either the
	// identity override, or the module identity at HEAD of the branch.
	moduleIdentity bufmoduleref.ModuleIdentity
	// The dir where the module is found in the branch.
	moduleDir string
	// The commits to sync for this branch, ordered in the order in which they should be synced.
	commitsToSync []*syncableCommit
}

func newSyncableBranch(
	name string,
	dir string,
	identity bufmoduleref.ModuleIdentity,
	commits []*syncableCommit,
) *syncableBranch {
	return &syncableBranch{
		name:           name,
		moduleDir:      dir,
		moduleIdentity: identity,
		commitsToSync:  commits,
	}
}

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
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
)

type moduleBranch struct {
	name                 string
	moduleDir            string
	targetModuleIdentity bufmoduleref.ModuleIdentity
	commitsToSync        []ModuleCommit
}

func newModuleBranch(
	name string,
	dir string,
	targetModuleIdentity bufmoduleref.ModuleIdentity,
	commitsToSync []ModuleCommit,
) *moduleBranch {
	return &moduleBranch{
		name:                 name,
		moduleDir:            dir,
		targetModuleIdentity: targetModuleIdentity,
		commitsToSync:        commitsToSync,
	}
}

func (b *moduleBranch) BranchName() string {
	return b.name
}

func (b *moduleBranch) Directory() string {
	return b.moduleDir
}

func (b *moduleBranch) TargetModuleIdentity() bufmoduleref.ModuleIdentity {
	return b.targetModuleIdentity
}

func (b *moduleBranch) CommitsToSync() []ModuleCommit {
	return b.commitsToSync
}

var _ ModuleBranch = (*moduleBranch)(nil)

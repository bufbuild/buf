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

import "github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"

type moduleTags struct {
	targetModuleIdentity bufmoduleref.ModuleIdentity
	taggedCommitsToSync  []TaggedCommit
}

func newModuleTags(
	targetModuleIdentity bufmoduleref.ModuleIdentity,
	taggedCommitsToSync []TaggedCommit,
) *moduleTags {
	return &moduleTags{
		targetModuleIdentity: targetModuleIdentity,
		taggedCommitsToSync:  taggedCommitsToSync,
	}
}

func (b *moduleTags) TargetModuleIdentity() bufmoduleref.ModuleIdentity {
	return b.targetModuleIdentity
}

func (b *moduleTags) TaggedCommitsToSync() []TaggedCommit {
	return b.taggedCommitsToSync
}

var _ ModuleTags = (*moduleTags)(nil)

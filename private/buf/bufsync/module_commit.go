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
	"github.com/bufbuild/buf/private/pkg/storage"
)

type moduleCommit struct {
	identity bufmoduleref.ModuleIdentity
	bucket   storage.ReadBucket
	commit   git.Commit
	branch   string
	tags     []string
}

func newModuleCommit(
	identity bufmoduleref.ModuleIdentity,
	bucket storage.ReadBucket,
	commit git.Commit,
	branch string,
	tags []string,
) ModuleCommit {
	return &moduleCommit{
		identity: identity,
		bucket:   bucket,
		commit:   commit,
		branch:   branch,
		tags:     tags,
	}
}

func (m *moduleCommit) Identity() bufmoduleref.ModuleIdentity {
	return m.identity
}

func (m *moduleCommit) Bucket() storage.ReadBucket {
	return m.bucket
}

func (m *moduleCommit) Commit() git.Commit {
	return m.commit
}

func (m *moduleCommit) Branch() string {
	return m.branch
}

func (m *moduleCommit) Tags() []string {
	return m.tags
}

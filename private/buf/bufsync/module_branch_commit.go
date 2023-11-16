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

type moduleBranchCommit struct {
	moduleCommit ModuleCommit
	branch       string
	bucket       storage.ReadBucket
}

func newModuleBranchCommit(
	moduleCommit ModuleCommit,
	branch string,
	bucket storage.ReadBucket,
) ModuleBranchCommit {
	return &moduleBranchCommit{
		moduleCommit: moduleCommit,
		branch:       branch,
		bucket:       bucket,
	}
}

func (m *moduleBranchCommit) Commit() git.Commit {
	return m.moduleCommit.Commit()
}

func (m *moduleBranchCommit) Tags() []string {
	return m.moduleCommit.Tags()
}

func (m *moduleBranchCommit) Directory() string {
	return m.moduleCommit.Directory()
}

func (m *moduleBranchCommit) ModuleIdentity() bufmoduleref.ModuleIdentity {
	return m.moduleCommit.ModuleIdentity()
}

func (m *moduleBranchCommit) Branch() string {
	return m.branch
}

func (m *moduleBranchCommit) Bucket() storage.ReadBucket {
	return m.bucket
}

var _ ModuleBranchCommit = (*moduleBranchCommit)(nil)

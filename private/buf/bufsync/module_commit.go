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

type moduleCommit struct {
	commit         git.Commit
	tags           []string
	directory      string
	moduleIdentity bufmoduleref.ModuleIdentity
}

func newModuleCommit(
	commit git.Commit,
	tags []string,
	directory string,
	identity bufmoduleref.ModuleIdentity,
) ModuleCommit {
	return &moduleCommit{
		commit:         commit,
		tags:           tags,
		directory:      directory,
		moduleIdentity: identity,
	}
}

func (m *moduleCommit) Commit() git.Commit {
	return m.commit
}

func (m *moduleCommit) Tags() []string {
	return m.tags
}

func (m *moduleCommit) Directory() string {
	return m.directory
}

func (m *moduleCommit) ModuleIdentity() bufmoduleref.ModuleIdentity {
	return m.moduleIdentity
}

var _ ModuleCommit = (*moduleCommit)(nil)

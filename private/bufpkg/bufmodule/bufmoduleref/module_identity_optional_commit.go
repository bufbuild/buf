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

package bufmoduleref

type moduleIdentityOptionalCommit struct {
	remote     string
	owner      string
	repository string
	commit     string
}

func newModuleIdentityOptionalCommit(
	remote string,
	owner string,
	repository string,
	commit string,
) (*moduleIdentityOptionalCommit, error) {
	moduleIdentityOptionalCommit := &moduleIdentityOptionalCommit{
		remote:     remote,
		owner:      owner,
		repository: repository,
		commit:     commit,
	}
	if err := validateModuleIdentityOptionalCommit(moduleIdentityOptionalCommit); err != nil {
		return nil, err
	}
	return moduleIdentityOptionalCommit, nil
}

func (m *moduleIdentityOptionalCommit) Remote() string {
	return m.remote
}

func (m *moduleIdentityOptionalCommit) Owner() string {
	return m.owner
}

func (m *moduleIdentityOptionalCommit) Repository() string {
	return m.repository
}

func (m *moduleIdentityOptionalCommit) Commit() string {
	return m.commit
}

func (m *moduleIdentityOptionalCommit) String() string {
	if m.commit == "" {
		return m.remote + "/" + m.owner + "/" + m.repository
	}
	return m.remote + "/" + m.owner + "/" + m.repository + ":" + m.commit
}

func (m *moduleIdentityOptionalCommit) IdentityString() string {
	return m.remote + "/" + m.owner + "/" + m.repository
}

func (*moduleIdentityOptionalCommit) isModuleOwner()                  {}
func (*moduleIdentityOptionalCommit) isModuleIdentity()               {}
func (*moduleIdentityOptionalCommit) isModuleIdentityOptionalCommit() {}

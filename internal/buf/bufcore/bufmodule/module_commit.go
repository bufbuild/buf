// Copyright 2020-2021 Buf Technologies, Inc.
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

package bufmodule

type moduleCommit struct {
	remote     string
	owner      string
	repository string
	commit     string
}

func newModuleCommit(
	remote string,
	owner string,
	repository string,
	commit string,
) (*moduleCommit, error) {
	if err := validateRemote(remote); err != nil {
		return nil, err
	}
	if err := ValidateOwner(owner, "owner"); err != nil {
		return nil, err
	}
	if err := ValidateRepository(repository); err != nil {
		return nil, err
	}
	if err := ValidateCommit(commit); err != nil {
		return nil, err
	}
	return &moduleCommit{
		remote:     remote,
		owner:      owner,
		repository: repository,
		commit:     commit,
	}, nil
}

func (m *moduleCommit) Remote() string {
	return m.remote
}

func (m *moduleCommit) Owner() string {
	return m.owner
}

func (m *moduleCommit) Repository() string {
	return m.repository
}

func (m *moduleCommit) Commit() string {
	return m.commit
}

func (m *moduleCommit) String() string {
	return m.remote + "/" + m.owner + "/" + m.repository + ":" + m.commit
}

func (m *moduleCommit) IdentityString() string {
	return m.remote + "/" + m.owner + "/" + m.repository
}

func (*moduleCommit) isModuleOwner()    {}
func (*moduleCommit) isModuleIdentity() {}
func (*moduleCommit) isModuleCommit()   {}

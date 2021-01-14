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

import "time"

// externalLockFile represents the buf.lock configuration file.
type externalLockFile struct {
	Deps []*externalLockFileDep `json:"deps" yaml:"deps"`
}

// modulePins expected to be sorted and unique
func newExternalLockFile(modulePins []ModulePin) *externalLockFile {
	deps := make([]*externalLockFileDep, len(modulePins))
	for i, modulePin := range modulePins {
		deps[i] = newExternalLockFileDep(modulePin)
	}
	return &externalLockFile{
		Deps: deps,
	}
}

type externalLockFileDep struct {
	Remote     string    `json:"remote,omitempty" yaml:"remote,omitempty"`
	Owner      string    `json:"owner,omitempty" yaml:"owner,omitempty"`
	Repository string    `json:"repository,omitempty" yaml:"repository,omitempty"`
	Branch     string    `json:"branch,omitempty" yaml:"branch,omitempty"`
	Commit     string    `json:"commit,omitempty" yaml:"commit,omitempty"`
	Digest     string    `json:"digest,omitempty" yaml:"digest,omitempty"`
	CreateTime time.Time `json:"create_time,omitempty" yaml:"create_time,omitempty"`
}

func newExternalLockFileDep(modulePin ModulePin) *externalLockFileDep {
	return &externalLockFileDep{
		Remote:     modulePin.Remote(),
		Owner:      modulePin.Owner(),
		Repository: modulePin.Repository(),
		Branch:     modulePin.Branch(),
		Commit:     modulePin.Commit(),
		Digest:     modulePin.Digest(),
		CreateTime: modulePin.CreateTime(),
	}
}

func modulePinsForExternalLockFile(externalLockFile *externalLockFile) ([]ModulePin, error) {
	modulePins := make([]ModulePin, len(externalLockFile.Deps))
	for i, dep := range externalLockFile.Deps {
		modulePin, err := NewModulePin(
			dep.Remote,
			dep.Owner,
			dep.Repository,
			dep.Branch,
			dep.Commit,
			dep.Digest,
			dep.CreateTime,
		)
		if err != nil {
			return nil, err
		}
		modulePins[i] = modulePin
	}
	// just to be safe
	SortModulePins(modulePins)
	if err := ValidateModulePinsUniqueByIdentity(modulePins); err != nil {
		return nil, err
	}
	return modulePins, nil
}

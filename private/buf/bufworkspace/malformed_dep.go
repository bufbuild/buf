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

package bufworkspace

import (
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

const (
	// MalformedDepTypeUndeclared says that the dep was a transitive remote dependency of the
	// workspace, but was not declared in the buf.yaml.
	MalformedDepTypeUndeclared MalformedDepType = iota + 1
	// MalformedDepTypeUnused says that teh dep was declared in the buf.yaml but was not used.
	MalformedDepTypeUnused
)

// MalformedDepType is the type of malformed dep.
type MalformedDepType int

// MalformedDep is a dep that was malformed in some way in the buf.yaml.
type MalformedDep interface {
	// ModuleFullName returns the full name of the malformed dep.
	//
	// Always present.
	ModuleFullName() bufmodule.ModuleFullName
	// Type is why this dep was malformed.
	Type() MalformedDepType

	isMalformedDep()
}

// MalformedDepsForWorkspace gets the MalformedDeps for the workspace.
func MalformedDepsForWorkspace(workspace Workspace) ([]MalformedDep, error) {
	var malformedDeps []MalformedDep
	remoteDeps, err := bufmodule.RemoteDepsForModuleSet(workspace)
	if err != nil {
		return nil, err
	}
	for _, remoteDep := range remoteDeps {
		if !remoteDep.IsDirect() {
			moduleFullName := remoteDep.ModuleFullName()
			if moduleFullName == nil {
				return nil, syserror.Newf("ModuleFullName nil on remote Module dependency %q", remoteDep.OpaqueID())
			}
			malformedDeps = append(malformedDeps, newMalformedDep(moduleFullName, MalformedDepTypeUndeclared))
		}
	}
	moduleFullNameStringToRemoteDep, err := slicesext.ToUniqueValuesMapError(
		remoteDeps,
		func(remoteDep bufmodule.RemoteDep) (string, error) {
			moduleFullName := remoteDep.ModuleFullName()
			if moduleFullName == nil {
				return "", syserror.Newf("ModuleFullName nil on remote Module dependency %q", remoteDep.OpaqueID())
			}
			return moduleFullName.String(), nil
		},
	)
	if err != nil {
		return nil, err
	}
	configuredModuleFullNames, err := slicesext.MapError(
		workspace.ConfiguredDepModuleRefs(),
		func(moduleRef bufmodule.ModuleRef) (bufmodule.ModuleFullName, error) {
			moduleFullName := moduleRef.ModuleFullName()
			if moduleFullName == nil {
				return nil, syserror.New("ModuleFullName nil on ModuleRef")
			}
			return moduleFullName, nil
		},
	)
	if err != nil {
		return nil, err
	}
	for _, configuredModuleFullName := range configuredModuleFullNames {
		if _, ok := moduleFullNameStringToRemoteDep[configuredModuleFullName.String()]; !ok {
			malformedDeps = append(malformedDeps, newMalformedDep(configuredModuleFullName, MalformedDepTypeUnused))
		}
	}
	return malformedDeps, nil
}

// *** PRIVATE ***

type malformedDep struct {
	moduleFullName   bufmodule.ModuleFullName
	malformedDepType MalformedDepType
}

func newMalformedDep(moduleFullName bufmodule.ModuleFullName, malformedDepType MalformedDepType) *malformedDep {
	return &malformedDep{
		moduleFullName:   moduleFullName,
		malformedDepType: malformedDepType,
	}
}

func (m *malformedDep) ModuleFullName() bufmodule.ModuleFullName {
	return m.moduleFullName
}

func (m *malformedDep) Type() MalformedDepType {
	return m.malformedDepType
}

func (*malformedDep) isMalformedDep() {}

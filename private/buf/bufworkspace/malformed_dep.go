// Copyright 2020-2025 Buf Technologies, Inc.
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
	"sort"

	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

const (
	// MalformedDepTypeUnused says that the dep was declared in the buf.yaml but was not used.
	//
	// A dep is not used if no .proto file references it, and the dep is not a local Module within the Workspace.
	//
	// We ignore local Modules within the Workspace as v1 buf.yamls needed to declare deps within the Workspace,
	// and there's no easy way for us to determine if a dep is needed or not within our current
	// Workspace/Module model. We could get more complicated and warn if you are using a v2 buf.lock
	// and have deps on local Modules, but there's little benefit.
	MalformedDepTypeUnused MalformedDepType = iota + 1
)

// MalformedDepType is the type of malformed dep.
type MalformedDepType int

// MalformedDep is a dep that was malformed in some way in the buf.yaml.
// It provides the module ref information and the malformed dep type.
type MalformedDep interface {
	// ModuleRef is the module ref information of the malformed dep.
	//
	// Always present.
	ModuleRef() bufparse.Ref
	// Type is why this dep was malformed.
	//
	// Always present.
	Type() MalformedDepType

	isMalformedDep()
}

// MalformedDepsForWorkspace gets the MalformedDeps for the workspace.
func MalformedDepsForWorkspace(workspace Workspace) ([]MalformedDep, error) {
	localFullNameStringMap := xslices.ToStructMapOmitEmpty(
		xslices.Map(
			bufmodule.ModuleSetLocalModules(workspace),
			func(module bufmodule.Module) string {
				if moduleFullName := module.FullName(); moduleFullName != nil {
					return moduleFullName.String()
				}
				return ""
			},
		),
	)
	remoteDeps, err := bufmodule.RemoteDepsForModuleSet(workspace)
	if err != nil {
		return nil, err
	}
	moduleFullNameStringToRemoteDep, err := xslices.ToUniqueValuesMapError(
		remoteDeps,
		func(remoteDep bufmodule.RemoteDep) (string, error) {
			moduleFullName := remoteDep.FullName()
			if moduleFullName == nil {
				return "", syserror.Newf("FullName nil on remote Module dependency %q", remoteDep.OpaqueID())
			}
			return moduleFullName.String(), nil
		},
	)
	if err != nil {
		return nil, err
	}
	moduleFullNameStringToConfiguredDepModuleRef, err := xslices.ToUniqueValuesMapError(
		workspace.ConfiguredDepModuleRefs(),
		func(moduleRef bufparse.Ref) (string, error) {
			moduleFullName := moduleRef.FullName()
			if moduleFullName == nil {
				return "", syserror.New("FullName nil on ModuleRef")
			}
			return moduleFullName.String(), nil
		},
	)
	if err != nil {
		return nil, err
	}
	var malformedDeps []MalformedDep
	for moduleFullNameString, configuredDepModuleRef := range moduleFullNameStringToConfiguredDepModuleRef {
		_, isLocalModule := localFullNameStringMap[moduleFullNameString]
		_, isRemoteDep := moduleFullNameStringToRemoteDep[moduleFullNameString]
		if !isRemoteDep && !isLocalModule {
			// The module was in buf.yaml deps, but was not in the remote dep list after
			// adding all ModuleKeys and transitive dependency ModuleKeys. It is also not
			// a local module. Therefore it is unused.
			malformedDeps = append(
				malformedDeps,
				newMalformedDep(
					configuredDepModuleRef,
					MalformedDepTypeUnused,
				),
			)
		}
	}
	sort.Slice(
		malformedDeps,
		func(i int, j int) bool {
			return malformedDeps[i].ModuleRef().FullName().String() <
				malformedDeps[j].ModuleRef().FullName().String()
		},
	)
	return malformedDeps, nil
}

// *** PRIVATE ***

type malformedDep struct {
	moduleRef        bufparse.Ref
	malformedDepType MalformedDepType
}

func newMalformedDep(
	moduleRef bufparse.Ref,
	malformedDepType MalformedDepType,
) *malformedDep {
	return &malformedDep{
		moduleRef:        moduleRef,
		malformedDepType: malformedDepType,
	}
}

func (m *malformedDep) ModuleRef() bufparse.Ref {
	return m.moduleRef
}

func (m *malformedDep) Type() MalformedDepType {
	return m.malformedDepType
}

func (*malformedDep) isMalformedDep() {}

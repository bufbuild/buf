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

package bufmodule

import (
	"sort"

	"github.com/bufbuild/buf/private/pkg/syserror"
)

// RemoteDep is a remote dependency of some local Module in a ModuleSet.
//
// This is different than ModuleDep in that it doesn't specify the parent Module.
// There could be multiple modules that are the parents of a given RemoteDep.
//
// We don't care about targeting here - we want to know the remote dependencies for
// purposes such as figuring out what dependencies are unused and can be pruned.
type RemoteDep interface {
	// All RemoteDeps will have a ModuleFullName, as they are remote.
	Module

	// IsDirect returns true if the remote dependency is a direct dependency of a Module in the ModuleSet.
	IsDirect() bool

	isRemoteDep()
}

// RemoteDepsForModuleSet returns the remote dependencies of the local Modules in the ModuleSet.
//
// Sorted by ModuleFullName.
//
// TODO FUTURE: This needs a LOT of testing.
func RemoteDepsForModuleSet(moduleSet ModuleSet) ([]RemoteDep, error) {
	return RemoteDepsForModules(moduleSet.Modules())
}

// RemoteDepsForModules returns the remote dependencies of the local Modules.
//
// Sorted by ModuleFullName.
//
// This is used in situations where we have already filtered a ModuleSet down to a specific
// set of modules, such as in the Uploader. Generally, you want to use RemoteDepsForModuleSet.
//
// This function may validate that all Modules are from the same ModuleSet, although it
// currently does not.
//
// TODO FUTURE: This needs a LOT of testing.
func RemoteDepsForModules(modules []Module) ([]RemoteDep, error) {
	visitedOpaqueIDs := make(map[string]struct{})
	remoteDepModuleFullNameStringsThatAreDirectDepsOfLocal := make(map[string]struct{})
	var remoteDepModules []Module
	for _, module := range modules {
		if !module.IsLocal() {
			continue
		}
		moduleDeps, err := module.ModuleDeps()
		if err != nil {
			return nil, err
		}
		for _, moduleDep := range moduleDeps {
			if moduleDep.IsLocal() {
				continue
			}
			moduleDepFullName := moduleDep.ModuleFullName()
			if moduleDepFullName == nil {
				// Just a sanity check.
				return nil, syserror.New("remote module did not have a ModuleFullName")
			}
			if moduleDep.IsDirect() {
				remoteDepModuleFullNameStringsThatAreDirectDepsOfLocal[moduleDepFullName.String()] = struct{}{}
			}
			iRemoteDepModules, err := remoteDepsForModuleSetRec(
				moduleDep,
				visitedOpaqueIDs,
			)
			if err != nil {
				return nil, err
			}
			remoteDepModules = append(remoteDepModules, iRemoteDepModules...)
		}
	}
	remoteDeps := make([]RemoteDep, len(remoteDepModules))
	for i, remoteDepModule := range remoteDepModules {
		moduleFullName := remoteDepModule.ModuleFullName()
		if moduleFullName == nil {
			// Just a sanity check.
			return nil, syserror.New("remote module did not have a ModuleFullName")
		}
		_, isDirect := remoteDepModuleFullNameStringsThatAreDirectDepsOfLocal[moduleFullName.String()]
		remoteDeps[i] = newRemoteDep(remoteDepModule, isDirect)
	}
	sort.Slice(
		remoteDeps,
		func(i int, j int) bool {
			return remoteDeps[i].OpaqueID() < remoteDeps[j].OpaqueID()
		},
	)
	return remoteDeps, nil
}

// *** PRIVATE ***

type remoteDep struct {
	Module

	isDirect bool
}

func newRemoteDep(module Module, isDirect bool) *remoteDep {
	return &remoteDep{
		Module:   module,
		isDirect: isDirect,
	}
}

func (l *remoteDep) IsDirect() bool {
	return l.isDirect
}

func (*remoteDep) isRemoteDep() {}

func remoteDepsForModuleSetRec(
	remoteModule Module,
	visitedOpaqueIDs map[string]struct{},
) ([]Module, error) {
	if remoteModule.IsLocal() {
		return nil, syserror.New("only pass remote modules to remoteDepsForModuleSetRec")
	}
	if remoteModule.ModuleFullName() == nil {
		// Just a sanity check.
		return nil, syserror.New("ModuleFullName is nil for a remote Module")
	}
	opaqueID := remoteModule.OpaqueID()
	if _, ok := visitedOpaqueIDs[opaqueID]; ok {
		return nil, nil
	}
	visitedOpaqueIDs[opaqueID] = struct{}{}
	recModuleDeps, err := remoteModule.ModuleDeps()
	if err != nil {
		return nil, err
	}
	recDeps := make([]Module, 0, len(recModuleDeps)+1)
	recDeps = append(recDeps, remoteModule)
	for _, recModuleDep := range recModuleDeps {
		if recModuleDep.IsLocal() {
			continue
		}
		// We deal with local vs remote in the recursive call.
		iRecDeps, err := remoteDepsForModuleSetRec(
			recModuleDep,
			visitedOpaqueIDs,
		)
		if err != nil {
			return nil, err
		}
		recDeps = append(recDeps, iRecDeps...)
	}
	return recDeps, nil
}

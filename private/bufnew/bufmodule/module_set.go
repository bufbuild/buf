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

package bufmodule

import (
	"fmt"
	"sync"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/pkg/dag"
)

// ModuleSet is a set of Modules constructed by a ModuleBuilder.
type ModuleSet interface {
	// Modules returns the target Modules in the ModuleSet.
	//
	// Modules are either targets or non-targets.
	// A target Module is a module that we are directly targeting for operations.
	// All non-target Modules are dependencies of targets. Both targets and non-targets
	// can be retrieved via the Get.* functions.
	//
	// These will be sorted by OpaqueID.
	TargetModules() []Module
	// Modules returns the non-target Modules in the ModuleSet.
	//
	// Modules are either targets or non-targets.
	// A target Module is a module that we are directly targeting for operations.
	// All non-target Modules are dependencies of targets. Both targets and non-targets
	// can be retrieved via the Get.* functions.
	//
	// These will be sorted by OpaqueID.
	NonTargetModules() []Module

	// GetModuleForModuleFullName gets the Module for the ModuleFullName, if it exists.
	//
	// Returns nil if there is no Module with the given ModuleFullName.
	GetModuleForModuleFullName(moduleFullName ModuleFullName) Module
	// GetModuleForOpaqueID gets the Module for the OpaqueID, if it exists.
	//
	// Returns nil if there is no Module with the given OpaqueID.
	GetModuleForOpaqueID(opaqueID string) Module
	// GetModuleForBucketID gets the MOdule for the BucketID, if it exists.
	//
	// Returns nil if there is no Module with the given BucketID.
	GetModuleForBucketID(bucketID string) Module
	// GetModuleForDigest gets the MOdule for the Digest, if it exists.
	//
	// Note that this function will result in Digest() being called on every Module in
	// the ModuleSet, which is potentially expensive.
	//
	// Returns nil if there is no Module with the given Digest.
	// Returns an error if there was an error when calling Digest() on a Module.
	GetModuleForDigest(digest bufcas.Digest) (Module, error)

	isModuleSet()
}

// GetModuleSetOpaqueIDDAG gets a DAG of the OpaqueIDs of the given ModuleSet.
func GetModuleSetOpaqueIDDAG(moduleSet ModuleSet) (*dag.Graph[string], error) {
	graph := dag.NewGraph[string]()
	for _, module := range moduleSet.TargetModules() {
		if err := buildModuleOpaqueIDDAGRec(module, graph); err != nil {
			return nil, err
		}
	}
	return graph, nil
}

// *** PRIVATE ***

func buildModuleOpaqueIDDAGRec(
	module Module,
	graph *dag.Graph[string],
) error {
	graph.AddNode(module.OpaqueID())
	moduleDeps, err := module.ModuleDeps()
	if err != nil {
		return err
	}
	for _, moduleDep := range moduleDeps {
		if moduleDep.IsDirect() {
			graph.AddNode(moduleDep.OpaqueID())
			graph.AddEdge(module.OpaqueID(), moduleDep.OpaqueID())
			if err := buildModuleOpaqueIDDAGRec(moduleDep, graph); err != nil {
				return err
			}
		}
	}
	return nil
}

// moduleSet

type moduleSet struct {
	targetModules                []Module
	nonTargetModules             []Module
	moduleFullNameStringToModule map[string]Module
	opaqueIDToModule             map[string]Module
	bucketIDToModule             map[string]Module
	getDigestStringToModule      func() (map[string]Module, error)
}

func newModuleSet(
	moduleSetModules []*moduleSetModule,
) (*moduleSet, error) {
	targetModules := make([]Module, 0, len(moduleSetModules))
	nonTargetModules := make([]Module, 0, len(moduleSetModules))
	moduleFullNameStringToModule := make(map[string]Module, len(moduleSetModules))
	opaqueIDToModule := make(map[string]Module, len(moduleSetModules))
	bucketIDToModule := make(map[string]Module, len(moduleSetModules))
	for _, module := range moduleSetModules {
		if module.isTarget() {
			targetModules = append(targetModules, module)
		} else {
			nonTargetModules = append(nonTargetModules, module)
		}
		if moduleFullName := module.ModuleFullName(); moduleFullName != nil {
			moduleFullNameString := moduleFullName.String()
			if _, ok := moduleFullNameStringToModule[moduleFullNameString]; ok {
				// This should never happen.
				return nil, fmt.Errorf("duplicate ModuleFullName %q when constructing ModuleSet", moduleFullNameString)
			}
			moduleFullNameStringToModule[moduleFullNameString] = module
		}
		opaqueID := module.OpaqueID()
		if _, ok := opaqueIDToModule[opaqueID]; ok {
			// This should never happen.
			return nil, fmt.Errorf("duplicate OpaqueID %q when constructing ModuleSet", opaqueID)
		}
		opaqueIDToModule[opaqueID] = module
		bucketID := module.BucketID()
		if bucketID != "" {
			if _, ok := bucketIDToModule[bucketID]; ok {
				// This should never happen.
				return nil, fmt.Errorf("duplicate BucketID %q when constructing ModuleSet", bucketID)
			}
			bucketIDToModule[bucketID] = module
		}
	}
	return &moduleSet{
		targetModules:                targetModules,
		nonTargetModules:             nonTargetModules,
		moduleFullNameStringToModule: moduleFullNameStringToModule,
		opaqueIDToModule:             opaqueIDToModule,
		bucketIDToModule:             bucketIDToModule,
		getDigestStringToModule: sync.OnceValues(
			func() (map[string]Module, error) {
				digestStringToModule := make(map[string]Module, len(moduleSetModules))
				for _, module := range moduleSetModules {
					digest, err := module.Digest()
					if err != nil {
						return nil, err
					}
					digestString := digest.String()
					if _, ok := digestStringToModule[digestString]; ok {
						// Note that because we do this lazily, we're not getting built-in validation here
						// that a ModuleSet has unique Digests until we load them lazily. That's the best
						// we can do, and is likely OK.
						return nil, fmt.Errorf("duplicate Digest %q within ModuleSet", digestString)
					}
					digestStringToModule[digestString] = module
				}
				return digestStringToModule, nil
			},
		),
	}, nil
}

func (m *moduleSet) TargetModules() []Module {
	c := make([]Module, len(m.targetModules))
	copy(c, m.targetModules)
	return c
}

func (m *moduleSet) NonTargetModules() []Module {
	c := make([]Module, len(m.nonTargetModules))
	copy(c, m.nonTargetModules)
	return c
}

func (m *moduleSet) GetModuleForModuleFullName(moduleFullName ModuleFullName) Module {
	return m.moduleFullNameStringToModule[moduleFullName.String()]
}

func (m *moduleSet) GetModuleForOpaqueID(opaqueID string) Module {
	return m.opaqueIDToModule[opaqueID]
}

func (m *moduleSet) GetModuleForBucketID(bucketID string) Module {
	return m.bucketIDToModule[bucketID]
}

func (m *moduleSet) GetModuleForDigest(digest bufcas.Digest) (Module, error) {
	digestStringToModule, err := m.getDigestStringToModule()
	if err != nil {
		return nil, err
	}
	return digestStringToModule[digest.String()], nil
}

func (*moduleSet) isModuleSet() {}

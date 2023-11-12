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
	"context"
	"fmt"
	"sync"

	"github.com/bufbuild/buf/private/bufnew/bufmodule/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/pkg/dag"
	"github.com/bufbuild/buf/private/pkg/slicesextended"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/protocompile/parser/imports"
	"go.uber.org/multierr"
)

// ModuleSet is a set of Modules constructed by a ModuleBuilder.
type ModuleSet interface {
	// Modules returns the Modules in the ModuleSet.
	//
	// This will consist of both targets and non-targets.
	//
	// These will be sorted by OpaqueID.
	Modules() []Module

	// GetModuleForModuleFullName gets the Module for the ModuleFullName, if it exists.
	//
	// Returns nil if there is no Module with the given ModuleFullName.
	GetModuleForModuleFullName(moduleFullName ModuleFullName) Module
	// GetModuleForOpaqueID gets the Module for the OpaqueID, if it exists.
	//
	// Returns nil if there is no Module with the given OpaqueID. However, as long
	// as the OpaqueID came from a Module contained within Modules(), this will always
	// return a non-nil value.
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

	// getModuleForFilePath gets the Module for the File path of a File within the ModuleSet.
	//
	// This should only be used by Modules, and only for dependency calculations.
	getModuleForFilePath(ctx context.Context, filePath string) (Module, error)
	// getModuleForFilePath gets the imports for the File path of a File within the ModuleSet.
	//
	// This should only be used by Modules, and only for dependency calculations.
	getImportsForFilePath(ctx context.Context, filePath string) (map[string]struct{}, error)
	isModuleSet()
}

// ModuleSetToModuleReadBucketWithOnlyProtoFiles converts the ModuleSet to a
// ModuleReadBucket that contains all the .proto files of the target and non-target
// Modules of the ModuleSet.
//
// Targeting information will remain the same.
func ModuleSetToModuleReadBucketWithOnlyProtoFiles(moduleSet ModuleSet) ModuleReadBucket {
	return newMultiModuleReadBucket(
		slicesextended.Map(
			moduleSet.Modules(),
			func(module Module) ModuleReadBucket {
				return ModuleReadBucketWithOnlyProtoFiles(module)
			},
		),
	)
}

// ModuleSetTargetModules is a convenience function that returns the target Modules
// from a ModuleSet.
func ModuleSetTargetModules(moduleSet ModuleSet) []Module {
	return slicesextended.Filter(
		moduleSet.Modules(),
		func(module Module) bool { return module.IsTarget() },
	)
}

// ModuleSetOpaqueIDs is a conenience function that returns a slice of the OpaqueIDs of the
// Modules in the ModuleSet.
//
// Sorted.
func ModuleSetOpaqueIDs(moduleSet ModuleSet) []string {
	return slicesextended.Map(
		moduleSet.Modules(),
		func(module Module) string { return module.OpaqueID() },
	)
}

// ModuleSetTargetOpaqueIDs is a conenience function that returns a slice of the OpaqueIDs of the
// target Modules in the ModuleSet.
//
// Sorted.
func ModuleSetTargetOpaqueIDs(moduleSet ModuleSet) []string {
	return slicesextended.Map(
		ModuleSetTargetModules(moduleSet),
		func(module Module) string { return module.OpaqueID() },
	)
}

// ModuleSetToDAG gets a DAG of the OpaqueIDs of the given ModuleSet.
//
// This only starts at target Modules. If a Module is not part of a graph
// with a target Module as a source, it will not be added.
func ModuleSetToDAG(moduleSet ModuleSet) (*dag.Graph[string], error) {
	graph := dag.NewGraph[string]()
	for _, module := range ModuleSetTargetModules(moduleSet) {
		if err := moduleSetToDAGRec(module, graph); err != nil {
			return nil, err
		}
	}
	return graph, nil
}

// *** PRIVATE ***

func moduleSetToDAGRec(
	module Module,
	graph *dag.Graph[string],
) error {
	graph.AddNode(module.OpaqueID())
	directModuleDeps, err := ModuleDirectModuleDeps(module)
	if err != nil {
		return err
	}
	for _, directModuleDep := range directModuleDeps {
		graph.AddEdge(module.OpaqueID(), directModuleDep.OpaqueID())
		if err := moduleSetToDAGRec(directModuleDep, graph); err != nil {
			return err
		}
	}
	return nil
}

// moduleSet

type moduleSet struct {
	modules                      []Module
	moduleFullNameStringToModule map[string]Module
	opaqueIDToModule             map[string]Module
	bucketIDToModule             map[string]Module
	getDigestStringToModule      func() (map[string]Module, error)

	// filePathToImports is a cache of filePath -> imports, used for calculating dependencies.
	filePathToImports map[string]*internal.Tuple[map[string]struct{}, error]
	// filePathToModule is a cache of filePath -> module, used for calculating dependencies.
	filePathToModule map[string]*internal.Tuple[Module, error]
	// cacheLock is used to lock access to filePathToImports and filePathToModule.
	//
	// We could have a per-map lock but then we need to deal with lock ordering, not worth it for now.
	cacheLock sync.RWMutex
}

func newModuleSet(
	modules []Module,
) (*moduleSet, error) {
	moduleFullNameStringToModule := make(map[string]Module, len(modules))
	opaqueIDToModule := make(map[string]Module, len(modules))
	bucketIDToModule := make(map[string]Module, len(modules))
	for _, module := range modules {
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
		modules:                      modules,
		moduleFullNameStringToModule: moduleFullNameStringToModule,
		opaqueIDToModule:             opaqueIDToModule,
		bucketIDToModule:             bucketIDToModule,
		getDigestStringToModule: sync.OnceValues(
			func() (map[string]Module, error) {
				digestStringToModule := make(map[string]Module, len(modules))
				for _, module := range modules {
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
		filePathToImports: make(map[string]*internal.Tuple[map[string]struct{}, error]),
		filePathToModule:  make(map[string]*internal.Tuple[Module, error]),
	}, nil
}

func (m *moduleSet) Modules() []Module {
	c := make([]Module, len(m.modules))
	copy(c, m.modules)
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

// This should only be used by Modules, and only for dependency calculations.
func (m *moduleSet) getModuleForFilePath(ctx context.Context, filePath string) (Module, error) {
	return internal.GetOrAddToCacheDoubleLock(
		&m.cacheLock,
		m.filePathToModule,
		filePath,
		func() (Module, error) {
			return m.getModuleForFilePathUncached(ctx, filePath)
		},
	)
}

// This should only be used by Modules, and only for dependency calculations.
func (m *moduleSet) getImportsForFilePath(ctx context.Context, filePath string) (map[string]struct{}, error) {
	return internal.GetOrAddToCacheDoubleLock(
		&m.cacheLock,
		m.filePathToImports,
		filePath,
		func() (map[string]struct{}, error) {
			return m.getImportsForFilePathUncached(ctx, filePath)
		},
	)
}

// Assumed to be called within cacheLock.
// Only call from within *moduleSet.
func (m *moduleSet) getModuleForFilePathUncached(ctx context.Context, filePath string) (Module, error) {
	matchingOpaqueIDs := make(map[string]struct{})
	// Note that we're effectively doing an O(num_modules * num_files) operation here, which could be prohibitive.
	for _, module := range m.Modules() {
		if _, err := module.StatFileInfo(ctx, filePath); err == nil {
			matchingOpaqueIDs[module.OpaqueID()] = struct{}{}
		}
	}
	switch len(matchingOpaqueIDs) {
	case 0:
		// This should likely never happen given how we call the cache.
		return nil, fmt.Errorf("no Module contains file %q", filePath)
	case 1:
		var matchingOpaqueID string
		for matchingOpaqueID = range matchingOpaqueIDs {
		}
		return m.GetModuleForOpaqueID(matchingOpaqueID), nil
	default:
		// This actually could happen, and we will want to make this error message as clear as possible.
		// The addition of opaqueID should give us clearer error messages than we have today.
		return nil, fmt.Errorf("multiple Modules contained file %q: %v", filePath, stringutil.MapToSortedSlice(matchingOpaqueIDs))
	}
}

// Assumed to be called within cacheLock.
// Only call from within *moduleSet.
func (m *moduleSet) getImportsForFilePathUncached(ctx context.Context, filePath string) (_ map[string]struct{}, retErr error) {
	// Even when we know the file we want to get the imports for, we want to make sure the file
	// is not duplicated across multiple modules. By calling getModuleFileFilePathUncached,
	// we implicitly get this check for now.
	//
	// Note this basically kills the idea of only partially-lazily-loading some of the Modules
	// within a set of []Modules. We could optimize this later, and may want to. This means
	// that we're going to have to load all the modules within a workspace even if just building
	// a single module in the workspace, as an example. Luckily, modules within workspaces are
	// the cheapest to load (ie not remote).
	module, err := internal.GetOrAddToCache(
		m.filePathToModule,
		filePath,
		func() (Module, error) {
			return m.getModuleForFilePathUncached(ctx, filePath)
		},
	)
	if err != nil {
		return nil, err
	}
	file, err := module.GetFile(ctx, filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, file.Close())
	}()
	imports, err := imports.ScanForImports(file)
	if err != nil {
		return nil, err
	}
	return stringutil.SliceToMap(imports), nil
}

func (*moduleSet) isModuleSet() {}

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
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"sync"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/gen/data/datawkt"
	"github.com/bufbuild/buf/private/pkg/cache"
	"github.com/bufbuild/buf/private/pkg/dag"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/protocompile/parser/fastscan"
	"go.uber.org/multierr"
)

// errIsWKT is the error returned by getFastscanResultForFilePath or getModuleForFilePath if the
// input filePath is a well-known type.
var errIsWKT = errors.New("wkt")

// ModuleSet is a set of Modules constructed by a ModuleBuilder.
//
// A ModuleSet is expected to be self-contained, that is Modules only import
// from other Modules in this ModuleSet.
type ModuleSet interface {
	// Modules returns the Modules in the ModuleSet.
	//
	// This will consist of both targets and non-targets.
	// All dependencies of all Modules will be in this list, that is this list is self-contained.
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
	//
	// returns errIsWKT if the filePath is a WKT.
	// returns an error with fs.ErrNotExist if the file is not found.
	getModuleForFilePath(ctx context.Context, filePath string) (Module, error)
	// getModuleForFilePath gets the fastscan.Result for the File path of a File within the ModuleSet.
	//
	// This should only be used by Modules and FileInfos.
	//
	// returns errIsWKT if the filePath is a WKT.
	// returns an error with fs.ErrNotExist if the file is not found.
	getFastscanResultForFilePath(ctx context.Context, filePath string) (fastscan.Result, error)
	isModuleSet()
}

// ModuleSetToModuleReadBucketWithOnlyProtoFiles converts the ModuleSet to a
// ModuleReadBucket that contains all the .proto files of the target and non-target
// Modules of the ModuleSet.
//
// Targeting information will remain the same.
func ModuleSetToModuleReadBucketWithOnlyProtoFiles(moduleSet ModuleSet) ModuleReadBucket {
	return newMultiModuleReadBucket(
		slicesext.Map(
			moduleSet.Modules(),
			func(module Module) ModuleReadBucket {
				return ModuleReadBucketWithOnlyProtoFiles(module)
			},
		),
		true,
	)
}

// ModuleSetTargetModules is a convenience function that returns the target Modules
// from a ModuleSet.
func ModuleSetTargetModules(moduleSet ModuleSet) []Module {
	return slicesext.Filter(
		moduleSet.Modules(),
		func(module Module) bool { return module.IsTarget() },
	)
}

// ModuleSetNonTargetModules is a convenience function that returns the non-target Modules
// from a ModuleSet.
func ModuleSetNonTargetModules(moduleSet ModuleSet) []Module {
	return slicesext.Filter(
		moduleSet.Modules(),
		func(module Module) bool { return !module.IsTarget() },
	)
}

// ModuleSetLocalModules is a convenience function that returns the local Modules
// from a ModuleSet.
func ModuleSetLocalModules(moduleSet ModuleSet) []Module {
	return slicesext.Filter(
		moduleSet.Modules(),
		func(module Module) bool { return module.IsLocal() },
	)
}

// ModuleSetRemoteModules is a convenience function that returns the remote Modules
// from a ModuleSet.
func ModuleSetRemoteModules(moduleSet ModuleSet) []Module {
	return slicesext.Filter(
		moduleSet.Modules(),
		func(module Module) bool { return !module.IsLocal() },
	)
}

// ModuleSetOpaqueIDs is a conenience function that returns a slice of the OpaqueIDs of the
// Modules in the ModuleSet.
//
// Sorted.
func ModuleSetOpaqueIDs(moduleSet ModuleSet) []string {
	return modulesOpaqueIDs(moduleSet.Modules())
}

// ModuleSetTargetOpaqueIDs is a conenience function that returns a slice of the OpaqueIDs of the
// target Modules in the ModuleSet.
//
// Sorted.
func ModuleSetTargetOpaqueIDs(moduleSet ModuleSet) []string {
	return modulesOpaqueIDs(ModuleSetTargetModules(moduleSet))
}

// ModuleSetNonTargetOpaqueIDs is a conenience function that returns a slice of the OpaqueIDs of the
// non-target Modules in the ModuleSet.
//
// Sorted.
func ModuleSetNonTargetOpaqueIDs(moduleSet ModuleSet) []string {
	return modulesOpaqueIDs(ModuleSetNonTargetModules(moduleSet))
}

// ModuleSetLocalOpaqueIDs is a conenience function that returns a slice of the OpaqueIDs of the
// local Modules in the ModuleSet.
//
// Sorted.
func ModuleSetLocalOpaqueIDs(moduleSet ModuleSet) []string {
	return modulesOpaqueIDs(ModuleSetLocalModules(moduleSet))
}

// ModuleSetRemoteOpaqueIDs is a conenience function that returns a slice of the OpaqueIDs of the
// remote Modules in the ModuleSet.
//
// Sorted.
func ModuleSetRemoteOpaqueIDs(moduleSet ModuleSet) []string {
	return modulesOpaqueIDs(ModuleSetRemoteModules(moduleSet))
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

// ModuleSetRemoteDepsOfLocalModules is a convenience function that returns the remote dependencies
// of the local Modules in the ModuleSet.
//
// We don't care about targeting here - we want to know the remote dependencies for
// purposes such as figuring out what dependencies are unused and can be pruned.
//
// Returns Modules instead of ModuleDeps as IsDirect() has no meaning in this context,
// and there could be multiple parents for a given ModuleDep. Technically, could determine
// if a Module is a direct dependency of the ModuleSet, but this is not what IsDirect() means currently.
//
// Optionally allows filtering for only transitive remote dependencies via WithOnlyTransitiveRemoteDeps().
//
// All returned Modules will have a ModuleFullName, as they are remote.
// All returned Modules will be unique.
//
// Sorted by ModuleFullName.
//
// TODO: This needs a LOT of testing.
func ModuleSetRemoteDepsOfLocalModules(
	moduleSet ModuleSet,
	options ...ModuleSetRemoteDepsOfLocalModulesOption,
) ([]Module, error) {
	moduleSetRemoteDepsOfLocalModuleOptions := newModuleSetRemoteDepsOfLocalModuleOptions()
	for _, option := range options {
		option(moduleSetRemoteDepsOfLocalModuleOptions)
	}
	visitedOpaqueIDs := make(map[string]struct{})
	remoteDepModuleFullNameStringsThatAreDirectDepsOfLocal := make(map[string]struct{})
	var remoteDeps []Module
	for _, module := range moduleSet.Modules() {
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
			iRemoteDeps, err := moduleSetRemoteDepsRec(
				moduleDep,
				visitedOpaqueIDs,
			)
			if err != nil {
				return nil, err
			}
			remoteDeps = append(remoteDeps, iRemoteDeps...)
		}
	}
	if moduleSetRemoteDepsOfLocalModuleOptions.onlyTransitiveRemoteDeps {
		var resultRemoteDeps []Module
		for _, remoteDep := range remoteDeps {
			moduleFullName := remoteDep.ModuleFullName()
			if moduleFullName == nil {
				// Just a sanity check.
				return nil, syserror.New("remote module did not have a ModuleFullName")
			}
			if _, ok := remoteDepModuleFullNameStringsThatAreDirectDepsOfLocal[moduleFullName.String()]; !ok {
				resultRemoteDeps = append(resultRemoteDeps, remoteDep)
			}
		}
		remoteDeps = resultRemoteDeps
	}
	sort.Slice(
		remoteDeps,
		func(i int, j int) bool {
			return remoteDeps[i].OpaqueID() < remoteDeps[j].OpaqueID()
		},
	)
	return remoteDeps, nil
}

// ModuleSetRemoteDepsOfLocalModulesOption is an option for ModuleSetRemoteDepsOfLocalModules.
type ModuleSetRemoteDepsOfLocalModulesOption func(*moduleSetRemoteDepsOfLocalModuleOption)

// WithOnlyTransitiveRemoteDeps returns a new ModuleSetRemoteDepsOfLocalModulesOption that says
// to only return transitive remote dependencies.
func WithOnlyTransitiveRemoteDeps() ModuleSetRemoteDepsOfLocalModulesOption {
	return func(moduleSetRemoteDepsOfLocalModuleOption *moduleSetRemoteDepsOfLocalModuleOption) {
		moduleSetRemoteDepsOfLocalModuleOption.onlyTransitiveRemoteDeps = true
	}
}

// *** PRIVATE ***

// moduleSet

type moduleSet struct {
	modules                      []Module
	moduleFullNameStringToModule map[string]Module
	opaqueIDToModule             map[string]Module
	bucketIDToModule             map[string]Module
	getDigestStringToModule      func() (map[string]Module, error)

	// filePathToModule is a cache of filePath -> module.
	//
	// If you are calling both the imports and module caches, you must call the imports cache first,
	// i.e. lock ordering.
	filePathToModuleCache cache.Cache[string, Module]
	// filePathToFastscanResult is a cache of filePath -> fastscan.Result.
	filePathToFastscanResultCache cache.Cache[string, fastscan.Result]
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
				return nil, syserror.Newf("duplicate ModuleFullName %q when constructing ModuleSet", moduleFullNameString)
			}
			moduleFullNameStringToModule[moduleFullNameString] = module
		}
		opaqueID := module.OpaqueID()
		if _, ok := opaqueIDToModule[opaqueID]; ok {
			// This should never happen.
			return nil, syserror.Newf("duplicate OpaqueID %q when constructing ModuleSet", opaqueID)
		}
		opaqueIDToModule[opaqueID] = module
		bucketID := module.BucketID()
		if bucketID != "" {
			if _, ok := bucketIDToModule[bucketID]; ok {
				// This should never happen.
				return nil, syserror.Newf("duplicate BucketID %q when constructing ModuleSet", bucketID)
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

// This should only be used by Modules and FileInfos.
func (m *moduleSet) getModuleForFilePath(ctx context.Context, filePath string) (Module, error) {
	return m.filePathToModuleCache.GetOrAdd(
		filePath,
		func() (Module, error) {
			return m.getModuleForFilePathUncached(ctx, filePath)
		},
	)
}

// This should only be used by Modules and FileInfos.
func (m *moduleSet) getFastscanResultForFilePath(ctx context.Context, filePath string) (fastscan.Result, error) {
	return m.filePathToFastscanResultCache.GetOrAdd(
		filePath,
		func() (fastscan.Result, error) {
			return m.getFastscanResultForFilePathUncached(ctx, filePath)
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
		// TODO: This is likely a problem now as well, but we do not include WKTs in our
		// digest calculations. We should discuss whether this is important or not - we could
		// make an argument that it is not since WKTs are not downloaded in this usage.
		if datawkt.Exists(filePath) {
			return nil, errIsWKT
		}
		// This will happen if there is a file path we cannot find in our modules, which will result
		// in an error on ModuleDeps() or Digest().
		return nil, &fs.PathError{Op: "stat", Path: filePath, Err: fs.ErrNotExist}
	case 1:
		var matchingOpaqueID string
		for matchingOpaqueID = range matchingOpaqueIDs {
		}
		return m.GetModuleForOpaqueID(matchingOpaqueID), nil
	default:
		// This actually could happen, and we will want to make this error message as clear as possible.
		// The addition of opaqueID should give us clearer error messages than we have today.
		return nil, fmt.Errorf("%s is contained in multiple modules: %v", filePath, slicesext.MapKeysToSortedSlice(matchingOpaqueIDs))
	}
}

// Assumed to be called within cacheLock.
// Only call from within *moduleSet.
func (m *moduleSet) getFastscanResultForFilePathUncached(
	ctx context.Context,
	filePath string,
) (_ fastscan.Result, retErr error) {
	// Even when we know the file we want to get the imports for, we want to make sure the file
	// is not duplicated across multiple modules. By calling getModuleForFilePath,
	// we implicitly get this check for now.
	//
	// Note this basically kills the idea of only partially-lazily-loading some of the Modules
	// within a set of []Modules. We could optimize this later, and may want to. This means
	// that we're going to have to load all the modules within a workspace even if just building
	// a single module in the workspace, as an example. Luckily, modules within workspaces are
	// the cheapest to load (ie not remote).
	module, err := m.getModuleForFilePath(ctx, filePath)
	if err != nil {
		return fastscan.Result{}, err
	}
	file, err := module.GetFile(ctx, filePath)
	if err != nil {
		return fastscan.Result{}, err
	}
	defer func() {
		retErr = multierr.Append(retErr, file.Close())
	}()
	fastscanResult, err := fastscan.Scan(file)
	if err != nil {
		return fastscan.Result{}, fmt.Errorf("%s had parse error: %w", filePath, err)
	}
	return fastscanResult, nil
}

func (*moduleSet) isModuleSet() {}

// utils

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

func moduleSetRemoteDepsRec(
	remoteModule Module,
	visitedOpaqueIDs map[string]struct{},
) ([]Module, error) {
	if remoteModule.IsLocal() {
		return nil, syserror.New("only pass remote modules to moduleSetRemoteDepsRec")
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
		iRecDeps, err := moduleSetRemoteDepsRec(
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

func modulesOpaqueIDs(modules []Module) []string {
	return slicesext.Map(
		modules,
		func(module Module) string { return module.OpaqueID() },
	)
}

type moduleSetRemoteDepsOfLocalModuleOption struct {
	onlyTransitiveRemoteDeps bool
}

func newModuleSetRemoteDepsOfLocalModuleOptions() *moduleSetRemoteDepsOfLocalModuleOption {
	return &moduleSetRemoteDepsOfLocalModuleOption{}
}

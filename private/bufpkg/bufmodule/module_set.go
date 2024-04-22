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
	"context"
	"errors"
	"io/fs"
	"sort"

	"github.com/bufbuild/buf/private/pkg/cache"
	"github.com/bufbuild/buf/private/pkg/dag"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/gofrs/uuid/v5"
)

// ModuleSet is a set of Modules constructed by a ModuleBuilder.
//
// A ModuleSet is expected to be self-contained, that is Modules only import
// from other Modules in this ModuleSet.
//
// A ModuleSet may be empty and have no Modules. This is primarily done in testing.
type ModuleSet interface {
	// Modules returns the Modules in the ModuleSet.
	//
	// This will consist of both targets and non-targets.
	// All dependencies of all Modules will be in this list, that is this list is self-contained.
	//
	// All Modules will have unique Digests and CommitIDs.
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
	// GetModuleForBucketID gets the Module for the BucketID, if it exists.
	//
	// Returns nil if there is no Module with the given BucketID.
	GetModuleForBucketID(bucketID string) Module
	// GetModuleForCommitID gets the Module for the CommitID, if it exists.
	//
	// Returns nil if there is no Module with the given CommitID.
	GetModuleForCommitID(commitID uuid.UUID) Module

	// WithTargetOpaqueIDs returns a new ModuleSet that changes the targeted Modules to
	// the Modules with the specified OpaqueIDs.
	WithTargetOpaqueIDs(opaqueIDs ...string) (ModuleSet, error)

	// getModuleForFilePath gets the Module for the File path of a File within the ModuleSet.
	//
	// This should only be used by Modules, and only for dependency calculations. Any used of this outside
	// of getModuleDeps will result in nasty bugs down the line - we effectively ignore WKTs in this function,
	// but WKTs may be included in one of the Modules as files.
	//
	// returns an error with fs.ErrNotExist if the file is not found.
	getModuleForFilePath(ctx context.Context, filePath string) (Module, error)

	isModuleSet()
}

// ModuleSetToModuleReadBucketWithOnlyProtoFiles converts the ModuleSet to a
// ModuleReadBucket that contains all the .proto files of the target and non-target
// Modules of the ModuleSet.
//
// Targeting information will remain the same.
func ModuleSetToModuleReadBucketWithOnlyProtoFiles(moduleSet ModuleSet) ModuleReadBucket {
	return newMultiProtoFileModuleReadBucket(moduleSet.Modules(), true)
}

// ModuleSetToModuleReadBucketWithOnlyProtoFilesForTargetModules converts the ModuleSet to a
// ModuleReadBucket that contains all the .proto files of the target
// Modules of the ModuleSet.
//
// Targeting information will remain the same.
func ModuleSetToModuleReadBucketWithOnlyProtoFilesForTargetModules(moduleSet ModuleSet) ModuleReadBucket {
	return newMultiProtoFileModuleReadBucket(ModuleSetTargetModules(moduleSet), true)
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

// ModuleSetTargetLocalModulesAndTransitiveLocalDeps is a convenience function that returns
// the targeted local Modules of the ModuleSet, and their transitive local dependencies.
//
// This is used by push to determine what Modules should be uploaded.
//
// Sorted by OpaqueID.
func ModuleSetTargetLocalModulesAndTransitiveLocalDeps(moduleSet ModuleSet) ([]Module, error) {
	targetLocalModules := slicesext.Filter(
		moduleSet.Modules(),
		func(module Module) bool { return module.IsTarget() && module.IsLocal() },
	)
	// It is technically possible for us to have a targeted local Module depend on a
	// Remote dep, which depends on a local dep. In this case, we want to also
	// include this transitive local dep. The resultOpaqueIDToLocalModule map only
	// includes our result Modules, but we want the visited map to include everything
	// we have visited so we can stop recursion.
	visitedOpaqueIDs := make(map[string]struct{})
	resultOpaqueIDToLocalModule := make(map[string]Module)
	for _, module := range targetLocalModules {
		if err := moduleSetTargetLocalModulesAndTransitiveLocalDepsRec(
			visitedOpaqueIDs,
			resultOpaqueIDToLocalModule,
			module,
		); err != nil {
			return nil, err
		}
	}
	resultLocalModules := slicesext.MapValuesToSlice(resultOpaqueIDToLocalModule)
	sort.Slice(
		resultLocalModules,
		func(i int, j int) bool {
			return resultLocalModules[i].OpaqueID() < resultLocalModules[j].OpaqueID()
		},
	)
	return resultLocalModules, nil
}

// ModuleSetOpaqueIDs is a convenience function that returns a slice of the OpaqueIDs of the
// Modules in the ModuleSet.
//
// Sorted.
func ModuleSetOpaqueIDs(moduleSet ModuleSet) []string {
	return modulesOpaqueIDs(moduleSet.Modules())
}

// ModuleSetTargetOpaqueIDs is a convenience function that returns a slice of the OpaqueIDs of the
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

// ModuleSetToDAG gets a DAG of the given ModuleSet.
//
// This only starts at target Modules. If a Module is not part of a graph
// with a target Module as a source, it will not be added.
func ModuleSetToDAG(moduleSet ModuleSet) (*dag.Graph[string, Module], error) {
	graph := dag.NewGraph[string, Module](Module.OpaqueID)
	for _, module := range ModuleSetTargetModules(moduleSet) {
		if err := moduleSetToDAGRec(module, graph); err != nil {
			return nil, err
		}
	}
	return graph, nil
}

// *** PRIVATE ***

// moduleSet

type moduleSet struct {
	tracer                       tracing.Tracer
	modules                      []Module
	moduleFullNameStringToModule map[string]Module
	opaqueIDToModule             map[string]Module
	bucketIDToModule             map[string]Module
	commitIDToModule             map[uuid.UUID]Module

	// filePathToModule is a cache of filePath -> module.
	//
	// If you are calling both the imports and module caches, you must call the imports cache first,
	// i.e. lock ordering.
	filePathToModuleCache cache.Cache[string, Module]
}

func newModuleSet(
	tracer tracing.Tracer,
	modules []Module,
) (*moduleSet, error) {
	moduleFullNameStringToModule := make(map[string]Module, len(modules))
	opaqueIDToModule := make(map[string]Module, len(modules))
	bucketIDToModule := make(map[string]Module, len(modules))
	commitIDToModule := make(map[uuid.UUID]Module, len(modules))
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
		commitID := module.CommitID()
		if !commitID.IsNil() {
			if _, ok := commitIDToModule[commitID]; ok {
				// This should never happen.
				return nil, syserror.Newf("duplicate CommitID %q when constructing ModuleSet", uuidutil.ToDashless(commitID))
			}
			commitIDToModule[commitID] = module
		}
	}
	moduleSet := &moduleSet{
		tracer:                       tracer,
		modules:                      modules,
		moduleFullNameStringToModule: moduleFullNameStringToModule,
		opaqueIDToModule:             opaqueIDToModule,
		bucketIDToModule:             bucketIDToModule,
		commitIDToModule:             commitIDToModule,
	}
	for _, module := range modules {
		module.setModuleSet(moduleSet)
	}
	return moduleSet, nil
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

func (m *moduleSet) GetModuleForCommitID(commitID uuid.UUID) Module {
	return m.commitIDToModule[commitID]
}

func (m *moduleSet) WithTargetOpaqueIDs(opaqueIDs ...string) (ModuleSet, error) {
	if len(opaqueIDs) == 0 {
		return nil, errors.New("at least one Module must be targeted")
	}
	opaqueIDMap := slicesext.ToStructMap(opaqueIDs)
	modules := make([]Module, len(m.modules))
	for i, module := range m.modules {
		_, isTarget := opaqueIDMap[module.OpaqueID()]
		// Always make a copy regardless of if targeting changes. We're going to set a new ModuleSet on the Module.
		module, err := module.withIsTarget(isTarget)
		if err != nil {
			return nil, err
		}
		modules[i] = module
	}
	return newModuleSet(m.tracer, modules)
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

func (m *moduleSet) getModuleForFilePathUncached(ctx context.Context, filePath string) (_ Module, retErr error) {
	matchingOpaqueIDs := make(map[string]struct{})
	// Note that we're effectively doing an O(num_modules * num_files) operation here, which could be prohibitive.
	for _, module := range m.Modules() {
		_, err := module.StatFileInfo(ctx, filePath)
		if err == nil {
			matchingOpaqueIDs[module.OpaqueID()] = struct{}{}
		} else if !errors.Is(err, fs.ErrNotExist) {
			// This is important! If we have any error other than fs.ErrNotExist, make sure we return that error.
			// Not doing so results in important errors not being propagated. In the case where this was found,
			// we ended up returning a fs.ErrNotExist when the underlying error was a digest verification error
			// that we expected. We want to return the verification error up the stack.
			return nil, err
		}
	}
	switch len(matchingOpaqueIDs) {
	case 0:
		// This will happen if there is a file path we cannot find in our modules, which will result
		// in an error on ModuleDeps() or Digest().
		//
		// Note that we will commonly call getModuleForFilePathUncached for a WKT, which will result
		// in this error (this is desired).
		return nil, &fs.PathError{Op: "stat", Path: filePath, Err: fs.ErrNotExist}
	case 1:
		var matchingOpaqueID string
		for matchingOpaqueID = range matchingOpaqueIDs {
		}
		return m.GetModuleForOpaqueID(matchingOpaqueID), nil
	default:
		// This actually could happen, and we will want to make this error message as clear as possible.
		// The addition of opaqueID should give us clearer error messages than we have today.
		return nil, &DuplicateProtoPathError{
			ProtoPath: filePath,
			OpaqueIDs: slicesext.MapKeysToSortedSlice(matchingOpaqueIDs),
		}
	}
}

func (*moduleSet) isModuleSet() {}

// utils

func moduleSetTargetLocalModulesAndTransitiveLocalDepsRec(
	visitedOpaqueIDs map[string]struct{},
	resultOpaqueIDToLocalModule map[string]Module,
	module Module,
) error {
	opaqueID := module.OpaqueID()
	if _, ok := visitedOpaqueIDs[opaqueID]; ok {
		return nil
	}
	visitedOpaqueIDs[opaqueID] = struct{}{}
	if module.IsLocal() {
		resultOpaqueIDToLocalModule[opaqueID] = module
	}
	moduleDeps, err := module.ModuleDeps()
	if err != nil {
		return err
	}
	for _, moduleDep := range moduleDeps {
		// We want to recurse on both local and remote deps.
		//
		// It is technically possible for us to have a targeted local Module depend on a
		// Remote dep, which depends on a local dep. In this case, we want to also
		// include this transitive local dep.
		if err := moduleSetTargetLocalModulesAndTransitiveLocalDepsRec(
			visitedOpaqueIDs,
			resultOpaqueIDToLocalModule,
			moduleDep,
		); err != nil {
			return err
		}
	}
	return nil
}

func moduleSetToDAGRec(
	module Module,
	graph *dag.Graph[string, Module],
) error {
	graph.AddNode(module)
	directModuleDeps, err := ModuleDirectModuleDeps(module)
	if err != nil {
		return err
	}
	for _, directModuleDep := range directModuleDeps {
		graph.AddEdge(module, directModuleDep)
		if err := moduleSetToDAGRec(directModuleDep, graph); err != nil {
			return err
		}
	}
	return nil
}

func modulesOpaqueIDs(modules []Module) []string {
	return slicesext.Map(
		modules,
		func(module Module) string { return module.OpaqueID() },
	)
}

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

package bufmodulebuild

import (
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"go.uber.org/zap"
)

type moduleFileSetBuilder struct {
	logger         *zap.Logger
	moduleReader   bufmodule.ModuleReader
	moduleResolver bufmodule.ModuleResolver
}

func newModuleFileSetBuilder(
	logger *zap.Logger,
	moduleReader bufmodule.ModuleReader,
	moduleResolver bufmodule.ModuleResolver,
) *moduleFileSetBuilder {
	return &moduleFileSetBuilder{
		logger:         logger,
		moduleReader:   moduleReader,
		moduleResolver: moduleResolver,
	}
}
func (m *moduleFileSetBuilder) Build(
	ctx context.Context,
	module bufmodule.Module,
	options ...BuildModuleFileSetOption,
) (bufmodule.ModuleFileSet, error) {
	buildModuleFileSetOptions := &buildModuleFileSetOptions{}
	for _, option := range options {
		option(buildModuleFileSetOptions)
	}
	return m.build(
		ctx,
		module,
		buildModuleFileSetOptions.workspace,
	)
}

func (m *moduleFileSetBuilder) build(
	ctx context.Context,
	module bufmodule.Module,
	workspace bufmodule.Workspace,
) (bufmodule.ModuleFileSet, error) {
	if workspace == nil {
		// If we don't have a workspace, we can simply include the module and its dependencies.
		var dependencyModules []bufmodule.Module
		for _, dependencyModulePin := range module.DependencyModulePins() {
			dependencyModule, err := m.moduleReader.GetModule(ctx, dependencyModulePin)
			if err != nil {
				return nil, err
			}
			dependencyModules = append(dependencyModules, dependencyModule)
		}
		return bufmodule.NewModuleFileSet(module, dependencyModules), nil
	}
	// If we have a workspace, we need to merge its dependencies with other modules in the
	// workspace, including transitive dependencies, and resolve conflicts.
	dependencyModules, err := m.getDependenciesForWorkspaceModule(ctx, module, workspace)
	if err != nil {
		return nil, err
	}
	return bufmodule.NewModuleFileSet(module, dependencyModules), nil
}

func (m *moduleFileSetBuilder) getDependenciesForWorkspaceModule(
	ctx context.Context,
	module bufmodule.Module,
	workspace bufmodule.Workspace,
) ([]bufmodule.Module, error) {
	// We do a first pass to collect all module pins that we are going to resolve
	// through the module reader. If there are any conflicts, we will need to go
	// through dependency resolution.
	var (
		dependencyModulePins              []bufmoduleref.ModulePin
		seenDependencyModulePinIdentities = make(map[string]bufmoduleref.ModulePin)
		conflicts                         []bufmoduleref.ModuleReference
	)
	// This includes all dependencies of this module, including transitive dependencies.
	// However this transitive closure does not take into account implicit dependencies
	// on workspace modules, who may have conflicting module pins as compared to these ones.
	for _, dependencyModulePin := range module.DependencyModulePins() {
		dependencyModulePins = append(dependencyModulePins, dependencyModulePin)
		seenDependencyModulePinIdentities[dependencyModulePin.IdentityString()] = dependencyModulePin
	}
	// From the perspective of the ModuleFileSet, we include all of the files
	// specified in the workspace. When we build the Image from the ModuleFileSet,
	// we construct it based on the TargetFileInfos, and thus only include the files
	// in the transitive closure.
	//
	// This is defensible as we're saying that everything in the workspace is a potential
	// dependency, even if some are not actual dependencies of this specific module. In this
	// case, the extra modules are no different than unused dependencies in a buf.yaml/buf.lock.
	//
	// By including all the Modules from the workspace, we are potentially including the input
	// Module itself. This is bad, and will result in errors when using the result ModuleFileSet.
	// The ModuleFileSet expects a Module, and its dependency Modules, but it is not OK for
	// a Module to both be the input Module and a dependency Module. This means we need to check
	// each module in the workspace to see if it is the same as the input module, and only add
	// it as a dependency if it's not the same as the input module. We determine this by comparing
	// each workspace module's WorkspaceDirectory with the input module's WorkspaceDirectory,
	// where having the same WorkspaceDirectory means being the same module. This is correct
	// because each module in a workspace are constructed with its WorkspaceDirectory set to its
	// path relative to buf.work.yaml and this value is guaranteed to be unique within the same
	// workspace. This is predicated on the input module belonging to the workspace, and it would
	// be a bug if the input module doesn't belong to this workspace.
	//
	// We could also determine which modules could be omitted here, but it would incur
	// the cost of parsing the target files and detecting exactly which imports are
	// used. We already get this for free in Image construction, so it's simplest and
	// most efficient to bundle all of the modules together like so.
	for _, workspaceModule := range workspace.GetModules() {
		if module.WorkspaceDirectory() == workspaceModule.WorkspaceDirectory() {
			// Don't add a dependency to itself.
			continue
		}
		for _, pin := range workspaceModule.DependencyModulePins() {
			if _, ok := workspace.GetModule(pin); ok {
				// This dependency will already be provided by the workspace, so we don't need to collect it.
				continue
			}
			if _, conflict := seenDependencyModulePinIdentities[pin.IdentityString()]; conflict {
				// Conflicting dependency module. We carry on, but we'll need to
				// go through dependency resolution now.
				conflictingPinAsRef, err := bufmoduleref.NewModuleReference(
					pin.Remote(),
					pin.Owner(),
					pin.Repository(),
					pin.Commit(),
				)
				if err != nil {
					return nil, err
				}
				conflicts = append(conflicts, conflictingPinAsRef)
			} else {
				seenDependencyModulePinIdentities[pin.IdentityString()] = pin
				dependencyModulePins = append(dependencyModulePins, pin)
			}
		}
	}
	if len(conflicts) > 0 {
		// We've found some pins across the workspace with conflicts. We need to go through
		// dependency resolution to obtain a new set of pins before we resolve the pins via
		// the module reader. These pins are guaranteed to not contain any modules within
		// the workspace itself.
		m.logger.Debug("found conflicts in dependenies across modules in a workspace, running dependency resolution")
		resolvedPins, err := m.moduleResolver.GetModulePins(
			ctx,
			conflicts,
			dependencyModulePins,
		)
		if err != nil {
			return nil, err
		}
		dependencyModulePins = resolvedPins
	}
	// We have collected all dependency module pins and resolved conflicts. We can now
	// resolve the pins and finally add the workspace modules as dependencies.
	var dependencyModules []bufmodule.Module
	for _, workspaceModule := range workspace.GetModules() {
		if module.WorkspaceDirectory() == workspaceModule.WorkspaceDirectory() {
			// Don't add a dependency to itself.
			continue
		}
		// Include this module as a dependency, as well as its transitive dependencies.
		// If we've already seen this transitive dependency before, but for a different
		// commit, we will have to go through dependency resolution.
		dependencyModules = append(dependencyModules, workspaceModule)
	}
	for _, dependencyModulePin := range dependencyModulePins {
		dependencyModule, err := m.moduleReader.GetModule(ctx, dependencyModulePin)
		if err != nil {
			return nil, err
		}
		dependencyModules = append(dependencyModules, dependencyModule)
	}
	return dependencyModules, nil
}

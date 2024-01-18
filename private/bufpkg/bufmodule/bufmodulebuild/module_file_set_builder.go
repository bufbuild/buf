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

package bufmodulebuild

import (
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"go.uber.org/zap"
)

type moduleFileSetBuilder struct {
	logger       *zap.Logger
	moduleReader bufmodule.ModuleReader
}

func newModuleFileSetBuilder(
	logger *zap.Logger,
	moduleReader bufmodule.ModuleReader,
) *moduleFileSetBuilder {
	return &moduleFileSetBuilder{
		logger:       logger,
		moduleReader: moduleReader,
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
	var dependencyModules []bufmodule.Module
	moduleWorkspaceDirectory := module.WorkspaceDirectory()
	if workspace != nil {
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
		for _, potentialDependencyModule := range workspace.GetModules() {
			if moduleWorkspaceDirectory != potentialDependencyModule.WorkspaceDirectory() {
				dependencyModules = append(dependencyModules, potentialDependencyModule)
			}
		}
	}
	// We know these are unique by remote, owner, repository and
	// contain all transitive dependencies.
	for _, dependencyModulePin := range module.DependencyModulePins() {
		if workspace != nil {
			if _, ok := workspace.GetModule(dependencyModulePin); ok {
				// This dependency is already provided by the workspace, so we don't
				// need to consult the ModuleReader.
				continue
			}
		}
		dependencyModule, err := m.moduleReader.GetModule(ctx, dependencyModulePin)
		if err != nil {
			return nil, err
		}
		dependencyModules = append(dependencyModules, dependencyModule)
	}
	return bufmodule.NewModuleFileSet(module, dependencyModules), nil
}

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
	"bytes"
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"go.uber.org/zap"
	"golang.org/x/crypto/sha3"
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
	if workspace != nil {
		moduleProtoPathsHash, err := protoPathsHash(ctx, module)
		if err != nil {
			return nil, err
		}
		// From the perspective of the ModuleFileSet, we include all of the files
		// specified in the workspace. When we build the Image from the ModuleFileSet,
		// we construct it based on the TargetFileInfos, and thus only include the files
		// in the transitive closure.
		//
		// We *could* determine which modules could be omitted here, but it would incur
		// the cost of parsing the target files and detecting exactly which imports are
		// used. We already get this for free in Image construction, so it's simplest and
		// most efficient to bundle all of the modules together like so.
		for _, potentialDependencyModule := range workspace.GetModules() {
			potentialDependencyModuleProtoPathsHash, err := protoPathsHash(ctx, potentialDependencyModule)
			if err != nil {
				return nil, err
			}
			if !bytes.Equal(moduleProtoPathsHash, potentialDependencyModuleProtoPathsHash) {
				dependencyModules = append(dependencyModules, potentialDependencyModule)
			}
			// We have to make sure that the dependency module is not the input source modules
			//
			// TODO: this is hacky and relies on Golang semantics, and that the Module objects
			// in the Workspace are the same as the potential source module. We really need a
			// better way to ID Modules, as this is still a bug in its current form. In the
			// best case, we could use ModuleIdentity and Commit to ID a module, but we aren't
			// guaranteed that these are set.
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

func protoPathsHash(ctx context.Context, module bufmodule.Module) ([]byte, error) {
	fileInfos, err := module.SourceFileInfos(ctx)
	if err != nil {
		return nil, err
	}
	shakeHash := sha3.NewShake256()
	for _, fileInfo := range fileInfos {
		_, err := shakeHash.Write([]byte(fileInfo.Path()))
		if err != nil {
			return nil, err
		}
	}
	data := make([]byte, 64)
	if _, err := shakeHash.Read(data); err != nil {
		return nil, err
	}
	return data, nil
}

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
	"encoding/hex"
	"errors"

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
	hashes := make(map[string]struct{})
	moduleHash, err := protoPathsHash(ctx, module)
	if err != nil {
		return nil, err
	}
	hashes[moduleHash] = struct{}{}
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
		// a Module to both be the input Module and a dependency Module. We have no concept
		// of Module "ID" - a Module may have a ModuleIdentity and commit associated with it,
		// but there is no guarantee of this. To get around this, we do a hash of the .proto file
		// paths within a Module, and say that Modules are equivalent if they contain the exact
		// same .proto file paths. If they have the same .proto file paths, then we do not
		// add the Module as a dependency.
		//
		// We know from bufmodule.Workspace that no two Modules in a Workspace will have overlapping
		// file paths, therefore if the Module is in the Workspace and has equivalent file paths,
		// we know that it must be the same module. If the Module is not in the workspace...why
		// did we provide a workspace?
		//
		// We could use other methods for equivalence or to say "do not add":
		//
		//   - If there are any overlapping files: for example, one module has a.proto, one module
		//     has b.proto, and both have c.proto. We don't use this heuristic as what we are looking
		//     for here is a situation where based on our Module construction, we have two actually-equivalent
		//     Modules. The existence of any overlapping files will result in an error during build, which
		//     is what we want. This would also indicate this Module did not come from the Workspace, given
		//     the property of file uniqueness in Workspaces.
		//   - Golang object equivalence: for example, doing "module != potentialDependencyModule". This
		//     happens to work since we only construct Modules once, but it's error-prone: it's totally
		//     possible to create two Module objects from the same source, and if they represent the
		//     same Module on disk/in the BSR, we don't want to include these as duplicates.
		//   - Full module digest and/or proto file content: We could include buf.yaml, buf.lock,
		//     README.md, etc, and also hash the actual content of the .proto files, but we're saying
		//     that this doesn't help us any more than just comparing .proto files, and may lead to
		//     false negatives. However, this is the most likely candidate as an alternative, as you
		//     could argue that at the ModuleFileSetBuilder level, we should say "assume any difference
		//     is a real difference".
		//
		// We could also determine which modules could be omitted here, but it would incur
		// the cost of parsing the target files and detecting exactly which imports are
		// used. We already get this for free in Image construction, so it's simplest and
		// most efficient to bundle all of the modules together like so.
		for _, potentialDependencyModule := range workspace.GetModules() {
			potentialDependencyModuleHash, err := protoPathsHash(ctx, potentialDependencyModule)
			if err != nil {
				return nil, err
			}
			if _, ok := hashes[potentialDependencyModuleHash]; !ok {
				dependencyModules = append(dependencyModules, potentialDependencyModule)
			} else {
				hashes[potentialDependencyModuleHash] = struct{}{}
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
		dependencyModuleHash, err := protoPathsHash(ctx, dependencyModule)
		if err != nil {
			return nil, err
		}
		// At this point, this is really just a safety check.
		if _, ok := hashes[dependencyModuleHash]; ok {
			return nil, errors.New("module declared in DependencyModulePins but not in workspace was already added to the dependency Module set, this is a system error")
		}
		dependencyModules = append(dependencyModules, dependencyModule)
		hashes[dependencyModuleHash] = struct{}{}
	}
	return bufmodule.NewModuleFileSet(module, dependencyModules), nil
}

// protoPathsHash returns a hash representing the paths of the .proto files within the Module.
func protoPathsHash(ctx context.Context, module bufmodule.Module) (string, error) {
	fileInfos, err := module.SourceFileInfos(ctx)
	if err != nil {
		return "", err
	}
	shakeHash := sha3.NewShake256()
	for _, fileInfo := range fileInfos {
		_, err := shakeHash.Write([]byte(fileInfo.Path()))
		if err != nil {
			return "", err
		}
	}
	data := make([]byte, 64)
	if _, err := shakeHash.Read(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

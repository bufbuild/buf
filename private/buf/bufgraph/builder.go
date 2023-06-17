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

package bufgraph

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufbuild"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/dag"
	"go.uber.org/zap"
)

type builder struct {
	logger         *zap.Logger
	moduleResolver bufmodule.ModuleResolver
	moduleReader   bufmodule.ModuleReader
	buildBuilder   bufbuild.Builder
}

func newBuilder(
	logger *zap.Logger,
	moduleResolver bufmodule.ModuleResolver,
	moduleReader bufmodule.ModuleReader,
) *builder {
	return &builder{
		logger:         logger,
		moduleResolver: moduleResolver,
		moduleReader:   moduleReader,
		buildBuilder: bufbuild.NewBuilder(
			logger,
			moduleReader,
		),
	}
}

func (b *builder) Build(
	ctx context.Context,
	modules []bufmodule.Module,
	options ...BuildOption,
) (*dag.Graph[Node], []bufanalysis.FileAnnotation, error) {
	buildOptions := newBuildOptions()
	for _, option := range options {
		option(buildOptions)
	}
	return b.build(
		ctx,
		modules,
		buildOptions.workspace,
	)
}

func (b *builder) build(
	ctx context.Context,
	modules []bufmodule.Module,
	workspace bufmodule.Workspace,
) (*dag.Graph[Node], []bufanalysis.FileAnnotation, error) {
	graph := dag.NewGraph[Node]()
	for i, module := range modules {
		// TODO: this is because we don't have an identifier for Module
		// We likely want to use the ModuleIdentityOptionalCommit if present,
		// or the path on disk otherwise. This will optimally require refactors
		// to the Module and the workspace-related code.
		node := Node{
			Value: "root",
		}
		if i > 1 {
			node.Value += fmt.Sprintf("-%d", i)
		}
		fileAnnotations, err := b.buildForModule(
			ctx,
			module,
			node,
			workspace,
			graph,
		)
		if err != nil {
			return nil, nil, err
		}
		if len(fileAnnotations) > 0 {
			return nil, fileAnnotations, nil
		}
	}
	return graph, nil, nil
}

func (b *builder) buildForModule(
	ctx context.Context,
	module bufmodule.Module,
	node Node,
	workspace bufmodule.Workspace,
	graph *dag.Graph[Node],
) ([]bufanalysis.FileAnnotation, error) {
	image, fileAnnotations, err := b.buildBuilder.Build(
		ctx,
		module,
		bufbuild.BuildWithWorkspace(workspace),
	)
	if err != nil {
		return nil, err
	}
	if len(fileAnnotations) > 0 {
		return fileAnnotations, nil
	}
	for _, imageModuleDependency := range bufimage.ImageModuleDependencies(image) {
		dependencyNode := newNodeForModuleIdentityOptionalCommit(imageModuleDependency)
		if imageModuleDependency.IsDirect() {
			graph.AddEdge(node, dependencyNode)
			//fmt.Printf("adding %v to %v nodes\n", node, dependencyNode)
		} else {
			dependencyModule, err := b.getModuleForModuleIdentityOptionalCommit(
				ctx,
				imageModuleDependency,
				workspace,
			)
			if err != nil {
				return nil, err
			}
			// TODO: deal with the case where there are differing commits for a given ModuleIdentity.
			fileAnnotations, err := b.buildForModule(
				ctx,
				dependencyModule,
				dependencyNode,
				workspace,
				graph,
			)
			if err != nil {
				return nil, err
			}
			if len(fileAnnotations) > 0 {
				return fileAnnotations, nil
			}
		}
	}
	return nil, nil
}

func (b *builder) getModuleForModuleIdentityOptionalCommit(
	ctx context.Context,
	moduleIdentityOptionalCommit bufmoduleref.ModuleIdentityOptionalCommit,
	workspace bufmodule.Workspace,
) (bufmodule.Module, error) {
	if workspace != nil {
		module, ok := workspace.GetModule(moduleIdentityOptionalCommit)
		if ok {
			return module, nil
		}
	}
	modulePin, err := b.moduleResolver.GetModulePin(
		ctx,
		bufmoduleref.NewModuleReferenceForModuleIdentityOptionalCommit(
			moduleIdentityOptionalCommit,
		),
	)
	if err != nil {
		return nil, err
	}
	return b.moduleReader.GetModule(
		ctx,
		modulePin,
	)
}

func newNodeForModuleIdentityOptionalCommit(
	moduleIdentityOptionalCommit bufmoduleref.ModuleIdentityOptionalCommit,
) Node {
	return Node{
		Value: moduleIdentityOptionalCommit.String(),
	}
}

type buildOptions struct {
	workspace bufmodule.Workspace
}

func newBuildOptions() *buildOptions {
	return &buildOptions{}
}

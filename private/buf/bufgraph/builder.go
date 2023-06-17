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
	moduleReader   bufmodule.ModuleReader
	moduleResolver bufmodule.ModuleResolver
	buildBuilder   bufbuild.Builder
}

func newBuilder(
	logger *zap.Logger,
	moduleReader bufmodule.ModuleReader,
	moduleResolver bufmodule.ModuleResolver,
) *builder {
	return &builder{
		logger:         logger,
		moduleReader:   moduleReader,
		moduleResolver: moduleResolver,
		buildBuilder: bufbuild.NewBuilder(
			logger,
			moduleReader,
		),
	}
}

func (b *builder) Build(
	ctx context.Context,
	moduleNodePairs []ModuleNodePair,
	options ...BuildOption,
) (*dag.Graph[Node], []bufanalysis.FileAnnotation, error) {
	buildOptions := newBuildOptions()
	for _, option := range options {
		option(buildOptions)
	}
	return b.build(
		ctx,
		moduleNodePairs,
		buildOptions.workspace,
	)
}

func (b *builder) build(
	ctx context.Context,
	moduleNodePairs []ModuleNodePair,
	workspace bufmodule.Workspace,
) (*dag.Graph[Node], []bufanalysis.FileAnnotation, error) {
	graph := dag.NewGraph[Node]()
	for _, moduleNodePair := range moduleNodePairs {
		fileAnnotations, err := b.buildForModule(
			ctx,
			moduleNodePair.Module,
			moduleNodePair.Node,
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
			if !graph.ContainsNode(dependencyNode) {
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

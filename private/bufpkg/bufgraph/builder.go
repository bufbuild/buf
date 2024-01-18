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

package bufgraph

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/dag"
	"go.uber.org/zap"
)

type builder struct {
	logger         *zap.Logger
	moduleResolver bufmodule.ModuleResolver
	moduleReader   bufmodule.ModuleReader
	imageBuilder   bufimagebuild.Builder
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
		imageBuilder: bufimagebuild.NewBuilder(
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
	alreadyProcessedNodes := make(map[Node]struct{})
	for _, module := range modules {
		fileAnnotations, err := b.buildForModule(
			ctx,
			module,
			newNodeForModule(module),
			workspace,
			graph,
			alreadyProcessedNodes,
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
	alreadyProcessedNodes map[Node]struct{},
) ([]bufanalysis.FileAnnotation, error) {
	// We can't rely on the existence of a node in the graph for this, as when we add an edge
	// to the graph, the node is added, and we still need to process the node as a potential
	// source node.
	if _, ok := alreadyProcessedNodes[node]; ok {
		return nil, nil
	}
	alreadyProcessedNodes[node] = struct{}{}
	graph.AddNode(node)
	image, fileAnnotations, err := b.imageBuilder.Build(
		ctx,
		module,
		bufimagebuild.WithWorkspace(workspace),
		bufimagebuild.WithExpectedDirectDependencies(module.DeclaredDirectDependencies()),
	)
	if err != nil {
		return nil, err
	}
	if len(fileAnnotations) > 0 {
		return fileAnnotations, nil
	}
	for _, imageModuleDependency := range bufimage.ImageModuleDependencies(image) {
		dependencyNode := newNodeForImageModuleDependency(imageModuleDependency)
		if imageModuleDependency.IsDirect() {
			graph.AddEdge(node, dependencyNode)
		}
		dependencyModule, err := b.getModuleForImageModuleDependency(
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
			alreadyProcessedNodes,
		)
		if err != nil {
			return nil, err
		}
		if len(fileAnnotations) > 0 {
			return fileAnnotations, nil
		}
	}
	return nil, nil
}

func (b *builder) getModuleForImageModuleDependency(
	ctx context.Context,
	imageModuleDependency bufimage.ImageModuleDependency,
	workspace bufmodule.Workspace,
) (bufmodule.Module, error) {
	moduleIdentity := imageModuleDependency.ModuleIdentity()
	commit := imageModuleDependency.Commit()
	if workspace != nil {
		module, ok := workspace.GetModule(moduleIdentity)
		if ok {
			return module, nil
		}
	}
	if commit == "" {
		// TODO: can we error here? The only
		// case we should likely not have a commit is when we are using a workspace.
		// There's no enforcement of this property, so erroring here is a bit weird,
		// but it might be better to check our assumptions and figure out if there
		// are exceptions after the fact, as opposed to resolving a ModulePin for
		// main when we don't know if main is what we want.
		return nil, fmt.Errorf("had ModuleIdentity %v with no associated commit, but did not have the module in a workspace", moduleIdentity)
	}
	moduleReference, err := bufmoduleref.NewModuleReference(
		moduleIdentity.Remote(),
		moduleIdentity.Owner(),
		moduleIdentity.Repository(),
		commit,
	)
	if err != nil {
		return nil, err
	}
	modulePin, err := b.moduleResolver.GetModulePin(
		ctx,
		moduleReference,
	)
	if err != nil {
		return nil, err
	}
	return b.moduleReader.GetModule(
		ctx,
		modulePin,
	)
}

func newNodeForImageModuleDependency(imageModuleDependency bufimage.ImageModuleDependency) Node {
	return Node{
		Remote:     imageModuleDependency.ModuleIdentity().Remote(),
		Owner:      imageModuleDependency.ModuleIdentity().Owner(),
		Repository: imageModuleDependency.ModuleIdentity().Repository(),
		Commit:     imageModuleDependency.Commit(),
	}
}

func newNodeForModule(module bufmodule.Module) Node {
	// TODO: deal with unnamed Modules
	var node Node
	if moduleIdentity := module.ModuleIdentity(); moduleIdentity != nil {
		node.Remote = moduleIdentity.Remote()
		node.Owner = moduleIdentity.Owner()
		node.Repository = moduleIdentity.Repository()
		node.Commit = module.Commit()
	}
	return node
}

type buildOptions struct {
	workspace bufmodule.Workspace
}

func newBuildOptions() *buildOptions {
	return &buildOptions{}
}

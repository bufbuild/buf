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
	"errors"

	"github.com/bufbuild/buf/private/buf/bufbuild"
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
	modules []bufmodule.Module,
	options ...BuildOption,
) (*dag.Graph[Node], error) {
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
) (*dag.Graph[Node], error) {
	graph := dag.NewGraph[Node]()
	for _, module := range modules {
		if err := b.buildForModule(
			ctx,
			module,
			workspace,
			graph,
		); err != nil {
			return nil, err
		}
	}
	return graph, nil
}

func (b *builder) buildForModule(
	ctx context.Context,
	module bufmodule.Module,
	workspace bufmodule.Workspace,
	graph *dag.Graph[Node],
) error {
	image, err := b.buildBuilder.Build(
		ctx,
		module,
		bufbuild.BuildWithWorkspace(workspace),
	)
	if err != nil {
		return err
	}
	for _, imageModuleDependency := range bufimage.ImageModuleDependencies(image) {
		if imageModuleDependency.IsDirect() {
			// TODO: add to graph
			// TODO: need an identity for the input module to be able to do this
		} else {
			dependencyModule, err := b.getModuleForModuleIdentityOptionalCommit(
				ctx,
				imageModuleDependency,
			)
			if err != nil {
				return err
			}
			// TODO: do not build if the graph already contains a node
			// that represents this module
			if err := b.buildForModule(
				ctx,
				dependencyModule,
				workspace,
				graph,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *builder) getModuleForModuleIdentityOptionalCommit(
	ctx context.Context,
	moduleIdentityOptionalCommit bufmoduleref.ModuleIdentityOptionalCommit,
) (bufmodule.Module, error) {
	return nil, errors.New("TODO")
}

func newNode(moduleIdentityOptionalCommit bufmoduleref.ModuleIdentityOptionalCommit) *Node {
	return &Node{
		Remote:     moduleIdentityOptionalCommit.Remote(),
		Owner:      moduleIdentityOptionalCommit.Owner(),
		Repository: moduleIdentityOptionalCommit.Repository(),
		Commit:     moduleIdentityOptionalCommit.Commit(),
	}
}

type buildOptions struct {
	workspace bufmodule.Workspace
}

func newBuildOptions() *buildOptions {
	return &buildOptions{}
}

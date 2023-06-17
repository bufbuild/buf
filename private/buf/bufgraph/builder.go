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

	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/pkg/dag"
	"go.uber.org/zap"
)

type builder struct {
	logger               *zap.Logger
	moduleFileSetBuilder bufmodulebuild.ModuleFileSetBuilder
	imageBuilder         bufimagebuild.Builder
}

func newBuilder(
	logger *zap.Logger,
	moduleFileSetBuilder bufmodulebuild.ModuleFileSetBuilder,
	imageBuilder bufimagebuild.Builder,
) *builder {
	return &builder{
		logger:               logger,
		moduleFileSetBuilder: moduleFileSetBuilder,
		imageBuilder:         imageBuilder,
	}
}

func (b *builder) Build(
	ctx context.Context,
	modules []bufmodule.Module,
	options ...BuildOption,
) (*dag.Graph[Dependency], error) {
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
) (*dag.Graph[Dependency], error) {
	return nil, errors.New("TODO")
}

type buildOptions struct {
	workspace bufmodule.Workspace
}

func newBuildOptions() *buildOptions {
	return &buildOptions{}
}

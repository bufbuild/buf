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

package bufbuild

import (
	"context"
	"errors"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"go.uber.org/zap"
)

type builder struct {
	logger               *zap.Logger
	moduleFileSetBuilder bufmodulebuild.ModuleFileSetBuilder
	imageBuilder         bufimagebuild.Builder
}

func newBuilder(
	logger *zap.Logger,
	moduleReader bufmodule.ModuleReader,
) *builder {
	return &builder{
		logger: logger,
		moduleFileSetBuilder: bufmodulebuild.NewModuleFileSetBuilder(
			logger,
			moduleReader,
		),
		imageBuilder: bufimagebuild.NewBuilder(
			logger,
		),
	}
}

func (b *builder) Build(
	ctx context.Context,
	module bufmodule.Module,
	options ...BuildOption,
) (bufimage.Image, error) {
	buildOptions := newBuildOptions()
	for _, option := range options {
		option(buildOptions)
	}
	return b.build(
		ctx,
		module,
		buildOptions.workspace,
	)
}

func (b *builder) build(
	ctx context.Context,
	module bufmodule.Module,
	workspace bufmodule.Workspace,
) (bufimage.Image, error) {
	return nil, errors.New("TODO")
}

type buildOptions struct {
	workspace bufmodule.Workspace
}

func newBuildOptions() *buildOptions {
	return &buildOptions{}
}

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

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
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
) (bufimage.Image, []bufanalysis.FileAnnotation, error) {
	buildOptions := newBuildOptions()
	for _, option := range options {
		option(buildOptions)
	}
	return b.build(
		ctx,
		module,
		buildOptions.workspace,
		buildOptions.excludeSourceCodeInfo,
	)
}

func (b *builder) build(
	ctx context.Context,
	module bufmodule.Module,
	workspace bufmodule.Workspace,
	excludeSourceCodeInfo bool,
) (_ bufimage.Image, _ []bufanalysis.FileAnnotation, retErr error) {
	ctx, span := otel.GetTracerProvider().Tracer("bufbuild/buf").Start(ctx, "build_module")
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
		span.End()
	}()

	moduleFileSet, err := b.moduleFileSetBuilder.Build(
		ctx,
		module,
		bufmodulebuild.WithWorkspace(workspace),
	)
	if err != nil {
		return nil, nil, err
	}
	imageBuildOptions := []bufimagebuild.BuildOption{
		bufimagebuild.WithExpectedDirectDependencies(module.DeclaredDirectDependencies()),
		bufimagebuild.WithLocalWorkspace(workspace),
	}
	if excludeSourceCodeInfo {
		imageBuildOptions = append(imageBuildOptions, bufimagebuild.WithExcludeSourceCodeInfo())
	}
	return b.imageBuilder.Build(
		ctx,
		moduleFileSet,
		imageBuildOptions...,
	)
}

type buildOptions struct {
	workspace             bufmodule.Workspace
	excludeSourceCodeInfo bool
}

func newBuildOptions() *buildOptions {
	return &buildOptions{}
}

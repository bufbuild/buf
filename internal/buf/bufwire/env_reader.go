// Copyright 2020 Buf Technologies, Inc.
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

package bufwire

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/internal/buf/buffetch"
	"github.com/bufbuild/buf/internal/pkg/app"
	"go.opencensus.io/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type envReader struct {
	logger               *zap.Logger
	fetchReader          buffetch.Reader
	moduleBucketBuilder  bufmodulebuild.ModuleBucketBuilder
	moduleFileSetBuilder bufmodulebuild.ModuleFileSetBuilder
	imageBuilder         bufimagebuild.Builder
	imageReader          *imageReader
	configReader         *configReader
}

func newEnvReader(
	logger *zap.Logger,
	fetchReader buffetch.Reader,
	configProvider bufconfig.Provider,
	moduleBucketBuilder bufmodulebuild.ModuleBucketBuilder,
	moduleFileSetBuilder bufmodulebuild.ModuleFileSetBuilder,
	imageBuilder bufimagebuild.Builder,
	configOverrideFlagName string,
) *envReader {
	return &envReader{
		logger:               logger.Named("bufwire"),
		fetchReader:          fetchReader,
		moduleBucketBuilder:  moduleBucketBuilder,
		moduleFileSetBuilder: moduleFileSetBuilder,
		imageBuilder:         imageBuilder,
		imageReader: newImageReader(
			logger,
			fetchReader,
		),
		configReader: newConfigReader(
			logger,
			configProvider,
			configOverrideFlagName,
		),
	}
}

func (e *envReader) GetEnv(
	ctx context.Context,
	container app.EnvStdinContainer,
	ref buffetch.Ref,
	configOverride string,
	externalFilePaths []string,
	externalFilePathsAllowNotExist bool,
	excludeSourceCodeInfo bool,
) (_ Env, _ []bufanalysis.FileAnnotation, retErr error) {
	ctx, span := trace.StartSpan(ctx, "get_env")
	defer span.End()
	switch t := ref.(type) {
	case buffetch.ImageRef:
		env, err := e.getImageEnv(
			ctx,
			container,
			t,
			configOverride,
			externalFilePaths,
			externalFilePathsAllowNotExist,
			excludeSourceCodeInfo,
		)
		return env, nil, err
	case buffetch.SourceRef:
		return e.getSourceEnv(
			ctx,
			container,
			t,
			configOverride,
			externalFilePaths,
			externalFilePathsAllowNotExist,
			excludeSourceCodeInfo,
		)
	case buffetch.ModuleRef:
		return e.getModuleEnv(
			ctx,
			container,
			t,
			configOverride,
			externalFilePaths,
			externalFilePathsAllowNotExist,
			excludeSourceCodeInfo,
		)
	default:
		return nil, nil, fmt.Errorf("invalid ref: %T", ref)
	}
}

func (e *envReader) GetSourceOrModuleEnv(
	ctx context.Context,
	container app.EnvStdinContainer,
	sourceOrModuleRef buffetch.SourceOrModuleRef,
	configOverride string,
	externalFilePaths []string,
	externalFilePathsAllowNotExist bool,
	excludeSourceCodeInfo bool,
) (_ Env, _ []bufanalysis.FileAnnotation, retErr error) {
	ctx, span := trace.StartSpan(ctx, "get_source_or_module_env")
	defer span.End()
	switch t := sourceOrModuleRef.(type) {
	case buffetch.SourceRef:
		return e.getSourceEnv(
			ctx,
			container,
			t,
			configOverride,
			externalFilePaths,
			externalFilePathsAllowNotExist,
			excludeSourceCodeInfo,
		)
	case buffetch.ModuleRef:
		return e.getModuleEnv(
			ctx,
			container,
			t,
			configOverride,
			externalFilePaths,
			externalFilePathsAllowNotExist,
			excludeSourceCodeInfo,
		)
	default:
		return nil, nil, fmt.Errorf("invalid ref: %T", sourceOrModuleRef)
	}
}

func (e *envReader) getImageEnv(
	ctx context.Context,
	container app.EnvStdinContainer,
	imageRef buffetch.ImageRef,
	configOverride string,
	externalFilePaths []string,
	externalFilePathsAllowNotExist bool,
	excludeSourceCodeInfo bool,
) (_ Env, retErr error) {
	image, err := e.imageReader.GetImage(
		ctx,
		container,
		imageRef,
		externalFilePaths,
		externalFilePathsAllowNotExist,
		excludeSourceCodeInfo,
	)
	if err != nil {
		return nil, err
	}
	config, err := e.configReader.GetConfig(ctx, configOverride)
	if err != nil {
		return nil, err
	}
	return newEnv(image, config), nil
}

func (e *envReader) getSourceEnv(
	ctx context.Context,
	container app.EnvStdinContainer,
	sourceRef buffetch.SourceRef,
	configOverride string,
	externalFilePaths []string,
	externalFilePathsAllowNotExist bool,
	excludeSourceCodeInfo bool,
) (_ Env, _ []bufanalysis.FileAnnotation, retErr error) {
	readBucketCloser, err := e.fetchReader.GetSourceBucket(ctx, container, sourceRef)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readBucketCloser.Close())
	}()
	config, err := e.configReader.getConfig(ctx, readBucketCloser, configOverride)
	if err != nil {
		return nil, nil, err
	}

	var buildOptions []bufmodulebuild.BuildOption
	if len(externalFilePaths) > 0 {
		bucketRelPaths := make([]string, len(externalFilePaths))
		for i, externalFilePath := range externalFilePaths {
			bucketRelPath, err := sourceRef.PathForExternalPath(externalFilePath)
			if err != nil {
				return nil, nil, err
			}
			bucketRelPaths[i] = bucketRelPath
		}
		if externalFilePathsAllowNotExist {
			buildOptions = append(
				buildOptions,
				bufmodulebuild.WithPathsAllowNotExist(bucketRelPaths),
			)
		} else {
			buildOptions = append(
				buildOptions,
				bufmodulebuild.WithPaths(bucketRelPaths),
			)
		}
	}
	module, err := e.moduleBucketBuilder.BuildForBucket(
		ctx,
		readBucketCloser,
		config.Build,
		buildOptions...,
	)
	if err != nil {
		return nil, nil, err
	}
	return e.buildModule(ctx, module, config, excludeSourceCodeInfo)
}

func (e *envReader) getModuleEnv(
	ctx context.Context,
	container app.EnvStdinContainer,
	moduleRef buffetch.ModuleRef,
	configOverride string,
	externalFilePaths []string,
	externalFilePathsAllowNotExist bool,
	excludeSourceCodeInfo bool,
) (_ Env, _ []bufanalysis.FileAnnotation, retErr error) {
	module, err := e.fetchReader.GetModule(ctx, container, moduleRef)
	if err != nil {
		return nil, nil, err
	}
	if len(externalFilePaths) > 0 {
		targetPaths := make([]string, len(externalFilePaths))
		for i, externalFilePath := range externalFilePaths {
			targetPath, err := moduleRef.PathForExternalPath(externalFilePath)
			if err != nil {
				return nil, nil, err
			}
			targetPaths[i] = targetPath
		}
		if externalFilePathsAllowNotExist {
			module, err = bufmodule.ModuleWithTargetPaths(module, targetPaths)
			if err != nil {
				return nil, nil, err
			}
		} else {
			module, err = bufmodule.ModuleWithTargetPathsAllowNotExist(module, targetPaths)
			if err != nil {
				return nil, nil, err
			}
		}
	}
	// TODO: we should read the config from the module when configuration
	// is added to modules
	config, err := e.configReader.GetConfig(ctx, configOverride)
	if err != nil {
		return nil, nil, err
	}
	return e.buildModule(ctx, module, config, excludeSourceCodeInfo)
}

func (e *envReader) buildModule(
	ctx context.Context,
	module bufmodule.Module,
	config *bufconfig.Config,
	excludeSourceCodeInfo bool,
) (Env, []bufanalysis.FileAnnotation, error) {
	moduleFileSet, err := e.moduleFileSetBuilder.Build(ctx, module)
	if err != nil {
		return nil, nil, err
	}
	var options []bufimagebuild.BuildOption
	if excludeSourceCodeInfo {
		options = append(options, bufimagebuild.WithExcludeSourceCodeInfo())
	}
	image, fileAnnotations, err := e.imageBuilder.Build(
		ctx,
		moduleFileSet,
		options...,
	)
	if err != nil {
		return nil, nil, err
	}
	if len(fileAnnotations) > 0 {
		return nil, fileAnnotations, nil
	}
	return newEnv(image, config), nil, nil
}

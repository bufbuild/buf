// Copyright 2020-2021 Buf Technologies, Inc.
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
	"github.com/bufbuild/buf/internal/buf/bufwork"
	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
)

type imageConfigReader struct {
	logger               *zap.Logger
	storageosProvider    storageos.Provider
	fetchReader          buffetch.Reader
	configProvider       bufconfig.Provider
	moduleBucketBuilder  bufmodulebuild.ModuleBucketBuilder
	moduleFileSetBuilder bufmodulebuild.ModuleFileSetBuilder
	imageBuilder         bufimagebuild.Builder
	moduleConfigReader   *moduleConfigReader
	imageReader          *imageReader
}

func newImageConfigReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	fetchReader buffetch.Reader,
	configProvider bufconfig.Provider,
	workspaceConfigProvider bufwork.Provider,
	moduleBucketBuilder bufmodulebuild.ModuleBucketBuilder,
	moduleFileSetBuilder bufmodulebuild.ModuleFileSetBuilder,
	imageBuilder bufimagebuild.Builder,
) *imageConfigReader {
	return &imageConfigReader{
		logger:               logger.Named("bufwire"),
		storageosProvider:    storageosProvider,
		fetchReader:          fetchReader,
		configProvider:       configProvider,
		moduleBucketBuilder:  moduleBucketBuilder,
		moduleFileSetBuilder: moduleFileSetBuilder,
		imageBuilder:         imageBuilder,
		moduleConfigReader: newModuleConfigReader(
			logger,
			storageosProvider,
			fetchReader,
			configProvider,
			workspaceConfigProvider,
			moduleBucketBuilder,
		),
		imageReader: newImageReader(
			logger,
			fetchReader,
		),
	}
}

func (i *imageConfigReader) GetImageConfigs(
	ctx context.Context,
	container app.EnvStdinContainer,
	ref buffetch.Ref,
	configOverride string,
	externalDirOrFilePaths []string,
	externalDirOrFilePathsAllowNotExist bool,
	excludeSourceCodeInfo bool,
) ([]ImageConfig, []bufanalysis.FileAnnotation, error) {
	switch t := ref.(type) {
	case buffetch.ImageRef:
		env, err := i.getImageImageConfig(
			ctx,
			container,
			t,
			configOverride,
			externalDirOrFilePaths,
			externalDirOrFilePathsAllowNotExist,
			excludeSourceCodeInfo,
		)
		return []ImageConfig{env}, nil, err
	case buffetch.SourceRef:
		return i.GetSourceOrModuleImageConfigs(
			ctx,
			container,
			t,
			configOverride,
			externalDirOrFilePaths,
			externalDirOrFilePathsAllowNotExist,
			excludeSourceCodeInfo,
		)
	case buffetch.ModuleRef:
		return i.GetSourceOrModuleImageConfigs(
			ctx,
			container,
			t,
			configOverride,
			externalDirOrFilePaths,
			externalDirOrFilePathsAllowNotExist,
			excludeSourceCodeInfo,
		)
	default:
		return nil, nil, fmt.Errorf("invalid ref: %T", ref)
	}
}

func (i *imageConfigReader) GetSourceOrModuleImageConfigs(
	ctx context.Context,
	container app.EnvStdinContainer,
	sourceOrModuleRef buffetch.SourceOrModuleRef,
	configOverride string,
	externalDirOrFilePaths []string,
	externalDirOrFilePathsAllowNotExist bool,
	excludeSourceCodeInfo bool,
) ([]ImageConfig, []bufanalysis.FileAnnotation, error) {
	moduleConfigs, err := i.moduleConfigReader.GetModuleConfigs(
		ctx,
		container,
		sourceOrModuleRef,
		configOverride,
		externalDirOrFilePaths,
		externalDirOrFilePathsAllowNotExist,
	)
	if err != nil {
		return nil, nil, err
	}
	imageConfigs := make([]ImageConfig, 0, len(moduleConfigs))
	var allFileAnnotations []bufanalysis.FileAnnotation
	for _, moduleConfig := range moduleConfigs {
		imageConfig, fileAnnotations, err := i.buildModule(
			ctx,
			moduleConfig.Module(),
			moduleConfig.Config(),
			moduleConfig.Workspace(),
			excludeSourceCodeInfo,
		)
		if err != nil {
			return nil, nil, err
		}
		if imageConfig != nil {
			imageConfigs = append(imageConfigs, imageConfig)
		}
		allFileAnnotations = append(allFileAnnotations, fileAnnotations...)
	}
	if len(allFileAnnotations) > 0 {
		// Deduplicate and sort the file annotations again now that we've
		// consolidated them across multiple images.
		deduplicated, err := bufanalysis.DeduplicateAndSortFileAnnotations(allFileAnnotations)
		if err != nil {
			return nil, nil, err
		}
		return nil, deduplicated, nil
	}
	return imageConfigs, nil, nil
}

func (i *imageConfigReader) getImageImageConfig(
	ctx context.Context,
	container app.EnvStdinContainer,
	imageRef buffetch.ImageRef,
	configOverride string,
	externalDirOrFilePaths []string,
	externalDirOrFilePathsAllowNotExist bool,
	excludeSourceCodeInfo bool,
) (_ ImageConfig, retErr error) {
	image, err := i.imageReader.GetImage(
		ctx,
		container,
		imageRef,
		externalDirOrFilePaths,
		externalDirOrFilePathsAllowNotExist,
		excludeSourceCodeInfo,
	)
	if err != nil {
		return nil, err
	}
	readWriteBucket, err := i.storageosProvider.NewReadWriteBucket(
		".",
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return nil, err
	}
	config, err := bufconfig.ReadConfig(
		ctx,
		i.configProvider,
		readWriteBucket,
		bufconfig.ReadConfigWithOverride(configOverride),
	)
	if err != nil {
		return nil, err
	}
	return newImageConfig(image, config), nil
}

func (i *imageConfigReader) buildModule(
	ctx context.Context,
	module bufmodule.Module,
	config *bufconfig.Config,
	workspace bufmodule.Workspace,
	excludeSourceCodeInfo bool,
) (ImageConfig, []bufanalysis.FileAnnotation, error) {
	ctx, span := trace.StartSpan(ctx, "build_module")
	defer span.End()
	moduleFileSet, err := i.moduleFileSetBuilder.Build(ctx, module, bufmodulebuild.WithWorkspace(workspace))
	if err != nil {
		return nil, nil, err
	}
	var options []bufimagebuild.BuildOption
	if excludeSourceCodeInfo {
		options = append(options, bufimagebuild.WithExcludeSourceCodeInfo())
	}
	image, fileAnnotations, err := i.imageBuilder.Build(
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
	return newImageConfig(image, config), nil, nil
}

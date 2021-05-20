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

	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/internal/buf/buffetch"
	"github.com/bufbuild/buf/internal/buf/bufwork"
	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"go.opencensus.io/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type moduleConfigReader struct {
	logger                  *zap.Logger
	storageosProvider       storageos.Provider
	fetchReader             buffetch.Reader
	configProvider          bufconfig.Provider
	workspaceConfigProvider bufwork.Provider
	moduleBucketBuilder     bufmodulebuild.ModuleBucketBuilder
}

func newModuleConfigReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	fetchReader buffetch.Reader,
	configProvider bufconfig.Provider,
	workspaceConfigProvider bufwork.Provider,
	moduleBucketBuilder bufmodulebuild.ModuleBucketBuilder,
) *moduleConfigReader {
	return &moduleConfigReader{
		logger:                  logger.Named("bufwire"),
		storageosProvider:       storageosProvider,
		fetchReader:             fetchReader,
		configProvider:          configProvider,
		workspaceConfigProvider: workspaceConfigProvider,
		moduleBucketBuilder:     moduleBucketBuilder,
	}
}

func (m *moduleConfigReader) GetModuleConfigs(
	ctx context.Context,
	container app.EnvStdinContainer,
	sourceOrModuleRef buffetch.SourceOrModuleRef,
	configOverride string,
	externalDirOrFilePaths []string,
	externalDirOrFilePathsAllowNotExist bool,
) ([]ModuleConfig, error) {
	ctx, span := trace.StartSpan(ctx, "get_module_config")
	defer span.End()
	switch t := sourceOrModuleRef.(type) {
	case buffetch.SourceRef:
		return m.getSourceModuleConfigs(
			ctx,
			container,
			t,
			configOverride,
			externalDirOrFilePaths,
			externalDirOrFilePathsAllowNotExist,
		)
	case buffetch.ModuleRef:
		moduleConfig, err := m.getModuleModuleConfig(
			ctx,
			container,
			t,
			configOverride,
			externalDirOrFilePaths,
			externalDirOrFilePathsAllowNotExist,
		)
		if err != nil {
			return nil, err
		}
		return []ModuleConfig{
			moduleConfig,
		}, nil
	default:
		return nil, fmt.Errorf("invalid ref: %T", sourceOrModuleRef)
	}
}

func (m *moduleConfigReader) getSourceModuleConfigs(
	ctx context.Context,
	container app.EnvStdinContainer,
	sourceRef buffetch.SourceRef,
	configOverride string,
	externalDirOrFilePaths []string,
	externalDirOrFilePathsAllowNotExist bool,
) (_ []ModuleConfig, retErr error) {
	var buildOptions []bufmodulebuild.BuildOption
	if len(externalDirOrFilePaths) > 0 {
		bucketRelPaths := make([]string, len(externalDirOrFilePaths))
		for i, externalDirOrFilePath := range externalDirOrFilePaths {
			bucketRelPath, err := sourceRef.PathForExternalPath(externalDirOrFilePath)
			if err != nil {
				return nil, err
			}
			bucketRelPaths[i] = bucketRelPath
		}
		if externalDirOrFilePathsAllowNotExist {
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
	readBucketCloser, err := m.fetchReader.GetSourceBucket(ctx, container, sourceRef)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readBucketCloser.Close())
	}()
	exists, err := storage.Exists(ctx, readBucketCloser, bufwork.ExternalConfigV1Beta1FilePath)
	if err != nil {
		return nil, err
	}
	if exists {
		return m.getWorkspaceModuleConfigs(
			ctx,
			readBucketCloser,
			readBucketCloser.RelativeRootPath(),
			readBucketCloser.SubDirPath(),
			configOverride,
			buildOptions...,
		)
	}
	moduleConfig, err := m.getSourceModuleConfig(
		ctx,
		readBucketCloser,
		readBucketCloser.SubDirPath(),
		configOverride,
		nil,
		buildOptions...,
	)
	if err != nil {
		return nil, err
	}
	return []ModuleConfig{
		moduleConfig,
	}, nil
}

func (m *moduleConfigReader) getModuleModuleConfig(
	ctx context.Context,
	container app.EnvStdinContainer,
	moduleRef buffetch.ModuleRef,
	configOverride string,
	externalDirOrFilePaths []string,
	externalDirOrFilePathsAllowNotExist bool,
) (_ ModuleConfig, retErr error) {
	module, err := m.fetchReader.GetModule(ctx, container, moduleRef)
	if err != nil {
		return nil, err
	}
	if len(externalDirOrFilePaths) > 0 {
		targetPaths := make([]string, len(externalDirOrFilePaths))
		for i, externalDirOrFilePath := range externalDirOrFilePaths {
			targetPath, err := moduleRef.PathForExternalPath(externalDirOrFilePath)
			if err != nil {
				return nil, err
			}
			targetPaths[i] = targetPath
		}
		if externalDirOrFilePathsAllowNotExist {
			module, err = bufmodule.ModuleWithTargetPaths(module, targetPaths)
			if err != nil {
				return nil, err
			}
		} else {
			module, err = bufmodule.ModuleWithTargetPathsAllowNotExist(module, targetPaths)
			if err != nil {
				return nil, err
			}
		}
	}
	// TODO: we should read the config from the module when configuration
	// is added to modules
	readWriteBucket, err := m.storageosProvider.NewReadWriteBucket(
		".",
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return nil, err
	}
	config, err := bufconfig.ReadConfig(
		ctx,
		m.configProvider,
		readWriteBucket,
		bufconfig.ReadConfigWithOverride(configOverride),
	)
	if err != nil {
		return nil, err
	}
	return newModuleConfig(module, config, nil /* Workspaces aren't supported for ModuleRefs */), nil
}

func (m *moduleConfigReader) getWorkspaceModuleConfigs(
	ctx context.Context,
	readBucket storage.ReadBucket,
	relativeRootPath string,
	subDirPath string,
	configOverride string,
	buildOptions ...bufmodulebuild.BuildOption,
) ([]ModuleConfig, error) {
	workspaceConfig, err := m.workspaceConfigProvider.GetConfig(ctx, readBucket, relativeRootPath)
	if err != nil {
		return nil, err
	}
	if subDirPath != "." {
		// There's only a single ModuleConfig based on the subDirPath,
		// so we only need to create a single workspace.
		workspace, err := bufwork.NewWorkspace(ctx, workspaceConfig, readBucket, m.configProvider, relativeRootPath, subDirPath)
		if err != nil {
			return nil, err
		}
		moduleConfig, err := m.getSourceModuleConfig(
			ctx,
			readBucket,
			subDirPath,
			configOverride,
			workspace,
			buildOptions...,
		)
		if err != nil {
			return nil, err
		}
		return []ModuleConfig{
			moduleConfig,
		}, nil
	}
	// The target subDirPath points to the workspace configuration,
	// so we construct a separate workspace for each of the configured
	// directories.
	var moduleConfigs []ModuleConfig
	for _, directory := range workspaceConfig.Directories {
		// TODO: We need to construct a separate workspace for each module,
		// but this is fairly duplicative in its current state. Specifically,
		// we build the same module multiple times.
		//
		// We can refactor this with a bufworkbuild.WorkspaceBuilder that
		// caches modules so that workspace modules are only ever built once.
		workspace, err := bufwork.NewWorkspace(ctx, workspaceConfig, readBucket, m.configProvider, relativeRootPath, directory)
		if err != nil {
			return nil, err
		}
		moduleConfig, err := m.getSourceModuleConfig(
			ctx,
			readBucket,
			directory,
			configOverride,
			workspace,
			buildOptions...,
		)
		if err != nil {
			return nil, err
		}
		moduleConfigs = append(moduleConfigs, moduleConfig)
	}
	return moduleConfigs, nil
}

func (m *moduleConfigReader) getSourceModuleConfig(
	ctx context.Context,
	readBucket storage.ReadBucket,
	subDirPath string,
	configOverride string,
	workspace bufmodule.Workspace,
	buildOptions ...bufmodulebuild.BuildOption,
) (ModuleConfig, error) {
	mappedReadBucket := readBucket
	if subDirPath != "." {
		mappedReadBucket = storage.MapReadBucket(readBucket, storage.MapOnPrefix(subDirPath))
	}
	config, err := bufconfig.ReadConfig(
		ctx,
		m.configProvider,
		mappedReadBucket,
		bufconfig.ReadConfigWithOverride(configOverride),
	)
	if err != nil {
		return nil, err
	}
	module, err := m.moduleBucketBuilder.BuildForBucket(
		ctx,
		mappedReadBucket,
		config.Build,
		buildOptions...,
	)
	if err != nil {
		return nil, err
	}
	return newModuleConfig(module, config, workspace), nil
}

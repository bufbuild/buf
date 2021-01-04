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
	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"go.opencensus.io/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type moduleConfigReader struct {
	logger              *zap.Logger
	storageosProvider   storageos.Provider
	fetchReader         buffetch.Reader
	configProvider      bufconfig.Provider
	moduleBucketBuilder bufmodulebuild.ModuleBucketBuilder
}

func newModuleConfigReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	fetchReader buffetch.Reader,
	configProvider bufconfig.Provider,
	moduleBucketBuilder bufmodulebuild.ModuleBucketBuilder,
) *moduleConfigReader {
	return &moduleConfigReader{
		logger:              logger.Named("bufwire"),
		storageosProvider:   storageosProvider,
		fetchReader:         fetchReader,
		configProvider:      configProvider,
		moduleBucketBuilder: moduleBucketBuilder,
	}
}

func (m *moduleConfigReader) GetModuleConfig(
	ctx context.Context,
	container app.EnvStdinContainer,
	sourceOrModuleRef buffetch.SourceOrModuleRef,
	configOverride string,
	externalDirOrFilePaths []string,
	externalDirOrFilePathsAllowNotExist bool,
) (ModuleConfig, error) {
	ctx, span := trace.StartSpan(ctx, "get_module_config")
	defer span.End()
	switch t := sourceOrModuleRef.(type) {
	case buffetch.SourceRef:
		return m.getSourceModuleConfig(
			ctx,
			container,
			t,
			configOverride,
			externalDirOrFilePaths,
			externalDirOrFilePathsAllowNotExist,
		)
	case buffetch.ModuleRef:
		return m.getModuleModuleConfig(
			ctx,
			container,
			t,
			configOverride,
			externalDirOrFilePaths,
			externalDirOrFilePathsAllowNotExist,
		)
	default:
		return nil, fmt.Errorf("invalid ref: %T", sourceOrModuleRef)
	}
}

func (m *moduleConfigReader) getSourceModuleConfig(
	ctx context.Context,
	container app.EnvStdinContainer,
	sourceRef buffetch.SourceRef,
	configOverride string,
	externalDirOrFilePaths []string,
	externalDirOrFilePathsAllowNotExist bool,
) (_ ModuleConfig, retErr error) {
	readBucketCloser, err := m.fetchReader.GetSourceBucket(ctx, container, sourceRef)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readBucketCloser.Close())
	}()
	config, err := bufconfig.ReadConfig(
		ctx,
		m.configProvider,
		readBucketCloser,
		bufconfig.ReadConfigWithOverride(configOverride),
	)
	if err != nil {
		return nil, err
	}

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
	module, err := m.moduleBucketBuilder.BuildForBucket(
		ctx,
		readBucketCloser,
		config.Build,
		buildOptions...,
	)
	if err != nil {
		return nil, err
	}

	return newModuleConfig(module, config), nil
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
	return newModuleConfig(module, config), nil
}

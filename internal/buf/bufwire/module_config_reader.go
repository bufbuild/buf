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
	fetchReader         buffetch.Reader
	configProvider      bufconfig.Provider
	moduleBucketBuilder bufmodulebuild.ModuleBucketBuilder
}

func newModuleConfigReader(
	logger *zap.Logger,
	fetchReader buffetch.Reader,
	configProvider bufconfig.Provider,
	moduleBucketBuilder bufmodulebuild.ModuleBucketBuilder,
) *moduleConfigReader {
	return &moduleConfigReader{
		logger:              logger.Named("bufwire"),
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
	externalFilePaths []string,
	externalFilePathsAllowNotExist bool,
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
			externalFilePaths,
			externalFilePathsAllowNotExist,
		)
	case buffetch.ModuleRef:
		return m.getModuleModuleConfig(
			ctx,
			container,
			t,
			configOverride,
			externalFilePaths,
			externalFilePathsAllowNotExist,
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
	externalFilePaths []string,
	externalFilePathsAllowNotExist bool,
) (_ ModuleConfig, retErr error) {
	readBucketCloser, err := m.fetchReader.GetSourceBucket(ctx, container, sourceRef)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readBucketCloser.Close())
	}()
	config, err := bufconfig.ReadConfig(ctx, m.configProvider, readBucketCloser, configOverride)
	if err != nil {
		return nil, err
	}

	var buildOptions []bufmodulebuild.BuildOption
	if len(externalFilePaths) > 0 {
		bucketRelPaths := make([]string, len(externalFilePaths))
		for i, externalFilePath := range externalFilePaths {
			bucketRelPath, err := sourceRef.PathForExternalPath(externalFilePath)
			if err != nil {
				return nil, err
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
	externalFilePaths []string,
	externalFilePathsAllowNotExist bool,
) (_ ModuleConfig, retErr error) {
	module, err := m.fetchReader.GetModule(ctx, container, moduleRef)
	if err != nil {
		return nil, err
	}
	if len(externalFilePaths) > 0 {
		targetPaths := make([]string, len(externalFilePaths))
		for i, externalFilePath := range externalFilePaths {
			targetPath, err := moduleRef.PathForExternalPath(externalFilePath)
			if err != nil {
				return nil, err
			}
			targetPaths[i] = targetPath
		}
		if externalFilePathsAllowNotExist {
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
	readWriteBucket, err := storageos.NewReadWriteBucket(".")
	if err != nil {
		return nil, err
	}
	config, err := bufconfig.ReadConfig(ctx, m.configProvider, readWriteBucket, configOverride)
	if err != nil {
		return nil, err
	}
	return newModuleConfig(module, config), nil
}

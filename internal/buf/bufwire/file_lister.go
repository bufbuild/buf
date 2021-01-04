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
	"github.com/bufbuild/buf/internal/buf/bufcore"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/internal/buf/buffetch"
	"github.com/bufbuild/buf/internal/pkg/app"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type fileLister struct {
	logger              *zap.Logger
	fetchReader         buffetch.Reader
	configProvider      bufconfig.Provider
	moduleBucketBuilder bufmodulebuild.ModuleBucketBuilder
	imageBuilder        bufimagebuild.Builder
	imageReader         *imageReader
}

func newFileLister(
	logger *zap.Logger,
	fetchReader buffetch.Reader,
	configProvider bufconfig.Provider,
	moduleBucketBuilder bufmodulebuild.ModuleBucketBuilder,
	imageBuilder bufimagebuild.Builder,
) *fileLister {
	return &fileLister{
		logger:              logger.Named("bufwire"),
		fetchReader:         fetchReader,
		configProvider:      configProvider,
		moduleBucketBuilder: moduleBucketBuilder,
		imageBuilder:        imageBuilder,
		imageReader: newImageReader(
			logger,
			fetchReader,
		),
	}
}

func (e *fileLister) ListFiles(
	ctx context.Context,
	container app.EnvStdinContainer,
	ref buffetch.Ref,
	configOverride string,
) (_ []bufcore.FileInfo, retErr error) {
	switch t := ref.(type) {
	case buffetch.ImageRef:
		// if we have an image, list the files in the image
		image, err := e.imageReader.GetImage(
			ctx,
			container,
			t,
			nil,
			false,
			true,
		)
		if err != nil {
			return nil, err
		}
		files := image.Files()
		fileInfos := make([]bufcore.FileInfo, len(files))
		for i, file := range files {
			fileInfos[i] = file
		}
		return fileInfos, nil
	case buffetch.SourceRef:
		readBucketCloser, err := e.fetchReader.GetSourceBucket(ctx, container, t)
		if err != nil {
			return nil, err
		}
		defer func() {
			retErr = multierr.Append(retErr, readBucketCloser.Close())
		}()
		config, err := bufconfig.ReadConfig(
			ctx,
			e.configProvider,
			readBucketCloser,
			bufconfig.ReadConfigWithOverride(configOverride),
		)
		if err != nil {
			return nil, err
		}
		module, err := e.moduleBucketBuilder.BuildForBucket(
			ctx,
			readBucketCloser,
			config.Build,
		)
		if err != nil {
			return nil, err
		}
		return module.SourceFileInfos(ctx)
	case buffetch.ModuleRef:
		module, err := e.fetchReader.GetModule(ctx, container, t)
		if err != nil {
			return nil, err
		}
		return module.SourceFileInfos(ctx)
	default:
		return nil, fmt.Errorf("invalid ref: %T", ref)
	}
}

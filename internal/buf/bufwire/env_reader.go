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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufbuild"
	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/bufcore"
	"github.com/bufbuild/buf/internal/buf/buffetch"
	"github.com/bufbuild/buf/internal/buf/bufmod"
	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.opencensus.io/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type envReader struct {
	logger                 *zap.Logger
	fetchRefParser         buffetch.RefParser
	fetchReader            buffetch.Reader
	configProvider         bufconfig.Provider
	modBucketBuilder       bufmod.BucketBuilder
	buildBuilder           bufbuild.Builder
	imageReader            *imageReader
	valueFlagName          string
	configOverrideFlagName string
}

func newEnvReader(
	logger *zap.Logger,
	fetchRefParser buffetch.RefParser,
	fetchReader buffetch.Reader,
	configProvider bufconfig.Provider,
	modBucketBuilder bufmod.BucketBuilder,
	buildBuilder bufbuild.Builder,
	valueFlagName string,
	configOverrideFlagName string,
) *envReader {
	return &envReader{
		logger:           logger.Named("bufwire"),
		fetchRefParser:   fetchRefParser,
		fetchReader:      fetchReader,
		configProvider:   configProvider,
		modBucketBuilder: modBucketBuilder,
		buildBuilder:     buildBuilder,
		imageReader: newImageReader(
			logger,
			fetchRefParser,
			fetchReader,
			valueFlagName,
		),
		valueFlagName:          valueFlagName,
		configOverrideFlagName: configOverrideFlagName,
	}
}

func (e *envReader) GetEnv(
	ctx context.Context,
	container app.EnvStdinContainer,
	value string,
	configOverride string,
	externalFilePaths []string,
	externalFilePathsAllowNotExist bool,
	excludeSourceCodeInfo bool,
) (_ Env, _ []bufanalysis.FileAnnotation, retErr error) {
	ctx, span := trace.StartSpan(ctx, "get_env")
	defer span.End()
	defer func() {
		if retErr != nil {
			retErr = fmt.Errorf("%v: %w", e.valueFlagName, retErr)
		}
	}()

	ref, err := e.fetchRefParser.GetRef(ctx, value)
	if err != nil {
		return nil, nil, err
	}
	switch t := ref.(type) {
	case buffetch.ImageRef:
		env, err := e.getEnvFromImage(
			ctx,
			container,
			configOverride,
			externalFilePaths,
			externalFilePathsAllowNotExist,
			excludeSourceCodeInfo,
			t,
		)
		return env, nil, err
	case buffetch.SourceRef:
		return e.getEnvFromSource(
			ctx,
			container,
			configOverride,
			externalFilePaths,
			externalFilePathsAllowNotExist,
			excludeSourceCodeInfo,
			t,
		)
	default:
		return nil, nil, fmt.Errorf("invalid ref: %T", ref)
	}
}

func (e *envReader) GetImageEnv(
	ctx context.Context,
	container app.EnvStdinContainer,
	value string,
	configOverride string,
	externalFilePaths []string,
	externalFilePathsAllowNotExist bool,
	excludeSourceCodeInfo bool,
) (_ Env, retErr error) {
	ctx, span := trace.StartSpan(ctx, "get_image_env")
	defer span.End()
	defer func() {
		if retErr != nil {
			retErr = fmt.Errorf("%v: %w", e.valueFlagName, retErr)
		}
	}()

	imageRef, err := e.fetchRefParser.GetImageRef(ctx, value)
	if err != nil {
		return nil, err
	}
	return e.getEnvFromImage(
		ctx,
		container,
		configOverride,
		externalFilePaths,
		externalFilePathsAllowNotExist,
		excludeSourceCodeInfo,
		imageRef,
	)
}

func (e *envReader) GetSourceEnv(
	ctx context.Context,
	container app.EnvStdinContainer,
	value string,
	configOverride string,
	externalFilePaths []string,
	externalFilePathsAllowNotExist bool,
	excludeSourceCodeInfo bool,
) (_ Env, _ []bufanalysis.FileAnnotation, retErr error) {
	ctx, span := trace.StartSpan(ctx, "get_source_env")
	defer span.End()
	defer func() {
		if retErr != nil {
			retErr = fmt.Errorf("%v: %w", e.valueFlagName, retErr)
		}
	}()

	sourceRef, err := e.fetchRefParser.GetSourceRef(ctx, value)
	if err != nil {
		return nil, nil, err
	}
	return e.getEnvFromSource(
		ctx,
		container,
		configOverride,
		externalFilePaths,
		externalFilePathsAllowNotExist,
		excludeSourceCodeInfo,
		sourceRef,
	)
}

func (e *envReader) ListFiles(
	ctx context.Context,
	container app.EnvStdinContainer,
	value string,
	configOverride string,
) (_ []bufcore.FileInfo, retErr error) {
	defer func() {
		if retErr != nil {
			retErr = fmt.Errorf("%v: %w", e.valueFlagName, retErr)
		}
	}()
	ref, err := e.fetchRefParser.GetRef(ctx, value)
	if err != nil {
		return nil, err
	}
	switch t := ref.(type) {
	case buffetch.ImageRef:
		// if we have an image, list the files in the image
		image, err := e.imageReader.getImageForImageRef(
			ctx,
			container,
			nil,
			false,
			true,
			t,
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
		readBucketCloser, config, err := e.getSourceBucketAndConfig(ctx, container, t, configOverride)
		if err != nil {
			return nil, err
		}
		defer func() {
			retErr = multierr.Append(retErr, readBucketCloser.Close())
		}()
		module, err := e.modBucketBuilder.BuildForBucket(
			ctx,
			readBucketCloser,
			config.Build,
		)
		if err != nil {
			return nil, err
		}
		return module.TargetFileInfos(ctx)
	default:
		return nil, fmt.Errorf("invalid ref: %T", ref)
	}
}

func (e *envReader) GetConfig(
	ctx context.Context,
	configOverride string,
) (*bufconfig.Config, error) {
	if configOverride != "" {
		return e.parseConfigOverride(ctx, configOverride)
	}
	// if there is no config override, we read the config from the current directory
	data, err := ioutil.ReadFile(bufconfig.ConfigFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		// just in case
		data = nil
	}
	// if there was no file, this just returns default config
	return e.configProvider.GetConfigForData(ctx, data)
}

func (e *envReader) getEnvFromImage(
	ctx context.Context,
	container app.EnvStdinContainer,
	configOverride string,
	externalFilePaths []string,
	externalFilePathsAllowNotExist bool,
	excludeSourceCodeInfo bool,
	imageRef buffetch.ImageRef,
) (_ Env, retErr error) {
	image, err := e.imageReader.getImageForImageRef(
		ctx,
		container,
		externalFilePaths,
		externalFilePathsAllowNotExist,
		excludeSourceCodeInfo,
		imageRef,
	)
	if err != nil {
		return nil, err
	}
	config, err := e.GetConfig(ctx, configOverride)
	if err != nil {
		return nil, err
	}
	return newEnv(image, config), nil
}

func (e *envReader) getEnvFromSource(
	ctx context.Context,
	container app.EnvStdinContainer,
	configOverride string,
	externalFilePaths []string,
	externalFilePathsAllowNotExist bool,
	excludeSourceCodeInfo bool,
	sourceRef buffetch.SourceRef,
) (_ Env, _ []bufanalysis.FileAnnotation, retErr error) {
	readBucketCloser, config, err := e.getSourceBucketAndConfig(ctx, container, sourceRef, configOverride)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readBucketCloser.Close())
	}()

	var buildOptions []bufmod.BuildOption
	if len(externalFilePaths) > 0 {
		bucketRelPaths := make([]string, len(externalFilePaths))
		for i, externalFilePath := range externalFilePaths {
			bucketRelPath, err := sourceRef.PathForExternalPath(externalFilePath)
			if err != nil {
				return nil, nil, err
			}
			bucketRelPaths[i] = bucketRelPath
		}
		buildOptions = append(
			buildOptions,
			bufmod.WithPaths(bucketRelPaths...),
		)
	}
	if externalFilePathsAllowNotExist {
		buildOptions = append(
			buildOptions,
			bufmod.WithPathsAllowNotExistOnWalk(),
		)
	}
	module, err := e.modBucketBuilder.BuildForBucket(
		ctx,
		readBucketCloser,
		config.Build,
		buildOptions...,
	)
	if err != nil {
		return nil, nil, err
	}
	var options []bufbuild.BuildOption
	if excludeSourceCodeInfo {
		options = append(options, bufbuild.WithExcludeSourceCodeInfo())
	}
	image, fileAnnotations, err := e.buildBuilder.Build(
		ctx,
		module,
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

func (e *envReader) getSourceBucketAndConfig(
	ctx context.Context,
	container app.EnvStdinContainer,
	sourceRef buffetch.SourceRef,
	configOverride string,
) (_ storage.ReadBucketCloser, _ *bufconfig.Config, retErr error) {
	readBucketCloser, err := e.fetchReader.GetSourceBucket(ctx, container, sourceRef)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if retErr != nil {
			retErr = multierr.Append(retErr, readBucketCloser.Close())
		}
	}()
	var config *bufconfig.Config
	if configOverride != "" {
		config, err = e.parseConfigOverride(ctx, configOverride)
	} else {
		// if there is no config override, we read the config from the bucket
		// if there was no file, this just returns default config
		config, err = e.configProvider.GetConfig(ctx, readBucketCloser)
	}
	if err != nil {
		return nil, nil, err
	}
	return readBucketCloser, config, nil
}

func (e *envReader) parseConfigOverride(ctx context.Context, value string) (*bufconfig.Config, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("config override value is empty")
	}
	var data []byte
	var err error
	switch filepath.Ext(value) {
	case ".json", ".yaml":
		data, err = ioutil.ReadFile(value)
		if err != nil {
			return nil, fmt.Errorf("%s: could not read file: %v", e.configOverrideFlagName, err)
		}
	default:
		data = []byte(value)
	}
	config, err := e.configProvider.GetConfigForData(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", e.configOverrideFlagName, err)
	}
	return config, nil
}

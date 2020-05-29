// Copyright 2020 Buf Technologies Inc.
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

package bufcli

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
	"github.com/bufbuild/buf/internal/buf/buffetch"
	"github.com/bufbuild/buf/internal/buf/bufimage"
	imagev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/image/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/instrument"
	"github.com/bufbuild/buf/internal/pkg/proto/protoencoding"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type envReader struct {
	logger                 *zap.Logger
	fetchRefParser         buffetch.RefParser
	fetchReader            buffetch.Reader
	configProvider         bufconfig.Provider
	buildFileRefProvider   bufbuild.FileRefProvider
	buildBuilder           bufbuild.Builder
	valueFlagName          string
	configOverrideFlagName string
}

func newEnvReader(
	logger *zap.Logger,
	fetchRefParser buffetch.RefParser,
	fetchReader buffetch.Reader,
	configProvider bufconfig.Provider,
	buildFileRefProvider bufbuild.FileRefProvider,
	buildBuilder bufbuild.Builder,
	valueFlagName string,
	configOverrideFlagName string,
) *envReader {
	return &envReader{
		logger:                 logger,
		fetchRefParser:         fetchRefParser,
		fetchReader:            fetchReader,
		configProvider:         configProvider,
		buildFileRefProvider:   buildFileRefProvider,
		buildBuilder:           buildBuilder,
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
	defer instrument.Start(e.logger, "get_env").End()
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
) (_ Env, retErr error) {
	defer instrument.Start(e.logger, "get_image_env").End()
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
	defer instrument.Start(e.logger, "get_source_env").End()
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
) (_ []bufimage.FileRef, retErr error) {
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
		image, err := e.getImage(ctx, container, t)
		if err != nil {
			return nil, err
		}
		files := image.Files()
		fileRefs := make([]bufimage.FileRef, len(files))
		for i, file := range files {
			fileRefs[i] = file
		}
		return fileRefs, nil
	case buffetch.SourceRef:
		readBucketCloser, config, err := e.getSourceBucketAndConfig(ctx, container, t, configOverride)
		if err != nil {
			return nil, err
		}
		defer func() {
			retErr = multierr.Append(retErr, readBucketCloser.Close())
		}()
		return e.buildFileRefProvider.GetAllFileRefs(
			ctx,
			readBucketCloser,
			t.PathResolver(),
			config.Build.Roots,
			config.Build.Excludes,
		)
	default:
		return nil, fmt.Errorf("invalid ref: %T", ref)
	}
}

func (e *envReader) GetConfig(
	ctx context.Context,
	configOverride string,
) (*bufconfig.Config, error) {
	if configOverride != "" {
		return e.parseConfigOverride(configOverride)
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
	return e.configProvider.GetConfigForData(data)
}

func (e *envReader) getEnvFromImage(
	ctx context.Context,
	container app.EnvStdinContainer,
	configOverride string,
	externalFilePaths []string,
	externalFilePathsAllowNotExist bool,
	imageRef buffetch.ImageRef,
) (_ Env, retErr error) {
	image, err := e.getImage(ctx, container, imageRef)
	if err != nil {
		return nil, err
	}
	config, err := e.GetConfig(ctx, configOverride)
	if err != nil {
		return nil, err
	}
	if len(externalFilePaths) > 0 {
		// this is usually a full rel file path, but because this is a read image,
		// the root is always ".", so this is OK, although awkward
		rootRelFilePaths := make([]string, len(externalFilePaths))
		for i, externalFilePath := range externalFilePaths {
			rootRelFilePath, err := imageRef.PathResolver().ExternalPathToRelPath(externalFilePath)
			if err != nil {
				return nil, err
			}
			rootRelFilePaths[i] = rootRelFilePath
		}
		if externalFilePathsAllowNotExist {
			image, err = bufimage.ImageWithOnlyRootRelFilePathsAllowNotExist(image, rootRelFilePaths)
		} else {
			image, err = bufimage.ImageWithOnlyRootRelFilePaths(image, rootRelFilePaths)
		}
		if err != nil {
			return nil, err
		}
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

	var fileRefs []bufimage.FileRef
	if len(externalFilePaths) == 0 {
		fileRefs, err = e.buildFileRefProvider.GetAllFileRefs(
			ctx,
			readBucketCloser,
			sourceRef.PathResolver(),
			config.Build.Roots,
			config.Build.Excludes,
		)
		if err != nil {
			return nil, nil, err
		}
	} else {
		var options []bufbuild.GetFileRefsForExternalFilePathsOption
		if externalFilePathsAllowNotExist {
			options = append(options, bufbuild.WithAllowNotExist())
		}
		fileRefs, err = e.buildFileRefProvider.GetFileRefsForExternalFilePaths(
			ctx,
			readBucketCloser,
			sourceRef.PathResolver(),
			config.Build.Roots,
			externalFilePaths,
			options...,
		)
		if err != nil {
			return nil, nil, err
		}
	}

	var options []bufbuild.BuildOption
	if excludeSourceCodeInfo {
		options = append(options, bufbuild.WithExcludeSourceCodeInfo())
	}
	image, fileAnnotations, err := e.buildBuilder.Build(
		ctx,
		readBucketCloser,
		sourceRef.PathResolver(),
		fileRefs,
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

func (e *envReader) getImage(
	ctx context.Context,
	container app.EnvStdinContainer,
	imageRef buffetch.ImageRef,
) (_ bufimage.Image, retErr error) {
	readCloser, err := e.fetchReader.GetImageFile(ctx, container, imageRef)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readCloser.Close())
	}()
	data, err := ioutil.ReadAll(readCloser)
	if err != nil {
		return nil, err
	}
	// we cannot determine fileDescriptorProtos ahead of time so we cannot handle extensions
	// TODO: we do not happen to need them for our use case with linting, but we need to dicuss this
	protoImage := &imagev1beta1.Image{}
	switch imageEncoding := imageRef.ImageEncoding(); imageEncoding {
	case buffetch.ImageEncodingBin:
		err = protoencoding.NewWireUnmarshaler(nil).Unmarshal(data, protoImage)
	case buffetch.ImageEncodingJSON:
		err = protoencoding.NewJSONUnmarshaler(nil).Unmarshal(data, protoImage)
	default:
		return nil, fmt.Errorf("unknown image encoding: %v", imageEncoding)
	}
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal Image: %v", err)
	}
	return bufimage.NewImageForProto(protoImage)
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
		config, err = e.parseConfigOverride(configOverride)
	} else {
		// if there is no config override, we read the config from the bucket
		// if there was no file, this just returns default config
		config, err = e.configProvider.GetConfigForReadBucket(ctx, readBucketCloser)
	}
	if err != nil {
		return nil, nil, err
	}
	return readBucketCloser, config, nil
}

func (e *envReader) parseConfigOverride(value string) (*bufconfig.Config, error) {
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
	config, err := e.configProvider.GetConfigForData(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", e.configOverrideFlagName, err)
	}
	return config, nil
}

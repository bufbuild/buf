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

package bufos

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bufbuild/buf/internal/buf/bufbuild"
	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/buffetch"
	"github.com/bufbuild/buf/internal/buf/ext/extimage"
	filev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/file/v1beta1"
	imagev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/image/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/instrument"
	"github.com/bufbuild/buf/internal/pkg/proto/protoencoding"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type envReader struct {
	logger                 *zap.Logger
	fetchRefParser         buffetch.RefParser
	fetchReader            buffetch.Reader
	configProvider         bufconfig.Provider
	buildHandler           bufbuild.Handler
	valueFlagName          string
	configOverrideFlagName string
}

func newEnvReader(
	logger *zap.Logger,
	fetchRefParser buffetch.RefParser,
	fetchReader buffetch.Reader,
	configProvider bufconfig.Provider,
	buildHandler bufbuild.Handler,
	valueFlagName string,
	configOverrideFlagName string,
) *envReader {
	return &envReader{
		logger:                 logger.Named("bufos"),
		fetchRefParser:         fetchRefParser,
		fetchReader:            fetchReader,
		configProvider:         configProvider,
		buildHandler:           buildHandler,
		valueFlagName:          valueFlagName,
		configOverrideFlagName: configOverrideFlagName,
	}
}

func (e *envReader) ReadEnv(
	ctx context.Context,
	container app.EnvStdinContainer,
	value string,
	configOverride string,
	specificFilePaths []string,
	specificFilePathsAllowNotExist bool,
	includeImports bool,
	includeSourceInfo bool,
) (_ *Env, _ []*filev1beta1.FileAnnotation, retErr error) {
	defer instrument.Start(e.logger, "read_env").End()
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
		env, err := e.readEnvFromImage(
			ctx,
			container,
			configOverride,
			specificFilePaths,
			specificFilePathsAllowNotExist,
			includeImports,
			t,
		)
		return env, nil, err
	case buffetch.SourceRef:
		return e.readEnvFromSource(
			ctx,
			container,
			configOverride,
			specificFilePaths,
			specificFilePathsAllowNotExist,
			includeImports,
			includeSourceInfo,
			t,
		)
	default:
		return nil, nil, fmt.Errorf("invalid ref: %T", ref)
	}
}

func (e *envReader) ReadSourceEnv(
	ctx context.Context,
	container app.EnvStdinContainer,
	value string,
	configOverride string,
	specificFilePaths []string,
	specificFilePathsAllowNotExist bool,
	includeImports bool,
	includeSourceInfo bool,
) (_ *Env, _ []*filev1beta1.FileAnnotation, retErr error) {
	defer instrument.Start(e.logger, "read_source_env").End()
	defer func() {
		if retErr != nil {
			retErr = fmt.Errorf("%v: %w", e.valueFlagName, retErr)
		}
	}()
	sourceRef, err := e.fetchRefParser.GetSourceRef(ctx, value)
	if err != nil {
		return nil, nil, err
	}
	return e.readEnvFromSource(
		ctx,
		container,
		configOverride,
		specificFilePaths,
		specificFilePathsAllowNotExist,
		includeImports,
		includeSourceInfo,
		sourceRef,
	)
}

func (e *envReader) ReadImageEnv(
	ctx context.Context,
	container app.EnvStdinContainer,
	value string,
	configOverride string,
	specificFilePaths []string,
	specificFilePathsAllowNotExist bool,
	includeImports bool,
) (_ *Env, retErr error) {
	defer instrument.Start(e.logger, "read_image_env").End()
	defer func() {
		if retErr != nil {
			retErr = fmt.Errorf("%v: %w", e.valueFlagName, retErr)
		}
	}()
	imageRef, err := e.fetchRefParser.GetImageRef(ctx, value)
	if err != nil {
		return nil, err
	}
	return e.readEnvFromImage(
		ctx,
		container,
		configOverride,
		specificFilePaths,
		specificFilePathsAllowNotExist,
		includeImports,
		imageRef,
	)
}

func (e *envReader) ListFiles(
	ctx context.Context,
	container app.EnvStdinContainer,
	value string,
	configOverride string,
) (_ []string, retErr error) {
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
		readCloser, err := e.fetchReader.GetImage(ctx, container, t)
		if err != nil {
			return nil, err
		}
		defer func() {
			retErr = multierr.Append(retErr, readCloser.Close())
		}()
		// if we have an image, list the files in the image
		image, err := e.getImage(readCloser, t.ImageEncoding())
		if err != nil {
			return nil, err
		}
		files := image.GetFile()
		filePaths := make([]string, len(files))
		for i, file := range image.GetFile() {
			filePath, err := t.RelPathToExternalPath(file.GetName())
			if err != nil {
				return nil, err
			}
			filePaths[i] = filePath
		}
		sort.Strings(filePaths)
		return filePaths, nil
	case buffetch.SourceRef:
		readBucketCloser, err := e.fetchReader.GetSource(ctx, container, t)
		if err != nil {
			return nil, err
		}
		defer func() {
			retErr = multierr.Append(retErr, readBucketCloser.Close())
		}()
		var config *bufconfig.Config
		if configOverride != "" {
			config, err = e.parseConfigOverride(configOverride)
			if err != nil {
				return nil, err
			}
		} else {
			// if there is no config override, we read the config from the bucket
			// if there was no file, this just returns default config
			config, err = e.configProvider.GetConfigForReadBucket(ctx, readBucketCloser)
			if err != nil {
				return nil, err
			}
		}
		protoFileSet, err := e.buildHandler.GetProtoFileSet(
			ctx,
			readBucketCloser,
			bufbuild.GetProtoFileSetOptions{
				Roots:    config.Build.Roots,
				Excludes: config.Build.Excludes,
			},
		)
		if err != nil {
			return nil, err
		}
		relPaths := protoFileSet.RealFilePaths()
		externalPaths := make([]string, len(relPaths))
		for i, relPath := range relPaths {
			externalPath, err := t.RelPathToExternalPath(relPath)
			if err != nil {
				// This is an internal error if we cannot resolve this file path.
				return nil, err
			}
			externalPaths[i] = externalPath
		}
		//// The files are in the order of the root file paths, we want to sort them for output.
		sort.Strings(externalPaths)
		return externalPaths, nil
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

func (e *envReader) readEnvFromSource(
	ctx context.Context,
	container app.EnvStdinContainer,
	configOverride string,
	specificFilePaths []string,
	specificFilePathsAllowNotExist bool,
	includeImports bool,
	includeSourceInfo bool,
	sourceRef buffetch.SourceRef,
) (_ *Env, _ []*filev1beta1.FileAnnotation, retErr error) {
	readBucketCloser, err := e.fetchReader.GetSource(ctx, container, sourceRef)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readBucketCloser.Close())
	}()

	var config *bufconfig.Config
	if configOverride != "" {
		config, err = e.parseConfigOverride(configOverride)
		if err != nil {
			return nil, nil, err
		}
	} else {
		// if there is no config override, we read the config from the bucket
		// if there was no file, this just returns default config
		config, err = e.configProvider.GetConfigForReadBucket(ctx, readBucketCloser)
		if err != nil {
			return nil, nil, err
		}
	}
	relFilePaths := make([]string, len(specificFilePaths))
	for i, specificFilePath := range specificFilePaths {
		relFilePath, err := sourceRef.ExternalPathToRelPath(specificFilePath)
		if err != nil {
			return nil, nil, err
		}
		relFilePaths[i] = relFilePath
	}

	// we now have everything we need, actually build the image
	var protoFileSet bufbuild.ProtoFileSet
	if len(relFilePaths) > 0 {
		protoFileSet, err = e.buildHandler.GetProtoFileSetForFiles(
			ctx,
			readBucketCloser,
			relFilePaths,
			bufbuild.GetProtoFileSetForFilesOptions{
				Roots:         config.Build.Roots,
				AllowNotExist: specificFilePathsAllowNotExist,
			},
		)
		if err != nil {
			return nil, nil, err
		}
	} else {
		protoFileSet, err = e.buildHandler.GetProtoFileSet(
			ctx,
			readBucketCloser,
			bufbuild.GetProtoFileSetOptions{
				Roots:    config.Build.Roots,
				Excludes: config.Build.Excludes,
			},
		)
		if err != nil {
			return nil, nil, err
		}
	}
	resolver := newRelRealProtoFilePathResolver(protoFileSet, sourceRef)
	buildResult, fileAnnotations, err := e.buildHandler.Build(
		ctx,
		readBucketCloser,
		protoFileSet,
		bufbuild.BuildOptions{
			IncludeImports:    includeImports,
			IncludeSourceInfo: includeSourceInfo,
		},
	)
	if err != nil {
		return nil, nil, err
	}
	if len(fileAnnotations) > 0 {
		// the documentation for EnvReader says we will resolve before returning
		if err := bufbuild.FixFileAnnotationPaths(resolver, fileAnnotations...); err != nil {
			return nil, nil, err
		}
		return nil, fileAnnotations, nil
	}
	return &Env{
		Image:            buildResult.Image,
		ImageWithImports: buildResult.ImageWithImports,
		Resolver:         resolver,
		Config:           config,
	}, nil, nil
}

func (e *envReader) readEnvFromImage(
	ctx context.Context,
	container app.EnvStdinContainer,
	configOverride string,
	specificFilePaths []string,
	specificFilePathsAllowNotExist bool,
	includeImports bool,
	imageRef buffetch.ImageRef,
) (_ *Env, retErr error) {
	readCloser, err := e.fetchReader.GetImage(ctx, container, imageRef)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readCloser.Close())
	}()
	image, err := e.getImage(readCloser, imageRef.ImageEncoding())
	if err != nil {
		return nil, err
	}
	config, err := e.GetConfig(ctx, configOverride)
	if err != nil {
		return nil, err
	}
	relFilePaths := make([]string, len(specificFilePaths))
	for i, specificFilePath := range specificFilePaths {
		relFilePath, err := imageRef.ExternalPathToRelPath(specificFilePath)
		if err != nil {
			return nil, err
		}
		relFilePaths[i] = relFilePath
	}
	if len(specificFilePaths) > 0 {
		// note this must include imports if these are required for whatever operation you are doing
		image, err = extimage.ImageWithSpecificNames(image, specificFilePathsAllowNotExist, relFilePaths...)
		if err != nil {
			return nil, err
		}
	}
	if !includeImports {
		// TODO: check if image is self-contained, if so, set ImageWithImports
		// need logic to check if image is self-contained in extimage
		image, err = extimage.ImageWithoutImports(image)
		if err != nil {
			return nil, err
		}
	}
	return &Env{
		Image:  image,
		Config: config,
	}, nil
}

func (e *envReader) getImage(reader io.Reader, imageEncoding buffetch.ImageEncoding) (_ *imagev1beta1.Image, retErr error) {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	// we cannot determine fileDescriptorProtos ahead of time so we cannot handle extensions
	// TODO: we do not happen to need them for our use case with linting, but we need to dicuss this
	image := &imagev1beta1.Image{}
	switch imageEncoding {
	case buffetch.ImageEncodingBin:
		err = protoencoding.NewWireUnmarshaler(nil).Unmarshal(data, image)
	case buffetch.ImageEncodingJSON:
		err = protoencoding.NewJSONUnmarshaler(nil).Unmarshal(data, image)
	default:
		return nil, fmt.Errorf("unknown image encoding: %v", imageEncoding)
	}
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal Image: %v", err)
	}
	if err := extimage.ValidateImage(image); err != nil {
		return nil, err
	}
	return image, nil
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

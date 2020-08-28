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

// Package bufwire wires everything together.
//
// TODO: This package should be split up into individual functionality.
package bufwire

import (
	"context"

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/bufcore"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/internal/buf/buffetch"
	"github.com/bufbuild/buf/internal/pkg/app"
	"go.uber.org/zap"
)

// Env is an environment.
type Env interface {
	Image() bufimage.Image
	Config() *bufconfig.Config
}

// EnvReader is an environment reader.
type EnvReader interface {
	// GetEnv gets an environment for the fetch value.
	//
	// If externalFilePaths is empty, this builds all files under Buf control.
	GetEnv(
		ctx context.Context,
		container app.EnvStdinContainer,
		ref buffetch.Ref,
		configOverride string,
		externalFilePaths []string,
		externalFileFilePathsAllowNotExist bool,
		excludeSourceCodeInfo bool,
	) (Env, []bufanalysis.FileAnnotation, error)
	// GetSourceOrModuleEnv is the same as GetEnv, but only allows source or module values, and always builds.
	GetSourceOrModuleEnv(
		ctx context.Context,
		container app.EnvStdinContainer,
		sourceOrModuleRef buffetch.SourceOrModuleRef,
		configOverride string,
		externalFilePaths []string,
		externalFileFilePathsAllowNotExist bool,
		excludeSourceCodeInfo bool,
	) (Env, []bufanalysis.FileAnnotation, error)
	// ListFiles lists the files.
	ListFiles(
		ctx context.Context,
		container app.EnvStdinContainer,
		ref buffetch.Ref,
		configOverride string,
	) ([]bufcore.FileInfo, error)
}

// NewEnvReader returns a new EnvReader.
func NewEnvReader(
	logger *zap.Logger,
	fetchReader buffetch.Reader,
	configProvider bufconfig.Provider,
	moduleBucketBuilder bufmodulebuild.ModuleBucketBuilder,
	moduleFileSetBuilder bufmodulebuild.ModuleFileSetBuilder,
	imageBuilder bufimagebuild.Builder,
	configOverrideFlagName string,
) EnvReader {
	return newEnvReader(
		logger,
		fetchReader,
		configProvider,
		moduleBucketBuilder,
		moduleFileSetBuilder,
		imageBuilder,
		configOverrideFlagName,
	)
}

// ImageReader is an image reader.
type ImageReader interface {
	// GetImage reads the image from the value.
	GetImage(
		ctx context.Context,
		container app.EnvStdinContainer,
		imageRef buffetch.ImageRef,
		externalFilePaths []string,
		externalFileFilePathsAllowNotExist bool,
		excludeSourceCodeInfo bool,
	) (bufimage.Image, error)
}

// NewImageReader returns a new ImageReader.
func NewImageReader(
	logger *zap.Logger,
	fetchReader buffetch.ImageReader,
) ImageReader {
	return newImageReader(
		logger,
		fetchReader,
	)
}

// ImageWriter is an image writer.
type ImageWriter interface {
	// PutImage writes the image to the value.
	//
	// The file must be an image format.
	// This is a no-np if value is the equivalent of /dev/null.
	PutImage(
		ctx context.Context,
		container app.EnvStdoutContainer,
		imageRef buffetch.ImageRef,
		image bufimage.Image,
		asFileDescriptorSet bool,
		excludeImports bool,
	) error
}

// NewImageWriter returns a new ImageWriter.
func NewImageWriter(
	logger *zap.Logger,
	fetchWriter buffetch.Writer,
) ImageWriter {
	return newImageWriter(
		logger,
		fetchWriter,
	)
}

// ConfigReader reads configs.
type ConfigReader interface {
	// GetConfig gets the config.
	GetConfig(
		ctx context.Context,
		configOverride string,
	) (*bufconfig.Config, error)
}

// NewConfigReader returns a new ConfigReader.
func NewConfigReader(
	logger *zap.Logger,
	configProvider bufconfig.Provider,
	configOverrideFlagName string,
) ConfigReader {
	return newConfigReader(
		logger,
		configProvider,
		configOverrideFlagName,
	)
}

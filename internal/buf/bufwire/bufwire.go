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
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/internal/buf/buffetch"
	"github.com/bufbuild/buf/internal/pkg/app"
	"go.uber.org/zap"
)

// ImageConfig is an image and configuration.
type ImageConfig interface {
	Image() bufimage.Image
	Config() *bufconfig.Config
}

// ImageConfigReader is an ImageConfig reader.
type ImageConfigReader interface {
	// GetImageConfig gets the ImageConfig for the fetch value.
	//
	// If externalFilePaths is empty, this builds all files under Buf control.
	GetImageConfig(
		ctx context.Context,
		container app.EnvStdinContainer,
		ref buffetch.Ref,
		configOverride string,
		externalFilePaths []string,
		externalFileFilePathsAllowNotExist bool,
		excludeSourceCodeInfo bool,
	) (ImageConfig, []bufanalysis.FileAnnotation, error)
	// GetSourceOrModuleImageConfig is the same as GetImageConfig, but only allows source or module values, and always builds.
	GetSourceOrModuleImageConfig(
		ctx context.Context,
		container app.EnvStdinContainer,
		sourceOrModuleRef buffetch.SourceOrModuleRef,
		configOverride string,
		externalFilePaths []string,
		externalFileFilePathsAllowNotExist bool,
		excludeSourceCodeInfo bool,
	) (ImageConfig, []bufanalysis.FileAnnotation, error)
}

// NewImageConfigReader returns a new ImageConfigReader.
func NewImageConfigReader(
	logger *zap.Logger,
	fetchReader buffetch.Reader,
	configProvider bufconfig.Provider,
	moduleBucketBuilder bufmodulebuild.ModuleBucketBuilder,
	moduleFileSetBuilder bufmodulebuild.ModuleFileSetBuilder,
	imageBuilder bufimagebuild.Builder,
) ImageConfigReader {
	return newImageConfigReader(
		logger,
		fetchReader,
		configProvider,
		moduleBucketBuilder,
		moduleFileSetBuilder,
		imageBuilder,
	)
}

// ModuleConfig is an module and configuration.
type ModuleConfig interface {
	Module() bufmodule.Module
	Config() *bufconfig.Config
}

// ModuleConfigReader is a ModuleConfig reader.
type ModuleConfigReader interface {
	// GetModuleConfig gets the ModuleConfig for the fetch value.
	//
	// If externalFilePaths is empty, this builds all files under Buf control.
	//
	// Note that as opposed to ModuleReader, this will return a Module for either
	// a source or module reference, not just a module reference.
	GetModuleConfig(
		ctx context.Context,
		container app.EnvStdinContainer,
		sourceOrModuleRef buffetch.SourceOrModuleRef,
		configOverride string,
		externalFilePaths []string,
		externalFilePathsAllowNotExist bool,
	) (ModuleConfig, error)
}

// NewModuleConfigReader returns a new ModuleConfigReader
func NewModuleConfigReader(
	logger *zap.Logger,
	fetchReader buffetch.Reader,
	configProvider bufconfig.Provider,
	moduleBucketBuilder bufmodulebuild.ModuleBucketBuilder,
) ModuleConfigReader {
	return newModuleConfigReader(
		logger,
		fetchReader,
		configProvider,
		moduleBucketBuilder,
	)
}

// FileLister lists files.
type FileLister interface {
	// ListFiles lists the files.
	ListFiles(
		ctx context.Context,
		container app.EnvStdinContainer,
		ref buffetch.Ref,
		configOverride string,
	) ([]bufcore.FileInfo, error)
}

// NewFileLister returns a new FileLister.
func NewFileLister(
	logger *zap.Logger,
	fetchReader buffetch.Reader,
	configProvider bufconfig.Provider,
	moduleBucketBuilder bufmodulebuild.ModuleBucketBuilder,
	imageBuilder bufimagebuild.Builder,
) FileLister {
	return newFileLister(
		logger,
		fetchReader,
		configProvider,
		moduleBucketBuilder,
		imageBuilder,
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

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

// Package bufos provides abstractions to read and write images from an OS context.
//
// This is primarily meant for the CLI tool, and isn't used in a service context.
package bufos

import (
	"context"

	"github.com/bufbuild/buf/internal/buf/bufbuild"
	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/buffetch"
	filev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/file/v1beta1"
	imagev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/image/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/app"
	"go.uber.org/zap"
)

// Env is an environment.
type Env struct {
	// Image is the image to use.
	//
	// Validated.
	Image *imagev1beta1.Image
	// ImageWithImports is a full Image with imports.
	//
	// Can be nil. This will only be present from source reads.
	// See bufbuild.Handler.
	// Validated.
	ImageWithImports *imagev1beta1.Image
	// Resolver is the resolver to apply before printing paths or FileAnnotations.
	// Can be nil.
	Resolver bufbuild.ProtoRealFilePathResolver
	// Config is the config to use.
	Config *bufconfig.Config
}

// EnvReader is an env reader.
type EnvReader interface {
	// ReadEnv reads an environment.
	//
	// If specificFilePaths is empty, this builds all the files under Buf control.
	//
	// Note that includeImports will only be respected for Images if the image was
	// built with buf - if it was built with protoc, we have no way of detecting
	// what is and isn't an import.
	//
	// Note that includeSourceInfo will only be respected for Sources. We make
	// no modifications for Images.
	//
	// FileAnnotations will be fixed per the resolver before returning.
	// If stdin is nil and this tries to read from stdin, returns user error.
	ReadEnv(
		ctx context.Context,
		container app.EnvStdinContainer,
		value string,
		configOverride string,
		specificFilePaths []string,
		specificFilePathsAllowNotExist bool,
		includeImports bool,
		includeSourceInfo bool,
	) (*Env, []*filev1beta1.FileAnnotation, error)
	// ReadSourceEnv reads an source environment.
	//
	// This is the same as ReadEnv but disallows image values and always builds.
	ReadSourceEnv(
		ctx context.Context,
		container app.EnvStdinContainer,
		value string,
		configOverride string,
		specificFilePaths []string,
		specificFilePathsAllowNotExist bool,
		includeImports bool,
		includeSourceInfo bool,
	) (*Env, []*filev1beta1.FileAnnotation, error)
	// ReadImageEnv reads an image environment.
	//
	// This is the same as ReadEnv but disallows source values and never builds.
	ReadImageEnv(
		ctx context.Context,
		container app.EnvStdinContainer,
		value string,
		configOverride string,
		specificFilePaths []string,
		specificFilePathsAllowNotExist bool,
		includeImports bool,
	) (*Env, error)

	// ListFiles lists the files.
	ListFiles(
		ctx context.Context,
		container app.EnvStdinContainer,
		value string,
		configOverride string,
	) ([]string, error)

	// GetConfig gets the config.
	GetConfig(
		ctx context.Context,
		configOverride string,
	) (*bufconfig.Config, error)
}

// NewEnvReader returns a new EnvReader.
func NewEnvReader(
	logger *zap.Logger,
	fetchRefParser buffetch.RefParser,
	fetchReader buffetch.Reader,
	configProvider bufconfig.Provider,
	buildHandler bufbuild.Handler,
	valueFlagName string,
	configOverrideFlagName string,
) EnvReader {
	return newEnvReader(
		logger,
		fetchRefParser,
		fetchReader,
		configProvider,
		buildHandler,
		valueFlagName,
		configOverrideFlagName,
	)
}

// ImageWriter is an image writer.
type ImageWriter interface {
	// WriteImage writes the image to the value.
	//
	// The file must be an image format.
	// This is a no-np if value is the equivalent of /dev/null.
	//
	// Validates the image before writing.
	WriteImage(
		ctx context.Context,
		container app.EnvStdoutContainer,
		value string,
		asFileDescriptorSet bool,
		image *imagev1beta1.Image,
		imageWithImports *imagev1beta1.Image,
	) error
}

// NewImageWriter returns a new ImageWriter.
func NewImageWriter(
	logger *zap.Logger,
	fetchRefParser buffetch.RefParser,
	fetchWriter buffetch.Writer,
) ImageWriter {
	return newImageWriter(
		logger,
		fetchRefParser,
		fetchWriter,
	)
}

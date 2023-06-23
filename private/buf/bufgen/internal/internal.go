// Copyright 2020-2023 Buf Technologies, Inc.
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

package internal

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/buf/bufwire"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimageutil"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/zap"
)

const (
	// ExternalConfigFilePath is the default external configuration file path.
	ExternalConfigFilePath = "buf.gen.yaml"
	// V1Version is the string used to identify the v1 version of the generate template.
	V1Version = "v1"
	// V1Beta1Version is the string used to identify the v1beta1 version of the generate template.
	V1Beta1Version = "v1beta1"
	// V2Version is the string used to identify the v2 version of the generate template.
	V2Version = "v2"
)

const (
	// StrategyDirectory is the strategy that says to generate per directory.
	//
	// This is the default value.
	StrategyDirectory Strategy = 1
	// StrategyAll is the strategy that says to generate with all files at once.
	StrategyAll Strategy = 2
)

// Strategy is a generation stategy.
type Strategy int

// ParseStrategy parses the Strategy.
//
// If the empty string is provided, this is interpreted as StrategyDirectory.
func ParseStrategy(s string) (Strategy, error) {
	switch s {
	case "", "directory":
		return StrategyDirectory, nil
	case "all":
		return StrategyAll, nil
	default:
		return 0, fmt.Errorf("unknown strategy: %s", s)
	}
}

// String implements fmt.Stringer.
func (s Strategy) String() string {
	switch s {
	case StrategyDirectory:
		return "directory"
	case StrategyAll:
		return "all"
	default:
		return strconv.Itoa(int(s))
	}
}

// ExternalConfigVersion defines the subset of all config
// file versions that is used to determine the configuration version.
type ExternalConfigVersion struct {
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}

// ConfigDataProvider is a provider for config data.
type ConfigDataProvider interface {
	// GetConfigData gets the Config's data in bytes at ExternalConfigFilePath,
	// as well as the id of the file, in the form of `File "<path>"`.
	GetConfigData(context.Context, storage.ReadBucket) ([]byte, string, error)
}

// New ConfigDataProvider returns a new ConfigDataProvider.
func NewConfigDataProvider(logger *zap.Logger) ConfigDataProvider {
	return newConfigDataProvider(logger)
}

// ReadDataFromConfig reads generation config data from the default path,
// or an override path, or override data, and returns these data, a file ID
// useful for building error messages, and two unmarshallers.
func ReadDataFromConfig(
	ctx context.Context,
	logger *zap.Logger,
	provider ConfigDataProvider,
	readBucket storage.ReadBucket,
	options ...ReadConfigOption,
) (
	data []byte,
	fileID string,
	unmarshalNonStrict func(data []byte, v interface{}) error,
	unmarshalStrict func(data []byte, v interface{}) error,
	err error,
) {
	return readDataFromConfig(
		ctx,
		logger,
		provider,
		readBucket,
		options...,
	)
}

// ReadConfigOption is an option for ReadConfig.
type ReadConfigOption func(*readConfigOptions)

// ReadConfigWithOverride sets the override.
//
// If override is set, this will first check if the override ends in .json or .yaml, if so,
// this reads the file at this path and uses it. Otherwise, this assumes this is configuration
// data in either JSON or YAML format, and unmarshals it.
//
// If no override is set, this reads ExternalConfigFilePath in the bucket.
func ReadConfigWithOverride(override string) ReadConfigOption {
	return func(readConfigOptions *readConfigOptions) {
		readConfigOptions.override = override
	}
}

// ReadConfig reads the configuration version from the OS or an override, if any.
//
// Only use in CLI tools.
func ReadConfigVersion(
	ctx context.Context,
	logger *zap.Logger,
	readBucket storage.ReadBucket,
	options ...ReadConfigOption,
) (string, error) {
	return readConfigVersion(
		ctx,
		logger,
		readBucket,
		options...,
	)
}

// GetInputImage returns an image from the given ref.
func GetInputImage(
	ctx context.Context,
	container appflag.Container,
	ref buffetch.Ref,
	imageConfigReader bufwire.ImageConfigReader,
	configLocationOverride string,
	includedPaths []string,
	excludedPaths []string,
	errorFormat string,
	includedTypes []string,
	fileAnnotationErr error,
) (bufimage.Image, error) {
	imageConfigs, fileAnnotations, err := imageConfigReader.GetImageConfigs(
		ctx,
		container,
		ref,
		configLocationOverride,
		includedPaths,
		excludedPaths,
		false, // input files must exist
		false, // we must include source info for generation
	)
	if err != nil {
		return nil, err
	}
	if len(fileAnnotations) > 0 {
		if err := bufanalysis.PrintFileAnnotations(container.Stderr(), fileAnnotations, errorFormat); err != nil {
			return nil, err
		}
		return nil, fileAnnotationErr
	}
	images := make([]bufimage.Image, 0, len(imageConfigs))
	for _, imageConfig := range imageConfigs {
		images = append(images, imageConfig.Image())
	}
	image, err := bufimage.MergeImages(images...)
	if err != nil {
		return nil, err
	}
	if len(includedTypes) > 0 {
		image, err = bufimageutil.ImageFilteredByTypes(image, includedTypes...)
		if err != nil {
			return nil, err
		}
	}
	return image, nil
}

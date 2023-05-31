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

package bufgen

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/pluginpb"
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

// ExternalConfigVersion defines the subset of all config
// file versions that is used to determine the configuration version.
type ExternalConfigVersion struct {
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}

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

// Provider is a provider.
type ConfigDataProvider interface {
	// GetConfigData gets the Config's data in bytes at ExternalConfigFilePath,
	// as well as the id of the file.
	GetConfigData(context.Context, storage.ReadBucket) ([]byte, string, error)
}

// NewProvider returns a new Provider.
func NewConfigDataProvider(logger *zap.Logger) ConfigDataProvider {
	return newConfigDataProvider(logger)
}

// ReadConfigOption is an option for ReadConfig.

type readConfigOptions struct {
	override string
}

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

// ConfigExists checks if a generation configuration file exists.
func ConfigExists(ctx context.Context, readBucket storage.ReadBucket) (bool, error) {
	return storage.Exists(ctx, readBucket, ExternalConfigFilePath)
}

// ReadConfig reads the configuration version from the OS or an override, if any.
//
// Only use in CLI tools.
func ReadConfigVersion(
	ctx context.Context,
	logger *zap.Logger,
	provider ConfigDataProvider,
	readBucket storage.ReadBucket,
	options ...ReadConfigOption,
) (string, error) {
	version, err := ReadFromConfig(
		ctx,
		logger,
		provider,
		readBucket,
		getConfigVersion,
		options...,
	)
	if err != nil || version == nil {
		return "", err
	}
	return *version, nil
}

func ReadFromConfig[V any](
	ctx context.Context,
	logger *zap.Logger,
	provider ConfigDataProvider,
	readBucket storage.ReadBucket,
	configGetter ConfigGetter[V],
	options ...ReadConfigOption,
) (*V, error) {
	return readFromConfig(ctx, logger, provider, readBucket, configGetter, options...)
}

type ConfigGetter[V any] func(
	*zap.Logger,
	func([]byte, interface{}) error,
	func([]byte, interface{}) error,
	[]byte,
	string,
) (*V, error)

func ComputeRequiredFeatures(image bufimage.Image) requiredFeatures {
	return computeRequiredFeatures(image)
}

type Plugin interface {
	PluginName() string
	// Out() string // TODO
}

func CheckRequiredFeatures(
	container app.StderrContainer,
	required requiredFeatures,
	responses []*pluginpb.CodeGeneratorResponse,
	plugins []Plugin,
) {
	checkRequiredFeatures(container, required, responses, plugins)
}

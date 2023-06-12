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

	"github.com/bufbuild/buf/private/buf/bufgen/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufwasm"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
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

// ExternalConfigVersion defines the subset of all config
// file versions that is used to determine the configuration version.
// TODO: investigate if this can be hidden in internal and if v1beta1_migrator
// can call ReadConfigVersion.
type ExternalConfigVersion struct {
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}

// Generator generates Protobuf stubs based on configurations.
type Generator interface {
	// Generate calls the generation logic.
	//
	// The config is assumed to be valid. If created by ReadConfig, it will
	// always be valid.
	Generate(
		ctx context.Context,
		container app.EnvStdioContainer,
		pluginConfigs []PluginConfig,
		image bufimage.Image,
		options ...GenerateOption,
	) error
}

// NewGenerator returns a new Generator.
func NewGenerator(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	runner command.Runner,
	wasmPluginExecutor bufwasm.PluginExecutor,
	clientConfig *connectclient.Config,
) Generator {
	return newGenerator(
		logger,
		storageosProvider,
		runner,
		wasmPluginExecutor,
		clientConfig,
	)
}

// GenerateOption is an option for Generate.
type GenerateOption func(*generateOptions)

// GenerateWithBaseOutDirPath returns a new GenerateOption that uses the given
// base directory as the output directory.
//
// The default is to use the current directory.
func GenerateWithBaseOutDirPath(baseOutDirPath string) GenerateOption {
	return func(generateOptions *generateOptions) {
		generateOptions.baseOutDirPath = baseOutDirPath
	}
}

// GenerateWithIncludeImports says to also generate imports.
//
// Note that this does NOT result in the Well-Known Types being generated, use
// GenerateWithIncludeWellKnownTypes to include the Well-Known Types.
func GenerateWithIncludeImports() GenerateOption {
	return func(generateOptions *generateOptions) {
		generateOptions.includeImports = true
	}
}

// GenerateWithIncludeWellKnownTypes says to also generate well known types.
//
// This option has no effect if GenerateWithIncludeImports is not set.
func GenerateWithIncludeWellKnownTypes() GenerateOption {
	return func(generateOptions *generateOptions) {
		generateOptions.includeWellKnownTypes = true
	}
}

// GenerateWithWASMEnabled says to enable WASM support.
func GenerateWithWASMEnabled() GenerateOption {
	return func(generateOptions *generateOptions) {
		generateOptions.wasmEnabled = true
	}
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
	provider ConfigDataProvider,
	readBucket storage.ReadBucket,
	options ...ReadConfigOption,
) (string, error) {
	return readConfigVersion(
		ctx,
		logger,
		provider,
		readBucket,
		options...,
	)
}

// ReadFromConfig reads the configuration as bytes from the OS or an override, if any,
// and interprets these bytes as a value of V, with configGetter.
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

// ConfigGetter is a function that interpret a slice of bytes as a value of type V.
type ConfigGetter[V any] func(
	logger *zap.Logger,
	unmarshalNonStrict func([]byte, interface{}) error,
	unmarshalStrict func([]byte, interface{}) error,
	data []byte,
	id string,
) (*V, error)

// PluginConfig is a plugin configuration.
type PluginConfig interface {
	PluginName() string
	Out() string
	Opt() string

	pluginConfig()
}

// LocalPluginConfig is a local plugin configuration.
type LocalPluginConfig interface {
	PluginConfig
	Strategy() internal.Strategy

	localPluginConfig()
}

// NewLocalPluginConfig creates a new local plugin configuration whose exact
// type is not yet determined, which could be either binary or protoc built-in.
func NewLocalPluginConfig(
	name string,
	strategy internal.Strategy,
	out string,
	opt string,
) LocalPluginConfig {
	return newLocalPluginConfig(
		name,
		strategy,
		out,
		opt,
	)
}

// BinaryPluginConfig is a binary plugin configuration.
type BinaryPluginConfig interface {
	LocalPluginConfig
	Path() []string

	binaryPluginConfig()
}

// NewBinaryPluginConfig returns a new binary plugin configuration.
func NewBinaryPluginConfig(
	name string,
	path []string,
	strategy internal.Strategy,
	out string,
	opt string,
) (BinaryPluginConfig, error) {
	return newBinaryPluginConfig(
		name,
		path,
		strategy,
		out,
		opt,
	)
}

// ProtocBuiltinPluginConfig is a protoc built-in plugin configuration.
type ProtocBuiltinPluginConfig interface {
	LocalPluginConfig
	ProtocPath() string

	protocBuiltinPluginConfig()
}

// NewProtocBuiltinPluginConfig returns a new protoc built-in plugin configuration.
func NewProtocBuiltinPluginConfig(
	name string,
	protocPath string,
	out string,
	opt string,
	strategy internal.Strategy,
) ProtocBuiltinPluginConfig {
	return newProtocBuiltinPluginConfig(
		name,
		protocPath,
		out,
		opt,
		strategy,
	)
}

// RemotePluginConfig is a remote plugin configuration.
type RemotePluginConfig interface {
	PluginConfig
	Remote() string

	remotePluginConfig()
}

// CuratedPluginConfig is a curated plugin configuration.
type CuratedPluginConfig interface {
	RemotePluginConfig
	Revision() int

	curatedPluginConfig()
}

// NewCuratedPluginConfig returns a new curated plugin configuration.
func NewCuratedPluginConfig(
	plugin string,
	revision int, // TODO: maybe pointer to indicate absence
	out string,
	opt string,
) CuratedPluginConfig {
	return newCuratedPluginConfig(
		plugin,
		revision,
		out,
		opt,
	)
}

// LegacyRemotePluginConfig is a legacy remote plugin configuration.
type LegacyRemotePluginConfig interface {
	RemotePluginConfig

	legacyRemotePluginConfig()
}

// NewLegacyRemotePluginConfig returns a new legacy remote plugin configuration.
func NewLegacyRemotePluginConfig(
	remote string,
	out string,
	opt string,
) LegacyRemotePluginConfig {
	return newLegacyRemotePluginConfig(
		remote,
		out,
		opt,
	)
}

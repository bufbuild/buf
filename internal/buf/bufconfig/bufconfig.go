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

// Package bufconfig contains the configuration functionality.
package bufconfig

import (
	"context"

	"github.com/bufbuild/buf/internal/buf/bufcheck/bufbreaking"
	"github.com/bufbuild/buf/internal/buf/bufcheck/buflint"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/zap"
)

// ExternalConfigV1Beta1FilePath is the default configuration file path for v1beta1.
const ExternalConfigV1Beta1FilePath = "buf.yaml"

// Config is the user config.
type Config struct {
	ModuleIdentity bufmodule.ModuleIdentity
	Build          *bufmodulebuild.Config
	Breaking       *bufbreaking.Config
	Lint           *buflint.Config
}

// Provider is a provider.
type Provider interface {
	// GetConfig gets the Config for the YAML data at ConfigFilePath.
	//
	// If the data is of length 0, returns the default config.
	GetConfig(ctx context.Context, readBucket storage.ReadBucket) (*Config, error)
	// GetConfig gets the Config for the given JSON or YAML data.
	//
	// If the data is of length 0, returns the default config.
	GetConfigForData(ctx context.Context, data []byte) (*Config, error)
}

// NewProvider returns a new Provider.
func NewProvider(logger *zap.Logger) Provider {
	return newProvider(logger)
}

// WriteConfig writes an initial configuration file into the bucket.
func WriteConfig(
	ctx context.Context,
	writeBucket storage.WriteBucket,
	options ...WriteConfigOption,
) error {
	return writeConfig(
		ctx,
		writeBucket,
		options...,
	)
}

// WriteConfigOption is an option for WriteConfig.
type WriteConfigOption func(*writeConfigOptions)

// WriteConfigWithModuleIdentity returns a new WriteConfigOption that sets the name of the
// module to the given ModuleIdentity.
//
// The default is to not set the name.
func WriteConfigWithModuleIdentity(moduleIdentity bufmodule.ModuleIdentity) WriteConfigOption {
	return func(writeConfigOptions *writeConfigOptions) {
		writeConfigOptions.moduleIdentity = moduleIdentity
	}
}

// WriteConfigWithDependencyModuleReferences returns a new WriteConfigOption that sets the
// dependencies of the module.
//
// The default is to not have any dependencies.
//
// If this option is used, WriteConfigWithModuleIdentity must also be used.
func WriteConfigWithDependencyModuleReferences(dependencyModuleReferences ...bufmodule.ModuleReference) WriteConfigOption {
	return func(writeConfigOptions *writeConfigOptions) {
		writeConfigOptions.dependencyModuleReferences = dependencyModuleReferences
	}
}

// WriteConfigWithDocumentationComments returns a new WriteConfigOption that documents the resulting configuration file.
func WriteConfigWithDocumentationComments() WriteConfigOption {
	return func(writeConfigOptions *writeConfigOptions) {
		writeConfigOptions.documentationComments = true
	}
}

// WriteConfigWithUncomment returns a new WriteConfigOption that uncomments the resulting configuration file
// options that are commented out by default.
//
// If this option is used, WriteConfigWithDocumentationComments must also be used.
func WriteConfigWithUncomment() WriteConfigOption {
	return func(writeConfigOptions *writeConfigOptions) {
		writeConfigOptions.uncomment = true
	}
}

// ReadConfig reads the configuration, including potentially reading from the OS for an override.
//
// Only use in CLI tools.
func ReadConfig(
	ctx context.Context,
	provider Provider,
	readBucket storage.ReadBucket,
	options ...ReadConfigOption,
) (*Config, error) {
	return readConfig(
		ctx,
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
// If no override is set, this reads ConfigFilePath in the bucket..
func ReadConfigWithOverride(override string) ReadConfigOption {
	return func(readConfigOptions *readConfigOptions) {
		readConfigOptions.override = override
	}
}

// ConfigExists checks if a configuration file exists.
func ConfigExists(ctx context.Context, readBucket storage.ReadBucket) (bool, error) {
	return storage.Exists(ctx, readBucket, ExternalConfigV1Beta1FilePath)
}

type externalConfigV1Beta1 struct {
	Version  string                               `json:"version,omitempty" yaml:"version,omitempty"`
	Name     string                               `json:"name,omitempty" yaml:"name,omitempty"`
	Build    bufmodulebuild.ExternalConfigV1Beta1 `json:"build,omitempty" yaml:"build,omitempty"`
	Breaking bufbreaking.ExternalConfigV1Beta1    `json:"breaking,omitempty" yaml:"breaking,omitempty"`
	Lint     buflint.ExternalConfigV1Beta1        `json:"lint,omitempty" yaml:"lint,omitempty"`
	Deps     []string                             `json:"deps,omitempty" yaml:"deps,omitempty"`
}

type externalConfigVersion struct {
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}

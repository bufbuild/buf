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

// Package bufgen does configuration-based generation.
//
// It is used by the buf generate command.
package bufgen

import (
	"context"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage"
	"github.com/bufbuild/buf/internal/pkg/app"
	"go.uber.org/zap"
)

// Generator generates Protobuf stubs based on configurations.
type Generator interface {
	// Generate calls the generation logic.
	//
	// The config is assumed to be valid. If created by ReadConfig, it will
	// always be valid.
	Generate(
		ctx context.Context,
		container app.EnvStdioContainer,
		config *Config,
		image bufimage.Image,
		options ...GenerateOption,
	) error
}

// NewGenerator returns a new Generator.
func NewGenerator(logger *zap.Logger) Generator {
	return newGenerator(logger)
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

// Config is a configuration.
type Config struct {
	// Required
	PluginConfigs []*PluginConfig
}

// PluginConfig is a plugin configuration.
type PluginConfig struct {
	// Required
	Name string
	// Required
	Out string
	// Optional
	Opt string
	// Optional
	Path string
}

// ReadConfig reads the configuration from the OS.
//
// This will first check if the override ends in .json or .yaml, if so,
// this reads the file at this path and uses it. Otherwise, this assumes this is configuration
// data in either JSON or YAML format, and unmarshals it.
//
// Only use in CLI tools.
func ReadConfig(fileOrData string) (*Config, error) {
	return readConfig(fileOrData)
}

// ExternalConfigV1Beta1 is an external configuration.
//
// Only use outside of this package for testing.
type ExternalConfigV1Beta1 struct {
	Version string                        `json:"version,omitempty" yaml:"version,omitempty"`
	Plugins []ExternalPluginConfigV1Beta1 `json:"plugins,omitempty" yaml:"plugins,omitempty"`
}

// ExternalPluginConfigV1Beta1 is an external plugin configuration.
//
// Only use outside of this package for testing.
type ExternalPluginConfigV1Beta1 struct {
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	Out  string `json:"out,omitempty" yaml:"out,omitempty"`
	Opt  string `json:"opt,omitempty" yaml:"opt,omitempty"`
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
}

type externalConfigVersion struct {
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}

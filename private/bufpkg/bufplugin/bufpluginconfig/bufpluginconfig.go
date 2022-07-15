// Copyright 2020-2022 Buf Technologies, Inc.
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

// Package bufpluginconfig defines the buf.plugin.yaml file.
package bufpluginconfig

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginref"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/storage"
)

const (
	// ExternalConfigFilePath is the default configuration file path for v1.
	ExternalConfigFilePath = "buf.plugin.yaml"
	// V1Version is the version string used to indicate the v1 version of the buf.plugin.yaml file.
	V1Version = "v1"
)

var (
	// AllConfigFilePaths are all acceptable config file paths without overrides.
	//
	// These are in the order we should check.
	AllConfigFilePaths = []string{
		ExternalConfigFilePath,
	}
)

// Config is the plugin config.
type Config struct {
	// Name is the name of the plugin (e.g. 'buf.build/protocolbuffers/go').
	Name bufpluginref.PluginIdentity
	// PluginVersion is the version of the plugin's implementation
	// (e.g the protoc-gen-connect-go implementation is v0.2.0).
	//
	// This excludes any other details found in the buf.plugin.yaml
	// or plugin source (e.g. Dockerfile) that would otherwise influence
	// the plugin's behavior.
	PluginVersion string
	// SourceURL is an optional attribute used to specify where the source
	// for the plugin can be found.
	SourceURL string
	// Description is an optional attribute to provide a more detailed
	// description for the plugin.
	Description string
	// Dependencies are the dependencies this plugin has on other plugins.
	//
	// An example of a dependency might be a 'protoc-gen-go-grpc' plugin
	// which depends on the 'protoc-gen-go' generated code.
	Dependencies []bufpluginref.PluginReference
	// Options is the default set of options passed into the plugin.
	//
	// For now, all options are string values. This could eventually
	// support other types (like JSON Schema and Terraform variables),
	// where strings are the default value unless otherwise specified.
	//
	// Note that some legacy plugins don't always express their options
	// as key value pairs. For example, protoc-gen-java has an option
	// that can be passed like so:
	//
	//  java_opt=annotate_code
	//
	// In those cases, the option value in this map will be set to
	// the empty string, and the option will be propagated to the
	// compiler without the '=' delimiter.
	Options map[string]string
	// Runtime is the runtime configuration, which lets the user specify
	// runtime dependencies, and other metadata that applies to a specific
	// remote generation registry (e.g. the Go module proxy, NPM registry,
	// etc).
	Runtime *RuntimeConfig
}

// RuntimeConfig is the configuration for the runtime of a plugin.
//
// Only one field will be set.
type RuntimeConfig struct {
	Go  *GoRuntimeConfig
	NPM *NPMRuntimeConfig
}

// GoRuntimeConfig is the runtime configuration for a Go plugin.
type GoRuntimeConfig struct {
	MinVersion string
	Deps       []*GoRuntimeDependencyConfig
}

// GoRuntimeDependencyConfig is the go runtime dependency configuration.
type GoRuntimeDependencyConfig struct {
	Module  string
	Version string
}

// NPMRuntimeConfig is the runtime configuration for a JavaScript NPM plugin.
type NPMRuntimeConfig struct {
	Deps []*NPMRuntimeDependencyConfig
}

// NPMRuntimeDependencyConfig is the npm runtime dependency configuration.
type NPMRuntimeDependencyConfig struct {
	Package string
	Version string
}

// GetConfigForBucket gets the Config for the YAML data at ConfigFilePath.
//
// If the data is of length 0, returns the default config.
func GetConfigForBucket(ctx context.Context, readBucket storage.ReadBucket) (*Config, error) {
	return getConfigForBucket(ctx, readBucket)
}

// GetConfigForData gets the Config for the given JSON or YAML data.
//
// If the data is of length 0, returns the default config.
func GetConfigForData(ctx context.Context, data []byte) (*Config, error) {
	return getConfigForData(ctx, data)
}

// ExistingConfigFilePath checks if a configuration file exists, and if so, returns the path
// within the ReadBucket of this configuration file.
//
// Returns empty string and no error if no configuration file exists.
func ExistingConfigFilePath(ctx context.Context, readBucket storage.ReadBucket) (string, error) {
	for _, configFilePath := range AllConfigFilePaths {
		exists, err := storage.Exists(ctx, readBucket, configFilePath)
		if err != nil {
			return "", err
		}
		if exists {
			return configFilePath, nil
		}
	}
	return "", nil
}

// ParseConfig parses the file at the given path as a Config.
func ParseConfig(config string) (*Config, error) {
	var data []byte
	var err error
	switch filepath.Ext(config) {
	case ".json", ".yaml", ".yml":
		data, err = os.ReadFile(config)
		if err != nil {
			return nil, fmt.Errorf("could not read file: %w", err)
		}
	default:
		return nil, fmt.Errorf("invalid extension %s, must be .json, .yaml or .yml", filepath.Ext(config))
	}
	var externalConfig ExternalConfig
	if err := encoding.UnmarshalJSONOrYAMLStrict(data, &externalConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plugin config: %w", err)
	}
	switch externalConfig.Version {
	case V1Version:
		return newConfig(externalConfig)
	}
	return nil, fmt.Errorf("invalid plugin configuration version: must be one of %v", AllConfigFilePaths)
}

// ExternalConfig represents the on-disk representation
// of the plugin configuration at version v1.
type ExternalConfig struct {
	Version       string                `json:"version,omitempty" yaml:"version,omitempty"`
	Name          string                `json:"name,omitempty" yaml:"name,omitempty"`
	PluginVersion string                `json:"plugin_version,omitempty" yaml:"plugin_version,omitempty"`
	SourceURL     string                `json:"source_url,omitempty" yaml:"source_url,omitempty"`
	Description   string                `json:"description,omitempty" yaml:"description,omitempty"`
	Deps          []string              `json:"deps,omitempty" yaml:"deps,omitempty"`
	Opts          []string              `json:"opts,omitempty" yaml:"opts,omitempty"`
	Runtime       ExternalRuntimeConfig `json:"runtime,omitempty" yaml:"runtime,omitempty"`
}

// ExternalRuntimeConfig is the external configuration for the runtime
// of a plugin.
type ExternalRuntimeConfig struct {
	Go  ExternalGoRuntimeConfig  `json:"go,omitempty" yaml:"go,omitempty"`
	NPM ExternalNPMRuntimeConfig `json:"npm,omitempty" yaml:"npm,omitempty"`
}

// ExternalGoRuntimeConfig is the external runtime configuration for a Go plugin.
type ExternalGoRuntimeConfig struct {
	// The minimum Go version required by the plugin.
	MinVersion string `json:"min_version,omitempty" yaml:"min_version,omitempty"`
	Deps       []struct {
		Module  string `json:"module,omitempty" yaml:"module,omitempty"`
		Version string `json:"version,omitempty" yaml:"version,omitempty"`
	} `json:"deps,omitempty" yaml:"deps,omitempty"`
}

// IsEmpty returns true if the configuration is empty.
func (e ExternalGoRuntimeConfig) IsEmpty() bool {
	return e.MinVersion == "" && len(e.Deps) == 0
}

// ExternalNPMRuntimeConfig is the external runtime configuration for a JavaScript NPM plugin.
type ExternalNPMRuntimeConfig struct {
	Deps []struct {
		Package string `json:"package,omitempty" yaml:"package,omitempty"`
		Version string `json:"version,omitempty" yaml:"version,omitempty"`
	} `json:"deps,omitempty" yaml:"deps,omitempty"`
}

// IsEmpty returns true if the configuration is empty.
func (e ExternalNPMRuntimeConfig) IsEmpty() bool {
	return len(e.Deps) == 0
}

type externalConfigVersion struct {
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}

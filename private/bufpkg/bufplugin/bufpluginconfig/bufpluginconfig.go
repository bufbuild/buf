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
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
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
	// Options is the set of options available to the plugin.
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
	Go      *GoRuntimeConfig
	NPM     *NPMRuntimeConfig
	Archive *ArchiveRuntimeConfig
}

// GoRuntimeConfig is the runtime configuration for a Go plugin.
type GoRuntimeConfig struct {
	MinVersion string
	Deps       []string
}

// NPMRuntimeConfig is the runtime configuration for a JavaScript NPM plugin.
type NPMRuntimeConfig struct {
	Deps []string
}

// ArchiveRuntimeConfig is the runtime configuration for a plugin that can be downloaded
// as an archive instead of a language-specific registry.
type ArchiveRuntimeConfig struct {
	Deps []string
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

// RuntimeConfigForProto maps the given *registryv1alpha1.RuntimeConfig into a *RuntimeConfig.
//
// TODO: This function will need to change if/when we adjust the structure of runtime dependencies
// in the buf.plugin.yaml representation.
func RuntimeConfigForProto(protoRuntimeConfig *registryv1alpha1.RuntimeConfig) (*RuntimeConfig, error) {
	if protoRuntimeConfig == nil {
		return nil, nil
	}
	switch protoRuntimeConfig.RuntimeConfig.(type) {
	case *registryv1alpha1.RuntimeConfig_GoConfig:
		protoGoConfig := protoRuntimeConfig.GetGoConfig()
		if len(protoGoConfig.GetRuntimeLibraries()) == 0 && protoGoConfig.GetMinimumVersion() == "" {
			return nil, fmt.Errorf("the plugin's go runtime configuration must have a non-empty value")
		}
		var deps []string
		for _, protoRuntimeLibrary := range protoGoConfig.GetRuntimeLibraries() {
			// TODO: We probably need more validation here, but we should be wary
			// of client-side validation whenever possible.
			deps = append(deps, protoRuntimeLibrary.GetModule()+":"+protoRuntimeLibrary.GetVersion())
		}
		return &RuntimeConfig{
			Go: &GoRuntimeConfig{
				MinVersion: protoGoConfig.GetMinimumVersion(),
				Deps:       deps,
			},
		}, nil
	case *registryv1alpha1.RuntimeConfig_NpmConfig:
		protoNPMConfig := protoRuntimeConfig.GetNpmConfig()
		if len(protoNPMConfig.GetRuntimeLibraries()) == 0 {
			return nil, fmt.Errorf("the plugin's NPM runtime configuration must have a non-empty value")
		}
		var deps []string
		for _, protoRuntimeLibrary := range protoNPMConfig.GetRuntimeLibraries() {
			// TODO: We probably need more validation here, but we should be wary
			// of client-side validation whenever possible.
			deps = append(deps, protoRuntimeLibrary.GetPackage()+":"+protoRuntimeLibrary.GetVersion())
		}
		return &RuntimeConfig{
			NPM: &NPMRuntimeConfig{
				Deps: deps,
			},
		}, nil
	}
	// We'd normally return an error here, but this case will occur whenever the CLI interacts
	// with a plugin that defines a runtime configuration it doesn't know about. In other words,
	// if we ever add another runtime configuration to the *registryv1alpha1.RuntimeConfig later
	// (e.g. Maven), old CLIs will hit this case if they interact with a plugin that defines it.
	return nil, nil
}

// RuntimeConfigToProto maps the *RuntimeConfig to a *registryv1alpha1.RuntimeConfig.
func RuntimeConfigToProto(runtimeConfig *RuntimeConfig) (*registryv1alpha1.RuntimeConfig, error) {
	// TODO: Map the runtime configuration based on the dependency structure.
	return nil, nil
}

// ExternalConfig represents the on-disk representation
// of the plugin configuration at version v1.
type ExternalConfig struct {
	Version string                `json:"version,omitempty" yaml:"version,omitempty"`
	Name    string                `json:"name,omitempty" yaml:"name,omitempty"`
	Opts    []string              `json:"opts,omitempty" yaml:"opts,omitempty"`
	Runtime ExternalRuntimeConfig `json:"runtime,omitempty" yaml:"runtime,omitempty"`
}

// ExternalRuntimeConfig is the external configuration for the runtime
// of a plugin.
type ExternalRuntimeConfig struct {
	Go      ExternalGoRuntimeConfig      `json:"go,omitempty" yaml:"go,omitempty"`
	NPM     ExternalNPMRuntimeConfig     `json:"npm,omitempty" yaml:"npm,omitempty"`
	Archive ExternalArchiveRuntimeConfig `json:"archive,omitempty" yaml:"archive,omitempty"`
}

// ExternalGoRuntimeConfig is the external runtime configuration for a Go plugin.
type ExternalGoRuntimeConfig struct {
	// The minimum Go version required by the plugin.
	MinVersion string   `json:"min_version,omitempty" yaml:"min_version,omitempty"`
	Deps       []string `json:"deps,omitempty" yaml:"deps,omitempty"`
}

// IsEmpty returns true if the configuration is empty.
func (e ExternalGoRuntimeConfig) IsEmpty() bool {
	return e.MinVersion == "" && len(e.Deps) == 0
}

// ExternalNPMRuntimeConfig is the external runtime configuration for a JavaScript NPM plugin.
type ExternalNPMRuntimeConfig struct {
	Deps []string `json:"deps,omitempty" yaml:"deps,omitempty"`
}

// IsEmpty returns true if the configuration is empty.
func (e ExternalNPMRuntimeConfig) IsEmpty() bool {
	return len(e.Deps) == 0
}

// ExternalArchiveRuntimeConfig is the external runtime configuration for a plugin that can be
// downloaded as an archive instead of a language-specific registry.
type ExternalArchiveRuntimeConfig struct {
	Deps []string `json:"deps,omitempty" yaml:"deps,omitempty"`
}

// IsEmpty returns true if the configuration is empty.
func (e ExternalArchiveRuntimeConfig) IsEmpty() bool {
	return len(e.Deps) == 0
}

type externalConfigVersion struct {
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}

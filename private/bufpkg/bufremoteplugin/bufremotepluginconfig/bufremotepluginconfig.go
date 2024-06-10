// Copyright 2020-2024 Buf Technologies, Inc.
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

// Package bufremotepluginconfig defines the buf.plugin.yaml file.
package bufremotepluginconfig

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginref"
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
	Name bufremotepluginref.PluginIdentity
	// PluginVersion is the version of the plugin's implementation
	// (e.g. the protoc-gen-connect-go implementation is v0.2.0).
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
	Dependencies []bufremotepluginref.PluginReference
	// OutputLanguages is a list of output languages the plugin supports.
	OutputLanguages []string
	// Registry is the registry configuration, which lets the user specify
	// dependencies and other metadata that applies to a specific
	// remote generation registry (e.g. the Go module proxy, NPM registry,
	// etc).
	Registry *RegistryConfig
	// SPDXLicenseID is the license of the plugin, which should be one of
	// the identifiers defined in https://spdx.org/licenses
	SPDXLicenseID string
	// LicenseURL specifies where the plugin's license can be found.
	LicenseURL string
	// IntegrationGuideURL is an optional attribute used to specify where
	// the plugin integration guide can be found.
	IntegrationGuideURL string
}

// RegistryConfig is the configuration for the registry of a plugin.
//
// Only one field will be set.
type RegistryConfig struct {
	Go     *GoRegistryConfig
	NPM    *NPMRegistryConfig
	Maven  *MavenRegistryConfig
	Swift  *SwiftRegistryConfig
	Python *PythonRegistryConfig
	Cargo  *CargoRegistryConfig
	// Options is the set of options passed into the plugin for the
	// remote registry.
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
}

// GoRegistryConfig is the registry configuration for a Go plugin.
type GoRegistryConfig struct {
	MinVersion string
	Deps       []*GoRegistryDependencyConfig
}

// GoRegistryDependencyConfig is the go registry dependency configuration.
type GoRegistryDependencyConfig struct {
	Module  string
	Version string
}

// NPMRegistryConfig is the registry configuration for a JavaScript NPM plugin.
type NPMRegistryConfig struct {
	RewriteImportPathSuffix string
	Deps                    []*NPMRegistryDependencyConfig
	ImportStyle             string
}

// NPMRegistryDependencyConfig is the npm registry dependency configuration.
type NPMRegistryDependencyConfig struct {
	Package string
	Version string
}

// MavenRegistryConfig is the registry configuration for a Maven plugin.
type MavenRegistryConfig struct {
	// Compiler specifies Java and/or Kotlin compiler settings for remote packages.
	Compiler MavenCompilerConfig
	// Deps are dependencies for the remote package.
	Deps []MavenDependencyConfig
	// AdditionalRuntimes tracks additional runtimes (like the 'lite' runtime).
	// This is used to support multiple artifacts targeting different runtimes, plugin options, and dependencies.
	AdditionalRuntimes []MavenRuntimeConfig
}

// MavenCompilerConfig specifies compiler settings for Java and/or Kotlin.
type MavenCompilerConfig struct {
	Java   MavenCompilerJavaConfig
	Kotlin MavenCompilerKotlinConfig
}

// MavenCompilerJavaConfig specifies compiler settings for Java code.
type MavenCompilerJavaConfig struct {
	// Encoding specifies the encoding of the source files (default: UTF-8).
	Encoding string
	// Release specifies the target Java release (default: 8).
	Release int
	// Source specifies the source bytecode level (default: 8).
	Source int
	// Target specifies the target bytecode level (default: 8).
	Target int
}

// MavenCompilerKotlinConfig specifies compiler settings for Kotlin code.
type MavenCompilerKotlinConfig struct {
	// APIVersion specifies the Kotlin API version to target.
	APIVersion string
	// JVMTarget specifies the target version of the JVM bytecode (default: 1.8)
	JVMTarget string
	// LanguageVersion is used to provide source compatibility with the specified Kotlin version.
	LanguageVersion string
	// Version of the Kotlin compiler to use (required for Kotlin plugins).
	Version string
}

// MavenDependencyConfig defines a runtime dependency for a remote package artifact.
type MavenDependencyConfig struct {
	GroupID    string
	ArtifactID string
	Version    string
	Classifier string
	// Extension is the file extension, also known as the Maven type.
	Extension string
}

// MavenRuntimeConfig is used to specify additional runtimes for a given plugin.
type MavenRuntimeConfig struct {
	// Name is the required, unique name for the runtime in MavenRegistryConfig.AdditionalRuntimes.
	Name string
	// Deps contains the Maven dependencies for the runtime. Overrides MavenRegistryConfig.Deps.
	Deps []MavenDependencyConfig
	// Options contains the Maven plugin options for the runtime. Overrides RegistryConfig.Options.
	Options []string
}

// SwiftRegistryConfig is the registry configuration for a Swift plugin.
type SwiftRegistryConfig struct {
	// Dependencies are dependencies for the remote package.
	Dependencies []SwiftRegistryDependencyConfig
}

// SwiftRegistryDependencyConfig is the swift registry dependency configuration.
type SwiftRegistryDependencyConfig struct {
	// Source specifies the source of the dependency.
	Source string
	// Package is the name of the Swift package.
	Package string
	// Version is the version of the Swift package.
	Version string
	// Products are the names of the products available to import.
	Products []string
	// Platforms are the minimum versions for platforms the package supports.
	Platforms SwiftRegistryDependencyPlatformConfig
	// SwiftVersions are the versions of Swift the package supports.
	SwiftVersions []string
}

// SwiftRegistryDependencyPlatformConfig is the swift registry dependency platform configuration.
type SwiftRegistryDependencyPlatformConfig struct {
	// macOS specifies the version of the macOS platform.
	MacOS string
	// iOS specifies the version of the iOS platform.
	IOS string
	// TVOS specifies the version of the tvOS platform.
	TVOS string
	// WatchOS specifies the version of the watchOS platform.
	WatchOS string
}

// PythonRegistryConfig is the registry configuration for a Python plugin.
type PythonRegistryConfig struct {
	// Deps are the dependency specifications for the generated SDK.
	Deps []string
	// RequiresPython is the `Requires-Python` for the generated SDK.
	RequiresPython string
	// PackageType is the package type for the generated SDK.
	PackageType string
}

// CargoRegistryDependency defines a runtime dependency for a generated crate.
// It is the subset of a full Cargo dependency specification, which contains
// fields that are currently irrelevant for Generated SDKs.
// Ref: https://doc.rust-lang.org/cargo/reference/specifying-dependencies.html
type CargoRegistryDependency struct {
	// Name specifies the name of the dependency.
	Name string
	// VersionRequirement specifies the version requirement of the dependency.
	VersionRequirement string
	// DefaultFeatures specifies whether or not the default features will
	// be enabled for the dependency.
	DefaultFeatures bool
	// Features specifies the features enabled for the dependency.
	Features []string
}

// CargoRegistryConfig is the registry configuration for a Cargo plugin.
type CargoRegistryConfig struct {
	// RustVersion specifies the minimum supported Rust version (MSRV) for the generated crate.
	// Ref: https://doc.rust-lang.org/cargo/reference/manifest.html#the-rust-version-field
	RustVersion string
	// Deps specifies the runtime dependencies of the crate.
	Deps []CargoRegistryDependency
}

// ConfigOption is an optional option used when loading a Config.
type ConfigOption func(*configOptions)

// WithOverrideRemote will update the remote found in the plugin name and dependencies.
func WithOverrideRemote(remote string) ConfigOption {
	return func(options *configOptions) {
		options.overrideRemote = remote
	}
}

// GetConfigForBucket gets the Config for the YAML data at ConfigFilePath.
//
// If the data is of length 0, returns the default config.
func GetConfigForBucket(ctx context.Context, readBucket storage.ReadBucket, options ...ConfigOption) (*Config, error) {
	return getConfigForBucket(ctx, readBucket, options)
}

// GetConfigForData gets the Config for the given JSON or YAML data.
//
// If the data is of length 0, returns the default config.
func GetConfigForData(ctx context.Context, data []byte, options ...ConfigOption) (*Config, error) {
	return getConfigForData(ctx, data, options)
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
func ParseConfig(config string, options ...ConfigOption) (*Config, error) {
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
		return newConfig(externalConfig, options)
	}
	return nil, fmt.Errorf("invalid plugin configuration version: must be one of %v", AllConfigFilePaths)
}

// PluginOptionsToOptionsSlice converts a map representation of plugin options to a slice of the form '<key>=<value>' or '<key>' for empty values.
func PluginOptionsToOptionsSlice(pluginOptions map[string]string) []string {
	if pluginOptions == nil {
		return nil
	}
	options := make([]string, 0, len(pluginOptions))
	for key, value := range pluginOptions {
		if len(value) > 0 {
			options = append(options, key+"="+value)
		} else {
			options = append(options, key)
		}
	}
	sort.Strings(options)
	return options
}

// OptionsSliceToPluginOptions converts a slice of plugin options to a map (using the first '=' as a delimiter between key and value).
// If no '=' is found, the option will be stored in the map with an empty string value.
func OptionsSliceToPluginOptions(options []string) map[string]string {
	if options == nil {
		return nil
	}
	pluginOptions := make(map[string]string, len(options))
	for _, option := range options {
		fields := strings.SplitN(option, "=", 2)
		if len(fields) == 2 {
			pluginOptions[fields[0]] = fields[1]
		} else {
			pluginOptions[option] = ""
		}
	}
	return pluginOptions
}

// ExternalConfig represents the on-disk representation
// of the plugin configuration at version v1.
type ExternalConfig struct {
	Version             string                 `json:"version,omitempty" yaml:"version,omitempty"`
	Name                string                 `json:"name,omitempty" yaml:"name,omitempty"`
	PluginVersion       string                 `json:"plugin_version,omitempty" yaml:"plugin_version,omitempty"`
	SourceURL           string                 `json:"source_url,omitempty" yaml:"source_url,omitempty"`
	Description         string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Deps                []ExternalDependency   `json:"deps,omitempty" yaml:"deps,omitempty"`
	OutputLanguages     []string               `json:"output_languages,omitempty" yaml:"output_languages,omitempty"`
	Registry            ExternalRegistryConfig `json:"registry,omitempty" yaml:"registry,omitempty"`
	SPDXLicenseID       string                 `json:"spdx_license_id,omitempty" yaml:"spdx_license_id,omitempty"`
	LicenseURL          string                 `json:"license_url,omitempty" yaml:"license_url,omitempty"`
	IntegrationGuideURL string                 `json:"integration_guide_url,omitempty" yaml:"integration_guide_url,omitempty"`
}

// ExternalDependency represents a dependency on another plugin.
type ExternalDependency struct {
	Plugin   string `json:"plugin,omitempty" yaml:"plugin,omitempty"`
	Revision int    `json:"revision,omitempty" yaml:"revision,omitempty"`
}

// ExternalRegistryConfig is the external configuration for the registry
// of a plugin.
type ExternalRegistryConfig struct {
	Go     *ExternalGoRegistryConfig     `json:"go,omitempty" yaml:"go,omitempty"`
	NPM    *ExternalNPMRegistryConfig    `json:"npm,omitempty" yaml:"npm,omitempty"`
	Maven  *ExternalMavenRegistryConfig  `json:"maven,omitempty" yaml:"maven,omitempty"`
	Swift  *ExternalSwiftRegistryConfig  `json:"swift,omitempty" yaml:"swift,omitempty"`
	Python *ExternalPythonRegistryConfig `json:"python,omitempty" yaml:"python,omitempty"`
	Cargo  *ExternalCargoRegistryConfig  `json:"cargo,omitempty" yaml:"cargo,omitempty"`
	Opts   []string                      `json:"opts,omitempty" yaml:"opts,omitempty"`
}

// ExternalGoRegistryConfig is the external registry configuration for a Go plugin.
type ExternalGoRegistryConfig struct {
	// The minimum Go version required by the plugin.
	MinVersion string `json:"min_version,omitempty" yaml:"min_version,omitempty"`
	Deps       []struct {
		Module  string `json:"module,omitempty" yaml:"module,omitempty"`
		Version string `json:"version,omitempty" yaml:"version,omitempty"`
	} `json:"deps,omitempty" yaml:"deps,omitempty"`
}

// ExternalNPMRegistryConfig is the external registry configuration for a JavaScript NPM plugin.
type ExternalNPMRegistryConfig struct {
	RewriteImportPathSuffix string `json:"rewrite_import_path_suffix,omitempty" yaml:"rewrite_import_path_suffix,omitempty"`
	Deps                    []struct {
		Package string `json:"package,omitempty" yaml:"package,omitempty"`
		Version string `json:"version,omitempty" yaml:"version,omitempty"`
	} `json:"deps,omitempty" yaml:"deps,omitempty"`
	// The import style used for the "type" field in the package.json file.
	// Must be one of "module" or "commonjs".
	ImportStyle string `json:"import_style,omitempty" yaml:"import_style,omitempty"`
}

// ExternalMavenRegistryConfig is the external registry configuration for a Maven plugin.
type ExternalMavenRegistryConfig struct {
	Compiler           ExternalMavenCompilerConfig  `json:"compiler" yaml:"compiler"`
	Deps               []string                     `json:"deps,omitempty" yaml:"deps,omitempty"`
	AdditionalRuntimes []ExternalMavenRuntimeConfig `json:"additional_runtimes,omitempty" yaml:"additional_runtimes,omitempty"`
}

// ExternalMavenCompilerConfig configures compiler settings for Maven remote packages.
type ExternalMavenCompilerConfig struct {
	Java   ExternalMavenCompilerJavaConfig   `json:"java" yaml:"java"`
	Kotlin ExternalMavenCompilerKotlinConfig `json:"kotlin" yaml:"kotlin"`
}

// ExternalMavenCompilerJavaConfig configures the Java compiler settings for remote packages.
type ExternalMavenCompilerJavaConfig struct {
	// Encoding specifies the encoding of the source files (default: UTF-8).
	Encoding string `json:"encoding" yaml:"encoding"`
	// Release specifies the target Java release (default: 8).
	Release int `json:"release" yaml:"release"`
	// Source specifies the source bytecode level (default: 8).
	Source int `json:"source" yaml:"source"`
	// Target specifies the target bytecode level (default: 8).
	Target int `json:"target" yaml:"target"`
}

// ExternalMavenCompilerKotlinConfig configures the Kotlin compiler settings for remote packages.
type ExternalMavenCompilerKotlinConfig struct {
	// APIVersion specifies the Kotlin API version to target.
	APIVersion string `json:"api_version" yaml:"api_version"`
	// JVMTarget specifies the target version of the JVM bytecode (default: 1.8)
	JVMTarget string `json:"jvm_target" yaml:"jvm_target"`
	// LanguageVersion is used to provide source compatibility with the specified Kotlin version.
	LanguageVersion string `json:"language_version" yaml:"language_version"`
	// Version of the Kotlin compiler to use (required for Kotlin plugins).
	Version string `json:"version" yaml:"version"`
}

// ExternalMavenRuntimeConfig allows configuring additional runtimes for remote packages.
// These can specify different dependencies and compiler options than the default runtime.
// This is used to support a single plugin supporting both full and lite Protobuf runtimes.
type ExternalMavenRuntimeConfig struct {
	// Name contains the Maven runtime name (e.g. 'lite').
	Name string `json:"name" yaml:"name"`
	// Deps contains the Maven dependencies for the runtime. Overrides ExternalMavenRuntimeConfig.Deps.
	Deps []string `json:"deps,omitempty" yaml:"deps,omitempty"`
	// Opts contains the Maven plugin options for the runtime. Overrides ExternalRegistryConfig.Opts.
	Opts []string `json:"opts,omitempty" yaml:"opts,omitempty"`
}

// ExternalSwiftRegistryConfig is the registry configuration for a Swift plugin.
type ExternalSwiftRegistryConfig struct {
	// Deps are dependencies for the remote package.
	Deps []ExternalSwiftRegistryDependencyConfig `json:"deps,omitempty" yaml:"deps,omitempty"`
}

// ExternalSwiftRegistryDependencyConfig is the swift registry dependency configuration.
type ExternalSwiftRegistryDependencyConfig struct {
	// Source is the URL of the Swift package.
	Source string `json:"source,omitempty" yaml:"source,omitempty"`
	// Package is the name of the Swift package.
	Package string `json:"package,omitempty" yaml:"package,omitempty"`
	// Version is the version of the Swift package.
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
	// Products are the names of the products available to import.
	Products []string `json:"products,omitempty" yaml:"products,omitempty"`
	// Platforms are the minimum versions for platforms the package supports.
	Platforms ExternalSwiftRegistryDependencyPlatformConfig `json:"platforms,omitempty" yaml:"platforms,omitempty"`
	// SwiftVersions are the versions of Swift the package supports.
	SwiftVersions []string `json:"swift_versions,omitempty" yaml:"swift_versions,omitempty"`
}

// ExternalSwiftRegistryDependencyPlatformConfig is the swift registry dependency platform configuration.
type ExternalSwiftRegistryDependencyPlatformConfig struct {
	// macOS specifies the version of the macOS platform.
	MacOS string `json:"macos,omitempty" yaml:"macos,omitempty"`
	// iOS specifies the version of the iOS platform.
	IOS string `json:"ios,omitempty" yaml:"ios,omitempty"`
	// TVOS specifies the version of the tvOS platform.
	TVOS string `json:"tvos,omitempty" yaml:"tvos,omitempty"`
	// WatchOS specifies the version of the watchOS platform.
	WatchOS string `json:"watchos,omitempty" yaml:"watchos,omitempty"`
}

// ExternalPythonRegistryConfig is the registry configuration for a Python plugin.
type ExternalPythonRegistryConfig struct {
	// Deps are dependency specifications for the generated SDK.
	Deps []string `json:"deps,omitempty" yaml:"deps,omitempty"`
	// RequiresPython specifies the `Requires-Python` of the generated metadata file.
	RequiresPython string `json:"requires_python,omitempty" yaml:"requires_python,omitempty"`
	// PackageType is the type of package generated.
	// Must be one of "runtime" or "stub-only".
	PackageType string `json:"package_type,omitempty" yaml:"package_type,omitempty"`
}

// ExternalCargoDependency specifies a runtime dependency for a Cargo generated SDK.
type ExternalCargoDependency struct {
	// Name specifies the name of the dependency.
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	// VersionRequirement specifies the version requirement of the dependency.
	VersionRequirement string `json:"req,omitempty" yaml:"req,omitempty"`
	// DefaultFeatures specifies whether or not the default features will
	// be enabled for the dependency.
	DefaultFeatures bool `json:"default_features,omitempty" yaml:"default_features,omitempty"`
	// Features specifies the features enabled for the dependency.
	Features []string `json:"features,omitempty" yaml:"features,omitempty"`
}

// ExternalCargoRegistryConfig is the registry configuration for a Rust plugin.
type ExternalCargoRegistryConfig struct {
	// RustVersion specifies the minimum supported Rust version (MSRV) for the generated crate.
	// Ref: https://doc.rust-lang.org/cargo/reference/manifest.html#the-rust-version-field
	RustVersion string `json:"rust_version,omitempty" yaml:"rust_version,omitempty"`
	// Deps specifies the dependencies for the generated SDK.
	Deps []ExternalCargoDependency `json:"deps,omitempty" yaml:"deps,omitempty"`
}

type externalConfigVersion struct {
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}

type configOptions struct {
	overrideRemote string
}

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

// Package bufgen does configuration-based generation.
//
// It is used by the buf generate command.
package bufgen

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginref"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufwasm"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	// ExternalConfigFilePath is the default external configuration file path.
	ExternalConfigFilePath = "buf.gen.yaml"
	// V1Version is the string used to identify the v1 version of the generate template.
	V1Version = "v1"
	// V1Beta1Version is the string used to identify the v1beta1 version of the generate template.
	V1Beta1Version = "v1beta1"
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

// Provider is a provider.
type Provider interface {
	// GetConfig gets the Config for the YAML data at ExternalConfigFilePath.
	//
	// If the data is of length 0, returns the default config.
	GetConfig(ctx context.Context, readBucket storage.ReadBucket) (*Config, error)
}

// NewProvider returns a new Provider.
func NewProvider(logger *zap.Logger) Provider {
	return newProvider(logger)
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
		config *Config,
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

// Config is a configuration.
type Config struct {
	// Required
	PluginConfigs []*PluginConfig
	// Optional
	ManagedConfig *ManagedConfig
	// Optional
	TypesConfig *TypesConfig
}

// PluginConfig is a plugin configuration.
type PluginConfig struct {
	// One of Plugin, Name or Remote is required
	Plugin string
	Name   string
	Remote string
	// Optional, used with Plugin to pin a specific revision
	Revision int
	// Required
	Out string
	// Optional
	Opt string
	// Optional, exclusive with Remote
	Path []string
	// Required
	Strategy Strategy
	// Optional
	ProtocPath string
}

// PluginName returns this PluginConfig's plugin name.
// Only one of Plugin, Name or Remote will be set.
func (p *PluginConfig) PluginName() string {
	if p == nil {
		return ""
	}
	if p.Plugin != "" {
		return p.Plugin
	}
	if p.Name != "" {
		return p.Name
	}
	if p.Remote != "" {
		return p.Remote
	}
	return ""
}

// IsRemote returns true if the PluginConfig uses a remotely executed plugin.
func (p *PluginConfig) IsRemote() bool {
	return p.GetRemoteHostname() != ""
}

// GetRemoteHostname returns the hostname of the remote plugin.
func (p *PluginConfig) GetRemoteHostname() string {
	if p == nil {
		return ""
	}
	if identity, err := bufpluginref.PluginIdentityForString(p.Plugin); err == nil {
		return identity.Remote()
	}
	if reference, err := bufpluginref.PluginReferenceForString(p.Plugin, 0); err == nil {
		return reference.Remote()
	}
	if p.Remote == "" {
		return ""
	}
	if remote, _, _, _, err := bufremoteplugin.ParsePluginVersionPath(p.Remote); err == nil {
		return remote
	}
	return ""
}

// ManagedConfig is the managed mode configuration.
type ManagedConfig struct {
	CcEnableArenas          *bool
	JavaMultipleFiles       *bool
	JavaStringCheckUtf8     *bool
	JavaPackagePrefixConfig *JavaPackagePrefixConfig
	CsharpNameSpaceConfig   *CsharpNameSpaceConfig
	OptimizeForConfig       *OptimizeForConfig
	GoPackagePrefixConfig   *GoPackagePrefixConfig
	ObjcClassPrefixConfig   *ObjcClassPrefixConfig
	RubyPackageConfig       *RubyPackageConfig
	Override                map[string]map[string]string
}

// JavaPackagePrefixConfig is the java_package prefix configuration.
type JavaPackagePrefixConfig struct {
	Default string
	Except  []bufmoduleref.ModuleIdentity
	// bufmoduleref.ModuleIdentity -> java_package prefix.
	Override map[bufmoduleref.ModuleIdentity]string
}

type OptimizeForConfig struct {
	Default descriptorpb.FileOptions_OptimizeMode
	Except  []bufmoduleref.ModuleIdentity
	// bufmoduleref.ModuleIdentity -> optimize_for.
	Override map[bufmoduleref.ModuleIdentity]descriptorpb.FileOptions_OptimizeMode
}

// GoPackagePrefixConfig is the go_package prefix configuration.
type GoPackagePrefixConfig struct {
	Default string
	Except  []bufmoduleref.ModuleIdentity
	// bufmoduleref.ModuleIdentity -> go_package prefix.
	Override map[bufmoduleref.ModuleIdentity]string
}

// ObjcClassPrefixConfig is the objc_class_prefix configuration.
type ObjcClassPrefixConfig struct {
	Default string
	Except  []bufmoduleref.ModuleIdentity
	// bufmoduleref.ModuleIdentity -> objc_class_prefix.
	Override map[bufmoduleref.ModuleIdentity]string
}

// RubyPackgeConfig is the ruby_package configuration.
type RubyPackageConfig struct {
	Except []bufmoduleref.ModuleIdentity
	// bufmoduleref.ModuleIdentity -> ruby_package.
	Override map[bufmoduleref.ModuleIdentity]string
}

// CsharpNameSpaceConfig is the csharp_namespace configuration.
type CsharpNameSpaceConfig struct {
	Except []bufmoduleref.ModuleIdentity
	// bufmoduleref.ModuleIdentity -> csharp_namespace prefix.
	Override map[bufmoduleref.ModuleIdentity]string
}

// TypesConfig is a types configuration
type TypesConfig struct {
	Include []string
}

// ReadConfig reads the configuration from the OS or an override, if any.
//
// Only use in CLI tools.
func ReadConfig(
	ctx context.Context,
	logger *zap.Logger,
	provider Provider,
	readBucket storage.ReadBucket,
	options ...ReadConfigOption,
) (*Config, error) {
	return readConfig(
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

// ConfigExists checks if a generation configuration file exists.
func ConfigExists(ctx context.Context, readBucket storage.ReadBucket) (bool, error) {
	return storage.Exists(ctx, readBucket, ExternalConfigFilePath)
}

// ExternalConfigV1 is an external configuration.
type ExternalConfigV1 struct {
	Version string                   `json:"version,omitempty" yaml:"version,omitempty"`
	Plugins []ExternalPluginConfigV1 `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	Managed ExternalManagedConfigV1  `json:"managed,omitempty" yaml:"managed,omitempty"`
	Types   ExternalTypesConfigV1    `json:"types,omitempty" yaml:"types,omitempty"`
}

// ExternalPluginConfigV1 is an external plugin configuration.
type ExternalPluginConfigV1 struct {
	Plugin     string      `json:"plugin,omitempty" yaml:"plugin,omitempty"`
	Revision   int         `json:"revision,omitempty" yaml:"revision,omitempty"`
	Name       string      `json:"name,omitempty" yaml:"name,omitempty"`
	Remote     string      `json:"remote,omitempty" yaml:"remote,omitempty"`
	Out        string      `json:"out,omitempty" yaml:"out,omitempty"`
	Opt        interface{} `json:"opt,omitempty" yaml:"opt,omitempty"`
	Path       interface{} `json:"path,omitempty" yaml:"path,omitempty"`
	ProtocPath string      `json:"protoc_path,omitempty" yaml:"protoc_path,omitempty"`
	Strategy   string      `json:"strategy,omitempty" yaml:"strategy,omitempty"`
}

// ExternalManagedConfigV1 is an external managed mode configuration.
//
// Only use outside of this package for testing.
type ExternalManagedConfigV1 struct {
	Enabled             bool                              `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	CcEnableArenas      *bool                             `json:"cc_enable_arenas,omitempty" yaml:"cc_enable_arenas,omitempty"`
	JavaMultipleFiles   *bool                             `json:"java_multiple_files,omitempty" yaml:"java_multiple_files,omitempty"`
	JavaStringCheckUtf8 *bool                             `json:"java_string_check_utf8,omitempty" yaml:"java_string_check_utf8,omitempty"`
	JavaPackagePrefix   ExternalJavaPackagePrefixConfigV1 `json:"java_package_prefix,omitempty" yaml:"java_package_prefix,omitempty"`
	CsharpNamespace     ExternalCsharpNamespaceConfigV1   `json:"csharp_namespace,omitempty" yaml:"csharp_namespace,omitempty"`
	OptimizeFor         ExternalOptimizeForConfigV1       `json:"optimize_for,omitempty" yaml:"optimize_for,omitempty"`
	GoPackagePrefix     ExternalGoPackagePrefixConfigV1   `json:"go_package_prefix,omitempty" yaml:"go_package_prefix,omitempty"`
	ObjcClassPrefix     ExternalObjcClassPrefixConfigV1   `json:"objc_class_prefix,omitempty" yaml:"objc_class_prefix,omitempty"`
	RubyPackage         ExternalRubyPackageConfigV1       `json:"ruby_package,omitempty" yaml:"ruby_package,omitempty"`
	Override            map[string]map[string]string      `json:"override,omitempty" yaml:"override,omitempty"`
}

// IsEmpty returns true if the config is empty, excluding the 'Enabled' setting.
func (e ExternalManagedConfigV1) IsEmpty() bool {
	return e.CcEnableArenas == nil &&
		e.JavaMultipleFiles == nil &&
		e.JavaStringCheckUtf8 == nil &&
		e.JavaPackagePrefix.IsEmpty() &&
		e.CsharpNamespace.IsEmpty() &&
		e.CsharpNamespace.IsEmpty() &&
		e.OptimizeFor.IsEmpty() &&
		e.GoPackagePrefix.IsEmpty() &&
		e.ObjcClassPrefix.IsEmpty() &&
		e.RubyPackage.IsEmpty() &&
		len(e.Override) == 0
}

// ExternalJavaPackagePrefixConfigV1 is the external java_package prefix configuration.
type ExternalJavaPackagePrefixConfigV1 struct {
	Default  string            `json:"default,omitempty" yaml:"default,omitempty"`
	Except   []string          `json:"except,omitempty" yaml:"except,omitempty"`
	Override map[string]string `json:"override,omitempty" yaml:"override,omitempty"`
}

// IsEmpty returns true if the config is empty.
func (e ExternalJavaPackagePrefixConfigV1) IsEmpty() bool {
	return e.Default == "" &&
		len(e.Except) == 0 &&
		len(e.Override) == 0
}

// UnmarshalYAML satisfies the yaml.Unmarshaler interface. This is done to maintain backward compatibility
// of accepting a plain string value for java_package_prefix.
func (e *ExternalJavaPackagePrefixConfigV1) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return e.unmarshalWith(unmarshal)
}

// UnmarshalJSON satisfies the json.Unmarshaler interface. This is done to maintain backward compatibility
// of accepting a plain string value for java_package_prefix.
func (e *ExternalJavaPackagePrefixConfigV1) UnmarshalJSON(data []byte) error {
	unmarshal := func(v interface{}) error {
		return json.Unmarshal(data, v)
	}

	return e.unmarshalWith(unmarshal)
}

// unmarshalWith is used to unmarshal into json/yaml. See https://abhinavg.net/posts/flexible-yaml for details.
func (e *ExternalJavaPackagePrefixConfigV1) unmarshalWith(unmarshal func(interface{}) error) error {
	var prefix string
	if err := unmarshal(&prefix); err == nil {
		e.Default = prefix
		return nil
	}

	type rawExternalJavaPackagePrefixConfigV1 ExternalJavaPackagePrefixConfigV1
	if err := unmarshal((*rawExternalJavaPackagePrefixConfigV1)(e)); err != nil {
		return err
	}

	return nil
}

// ExternalOptimizeForConfigV1 is the external optimize_for configuration.
type ExternalOptimizeForConfigV1 struct {
	Default  string            `json:"default,omitempty" yaml:"default,omitempty"`
	Except   []string          `json:"except,omitempty" yaml:"except,omitempty"`
	Override map[string]string `json:"override,omitempty" yaml:"override,omitempty"`
}

// IsEmpty returns true if the config is empty
func (e ExternalOptimizeForConfigV1) IsEmpty() bool {
	return e.Default == "" &&
		len(e.Except) == 0 &&
		len(e.Override) == 0
}

// UnmarshalYAML satisfies the yaml.Unmarshaler interface. This is done to maintain backward compatibility
// of accepting a plain string value for optimize_for.
func (e *ExternalOptimizeForConfigV1) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return e.unmarshalWith(unmarshal)
}

// UnmarshalJSON satisfies the json.Unmarshaler interface. This is done to maintain backward compatibility
// of accepting a plain string value for optimize_for.
func (e *ExternalOptimizeForConfigV1) UnmarshalJSON(data []byte) error {
	unmarshal := func(v interface{}) error {
		return json.Unmarshal(data, v)
	}

	return e.unmarshalWith(unmarshal)
}

// unmarshalWith is used to unmarshal into json/yaml. See https://abhinavg.net/posts/flexible-yaml for details.
func (e *ExternalOptimizeForConfigV1) unmarshalWith(unmarshal func(interface{}) error) error {
	var optimizeFor string
	if err := unmarshal(&optimizeFor); err == nil {
		e.Default = optimizeFor
		return nil
	}

	type rawExternalOptimizeForConfigV1 ExternalOptimizeForConfigV1
	if err := unmarshal((*rawExternalOptimizeForConfigV1)(e)); err != nil {
		return err
	}

	return nil
}

// ExternalGoPackagePrefixConfigV1 is the external go_package prefix configuration.
type ExternalGoPackagePrefixConfigV1 struct {
	Default  string            `json:"default,omitempty" yaml:"default,omitempty"`
	Except   []string          `json:"except,omitempty" yaml:"except,omitempty"`
	Override map[string]string `json:"override,omitempty" yaml:"override,omitempty"`
}

// IsEmpty returns true if the config is empty.
func (e ExternalGoPackagePrefixConfigV1) IsEmpty() bool {
	return e.Default == "" &&
		len(e.Except) == 0 &&
		len(e.Override) == 0
}

// ExternalCsharpNamespaceConfigV1 is the external csharp_namespace configuration.
type ExternalCsharpNamespaceConfigV1 struct {
	Except   []string          `json:"except,omitempty" yaml:"except,omitempty"`
	Override map[string]string `json:"override,omitempty" yaml:"override,omitempty"`
}

// IsEmpty returns true if the config is empty.
func (e ExternalCsharpNamespaceConfigV1) IsEmpty() bool {
	return len(e.Except) == 0 &&
		len(e.Override) == 0
}

// ExternalRubyPackageConfigV1 is the external ruby_package configuration
type ExternalRubyPackageConfigV1 struct {
	Except   []string          `json:"except,omitempty" yaml:"except,omitempty"`
	Override map[string]string `json:"override,omitempty" yaml:"override,omitempty"`
}

// IsEmpty returns true is the config is empty
func (e ExternalRubyPackageConfigV1) IsEmpty() bool {
	return len(e.Except) == 0 && len(e.Override) == 0
}

// ExternalObjcClassPrefixConfigV1 is the external objc_class_prefix configuration.
type ExternalObjcClassPrefixConfigV1 struct {
	Default  string            `json:"default,omitempty" yaml:"default,omitempty"`
	Except   []string          `json:"except,omitempty" yaml:"except,omitempty"`
	Override map[string]string `json:"override,omitempty" yaml:"override,omitempty"`
}

func (e ExternalObjcClassPrefixConfigV1) IsEmpty() bool {
	return e.Default == "" &&
		len(e.Except) == 0 &&
		len(e.Override) == 0
}

// ExternalConfigV1Beta1 is an external configuration.
type ExternalConfigV1Beta1 struct {
	Version string                        `json:"version,omitempty" yaml:"version,omitempty"`
	Managed bool                          `json:"managed,omitempty" yaml:"managed,omitempty"`
	Plugins []ExternalPluginConfigV1Beta1 `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	Options ExternalOptionsConfigV1Beta1  `json:"options,omitempty" yaml:"options,omitempty"`
}

// ExternalPluginConfigV1Beta1 is an external plugin configuration.
type ExternalPluginConfigV1Beta1 struct {
	Name     string      `json:"name,omitempty" yaml:"name,omitempty"`
	Out      string      `json:"out,omitempty" yaml:"out,omitempty"`
	Opt      interface{} `json:"opt,omitempty" yaml:"opt,omitempty"`
	Path     string      `json:"path,omitempty" yaml:"path,omitempty"`
	Strategy string      `json:"strategy,omitempty" yaml:"strategy,omitempty"`
}

// ExternalOptionsConfigV1Beta1 is an external options configuration.
type ExternalOptionsConfigV1Beta1 struct {
	CcEnableArenas    *bool  `json:"cc_enable_arenas,omitempty" yaml:"cc_enable_arenas,omitempty"`
	JavaMultipleFiles *bool  `json:"java_multiple_files,omitempty" yaml:"java_multiple_files,omitempty"`
	OptimizeFor       string `json:"optimize_for,omitempty" yaml:"optimize_for,omitempty"`
}

// ExternalConfigVersion defines the subset of all config
// file versions that is used to determine the configuration version.
type ExternalConfigVersion struct {
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}

// ExternalTypesConfigV1 is an external types configuration.
type ExternalTypesConfigV1 struct {
	Include []string `json:"include,omitempty" yaml:"include"`
}

// IsEmpty returns true if e is empty.
func (e ExternalTypesConfigV1) IsEmpty() bool {
	return len(e.Include) == 0
}

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

// Package bufgen does configuration-based generation.
//
// It is used by the buf generate command.
package bufgenv1

import (
	"context"
	"encoding/json"

	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/buf/bufgen/internal"
	"github.com/bufbuild/buf/private/buf/bufgen/internal/plugingen"
	"github.com/bufbuild/buf/private/buf/bufwire"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/bufpkg/bufwasm"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/descriptorpb"
)

const defaultInput = "."

type Generator struct {
	logger            *zap.Logger
	generator         plugingen.Generator
	imageConfigReader bufwire.ImageConfigReader
	readWriteBucket   storage.ReadWriteBucket
}

func NewGenerator(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	runner command.Runner,
	wasmPluginExecutor *bufwasm.WASMPluginExecutor,
	clientConfig *connectclient.Config,
	imageConfigReader bufwire.ImageConfigReader,
	readWriteBucket storage.ReadWriteBucket,
) *Generator {
	return &Generator{
		logger: logger,
		generator: plugingen.NewGenerator(
			logger,
			storageosProvider,
			runner,
			wasmPluginExecutor,
			clientConfig,
		),
		imageConfigReader: imageConfigReader,
		readWriteBucket:   readWriteBucket,
	}
}

func (g *Generator) Generate(
	ctx context.Context,
	container appflag.Container,
	genTemplatePath string,
	moduleConfigPathOverride string,
	inputSpecified string,
	baseOutDir string,
	typesIncludedOverride []string,
	pathsSpecified []string,
	pathsExcluded []string,
	includeImports bool,
	includeWellKnownTypes bool,
	errorFormat string,
	fileAnnotationErr error,
	wasmEnabled bool,
) error {
	genConfig, err := readConfigV1(
		ctx,
		g.logger,
		g.readWriteBucket,
		internal.ReadConfigWithOverride(genTemplatePath),
	)
	if err != nil {
		return err
	}
	input := defaultInput
	if inputSpecified != "" {
		input = inputSpecified
	}
	inputRef, err := buffetch.NewRefParser(container.Logger()).GetRef(ctx, input)
	if err != nil {
		return err
	}
	var typesIncluded []string
	if typesConfig := genConfig.TypesConfig; typesConfig != nil {
		typesIncluded = typesConfig.Include
	}
	if len(typesIncludedOverride) > 0 {
		typesIncluded = typesIncludedOverride
	}
	inputImage, err := internal.GetInputImage(
		ctx,
		container,
		inputRef,
		g.imageConfigReader,
		moduleConfigPathOverride,
		pathsSpecified,
		pathsExcluded,
		errorFormat,
		typesIncluded,
		fileAnnotationErr,
	)
	if err != nil {
		return err
	}
	imageModifier, err := NewModifier(
		g.logger,
		genConfig,
	)
	if err != nil {
		return err
	}
	if err := imageModifier.Modify(
		ctx,
		inputImage,
	); err != nil {
		return err
	}
	generateOptions := []plugingen.GenerateOption{
		plugingen.GenerateWithBaseOutDirPath(baseOutDir),
	}
	if includeImports {
		generateOptions = append(
			generateOptions,
			plugingen.GenerateWithAlwaysIncludeImports(),
		)
	}
	if includeWellKnownTypes {
		generateOptions = append(
			generateOptions,
			plugingen.GenerateWithAlwaysIncludeWellKnownTypes(),
		)
	}
	if wasmEnabled {
		generateOptions = append(
			generateOptions,
			plugingen.GenerateWithWASMEnabled(),
		)
	}
	if err := g.generator.Generate(
		ctx,
		container,
		genConfig.PluginConfigs,
		inputImage,
		generateOptions...,
	); err != nil {
		return err
	}
	return nil
}

// Config is a configuration.
type Config struct {
	// Required
	PluginConfigs []plugingen.PluginConfig
	// Optional
	ManagedConfig *ManagedConfig
	// Optional
	TypesConfig *TypesConfig
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

// ExternalTypesConfigV1 is an external types configuration.
type ExternalTypesConfigV1 struct {
	Include []string `json:"include,omitempty" yaml:"include"`
}

// IsEmpty returns true if e is empty.
func (e ExternalTypesConfigV1) IsEmpty() bool {
	return len(e.Include) == 0
}

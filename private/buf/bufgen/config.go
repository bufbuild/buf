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

package bufgen

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginref"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/descriptorpb"
)

func readConfig(
	ctx context.Context,
	logger *zap.Logger,
	provider Provider,
	readBucket storage.ReadBucket,
	options ...ReadConfigOption,
) (*Config, error) {
	readConfigOptions := newReadConfigOptions()
	for _, option := range options {
		option(readConfigOptions)
	}
	if override := readConfigOptions.override; override != "" {
		switch filepath.Ext(override) {
		case ".json":
			return getConfigJSONFile(logger, override)
		case ".yaml", ".yml":
			return getConfigYAMLFile(logger, override)
		default:
			return getConfigJSONOrYAMLData(logger, override)
		}
	}
	return provider.GetConfig(ctx, readBucket)
}

func getConfigJSONFile(logger *zap.Logger, file string) (*Config, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("could not read file %s: %v", file, err)
	}
	return getConfig(
		logger,
		encoding.UnmarshalJSONNonStrict,
		encoding.UnmarshalJSONStrict,
		data,
		file,
	)
}

func getConfigYAMLFile(logger *zap.Logger, file string) (*Config, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("could not read file %s: %v", file, err)
	}
	return getConfig(
		logger,
		encoding.UnmarshalYAMLNonStrict,
		encoding.UnmarshalYAMLStrict,
		data,
		file,
	)
}

func getConfigJSONOrYAMLData(logger *zap.Logger, data string) (*Config, error) {
	return getConfig(
		logger,
		encoding.UnmarshalJSONOrYAMLNonStrict,
		encoding.UnmarshalJSONOrYAMLStrict,
		[]byte(data),
		"Generate configuration data",
	)
}

func getConfig(
	logger *zap.Logger,
	unmarshalNonStrict func([]byte, interface{}) error,
	unmarshalStrict func([]byte, interface{}) error,
	data []byte,
	id string,
) (*Config, error) {
	var externalConfigVersion ExternalConfigVersion
	if err := unmarshalNonStrict(data, &externalConfigVersion); err != nil {
		return nil, err
	}
	switch externalConfigVersion.Version {
	case V1Beta1Version:
		var externalConfigV1Beta1 ExternalConfigV1Beta1
		if err := unmarshalStrict(data, &externalConfigV1Beta1); err != nil {
			return nil, err
		}
		if err := validateExternalConfigV1Beta1(externalConfigV1Beta1, id); err != nil {
			return nil, err
		}
		return newConfigV1Beta1(externalConfigV1Beta1, id)
	case V1Version:
		var externalConfigV1 ExternalConfigV1
		if err := unmarshalStrict(data, &externalConfigV1); err != nil {
			return nil, err
		}
		if err := validateExternalConfigV1(externalConfigV1, id); err != nil {
			return nil, err
		}
		return newConfigV1(logger, externalConfigV1, id)
	default:
		return nil, fmt.Errorf(`%s has no version set. Please add "version: %s"`, id, V1Version)
	}
}

func newConfigV1(logger *zap.Logger, externalConfig ExternalConfigV1, id string) (*Config, error) {
	managedConfig, err := newManagedConfigV1(logger, externalConfig.Managed)
	if err != nil {
		return nil, err
	}
	pluginConfigs := make([]*PluginConfig, 0, len(externalConfig.Plugins))
	for _, plugin := range externalConfig.Plugins {
		strategy, err := ParseStrategy(plugin.Strategy)
		if err != nil {
			return nil, err
		}
		opt, err := encoding.InterfaceSliceOrStringToCommaSepString(plugin.Opt)
		if err != nil {
			return nil, err
		}
		path, err := encoding.InterfaceSliceOrStringToStringSlice(plugin.Path)
		if err != nil {
			return nil, err
		}
		pluginConfig := &PluginConfig{
			Plugin:     plugin.Plugin,
			Revision:   plugin.Revision,
			Name:       plugin.Name,
			Remote:     plugin.Remote,
			Out:        plugin.Out,
			Opt:        opt,
			Path:       path,
			ProtocPath: plugin.ProtocPath,
			Strategy:   strategy,
		}
		if pluginConfig.IsRemote() {
			// Always use StrategyAll for remote plugins
			pluginConfig.Strategy = StrategyAll
		}
		pluginConfigs = append(pluginConfigs, pluginConfig)
	}
	typesConfig := newTypesConfigV1(externalConfig.Types)
	return &Config{
		PluginConfigs: pluginConfigs,
		ManagedConfig: managedConfig,
		TypesConfig:   typesConfig,
	}, nil
}

func validateExternalConfigV1(externalConfig ExternalConfigV1, id string) error {
	if len(externalConfig.Plugins) == 0 {
		return fmt.Errorf("%s: no plugins set", id)
	}
	for _, plugin := range externalConfig.Plugins {
		var numPluginIdentifiers int
		var pluginIdentifier string
		for _, possibleIdentifier := range []string{plugin.Plugin, plugin.Name, plugin.Remote} {
			if possibleIdentifier != "" {
				numPluginIdentifiers++
				// Doesn't matter if we reassign here - we only allow one to be set below
				pluginIdentifier = possibleIdentifier
			}
		}
		if numPluginIdentifiers == 0 {
			return fmt.Errorf("%s: one of plugin, name or remote is required", id)
		}
		if numPluginIdentifiers > 1 {
			return fmt.Errorf("%s: only one of plugin, name, or remote can be set", id)
		}
		if plugin.Out == "" {
			return fmt.Errorf("%s: plugin %s out is required", id, pluginIdentifier)
		}
		switch {
		case plugin.Plugin != "":
			if bufpluginref.IsPluginReferenceOrIdentity(pluginIdentifier) {
				// plugin.Plugin is a remote plugin
				if err := checkPathAndStrategyUnset(id, plugin, pluginIdentifier); err != nil {
					return err
				}
			} else {
				// plugin.Plugin is a local plugin - verify it isn't using an alpha remote plugin path
				if _, _, _, _, err := bufremoteplugin.ParsePluginVersionPath(pluginIdentifier); err == nil {
					return fmt.Errorf("%s: invalid plugin reference: %s", id, pluginIdentifier)
				}
			}
		case plugin.Remote != "":
			// Remote generation alpha features have been deprecated, but we continue to detect
			// the remote field to surface a better error message.
			return fmt.Errorf("%s: the remote field no longer works as the remote generation alpha has been deprecated, see the migration guide to now-stable remote plugins: %s",
				id,
				"https://buf.build/docs/migration-guides/migrate-remote-generation-alpha/#migrate-to-remote-plugins",
			)
		case plugin.Name != "":
			// Check that the plugin name doesn't look like a plugin reference
			if bufpluginref.IsPluginReferenceOrIdentity(pluginIdentifier) {
				return fmt.Errorf("%s: invalid local plugin name: %s", id, pluginIdentifier)
			}
			// Check that the plugin name doesn't look like an alpha remote plugin
			if _, _, _, _, err := bufremoteplugin.ParsePluginVersionPath(pluginIdentifier); err == nil {
				return fmt.Errorf("%s: invalid plugin name %s, did you mean to use a remote plugin?", id, pluginIdentifier)
			}
		default:
			// unreachable - validated above
			return errors.New("one of plugin, name, or remote is required")
		}
	}
	return nil
}

func checkPathAndStrategyUnset(id string, plugin ExternalPluginConfigV1, pluginIdentifier string) error {
	if plugin.Path != nil {
		return fmt.Errorf("%s: remote plugin %s cannot specify a path", id, pluginIdentifier)
	}
	if plugin.Strategy != "" {
		return fmt.Errorf("%s: remote plugin %s cannot specify a strategy", id, pluginIdentifier)
	}
	if plugin.ProtocPath != "" {
		return fmt.Errorf("%s: remote plugin %s cannot specify a protoc path", id, pluginIdentifier)
	}
	return nil
}

func newManagedConfigV1(logger *zap.Logger, externalManagedConfig ExternalManagedConfigV1) (*ManagedConfig, error) {
	if !externalManagedConfig.Enabled {
		if !externalManagedConfig.IsEmpty() && logger != nil {
			logger.Sugar().Warn("managed mode options are set but are not enabled")
		}
		return nil, nil
	}
	javaPackagePrefixConfig, err := newJavaPackagePrefixConfigV1(externalManagedConfig.JavaPackagePrefix)
	if err != nil {
		return nil, err
	}
	csharpNamespaceConfig, err := newCsharpNamespaceConfigV1(externalManagedConfig.CsharpNamespace)
	if err != nil {
		return nil, err
	}
	optimizeForConfig, err := newOptimizeForConfigV1(externalManagedConfig.OptimizeFor)
	if err != nil {
		return nil, err
	}
	goPackagePrefixConfig, err := newGoPackagePrefixConfigV1(externalManagedConfig.GoPackagePrefix)
	if err != nil {
		return nil, err
	}
	objcClassPrefixConfig, err := newObjcClassPrefixConfigV1(externalManagedConfig.ObjcClassPrefix)
	if err != nil {
		return nil, err
	}
	rubyPackageConfig, err := newRubyPackageConfigV1(externalManagedConfig.RubyPackage)
	if err != nil {
		return nil, err
	}
	override := externalManagedConfig.Override
	for overrideID, overrideValue := range override {
		for importPath := range overrideValue {
			normalizedImportPath, err := normalpath.NormalizeAndValidate(importPath)
			if err != nil {
				return nil, fmt.Errorf(
					"failed to normalize import path: %s provided for override: %s",
					importPath,
					overrideID,
				)
			}
			if importPath != normalizedImportPath {
				return nil, fmt.Errorf(
					"override can only take normalized import paths, invalid import path: %s provided for override: %s",
					importPath,
					overrideID,
				)
			}
		}
	}
	return &ManagedConfig{
		CcEnableArenas:          externalManagedConfig.CcEnableArenas,
		JavaMultipleFiles:       externalManagedConfig.JavaMultipleFiles,
		JavaStringCheckUtf8:     externalManagedConfig.JavaStringCheckUtf8,
		JavaPackagePrefixConfig: javaPackagePrefixConfig,
		CsharpNameSpaceConfig:   csharpNamespaceConfig,
		OptimizeForConfig:       optimizeForConfig,
		GoPackagePrefixConfig:   goPackagePrefixConfig,
		ObjcClassPrefixConfig:   objcClassPrefixConfig,
		RubyPackageConfig:       rubyPackageConfig,
		Override:                override,
	}, nil
}

func newJavaPackagePrefixConfigV1(externalJavaPackagePrefixConfig ExternalJavaPackagePrefixConfigV1) (*JavaPackagePrefixConfig, error) {
	if externalJavaPackagePrefixConfig.IsEmpty() {
		return nil, nil
	}
	if externalJavaPackagePrefixConfig.Default == "" {
		return nil, errors.New("java_package_prefix setting requires a default value")
	}
	seenModuleIdentities := make(map[string]struct{}, len(externalJavaPackagePrefixConfig.Except))
	except := make([]bufmoduleref.ModuleIdentity, 0, len(externalJavaPackagePrefixConfig.Except))
	for _, moduleName := range externalJavaPackagePrefixConfig.Except {
		moduleIdentity, err := bufmoduleref.ModuleIdentityForString(moduleName)
		if err != nil {
			return nil, fmt.Errorf("invalid java_package_prefix except: %w", err)
		}
		if _, ok := seenModuleIdentities[moduleIdentity.IdentityString()]; ok {
			return nil, fmt.Errorf("invalid java_package_prefix except: %q is defined multiple times", moduleIdentity.IdentityString())
		}
		seenModuleIdentities[moduleIdentity.IdentityString()] = struct{}{}
		except = append(except, moduleIdentity)
	}
	override := make(map[bufmoduleref.ModuleIdentity]string, len(externalJavaPackagePrefixConfig.Override))
	for moduleName, javaPackagePrefix := range externalJavaPackagePrefixConfig.Override {
		moduleIdentity, err := bufmoduleref.ModuleIdentityForString(moduleName)
		if err != nil {
			return nil, fmt.Errorf("invalid java_package_prefix override key: %w", err)
		}
		if _, ok := seenModuleIdentities[moduleIdentity.IdentityString()]; ok {
			return nil, fmt.Errorf("invalid java_package_prefix override: %q is already defined as an except", moduleIdentity.IdentityString())
		}
		seenModuleIdentities[moduleIdentity.IdentityString()] = struct{}{}
		override[moduleIdentity] = javaPackagePrefix
	}
	return &JavaPackagePrefixConfig{
		Default:  externalJavaPackagePrefixConfig.Default,
		Except:   except,
		Override: override,
	}, nil
}

func newOptimizeForConfigV1(externalOptimizeForConfigV1 ExternalOptimizeForConfigV1) (*OptimizeForConfig, error) {
	if externalOptimizeForConfigV1.IsEmpty() {
		return nil, nil
	}
	if externalOptimizeForConfigV1.Default == "" {
		return nil, errors.New("optimize_for setting requires a default value")
	}
	value, ok := descriptorpb.FileOptions_OptimizeMode_value[externalOptimizeForConfigV1.Default]
	if !ok {
		return nil, fmt.Errorf(
			"invalid optimize_for default value; expected one of %v",
			enumMapToStringSlice(descriptorpb.FileOptions_OptimizeMode_value),
		)
	}
	defaultOptimizeFor := descriptorpb.FileOptions_OptimizeMode(value)
	seenModuleIdentities := make(map[string]struct{}, len(externalOptimizeForConfigV1.Except))
	except := make([]bufmoduleref.ModuleIdentity, 0, len(externalOptimizeForConfigV1.Except))
	for _, moduleName := range externalOptimizeForConfigV1.Except {
		moduleIdentity, err := bufmoduleref.ModuleIdentityForString(moduleName)
		if err != nil {
			return nil, fmt.Errorf("invalid optimize_for except: %w", err)
		}
		if _, ok := seenModuleIdentities[moduleIdentity.IdentityString()]; ok {
			return nil, fmt.Errorf("invalid optimize_for except: %q is defined multiple times", moduleIdentity.IdentityString())
		}
		seenModuleIdentities[moduleIdentity.IdentityString()] = struct{}{}
		except = append(except, moduleIdentity)
	}
	override := make(map[bufmoduleref.ModuleIdentity]descriptorpb.FileOptions_OptimizeMode, len(externalOptimizeForConfigV1.Override))
	for moduleName, optimizeFor := range externalOptimizeForConfigV1.Override {
		moduleIdentity, err := bufmoduleref.ModuleIdentityForString(moduleName)
		if err != nil {
			return nil, fmt.Errorf("invalid optimize_for override key: %w", err)
		}
		value, ok := descriptorpb.FileOptions_OptimizeMode_value[optimizeFor]
		if !ok {
			return nil, fmt.Errorf(
				"invalid optimize_for override value; expected one of %v",
				enumMapToStringSlice(descriptorpb.FileOptions_OptimizeMode_value),
			)
		}
		if _, ok := seenModuleIdentities[moduleIdentity.IdentityString()]; ok {
			return nil, fmt.Errorf(
				"invalid optimize_for override: %q is already defined as an except",
				moduleIdentity.IdentityString(),
			)
		}
		seenModuleIdentities[moduleIdentity.IdentityString()] = struct{}{}
		override[moduleIdentity] = descriptorpb.FileOptions_OptimizeMode(value)
	}
	return &OptimizeForConfig{
		Default:  defaultOptimizeFor,
		Except:   except,
		Override: override,
	}, nil
}

func newGoPackagePrefixConfigV1(externalGoPackagePrefixConfig ExternalGoPackagePrefixConfigV1) (*GoPackagePrefixConfig, error) {
	if externalGoPackagePrefixConfig.IsEmpty() {
		return nil, nil
	}
	if externalGoPackagePrefixConfig.Default == "" {
		return nil, errors.New("go_package_prefix setting requires a default value")
	}
	defaultGoPackagePrefix, err := normalpath.NormalizeAndValidate(externalGoPackagePrefixConfig.Default)
	if err != nil {
		return nil, fmt.Errorf("invalid go_package_prefix default: %w", err)
	}
	seenModuleIdentities := make(map[string]struct{}, len(externalGoPackagePrefixConfig.Except))
	except := make([]bufmoduleref.ModuleIdentity, 0, len(externalGoPackagePrefixConfig.Except))
	for _, moduleName := range externalGoPackagePrefixConfig.Except {
		moduleIdentity, err := bufmoduleref.ModuleIdentityForString(moduleName)
		if err != nil {
			return nil, fmt.Errorf("invalid go_package_prefix except: %w", err)
		}
		if _, ok := seenModuleIdentities[moduleIdentity.IdentityString()]; ok {
			return nil, fmt.Errorf("invalid go_package_prefix except: %q is defined multiple times", moduleIdentity.IdentityString())
		}
		seenModuleIdentities[moduleIdentity.IdentityString()] = struct{}{}
		except = append(except, moduleIdentity)
	}
	override := make(map[bufmoduleref.ModuleIdentity]string, len(externalGoPackagePrefixConfig.Override))
	for moduleName, goPackagePrefix := range externalGoPackagePrefixConfig.Override {
		moduleIdentity, err := bufmoduleref.ModuleIdentityForString(moduleName)
		if err != nil {
			return nil, fmt.Errorf("invalid go_package_prefix override key: %w", err)
		}
		normalizedGoPackagePrefix, err := normalpath.NormalizeAndValidate(goPackagePrefix)
		if err != nil {
			return nil, fmt.Errorf("invalid go_package_prefix override value: %w", err)
		}
		if _, ok := seenModuleIdentities[moduleIdentity.IdentityString()]; ok {
			return nil, fmt.Errorf("invalid go_package_prefix override: %q is already defined as an except", moduleIdentity.IdentityString())
		}
		seenModuleIdentities[moduleIdentity.IdentityString()] = struct{}{}
		override[moduleIdentity] = normalizedGoPackagePrefix
	}
	return &GoPackagePrefixConfig{
		Default:  defaultGoPackagePrefix,
		Except:   except,
		Override: override,
	}, nil
}

func newRubyPackageConfigV1(
	externalRubyPackageConfig ExternalRubyPackageConfigV1,
) (*RubyPackageConfig, error) {
	if externalRubyPackageConfig.IsEmpty() {
		return nil, nil
	}
	seenModuleIdentities := make(map[string]struct{}, len(externalRubyPackageConfig.Except))
	except := make([]bufmoduleref.ModuleIdentity, 0, len(externalRubyPackageConfig.Except))
	for _, moduleName := range externalRubyPackageConfig.Except {
		moduleIdentity, err := bufmoduleref.ModuleIdentityForString(moduleName)
		if err != nil {
			return nil, fmt.Errorf("invalid ruby_package except: %w", err)
		}
		if _, ok := seenModuleIdentities[moduleIdentity.IdentityString()]; ok {
			return nil, fmt.Errorf("invalid ruby_package except: %q is defined multiple times", moduleIdentity.IdentityString())
		}
		seenModuleIdentities[moduleIdentity.IdentityString()] = struct{}{}
		except = append(except, moduleIdentity)
	}
	override := make(map[bufmoduleref.ModuleIdentity]string, len(externalRubyPackageConfig.Override))
	for moduleName, rubyPackage := range externalRubyPackageConfig.Override {
		moduleIdentity, err := bufmoduleref.ModuleIdentityForString(moduleName)
		if err != nil {
			return nil, fmt.Errorf("invalid ruby_package override key: %w", err)
		}
		if _, ok := seenModuleIdentities[moduleIdentity.IdentityString()]; ok {
			return nil, fmt.Errorf("invalid ruby_package override: %q is already defined as an except", moduleIdentity.IdentityString())
		}
		seenModuleIdentities[moduleIdentity.IdentityString()] = struct{}{}
		override[moduleIdentity] = rubyPackage
	}
	return &RubyPackageConfig{
		Except:   except,
		Override: override,
	}, nil
}

func newCsharpNamespaceConfigV1(
	externalCsharpNamespaceConfig ExternalCsharpNamespaceConfigV1,
) (*CsharpNameSpaceConfig, error) {
	if externalCsharpNamespaceConfig.IsEmpty() {
		return nil, nil
	}
	seenModuleIdentities := make(map[string]struct{}, len(externalCsharpNamespaceConfig.Except))
	except := make([]bufmoduleref.ModuleIdentity, 0, len(externalCsharpNamespaceConfig.Except))
	for _, moduleName := range externalCsharpNamespaceConfig.Except {
		moduleIdentity, err := bufmoduleref.ModuleIdentityForString(moduleName)
		if err != nil {
			return nil, fmt.Errorf("invalid csharp_namespace except: %w", err)
		}
		if _, ok := seenModuleIdentities[moduleIdentity.IdentityString()]; ok {
			return nil, fmt.Errorf("invalid csharp_namespace except: %q is defined multiple times", moduleIdentity.IdentityString())
		}
		seenModuleIdentities[moduleIdentity.IdentityString()] = struct{}{}
		except = append(except, moduleIdentity)
	}
	override := make(map[bufmoduleref.ModuleIdentity]string, len(externalCsharpNamespaceConfig.Override))
	for moduleName, csharpNamespace := range externalCsharpNamespaceConfig.Override {
		moduleIdentity, err := bufmoduleref.ModuleIdentityForString(moduleName)
		if err != nil {
			return nil, fmt.Errorf("invalid csharp_namespace override key: %w", err)
		}
		if _, ok := seenModuleIdentities[moduleIdentity.IdentityString()]; ok {
			return nil, fmt.Errorf("invalid csharp_namespace override: %q is already defined as an except", moduleIdentity.IdentityString())
		}
		seenModuleIdentities[moduleIdentity.IdentityString()] = struct{}{}
		override[moduleIdentity] = csharpNamespace
	}
	return &CsharpNameSpaceConfig{
		Except:   except,
		Override: override,
	}, nil
}

func newObjcClassPrefixConfigV1(externalObjcClassPrefixConfig ExternalObjcClassPrefixConfigV1) (*ObjcClassPrefixConfig, error) {
	if externalObjcClassPrefixConfig.IsEmpty() {
		return nil, nil
	}
	// It's ok to have an empty default, which will have the same effect as previously enabling managed mode.
	defaultObjcClassPrefix := externalObjcClassPrefixConfig.Default
	seenModuleIdentities := make(map[string]struct{}, len(externalObjcClassPrefixConfig.Except))
	except := make([]bufmoduleref.ModuleIdentity, 0, len(externalObjcClassPrefixConfig.Except))
	for _, moduleName := range externalObjcClassPrefixConfig.Except {
		moduleIdentity, err := bufmoduleref.ModuleIdentityForString(moduleName)
		if err != nil {
			return nil, fmt.Errorf("invalid objc_class_prefix except: %w", err)
		}
		if _, ok := seenModuleIdentities[moduleIdentity.IdentityString()]; ok {
			return nil, fmt.Errorf("invalid objc_class_prefix except: %q is defined multiple times", moduleIdentity.IdentityString())
		}
		seenModuleIdentities[moduleIdentity.IdentityString()] = struct{}{}
		except = append(except, moduleIdentity)
	}
	override := make(map[bufmoduleref.ModuleIdentity]string, len(externalObjcClassPrefixConfig.Override))
	for moduleName, objcClassPrefix := range externalObjcClassPrefixConfig.Override {
		moduleIdentity, err := bufmoduleref.ModuleIdentityForString(moduleName)
		if err != nil {
			return nil, fmt.Errorf("invalid objc_class_prefix override key: %w", err)
		}
		if _, ok := seenModuleIdentities[moduleIdentity.IdentityString()]; ok {
			return nil, fmt.Errorf("invalid objc_class_prefix override: %q is already defined as an except", moduleIdentity.IdentityString())
		}
		seenModuleIdentities[moduleIdentity.IdentityString()] = struct{}{}
		override[moduleIdentity] = objcClassPrefix
	}
	return &ObjcClassPrefixConfig{
		Default:  defaultObjcClassPrefix,
		Except:   except,
		Override: override,
	}, nil
}

func newConfigV1Beta1(externalConfig ExternalConfigV1Beta1, id string) (*Config, error) {
	managedConfig, err := newManagedConfigV1Beta1(externalConfig.Options, externalConfig.Managed)
	if err != nil {
		return nil, err
	}
	pluginConfigs := make([]*PluginConfig, 0, len(externalConfig.Plugins))
	for _, plugin := range externalConfig.Plugins {
		strategy, err := ParseStrategy(plugin.Strategy)
		if err != nil {
			return nil, err
		}
		opt, err := encoding.InterfaceSliceOrStringToCommaSepString(plugin.Opt)
		if err != nil {
			return nil, err
		}
		pluginConfig := &PluginConfig{
			Name:     plugin.Name,
			Out:      plugin.Out,
			Opt:      opt,
			Strategy: strategy,
		}
		if plugin.Path != "" {
			pluginConfig.Path = []string{plugin.Path}
		}
		pluginConfigs = append(pluginConfigs, pluginConfig)
	}
	return &Config{
		PluginConfigs: pluginConfigs,
		ManagedConfig: managedConfig,
	}, nil
}

func validateExternalConfigV1Beta1(externalConfig ExternalConfigV1Beta1, id string) error {
	if len(externalConfig.Plugins) == 0 {
		return fmt.Errorf("%s: no plugins set", id)
	}
	for _, plugin := range externalConfig.Plugins {
		if plugin.Name == "" {
			return fmt.Errorf("%s: plugin name is required", id)
		}
		if plugin.Out == "" {
			return fmt.Errorf("%s: plugin %s out is required", id, plugin.Name)
		}
	}
	return nil
}

func newManagedConfigV1Beta1(externalOptionsConfig ExternalOptionsConfigV1Beta1, enabled bool) (*ManagedConfig, error) {
	if !enabled || externalOptionsConfig == (ExternalOptionsConfigV1Beta1{}) {
		return nil, nil
	}
	var optimizeForConfig *OptimizeForConfig
	if externalOptionsConfig.OptimizeFor != "" {
		value, ok := descriptorpb.FileOptions_OptimizeMode_value[externalOptionsConfig.OptimizeFor]
		if !ok {
			return nil, fmt.Errorf(
				"invalid optimize_for value; expected one of %v",
				enumMapToStringSlice(descriptorpb.FileOptions_OptimizeMode_value),
			)
		}
		optimizeForConfig = &OptimizeForConfig{
			Default:  descriptorpb.FileOptions_OptimizeMode(value),
			Except:   make([]bufmoduleref.ModuleIdentity, 0),
			Override: make(map[bufmoduleref.ModuleIdentity]descriptorpb.FileOptions_OptimizeMode),
		}
	}
	return &ManagedConfig{
		CcEnableArenas:    externalOptionsConfig.CcEnableArenas,
		JavaMultipleFiles: externalOptionsConfig.JavaMultipleFiles,
		OptimizeForConfig: optimizeForConfig,
	}, nil
}

// enumMapToStringSlice is a convenience function for mapping Protobuf enums
// into a slice of strings.
func enumMapToStringSlice(enums map[string]int32) []string {
	slice := make([]string, 0, len(enums))
	for enum := range enums {
		slice = append(slice, enum)
	}
	return slice
}

type readConfigOptions struct {
	override string
}

func newReadConfigOptions() *readConfigOptions {
	return &readConfigOptions{}
}

func newTypesConfigV1(externalConfig ExternalTypesConfigV1) *TypesConfig {
	if externalConfig.IsEmpty() {
		return nil
	}
	return &TypesConfig{
		Include: externalConfig.Include,
	}
}

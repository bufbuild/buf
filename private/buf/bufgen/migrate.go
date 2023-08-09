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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/buf/bufgen/internal"
	"github.com/bufbuild/buf/private/buf/bufgen/internal/bufgenplugin"
	"github.com/bufbuild/buf/private/buf/bufgen/internal/bufgenv1"
	"github.com/bufbuild/buf/private/buf/bufgen/internal/bufgenv2"
	"github.com/bufbuild/buf/private/buf/bufref"
	"github.com/bufbuild/buf/private/bufpkg/bufpluginexec"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/zap"
)

type migrateV1ToV2Options struct {
	input          string
	genTemplate    string
	types          []string
	includePaths   []string
	excludePaths   []string
	includeImports bool
	includeWKT     bool
}

func migrateV1ToV2(
	ctx context.Context,
	logger *zap.Logger,
	readBucket storage.ReadBucket,
	migrateOptions ...MigrateV1ToV2Option,
) error {
	options := &migrateV1ToV2Options{
		input:       ".",
		genTemplate: ExternalConfigFilePath,
	}
	for _, option := range migrateOptions {
		option(options)
	}
	configVersion, err := internal.ReadConfigVersion(
		ctx,
		logger,
		readBucket,
		internal.ReadConfigWithOverride(options.genTemplate),
	)
	if err != nil {
		return err
	}
	switch configVersion {
	case internal.V1Beta1Version, internal.V1Version:
		// OK. Also note that a file in v1beta1 is accepted by bufgenv1.ReadConfigV1.
	case internal.V2Version:
		return errors.New("configuration is already in V2")
	default:
		return fmt.Errorf("unknown version: %s", configVersion)
	}
	configV1, err := bufgenv1.ReadConfigV1(
		ctx,
		logger,
		readBucket,
		internal.ReadConfigWithOverride(options.genTemplate),
	)
	if err != nil {
		return err
	}
	externalConifgV2, err := convertConfigV1ToExternalConfigV2(
		ctx,
		logger,
		configV1,
		options.input,
		options.types,
		options.includePaths,
		options.excludePaths,
		options.includeImports,
		options.includeWKT,
	)
	if err != nil {
		return err
	}
	// Write the external config v2.
	var configV2Data []byte
	outPath := options.genTemplate
	switch filepath.Ext(options.genTemplate) {
	case ".json":
		configV2Data, err = json.Marshal(&externalConifgV2)
		if err != nil {
			return err
		}
	case ".yaml", ".yml":
		configV2Data, err = encoding.MarshalYAML(&externalConifgV2)
		if err != nil {
			return err
		}
	default:
		// check whether the format should be JSON or YAML
		placeHolderConfig := ExternalConfigVersion{}
		// Note: must try JSON first instead of YAML, because YAML is a superset of JSON
		err = json.Unmarshal([]byte(options.genTemplate), &placeHolderConfig)
		if err == nil {
			// json data
			configV2Data, err = json.Marshal(&externalConifgV2)
			if err != nil {
				return err
			}
			// TODO: maybe this file already exists
			outPath = "buf.gen.migratev1.json"
			break
		}
		// yaml data
		configV2Data, err = encoding.MarshalYAML(&externalConifgV2)
		if err != nil {
			return err
		}
		// TODO: maybe this file already exists
		outPath = "buf.gen.migratev1.yaml"
	}
	return os.WriteFile(outPath, configV2Data, 0600)
}

func convertConfigV1ToExternalConfigV2(
	ctx context.Context,
	logger *zap.Logger,
	configV1 *bufgenv1.Config,
	input string,
	types []string,
	includePaths []string,
	excludePaths []string,
	includeImports bool,
	includeWKT bool,
) (*ExternalConfigV2, error) {
	externalConfigV2 := bufgenv2.ExternalConfigV2{
		Version: "v2",
	}
	inputConfig, err := getExternalInputConfigV2(
		ctx,
		logger,
		input,
		types,
		includePaths,
		excludePaths,
	)
	if err != nil {
		return nil, err
	}
	externalConfigV2.Inputs = []bufgenv2.ExternalInputConfigV2{
		*inputConfig,
	}
	for _, pluginConfigV1 := range configV1.PluginConfigs {
		pluginConfigV2, err := pluginConfigToExternalPluginConfigV2(
			pluginConfigV1,
			includeImports,
			includeWKT,
		)
		if err != nil {
			return nil, fmt.Errorf("unable to migrate plugins: %w", err)
		}
		externalConfigV2.Plugins = append(externalConfigV2.Plugins, *pluginConfigV2)
	}
	externalConfigV2.Managed = *managedConfigV1ToExternalManagedConfigV2(configV1.ManagedConfig)
	return &externalConfigV2, nil
}

func pluginConfigToExternalPluginConfigV2(
	pluginConfig bufgenplugin.PluginConfig,
	includeImports bool,
	includeWKT bool,
) (*bufgenv2.ExternalPluginConfigV2, error) {
	externalPluginConfig := bufgenv2.ExternalPluginConfigV2{}
	externalPluginConfig.Out = pluginConfig.Out()
	externalPluginConfig.Opt = pluginConfig.Opt()
	pluginName := pluginConfig.PluginName()
	switch t := pluginConfig.(type) {
	case bufgenplugin.BinaryPluginConfig:
		externalPluginConfig.Binary = t.Path()
	case bufgenplugin.ProtocBuiltinPluginConfig:
		externalPluginConfig.ProtocBuiltin = &pluginName
		if protocPath := t.ProtocPath(); protocPath != "" {
			externalPluginConfig.ProtocPath = &protocPath
		}
	case bufgenplugin.LocalPluginConfig:
		if binaryPath, err := bufpluginexec.FindPluginPath("protoc-gen-" + pluginName); err != nil {
			// this is a binary plugin
			externalPluginConfig.Binary = binaryPath
			break
		}
		if _, isProtocBuiltin := bufpluginexec.ProtocProxyPluginNames[pluginName]; isProtocBuiltin {
			externalPluginConfig.ProtocBuiltin = &pluginName
			break
		}
		// At this point, we know for certain that this plugin is not protoc-builtin.
		// It's possible that the plugin is a valid binary plugin but not installed on
		// the user's environment.
		// TODO: we can also treat it as a binary plugin, leaving a comment like so:
		// # we couldn't find protoc-gen-xyz locally and you should verify that it is
		// # a binary installed locally
		// binary_plugin: protoc-gen-xyz
		return nil, fmt.Errorf("unable to migrate plugin %q: plugin protoc-gen-%s is not found locally and %s is not built-in to protoc", pluginName, pluginName, pluginName)
	case bufgenplugin.CuratedPluginConfig:
		externalPluginConfig.Remote = &pluginName
		revision := t.Revision()
		if revision != 0 {
			externalPluginConfig.Revision = &revision
		}
	case bufgenplugin.LegacyRemotePluginConfig:
		// TODO: maybe NewConfigV1 can return error when it sees a legacy remote plugin,
		// and type LegacyRemotePluginConfig can be removed.
		return nil, fmt.Errorf("%s is a deprecated alpha remote plugin and is no longer supported", t.PluginName())
	default:
		// this should not happen
		return nil, fmt.Errorf("unknown plugin type: %T", t)
	}
	if includeImports {
		externalPluginConfig.IncludeImports = true
		if includeWKT {
			externalPluginConfig.IncludeWKT = true
		}
	}
	return &externalPluginConfig, nil
}

func managedConfigV1ToExternalManagedConfigV2(managedConfigV1 *bufgenv1.ManagedConfig) *bufgenv2.ExternalManagedConfigV2 {
	managedConfigV2 := bufgenv2.ExternalManagedConfigV2{}
	if managedConfigV1 == nil {
		managedConfigV2.Enabled = false
		return &managedConfigV2
	}
	managedConfigV2.Enabled = true
	if ccEnableArenas := managedConfigV1.CcEnableArenas; ccEnableArenas != nil {
		defaulOverrideRule := bufgenv2.ExternalManagedOverrideConfigV2{
			FileOption: bufgenv2.FileOptionCcEnableArenas.String(),
			Value:      *ccEnableArenas,
		}
		managedConfigV2.Override = append(managedConfigV2.Override, defaulOverrideRule)
	}
	if javaMultipleFiles := managedConfigV1.JavaMultipleFiles; javaMultipleFiles != nil {
		defaultOverrideRule := bufgenv2.ExternalManagedOverrideConfigV2{
			FileOption: bufgenv2.FileOptionJavaMultipleFiles.String(),
			Value:      *javaMultipleFiles,
		}
		managedConfigV2.Override = append(managedConfigV2.Override, defaultOverrideRule)
	}
	if javaStringCheckUtf8 := managedConfigV1.JavaStringCheckUtf8; javaStringCheckUtf8 != nil {
		defaultOverrideRule := bufgenv2.ExternalManagedOverrideConfigV2{
			FileOption: bufgenv2.FileOptionJavaStringCheckUtf8.String(),
			Value:      *javaStringCheckUtf8,
		}
		managedConfigV2.Override = append(managedConfigV2.Override, defaultOverrideRule)
	}
	if javaPackagePrefixConfig := managedConfigV1.JavaPackagePrefixConfig; javaPackagePrefixConfig != nil {
		defaultOverrideRule := bufgenv2.ExternalManagedOverrideConfigV2{
			FileOption: bufgenv2.FileOptionJavaPackagePrefix.String(),
			Value:      javaPackagePrefixConfig.Default,
		}
		managedConfigV2.Override = append(managedConfigV2.Override, defaultOverrideRule)
		for _, excludedModule := range javaPackagePrefixConfig.Except {
			moduleDisableRule := bufgenv2.ExternalManagedDisableConfigV2{
				// java_package disables modifying this option completely, which is the intended behavior
				FileOption: bufgenv2.FileOptionJavaPackage.String(),
				Module:     excludedModule.IdentityString(),
			}
			managedConfigV2.Disable = append(managedConfigV2.Disable, moduleDisableRule)
		}
		for module, overridePrefix := range javaPackagePrefixConfig.Override {
			moduleOverrideRule := bufgenv2.ExternalManagedOverrideConfigV2{
				FileOption: bufgenv2.FileOptionJavaPackagePrefix.String(),
				Module:     module.IdentityString(),
				Value:      overridePrefix,
			}
			managedConfigV2.Override = append(managedConfigV2.Override, moduleOverrideRule)
		}
	}
	if csharpNamespaceConfig := managedConfigV1.CsharpNameSpaceConfig; csharpNamespaceConfig != nil {
		for _, excludedModule := range csharpNamespaceConfig.Except {
			moduleDisableRule := bufgenv2.ExternalManagedDisableConfigV2{
				FileOption: bufgenv2.FileOptionCsharpNamespace.String(),
				Module:     excludedModule.IdentityString(),
			}
			managedConfigV2.Disable = append(managedConfigV2.Disable, moduleDisableRule)
		}
		for module, namespaceOverride := range csharpNamespaceConfig.Override {
			ModuleOverrideRule := bufgenv2.ExternalManagedOverrideConfigV2{
				FileOption: bufgenv2.FileOptionCsharpNamespace.String(),
				Module:     module.IdentityString(),
				Value:      namespaceOverride,
			}
			managedConfigV2.Override = append(managedConfigV2.Override, ModuleOverrideRule)
		}
	}
	if optimizeForConfig := managedConfigV1.OptimizeForConfig; optimizeForConfig != nil {
		defaultOverrideRule := bufgenv2.ExternalManagedOverrideConfigV2{
			FileOption: bufgenv2.FileOptionOptimizeFor.String(),
			Value:      optimizeForConfig.Default.String(), // TODO: verify String() works here (probably won't)
		}
		managedConfigV2.Override = append(managedConfigV2.Override, defaultOverrideRule)
		for _, excludedModule := range optimizeForConfig.Except {
			moduleDisableRule := bufgenv2.ExternalManagedDisableConfigV2{
				FileOption: bufgenv2.FileOptionOptimizeFor.String(),
				Module:     excludedModule.IdentityString(),
			}
			managedConfigV2.Disable = append(managedConfigV2.Disable, moduleDisableRule)
		}
		for module, optimizeForOverride := range optimizeForConfig.Override {
			ModuleOverrideRule := bufgenv2.ExternalManagedOverrideConfigV2{
				FileOption: bufgenv2.FileOptionOptimizeFor.String(),
				Module:     module.IdentityString(),
				Value:      optimizeForOverride.String(),
			}
			managedConfigV2.Override = append(managedConfigV2.Override, ModuleOverrideRule)
		}
	}
	if goPackagePrefixConfig := managedConfigV1.GoPackagePrefixConfig; goPackagePrefixConfig != nil {
		defaultOverrideRule := bufgenv2.ExternalManagedOverrideConfigV2{
			FileOption: bufgenv2.FileOptionGoPackagePrefix.String(),
			Value:      goPackagePrefixConfig.Default,
		}
		managedConfigV2.Override = append(managedConfigV2.Override, defaultOverrideRule)
		for _, excludedModule := range goPackagePrefixConfig.Except {
			moduleDisableRule := bufgenv2.ExternalManagedDisableConfigV2{
				FileOption: bufgenv2.FileOptionGoPackage.String(),
				Module:     excludedModule.IdentityString(),
			}
			managedConfigV2.Disable = append(managedConfigV2.Disable, moduleDisableRule)
		}
		for module, override := range goPackagePrefixConfig.Override {
			moduleOverrideRule := bufgenv2.ExternalManagedOverrideConfigV2{
				FileOption: bufgenv2.FileOptionGoPackagePrefix.String(),
				Module:     module.IdentityString(),
				Value:      override,
			}
			managedConfigV2.Override = append(managedConfigV2.Override, moduleOverrideRule)
		}
	}
	if objcClassPrefixConfig := managedConfigV1.ObjcClassPrefixConfig; objcClassPrefixConfig != nil {
		defaultOverrideRule := bufgenv2.ExternalManagedOverrideConfigV2{
			FileOption: bufgenv2.FileOptionObjcClassPrefix.String(),
			Value:      objcClassPrefixConfig.Default,
		}
		managedConfigV2.Override = append(managedConfigV2.Override, defaultOverrideRule)
		for _, excludedModule := range objcClassPrefixConfig.Except {
			moduleDisableRule := bufgenv2.ExternalManagedDisableConfigV2{
				FileOption: bufgenv2.FileOptionObjcClassPrefix.String(),
				Module:     excludedModule.IdentityString(),
			}
			managedConfigV2.Disable = append(managedConfigV2.Disable, moduleDisableRule)
		}
		for module, override := range objcClassPrefixConfig.Override {
			moduleOverrideRule := bufgenv2.ExternalManagedOverrideConfigV2{
				FileOption: bufgenv2.FileOptionObjcClassPrefix.String(),
				Module:     module.IdentityString(),
				Value:      override,
			}
			managedConfigV2.Override = append(managedConfigV2.Override, moduleOverrideRule)
		}
	}
	if rubyPackageConfig := managedConfigV1.RubyPackageConfig; rubyPackageConfig != nil {
		for _, excludedModule := range rubyPackageConfig.Except {
			moduleDisableRule := bufgenv2.ExternalManagedDisableConfigV2{
				FileOption: bufgenv2.FileOptionRubyPackage.String(),
				Module:     excludedModule.IdentityString(),
			}
			managedConfigV2.Disable = append(managedConfigV2.Disable, moduleDisableRule)
		}
		for module, override := range rubyPackageConfig.Override {
			moduleOverrideRule := bufgenv2.ExternalManagedOverrideConfigV2{
				FileOption: bufgenv2.FileOptionRubyPackage.String(),
				Module:     module.IdentityString(),
				Value:      override,
			}
			managedConfigV2.Override = append(managedConfigV2.Override, moduleOverrideRule)
		}
	}
	for fileOption, fileToOverride := range managedConfigV1.Override {
		for file, override := range fileToOverride {
			fileOverrideRule := bufgenv2.ExternalManagedOverrideConfigV2{
				FileOption: strings.ToLower(fileOption),
				Path:       file,
				Value:      override,
			}
			managedConfigV2.Override = append(managedConfigV2.Override, fileOverrideRule)
		}
	}
	return &managedConfigV2
}

func getExternalInputConfigV2(
	ctx context.Context,
	logger *zap.Logger,
	input string,
	types []string,
	includePaths []string,
	excludedPaths []string,
) (*bufgenv2.ExternalInputConfigV2, error) {
	inputConfig := bufgenv2.ExternalInputConfigV2{}
	path, options, err := bufref.GetRawPathAndOptions(input)
	if err != nil {
		return nil, err
	}
	format, err := buffetch.NewRefParser(logger).GetRefFormat(ctx, input)
	if err != nil {
		return nil, err
	}
	switch format {
	case buffetch.FormatBin:
		inputConfig.BinaryImage = &path
	case buffetch.FormatBingz:
		inputConfig.BinaryImage = &path
	case buffetch.FormatDir:
		inputConfig.Directory = &path
	case buffetch.FormatGit:
		inputConfig.GitRepo = &path
	case buffetch.FormatJSON:
		inputConfig.JSONImage = &path
	case buffetch.FormatJSONGZ:
		inputConfig.JSONImage = &path
	case buffetch.FormatMod:
		inputConfig.Module = &path
	case buffetch.FormatTar:
		inputConfig.Tarball = &path
	case buffetch.FormatTargz:
		inputConfig.Tarball = &path
	case buffetch.FormatZip:
		inputConfig.ZipArchive = &path
	case buffetch.FormatProtoFile:
		inputConfig.ProtoFile = &path
	// TODO: txtpb and jsonpb
	default:
		return nil, fmt.Errorf("unrecognized format: %s", format)
	}
	for key, value := range options {
		switch key {
		case "format":
			// No-op, because ref parser has already returned the correct format.
		case "compression":
			inputConfig.Compression = &value
		case "branch":
			inputConfig.Branch = &value
		case "tag":
			inputConfig.Tag = &value
		case "ref":
			inputConfig.Ref = &value
		case "depth":
			depth, err := parseStringToUint32Ptr(key, value)
			if err != nil {
				return nil, err
			}
			inputConfig.Depth = depth
		case "recurse_submodules":
			recurseSubmodules, err := parseStringToBoolPtr(key, value)
			if err != nil {
				return nil, err
			}
			inputConfig.RecurseSubmodules = recurseSubmodules
		case "strip_components":
			stripComponents, err := parseStringToUint32Ptr(key, value)
			if err != nil {
				return nil, err
			}
			inputConfig.StripComponents = stripComponents
		case "subdir":
			inputConfig.Subdir = &value
		case "include_package_files":
			includePackageFiles, err := parseStringToBoolPtr(key, value)
			if err != nil {
				return nil, err
			}
			inputConfig.IncludePackageFiles = includePackageFiles
		default:
			return nil, fmt.Errorf("%q is not a valid option", key)
		}
	}
	inputConfig.Types = types
	inputConfig.IncludePaths = includePaths
	inputConfig.ExcludePaths = excludedPaths
	return &inputConfig, nil
}

func parseStringToBoolPtr(keyName string, value string) (*bool, error) {
	var parsedValue bool
	switch value {
	case "true":
		parsedValue = true
	case "false":
		parsedValue = false
	default:
		return nil, fmt.Errorf("unable to parse %s, must provide true or false", keyName)
	}
	return &parsedValue, nil
}

func parseStringToUint32Ptr(keyName string, value string) (*uint32, error) {
	parsedValueUint64, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("unable to parse %s, must provide an unsigned 32-bit integer", keyName)
	}
	result := uint32(parsedValueUint64)
	return &result, nil
}

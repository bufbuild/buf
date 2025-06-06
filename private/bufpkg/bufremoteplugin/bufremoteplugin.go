// Copyright 2020-2025 Buf Technologies, Inc.
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

package bufremoteplugin

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginref"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"google.golang.org/protobuf/proto"
)

// Plugin represents a plugin defined by a buf.plugin.yaml.
type Plugin interface {
	// Version is the version of the plugin's implementation
	// (e.g. the protoc-gen-connect-go implementation is v0.2.0).
	Version() string
	// SourceURL is an optional attribute used to specify where the source
	// for the plugin can be found.
	SourceURL() string
	// Description is an optional attribute to provide a more detailed
	// description for the plugin.
	Description() string
	// Dependencies are the dependencies this plugin has on other plugins.
	//
	// An example of a dependency might be a 'protoc-gen-go-grpc' plugin
	// which depends on the 'protoc-gen-go' generated code.
	Dependencies() []bufremotepluginref.PluginReference
	// Registry is the registry configuration, which lets the user specify
	// registry dependencies, and other metadata that applies to a specific
	// remote generation registry (e.g. the Go module proxy, NPM registry,
	// etc).
	Registry() *bufremotepluginconfig.RegistryConfig
	// ContainerImageDigest returns the plugin's source image digest.
	//
	// For now, we only support docker image sources, but this
	// might evolve to support others later on.
	ContainerImageDigest() string
}

// NewPlugin creates a new plugin from the given configuration and image digest.
func NewPlugin(
	version string,
	dependencies []bufremotepluginref.PluginReference,
	registryConfig *bufremotepluginconfig.RegistryConfig,
	imageDigest string,
	sourceURL string,
	description string,
) (Plugin, error) {
	return newPlugin(version, dependencies, registryConfig, imageDigest, sourceURL, description)
}

// PluginToProtoPluginRegistryType determines the appropriate registryv1alpha1.PluginRegistryType for the plugin.
func PluginToProtoPluginRegistryType(plugin Plugin) registryv1alpha1.PluginRegistryType {
	registryType := registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_UNSPECIFIED
	if registry := plugin.Registry(); registry != nil {
		switch {
		case registry.Go != nil:
			registryType = registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_GO
		case registry.NPM != nil:
			registryType = registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_NPM
		case registry.Maven != nil:
			registryType = registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_MAVEN
		case registry.Swift != nil:
			registryType = registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_SWIFT
		case registry.Python != nil:
			registryType = registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_PYTHON
		case registry.Cargo != nil:
			registryType = registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_CARGO
		case registry.Nuget != nil:
			registryType = registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_NUGET
		case registry.Cmake != nil:
			registryType = registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_CMAKE
		}
	}
	return registryType
}

// OutputLanguagesToProtoLanguages determines the appropriate registryv1alpha1.PluginRegistryType for the plugin.
func OutputLanguagesToProtoLanguages(languages []string) ([]registryv1alpha1.PluginLanguage, error) {
	languageToEnum := make(map[string]registryv1alpha1.PluginLanguage)
	var supportedLanguages []string
	for pluginLanguageKey, pluginLanguage := range registryv1alpha1.PluginLanguage_value {
		if pluginLanguage == 0 {
			continue
		}
		pluginLanguageKey := strings.TrimPrefix(pluginLanguageKey, "PLUGIN_LANGUAGE_")
		pluginLanguageKey = strings.ToLower(pluginLanguageKey)
		// Example:
		// { go: 1, javascript: 2 }
		languageToEnum[pluginLanguageKey] = registryv1alpha1.PluginLanguage(pluginLanguage)
		supportedLanguages = append(supportedLanguages, pluginLanguageKey)
	}
	sort.Strings(supportedLanguages)
	var protoLanguages []registryv1alpha1.PluginLanguage
	for _, language := range languages {
		if pluginLanguage, ok := languageToEnum[language]; ok {
			protoLanguages = append(protoLanguages, pluginLanguage)
			continue
		}
		return nil, fmt.Errorf("invalid plugin output language: %q\nsupported languages: %s", language, strings.Join(supportedLanguages, ", "))
	}
	slices.Sort(protoLanguages)
	return protoLanguages, nil
}

// PluginRegistryToProtoRegistryConfig converts a bufremotepluginconfig.RegistryConfig to a registryv1alpha1.RegistryConfig.
func PluginRegistryToProtoRegistryConfig(pluginRegistry *bufremotepluginconfig.RegistryConfig) (*registryv1alpha1.RegistryConfig, error) {
	if pluginRegistry == nil {
		return nil, nil
	}
	registryConfig := registryv1alpha1.RegistryConfig_builder{
		Options: bufremotepluginconfig.PluginOptionsToOptionsSlice(pluginRegistry.Options),
	}.Build()
	switch {
	case pluginRegistry.Go != nil:
		goConfig := &registryv1alpha1.GoConfig{}
		goConfig.SetMinimumVersion(pluginRegistry.Go.MinVersion)
		if pluginRegistry.Go.BasePlugin != nil {
			goConfig.SetBasePlugin(pluginRegistry.Go.BasePlugin.IdentityString())
		}
		if pluginRegistry.Go.Deps != nil {
			goConfig.SetRuntimeLibraries(make([]*registryv1alpha1.GoConfig_RuntimeLibrary, 0, len(pluginRegistry.Go.Deps)))
			for _, dependency := range pluginRegistry.Go.Deps {
				goConfig.SetRuntimeLibraries(append(goConfig.GetRuntimeLibraries(), goRuntimeDependencyToProtoGoRuntimeLibrary(dependency)))
			}
		}
		registryConfig.SetGoConfig(proto.ValueOrDefault(goConfig))
	case pluginRegistry.NPM != nil:
		importStyle, err := npmImportStyleToNPMProtoImportStyle(pluginRegistry.NPM.ImportStyle)
		if err != nil {
			return nil, err
		}
		npmConfig := registryv1alpha1.NPMConfig_builder{
			RewriteImportPathSuffix: pluginRegistry.NPM.RewriteImportPathSuffix,
			ImportStyle:             importStyle,
		}.Build()
		if pluginRegistry.NPM.Deps != nil {
			npmConfig.SetRuntimeLibraries(make([]*registryv1alpha1.NPMConfig_RuntimeLibrary, 0, len(pluginRegistry.NPM.Deps)))
			for _, dependency := range pluginRegistry.NPM.Deps {
				npmConfig.SetRuntimeLibraries(append(npmConfig.GetRuntimeLibraries(), npmRuntimeDependencyToProtoNPMRuntimeLibrary(dependency)))
			}
		}
		registryConfig.SetNpmConfig(proto.ValueOrDefault(npmConfig))
	case pluginRegistry.Maven != nil:
		mavenConfig := &registryv1alpha1.MavenConfig{}
		var javaCompilerConfig *registryv1alpha1.MavenConfig_CompilerJavaConfig
		if compiler := pluginRegistry.Maven.Compiler.Java; compiler != (bufremotepluginconfig.MavenCompilerJavaConfig{}) {
			javaCompilerConfig = registryv1alpha1.MavenConfig_CompilerJavaConfig_builder{
				Encoding: compiler.Encoding,
				Release:  int32(compiler.Release),
				Source:   int32(compiler.Source),
				Target:   int32(compiler.Target),
			}.Build()
		}
		var kotlinCompilerConfig *registryv1alpha1.MavenConfig_CompilerKotlinConfig
		if compiler := pluginRegistry.Maven.Compiler.Kotlin; compiler != (bufremotepluginconfig.MavenCompilerKotlinConfig{}) {
			kotlinCompilerConfig = registryv1alpha1.MavenConfig_CompilerKotlinConfig_builder{
				Version:         compiler.Version,
				ApiVersion:      compiler.APIVersion,
				JvmTarget:       compiler.JVMTarget,
				LanguageVersion: compiler.LanguageVersion,
			}.Build()
		}
		if javaCompilerConfig != nil || kotlinCompilerConfig != nil {
			mavenConfig.SetCompiler(registryv1alpha1.MavenConfig_CompilerConfig_builder{
				Java:   javaCompilerConfig,
				Kotlin: kotlinCompilerConfig,
			}.Build())
		}
		if pluginRegistry.Maven.Deps != nil {
			mavenConfig.SetRuntimeLibraries(make([]*registryv1alpha1.MavenConfig_RuntimeLibrary, len(pluginRegistry.Maven.Deps)))
			for i, dependency := range pluginRegistry.Maven.Deps {
				mavenConfig.GetRuntimeLibraries()[i] = MavenDependencyConfigToProtoRuntimeLibrary(dependency)
			}
		}
		if pluginRegistry.Maven.AdditionalRuntimes != nil {
			mavenConfig.SetAdditionalRuntimes(make([]*registryv1alpha1.MavenConfig_RuntimeConfig, len(pluginRegistry.Maven.AdditionalRuntimes)))
			for i, runtime := range pluginRegistry.Maven.AdditionalRuntimes {
				mavenConfig.GetAdditionalRuntimes()[i] = MavenRuntimeConfigToProtoRuntimeConfig(runtime)
			}
		}
		registryConfig.SetMavenConfig(proto.ValueOrDefault(mavenConfig))
	case pluginRegistry.Swift != nil:
		swiftConfig := SwiftRegistryConfigToProtoSwiftConfig(pluginRegistry.Swift)
		registryConfig.SetSwiftConfig(proto.ValueOrDefault(swiftConfig))
	case pluginRegistry.Python != nil:
		pythonConfig, err := PythonRegistryConfigToProtoPythonConfig(pluginRegistry.Python)
		if err != nil {
			return nil, err
		}
		registryConfig.SetPythonConfig(proto.ValueOrDefault(pythonConfig))
	case pluginRegistry.Cargo != nil:
		cargoConfig, err := CargoRegistryConfigToProtoCargoConfig(pluginRegistry.Cargo)
		if err != nil {
			return nil, err
		}
		registryConfig.SetCargoConfig(proto.ValueOrDefault(cargoConfig))
	case pluginRegistry.Nuget != nil:
		nugetConfig, err := NugetRegistryConfigToProtoNugetConfig(pluginRegistry.Nuget)
		if err != nil {
			return nil, err
		}
		registryConfig.SetNugetConfig(proto.ValueOrDefault(nugetConfig))
	case pluginRegistry.Cmake != nil:
		cmakeConfig, err := CmakeRegistryConfigToProtoCmakeConfig(pluginRegistry.Cmake)
		if err != nil {
			return nil, err
		}
		registryConfig.SetCmakeConfig(proto.ValueOrDefault(cmakeConfig))
	}
	return registryConfig, nil
}

// MavenDependencyConfigToProtoRuntimeLibrary converts a bufremotepluginconfig.MavenDependencyConfig to an equivalent registryv1alpha1.MavenConfig_RuntimeLibrary.
func MavenDependencyConfigToProtoRuntimeLibrary(dependency bufremotepluginconfig.MavenDependencyConfig) *registryv1alpha1.MavenConfig_RuntimeLibrary {
	return registryv1alpha1.MavenConfig_RuntimeLibrary_builder{
		GroupId:    dependency.GroupID,
		ArtifactId: dependency.ArtifactID,
		Version:    dependency.Version,
		Classifier: dependency.Classifier,
		Extension:  dependency.Extension,
	}.Build()
}

// ProtoRegistryConfigToPluginRegistry converts a registryv1alpha1.RegistryConfig to a bufremotepluginconfig.RegistryConfig .
func ProtoRegistryConfigToPluginRegistry(config *registryv1alpha1.RegistryConfig) (*bufremotepluginconfig.RegistryConfig, error) {
	if config == nil {
		return nil, nil
	}
	registryConfig := &bufremotepluginconfig.RegistryConfig{
		Options: bufremotepluginconfig.OptionsSliceToPluginOptions(config.GetOptions()),
	}
	switch {
	case config.GetGoConfig() != nil:
		goConfig := &bufremotepluginconfig.GoRegistryConfig{}
		goConfig.MinVersion = config.GetGoConfig().GetMinimumVersion()
		if config.GetGoConfig().GetBasePlugin() != "" {
			basePluginIdentity, err := bufremotepluginref.PluginIdentityForString(config.GetGoConfig().GetBasePlugin())
			if err != nil {
				return nil, err
			}
			goConfig.BasePlugin = basePluginIdentity
		}
		runtimeLibraries := config.GetGoConfig().GetRuntimeLibraries()
		if runtimeLibraries != nil {
			goConfig.Deps = make([]*bufremotepluginconfig.GoRegistryDependencyConfig, 0, len(runtimeLibraries))
			for _, library := range runtimeLibraries {
				goConfig.Deps = append(goConfig.Deps, protoGoRuntimeLibraryToGoRuntimeDependency(library))
			}
		}
		registryConfig.Go = goConfig
	case config.GetNpmConfig() != nil:
		importStyle, err := npmProtoImportStyleToNPMImportStyle(config.GetNpmConfig().GetImportStyle())
		if err != nil {
			return nil, err
		}
		npmConfig := &bufremotepluginconfig.NPMRegistryConfig{
			RewriteImportPathSuffix: config.GetNpmConfig().GetRewriteImportPathSuffix(),
			ImportStyle:             importStyle,
		}
		runtimeLibraries := config.GetNpmConfig().GetRuntimeLibraries()
		if runtimeLibraries != nil {
			npmConfig.Deps = make([]*bufremotepluginconfig.NPMRegistryDependencyConfig, 0, len(runtimeLibraries))
			for _, library := range runtimeLibraries {
				npmConfig.Deps = append(npmConfig.Deps, protoNPMRuntimeLibraryToNPMRuntimeDependency(library))
			}
		}
		registryConfig.NPM = npmConfig
	case config.GetMavenConfig() != nil:
		mavenConfig, err := ProtoMavenConfigToMavenRegistryConfig(config.GetMavenConfig())
		if err != nil {
			return nil, err
		}
		registryConfig.Maven = mavenConfig
	case config.GetSwiftConfig() != nil:
		swiftConfig, err := ProtoSwiftConfigToSwiftRegistryConfig(config.GetSwiftConfig())
		if err != nil {
			return nil, err
		}
		registryConfig.Swift = swiftConfig
	case config.GetPythonConfig() != nil:
		pythonConfig, err := ProtoPythonConfigToPythonRegistryConfig(config.GetPythonConfig())
		if err != nil {
			return nil, err
		}
		registryConfig.Python = pythonConfig
	case config.GetCargoConfig() != nil:
		cargoConfig, err := ProtoCargoConfigToCargoRegistryConfig(config.GetCargoConfig())
		if err != nil {
			return nil, err
		}
		registryConfig.Cargo = cargoConfig
	case config.GetNugetConfig() != nil:
		nugetConfig, err := ProtoNugetConfigToNugetRegistryConfig(config.GetNugetConfig())
		if err != nil {
			return nil, err
		}
		registryConfig.Nuget = nugetConfig
	case config.GetCmakeConfig() != nil:
		cmakeConfig, err := ProtoCmakeConfigToCmakeRegistryConfig(config.GetCmakeConfig())
		if err != nil {
			return nil, err
		}
		registryConfig.Cmake = cmakeConfig
	}
	return registryConfig, nil
}

// ProtoCargoConfigToCargoRegistryConfig converts protoCargoConfig to an equivalent [*bufremotepluginconfig.CargoRegistryConfig].
func ProtoCargoConfigToCargoRegistryConfig(protoCargoConfig *registryv1alpha1.CargoConfig) (*bufremotepluginconfig.CargoRegistryConfig, error) {
	cargoConfig := &bufremotepluginconfig.CargoRegistryConfig{
		RustVersion: protoCargoConfig.GetRustVersion(),
	}
	for _, dependency := range protoCargoConfig.GetRuntimeLibraries() {
		cargoConfig.Deps = append(cargoConfig.Deps, bufremotepluginconfig.CargoRegistryDependency{
			Name:               dependency.GetName(),
			VersionRequirement: dependency.GetVersionRequirement(),
			DefaultFeatures:    dependency.GetDefaultFeatures(),
			Features:           dependency.GetFeatures(),
		})
	}
	return cargoConfig, nil
}

// ProtoNugetConfigToNugetRegistryConfig converts protoConfig to an equivalent [*bufremotepluginconfig.NugetRegistryConfig].
func ProtoNugetConfigToNugetRegistryConfig(protoConfig *registryv1alpha1.NugetConfig) (*bufremotepluginconfig.NugetRegistryConfig, error) {
	targetFrameworks, err := xslices.MapError(protoConfig.GetTargetFrameworks(), DotnetTargetFrameworkToString)
	if err != nil {
		return nil, err
	}
	config := &bufremotepluginconfig.NugetRegistryConfig{
		TargetFrameworks: targetFrameworks,
	}
	for _, dependency := range protoConfig.GetRuntimeLibraries() {
		var depTargetFrameworks []string
		if len(dependency.GetTargetFrameworks()) > 0 {
			depTargetFrameworks, err = xslices.MapError(dependency.GetTargetFrameworks(), DotnetTargetFrameworkToString)
			if err != nil {
				return nil, err
			}
		}
		config.Deps = append(config.Deps, bufremotepluginconfig.NugetDependencyConfig{
			Name:             dependency.GetName(),
			Version:          dependency.GetVersion(),
			TargetFrameworks: depTargetFrameworks,
		})
	}
	return config, err
}

// ProtoCmakeConfigToCmakeRegistryConfig converts protoCmakeConfig to an equivalent [*bufremotepluginconfig.CmakeRegistryConfig].
func ProtoCmakeConfigToCmakeRegistryConfig(protoCmakeConfig *registryv1alpha1.CmakeConfig) (*bufremotepluginconfig.CmakeRegistryConfig, error) {
	return &bufremotepluginconfig.CmakeRegistryConfig{}, nil
}

// CargoRegistryConfigToProtoCargoConfig converts cargoConfig to an equivalent [*registryv1alpha1.CargoConfig].
func CargoRegistryConfigToProtoCargoConfig(cargoConfig *bufremotepluginconfig.CargoRegistryConfig) (*registryv1alpha1.CargoConfig, error) {
	protoCargoConfig := registryv1alpha1.CargoConfig_builder{
		RustVersion: cargoConfig.RustVersion,
	}.Build()
	for _, dependency := range cargoConfig.Deps {
		protoCargoConfig.SetRuntimeLibraries(append(protoCargoConfig.GetRuntimeLibraries(), registryv1alpha1.CargoConfig_RuntimeLibrary_builder{
			Name:               dependency.Name,
			VersionRequirement: dependency.VersionRequirement,
			DefaultFeatures:    dependency.DefaultFeatures,
			Features:           dependency.Features,
		}.Build()))
	}
	return protoCargoConfig, nil
}

// NugetRegistryConfigToProtoNugetConfig converts nugetConfig to an equivalent [*registryv1alpha1.NugetConfig].
func NugetRegistryConfigToProtoNugetConfig(nugetConfig *bufremotepluginconfig.NugetRegistryConfig) (*registryv1alpha1.NugetConfig, error) {
	targetFrameworks, err := xslices.MapError(nugetConfig.TargetFrameworks, DotnetTargetFrameworkFromString)
	if err != nil {
		return nil, err
	}
	protoNugetConfig := registryv1alpha1.NugetConfig_builder{
		TargetFrameworks: targetFrameworks,
	}.Build()
	for _, dependency := range nugetConfig.Deps {
		var depTargetFrameworks []registryv1alpha1.DotnetTargetFramework
		if len(dependency.TargetFrameworks) > 0 {
			depTargetFrameworks, err = xslices.MapError(dependency.TargetFrameworks, DotnetTargetFrameworkFromString)
			if err != nil {
				return nil, err
			}
		}
		protoNugetConfig.SetRuntimeLibraries(append(protoNugetConfig.GetRuntimeLibraries(), registryv1alpha1.NugetConfig_RuntimeLibrary_builder{
			Name:             dependency.Name,
			Version:          dependency.Version,
			TargetFrameworks: depTargetFrameworks,
		}.Build()))
	}
	return protoNugetConfig, nil
}

// CmakeRegistryConfigToProtoCmakeConfig converts cmakeConfig to an equivalent [*registryv1alpha1.CmakeConfig].
func CmakeRegistryConfigToProtoCmakeConfig(cmakeConfig *bufremotepluginconfig.CmakeRegistryConfig) (*registryv1alpha1.CmakeConfig, error) {
	return &registryv1alpha1.CmakeConfig{}, nil
}

// ProtoPythonConfigToPythonRegistryConfig converts protoPythonConfig to an equivalent [*bufremotepluginconfig.PythonRegistryConfig].
func ProtoPythonConfigToPythonRegistryConfig(protoPythonConfig *registryv1alpha1.PythonConfig) (*bufremotepluginconfig.PythonRegistryConfig, error) {
	pythonConfig := &bufremotepluginconfig.PythonRegistryConfig{
		RequiresPython: protoPythonConfig.GetRequiresPython(),
	}
	switch protoPythonConfig.GetPackageType() {
	case registryv1alpha1.PythonPackageType_PYTHON_PACKAGE_TYPE_RUNTIME:
		pythonConfig.PackageType = "runtime"
	case registryv1alpha1.PythonPackageType_PYTHON_PACKAGE_TYPE_STUB_ONLY:
		pythonConfig.PackageType = "stub-only"
	default:
		return nil, fmt.Errorf("unknown package type: %v", protoPythonConfig.GetPackageType())
	}
	for _, runtimeLibrary := range protoPythonConfig.GetRuntimeLibraries() {
		pythonConfig.Deps = append(pythonConfig.Deps, runtimeLibrary.GetDependencySpecification())
	}
	return pythonConfig, nil
}

// PythonRegistryConfigToProtoPythonConfig converts pythonConfig to an equivalent [*registryv1alpha1.PythonConfig].
func PythonRegistryConfigToProtoPythonConfig(pythonConfig *bufremotepluginconfig.PythonRegistryConfig) (*registryv1alpha1.PythonConfig, error) {
	protoPythonConfig := registryv1alpha1.PythonConfig_builder{
		RequiresPython: pythonConfig.RequiresPython,
	}.Build()
	switch pythonConfig.PackageType {
	case "runtime":
		protoPythonConfig.SetPackageType(registryv1alpha1.PythonPackageType_PYTHON_PACKAGE_TYPE_RUNTIME)
	case "stub-only":
		protoPythonConfig.SetPackageType(registryv1alpha1.PythonPackageType_PYTHON_PACKAGE_TYPE_STUB_ONLY)
	default:
		return nil, fmt.Errorf(`invalid python config package_type; expecting one of "runtime" or "stub-only", got %q`, pythonConfig.PackageType)
	}
	for _, dependencySpecification := range pythonConfig.Deps {
		protoPythonConfig.SetRuntimeLibraries(append(protoPythonConfig.GetRuntimeLibraries(), registryv1alpha1.PythonConfig_RuntimeLibrary_builder{
			DependencySpecification: dependencySpecification,
		}.Build()))
	}
	return protoPythonConfig, nil
}

// ProtoSwiftConfigToSwiftRegistryConfig converts protoSwiftConfig to an equivalent [*bufremotepluginconfig.SwiftRegistryConfig].
func ProtoSwiftConfigToSwiftRegistryConfig(protoSwiftConfig *registryv1alpha1.SwiftConfig) (*bufremotepluginconfig.SwiftRegistryConfig, error) {
	swiftConfig := &bufremotepluginconfig.SwiftRegistryConfig{}
	runtimeLibs := protoSwiftConfig.GetRuntimeLibraries()
	if runtimeLibs != nil {
		swiftConfig.Dependencies = make([]bufremotepluginconfig.SwiftRegistryDependencyConfig, 0, len(runtimeLibs))
		for _, runtimeLib := range runtimeLibs {
			dependencyConfig := bufremotepluginconfig.SwiftRegistryDependencyConfig{
				Source:        runtimeLib.GetSource(),
				Package:       runtimeLib.GetPackage(),
				Version:       runtimeLib.GetVersion(),
				Products:      runtimeLib.GetProducts(),
				SwiftVersions: runtimeLib.GetSwiftVersions(),
			}
			platforms := runtimeLib.GetPlatforms()
			for _, platform := range platforms {
				switch platform.GetName() {
				case registryv1alpha1.SwiftPlatformType_SWIFT_PLATFORM_TYPE_MACOS:
					dependencyConfig.Platforms.MacOS = platform.GetVersion()
				case registryv1alpha1.SwiftPlatformType_SWIFT_PLATFORM_TYPE_IOS:
					dependencyConfig.Platforms.IOS = platform.GetVersion()
				case registryv1alpha1.SwiftPlatformType_SWIFT_PLATFORM_TYPE_TVOS:
					dependencyConfig.Platforms.TVOS = platform.GetVersion()
				case registryv1alpha1.SwiftPlatformType_SWIFT_PLATFORM_TYPE_WATCHOS:
					dependencyConfig.Platforms.WatchOS = platform.GetVersion()
				default:
					return nil, fmt.Errorf("unknown platform type: %v", platform.GetName())
				}
			}
			swiftConfig.Dependencies = append(swiftConfig.Dependencies, dependencyConfig)
		}
	}
	return swiftConfig, nil
}

// SwiftRegistryConfigToProtoSwiftConfig converts swiftConfig to an equivalent [*registryv1alpha1.SwiftConfig].
func SwiftRegistryConfigToProtoSwiftConfig(swiftConfig *bufremotepluginconfig.SwiftRegistryConfig) *registryv1alpha1.SwiftConfig {
	protoSwiftConfig := &registryv1alpha1.SwiftConfig{}
	if swiftConfig.Dependencies != nil {
		protoSwiftConfig.SetRuntimeLibraries(make([]*registryv1alpha1.SwiftConfig_RuntimeLibrary, 0, len(swiftConfig.Dependencies)))
		for _, dependency := range swiftConfig.Dependencies {
			depConfig := registryv1alpha1.SwiftConfig_RuntimeLibrary_builder{
				Source:        dependency.Source,
				Package:       dependency.Package,
				Version:       dependency.Version,
				Products:      dependency.Products,
				SwiftVersions: dependency.SwiftVersions,
			}.Build()
			if dependency.Platforms.MacOS != "" {
				depConfig.SetPlatforms(append(depConfig.GetPlatforms(), registryv1alpha1.SwiftConfig_RuntimeLibrary_Platform_builder{
					Name:    registryv1alpha1.SwiftPlatformType_SWIFT_PLATFORM_TYPE_MACOS,
					Version: dependency.Platforms.MacOS,
				}.Build()))
			}
			if dependency.Platforms.IOS != "" {
				depConfig.SetPlatforms(append(depConfig.GetPlatforms(), registryv1alpha1.SwiftConfig_RuntimeLibrary_Platform_builder{
					Name:    registryv1alpha1.SwiftPlatformType_SWIFT_PLATFORM_TYPE_IOS,
					Version: dependency.Platforms.IOS,
				}.Build()))
			}
			if dependency.Platforms.TVOS != "" {
				depConfig.SetPlatforms(append(depConfig.GetPlatforms(), registryv1alpha1.SwiftConfig_RuntimeLibrary_Platform_builder{
					Name:    registryv1alpha1.SwiftPlatformType_SWIFT_PLATFORM_TYPE_TVOS,
					Version: dependency.Platforms.TVOS,
				}.Build()))
			}
			if dependency.Platforms.WatchOS != "" {
				depConfig.SetPlatforms(append(depConfig.GetPlatforms(), registryv1alpha1.SwiftConfig_RuntimeLibrary_Platform_builder{
					Name:    registryv1alpha1.SwiftPlatformType_SWIFT_PLATFORM_TYPE_WATCHOS,
					Version: dependency.Platforms.WatchOS,
				}.Build()))
			}
			protoSwiftConfig.SetRuntimeLibraries(append(protoSwiftConfig.GetRuntimeLibraries(), depConfig))
		}
	}
	return protoSwiftConfig
}

// ProtoMavenConfigToMavenRegistryConfig converts a registryv1alpha1.MavenConfig to a bufremotepluginconfig.MavenRegistryConfig.
func ProtoMavenConfigToMavenRegistryConfig(protoMavenConfig *registryv1alpha1.MavenConfig) (*bufremotepluginconfig.MavenRegistryConfig, error) {
	mavenConfig := &bufremotepluginconfig.MavenRegistryConfig{}
	if protoCompiler := protoMavenConfig.GetCompiler(); protoCompiler != nil {
		mavenConfig.Compiler = bufremotepluginconfig.MavenCompilerConfig{}
		if protoJavaCompiler := protoCompiler.GetJava(); protoJavaCompiler != nil {
			mavenConfig.Compiler.Java = bufremotepluginconfig.MavenCompilerJavaConfig{
				Encoding: protoJavaCompiler.GetEncoding(),
				Release:  int(protoJavaCompiler.GetRelease()),
				Source:   int(protoJavaCompiler.GetSource()),
				Target:   int(protoJavaCompiler.GetTarget()),
			}
		}
		if protoKotlinCompiler := protoCompiler.GetKotlin(); protoKotlinCompiler != nil {
			mavenConfig.Compiler.Kotlin = bufremotepluginconfig.MavenCompilerKotlinConfig{
				APIVersion:      protoKotlinCompiler.GetApiVersion(),
				JVMTarget:       protoKotlinCompiler.GetJvmTarget(),
				LanguageVersion: protoKotlinCompiler.GetLanguageVersion(),
				Version:         protoKotlinCompiler.GetVersion(),
			}
		}
	}
	runtimeLibraries := protoMavenConfig.GetRuntimeLibraries()
	if runtimeLibraries != nil {
		mavenConfig.Deps = make([]bufremotepluginconfig.MavenDependencyConfig, len(runtimeLibraries))
		for i, library := range runtimeLibraries {
			mavenConfig.Deps[i] = ProtoMavenRuntimeLibraryToDependencyConfig(library)
		}
	}
	additionalRuntimes := protoMavenConfig.GetAdditionalRuntimes()
	if additionalRuntimes != nil {
		mavenConfig.AdditionalRuntimes = make([]bufremotepluginconfig.MavenRuntimeConfig, len(additionalRuntimes))
		for i, additionalRuntime := range additionalRuntimes {
			runtime, err := MavenProtoRuntimeConfigToRuntimeConfig(additionalRuntime)
			if err != nil {
				return nil, err
			}
			mavenConfig.AdditionalRuntimes[i] = runtime
		}
	}
	return mavenConfig, nil
}

// MavenProtoRuntimeConfigToRuntimeConfig converts a registryv1alpha1.MavenConfig_RuntimeConfig to a bufremotepluginconfig.MavenRuntimeConfig.
func MavenProtoRuntimeConfigToRuntimeConfig(proto *registryv1alpha1.MavenConfig_RuntimeConfig) (bufremotepluginconfig.MavenRuntimeConfig, error) {
	libraries := proto.GetRuntimeLibraries()
	var dependencies []bufremotepluginconfig.MavenDependencyConfig
	for _, library := range libraries {
		dependencies = append(dependencies, ProtoMavenRuntimeLibraryToDependencyConfig(library))
	}
	return bufremotepluginconfig.MavenRuntimeConfig{
		Name:    proto.GetName(),
		Deps:    dependencies,
		Options: proto.GetOptions(),
	}, nil
}

// MavenRuntimeConfigToProtoRuntimeConfig converts a bufremotepluginconfig.MavenRuntimeConfig to a registryv1alpha1.MavenConfig_RuntimeLibrary.
func MavenRuntimeConfigToProtoRuntimeConfig(runtime bufremotepluginconfig.MavenRuntimeConfig) *registryv1alpha1.MavenConfig_RuntimeConfig {
	var libraries []*registryv1alpha1.MavenConfig_RuntimeLibrary
	for _, dependency := range runtime.Deps {
		libraries = append(libraries, MavenDependencyConfigToProtoRuntimeLibrary(dependency))
	}
	return registryv1alpha1.MavenConfig_RuntimeConfig_builder{
		Name:             runtime.Name,
		RuntimeLibraries: libraries,
		Options:          runtime.Options,
	}.Build()
}

// ProtoMavenRuntimeLibraryToDependencyConfig converts a registryv1alpha1 to a bufremotepluginconfig.MavenDependencyConfig.
func ProtoMavenRuntimeLibraryToDependencyConfig(proto *registryv1alpha1.MavenConfig_RuntimeLibrary) bufremotepluginconfig.MavenDependencyConfig {
	return bufremotepluginconfig.MavenDependencyConfig{
		GroupID:    proto.GetGroupId(),
		ArtifactID: proto.GetArtifactId(),
		Version:    proto.GetVersion(),
		Classifier: proto.GetClassifier(),
		Extension:  proto.GetExtension(),
	}
}

func npmImportStyleToNPMProtoImportStyle(importStyle string) (registryv1alpha1.NPMImportStyle, error) {
	switch importStyle {
	case "commonjs":
		return registryv1alpha1.NPMImportStyle_NPM_IMPORT_STYLE_COMMONJS, nil
	case "module":
		return registryv1alpha1.NPMImportStyle_NPM_IMPORT_STYLE_MODULE, nil
	}
	return 0, fmt.Errorf(`invalid import style %q: must be one of "module" or "commonjs"`, importStyle)
}

func npmProtoImportStyleToNPMImportStyle(importStyle registryv1alpha1.NPMImportStyle) (string, error) {
	switch importStyle {
	case registryv1alpha1.NPMImportStyle_NPM_IMPORT_STYLE_COMMONJS:
		return "commonjs", nil
	case registryv1alpha1.NPMImportStyle_NPM_IMPORT_STYLE_MODULE:
		return "module", nil
	}
	return "", fmt.Errorf("unknown import style: %v", importStyle)
}

// goRuntimeDependencyToProtoGoRuntimeLibrary converts a bufremotepluginconfig.GoRegistryDependencyConfig to a registryv1alpha1.GoConfig_RuntimeLibrary.
func goRuntimeDependencyToProtoGoRuntimeLibrary(config *bufremotepluginconfig.GoRegistryDependencyConfig) *registryv1alpha1.GoConfig_RuntimeLibrary {
	return registryv1alpha1.GoConfig_RuntimeLibrary_builder{
		Module:  config.Module,
		Version: config.Version,
	}.Build()
}

// protoGoRuntimeLibraryToGoRuntimeDependency converts a registryv1alpha1.GoConfig_RuntimeLibrary to a bufremotepluginconfig.GoRegistryDependencyConfig.
func protoGoRuntimeLibraryToGoRuntimeDependency(config *registryv1alpha1.GoConfig_RuntimeLibrary) *bufremotepluginconfig.GoRegistryDependencyConfig {
	return &bufremotepluginconfig.GoRegistryDependencyConfig{
		Module:  config.GetModule(),
		Version: config.GetVersion(),
	}
}

// npmRuntimeDependencyToProtoNPMRuntimeLibrary converts a bufremotepluginconfig.NPMRegistryConfig to a registryv1alpha1.NPMConfig_RuntimeLibrary.
func npmRuntimeDependencyToProtoNPMRuntimeLibrary(config *bufremotepluginconfig.NPMRegistryDependencyConfig) *registryv1alpha1.NPMConfig_RuntimeLibrary {
	return registryv1alpha1.NPMConfig_RuntimeLibrary_builder{
		Package: config.Package,
		Version: config.Version,
	}.Build()
}

// protoNPMRuntimeLibraryToNPMRuntimeDependency converts a registryv1alpha1.NPMConfig_RuntimeLibrary to a bufremotepluginconfig.NPMRegistryDependencyConfig.
func protoNPMRuntimeLibraryToNPMRuntimeDependency(config *registryv1alpha1.NPMConfig_RuntimeLibrary) *bufremotepluginconfig.NPMRegistryDependencyConfig {
	return &bufremotepluginconfig.NPMRegistryDependencyConfig{
		Package: config.GetPackage(),
		Version: config.GetVersion(),
	}
}

// PluginReferencesToCuratedProtoPluginReferences converts a slice of bufremotepluginref.PluginReference to a slice of registryv1alpha1.CuratedPluginReference.
func PluginReferencesToCuratedProtoPluginReferences(references []bufremotepluginref.PluginReference) []*registryv1alpha1.CuratedPluginReference {
	if references == nil {
		return nil
	}
	protoReferences := make([]*registryv1alpha1.CuratedPluginReference, 0, len(references))
	for _, reference := range references {
		protoReferences = append(protoReferences, PluginReferenceToProtoCuratedPluginReference(reference))
	}
	return protoReferences
}

// PluginReferenceToProtoCuratedPluginReference converts a bufremotepluginref.PluginReference to a registryv1alpha1.CuratedPluginReference.
func PluginReferenceToProtoCuratedPluginReference(reference bufremotepluginref.PluginReference) *registryv1alpha1.CuratedPluginReference {
	if reference == nil {
		return nil
	}
	return registryv1alpha1.CuratedPluginReference_builder{
		Owner:    reference.Owner(),
		Name:     reference.Plugin(),
		Version:  reference.Version(),
		Revision: uint32(reference.Revision()),
	}.Build()
}

// PluginIdentityToProtoCuratedPluginReference converts a bufremotepluginref.PluginIdentity to a registryv1alpha1.CuratedPluginReference.
//
// The returned CuratedPluginReference contains no Version/Revision information.
func PluginIdentityToProtoCuratedPluginReference(identity bufremotepluginref.PluginIdentity) *registryv1alpha1.CuratedPluginReference {
	if identity == nil {
		return nil
	}
	return registryv1alpha1.CuratedPluginReference_builder{
		Owner: identity.Owner(),
		Name:  identity.Plugin(),
	}.Build()
}

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

package bufremoteplugin

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginref"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
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
	if plugin.Registry() != nil {
		if plugin.Registry().Go != nil {
			registryType = registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_GO
		} else if plugin.Registry().NPM != nil {
			registryType = registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_NPM
		} else if plugin.Registry().Maven != nil {
			registryType = registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_MAVEN
		} else if plugin.Registry().Swift != nil {
			registryType = registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_SWIFT
		} else if plugin.Registry().Python != nil {
			registryType = registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_PYTHON
		} else if plugin.Registry().Cargo != nil {
			registryType = registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_CARGO
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
	sort.Slice(protoLanguages, func(i, j int) bool {
		return protoLanguages[i] < protoLanguages[j]
	})
	return protoLanguages, nil
}

// PluginRegistryToProtoRegistryConfig converts a bufremotepluginconfig.RegistryConfig to a registryv1alpha1.RegistryConfig.
func PluginRegistryToProtoRegistryConfig(pluginRegistry *bufremotepluginconfig.RegistryConfig) (*registryv1alpha1.RegistryConfig, error) {
	if pluginRegistry == nil {
		return nil, nil
	}
	registryConfig := &registryv1alpha1.RegistryConfig{
		Options: bufremotepluginconfig.PluginOptionsToOptionsSlice(pluginRegistry.Options),
	}
	if pluginRegistry.Go != nil {
		goConfig := &registryv1alpha1.GoConfig{}
		goConfig.MinimumVersion = pluginRegistry.Go.MinVersion
		if pluginRegistry.Go.Deps != nil {
			goConfig.RuntimeLibraries = make([]*registryv1alpha1.GoConfig_RuntimeLibrary, 0, len(pluginRegistry.Go.Deps))
			for _, dependency := range pluginRegistry.Go.Deps {
				goConfig.RuntimeLibraries = append(goConfig.RuntimeLibraries, goRuntimeDependencyToProtoGoRuntimeLibrary(dependency))
			}
		}
		registryConfig.RegistryConfig = &registryv1alpha1.RegistryConfig_GoConfig{GoConfig: goConfig}
	} else if pluginRegistry.NPM != nil {
		importStyle, err := npmImportStyleToNPMProtoImportStyle(pluginRegistry.NPM.ImportStyle)
		if err != nil {
			return nil, err
		}
		npmConfig := &registryv1alpha1.NPMConfig{
			RewriteImportPathSuffix: pluginRegistry.NPM.RewriteImportPathSuffix,
			ImportStyle:             importStyle,
		}
		if pluginRegistry.NPM.Deps != nil {
			npmConfig.RuntimeLibraries = make([]*registryv1alpha1.NPMConfig_RuntimeLibrary, 0, len(pluginRegistry.NPM.Deps))
			for _, dependency := range pluginRegistry.NPM.Deps {
				npmConfig.RuntimeLibraries = append(npmConfig.RuntimeLibraries, npmRuntimeDependencyToProtoNPMRuntimeLibrary(dependency))
			}
		}
		registryConfig.RegistryConfig = &registryv1alpha1.RegistryConfig_NpmConfig{NpmConfig: npmConfig}
	} else if pluginRegistry.Maven != nil {
		mavenConfig := &registryv1alpha1.MavenConfig{}
		var javaCompilerConfig *registryv1alpha1.MavenConfig_CompilerJavaConfig
		if compiler := pluginRegistry.Maven.Compiler.Java; compiler != (bufremotepluginconfig.MavenCompilerJavaConfig{}) {
			javaCompilerConfig = &registryv1alpha1.MavenConfig_CompilerJavaConfig{
				Encoding: compiler.Encoding,
				Release:  int32(compiler.Release),
				Source:   int32(compiler.Source),
				Target:   int32(compiler.Target),
			}
		}
		var kotlinCompilerConfig *registryv1alpha1.MavenConfig_CompilerKotlinConfig
		if compiler := pluginRegistry.Maven.Compiler.Kotlin; compiler != (bufremotepluginconfig.MavenCompilerKotlinConfig{}) {
			kotlinCompilerConfig = &registryv1alpha1.MavenConfig_CompilerKotlinConfig{
				Version:         compiler.Version,
				ApiVersion:      compiler.APIVersion,
				JvmTarget:       compiler.JVMTarget,
				LanguageVersion: compiler.LanguageVersion,
			}
		}
		if javaCompilerConfig != nil || kotlinCompilerConfig != nil {
			mavenConfig.Compiler = &registryv1alpha1.MavenConfig_CompilerConfig{
				Java:   javaCompilerConfig,
				Kotlin: kotlinCompilerConfig,
			}
		}
		if pluginRegistry.Maven.Deps != nil {
			mavenConfig.RuntimeLibraries = make([]*registryv1alpha1.MavenConfig_RuntimeLibrary, len(pluginRegistry.Maven.Deps))
			for i, dependency := range pluginRegistry.Maven.Deps {
				mavenConfig.RuntimeLibraries[i] = MavenDependencyConfigToProtoRuntimeLibrary(dependency)
			}
		}
		if pluginRegistry.Maven.AdditionalRuntimes != nil {
			mavenConfig.AdditionalRuntimes = make([]*registryv1alpha1.MavenConfig_RuntimeConfig, len(pluginRegistry.Maven.AdditionalRuntimes))
			for i, runtime := range pluginRegistry.Maven.AdditionalRuntimes {
				mavenConfig.AdditionalRuntimes[i] = MavenRuntimeConfigToProtoRuntimeConfig(runtime)
			}
		}
		registryConfig.RegistryConfig = &registryv1alpha1.RegistryConfig_MavenConfig{MavenConfig: mavenConfig}
	} else if pluginRegistry.Swift != nil {
		swiftConfig := SwiftRegistryConfigToProtoSwiftConfig(pluginRegistry.Swift)
		registryConfig.RegistryConfig = &registryv1alpha1.RegistryConfig_SwiftConfig{SwiftConfig: swiftConfig}
	} else if pluginRegistry.Python != nil {
		pythonConfig, err := PythonRegistryConfigToProtoPythonConfig(pluginRegistry.Python)
		if err != nil {
			return nil, err
		}
		registryConfig.RegistryConfig = &registryv1alpha1.RegistryConfig_PythonConfig{PythonConfig: pythonConfig}
	} else if pluginRegistry.Cargo != nil {
		cargoConfig, err := CargoRegistryConfigToProtoCargoConfig(pluginRegistry.Cargo)
		if err != nil {
			return nil, err
		}
		registryConfig.RegistryConfig = &registryv1alpha1.RegistryConfig_CargoConfig{CargoConfig: cargoConfig}
	}
	return registryConfig, nil
}

// MavenDependencyConfigToProtoRuntimeLibrary converts a bufremotepluginconfig.MavenDependencyConfig to an equivalent registryv1alpha1.MavenConfig_RuntimeLibrary.
func MavenDependencyConfigToProtoRuntimeLibrary(dependency bufremotepluginconfig.MavenDependencyConfig) *registryv1alpha1.MavenConfig_RuntimeLibrary {
	return &registryv1alpha1.MavenConfig_RuntimeLibrary{
		GroupId:    dependency.GroupID,
		ArtifactId: dependency.ArtifactID,
		Version:    dependency.Version,
		Classifier: dependency.Classifier,
		Extension:  dependency.Extension,
	}
}

// ProtoRegistryConfigToPluginRegistry converts a registryv1alpha1.RegistryConfig to a bufremotepluginconfig.RegistryConfig .
func ProtoRegistryConfigToPluginRegistry(config *registryv1alpha1.RegistryConfig) (*bufremotepluginconfig.RegistryConfig, error) {
	if config == nil {
		return nil, nil
	}
	registryConfig := &bufremotepluginconfig.RegistryConfig{
		Options: bufremotepluginconfig.OptionsSliceToPluginOptions(config.Options),
	}
	if config.GetGoConfig() != nil {
		goConfig := &bufremotepluginconfig.GoRegistryConfig{}
		goConfig.MinVersion = config.GetGoConfig().GetMinimumVersion()
		runtimeLibraries := config.GetGoConfig().GetRuntimeLibraries()
		if runtimeLibraries != nil {
			goConfig.Deps = make([]*bufremotepluginconfig.GoRegistryDependencyConfig, 0, len(runtimeLibraries))
			for _, library := range runtimeLibraries {
				goConfig.Deps = append(goConfig.Deps, protoGoRuntimeLibraryToGoRuntimeDependency(library))
			}
		}
		registryConfig.Go = goConfig
	} else if config.GetNpmConfig() != nil {
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
	} else if protoMavenConfig := config.GetMavenConfig(); protoMavenConfig != nil {
		mavenConfig, err := ProtoMavenConfigToMavenRegistryConfig(protoMavenConfig)
		if err != nil {
			return nil, err
		}
		registryConfig.Maven = mavenConfig
	} else if protoSwiftConfig := config.GetSwiftConfig(); protoSwiftConfig != nil {
		swiftConfig, err := ProtoSwiftConfigToSwiftRegistryConfig(protoSwiftConfig)
		if err != nil {
			return nil, err
		}
		registryConfig.Swift = swiftConfig
	} else if protoPythonConfig := config.GetPythonConfig(); protoPythonConfig != nil {
		pythonConfig, err := ProtoPythonConfigToPythonRegistryConfig(protoPythonConfig)
		if err != nil {
			return nil, err
		}
		registryConfig.Python = pythonConfig
	} else if protoCargoConfig := config.GetCargoConfig(); protoCargoConfig != nil {
		cargoConfig, err := ProtoCargoConfigToCargoRegistryConfig(protoCargoConfig)
		if err != nil {
			return nil, err
		}
		registryConfig.Cargo = cargoConfig
	}
	return registryConfig, nil
}

// ProtoCargoConfigToCargoRegistryConfig converts protoCargoConfig to an equivalent [*bufremotepluginconfig.CargoRegistryConfig].
func ProtoCargoConfigToCargoRegistryConfig(protoCargoConfig *registryv1alpha1.CargoConfig) (*bufremotepluginconfig.CargoRegistryConfig, error) {
	cargoConfig := &bufremotepluginconfig.CargoRegistryConfig{
		RustVersion: protoCargoConfig.RustVersion,
	}
	for _, dependency := range protoCargoConfig.RuntimeLibraries {
		cargoConfig.Deps = append(cargoConfig.Deps, bufremotepluginconfig.CargoRegistryDependency{
			Name:               dependency.Name,
			VersionRequirement: dependency.VersionRequirement,
			DefaultFeatures:    dependency.DefaultFeatures,
			Features:           dependency.Features,
		})
	}
	return cargoConfig, nil
}

// CargoRegistryConfigToProtoCargoConfig converts cargoConfig to an equivalent [*registryv1alpha1.CargoConfig].
func CargoRegistryConfigToProtoCargoConfig(cargoConfig *bufremotepluginconfig.CargoRegistryConfig) (*registryv1alpha1.CargoConfig, error) {
	protoCargoConfig := &registryv1alpha1.CargoConfig{
		RustVersion: cargoConfig.RustVersion,
	}
	for _, dependency := range cargoConfig.Deps {
		protoCargoConfig.RuntimeLibraries = append(protoCargoConfig.RuntimeLibraries, &registryv1alpha1.CargoConfig_RuntimeLibrary{
			Name:               dependency.Name,
			VersionRequirement: dependency.VersionRequirement,
			DefaultFeatures:    dependency.DefaultFeatures,
			Features:           dependency.Features,
		})
	}
	return protoCargoConfig, nil
}

// ProtoPythonConfigToPythonRegistryConfig converts protoPythonConfig to an equivalent [*bufremotepluginconfig.PythonRegistryConfig].
func ProtoPythonConfigToPythonRegistryConfig(protoPythonConfig *registryv1alpha1.PythonConfig) (*bufremotepluginconfig.PythonRegistryConfig, error) {
	pythonConfig := &bufremotepluginconfig.PythonRegistryConfig{
		RequiresPython: protoPythonConfig.RequiresPython,
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
		pythonConfig.Deps = append(pythonConfig.Deps, runtimeLibrary.DependencySpecification)
	}
	return pythonConfig, nil
}

// PythonRegistryConfigToProtoPythonConfig converts pythonConfig to an equivalent [*registryv1alpha1.PythonConfig].
func PythonRegistryConfigToProtoPythonConfig(pythonConfig *bufremotepluginconfig.PythonRegistryConfig) (*registryv1alpha1.PythonConfig, error) {
	protoPythonConfig := &registryv1alpha1.PythonConfig{
		RequiresPython: pythonConfig.RequiresPython,
	}
	switch pythonConfig.PackageType {
	case "runtime":
		protoPythonConfig.PackageType = registryv1alpha1.PythonPackageType_PYTHON_PACKAGE_TYPE_RUNTIME
	case "stub-only":
		protoPythonConfig.PackageType = registryv1alpha1.PythonPackageType_PYTHON_PACKAGE_TYPE_STUB_ONLY
	default:
		return nil, fmt.Errorf(`invalid python config package_type; expecting one of "runtime" or "stub-only", got %q`, pythonConfig.PackageType)
	}
	for _, dependencySpecification := range pythonConfig.Deps {
		protoPythonConfig.RuntimeLibraries = append(protoPythonConfig.RuntimeLibraries, &registryv1alpha1.PythonConfig_RuntimeLibrary{
			DependencySpecification: dependencySpecification,
		})
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
		protoSwiftConfig.RuntimeLibraries = make([]*registryv1alpha1.SwiftConfig_RuntimeLibrary, 0, len(swiftConfig.Dependencies))
		for _, dependency := range swiftConfig.Dependencies {
			depConfig := &registryv1alpha1.SwiftConfig_RuntimeLibrary{
				Source:        dependency.Source,
				Package:       dependency.Package,
				Version:       dependency.Version,
				Products:      dependency.Products,
				SwiftVersions: dependency.SwiftVersions,
			}
			if dependency.Platforms.MacOS != "" {
				depConfig.Platforms = append(depConfig.Platforms, &registryv1alpha1.SwiftConfig_RuntimeLibrary_Platform{
					Name:    registryv1alpha1.SwiftPlatformType_SWIFT_PLATFORM_TYPE_MACOS,
					Version: dependency.Platforms.MacOS,
				})
			}
			if dependency.Platforms.IOS != "" {
				depConfig.Platforms = append(depConfig.Platforms, &registryv1alpha1.SwiftConfig_RuntimeLibrary_Platform{
					Name:    registryv1alpha1.SwiftPlatformType_SWIFT_PLATFORM_TYPE_IOS,
					Version: dependency.Platforms.IOS,
				})
			}
			if dependency.Platforms.TVOS != "" {
				depConfig.Platforms = append(depConfig.Platforms, &registryv1alpha1.SwiftConfig_RuntimeLibrary_Platform{
					Name:    registryv1alpha1.SwiftPlatformType_SWIFT_PLATFORM_TYPE_TVOS,
					Version: dependency.Platforms.TVOS,
				})
			}
			if dependency.Platforms.WatchOS != "" {
				depConfig.Platforms = append(depConfig.Platforms, &registryv1alpha1.SwiftConfig_RuntimeLibrary_Platform{
					Name:    registryv1alpha1.SwiftPlatformType_SWIFT_PLATFORM_TYPE_WATCHOS,
					Version: dependency.Platforms.WatchOS,
				})
			}
			protoSwiftConfig.RuntimeLibraries = append(protoSwiftConfig.RuntimeLibraries, depConfig)
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
	return &registryv1alpha1.MavenConfig_RuntimeConfig{
		Name:             runtime.Name,
		RuntimeLibraries: libraries,
		Options:          runtime.Options,
	}
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
	return &registryv1alpha1.GoConfig_RuntimeLibrary{
		Module:  config.Module,
		Version: config.Version,
	}
}

// protoGoRuntimeLibraryToGoRuntimeDependency converts a registryv1alpha1.GoConfig_RuntimeLibrary to a bufremotepluginconfig.GoRegistryDependencyConfig.
func protoGoRuntimeLibraryToGoRuntimeDependency(config *registryv1alpha1.GoConfig_RuntimeLibrary) *bufremotepluginconfig.GoRegistryDependencyConfig {
	return &bufremotepluginconfig.GoRegistryDependencyConfig{
		Module:  config.Module,
		Version: config.Version,
	}
}

// npmRuntimeDependencyToProtoNPMRuntimeLibrary converts a bufremotepluginconfig.NPMRegistryConfig to a registryv1alpha1.NPMConfig_RuntimeLibrary.
func npmRuntimeDependencyToProtoNPMRuntimeLibrary(config *bufremotepluginconfig.NPMRegistryDependencyConfig) *registryv1alpha1.NPMConfig_RuntimeLibrary {
	return &registryv1alpha1.NPMConfig_RuntimeLibrary{
		Package: config.Package,
		Version: config.Version,
	}
}

// protoNPMRuntimeLibraryToNPMRuntimeDependency converts a registryv1alpha1.NPMConfig_RuntimeLibrary to a bufremotepluginconfig.NPMRegistryDependencyConfig.
func protoNPMRuntimeLibraryToNPMRuntimeDependency(config *registryv1alpha1.NPMConfig_RuntimeLibrary) *bufremotepluginconfig.NPMRegistryDependencyConfig {
	return &bufremotepluginconfig.NPMRegistryDependencyConfig{
		Package: config.Package,
		Version: config.Version,
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
	return &registryv1alpha1.CuratedPluginReference{
		Owner:    reference.Owner(),
		Name:     reference.Plugin(),
		Version:  reference.Version(),
		Revision: uint32(reference.Revision()),
	}
}

// PluginIdentityToProtoCuratedPluginReference converts a bufremotepluginref.PluginIdentity to a registryv1alpha1.CuratedPluginReference.
//
// The returned CuratedPluginReference contains no Version/Revision information.
func PluginIdentityToProtoCuratedPluginReference(identity bufremotepluginref.PluginIdentity) *registryv1alpha1.CuratedPluginReference {
	if identity == nil {
		return nil
	}
	return &registryv1alpha1.CuratedPluginReference{
		Owner: identity.Owner(),
		Name:  identity.Plugin(),
	}
}

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

package bufplugin

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginref"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
)

// Plugin represents a plugin defined by a buf.plugin.yaml.
type Plugin interface {
	// Version is the version of the plugin's implementation
	// (e.g the protoc-gen-connect-go implementation is v0.2.0).
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
	Dependencies() []bufpluginref.PluginReference
	// DefaultOptions is the set of default options passed to the plugin.
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
	DefaultOptions() map[string]string
	// Registry is the registry configuration, which lets the user specify
	// registry dependencies, and other metadata that applies to a specific
	// remote generation registry (e.g. the Go module proxy, NPM registry,
	// etc).
	Registry() *bufpluginconfig.RegistryConfig
	// ContainerImageDigest returns the plugin's source image digest.
	//
	// For now we only support docker image sources, but this
	// might evolve to support others later on.
	ContainerImageDigest() string
}

// NewPlugin creates a new plugin from the given configuration and image digest.
func NewPlugin(
	version string,
	dependencies []bufpluginref.PluginReference,
	defaultOptions map[string]string,
	registryConfig *bufpluginconfig.RegistryConfig,
	imageDigest string,
	sourceURL string,
	description string,
) (Plugin, error) {
	return newPlugin(version, dependencies, defaultOptions, registryConfig, imageDigest, sourceURL, description)
}

// PluginToProtoPluginRegistryType determines the appropriate registryv1alpha1.PluginRegistryType for the plugin.
func PluginToProtoPluginRegistryType(plugin Plugin) registryv1alpha1.PluginRegistryType {
	registryType := registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_UNSPECIFIED
	if plugin.Registry() != nil {
		if plugin.Registry().Go != nil {
			registryType = registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_GO
		} else if plugin.Registry().NPM != nil {
			registryType = registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_NPM
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

// PluginRegistryToProtoRegistryConfig converts a bufpluginconfig.RegistryConfig to a registryv1alpha1.RegistryConfig.
func PluginRegistryToProtoRegistryConfig(pluginRegistry *bufpluginconfig.RegistryConfig) *registryv1alpha1.RegistryConfig {
	if pluginRegistry == nil {
		return nil
	}
	registryConfig := &registryv1alpha1.RegistryConfig{
		Options: bufpluginconfig.PluginOptionsToOptionsSlice(pluginRegistry.Options),
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
		npmConfig := &registryv1alpha1.NPMConfig{
			RewriteImportPathSuffix: pluginRegistry.NPM.RewriteImportPathSuffix,
			ImportStyle:             pluginRegistry.NPM.ImportStyle,
		}
		if pluginRegistry.NPM.Deps != nil {
			npmConfig.RuntimeLibraries = make([]*registryv1alpha1.NPMConfig_RuntimeLibrary, 0, len(pluginRegistry.NPM.Deps))
			for _, dependency := range pluginRegistry.NPM.Deps {
				npmConfig.RuntimeLibraries = append(npmConfig.RuntimeLibraries, npmRuntimeDependencyToProtoNPMRuntimeLibrary(dependency))
			}
		}
		registryConfig.RegistryConfig = &registryv1alpha1.RegistryConfig_NpmConfig{NpmConfig: npmConfig}
	}
	return registryConfig
}

// ProtoRegistryConfigToPluginRegistry converts a registryv1alpha1.RegistryConfig to a bufpluginconfig.RegistryConfig .
func ProtoRegistryConfigToPluginRegistry(config *registryv1alpha1.RegistryConfig) *bufpluginconfig.RegistryConfig {
	if config == nil {
		return nil
	}
	registryConfig := &bufpluginconfig.RegistryConfig{
		Options: bufpluginconfig.OptionsSliceToPluginOptions(config.Options),
	}
	if config.GetGoConfig() != nil {
		goConfig := &bufpluginconfig.GoRegistryConfig{}
		goConfig.MinVersion = config.GetGoConfig().GetMinimumVersion()
		runtimeLibraries := config.GetGoConfig().GetRuntimeLibraries()
		if runtimeLibraries != nil {
			goConfig.Deps = make([]*bufpluginconfig.GoRegistryDependencyConfig, 0, len(runtimeLibraries))
			for _, library := range runtimeLibraries {
				goConfig.Deps = append(goConfig.Deps, protoGoRuntimeLibraryToGoRuntimeDependency(library))
			}
		}
		registryConfig.Go = goConfig
	} else if config.GetNpmConfig() != nil {
		npmConfig := &bufpluginconfig.NPMRegistryConfig{
			RewriteImportPathSuffix: config.GetNpmConfig().GetRewriteImportPathSuffix(),
			ImportStyle:             config.GetNpmConfig().GetImportStyle(),
		}
		runtimeLibraries := config.GetNpmConfig().GetRuntimeLibraries()
		if runtimeLibraries != nil {
			npmConfig.Deps = make([]*bufpluginconfig.NPMRegistryDependencyConfig, 0, len(runtimeLibraries))
			for _, library := range runtimeLibraries {
				npmConfig.Deps = append(npmConfig.Deps, protoNPMRuntimeLibraryToNPMRuntimeDependency(library))
			}
		}
		registryConfig.NPM = npmConfig
	}
	return registryConfig
}

// goRuntimeDependencyToProtoGoRuntimeLibrary converts a bufpluginconfig.GoRegistryDependencyConfig to a registryv1alpha1.GoConfig_RuntimeLibrary.
func goRuntimeDependencyToProtoGoRuntimeLibrary(config *bufpluginconfig.GoRegistryDependencyConfig) *registryv1alpha1.GoConfig_RuntimeLibrary {
	return &registryv1alpha1.GoConfig_RuntimeLibrary{
		Module:  config.Module,
		Version: config.Version,
	}
}

// protoGoRuntimeLibraryToGoRuntimeDependency converts a registryv1alpha1.GoConfig_RuntimeLibrary to a bufpluginconfig.GoRegistryDependencyConfig.
func protoGoRuntimeLibraryToGoRuntimeDependency(config *registryv1alpha1.GoConfig_RuntimeLibrary) *bufpluginconfig.GoRegistryDependencyConfig {
	return &bufpluginconfig.GoRegistryDependencyConfig{
		Module:  config.Module,
		Version: config.Version,
	}
}

// npmRuntimeDependencyToProtoNPMRuntimeLibrary converts a bufpluginconfig.NPMRegistryConfig to a registryv1alpha1.NPMConfig_RuntimeLibrary.
func npmRuntimeDependencyToProtoNPMRuntimeLibrary(config *bufpluginconfig.NPMRegistryDependencyConfig) *registryv1alpha1.NPMConfig_RuntimeLibrary {
	return &registryv1alpha1.NPMConfig_RuntimeLibrary{
		Package: config.Package,
		Version: config.Version,
	}
}

// protoNPMRuntimeLibraryToNPMRuntimeDependency converts a registryv1alpha1.NPMConfig_RuntimeLibrary to a bufpluginconfig.NPMRegistryDependencyConfig.
func protoNPMRuntimeLibraryToNPMRuntimeDependency(config *registryv1alpha1.NPMConfig_RuntimeLibrary) *bufpluginconfig.NPMRegistryDependencyConfig {
	return &bufpluginconfig.NPMRegistryDependencyConfig{
		Package: config.Package,
		Version: config.Version,
	}
}

// PluginReferencesToCuratedProtoPluginReferences converts a slice of bufpluginref.PluginReference to a slice of registryv1alpha1.CuratedPluginReference.
func PluginReferencesToCuratedProtoPluginReferences(references []bufpluginref.PluginReference) []*registryv1alpha1.CuratedPluginReference {
	if references == nil {
		return nil
	}
	protoReferences := make([]*registryv1alpha1.CuratedPluginReference, 0, len(references))
	for _, reference := range references {
		protoReferences = append(protoReferences, PluginReferenceToProtoCuratedPluginReference(reference))
	}
	return protoReferences
}

// PluginReferenceToProtoCuratedPluginReference converts a bufpluginref.PluginReference to a registryv1alpha1.CuratedPluginReference.
func PluginReferenceToProtoCuratedPluginReference(reference bufpluginref.PluginReference) *registryv1alpha1.CuratedPluginReference {
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

// PluginIdentityToProtoCuratedPluginReference converts a bufpluginref.PluginIdentity to a registryv1alpha1.CuratedPluginReference.
//
// The returned CuratedPluginReference contains no Version/Revision information.
func PluginIdentityToProtoCuratedPluginReference(identity bufpluginref.PluginIdentity) *registryv1alpha1.CuratedPluginReference {
	if identity == nil {
		return nil
	}
	return &registryv1alpha1.CuratedPluginReference{
		Owner: identity.Owner(),
		Name:  identity.Plugin(),
	}
}

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
	// Runtime is the runtime configuration, which lets the user specify
	// runtime dependencies, and other metadata that applies to a specific
	// remote generation registry (e.g. the Go module proxy, NPM registry,
	// etc).
	Runtime() *bufpluginconfig.RuntimeConfig
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
	runtimeConfig *bufpluginconfig.RuntimeConfig,
	imageDigest string,
	sourceURL string,
	description string,
) (Plugin, error) {
	return newPlugin(version, dependencies, defaultOptions, runtimeConfig, imageDigest, sourceURL, description)
}

// PluginToProtoPluginLanguage determines the appropriate registryv1alpha1.PluginLanguage for the plugin.
func PluginToProtoPluginLanguage(plugin Plugin) registryv1alpha1.PluginLanguage {
	language := registryv1alpha1.PluginLanguage_PLUGIN_LANGUAGE_UNSPECIFIED
	if plugin.Runtime() != nil {
		if plugin.Runtime().Go != nil {
			language = registryv1alpha1.PluginLanguage_PLUGIN_LANGUAGE_GO
		} else if plugin.Runtime().NPM != nil {
			language = registryv1alpha1.PluginLanguage_PLUGIN_LANGUAGE_NPM
		}
	}
	return language
}

// PluginRuntimeToProtoRuntimeConfig converts a bufpluginconfig.RuntimeConfig to a registryv1alpha1.RuntimeConfig.
func PluginRuntimeToProtoRuntimeConfig(pluginRuntime *bufpluginconfig.RuntimeConfig) *registryv1alpha1.RuntimeConfig {
	if pluginRuntime == nil {
		return nil
	}
	runtimeConfig := &registryv1alpha1.RuntimeConfig{}
	if pluginRuntime.Go != nil {
		goConfig := &registryv1alpha1.GoConfig{}
		goConfig.MinimumVersion = pluginRuntime.Go.MinVersion
		goConfig.RuntimeLibraries = make([]*registryv1alpha1.GoConfig_RuntimeLibrary, 0, len(pluginRuntime.Go.Deps))
		for _, dependency := range pluginRuntime.Go.Deps {
			goConfig.RuntimeLibraries = append(goConfig.RuntimeLibraries, goRuntimeDependencyToProtoGoRuntimeLibrary(dependency))
		}
		runtimeConfig.RuntimeConfig = &registryv1alpha1.RuntimeConfig_GoConfig{GoConfig: goConfig}
	} else if pluginRuntime.NPM != nil {
		npmConfig := &registryv1alpha1.NPMConfig{}
		npmConfig.RuntimeLibraries = make([]*registryv1alpha1.NPMConfig_RuntimeLibrary, 0, len(pluginRuntime.NPM.Deps))
		for _, dependency := range pluginRuntime.NPM.Deps {
			npmConfig.RuntimeLibraries = append(npmConfig.RuntimeLibraries, npmRuntimeDependencyToProtoNPMRuntimeLibrary(dependency))
		}
		runtimeConfig.RuntimeConfig = &registryv1alpha1.RuntimeConfig_NpmConfig{NpmConfig: npmConfig}
	}
	return runtimeConfig
}

// ProtoRuntimeConfigToPluginRuntime converts a registryv1alpha1.RuntimeConfig to a bufpluginconfig.RuntimeConfig .
func ProtoRuntimeConfigToPluginRuntime(config *registryv1alpha1.RuntimeConfig) *bufpluginconfig.RuntimeConfig {
	if config == nil {
		return nil
	}
	runtimeConfig := &bufpluginconfig.RuntimeConfig{}
	if config.GetGoConfig() != nil {
		goConfig := &bufpluginconfig.GoRuntimeConfig{}
		goConfig.MinVersion = config.GetGoConfig().GetMinimumVersion()
		goConfig.Deps = make([]*bufpluginconfig.GoRuntimeDependencyConfig, 0, len(config.GetGoConfig().GetRuntimeLibraries()))
		for _, library := range config.GetGoConfig().GetRuntimeLibraries() {
			goConfig.Deps = append(goConfig.Deps, protoGoRuntimeLibraryToGoRuntimeDependency(library))
		}
		runtimeConfig.Go = goConfig
	} else if config.GetNpmConfig() != nil {
		npmConfig := &bufpluginconfig.NPMRuntimeConfig{}
		npmConfig.Deps = make([]*bufpluginconfig.NPMRuntimeDependencyConfig, 0, len(config.GetNpmConfig().GetRuntimeLibraries()))
		for _, library := range config.GetNpmConfig().GetRuntimeLibraries() {
			npmConfig.Deps = append(npmConfig.Deps, protoNPMRuntimeLibraryToNPMRuntimeDependency(library))
		}
		runtimeConfig.NPM = npmConfig
	}
	return runtimeConfig
}

// goRuntimeDependencyToProtoGoRuntimeLibrary converts a bufpluginconfig.GoRuntimeDependencyConfig to a registryv1alpha1.GoConfig_RuntimeLibrary.
func goRuntimeDependencyToProtoGoRuntimeLibrary(config *bufpluginconfig.GoRuntimeDependencyConfig) *registryv1alpha1.GoConfig_RuntimeLibrary {
	return &registryv1alpha1.GoConfig_RuntimeLibrary{
		Module:  config.Module,
		Version: config.Version,
	}
}

// protoGoRuntimeLibraryToGoRuntimeDependency converts a registryv1alpha1.GoConfig_RuntimeLibrary to a bufpluginconfig.GoRuntimeDependencyConfig.
func protoGoRuntimeLibraryToGoRuntimeDependency(config *registryv1alpha1.GoConfig_RuntimeLibrary) *bufpluginconfig.GoRuntimeDependencyConfig {
	return &bufpluginconfig.GoRuntimeDependencyConfig{
		Module:  config.Module,
		Version: config.Version,
	}
}

// npmRuntimeDependencyToProtoNPMRuntimeLibrary converts a bufpluginconfig.NPMRuntimeConfig to a registryv1alpha1.NPMConfig_RuntimeLibrary.
func npmRuntimeDependencyToProtoNPMRuntimeLibrary(config *bufpluginconfig.NPMRuntimeDependencyConfig) *registryv1alpha1.NPMConfig_RuntimeLibrary {
	return &registryv1alpha1.NPMConfig_RuntimeLibrary{
		Package: config.Package,
		Version: config.Version,
	}
}

// protoNPMRuntimeLibraryToNPMRuntimeDependency converts a registryv1alpha1.NPMConfig_RuntimeLibrary to a bufpluginconfig.NPMRuntimeDependencyConfig.
func protoNPMRuntimeLibraryToNPMRuntimeDependency(config *registryv1alpha1.NPMConfig_RuntimeLibrary) *bufpluginconfig.NPMRuntimeDependencyConfig {
	return &bufpluginconfig.NPMRuntimeDependencyConfig{
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

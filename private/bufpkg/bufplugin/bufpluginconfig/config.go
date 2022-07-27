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

package bufpluginconfig

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginref"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
)

func newConfig(externalConfig ExternalConfig) (*Config, error) {
	pluginIdentity, err := bufpluginref.PluginIdentityForString(externalConfig.Name)
	if err != nil {
		return nil, err
	}
	pluginVersion := externalConfig.PluginVersion
	if pluginVersion == "" {
		return nil, errors.New("a plugin_version is required")
	}
	if !semver.IsValid(pluginVersion) {
		return nil, fmt.Errorf("plugin_version %q must be a valid semantic version", externalConfig.PluginVersion)
	}
	var defaultOptions map[string]string
	if len(externalConfig.DefaultOpts) > 0 {
		// We only want to create a non-nil map if the user
		// actually specified any options.
		defaultOptions = make(map[string]string)
	}
	for _, option := range externalConfig.DefaultOpts {
		split := strings.Split(option, "=")
		if len(split) > 2 {
			return nil, errors.New(`plugin default_options must be specified as "<key>=<value>" strings`)
		}
		if len(split) == 1 {
			// Some plugins don't actually specify the '=' delimiter
			// (e.g. protoc-gen-java's java_opt=annotate_code). To
			// support these legacy options, we map the key to an empty
			// string value.
			//
			// This means that plugin options with an explicit
			// 'something=""' option are actually passed in as
			// --something (omitting the explicit "" entirely).
			//
			// This behavior might need to change depending on if
			// there are valid use cases here, but we eventually
			// want to support structured options as key, value
			// pairs so we enforce this implicit behavior for now.
			split = append(split, "")
		}
		key, value := split[0], split[1]
		if _, ok := defaultOptions[key]; ok {
			return nil, fmt.Errorf("plugin default option %q was specified more than once", key)
		}
		defaultOptions[key] = value
	}
	var dependencies []bufpluginref.PluginReference
	if len(externalConfig.Deps) > 0 {
		existingDeps := make(map[string]struct{})
		for _, dependency := range externalConfig.Deps {
			reference, err := bufpluginref.PluginReferenceForString(dependency.Plugin, dependency.Revision)
			if err != nil {
				return nil, err
			}
			if reference.Remote() != pluginIdentity.Remote() {
				return nil, fmt.Errorf("plugin dependency %q must use same remote as plugin %q", dependency, pluginIdentity.Remote())
			}
			if _, ok := existingDeps[reference.IdentityString()]; ok {
				return nil, fmt.Errorf("plugin dependency %q was specified more than once", dependency)
			}
			existingDeps[reference.IdentityString()] = struct{}{}
			dependencies = append(dependencies, reference)
		}
	}
	runtimeConfig, err := newRuntimeConfig(externalConfig.Runtime)
	if err != nil {
		return nil, err
	}
	return &Config{
		Name:           pluginIdentity,
		PluginVersion:  pluginVersion,
		DefaultOptions: defaultOptions,
		Dependencies:   dependencies,
		Runtime:        runtimeConfig,
		SourceURL:      externalConfig.SourceURL,
		Description:    externalConfig.Description,
	}, nil
}

func newRuntimeConfig(externalRuntimeConfig ExternalRuntimeConfig) (*RuntimeConfig, error) {
	var (
		isGoEmpty  = externalRuntimeConfig.Go.IsEmpty()
		isNPMEmpty = externalRuntimeConfig.NPM.IsEmpty()
	)
	var runtimeCount int
	for _, isEmpty := range []bool{
		isGoEmpty,
		isNPMEmpty,
	} {
		if !isEmpty {
			runtimeCount++
		}
		if runtimeCount > 1 {
			// We might eventually want to support multiple runtime configuration,
			// but it's safe to start with an error for now.
			return nil, fmt.Errorf("%s configuration contains multiple runtime languages", ExternalConfigFilePath)
		}
	}
	if runtimeCount == 0 {
		// It's possible that the plugin doesn't have any runtime dependencies.
		return nil, nil
	}
	if !isNPMEmpty {
		npmRuntimeConfig, err := newNPMRuntimeConfig(externalRuntimeConfig.NPM)
		if err != nil {
			return nil, err
		}
		return &RuntimeConfig{
			NPM: npmRuntimeConfig,
		}, nil
	}
	// At this point, the Go runtime is guaranteed to be specified. Note
	// that this will change if/when there are more runtime languages supported.
	goRuntimeConfig, err := newGoRuntimeConfig(externalRuntimeConfig.Go)
	if err != nil {
		return nil, err
	}
	return &RuntimeConfig{
		Go: goRuntimeConfig,
	}, nil
}

func newNPMRuntimeConfig(externalNPMRuntimeConfig ExternalNPMRuntimeConfig) (*NPMRuntimeConfig, error) {
	var dependencies []*NPMRuntimeDependencyConfig
	for _, dep := range externalNPMRuntimeConfig.Deps {
		if dep.Package == "" {
			return nil, errors.New("npm runtime dependency requires a non-empty package name")
		}
		if dep.Version == "" {
			return nil, errors.New("npm runtime dependency requires a non-empty version name")
		}
		// TODO: Note that we don't have NPM-specific validation yet - any
		// non-empty string will work for the package and version.
		//
		// For a complete set of the version syntax we need to support, see
		// https://docs.npmjs.com/cli/v6/using-npm/semver
		//
		// https://github.com/Masterminds/semver might be a good candidate for
		// this, but it might not support all of the constraints supported
		// by NPM.
		dependencies = append(
			dependencies,
			&NPMRuntimeDependencyConfig{
				Package: dep.Package,
				Version: dep.Version,
			},
		)
	}
	return &NPMRuntimeConfig{
		Deps: dependencies,
	}, nil
}

func newGoRuntimeConfig(externalGoRuntimeConfig ExternalGoRuntimeConfig) (*GoRuntimeConfig, error) {
	if externalGoRuntimeConfig.MinVersion != "" && !modfile.GoVersionRE.MatchString(externalGoRuntimeConfig.MinVersion) {
		return nil, fmt.Errorf("the go minimum version %q must be a valid semantic version in the form of <major>.<minor>", externalGoRuntimeConfig.MinVersion)
	}
	var dependencies []*GoRuntimeDependencyConfig
	for _, dep := range externalGoRuntimeConfig.Deps {
		if dep.Module == "" {
			return nil, errors.New("go runtime dependency requires a non-empty module name")
		}
		if dep.Version == "" {
			return nil, errors.New("go runtime dependency requires a non-empty version name")
		}
		if !semver.IsValid(dep.Version) {
			return nil, fmt.Errorf("go runtime dependency %s:%s does not have a valid semantic version", dep.Module, dep.Version)
		}
		dependencies = append(
			dependencies,
			&GoRuntimeDependencyConfig{
				Module:  dep.Module,
				Version: dep.Version,
			},
		)
	}
	return &GoRuntimeConfig{
		MinVersion: externalGoRuntimeConfig.MinVersion,
		Deps:       dependencies,
	}, nil
}

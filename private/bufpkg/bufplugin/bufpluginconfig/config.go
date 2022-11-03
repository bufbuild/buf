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
	"github.com/bufbuild/buf/private/gen/data/dataspdx"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
)

func newConfig(externalConfig ExternalConfig, options []ConfigOption) (*Config, error) {
	opts := &configOptions{}
	for _, option := range options {
		option(opts)
	}
	pluginIdentity, err := pluginIdentityForStringWithOverrideRemote(externalConfig.Name, opts.overrideRemote)
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
			// pairs, so we enforce this implicit behavior for now.
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
			reference, err := pluginReferenceForStringWithOverrideRemote(dependency.Plugin, dependency.Revision, opts.overrideRemote)
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
	registryConfig, err := newRegistryConfig(externalConfig.Registry)
	if err != nil {
		return nil, err
	}
	spdxLicenseID := externalConfig.SPDXLicenseID
	if spdxLicenseID != "" {
		if licenseInfo, ok := dataspdx.GetLicenseInfo(spdxLicenseID); ok {
			spdxLicenseID = licenseInfo.ID()
		} else {
			return nil, fmt.Errorf("unknown SPDX License ID %q", spdxLicenseID)
		}
	}
	return &Config{
		Name:            pluginIdentity,
		PluginVersion:   pluginVersion,
		DefaultOptions:  defaultOptions,
		Dependencies:    dependencies,
		Registry:        registryConfig,
		SourceURL:       externalConfig.SourceURL,
		Description:     externalConfig.Description,
		OutputLanguages: externalConfig.OutputLanguages,
		SPDXLicenseID:   spdxLicenseID,
		LicenseURL:      externalConfig.LicenseURL,
	}, nil
}

func newRegistryConfig(externalRegistryConfig ExternalRegistryConfig) (*RegistryConfig, error) {
	var (
		isGoEmpty  = externalRegistryConfig.Go == nil
		isNPMEmpty = externalRegistryConfig.NPM == nil
	)
	var registryCount int
	for _, isEmpty := range []bool{
		isGoEmpty,
		isNPMEmpty,
	} {
		if !isEmpty {
			registryCount++
		}
		if registryCount > 1 {
			// We might eventually want to support multiple runtime configuration,
			// but it's safe to start with an error for now.
			return nil, fmt.Errorf("%s configuration contains multiple registry configurations", ExternalConfigFilePath)
		}
	}
	if registryCount == 0 {
		// It's possible that the plugin doesn't have any runtime dependencies.
		return nil, nil
	}
	options := OptionsSliceToPluginOptions(externalRegistryConfig.Opts)
	if !isNPMEmpty {
		npmRegistryConfig, err := newNPMRegistryConfig(externalRegistryConfig.NPM)
		if err != nil {
			return nil, err
		}
		return &RegistryConfig{
			NPM:     npmRegistryConfig,
			Options: options,
		}, nil
	}
	// At this point, the Go runtime is guaranteed to be specified. Note
	// that this will change if/when there are more runtime languages supported.
	goRegistryConfig, err := newGoRegistryConfig(externalRegistryConfig.Go)
	if err != nil {
		return nil, err
	}
	return &RegistryConfig{
		Go:      goRegistryConfig,
		Options: options,
	}, nil
}

func newNPMRegistryConfig(externalNPMRegistryConfig *ExternalNPMRegistryConfig) (*NPMRegistryConfig, error) {
	if externalNPMRegistryConfig == nil {
		return nil, nil
	}
	var dependencies []*NPMRegistryDependencyConfig
	for _, dep := range externalNPMRegistryConfig.Deps {
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
			&NPMRegistryDependencyConfig{
				Package: dep.Package,
				Version: dep.Version,
			},
		)
	}
	switch externalNPMRegistryConfig.ImportStyle {
	case "module", "commonjs":
	default:
		return nil, errors.New(`npm registry config import_style must be one of: "module" or "commonjs"`)
	}
	return &NPMRegistryConfig{
		RewriteImportPathSuffix: externalNPMRegistryConfig.RewriteImportPathSuffix,
		Deps:                    dependencies,
		ImportStyle:             externalNPMRegistryConfig.ImportStyle,
	}, nil
}

func newGoRegistryConfig(externalGoRegistryConfig *ExternalGoRegistryConfig) (*GoRegistryConfig, error) {
	if externalGoRegistryConfig == nil {
		return nil, nil
	}
	if externalGoRegistryConfig.MinVersion != "" && !modfile.GoVersionRE.MatchString(externalGoRegistryConfig.MinVersion) {
		return nil, fmt.Errorf("the go minimum version %q must be a valid semantic version in the form of <major>.<minor>", externalGoRegistryConfig.MinVersion)
	}
	var dependencies []*GoRegistryDependencyConfig
	for _, dep := range externalGoRegistryConfig.Deps {
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
			&GoRegistryDependencyConfig{
				Module:  dep.Module,
				Version: dep.Version,
			},
		)
	}
	return &GoRegistryConfig{
		MinVersion: externalGoRegistryConfig.MinVersion,
		Deps:       dependencies,
	}, nil
}

func pluginIdentityForStringWithOverrideRemote(identityStr string, overrideRemote string) (bufpluginref.PluginIdentity, error) {
	identity, err := bufpluginref.PluginIdentityForString(identityStr)
	if err != nil {
		return nil, err
	}
	if len(overrideRemote) == 0 {
		return identity, nil
	}
	return bufpluginref.NewPluginIdentity(overrideRemote, identity.Owner(), identity.Plugin())
}

func pluginReferenceForStringWithOverrideRemote(
	referenceStr string,
	revision int,
	overrideRemote string,
) (bufpluginref.PluginReference, error) {
	reference, err := bufpluginref.PluginReferenceForString(referenceStr, revision)
	if err != nil {
		return nil, err
	}
	if len(overrideRemote) == 0 {
		return reference, nil
	}
	overrideIdentity, err := pluginIdentityForStringWithOverrideRemote(reference.IdentityString(), overrideRemote)
	if err != nil {
		return nil, err
	}
	return bufpluginref.NewPluginReference(overrideIdentity, reference.Version(), reference.Revision())
}

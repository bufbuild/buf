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
	// Note that the semver package we're using conforms to the
	// support SemVer syntax found in the go.mod file. This means
	// that runtime dependencies will need to specify the 'v' prefix
	// in their semantic version even if it isn't directly applicable
	// to that runtime environment (e.g. NPM).
	//
	// We'll use this for now so that runtime dependencies are
	// consistent across each runtime configuration, but we might need
	// to change this later.
	"golang.org/x/mod/semver"
)

func newConfig(externalConfig ExternalConfig) (*Config, error) {
	pluginIdentity, err := bufpluginref.PluginIdentityForString(externalConfig.Name)
	if err != nil {
		return nil, err
	}
	var options map[string]string
	if len(externalConfig.Opts) > 0 {
		// We only want to create a non-nil map if the user
		// actually specified any options.
		options = make(map[string]string)
	}
	for _, option := range externalConfig.Opts {
		split := strings.Split(option, "=")
		if len(split) > 2 {
			return nil, errors.New(`plugin options must be specified as "<key>=<value>" strings`)
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
		if _, ok := options[key]; ok {
			return nil, fmt.Errorf("plugin option %q was specified more than once", key)
		}
		options[key] = value
	}
	runtimeConfig, err := newRuntimeConfig(externalConfig.Runtime)
	if err != nil {
		return nil, err
	}
	return &Config{
		Name:    pluginIdentity,
		Options: options,
		Runtime: runtimeConfig,
	}, nil
}

func newRuntimeConfig(externalRuntimeConfig ExternalRuntimeConfig) (*RuntimeConfig, error) {
	var (
		isArchiveEmpty = externalRuntimeConfig.Archive.IsEmpty()
		isGoEmpty      = externalRuntimeConfig.Go.IsEmpty()
		isNPMEmpty     = externalRuntimeConfig.NPM.IsEmpty()
	)
	if isArchiveEmpty && isGoEmpty && isNPMEmpty {
		// It's possible that the plugin doesn't have any runtime dependencies.
		return nil, nil
	}
	if isArchiveEmpty && isGoEmpty && !isNPMEmpty {
		npmRuntimeConfig, err := newNPMRuntimeConfig(externalRuntimeConfig.NPM)
		if err != nil {
			return nil, err
		}
		return &RuntimeConfig{
			NPM: npmRuntimeConfig,
		}, nil
	}
	if isArchiveEmpty && !isGoEmpty && isNPMEmpty {
		goRuntimeConfig, err := newGoRuntimeConfig(externalRuntimeConfig.Go)
		if err != nil {
			return nil, err
		}
		return &RuntimeConfig{
			Go: goRuntimeConfig,
		}, nil
	}
	if !isArchiveEmpty && isGoEmpty && isNPMEmpty {
		archiveRuntimeConfig, err := newArchiveRuntimeConfig(externalRuntimeConfig.Archive)
		if err != nil {
			return nil, err
		}
		return &RuntimeConfig{
			Archive: archiveRuntimeConfig,
		}, nil
	}
	// If we made it this far, that means the config specifies multiple
	// runtime languages.
	//
	// We might eventually want to support multiple runtime configuration
	// (e.g. 'go' and 'archive'), but it's safe to start with an error for
	// now.
	return nil, fmt.Errorf("%s configuration contains multiple runtime languages", ExternalConfigFilePath)
}

func newNPMRuntimeConfig(externalNPMRuntimeConfig ExternalNPMRuntimeConfig) (*NPMRuntimeConfig, error) {
	if err := validateRuntimeDeps(externalNPMRuntimeConfig.Deps); err != nil {
		return nil, err
	}
	return &NPMRuntimeConfig{
		Deps: externalNPMRuntimeConfig.Deps,
	}, nil
}

func newGoRuntimeConfig(externalGoRuntimeConfig ExternalGoRuntimeConfig) (*GoRuntimeConfig, error) {
	if err := validateRuntimeDeps(externalGoRuntimeConfig.Deps); err != nil {
		return nil, err
	}
	// The best we can do is verify that the minimum version
	// is a valid semantic version, just like we do for the
	// runtime dependencies.
	//
	// This will not actually verify that the go version is
	// in the valid set. It's impossible to capture the
	// real set of valid identifiers at any given time (for
	// an old version of the buf CLI) without reaching out to
	// some external source at runtime.
	//
	// Note that this ensures the user's configuration specifies
	// a 'v' prefix in the version (e.g. v1.18) even though the
	// minimum version is rendered without it in the go.mod.
	if externalGoRuntimeConfig.MinVersion != "" && !semver.IsValid(externalGoRuntimeConfig.MinVersion) {
		return nil, fmt.Errorf("the go minimum version %q must be a valid semantic version", externalGoRuntimeConfig.MinVersion)
	}
	return &GoRuntimeConfig{
		MinVersion: externalGoRuntimeConfig.MinVersion,
		Deps:       externalGoRuntimeConfig.Deps,
	}, nil
}

func newArchiveRuntimeConfig(externalArchiveRuntimeConfig ExternalArchiveRuntimeConfig) (*ArchiveRuntimeConfig, error) {
	if err := validateRuntimeDeps(externalArchiveRuntimeConfig.Deps); err != nil {
		return nil, err
	}
	return &ArchiveRuntimeConfig{
		Deps: externalArchiveRuntimeConfig.Deps,
	}, nil
}

func validateRuntimeDeps(dependencies []string) error {
	seen := make(map[string]struct{}, len(dependencies))
	for _, dependency := range dependencies {
		split := strings.Split(dependency, ":")
		if len(split) < 2 {
			return fmt.Errorf(`runtime dependency %q must be specified as "<name>:<version>"`, dependency)
		}
		name, version := strings.Join(split[:len(split)-1], ":"), split[len(split)-1]
		if _, ok := seen[name]; ok {
			return fmt.Errorf("runtime dependency %q was specified more than once", name)
		}
		if !semver.IsValid(version) {
			return fmt.Errorf("runtime dependency %q does not have a valid semantic version", dependency)
		}
		seen[name] = struct{}{}
	}
	return nil
}

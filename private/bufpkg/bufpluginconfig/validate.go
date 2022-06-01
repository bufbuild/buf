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

	"golang.org/x/mod/semver"
)

func validateConfig(config PluginConfig) error {
	if config.Owner == "" {
		return errors.New("plugin owner cannot be empty")
	}
	if config.Name == "" {
		return errors.New("plugin name cannot be empty")
	}
	if !semver.IsValid(config.Version) {
		return fmt.Errorf("%s is not a valid semantic version", config.Version)
	}
	return validateRuntime(config.Runtime)
}

func validateRuntime(runtime Runtime) error {
	if runtime.Archive == nil && runtime.Go == nil && runtime.NPM == nil {
		return errors.New("no runtime language specified")
	}
	if runtime.Archive == nil && runtime.Go == nil && runtime.NPM != nil {
		return validateNPMConfig(*runtime.NPM)
	}
	if runtime.Archive == nil && runtime.Go != nil && runtime.NPM == nil {
		return validateGoConfig(*runtime.Go)
	}
	if runtime.Archive != nil && runtime.Go == nil && runtime.NPM == nil {
		return validateArchiveConfig(*runtime.Archive)
	}
	// If we made it this far, that means the config specifies multiple
	// runtime languages.
	return errors.New("invalid configuration contains multiple runtime languages")
}

func validateArchiveConfig(config ArchiveConfig) error {
	deps := make(map[string]string, len(config.Deps))
	for _, dep := range config.Deps {
		if dep.Name == "" || dep.Version == "" {
			return errors.New("invalid dependency, must include name and version")
		}
		if _, ok := deps[dep.Name]; ok {
			return errors.New("invalid configuration contains duplicate dependencies")
		}
		if !semver.IsValid(dep.Version) {
			return fmt.Errorf("%s is not a valid semantic version", dep.Version)
		}
		deps[dep.Name] = dep.Version
	}
	return nil
}

func validateGoConfig(config GoConfig) error {
	deps := make(map[string]string, len(config.Deps))
	for _, dep := range config.Deps {
		if dep.Module == "" || dep.Version == "" {
			return errors.New("invalid go dependency, must include module name and version")
		}
		if _, ok := deps[dep.Module]; ok {
			return errors.New("invalid configuration contains duplicate dependencies")
		}
		if !semver.IsValid(dep.Version) {
			return fmt.Errorf("%s is not a valid semantic version", dep.Version)
		}
		deps[dep.Module] = dep.Version
	}
	return nil
}

func validateNPMConfig(config NPMConfig) error {
	deps := make(map[string]string, len(config.Deps))
	for _, dep := range config.Deps {
		if dep.Package == "" || dep.Version == "" {
			return errors.New("invalid npm dependency, must include package name and version")
		}
		if _, ok := deps[dep.Package]; ok {
			return errors.New("invalid configuration contains duplicate dependencies")
		}
		if !semver.IsValid(dep.Version) {
			return fmt.Errorf("%s is not a valid semantic version", dep.Version)
		}
		deps[dep.Package] = dep.Version
	}
	return nil
}

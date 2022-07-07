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
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginconfig"
	"golang.org/x/mod/semver"
)

type plugin struct {
	version              string
	dependencies         []string
	options              map[string]string
	runtime              *bufpluginconfig.RuntimeConfig
	containerImageDigest string
}

func newPlugin(
	version string,
	dependencies []string,
	options map[string]string,
	runtimeConfig *bufpluginconfig.RuntimeConfig,
	containerImageDigest string,
) (*plugin, error) {
	if version == "" {
		return nil, errors.New("plugin version is required")
	}
	if !semver.IsValid(version) {
		// This will probably already be validated in other call-sites
		// (e.g. when we construct a *bufpluginconfig.Config or when we
		// map from the Protobuf representation), but we may as well
		// include it at the lowest common denominator, too.
		return nil, fmt.Errorf("plugin version %q must be a valid semantic version", version)
	}
	if containerImageDigest == "" {
		return nil, errors.New("plugin image digest is required")
	}
	return &plugin{
		version:              version,
		dependencies:         dependencies,
		options:              options,
		runtime:              runtimeConfig,
		containerImageDigest: containerImageDigest,
	}, nil
}

// Version returns the plugin's version.
func (p *plugin) Version() string {
	return p.version
}

// Dependencies returns the plugin's dependencies on other plugins.
func (p *plugin) Dependencies() []string {
	return p.dependencies
}

// Options returns the plugin's options.
func (p *plugin) Options() map[string]string {
	return p.options
}

// Runtime returns the plugin's runtime configuration.
func (p *plugin) Runtime() *bufpluginconfig.RuntimeConfig {
	return p.runtime
}

// ContainerImageDigest returns the plugin's image digest.
func (p *plugin) ContainerImageDigest() string {
	return p.containerImageDigest
}

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
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginref"
	"golang.org/x/mod/semver"
)

type plugin struct {
	version              string
	dependencies         []bufremotepluginref.PluginReference
	registry             *bufremotepluginconfig.RegistryConfig
	containerImageDigest string
	sourceURL            string
	description          string
}

var _ Plugin = (*plugin)(nil)

func newPlugin(
	version string,
	dependencies []bufremotepluginref.PluginReference,
	registryConfig *bufremotepluginconfig.RegistryConfig,
	containerImageDigest string,
	sourceURL string,
	description string,
) (*plugin, error) {
	if version == "" {
		return nil, errors.New("plugin version is required")
	}
	if !semver.IsValid(version) {
		// This will probably already be validated in other call-sites
		// (e.g. when we construct a *bufremotepluginconfig.Config or when we
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
		registry:             registryConfig,
		containerImageDigest: containerImageDigest,
		sourceURL:            sourceURL,
		description:          description,
	}, nil
}

// Version returns the plugin's version.
func (p *plugin) Version() string {
	return p.version
}

// Dependencies returns the plugin's dependencies on other plugins.
func (p *plugin) Dependencies() []bufremotepluginref.PluginReference {
	return p.dependencies
}

// Registry returns the plugin's registry configuration.
func (p *plugin) Registry() *bufremotepluginconfig.RegistryConfig {
	return p.registry
}

// ContainerImageDigest returns the plugin's image digest.
func (p *plugin) ContainerImageDigest() string {
	return p.containerImageDigest
}

// SourceURL is an optional attribute used to specify where the source for the plugin can be found.
func (p *plugin) SourceURL() string {
	return p.sourceURL
}

// Description is an optional attribute to provide a more detailed description for the plugin.
func (p *plugin) Description() string {
	return p.description
}

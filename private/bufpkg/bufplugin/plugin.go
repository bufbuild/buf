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
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
)

type plugin struct {
	options              map[string]string
	runtime              *bufpluginconfig.RuntimeConfig
	containerImageDigest string
}

func newPluginForProto(
	protoPlugin *registryv1alpha1.RemotePlugin,
) (*plugin, error) {
	if protoPlugin.GetContainerImageDigest() == "" {
		return nil, errors.New("plugin must have a non-empty container image digest")
	}
	var options map[string]string
	if opts := protoPlugin.GetOptions(); len(opts) > 0 {
		options = make(map[string]string)
		for _, opt := range opts {
			if opt.Name == "" {
				return nil, errors.New("plugin option must have a non-empty name")
			}
			if _, ok := options[opt.Name]; ok {
				return nil, fmt.Errorf("plugin option %q was specified more than once", opt.Name)
			}
			options[opt.Name] = opt.StringValue
		}
	}
	runtimeConfig, err := bufpluginconfig.RuntimeConfigForProto(protoPlugin.RuntimeConfig)
	if err != nil {
		return nil, err
	}
	return newPlugin(
		options,
		runtimeConfig,
		protoPlugin.GetContainerImageDigest(),
	)
}

func newPlugin(
	options map[string]string,
	runtimeConfig *bufpluginconfig.RuntimeConfig,
	containerImageDigest string,
) (*plugin, error) {
	if containerImageDigest == "" {
		return nil, errors.New("plugin image digest is required")
	}
	return &plugin{
		options:              options,
		runtime:              runtimeConfig,
		containerImageDigest: containerImageDigest,
	}, nil
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

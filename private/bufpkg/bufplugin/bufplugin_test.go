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
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginconfig"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginToProtoPluginRegistryType(t *testing.T) {
	assertPluginToPluginRegistryType(t, nil, registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_UNSPECIFIED)
	assertPluginToPluginRegistryType(t, &bufpluginconfig.RegistryConfig{Go: &bufpluginconfig.GoRegistryConfig{}}, registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_GO)
	assertPluginToPluginRegistryType(t, &bufpluginconfig.RegistryConfig{NPM: &bufpluginconfig.NPMRegistryConfig{}}, registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_NPM)
}

func assertPluginToPluginRegistryType(t testing.TB, config *bufpluginconfig.RegistryConfig, registryType registryv1alpha1.PluginRegistryType) {
	plugin, err := NewPlugin("v1.0.0", nil, nil, config, "sha256:digest", "", "")
	require.Nil(t, err)
	assert.Equal(t, registryType, PluginToProtoPluginRegistryType(plugin))
}

func TestPluginRegistryRoundTrip(t *testing.T) {
	assertPluginRegistryRoundTrip(t, nil)
	assertPluginRegistryRoundTrip(t, &bufpluginconfig.RegistryConfig{})
	assertPluginRegistryRoundTrip(t, &bufpluginconfig.RegistryConfig{
		Go: &bufpluginconfig.GoRegistryConfig{
			MinVersion: "1.18",
			Deps: []*bufpluginconfig.GoRegistryDependencyConfig{
				{
					Module:  "github.com/bufbuild/connect-go",
					Version: "v0.1.1",
				},
			},
		},
	})
	assertPluginRegistryRoundTrip(t, &bufpluginconfig.RegistryConfig{
		NPM: &bufpluginconfig.NPMRegistryConfig{
			RewriteImportPathSuffix: "connectweb.js",
			Deps: []*bufpluginconfig.NPMRegistryDependencyConfig{
				{
					Package: "@bufbuild/protobuf",
					Version: "^0.0.4",
				},
			},
		},
	})
}

func assertPluginRegistryRoundTrip(t testing.TB, config *bufpluginconfig.RegistryConfig) {
	assert.Equal(t, config, ProtoRegistryConfigToPluginRegistry(PluginRegistryToProtoRegistryConfig(config)))
}

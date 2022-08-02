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
	"github.com/stretchr/testify/assert"
)

func TestPluginRuntimeRoundTrip(t *testing.T) {
	assertPluginRuntimeRoundTrip(t, nil)
	assertPluginRuntimeRoundTrip(t, &bufpluginconfig.RuntimeConfig{})
	assertPluginRuntimeRoundTrip(t, &bufpluginconfig.RuntimeConfig{
		Go: &bufpluginconfig.GoRuntimeConfig{
			MinVersion: "1.18",
			Deps: []*bufpluginconfig.GoRuntimeDependencyConfig{
				{
					Module:  "github.com/bufbuild/connect-go",
					Version: "v0.1.1",
				},
			},
		},
	})
	assertPluginRuntimeRoundTrip(t, &bufpluginconfig.RuntimeConfig{
		NPM: &bufpluginconfig.NPMRuntimeConfig{
			Deps: []*bufpluginconfig.NPMRuntimeDependencyConfig{
				{
					Package: "@bufbuild/protobuf",
					Version: "^0.0.4",
				},
			},
		},
	})
}

func assertPluginRuntimeRoundTrip(t testing.TB, config *bufpluginconfig.RuntimeConfig) {
	assert.Equal(t, config, ProtoRuntimeConfigToPluginRuntime(PluginRuntimeToProtoRuntimeConfig(config)))
}

func TestPluginOptionsRoundTrip(t *testing.T) {
	assertPluginOptionsRoundTrip(t, nil)
	assertPluginOptionsRoundTrip(t, map[string]string{})
	assertPluginOptionsRoundTrip(t, map[string]string{
		"option-1":          "value-1",
		"option-2":          "value-2",
		"option-no-value-3": "",
	})
}

func assertPluginOptionsRoundTrip(t testing.TB, options map[string]string) {
	assert.Equal(t, options, OptionsSliceToPluginOptions(PluginOptionsToOptionsSlice(options)))
}

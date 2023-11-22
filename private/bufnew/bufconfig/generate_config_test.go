// Copyright 2020-2023 Buf Technologies, Inc.
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

package bufconfig

import (
	"testing"

	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/stretchr/testify/require"
)

func TestParseConfigFromExternalV1(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		description    string
		externalConfig externalBufGenYAMLFileV1
		expectedConfig GenerateConfig
	}{
		{
			description: "name_local_plugin_strategy",
			externalConfig: externalBufGenYAMLFileV1{
				Version: "v1",
				Plugins: []externalGeneratePluginConfigV1{
					{
						Name:     "java",
						Out:      "java/out",
						Opt:      "a=b,c",
						Strategy: "all",
					},
				},
			},
			expectedConfig: &generateConfig{
				pluginConfigs: []GeneratePluginConfig{
					&pluginConfig{
						pluginConfigType: PluginConfigTypeLocal,
						name:             "java",
						out:              "java/out",
						opt:              "a=b,c",
						strategy:         GenerateStrategyAll,
					},
				},
			},
		},
		{
			description: "name_local_plugin_strategy",
			externalConfig: externalBufGenYAMLFileV1{
				Version: "v1",
				Plugins: []externalGeneratePluginConfigV1{
					{
						Plugin:   "java",
						Out:      "java/out",
						Opt:      "a=b,c",
						Strategy: "all",
					},
				},
			},
			expectedConfig: &generateConfig{
				pluginConfigs: []GeneratePluginConfig{
					&pluginConfig{
						pluginConfigType: PluginConfigTypeLocal,
						name:             "java",
						out:              "java/out",
						opt:              "a=b,c",
						strategy:         GenerateStrategyAll,
					},
				},
			},
		},
		{
			description: "name_binary_plugin_with_string_slice_path_and_opts",
			externalConfig: externalBufGenYAMLFileV1{
				Version: "v1",
				Plugins: []externalGeneratePluginConfigV1{
					{
						Name:     "go",
						Out:      "go/out",
						Path:     slicesext.Map([]string{"go", "run", "goplugin"}, func(s string) interface{} { return s }),
						Opt:      slicesext.Map([]string{"a=b", "c"}, func(s string) interface{} { return s }),
						Strategy: "directory",
					},
				},
			},
			expectedConfig: &generateConfig{
				pluginConfigs: []GeneratePluginConfig{
					&pluginConfig{
						pluginConfigType: PluginConfigTypeBinary,
						name:             "go",
						out:              "go/out",
						path:             []string{"go", "run", "goplugin"},
						opt:              "a=b,c",
						strategy:         GenerateStrategyDirectory,
					},
				},
			},
		},
		{
			description: "plugin_binary_plugin_with_string_slice_path_and_opts",
			externalConfig: externalBufGenYAMLFileV1{
				Version: "v1",
				Plugins: []externalGeneratePluginConfigV1{
					{
						Plugin:   "go",
						Out:      "go/out",
						Path:     slicesext.Map([]string{"go", "run", "goplugin"}, func(s string) interface{} { return s }),
						Opt:      slicesext.Map([]string{"a=b", "c"}, func(s string) interface{} { return s }),
						Strategy: "directory",
					},
				},
			},
			expectedConfig: &generateConfig{
				pluginConfigs: []GeneratePluginConfig{
					&pluginConfig{
						pluginConfigType: PluginConfigTypeBinary,
						name:             "go",
						out:              "go/out",
						path:             []string{"go", "run", "goplugin"},
						opt:              "a=b,c",
						strategy:         GenerateStrategyDirectory,
					},
				},
			},
		},
		{
			description: "name_binary_plugin_with_string_path",
			externalConfig: externalBufGenYAMLFileV1{
				Version: "v1",
				Plugins: []externalGeneratePluginConfigV1{
					{
						Name: "go2",
						Out:  "go2/out",
						Path: "protoc-gen-go",
					},
				},
			},
			expectedConfig: &generateConfig{
				pluginConfigs: []GeneratePluginConfig{
					&pluginConfig{
						pluginConfigType: PluginConfigTypeBinary,
						name:             "go2",
						out:              "go2/out",
						path:             []string{"protoc-gen-go"},
						strategy:         GenerateStrategyDirectory,
					},
				},
			},
		},
		{
			description: "plugin_binary_plugin_with_string_path",
			externalConfig: externalBufGenYAMLFileV1{
				Version: "v1",
				Plugins: []externalGeneratePluginConfigV1{
					{
						Plugin: "go2",
						Out:    "go2/out",
						Path:   "protoc-gen-go",
					},
				},
			},
			expectedConfig: &generateConfig{
				pluginConfigs: []GeneratePluginConfig{
					&pluginConfig{
						pluginConfigType: PluginConfigTypeBinary,
						name:             "go2",
						out:              "go2/out",
						path:             []string{"protoc-gen-go"},
						strategy:         GenerateStrategyDirectory,
					},
				},
			},
		},
		{
			description: "name_protoc_builtin_plugin",
			externalConfig: externalBufGenYAMLFileV1{
				Version: "v1",
				Plugins: []externalGeneratePluginConfigV1{
					{
						Name:       "cpp",
						Out:        "cpp/out",
						ProtocPath: "path/to/protoc",
					},
				},
			},
			expectedConfig: &generateConfig{
				pluginConfigs: []GeneratePluginConfig{
					&pluginConfig{
						pluginConfigType: PluginConfigTypeProtocBuiltin,
						name:             "cpp",
						out:              "cpp/out",
						protocPath:       "path/to/protoc",
						strategy:         GenerateStrategyDirectory,
					},
				},
			},
		},
		{
			description: "plugin_protoc_builtin_plugin",
			externalConfig: externalBufGenYAMLFileV1{
				Version: "v1",
				Plugins: []externalGeneratePluginConfigV1{
					{
						Plugin:     "cpp",
						Out:        "cpp/out",
						ProtocPath: "path/to/protoc",
					},
				},
			},
			expectedConfig: &generateConfig{
				pluginConfigs: []GeneratePluginConfig{
					&pluginConfig{
						pluginConfigType: PluginConfigTypeProtocBuiltin,
						name:             "cpp",
						out:              "cpp/out",
						protocPath:       "path/to/protoc",
						strategy:         GenerateStrategyDirectory,
					},
				},
			},
		},
		{
			description: "remote_plugin_reference",
			externalConfig: externalBufGenYAMLFileV1{
				Version: "v1",
				Plugins: []externalGeneratePluginConfigV1{
					{
						Plugin:   "buf.build/protocolbuffers/go:v1.31.0",
						Out:      "go/out",
						Revision: 1,
					},
				},
			},
			expectedConfig: &generateConfig{
				pluginConfigs: []GeneratePluginConfig{
					&pluginConfig{
						pluginConfigType: PluginConfigTypeRemote,
						remoteHost:       "buf.build",
						revision:         1,
						name:             "buf.build/protocolbuffers/go:v1.31.0",
						out:              "go/out",
					},
				},
			},
		},
		{
			description: "remote_plugin_identity",
			externalConfig: externalBufGenYAMLFileV1{
				Version: "v1",
				Plugins: []externalGeneratePluginConfigV1{
					{
						Plugin:   "buf.build/protocolbuffers/go",
						Out:      "go/out",
						Revision: 1,
					},
				},
			},
			expectedConfig: &generateConfig{
				pluginConfigs: []GeneratePluginConfig{
					&pluginConfig{
						pluginConfigType: PluginConfigTypeRemote,
						remoteHost:       "buf.build",
						revision:         1,
						name:             "buf.build/protocolbuffers/go",
						out:              "go/out",
					},
				},
			},
		},
	}
	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()
			parsedConfig, err := newGenerateConfigFromExternalFileV1(testcase.externalConfig)
			require.NoError(t, err)
			require.Equal(t, testcase.expectedConfig, parsedConfig)
		})
	}
}

func TestParseConfigFromExternalV1Fail(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		description    string
		externalConfig externalBufGenYAMLFileV1
	}{
		{
			description: "empty_out",
			externalConfig: externalBufGenYAMLFileV1{
				Version: "v1",
				Plugins: []externalGeneratePluginConfigV1{
					{
						Name:     "java",
						Out:      "",
						Opt:      "a=b,c",
						Strategy: "all",
					},
				},
			},
		},
		{
			description: "both_plugin_and_name",
			externalConfig: externalBufGenYAMLFileV1{
				Version: "v1",
				Plugins: []externalGeneratePluginConfigV1{
					{
						Plugin: "java",
						Name:   "go",
						Out:    "java/out",
					},
				},
			},
		},
		{
			description: "neither_plugin_nor_name",
			externalConfig: externalBufGenYAMLFileV1{
				Version: "v1",
				Plugins: []externalGeneratePluginConfigV1{
					{
						Out: "java/out",
					},
				},
			},
		},
		{
			description: "no_plugins",
			externalConfig: externalBufGenYAMLFileV1{
				Version: "v1",
				Plugins: nil,
			},
		},
		{
			description: "invalid_strategy",
			externalConfig: externalBufGenYAMLFileV1{
				Version: "v1",
				Plugins: []externalGeneratePluginConfigV1{
					{
						Plugin:   "go",
						Out:      "go/out",
						Strategy: "invalid",
					},
				},
			},
		},
		{
			description: "deprecated_alpha_plugin",
			externalConfig: externalBufGenYAMLFileV1{
				Version: "v1",
				Plugins: []externalGeneratePluginConfigV1{
					{
						Remote: "buf.build/bufbuild/plugins/connect-go:v1.3.1-1",
						Out:    "connect/out",
					},
				},
			},
		},
		{
			description: "plugin_with_deprecated_alpha_plugin_name",
			externalConfig: externalBufGenYAMLFileV1{
				Version: "v1",
				Plugins: []externalGeneratePluginConfigV1{
					{
						Plugin: "buf.build/bufbuild/plugins/connect-go:v1.3.1-1",
						Out:    "connect/out",
					},
				},
			},
		},
	}
	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()
			_, err := newGenerateConfigFromExternalFileV1(testcase.externalConfig)
			require.Error(t, err)
		})
	}
}

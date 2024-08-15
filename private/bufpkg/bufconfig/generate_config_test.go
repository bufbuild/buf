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

package bufconfig

import (
	"testing"

	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
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
				generateManagedConfig: &generateManagedConfig{enabled: false},
				generatePluginConfigs: []GeneratePluginConfig{
					&generatePluginConfig{
						generatePluginConfigType: GeneratePluginConfigTypeLocalOrProtocBuiltin,
						name:                     "java",
						out:                      "java/out",
						// one string because it's one string in the config
						opts:     []string{"a=b,c"},
						strategy: toPointer(GenerateStrategyAll),
					},
				},
			},
		},
		{
			description: "plugin_local_plugin_strategy",
			externalConfig: externalBufGenYAMLFileV1{
				Version: "v1",
				Plugins: []externalGeneratePluginConfigV1{
					{
						Plugin:   "java",
						Out:      "java/out",
						Opt:      "a",
						Strategy: "all",
					},
				},
			},
			expectedConfig: &generateConfig{
				generateManagedConfig: &generateManagedConfig{enabled: false},
				generatePluginConfigs: []GeneratePluginConfig{
					&generatePluginConfig{
						generatePluginConfigType: GeneratePluginConfigTypeLocalOrProtocBuiltin,
						name:                     "java",
						out:                      "java/out",
						opts:                     []string{"a"},
						strategy:                 toPointer(GenerateStrategyAll),
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
				generateManagedConfig: &generateManagedConfig{enabled: false},
				generatePluginConfigs: []GeneratePluginConfig{
					&generatePluginConfig{
						generatePluginConfigType: GeneratePluginConfigTypeLocal,
						name:                     "go",
						out:                      "go/out",
						path:                     []string{"go", "run", "goplugin"},
						opts:                     []string{"a=b", "c"},
						strategy:                 toPointer(GenerateStrategyDirectory),
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
				generateManagedConfig: &generateManagedConfig{enabled: false},
				generatePluginConfigs: []GeneratePluginConfig{
					&generatePluginConfig{
						generatePluginConfigType: GeneratePluginConfigTypeLocal,
						name:                     "go",
						out:                      "go/out",
						path:                     []string{"go", "run", "goplugin"},
						opts:                     []string{"a=b", "c"},
						strategy:                 toPointer(GenerateStrategyDirectory),
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
				generateManagedConfig: &generateManagedConfig{enabled: false},
				generatePluginConfigs: []GeneratePluginConfig{
					&generatePluginConfig{
						generatePluginConfigType: GeneratePluginConfigTypeLocal,
						name:                     "go2",
						out:                      "go2/out",
						path:                     []string{"protoc-gen-go"},
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
				generateManagedConfig: &generateManagedConfig{enabled: false},
				generatePluginConfigs: []GeneratePluginConfig{
					&generatePluginConfig{
						generatePluginConfigType: GeneratePluginConfigTypeLocal,
						name:                     "go2",
						out:                      "go2/out",
						path:                     []string{"protoc-gen-go"},
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
				generateManagedConfig: &generateManagedConfig{enabled: false},
				generatePluginConfigs: []GeneratePluginConfig{
					&generatePluginConfig{
						generatePluginConfigType: GeneratePluginConfigTypeProtocBuiltin,
						name:                     "cpp",
						out:                      "cpp/out",
						protocPath:               []string{"path/to/protoc"},
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
						ProtocPath: []any{"path/to/protoc", "--experimental_editions"},
					},
				},
			},
			expectedConfig: &generateConfig{
				generateManagedConfig: &generateManagedConfig{enabled: false},
				generatePluginConfigs: []GeneratePluginConfig{
					&generatePluginConfig{
						generatePluginConfigType: GeneratePluginConfigTypeProtocBuiltin,
						name:                     "cpp",
						out:                      "cpp/out",
						protocPath:               []string{"path/to/protoc", "--experimental_editions"},
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
				generateManagedConfig: &generateManagedConfig{enabled: false},
				generatePluginConfigs: []GeneratePluginConfig{
					&generatePluginConfig{
						generatePluginConfigType: GeneratePluginConfigTypeRemote,
						remoteHost:               "buf.build",
						revision:                 1,
						name:                     "buf.build/protocolbuffers/go:v1.31.0",
						out:                      "go/out",
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
				generateManagedConfig: &generateManagedConfig{enabled: false},
				generatePluginConfigs: []GeneratePluginConfig{
					&generatePluginConfig{
						generatePluginConfigType: GeneratePluginConfigTypeRemote,
						remoteHost:               "buf.build",
						revision:                 1,
						name:                     "buf.build/protocolbuffers/go",
						out:                      "go/out",
					},
				},
			},
		},
		{
			description: "managed_mode_empty",
			externalConfig: externalBufGenYAMLFileV1{
				Version: "v1",
				Plugins: []externalGeneratePluginConfigV1{
					{
						Plugin: "go",
						Out:    "go/out",
					},
				},
				Managed: externalGenerateManagedConfigV1{
					Enabled: true,
				},
			},
			expectedConfig: &generateConfig{
				generatePluginConfigs: []GeneratePluginConfig{
					&generatePluginConfig{
						generatePluginConfigType: GeneratePluginConfigTypeLocalOrProtocBuiltin,
						name:                     "go",
						out:                      "go/out",
					},
				},
				generateManagedConfig: &generateManagedConfig{
					enabled: true,
				},
			},
		},
		{
			description: "managed_mode_bools_and_java_package",
			externalConfig: externalBufGenYAMLFileV1{
				Version: "v1",
				Plugins: []externalGeneratePluginConfigV1{
					{
						Plugin: "go",
						Out:    "go/out",
					},
				},
				Managed: externalGenerateManagedConfigV1{
					Enabled:             true,
					CcEnableArenas:      proto.Bool(true),
					JavaMultipleFiles:   proto.Bool(true),
					JavaStringCheckUtf8: proto.Bool(true),
					JavaPackagePrefix: externalJavaPackagePrefixConfigV1{
						Default: "foo",
						Except:  []string{"buf.build/acme/foo", "buf.build/acme/bar"},
						Override: map[string]string{
							"buf.build/acme/weatherapis": "weather",
							"buf.build/acme/paymentapis": "payment",
							"buf.build/acme/petapis":     "pet",
						},
					},
				},
			},
			expectedConfig: &generateConfig{
				generatePluginConfigs: []GeneratePluginConfig{
					&generatePluginConfig{
						generatePluginConfigType: GeneratePluginConfigTypeLocalOrProtocBuiltin,
						name:                     "go",
						out:                      "go/out",
					},
				},
				generateManagedConfig: &generateManagedConfig{
					enabled: true,
					disables: []ManagedDisableRule{
						&managedDisableRule{
							fileOption:     FileOptionJavaPackage,
							moduleFullName: "buf.build/acme/foo",
						},
						&managedDisableRule{
							fileOption:     FileOptionJavaPackage,
							moduleFullName: "buf.build/acme/bar",
						},
					},
					overrides: []ManagedOverrideRule{
						&managedOverrideRule{
							fileOption: FileOptionCcEnableArenas,
							value:      true,
						},
						&managedOverrideRule{
							fileOption: FileOptionJavaMultipleFiles,
							value:      true,
						},
						&managedOverrideRule{
							fileOption: FileOptionJavaStringCheckUtf8,
							value:      true,
						},
						&managedOverrideRule{
							fileOption: FileOptionJavaPackagePrefix,
							value:      "foo",
						},
						// the next three rules are ordered by their module names
						&managedOverrideRule{
							fileOption:     FileOptionJavaPackagePrefix,
							moduleFullName: "buf.build/acme/paymentapis",
							value:          "payment",
						},
						&managedOverrideRule{
							fileOption:     FileOptionJavaPackagePrefix,
							moduleFullName: "buf.build/acme/petapis",
							value:          "pet",
						},
						&managedOverrideRule{
							fileOption:     FileOptionJavaPackagePrefix,
							moduleFullName: "buf.build/acme/weatherapis",
							value:          "weather",
						},
					},
				},
			},
		},
		{
			description: "managed_mode_optimize_for",
			externalConfig: externalBufGenYAMLFileV1{
				Version: "v1",
				Plugins: []externalGeneratePluginConfigV1{
					{
						Plugin: "go",
						Out:    "go/out",
					},
				},
				Managed: externalGenerateManagedConfigV1{
					Enabled: true,
					OptimizeFor: externalOptimizeForConfigV1{
						Default: "LITE_RUNTIME",
						Except:  []string{"buf.build/acme/foo"},
						Override: map[string]string{
							"buf.build/acme/petapis":     "CODE_SIZE",
							"buf.build/acme/paymentapis": "SPEED",
						},
					},
				},
			},
			expectedConfig: &generateConfig{
				generatePluginConfigs: []GeneratePluginConfig{
					&generatePluginConfig{
						generatePluginConfigType: GeneratePluginConfigTypeLocalOrProtocBuiltin,
						name:                     "go",
						out:                      "go/out",
					},
				},
				generateManagedConfig: &generateManagedConfig{
					enabled: true,
					disables: []ManagedDisableRule{
						&managedDisableRule{
							fileOption:     FileOptionOptimizeFor,
							moduleFullName: "buf.build/acme/foo",
						},
					},
					overrides: []ManagedOverrideRule{
						&managedOverrideRule{
							fileOption: FileOptionOptimizeFor,
							value:      descriptorpb.FileOptions_LITE_RUNTIME,
						},
						&managedOverrideRule{
							fileOption:     FileOptionOptimizeFor,
							moduleFullName: "buf.build/acme/paymentapis",
							value:          descriptorpb.FileOptions_SPEED,
						},
						&managedOverrideRule{
							fileOption:     FileOptionOptimizeFor,
							moduleFullName: "buf.build/acme/petapis",
							value:          descriptorpb.FileOptions_CODE_SIZE,
						},
					},
				},
			},
		},
		{
			description: "managed_mode_go_package_prefix",
			externalConfig: externalBufGenYAMLFileV1{
				Version: "v1",
				Plugins: []externalGeneratePluginConfigV1{
					{
						Plugin: "go",
						Out:    "go/out",
					},
				},
				Managed: externalGenerateManagedConfigV1{
					Enabled: true,
					GoPackagePrefix: externalGoPackagePrefixConfigV1{
						Default: "foo",
						Except:  []string{"buf.build/acme/foo"},
						Override: map[string]string{
							"buf.build/acme/petapis": "pet",
						},
					},
				},
			},
			expectedConfig: &generateConfig{
				generatePluginConfigs: []GeneratePluginConfig{
					&generatePluginConfig{
						generatePluginConfigType: GeneratePluginConfigTypeLocalOrProtocBuiltin,
						name:                     "go",
						out:                      "go/out",
					},
				},
				generateManagedConfig: &generateManagedConfig{
					enabled: true,
					disables: []ManagedDisableRule{
						&managedDisableRule{
							fileOption:     FileOptionGoPackage,
							moduleFullName: "buf.build/acme/foo",
						},
					},
					overrides: []ManagedOverrideRule{
						&managedOverrideRule{
							fileOption: FileOptionGoPackagePrefix,
							value:      "foo",
						},
						&managedOverrideRule{
							fileOption:     FileOptionGoPackagePrefix,
							moduleFullName: "buf.build/acme/petapis",
							value:          "pet",
						},
					},
				},
			},
		},
		{
			description: "managed_mode_objc_class_prefix",
			externalConfig: externalBufGenYAMLFileV1{
				Version: "v1",
				Plugins: []externalGeneratePluginConfigV1{
					{
						Plugin: "go",
						Out:    "go/out",
					},
				},
				Managed: externalGenerateManagedConfigV1{
					Enabled: true,
					ObjcClassPrefix: externalObjcClassPrefixConfigV1{
						Default: "foo",
						Except:  []string{"buf.build/acme/foo"},
						Override: map[string]string{
							"buf.build/acme/petapis": "pet",
						},
					},
				},
			},
			expectedConfig: &generateConfig{
				generatePluginConfigs: []GeneratePluginConfig{
					&generatePluginConfig{
						generatePluginConfigType: GeneratePluginConfigTypeLocalOrProtocBuiltin,
						name:                     "go",
						out:                      "go/out",
					},
				},
				generateManagedConfig: &generateManagedConfig{
					enabled: true,
					disables: []ManagedDisableRule{
						&managedDisableRule{
							fileOption:     FileOptionObjcClassPrefix,
							moduleFullName: "buf.build/acme/foo",
						},
					},
					overrides: []ManagedOverrideRule{
						&managedOverrideRule{
							fileOption: FileOptionObjcClassPrefix,
							value:      "foo",
						},
						&managedOverrideRule{
							fileOption:     FileOptionObjcClassPrefix,
							moduleFullName: "buf.build/acme/petapis",
							value:          "pet",
						},
					},
				},
			},
		},
		{
			description: "managed_mode_ruby_package",
			externalConfig: externalBufGenYAMLFileV1{
				Version: "v1",
				Plugins: []externalGeneratePluginConfigV1{
					{
						Plugin: "go",
						Out:    "go/out",
					},
				},
				Managed: externalGenerateManagedConfigV1{
					Enabled: true,
					RubyPackage: externalRubyPackageConfigV1{
						Except: []string{"buf.build/acme/foo"},
						Override: map[string]string{
							"buf.build/acme/petapis": "pet",
						},
					},
				},
			},
			expectedConfig: &generateConfig{
				generatePluginConfigs: []GeneratePluginConfig{
					&generatePluginConfig{
						generatePluginConfigType: GeneratePluginConfigTypeLocalOrProtocBuiltin,
						name:                     "go",
						out:                      "go/out",
					},
				},
				generateManagedConfig: &generateManagedConfig{
					enabled: true,
					disables: []ManagedDisableRule{
						&managedDisableRule{
							fileOption:     FileOptionRubyPackage,
							moduleFullName: "buf.build/acme/foo",
						},
					},
					overrides: []ManagedOverrideRule{
						&managedOverrideRule{
							fileOption:     FileOptionRubyPackage,
							moduleFullName: "buf.build/acme/petapis",
							value:          "pet",
						},
					},
				},
			},
		},
		{
			description: "managed_mode_per_file_override",
			externalConfig: externalBufGenYAMLFileV1{
				Version: "v1",
				Plugins: []externalGeneratePluginConfigV1{
					{
						Plugin: "go",
						Out:    "go/out",
					},
				},
				Managed: externalGenerateManagedConfigV1{
					Enabled: true,
					Override: map[string]map[string]string{
						"JAVA_PACKAGE": {
							"foo.proto": "foo",
							"bar.proto": "bar",
							"baz.proto": "baz",
						},
						"CC_ENABLE_ARENAS": {
							"foo.proto": "false",
							"baz.proto": "true",
						},
						"OPTIMIZE_FOR": {
							"dir/baz.proto": "SPEED",
							"dir/foo.proto": "CODE_SIZE",
							"dir/bar.proto": "LITE_RUNTIME",
						},
					},
				},
			},
			expectedConfig: &generateConfig{
				generatePluginConfigs: []GeneratePluginConfig{
					&generatePluginConfig{
						generatePluginConfigType: GeneratePluginConfigTypeLocalOrProtocBuiltin,
						name:                     "go",
						out:                      "go/out",
					},
				},
				generateManagedConfig: &generateManagedConfig{
					enabled: true,
					overrides: []ManagedOverrideRule{
						// ordered by file option names and then by file paths
						&managedOverrideRule{
							fileOption: FileOptionCcEnableArenas,
							path:       "baz.proto",
							value:      true,
						},
						&managedOverrideRule{
							fileOption: FileOptionCcEnableArenas,
							path:       "foo.proto",
							value:      false,
						},
						&managedOverrideRule{
							fileOption: FileOptionJavaPackage,
							path:       "bar.proto",
							value:      "bar",
						},
						&managedOverrideRule{
							fileOption: FileOptionJavaPackage,
							path:       "baz.proto",
							value:      "baz",
						},
						&managedOverrideRule{
							fileOption: FileOptionJavaPackage,
							path:       "foo.proto",
							value:      "foo",
						},
						&managedOverrideRule{
							fileOption: FileOptionOptimizeFor,
							path:       "dir/bar.proto",
							value:      descriptorpb.FileOptions_LITE_RUNTIME,
						},
						&managedOverrideRule{
							fileOption: FileOptionOptimizeFor,
							path:       "dir/baz.proto",
							value:      descriptorpb.FileOptions_SPEED,
						},
						&managedOverrideRule{
							fileOption: FileOptionOptimizeFor,
							path:       "dir/foo.proto",
							value:      descriptorpb.FileOptions_CODE_SIZE,
						},
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

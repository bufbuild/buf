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

package bufplugin

import (
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginconfig"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginToProtoPluginRegistryType(t *testing.T) {
	t.Parallel()
	assertPluginToPluginRegistryType(t, nil, registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_UNSPECIFIED)
	assertPluginToPluginRegistryType(t, &bufpluginconfig.RegistryConfig{Go: &bufpluginconfig.GoRegistryConfig{}}, registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_GO)
	assertPluginToPluginRegistryType(t, &bufpluginconfig.RegistryConfig{NPM: &bufpluginconfig.NPMRegistryConfig{}}, registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_NPM)
	assertPluginToPluginRegistryType(t, &bufpluginconfig.RegistryConfig{Maven: &bufpluginconfig.MavenRegistryConfig{}}, registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_MAVEN)
	assertPluginToPluginRegistryType(t, &bufpluginconfig.RegistryConfig{Swift: &bufpluginconfig.SwiftRegistryConfig{}}, registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_SWIFT)
}

func assertPluginToPluginRegistryType(t testing.TB, config *bufpluginconfig.RegistryConfig, registryType registryv1alpha1.PluginRegistryType) {
	plugin, err := NewPlugin("v1.0.0", nil, config, "sha256:digest", "", "")
	require.Nil(t, err)
	assert.Equal(t, registryType, PluginToProtoPluginRegistryType(plugin))
}

func TestPluginRegistryRoundTrip(t *testing.T) {
	t.Parallel()
	assertPluginRegistryRoundTrip(t, nil)
	assertPluginRegistryRoundTrip(t, &bufpluginconfig.RegistryConfig{})
	assertPluginRegistryRoundTrip(t, &bufpluginconfig.RegistryConfig{
		Go: &bufpluginconfig.GoRegistryConfig{},
	})
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
			ImportStyle: "commonjs",
		},
	})
	assertPluginRegistryRoundTrip(t, &bufpluginconfig.RegistryConfig{
		NPM: &bufpluginconfig.NPMRegistryConfig{
			ImportStyle:             "module",
			RewriteImportPathSuffix: "connectweb.js",
			Deps: []*bufpluginconfig.NPMRegistryDependencyConfig{
				{
					Package: "@bufbuild/protobuf",
					Version: "^0.0.4",
				},
			},
		},
	})
	assertPluginRegistryRoundTrip(t, &bufpluginconfig.RegistryConfig{
		Go: &bufpluginconfig.GoRegistryConfig{
			MinVersion: "1.18",
			Deps: []*bufpluginconfig.GoRegistryDependencyConfig{
				{
					Module:  "github.com/bufbuild/connect-go",
					Version: "v0.4.0",
				},
			},
		},
		Options: map[string]string{
			"separate_package": "true",
		},
	})
	assertPluginRegistryRoundTrip(t, &bufpluginconfig.RegistryConfig{
		Maven: &bufpluginconfig.MavenRegistryConfig{},
	})
	assertPluginRegistryRoundTrip(t, &bufpluginconfig.RegistryConfig{
		Maven: &bufpluginconfig.MavenRegistryConfig{
			Compiler: bufpluginconfig.MavenCompilerConfig{
				Java: bufpluginconfig.MavenCompilerJavaConfig{
					Encoding: "UTF-8",
					Release:  7,
					Source:   8,
					Target:   9,
				},
				Kotlin: bufpluginconfig.MavenCompilerKotlinConfig{
					APIVersion:      "7",
					JVMTarget:       "8",
					LanguageVersion: "9",
					Version:         "1.8.0",
				},
			},
			Deps: []bufpluginconfig.MavenDependencyConfig{
				{
					GroupID:    "io.grpc",
					ArtifactID: "grpc-core",
					Version:    "1.52.1",
				},
				{
					GroupID:    "io.grpc",
					ArtifactID: "grpc-protobuf",
					Version:    "1.52.1",
				},
				{
					GroupID:    "io.grpc",
					ArtifactID: "protoc-gen-grpc-java",
					Version:    "1.52.1",
					Classifier: "linux-x86_64",
					Extension:  "exe",
				},
			},
			AdditionalRuntimes: []bufpluginconfig.MavenRuntimeConfig{
				{
					Name: "lite",
					Deps: []bufpluginconfig.MavenDependencyConfig{
						{
							GroupID:    "io.grpc",
							ArtifactID: "grpc-core",
							Version:    "1.52.1",
						},
						{
							GroupID:    "io.grpc",
							ArtifactID: "grpc-protobuflite",
							Version:    "1.52.1",
						},
						{
							GroupID:    "io.grpc",
							ArtifactID: "protoc-gen-grpc-java",
							Version:    "1.52.1",
							Classifier: "linux-x86_64",
							Extension:  "exe",
						},
					},
					Options: []string{"lite"},
				},
			},
		},
	})
	assertPluginRegistryRoundTrip(t, &bufpluginconfig.RegistryConfig{
		Swift: &bufpluginconfig.SwiftRegistryConfig{},
	})
	assertPluginRegistryRoundTrip(t, &bufpluginconfig.RegistryConfig{
		Swift: &bufpluginconfig.SwiftRegistryConfig{
			Dependencies: []bufpluginconfig.SwiftRegistryDependencyConfig{
				{
					Source:        "https://github.com/apple/swift-protobuf.git",
					Package:       "swift-protobuf",
					Version:       "1.12.0",
					Products:      []string{"SwiftProtobuf"},
					SwiftVersions: []string{".v5"},
					Platforms: bufpluginconfig.SwiftRegistryDependencyPlatformConfig{
						MacOS:   "v10_15",
						IOS:     "v10_15",
						TVOS:    "v10_15",
						WatchOS: "v10_15",
					},
				},
			},
		},
	})
}

func assertPluginRegistryRoundTrip(t testing.TB, config *bufpluginconfig.RegistryConfig) {
	protoRegistryConfig, err := PluginRegistryToProtoRegistryConfig(config)
	require.NoError(t, err)
	registryConfig, err := ProtoRegistryConfigToPluginRegistry(protoRegistryConfig)
	require.NoError(t, err)
	assert.Equal(t, config, registryConfig)
}

func TestLanguagesToProtoLanguages(t *testing.T) {
	t.Parallel()
	protoLanguages, err := OutputLanguagesToProtoLanguages([]string{"go"})
	require.NoError(t, err)
	assert.Equal(t,
		[]registryv1alpha1.PluginLanguage{
			registryv1alpha1.PluginLanguage_PLUGIN_LANGUAGE_GO,
		},
		protoLanguages,
	)
	protoLanguages, err = OutputLanguagesToProtoLanguages([]string{"typescript", "javascript"})
	require.NoError(t, err)
	assert.Equal(t,
		[]registryv1alpha1.PluginLanguage{
			registryv1alpha1.PluginLanguage_PLUGIN_LANGUAGE_JAVASCRIPT,
			registryv1alpha1.PluginLanguage_PLUGIN_LANGUAGE_TYPESCRIPT,
		},
		protoLanguages,
	)
	protoLanguages, err = OutputLanguagesToProtoLanguages([]string{"java", "kotlin"})
	require.NoError(t, err)
	assert.Equal(t,
		[]registryv1alpha1.PluginLanguage{
			registryv1alpha1.PluginLanguage_PLUGIN_LANGUAGE_JAVA,
			registryv1alpha1.PluginLanguage_PLUGIN_LANGUAGE_KOTLIN,
		},
		protoLanguages,
	)
	_, err = OutputLanguagesToProtoLanguages([]string{"unknown_language", "another_unknown_language"})
	require.Error(t, err)
	protoLanguages, err = OutputLanguagesToProtoLanguages(nil)
	require.NoError(t, err)
	assert.Equal(t, 0, len(protoLanguages))
}

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
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginconfig"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginToProtoPluginRegistryType(t *testing.T) {
	t.Parallel()
	assertPluginToPluginRegistryType(t, nil, registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_UNSPECIFIED)
	assertPluginToPluginRegistryType(t, &bufremotepluginconfig.RegistryConfig{Go: &bufremotepluginconfig.GoRegistryConfig{}}, registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_GO)
	assertPluginToPluginRegistryType(t, &bufremotepluginconfig.RegistryConfig{NPM: &bufremotepluginconfig.NPMRegistryConfig{}}, registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_NPM)
	assertPluginToPluginRegistryType(t, &bufremotepluginconfig.RegistryConfig{Maven: &bufremotepluginconfig.MavenRegistryConfig{}}, registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_MAVEN)
	assertPluginToPluginRegistryType(t, &bufremotepluginconfig.RegistryConfig{Swift: &bufremotepluginconfig.SwiftRegistryConfig{}}, registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_SWIFT)
	assertPluginToPluginRegistryType(t, &bufremotepluginconfig.RegistryConfig{Python: &bufremotepluginconfig.PythonRegistryConfig{}}, registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_PYTHON)
	assertPluginToPluginRegistryType(t, &bufremotepluginconfig.RegistryConfig{Cargo: &bufremotepluginconfig.CargoRegistryConfig{}}, registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_CARGO)
	assertPluginToPluginRegistryType(t, &bufremotepluginconfig.RegistryConfig{Nuget: &bufremotepluginconfig.NugetRegistryConfig{}}, registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_NUGET)
	assertPluginToPluginRegistryType(t, &bufremotepluginconfig.RegistryConfig{Cmake: &bufremotepluginconfig.CmakeRegistryConfig{}}, registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_CMAKE)
}

func assertPluginToPluginRegistryType(t testing.TB, config *bufremotepluginconfig.RegistryConfig, registryType registryv1alpha1.PluginRegistryType) {
	plugin, err := NewPlugin("v1.0.0", nil, config, "sha256:digest", "", "")
	require.Nil(t, err)
	assert.Equal(t, registryType, PluginToProtoPluginRegistryType(plugin))
}

func TestPluginRegistryRoundTrip(t *testing.T) {
	t.Parallel()
	assertPluginRegistryRoundTrip(t, nil)
	assertPluginRegistryRoundTrip(t, &bufremotepluginconfig.RegistryConfig{})
	assertPluginRegistryRoundTrip(t, &bufremotepluginconfig.RegistryConfig{
		Go: &bufremotepluginconfig.GoRegistryConfig{},
	})
	assertPluginRegistryRoundTrip(t, &bufremotepluginconfig.RegistryConfig{
		Go: &bufremotepluginconfig.GoRegistryConfig{
			MinVersion: "1.18",
			Deps: []*bufremotepluginconfig.GoRegistryDependencyConfig{
				{
					Module:  "connectrpc.com/connect",
					Version: "v0.1.1",
				},
			},
		},
	})
	assertPluginRegistryRoundTrip(t, &bufremotepluginconfig.RegistryConfig{
		NPM: &bufremotepluginconfig.NPMRegistryConfig{
			ImportStyle: "commonjs",
		},
	})
	assertPluginRegistryRoundTrip(t, &bufremotepluginconfig.RegistryConfig{
		NPM: &bufremotepluginconfig.NPMRegistryConfig{
			ImportStyle:             "module",
			RewriteImportPathSuffix: "connectweb.js",
			Deps: []*bufremotepluginconfig.NPMRegistryDependencyConfig{
				{
					Package: "@bufbuild/protobuf",
					Version: "^0.0.4",
				},
			},
		},
	})
	assertPluginRegistryRoundTrip(t, &bufremotepluginconfig.RegistryConfig{
		Go: &bufremotepluginconfig.GoRegistryConfig{
			MinVersion: "1.18",
			Deps: []*bufremotepluginconfig.GoRegistryDependencyConfig{
				{
					Module:  "connectrpc.com/connect",
					Version: "v0.4.0",
				},
			},
		},
		Options: map[string]string{
			"separate_package": "true",
		},
	})
	assertPluginRegistryRoundTrip(t, &bufremotepluginconfig.RegistryConfig{
		Maven: &bufremotepluginconfig.MavenRegistryConfig{},
	})
	assertPluginRegistryRoundTrip(t, &bufremotepluginconfig.RegistryConfig{
		Maven: &bufremotepluginconfig.MavenRegistryConfig{
			Compiler: bufremotepluginconfig.MavenCompilerConfig{
				Java: bufremotepluginconfig.MavenCompilerJavaConfig{
					Encoding: "UTF-8",
					Release:  7,
					Source:   8,
					Target:   9,
				},
				Kotlin: bufremotepluginconfig.MavenCompilerKotlinConfig{
					APIVersion:      "7",
					JVMTarget:       "8",
					LanguageVersion: "9",
					Version:         "1.8.0",
				},
			},
			Deps: []bufremotepluginconfig.MavenDependencyConfig{
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
			AdditionalRuntimes: []bufremotepluginconfig.MavenRuntimeConfig{
				{
					Name: "lite",
					Deps: []bufremotepluginconfig.MavenDependencyConfig{
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
	assertPluginRegistryRoundTrip(t, &bufremotepluginconfig.RegistryConfig{
		Swift: &bufremotepluginconfig.SwiftRegistryConfig{},
	})
	assertPluginRegistryRoundTrip(t, &bufremotepluginconfig.RegistryConfig{
		Swift: &bufremotepluginconfig.SwiftRegistryConfig{
			Dependencies: []bufremotepluginconfig.SwiftRegistryDependencyConfig{
				{
					Source:        "https://github.com/apple/swift-protobuf.git",
					Package:       "swift-protobuf",
					Version:       "1.12.0",
					Products:      []string{"SwiftProtobuf"},
					SwiftVersions: []string{".v5"},
					Platforms: bufremotepluginconfig.SwiftRegistryDependencyPlatformConfig{
						MacOS:   "v10_15",
						IOS:     "v10_15",
						TVOS:    "v10_15",
						WatchOS: "v10_15",
					},
				},
			},
		},
	})
	assertPluginRegistryRoundTrip(t, &bufremotepluginconfig.RegistryConfig{
		Cargo: &bufremotepluginconfig.CargoRegistryConfig{
			RustVersion: "1.60",
			Deps: []bufremotepluginconfig.CargoRegistryDependency{
				{
					Name:               "prost",
					VersionRequirement: "0.12.3",
					DefaultFeatures:    true,
					Features:           []string{"some/feature"},
				},
			},
		},
	})
	assertPluginRegistryRoundTrip(t, &bufremotepluginconfig.RegistryConfig{
		Nuget: &bufremotepluginconfig.NugetRegistryConfig{
			TargetFrameworks: []string{"netstandard2.0", "netstandard2.1"},
			Deps: []bufremotepluginconfig.NugetDependencyConfig{
				{
					Name:    "Grpc.Core.Api",
					Version: "1.2.3",
				},
				{
					Name:             "Grpc.Other.Api",
					Version:          "4.5.6",
					TargetFrameworks: []string{"netstandard2.1"},
				},
			},
		},
	})
	assertPluginRegistryRoundTrip(t, &bufremotepluginconfig.RegistryConfig{
		Cmake: &bufremotepluginconfig.CmakeRegistryConfig{},
	})
}

func assertPluginRegistryRoundTrip(t testing.TB, config *bufremotepluginconfig.RegistryConfig) {
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
	protoLanguages, err = OutputLanguagesToProtoLanguages([]string{"java", "kotlin", "c"})
	require.NoError(t, err)
	assert.Equal(t,
		[]registryv1alpha1.PluginLanguage{
			registryv1alpha1.PluginLanguage_PLUGIN_LANGUAGE_JAVA,
			registryv1alpha1.PluginLanguage_PLUGIN_LANGUAGE_KOTLIN,
			registryv1alpha1.PluginLanguage_PLUGIN_LANGUAGE_C,
		},
		protoLanguages,
	)
	_, err = OutputLanguagesToProtoLanguages([]string{"unknown_language", "another_unknown_language"})
	require.Error(t, err)
	protoLanguages, err = OutputLanguagesToProtoLanguages(nil)
	require.NoError(t, err)
	assert.Equal(t, 0, len(protoLanguages))
}

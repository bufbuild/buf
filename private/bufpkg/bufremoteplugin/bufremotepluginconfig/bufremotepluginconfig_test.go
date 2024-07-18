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

package bufremotepluginconfig

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginref"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGetConfigForBucket(t *testing.T) {
	t.Parallel()
	storageosProvider := storageos.NewProvider()
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(filepath.Join("testdata", "success", "go"))
	require.NoError(t, err)
	pluginConfig, err := GetConfigForBucket(context.Background(), readWriteBucket)
	require.NoError(t, err)
	pluginIdentity, err := bufremotepluginref.PluginIdentityForString("buf.build/library/go-grpc")
	require.NoError(t, err)
	pluginDependency, err := bufremotepluginref.PluginReferenceForString("buf.build/library/go:v1.28.0", 1)
	require.NoError(t, err)
	require.Equal(
		t,
		&Config{
			Name:          pluginIdentity,
			PluginVersion: "v1.2.0",
			SourceURL:     "https://github.com/grpc/grpc-go",
			Description:   "Generates Go language bindings of services in protobuf definition files for gRPC.",
			Dependencies: []bufremotepluginref.PluginReference{
				pluginDependency,
			},
			OutputLanguages: []string{"go"},
			Registry: &RegistryConfig{
				Go: &GoRegistryConfig{
					MinVersion: "1.18",
					Deps: []*GoRegistryDependencyConfig{
						{
							Module:  "google.golang.org/grpc",
							Version: "v1.32.0",
						},
					},
				},
				Options: map[string]string{
					"separate_package": "true",
				},
			},
			SPDXLicenseID:       "Apache-2.0",
			LicenseURL:          "https://github.com/grpc/grpc-go/blob/master/LICENSE",
			IntegrationGuideURL: "https://grpc.io/docs/languages/go/quickstart",
		},
		pluginConfig,
	)
}

func TestParsePluginConfigGoYAML(t *testing.T) {
	t.Parallel()
	pluginConfig, err := ParseConfig(filepath.Join("testdata", "success", "go", "buf.plugin.yaml"))
	require.NoError(t, err)
	pluginIdentity, err := bufremotepluginref.PluginIdentityForString("buf.build/library/go-grpc")
	require.NoError(t, err)
	pluginDependency, err := bufremotepluginref.PluginReferenceForString("buf.build/library/go:v1.28.0", 1)
	require.NoError(t, err)
	require.Equal(
		t,
		&Config{
			Name:          pluginIdentity,
			PluginVersion: "v1.2.0",
			SourceURL:     "https://github.com/grpc/grpc-go",
			Description:   "Generates Go language bindings of services in protobuf definition files for gRPC.",
			Dependencies: []bufremotepluginref.PluginReference{
				pluginDependency,
			},
			OutputLanguages: []string{"go"},
			Registry: &RegistryConfig{
				Go: &GoRegistryConfig{
					MinVersion: "1.18",
					Deps: []*GoRegistryDependencyConfig{
						{
							Module:  "google.golang.org/grpc",
							Version: "v1.32.0",
						},
					},
				},
				Options: map[string]string{
					"separate_package": "true",
				},
			},
			SPDXLicenseID:       "Apache-2.0",
			LicenseURL:          "https://github.com/grpc/grpc-go/blob/master/LICENSE",
			IntegrationGuideURL: "https://grpc.io/docs/languages/go/quickstart",
		},
		pluginConfig,
	)
}

func TestParsePluginConfigGoYAMLOverrideRemote(t *testing.T) {
	t.Parallel()
	pluginConfig, err := ParseConfig(filepath.Join("testdata", "success", "go", "buf.plugin.yaml"), WithOverrideRemote("buf.mydomain.com"))
	require.NoError(t, err)
	pluginIdentity, err := bufremotepluginref.PluginIdentityForString("buf.mydomain.com/library/go-grpc")
	require.NoError(t, err)
	pluginDependency, err := bufremotepluginref.PluginReferenceForString("buf.mydomain.com/library/go:v1.28.0", 1)
	require.NoError(t, err)
	assert.Equal(t, pluginIdentity, pluginConfig.Name)
	require.Len(t, pluginConfig.Dependencies, 1)
	assert.Equal(t, pluginDependency, pluginConfig.Dependencies[0])
}

func TestParsePluginConfigNPMYAML(t *testing.T) {
	t.Parallel()
	pluginConfig, err := ParseConfig(filepath.Join("testdata", "success", "npm", "buf.plugin.yaml"))
	require.NoError(t, err)
	pluginIdentity, err := bufremotepluginref.PluginIdentityForString("buf.build/protocolbuffers/js")
	require.NoError(t, err)
	require.Equal(
		t,
		&Config{
			Name:            pluginIdentity,
			PluginVersion:   "v1.0.0",
			OutputLanguages: []string{"typescript"},
			Registry: &RegistryConfig{
				NPM: &NPMRegistryConfig{
					ImportStyle: "commonjs",
					Deps: []*NPMRegistryDependencyConfig{
						{
							Package: "grpc-web",
							Version: "^1.3.1",
						},
						{
							Package: "@types/google-protobuf",
							Version: "^3.15.6",
						},
					},
				},
			},
			SPDXLicenseID: "BSD-3-Clause",
		},
		pluginConfig,
	)
}

func TestParsePluginConfigMavenYAML(t *testing.T) {
	t.Parallel()
	pluginConfig, err := ParseConfig(filepath.Join("testdata", "success", "maven", "buf.plugin.yaml"))
	require.NoError(t, err)
	pluginIdentity, err := bufremotepluginref.PluginIdentityForString("buf.build/grpc/java")
	require.NoError(t, err)
	pluginDep, err := bufremotepluginref.PluginReferenceForString("buf.build/protocolbuffers/java:v22.2", 0)
	require.NoError(t, err)
	require.Equal(
		t,
		&Config{
			Name:            pluginIdentity,
			Dependencies:    []bufremotepluginref.PluginReference{pluginDep},
			PluginVersion:   "v1.0.0",
			OutputLanguages: []string{"java"},
			Registry: &RegistryConfig{
				Maven: &MavenRegistryConfig{
					Compiler: MavenCompilerConfig{
						Java: MavenCompilerJavaConfig{
							Encoding: "UTF-8",
							Release:  11,
							Source:   8,
							Target:   17,
						},
						Kotlin: MavenCompilerKotlinConfig{
							APIVersion:      "1.8",
							JVMTarget:       "9",
							LanguageVersion: "1.7",
							Version:         "1.8.0",
						},
					},
					Deps: []MavenDependencyConfig{
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
							ArtifactID: "grpc-stub",
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
					AdditionalRuntimes: []MavenRuntimeConfig{
						{
							Name: "lite",
							Deps: []MavenDependencyConfig{
								{
									GroupID:    "io.grpc",
									ArtifactID: "grpc-core",
									Version:    "1.52.1",
								},
								{
									GroupID:    "io.grpc",
									ArtifactID: "grpc-protobuf-lite",
									Version:    "1.52.1",
								},
								{
									GroupID:    "io.grpc",
									ArtifactID: "grpc-stub",
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
							Options: []string{
								"lite",
							},
						},
					},
				},
			},
			SPDXLicenseID: "BSD-3-Clause",
		},
		pluginConfig,
	)
}

func TestParsePluginConfigSwiftYAML(t *testing.T) {
	t.Parallel()
	pluginConfig, err := ParseConfig(filepath.Join("testdata", "success", "swift", "buf.plugin.yaml"))
	require.NoError(t, err)
	pluginIdentity, err := bufremotepluginref.PluginIdentityForString("buf.build/connectrpc/swift")
	require.NoError(t, err)
	pluginDep, err := bufremotepluginref.PluginReferenceForString("buf.build/apple/swift:v1.23.0", 0)
	require.NoError(t, err)
	require.Equal(
		t,
		&Config{
			Name:            pluginIdentity,
			PluginVersion:   "v0.8.0",
			SourceURL:       "https://github.com/connectrpc/connect-swift",
			Description:     "Idiomatic gRPC & Connect RPCs for Swift.",
			Dependencies:    []bufremotepluginref.PluginReference{pluginDep},
			OutputLanguages: []string{"swift"},
			Registry: &RegistryConfig{
				Swift: &SwiftRegistryConfig{
					Dependencies: []SwiftRegistryDependencyConfig{
						{
							Source:        "https://github.com/connectrpc/connect-swift.git",
							Package:       "connect-swift",
							Version:       "0.8.0",
							Products:      []string{"Connect"},
							SwiftVersions: []string{".v5"},
							Platforms: SwiftRegistryDependencyPlatformConfig{
								MacOS: "v10_15",
								IOS:   "v12",
								TVOS:  "v13",
							},
						},
					},
				},
			},
			SPDXLicenseID: "Apache-2.0",
			LicenseURL:    "https://github.com/connectrpc/connect-swift/blob/0.8.0/LICENSE",
		},
		pluginConfig,
	)
}

func TestParsePluginConfigPythonYAML(t *testing.T) {
	t.Parallel()
	pluginConfig, err := ParseConfig(filepath.Join("testdata", "success", "python", "buf.plugin.yaml"))
	require.NoError(t, err)
	pluginIdentity, err := bufremotepluginref.PluginIdentityForString("buf.build/community/nipunn1313-mypy")
	require.NoError(t, err)
	require.Equal(
		t,
		&Config{
			Name:            pluginIdentity,
			PluginVersion:   "v3.5.0",
			SourceURL:       "https://github.com/nipunn1313/mypy-protobuf",
			Description:     "Generate mypy stub files from Protobuf definitions.",
			SPDXLicenseID:   "Apache-2.0",
			LicenseURL:      "https://github.com/nipunn1313/mypy-protobuf/blob/v3.5.0/LICENSE",
			OutputLanguages: []string{"python"},
			Registry: &RegistryConfig{
				Python: &PythonRegistryConfig{
					PackageType:    "stub-only",
					RequiresPython: ">=3.8",
					Deps: []string{
						"protobuf>=4.23.4",
						"types-protobuf>=4.23.0.2",
					},
				},
			},
		},
		pluginConfig,
	)
}

func TestParsePluginConfigCargoYAML(t *testing.T) {
	t.Parallel()
	pluginConfig, err := ParseConfig(filepath.Join("testdata", "success", "cargo", "buf.plugin.yaml"))
	require.NoError(t, err)
	pluginIdentity, err := bufremotepluginref.PluginIdentityForString("buf.build/community/neoeinstein-prost")
	require.NoError(t, err)
	require.Equal(
		t,
		&Config{
			Name:            pluginIdentity,
			PluginVersion:   "v0.3.1",
			SourceURL:       "https://github.com/neoeinstein/protoc-gen-prost",
			Description:     "Generates code using the Prost! code generation engine.",
			SPDXLicenseID:   "Apache-2.0",
			LicenseURL:      "https://github.com/neoeinstein/protoc-gen-prost/blob/protoc-gen-prost-v0.3.1/LICENSE",
			OutputLanguages: []string{"rust"},
			Registry: &RegistryConfig{
				Cargo: &CargoRegistryConfig{
					RustVersion: "1.60",
					Deps: []CargoRegistryDependency{
						{
							Name:               "prost",
							VersionRequirement: "0.12.3",
							DefaultFeatures:    true,
							Features:           []string{"a-feature"},
						},
					},
				},
				Options: map[string]string{"enable_type_names": "true"},
			},
		},
		pluginConfig,
	)
}

func TestParsePluginConfigNugetYAML(t *testing.T) {
	t.Parallel()
	pluginConfig, err := ParseConfig(filepath.Join("testdata", "success", "nuget", "buf.plugin.yaml"))
	require.NoError(t, err)
	pluginIdentity, err := bufremotepluginref.PluginIdentityForString("buf.build/grpc/csharp")
	require.NoError(t, err)
	depPluginRef, err := bufremotepluginref.PluginReferenceForString("buf.build/protocolbuffers/csharp:v26.1", 0)
	require.NoError(t, err)
	require.Equal(
		t,
		&Config{
			Name:            pluginIdentity,
			PluginVersion:   "v1.65.0",
			Dependencies:    []bufremotepluginref.PluginReference{depPluginRef},
			SourceURL:       "https://github.com/grpc/grpc",
			Description:     "Generates C# client and server stubs for the gRPC framework.",
			SPDXLicenseID:   "Apache-2.0",
			LicenseURL:      "https://github.com/grpc/grpc/blob/v1.65.0/LICENSE",
			OutputLanguages: []string{"csharp"},
			Registry: &RegistryConfig{
				Nuget: &NugetRegistryConfig{
					TargetFrameworks: []string{"netstandard2.0", "netstandard2.1"},
					Deps: []NugetDependencyConfig{
						{
							Name:    "Grpc.Core.Api",
							Version: "2.63.0",
						},
						{
							Name:             "Grpc.Other.Api",
							Version:          "1.0.31",
							TargetFrameworks: []string{"netstandard2.1"},
						},
					},
				},
				Options: map[string]string{"base_namespace": ""},
			},
		},
		pluginConfig,
	)
}

func TestParsePluginConfigCmakeYAML(t *testing.T) {
	t.Parallel()
	pluginConfig, err := ParseConfig(filepath.Join("testdata", "success", "cmake", "buf.plugin.yaml"))
	require.NoError(t, err)
	pluginIdentity, err := bufremotepluginref.PluginIdentityForString("buf.build/grpc/cpp")
	require.NoError(t, err)
	depPluginRef, err := bufremotepluginref.PluginReferenceForString("buf.build/protocolbuffers/cpp:v26.1", 0)
	require.NoError(t, err)
	require.Equal(
		t,
		&Config{
			Name:            pluginIdentity,
			PluginVersion:   "v1.65.0",
			Dependencies:    []bufremotepluginref.PluginReference{depPluginRef},
			SourceURL:       "https://github.com/grpc/grpc",
			Description:     "Generates C++ client and server stubs for the gRPC framework.",
			SPDXLicenseID:   "Apache-2.0",
			LicenseURL:      "https://github.com/grpc/grpc/blob/v1.65.0/LICENSE",
			OutputLanguages: []string{"cpp"},
			Registry: &RegistryConfig{
				Cmake:   &CmakeRegistryConfig{},
				Options: nil,
			},
		},
		pluginConfig,
	)
}

func TestParsePluginConfigOptionsYAML(t *testing.T) {
	t.Parallel()
	pluginConfig, err := ParseConfig(filepath.Join("testdata", "success", "options", "buf.plugin.yaml"))
	require.NoError(t, err)
	pluginIdentity, err := bufremotepluginref.PluginIdentityForString("buf.build/protocolbuffers/java")
	require.NoError(t, err)
	require.Equal(
		t,
		&Config{
			Name:          pluginIdentity,
			PluginVersion: "v2.0.0",
		},
		pluginConfig,
	)
}

func TestParsePluginConfigMultipleRegistryConfigsYAML(t *testing.T) {
	t.Parallel()
	_, err := ParseConfig(filepath.Join("testdata", "failure", "invalid-multiple-registries.yaml"))
	require.Error(t, err)
}

func TestParsePluginConfigEmptyVersionYAML(t *testing.T) {
	t.Parallel()
	_, err := ParseConfig(filepath.Join("testdata", "failure", "invalid-empty-version.yaml"))
	require.Error(t, err)
}

func TestParsePluginConfigGoNoDepsOrMinVersion(t *testing.T) {
	t.Parallel()
	cfg, err := ParseConfig(filepath.Join("testdata", "success", "go-empty-registry", "buf.plugin.yaml"))
	require.NoError(t, err)
	assert.NotNil(t, cfg.Registry)
	assert.NotNil(t, cfg.Registry.Go)
	assert.Equal(t, &GoRegistryConfig{}, cfg.Registry.Go)
}

func TestPluginOptionsRoundTrip(t *testing.T) {
	t.Parallel()
	assertPluginOptionsRoundTrip(t, nil)
	assertPluginOptionsRoundTrip(t, map[string]string{})
	assertPluginOptionsRoundTrip(t, map[string]string{
		"option-1":          "value-1",
		"option-2":          "value-2",
		"option-no-value-3": "",
	})
}

func assertPluginOptionsRoundTrip(t testing.TB, options map[string]string) {
	optionsSlice := PluginOptionsToOptionsSlice(options)
	assert.True(t, sort.SliceIsSorted(optionsSlice, func(i, j int) bool {
		return optionsSlice[i] < optionsSlice[j]
	}))
	assert.Equal(t, options, OptionsSliceToPluginOptions(optionsSlice))
}

func TestGetConfigForDataInvalidDependency(t *testing.T) {
	t.Parallel()
	validConfig, err := os.ReadFile(filepath.Join("testdata", "success", "go", "buf.plugin.yaml"))
	require.NoError(t, err)
	// Valid dependencies
	verifyDependencies(t, validConfig, false, ExternalDependency{Plugin: "buf.build/library/go:v1.27.1"})
	verifyDependencies(t, validConfig, false, ExternalDependency{Plugin: "buf.build/library/go:v1.27.1-rc.1"})
	// Invalid dependencies
	verifyDependencies(t, validConfig, true, ExternalDependency{Plugin: "library/go:v1.28.0"})
	verifyDependencies(t, validConfig, true, ExternalDependency{Plugin: "buf.build/library/go"})
	verifyDependencies(t, validConfig, true, ExternalDependency{Plugin: "other.buf.build/library/go:v1.28.0"})
	verifyDependencies(t, validConfig, true, ExternalDependency{Plugin: "buf.build/library/go:1.28.0"})
	verifyDependencies(t, validConfig, true, ExternalDependency{Plugin: "buf.build/library/go:v1.28.0", Revision: -1})
	verifyDependencies(t, validConfig, true, ExternalDependency{Plugin: "buf.build/library/go:v1.28.0", Revision: math.MaxInt32 + 1})
	// duplicate dependencies (doesn't matter if version differs)
	verifyDependencies(t, validConfig, true, ExternalDependency{Plugin: "buf.build/library/go:v1.28.0"}, ExternalDependency{Plugin: "buf.build/library/go:v1.27.0", Revision: 1})
}

func verifyDependencies(t testing.TB, validConfigBytes []byte, fail bool, invalidDependencies ...ExternalDependency) {
	t.Helper()
	// make a defensive copy of a valid parsed config
	var cloned *ExternalConfig
	err := yaml.Unmarshal(validConfigBytes, &cloned)
	require.NoError(t, err)
	cloned.Deps = append([]ExternalDependency{}, invalidDependencies...)
	yamlBytes, err := yaml.Marshal(cloned)
	require.NoError(t, err)
	_, err = GetConfigForData(context.Background(), yamlBytes)
	if fail {
		assert.Error(t, err)
	} else {
		assert.NoError(t, err)
	}
}

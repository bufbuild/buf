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

package bufpluginconfig

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsePluginConfigArchiveYAML(t *testing.T) {
	t.Parallel()
	plugin, err := ParsePluginConfig(filepath.Join("testdata", "success", "archive", "buf.plugin.yaml"))
	require.NoError(t, err)
	require.Equal(
		t,
		&PluginConfig{
			Owner:   "buf",
			Name:    "grpc-java",
			Version: "v1.46.0-1",
			Deps:    []string{"java:v1.21.0-1"},
			Runtime: Runtime{
				Archive: &ArchiveConfig{
					Deps: []struct {
						Name    string `json:"name" yaml:"name"`
						Version string `json:"version" yaml:"version"`
					}{
						{
							Name:    "io.grpc:grpc-protobuf",
							Version: "v1.46.0",
						},
						{
							Name:    "io.grpc:grpc-netty-shaded",
							Version: "v1.46.0",
						},
						{
							Name:    "io.grpc:grpc-stub",
							Version: "v1.46.0",
						},
						{
							Name:    "io.grpc:grpc-okhttp",
							Version: "v1.46.0",
						},
					},
				},
			},
		},
		plugin,
	)
}

func TestParsePluginConfigGoYAML(t *testing.T) {
	t.Parallel()
	plugin, err := ParsePluginConfig(filepath.Join("testdata", "success", "go", "buf.plugin.yaml"))
	require.NoError(t, err)
	require.Equal(
		t,
		&PluginConfig{
			Owner:   "buf",
			Name:    "grpc-go",
			Version: "v1.2.0-1",
			Deps:    []string{"java:v1.21.0-1"},
			Runtime: Runtime{
				Go: &GoConfig{
					Deps: []struct {
						Module  string `json:"module" yaml:"module"`
						Version string `json:"version" yaml:"version"`
					}{
						{
							Module:  "google.golang.org/grpc",
							Version: "v1.32.0",
						},
					},
				},
			},
		},
		plugin,
	)
}

func TestParsePluginConfigNPMYAML(t *testing.T) {
	t.Parallel()
	plugin, err := ParsePluginConfig(filepath.Join("testdata", "success", "npm", "buf.plugin.yaml"))
	require.NoError(t, err)
	require.Equal(
		t,
		&PluginConfig{
			Owner:   "buf",
			Name:    "grpc-web",
			Version: "v1.3.1-2",
			Opts:    []string{"path=source_relative"},
			Deps:    []string{"protocolbuffers/js:v1.27.0-1"},
			Runtime: Runtime{
				NPM: &NPMConfig{
					Deps: []struct {
						Package string `json:"package" yaml:"package"`
						Version string `json:"version" yaml:"version"`
					}{
						{
							Package: "grpc-web",
							Version: "v1.3.1",
						},
						{
							Package: "@types/google-protobuf",
							Version: "v3.15.6",
						},
					},
				},
			},
		},
		plugin,
	)
}

func TestParsePluginConfigMultipleRuntimeLangYAML(t *testing.T) {
	t.Parallel()
	_, err := ParsePluginConfig(filepath.Join("testdata", "failure", "invalid-multiple-languages.yaml"))
	require.Error(t, err)
}

func TestParsePluginConfigEmptyVersionYAML(t *testing.T) {
	t.Parallel()
	_, err := ParsePluginConfig(filepath.Join("testdata", "failure", "invalid-empty-version.yaml"))
	require.Error(t, err)
}

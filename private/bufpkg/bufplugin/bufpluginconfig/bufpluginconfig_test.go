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
	"context"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginref"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/require"
)

func TestGetConfigForBucket(t *testing.T) {
	t.Parallel()
	storageosProvider := storageos.NewProvider()
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(filepath.Join("testdata", "success", "go"))
	require.NoError(t, err)
	pluginConfig, err := GetConfigForBucket(context.Background(), readWriteBucket)
	require.NoError(t, err)
	pluginIdentity, err := bufpluginref.PluginIdentityForString("buf.build/grpc/go")
	require.NoError(t, err)
	require.Equal(
		t,
		&Config{
			Name:          pluginIdentity,
			PluginVersion: "v1.5.0",
			Options: map[string]string{
				"paths": "source_relative",
			},
			Runtime: &RuntimeConfig{
				Go: &GoRuntimeConfig{
					MinVersion: "1.18",
					Deps: []*GoRuntimeDependencyConfig{
						{
							Module:  "google.golang.org/grpc",
							Version: "v1.32.0",
						},
					},
				},
			},
		},
		pluginConfig,
	)
}

func TestParsePluginConfigGoYAML(t *testing.T) {
	t.Parallel()
	pluginConfig, err := ParseConfig(filepath.Join("testdata", "success", "go", "buf.plugin.yaml"))
	require.NoError(t, err)
	pluginIdentity, err := bufpluginref.PluginIdentityForString("buf.build/grpc/go")
	require.NoError(t, err)
	require.Equal(
		t,
		&Config{
			Name:          pluginIdentity,
			PluginVersion: "v1.5.0",
			Options: map[string]string{
				"paths": "source_relative",
			},
			Runtime: &RuntimeConfig{
				Go: &GoRuntimeConfig{
					MinVersion: "1.18",
					Deps: []*GoRuntimeDependencyConfig{
						{
							Module:  "google.golang.org/grpc",
							Version: "v1.32.0",
						},
					},
				},
			},
		},
		pluginConfig,
	)
}

func TestParsePluginConfigNPMYAML(t *testing.T) {
	t.Parallel()
	pluginConfig, err := ParseConfig(filepath.Join("testdata", "success", "npm", "buf.plugin.yaml"))
	require.NoError(t, err)
	pluginIdentity, err := bufpluginref.PluginIdentityForString("buf.build/protocolbuffers/js")
	require.NoError(t, err)
	require.Equal(
		t,
		&Config{
			Name:          pluginIdentity,
			PluginVersion: "v1.0.0",
			Options: map[string]string{
				"paths": "source_relative",
			},
			Runtime: &RuntimeConfig{
				NPM: &NPMRuntimeConfig{
					Deps: []*NPMRuntimeDependencyConfig{
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
		},
		pluginConfig,
	)
}

func TestParsePluginConfigOptionsYAML(t *testing.T) {
	t.Parallel()
	pluginConfig, err := ParseConfig(filepath.Join("testdata", "success", "options", "buf.plugin.yaml"))
	require.NoError(t, err)
	pluginIdentity, err := bufpluginref.PluginIdentityForString("buf.build/protocolbuffers/java")
	require.NoError(t, err)
	require.Equal(
		t,
		&Config{
			Name:          pluginIdentity,
			PluginVersion: "v2.0.0",
			Options: map[string]string{
				"annotate_code": "",
			},
		},
		pluginConfig,
	)
}

func TestParsePluginConfigMultipleRuntimeLangYAML(t *testing.T) {
	t.Parallel()
	_, err := ParseConfig(filepath.Join("testdata", "failure", "invalid-multiple-languages.yaml"))
	require.Error(t, err)
}

func TestParsePluginConfigEmptyVersionYAML(t *testing.T) {
	t.Parallel()
	_, err := ParseConfig(filepath.Join("testdata", "failure", "invalid-empty-version.yaml"))
	require.Error(t, err)
}

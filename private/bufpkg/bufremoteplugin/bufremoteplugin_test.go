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

package bufremoteplugin

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseTemplateConfigJSONFile(t *testing.T) {
	t.Parallel()
	template, err := ParseTemplateConfig(filepath.Join("testdata", "template-config", "success", "json", "template-config.json"))
	require.NoError(t, err)
	require.Equal(
		t,
		&TemplateConfig{
			Plugins: []PluginConfig{
				{
					Owner:      "testowner",
					Name:       "testplugin",
					Parameters: []string{"testparameter"},
				},
				{
					Owner:      "testowner2",
					Name:       "testplugin2",
					Parameters: []string{"testparameter2"},
				},
			},
		},
		template,
	)
}

func TestParseTemplateConfigJSONLiteral(t *testing.T) {
	t.Parallel()
	template, err := ParseTemplateConfig(`{"version": "v1","plugins": [{"owner": "testowner","name": "testplugin","opt": ["testparameter"]},{"owner": "testowner2","name": "testplugin2","opt":"testparameter2"}]}`)
	require.NoError(t, err)
	require.Equal(
		t,
		&TemplateConfig{
			Plugins: []PluginConfig{
				{
					Owner:      "testowner",
					Name:       "testplugin",
					Parameters: []string{"testparameter"},
				},
				{
					Owner:      "testowner2",
					Name:       "testplugin2",
					Parameters: []string{"testparameter2"},
				},
			},
		},
		template,
	)
}

func TestParseTemplateConfigYAMLFile(t *testing.T) {
	t.Parallel()
	template, err := ParseTemplateConfig(filepath.Join("testdata", "template-config", "success", "yaml", "template-config.yaml"))
	require.NoError(t, err)
	require.Equal(
		t,
		&TemplateConfig{
			Plugins: []PluginConfig{
				{
					Owner:      "testowner",
					Name:       "testplugin",
					Parameters: []string{"testparameter"},
				},
				{
					Owner:      "testowner2",
					Name:       "testplugin2",
					Parameters: []string{"testparameter2"},
				},
			},
		},
		template,
	)
}

func TestParseTemplateConfigYAMLLiteral(t *testing.T) {
	t.Parallel()
	template, err := ParseTemplateConfig(`
version: v1
plugins:
  - owner: testowner
    name: testplugin
    opt:
      - testparameter
  - owner: testowner2
    name: testplugin2
    opt: testparameter2
`)
	require.NoError(t, err)
	require.Equal(
		t,
		&TemplateConfig{
			Plugins: []PluginConfig{
				{
					Owner:      "testowner",
					Name:       "testplugin",
					Parameters: []string{"testparameter"},
				},
				{
					Owner:      "testowner2",
					Name:       "testplugin2",
					Parameters: []string{"testparameter2"},
				},
			},
		},
		template,
	)
}

func TestParseTemplateConfigYMLFile(t *testing.T) {
	t.Parallel()
	template, err := ParseTemplateConfig(filepath.Join("testdata", "template-config", "success", "yml", "template-config.yml"))
	require.NoError(t, err)
	require.Equal(
		t,
		&TemplateConfig{
			Plugins: []PluginConfig{
				{
					Owner:      "testowner",
					Name:       "testplugin",
					Parameters: []string{"testparameter"},
				},
				{
					Owner:      "testowner2",
					Name:       "testplugin2",
					Parameters: []string{"testparameter2"},
				},
			},
		},
		template,
	)
}

func TestParseTemplateConfigJSONFileWithoutVersion(t *testing.T) {
	t.Parallel()
	_, err := ParseTemplateConfig(filepath.Join("testdata", "template-config", "failure", "json-no-version", "template-config.json"))
	require.Error(t, err)
}

func TestParseTemplateConfigJSONLiteralWithoutVersion(t *testing.T) {
	t.Parallel()
	_, err := ParseTemplateConfig(`{"plugins": [{"owner": "testowner","name": "testplugin","parameters": ["testparameter"]},{"owner": "testowner2","name": "testplugin2"}]}`)
	require.Error(t, err)
}

func TestParseTemplateConfigYAMLFileWithoutVersion(t *testing.T) {
	t.Parallel()
	_, err := ParseTemplateConfig(filepath.Join("testdata", "template-config", "failure", "yaml-no-version", "template-config.yaml"))
	require.Error(t, err)
}

func TestParseTemplateConfigYAMLLiteralWithoutVersion(t *testing.T) {
	t.Parallel()
	_, err := ParseTemplateConfig(`
plugins:
  - owner: testowner
    name: testplugin
    opt:
      - testparameter
  - owner: testowner2
    name: testplugin2
	opt: testparameter2
`)
	require.Error(t, err)
}

func TestParseTemplateConfigJSONFileWithInvalidVersion(t *testing.T) {
	t.Parallel()
	_, err := ParseTemplateConfig(filepath.Join("testdata", "template-config", "failure", "json-invalid-version", "template-config.json"))
	require.Error(t, err)
}

func TestParseTemplateConfigJSONLiteralWithInvalidVersion(t *testing.T) {
	t.Parallel()
	_, err := ParseTemplateConfig(`{"version":"v1beta1", "plugins": [{"owner": "testowner","name": "testplugin","opt": ["testparameter"]},{"owner": "testowner2","name": "testplugin2"}]}`)
	require.Error(t, err)
}

func TestParseTemplateConfigYAMLFileWithInvalidVersion(t *testing.T) {
	t.Parallel()
	_, err := ParseTemplateConfig(filepath.Join("testdata", "template-config", "failure", "yaml-invalid-version", "template-config.yaml"))
	require.Error(t, err)
}

func TestParseTemplateConfigYAMLLiteralWithInvalidVersion(t *testing.T) {
	t.Parallel()
	_, err := ParseTemplateConfig(`
version: v1beta1
plugins:
  - owner: testowner
    name: testplugin
    opt:
      - testparameter
  - owner: testowner2
    name: testplugin2
	opt: testparameter2
`)
	require.Error(t, err)
}

func TestParseTemplateVersionConfigJSONFile(t *testing.T) {
	t.Parallel()
	templateVersion, err := ParseTemplateVersionConfig(filepath.Join("testdata", "template-version-config", "success", "json", "template-version-config.json"))
	require.NoError(t, err)
	require.Equal(
		t,
		&TemplateVersionConfig{
			PluginVersions: []PluginVersion{
				{
					Owner:   "testowner",
					Name:    "testplugin",
					Version: "v1.2.0",
				},
				{
					Owner:   "testowner2",
					Name:    "testplugin2",
					Version: "v2.1.0",
				},
			},
		},
		templateVersion,
	)
}

func TestParseTemplateVersionConfigJSONLiteral(t *testing.T) {
	t.Parallel()
	templateVersion, err := ParseTemplateVersionConfig(`{"version": "v1","plugin_versions": [{"owner": "testowner","name": "testplugin","version":"v1.2.0"},{"owner": "testowner2","name": "testplugin2","version":"v2.1.0"}]}`)
	require.NoError(t, err)
	require.Equal(
		t,
		&TemplateVersionConfig{
			PluginVersions: []PluginVersion{
				{
					Owner:   "testowner",
					Name:    "testplugin",
					Version: "v1.2.0",
				},
				{
					Owner:   "testowner2",
					Name:    "testplugin2",
					Version: "v2.1.0",
				},
			},
		},
		templateVersion,
	)
}

func TestParseTemplateVersionConfigYAMLFile(t *testing.T) {
	t.Parallel()
	templateVersion, err := ParseTemplateVersionConfig(filepath.Join("testdata", "template-version-config", "success", "yaml", "template-version-config.yaml"))
	require.NoError(t, err)
	require.Equal(
		t,
		&TemplateVersionConfig{
			PluginVersions: []PluginVersion{
				{
					Owner:   "testowner",
					Name:    "testplugin",
					Version: "v1.2.0",
				},
				{
					Owner:   "testowner2",
					Name:    "testplugin2",
					Version: "v2.1.0",
				},
			},
		},
		templateVersion,
	)
}

func TestParseTemplateVersionConfigYAMLLiteral(t *testing.T) {
	t.Parallel()
	templateVersion, err := ParseTemplateVersionConfig(`
version: v1
plugin_versions:
  - owner: testowner
    name: testplugin
    version: v1.2.0
  - owner: testowner2
    name: testplugin2
    version: v2.1.0
`)
	require.NoError(t, err)
	require.Equal(
		t,
		&TemplateVersionConfig{
			PluginVersions: []PluginVersion{
				{
					Owner:   "testowner",
					Name:    "testplugin",
					Version: "v1.2.0",
				},
				{
					Owner:   "testowner2",
					Name:    "testplugin2",
					Version: "v2.1.0",
				},
			},
		},
		templateVersion,
	)
}

func TestParseTemplateVersionConfigYMLFile(t *testing.T) {
	t.Parallel()
	templateVersion, err := ParseTemplateVersionConfig(filepath.Join("testdata", "template-version-config", "success", "yml", "template-version-config.yml"))
	require.NoError(t, err)
	require.Equal(
		t,
		&TemplateVersionConfig{
			PluginVersions: []PluginVersion{
				{
					Owner:   "testowner",
					Name:    "testplugin",
					Version: "v1.2.0",
				},
				{
					Owner:   "testowner2",
					Name:    "testplugin2",
					Version: "v2.1.0",
				},
			},
		},
		templateVersion,
	)
}

func TestParseTemplateVersionConfigJSONFileWithoutVersion(t *testing.T) {
	t.Parallel()
	_, err := ParseTemplateVersionConfig(filepath.Join("testdata", "template-version-config", "failure", "json-no-version", "template-version-config.json"))
	require.Error(t, err)
}

func TestParseTemplateVersionConfigJSONLiteralWithoutVersion(t *testing.T) {
	t.Parallel()
	_, err := ParseTemplateVersionConfig(`{"plugin_versions": [{"owner": "testowner","name": "testplugin","version":"v1.2.0"},{"owner": "testowner2","name": "testplugin2","version":"v2.1.0"}]}`)
	require.Error(t, err)
}

func TestParseTemplateVersionConfigYAMLFileWithoutVersion(t *testing.T) {
	t.Parallel()
	_, err := ParseTemplateVersionConfig(filepath.Join("testdata", "template-version-config", "failure", "yaml-no-version", "template-version-config.yaml"))
	require.Error(t, err)
}

func TestParseTemplateVersionConfigYAMLLiteralWithoutVersion(t *testing.T) {
	t.Parallel()
	_, err := ParseTemplateVersionConfig(`
plugin_versions:
  - owner: testowner
    name: testplugin
    version: v1.2.0
  - owner: testowner2
    name: testplugin2
    version: v2.1.0
`)
	require.Error(t, err)
}

func TestParseTemplateVersionConfigJSONFileWithInvalidVersion(t *testing.T) {
	t.Parallel()
	_, err := ParseTemplateVersionConfig(filepath.Join("testdata", "template-version-config", "failure", "json-invalid-version", "template-version-config.json"))
	require.Error(t, err)
}

func TestParseTemplateVersionConfigJSONLiteralWithInvalidVersion(t *testing.T) {
	t.Parallel()
	_, err := ParseTemplateVersionConfig(`{"version": "v1beta1","plugin_versions": [{"owner": "testowner","name": "testplugin","version":"v1.2.0"},{"owner": "testowner2","name": "testplugin2","version":"v2.1.0"}]}`)
	require.Error(t, err)
}

func TestParseTemplateVersionConfigYAMLFileWithInvalidVersion(t *testing.T) {
	t.Parallel()
	_, err := ParseTemplateVersionConfig(filepath.Join("testdata", "template-version-config", "failure", "yaml-invalid-version", "template-version-config.yaml"))
	require.Error(t, err)
}

func TestParseTemplateVersionConfigYAMLLiteralWithInvalidVersion(t *testing.T) {
	t.Parallel()
	_, err := ParseTemplateVersionConfig(`
version: v1beta1
plugin_versions:
  - owner: testowner
    name: testplugin
    version: v1.2.0
  - owner: testowner2
    name: testplugin2
    version: v2.1.0
`)
	require.Error(t, err)
}

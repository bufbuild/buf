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
	"fmt"
	"os"
	"path/filepath"

	"github.com/bufbuild/buf/private/pkg/encoding"
)

const (
	PluginConfigPath = "buf.plugin.yaml"
)

// PluginConfig is the config used to describe a plugin.
type PluginConfig struct {
	Owner   string   `json:"owner" yaml:"owner"`
	Name    string   `json:"name" yaml:"name"`
	Version string   `json:"version" yaml:"version"`
	Opts    []string `json:"opts,omitempty" yaml:"opts,omitempty"`
	Deps    []string `json:"deps,omitempty" yaml:"deps,omitempty"`
	Runtime Runtime  `json:"runtime" yaml:"runtime"`
}

// Runtime is the configuration for the runtime of a plugin.
type Runtime struct {
	Go      *GoConfig      `json:"go,omitempty" yaml:"go,omitempty"`
	NPM     *NPMConfig     `json:"npm,omitempty" yaml:"npm,omitempty"`
	Archive *ArchiveConfig `json:"archive,omitempty" yaml:"archive,omitempty"`
}

// GoConfig is the configuration for a Go plugin.
type GoConfig struct {
	// The minimum Go version required by the plugin.
	MinLangVersion string `json:"min_lang_version" yaml:"min_lang_version"`
	Deps           []struct {
		Module  string `json:"module" yaml:"module"`
		Version string `json:"version" yaml:"version"`
	} `json:"deps,omitempty" yaml:"deps,omitempty"`
}

// NPMConfig is the configuration for a JavaScript NPM plugin.
type NPMConfig struct {
	Deps []struct {
		Package string `json:"package" yaml:"package"`
		Version string `json:"version" yaml:"version"`
	} `json:"deps,omitempty" yaml:"deps,omitempty"`
}

// ArchiveConfig is the configuration for a plugin that can be downloaded as
// an archive instead of a language-specific registry.
type ArchiveConfig struct {
	Deps []struct {
		Name    string `json:"name" yaml:"name"`
		Version string `json:"version" yaml:"version"`
	} `json:"deps,omitempty" yaml:"deps,omitempty"`
}

// ParsePluginConfig parses the file at the given path as a PluginConfig.
func ParsePluginConfig(config string) (*PluginConfig, error) {
	var data []byte
	var err error
	switch filepath.Ext(config) {
	case ".json", ".yaml", ".yml":
		data, err = os.ReadFile(config)
		if err != nil {
			return nil, fmt.Errorf("could not read file: %w", err)
		}
	default:
		return nil, fmt.Errorf("invalid extension %s, must be .json, .yaml or .yml", filepath.Ext(config))
	}
	var pluginConfig PluginConfig
	if err := encoding.UnmarshalJSONOrYAMLStrict(data, &pluginConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plugin config: %w", err)
	}
	if err = validateConfig(pluginConfig); err != nil {
		return nil, fmt.Errorf("invalid plugin config: %w", err)
	}
	return &pluginConfig, nil
}

// Copyright 2020-2025 Buf Technologies, Inc.
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

package bufcobra

import (
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// TODO: convert to flags, the only tricky one is command weigh configs
// config represents the config file for the webpages command.
// For example:
//
//	exclude_commands:
//		- buf completion
//		- buf ls-files
//	weight_commands:
//	buf beta: 1
//	slug_prefix: /reference/cli/
//	output_dir: output/docs
type config struct {
	// ExcludeCommands will filter out these command paths from generation.
	ExcludeCommands []string `yaml:"exclude_commands,omitempty"`
	// WeightCommands will weight the command paths and show higher weighted commands later on the sidebar.
	WeightCommands map[string]int `yaml:"weight_commands,omitempty"`
	SlugPrefix     string         `yaml:"slug_prefix,omitempty"`
	OutputDir      string         `yaml:"output_dir,omitempty"`
	// SidebarPathThreshold will dictate if the sidebar label is the full path or just the name.
	// if the command path is longer than this then the `cobra.Command.Name()` is used,
	// otherwise `cobra.Command.CommandPath() is used.
	SidebarPathThreshold int `yaml:"sidebar_path_threshold,omitempty"`
}

func readConfigFromFile(path string) (*config, error) {
	var webpagesConfig config
	if path == "" {
		return &webpagesConfig, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, &webpagesConfig); err != nil {
		return nil, err
	}
	return &webpagesConfig, err
}

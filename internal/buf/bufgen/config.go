// Copyright 2020-2021 Buf Technologies, Inc.
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

package bufgen

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/bufbuild/buf/internal/pkg/encoding"
)

const v1beta1Version = "v1beta1"

func readConfig(fileOrData string) (*Config, error) {
	switch filepath.Ext(fileOrData) {
	case ".json":
		return getConfigJSONFile(fileOrData)
	case ".yaml":
		return getConfigYAMLFile(fileOrData)
	default:
		return getConfigJSONOrYAMLData(fileOrData)
	}
}

func getConfigJSONFile(file string) (*Config, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("could not read file %s: %v", file, err)
	}
	return getConfig(
		encoding.UnmarshalJSONNonStrict,
		encoding.UnmarshalJSONStrict,
		data,
		file,
	)
}

func getConfigYAMLFile(file string) (*Config, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("could not read file %s: %v", file, err)
	}
	return getConfig(
		encoding.UnmarshalYAMLNonStrict,
		encoding.UnmarshalYAMLStrict,
		data,
		file,
	)
}

func getConfigJSONOrYAMLData(data string) (*Config, error) {
	return getConfig(
		encoding.UnmarshalJSONOrYAMLNonStrict,
		encoding.UnmarshalJSONOrYAMLStrict,
		[]byte(data),
		"Generate configuration data",
	)
}

func getConfig(
	unmarshalNonStrict func([]byte, interface{}) error,
	unmarshalStrict func([]byte, interface{}) error,
	data []byte,
	id string,
) (*Config, error) {
	var externalConfigVersion externalConfigVersion
	if err := unmarshalNonStrict(data, &externalConfigVersion); err != nil {
		return nil, err
	}
	switch externalConfigVersion.Version {
	case v1beta1Version:
	default:
		return nil, fmt.Errorf(`%s has no version set. Please add "version: %s"`, id, v1beta1Version)
	}
	var externalConfigV1Beta1 ExternalConfigV1Beta1
	if err := unmarshalStrict(data, &externalConfigV1Beta1); err != nil {
		return nil, err
	}
	if err := validateExternalConfigV1Beta1(externalConfigV1Beta1, id); err != nil {
		return nil, err
	}
	return newConfigV1Beta1(externalConfigV1Beta1, id)
}

func validateExternalConfigV1Beta1(externalConfig ExternalConfigV1Beta1, id string) error {
	if len(externalConfig.Plugins) == 0 {
		return fmt.Errorf("%s: no plugins set", id)
	}
	for _, plugin := range externalConfig.Plugins {
		if plugin.Name == "" {
			return fmt.Errorf("%s: plugin name is required", id)
		}
		if plugin.Out == "" {
			return fmt.Errorf("%s: plugin %s out is required", id, plugin.Name)
		}
	}
	return nil
}

func newConfigV1Beta1(externalConfig ExternalConfigV1Beta1, id string) (*Config, error) {
	config := &Config{}
	for _, plugin := range externalConfig.Plugins {
		strategy, err := ParseStrategy(plugin.Strategy)
		if err != nil {
			return nil, err
		}
		var opt string
		switch t := plugin.Opt.(type) {
		case string:
			opt = t
		case []interface{}:
			opts := make([]string, len(t))
			for i, elem := range t {
				s, ok := elem.(string)
				if !ok {
					return nil, fmt.Errorf("%s: could not convert opt element %T to a string", id, elem)
				}
				opts[i] = s
			}
			opt = strings.Join(opts, ",")
		case nil:
			// If opt is omitted, plugin.Opt is nil
		default:
			return nil, fmt.Errorf("%s: unknown type %T for opt", id, t)
		}
		config.PluginConfigs = append(
			config.PluginConfigs,
			&PluginConfig{
				Name:     plugin.Name,
				Out:      plugin.Out,
				Opt:      opt,
				Path:     plugin.Path,
				Strategy: strategy,
			},
		)
	}
	return config, nil
}

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
	"os"
	"path/filepath"
	"strings"

	"github.com/bufbuild/buf/internal/pkg/encoding"
	"google.golang.org/protobuf/types/descriptorpb"
)

const v1beta1Version = "v1beta1"

func readConfig(fileOrData string) (*Config, error) {
	switch filepath.Ext(fileOrData) {
	case ".json":
		return getConfigJSONFile(fileOrData)
	case ".yaml", ".yml":
		return getConfigYAMLFile(fileOrData)
	default:
		return getConfigJSONOrYAMLData(fileOrData)
	}
}

func getConfigJSONFile(file string) (*Config, error) {
	data, err := os.ReadFile(file)
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
	data, err := os.ReadFile(file)
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
	options, err := newOptionsConfigV1Beta1(externalConfig.Options)
	if err != nil {
		return nil, err
	}
	pluginConfigs := make([]*PluginConfig, 0, len(externalConfig.Plugins))
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
		pluginConfigs = append(
			pluginConfigs,
			&PluginConfig{
				Name:     plugin.Name,
				Out:      plugin.Out,
				Opt:      opt,
				Path:     plugin.Path,
				Strategy: strategy,
			},
		)
	}
	return &Config{
		Managed:       externalConfig.Managed,
		Options:       options,
		PluginConfigs: pluginConfigs,
	}, nil
}

func newOptionsConfigV1Beta1(externalOptionsConfig ExternalOptionsConfigV1Beta1) (*Options, error) {
	if externalOptionsConfig == (ExternalOptionsConfigV1Beta1{}) {
		return nil, nil
	}
	var optimizeFor *descriptorpb.FileOptions_OptimizeMode
	if externalOptionsConfig.OptimizeFor != "" {
		value, ok := descriptorpb.FileOptions_OptimizeMode_value[externalOptionsConfig.OptimizeFor]
		if !ok {
			return nil, fmt.Errorf(
				"invalid optimize_for value; expected one of %v",
				enumMapToStringSlice(descriptorpb.FileOptions_OptimizeMode_value),
			)
		}
		optimizeFor = optimizeModePtr(descriptorpb.FileOptions_OptimizeMode(value))
	}
	return &Options{
		CcEnableArenas:    externalOptionsConfig.CcEnableArenas,
		JavaMultipleFiles: externalOptionsConfig.JavaMultipleFiles,
		OptimizeFor:       optimizeFor,
	}, nil
}

// enumMapToStringSlice is a convenience function for mapping Protobuf enums
// into a slice of strings.
func enumMapToStringSlice(enums map[string]int32) []string {
	slice := make([]string, 0, len(enums))
	for enum := range enums {
		slice = append(slice, enum)
	}
	return slice
}

// optimizeModePtr is a convenience function for initializing the
// *descriptorpb.FileOptions_OptimizeMode type in-line. This is
// also useful in unit tests.
func optimizeModePtr(value descriptorpb.FileOptions_OptimizeMode) *descriptorpb.FileOptions_OptimizeMode {
	return &value
}

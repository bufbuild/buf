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

package bufconfig

import (
	"errors"

	"github.com/bufbuild/buf/private/pkg/slicesext"
)

// GenerateConfig is a generation configuration.
type GenerateConfig interface {
	// GeneratePluginConfigs returns the plugin configurations. This will always be
	// non-empty. Zero plugin configs will cause an error at construction time.
	GeneratePluginConfigs() []GeneratePluginConfig
	// GenerateManagedConfig returns the managed mode configuration.
	// This may be nil.
	GenerateManagedConfig() GenerateManagedConfig
	// GenerateTypeConfig returns the types to generate code for. This overrides other type
	// filters from input configurations, which exist in v2.
	// This will always be nil in v2
	GenerateTypeConfig() GenerateTypeConfig
	// GenerateInputConfigs returns the input config.
	GenerateInputConfigs() []GenerateInputConfig

	isGenerateConfig()
}

func newGenerateConfigFromExternalFileV1(
	externalFile externalBufGenYAMLFileV1,
) (GenerateConfig, error) {
	managedConfig, err := newManagedOverrideRuleFromExternalV1(externalFile.Managed)
	if err != nil {
		return nil, err
	}
	if len(externalFile.Plugins) == 0 {
		return nil, errors.New("must specifiy at least one plugin")
	}
	pluginConfigs, err := slicesext.MapError(
		externalFile.Plugins,
		newPluginConfigFromExternalV1,
	)
	if err != nil {
		return nil, err
	}
	return &generateConfig{
		pluginConfigs: pluginConfigs,
		managedConfig: managedConfig,
		typeConfig:    newGenerateTypeConfig(externalFile.Types.Include),
		// TODO for v2
		inputConfigs: nil,
	}, nil
}

// *** PRIVATE ***

type generateConfig struct {
	pluginConfigs []GeneratePluginConfig
	managedConfig GenerateManagedConfig
	typeConfig    GenerateTypeConfig
	inputConfigs  []GenerateInputConfig
}

func newGenerateConfig() *generateConfig {
	return &generateConfig{}
}

func (*generateConfig) GeneratePluginConfigs() []GeneratePluginConfig {
	return nil
}

func (*generateConfig) GenerateManagedConfig() GenerateManagedConfig {
	return nil
}

func (*generateConfig) GenerateTypeConfig() GenerateTypeConfig {
	return nil
}

func (*generateConfig) GenerateInputConfigs() []GenerateInputConfig {
	return nil
}

func (*generateConfig) isGenerateConfig() {}

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
	// This may will never be nil.
	GenerateManagedConfig() GenerateManagedConfig
	// GenerateTypeConfig returns the types to generate code for. This overrides other type
	// filters from input configurations, which exist in v2.
	// This will always be nil in v2
	GenerateTypeConfig() GenerateTypeConfig

	isGenerateConfig()
}

// NewGenerateConfig returns a validated GenerateConfig.
func NewGenerateConfig(
	pluginConfigs []GeneratePluginConfig,
	managedConfig GenerateManagedConfig,
	typeConfig GenerateTypeConfig,
) (GenerateConfig, error) {
	if len(pluginConfigs) == 0 {
		return nil, newNoPluginsError()
	}
	return &generateConfig{
		pluginConfigs: pluginConfigs,
		managedConfig: managedConfig,
		typeConfig:    typeConfig,
	}, nil
}

// *** PRIVATE ***

type generateConfig struct {
	pluginConfigs []GeneratePluginConfig
	managedConfig GenerateManagedConfig
	typeConfig    GenerateTypeConfig
}

func newGenerateConfigFromExternalFileV1Beta1(
	externalFile externalBufGenYAMLFileV1Beta1,
) (GenerateConfig, error) {
	managedConfig, err := newManagedConfigFromExternalV1Beta1(externalFile.Managed, externalFile.Options)
	if err != nil {
		return nil, err
	}
	if len(externalFile.Plugins) == 0 {
		return nil, newNoPluginsError()
	}
	pluginConfigs, err := slicesext.MapError(
		externalFile.Plugins,
		newPluginConfigFromExternalV1Beta1,
	)
	if err != nil {
		return nil, err
	}
	return &generateConfig{
		pluginConfigs: pluginConfigs,
		managedConfig: managedConfig,
	}, nil
}

func newGenerateConfigFromExternalFileV1(
	externalFile externalBufGenYAMLFileV1,
) (GenerateConfig, error) {
	managedConfig, err := newManagedConfigFromExternalV1(externalFile.Managed)
	if err != nil {
		return nil, err
	}
	if len(externalFile.Plugins) == 0 {
		return nil, newNoPluginsError()
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
	}, nil
}

func newGenerateConfigFromExternalFileV2(
	externalFile externalBufGenYAMLFileV2,
) (GenerateConfig, error) {
	managedConfig, err := newManagedConfigFromExternalV2(externalFile.Managed)
	if err != nil {
		return nil, err
	}
	pluginConfigs, err := slicesext.MapError(
		externalFile.Plugins,
		newPluginConfigFromExternalV2,
	)
	if err != nil {
		return nil, err
	}
	return &generateConfig{
		managedConfig: managedConfig,
		pluginConfigs: pluginConfigs,
	}, nil
}

func (g *generateConfig) GeneratePluginConfigs() []GeneratePluginConfig {
	return g.pluginConfigs
}

func (g *generateConfig) GenerateManagedConfig() GenerateManagedConfig {
	return g.managedConfig
}

func (g *generateConfig) GenerateTypeConfig() GenerateTypeConfig {
	return g.typeConfig
}

func (*generateConfig) isGenerateConfig() {}

func newNoPluginsError() error {
	return errors.New("must specify at least one plugin")
}

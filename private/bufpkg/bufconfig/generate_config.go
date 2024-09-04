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
	// CleanPluginOuts is whether to delete the output directories, zip files, or jar files before
	// generation is run.
	CleanPluginOuts() bool
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
	cleanPluginOuts bool,
	generatePluginConfigs []GeneratePluginConfig,
	generateManagedConfig GenerateManagedConfig,
	generateTypeConfig GenerateTypeConfig,
) (GenerateConfig, error) {
	if len(generatePluginConfigs) == 0 {
		return nil, newNoPluginsError()
	}
	return &generateConfig{
		cleanPluginOuts:       cleanPluginOuts,
		generatePluginConfigs: generatePluginConfigs,
		generateManagedConfig: generateManagedConfig,
		generateTypeConfig:    generateTypeConfig,
	}, nil
}

// *** PRIVATE ***

type generateConfig struct {
	cleanPluginOuts       bool
	generatePluginConfigs []GeneratePluginConfig
	generateManagedConfig GenerateManagedConfig
	generateTypeConfig    GenerateTypeConfig
}

func newGenerateConfigFromExternalFileV1Beta1(
	externalFile externalBufGenYAMLFileV1Beta1,
) (GenerateConfig, error) {
	generateManagedConfig, err := newGenerateManagedConfigFromExternalV1Beta1(externalFile.Managed, externalFile.Options)
	if err != nil {
		return nil, err
	}
	if len(externalFile.Plugins) == 0 {
		return nil, newNoPluginsError()
	}
	generatePluginConfigs, err := slicesext.MapError(
		externalFile.Plugins,
		newGeneratePluginConfigFromExternalV1Beta1,
	)
	if err != nil {
		return nil, err
	}
	return &generateConfig{
		generatePluginConfigs: generatePluginConfigs,
		generateManagedConfig: generateManagedConfig,
	}, nil
}

func newGenerateConfigFromExternalFileV1(
	externalFile externalBufGenYAMLFileV1,
) (GenerateConfig, error) {
	generateManagedConfig, err := newGenerateManagedConfigFromExternalV1(externalFile.Managed)
	if err != nil {
		return nil, err
	}
	if len(externalFile.Plugins) == 0 {
		return nil, newNoPluginsError()
	}
	generatePluginConfigs, err := slicesext.MapError(
		externalFile.Plugins,
		newGeneratePluginConfigFromExternalV1,
	)
	if err != nil {
		return nil, err
	}
	return &generateConfig{
		generatePluginConfigs: generatePluginConfigs,
		generateManagedConfig: generateManagedConfig,
		generateTypeConfig:    newGenerateTypeConfig(externalFile.Types.Include),
	}, nil
}

func newGenerateConfigFromExternalFileV2(
	externalFile externalBufGenYAMLFileV2,
) (GenerateConfig, error) {
	generateManagedConfig, err := newGenerateManagedConfigFromExternalV2(externalFile.Managed)
	if err != nil {
		return nil, err
	}
	generatePluginConfigs, err := slicesext.MapError(
		externalFile.Plugins,
		newGeneratePluginConfigFromExternalV2,
	)
	if err != nil {
		return nil, err
	}
	return &generateConfig{
		cleanPluginOuts:       externalFile.Clean,
		generateManagedConfig: generateManagedConfig,
		generatePluginConfigs: generatePluginConfigs,
	}, nil
}

func (g *generateConfig) CleanPluginOuts() bool {
	return g.cleanPluginOuts
}

func (g *generateConfig) GeneratePluginConfigs() []GeneratePluginConfig {
	return g.generatePluginConfigs
}

func (g *generateConfig) GenerateManagedConfig() GenerateManagedConfig {
	return g.generateManagedConfig
}

func (g *generateConfig) GenerateTypeConfig() GenerateTypeConfig {
	return g.generateTypeConfig
}

func (*generateConfig) isGenerateConfig() {}

func newNoPluginsError() error {
	return errors.New("must specify at least one plugin")
}

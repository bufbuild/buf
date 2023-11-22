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

// GenerateTypeConfig is a type filter configuration.
type GenerateTypeConfig interface {
	// If IncludeTypes returns a non-empty list, it means that only those types are
	// generated. Otherwise all types are generated.
	IncludeTypes() []string

	isGenerateTypeConfig()
}

// *** PRIVATE ***

type generateConfig struct{}

func newGenerateConfig() *generateConfig {
	return &generateConfig{}
}

func (*generateConfig) isGenerateConfig() {}

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
//
// TODO
type GenerateConfig interface {
	GeneratePluginConfigs() []GeneratePluginConfig
	// may be nil
	GenerateManagedConfig() GenerateManagedConfig
	// may be empty
	// will always be empty in v2
	// TODO: we may need a way to attach inputs to make this consistent, but
	// can deal with that for v2.
	//GenerateInputConfigs() []GenerateInputConfig

	// may be nil
	// will always be nil in v2
	GenerateTypeConfig() GenerateTypeConfig
	isGenerateConfig()
}

type GeneratePluginConfig interface {
	Plugin() string
	Revision() int
	Out() string
	// TODO define enum in same pattern as FileVersion
	// GenerateStrategy() GenerateStrategy
	// TODO finish
	// TODO: figure out what to do with TypesConfig
	isGeneratePluginConfig()
}

type GenerateManagedConfig interface {
	// second value is whether or not this was present
	CCEnableArenas() (bool, bool)
	// TODO finish
	isGenerateManagedConfig()
}

//type GenerateInputConfig interface {
//isGenerateInputConfig()
//}

type GenerateTypeConfig interface {
	isGenerateTypeConfig()
}

// *** PRIVATE ***

type generateConfig struct{}

func newGenerateConfig() *generateConfig {
	return &generateConfig{}
}

func (*generateConfig) isGenerateConfig() {}

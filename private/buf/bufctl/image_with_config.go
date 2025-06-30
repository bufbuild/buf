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

package bufctl

import (
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
)

type imageWithConfig struct {
	bufimage.Image

	moduleFullName bufparse.FullName
	moduleOpaqueID string
	lintConfig     bufconfig.LintConfig
	breakingConfig bufconfig.BreakingConfig
	pluginConfigs  []bufconfig.PluginConfig
	policyConfigs  []bufconfig.PolicyConfig
}

func newImageWithConfig(
	image bufimage.Image,
	moduleFullName bufparse.FullName,
	moduleOpaqueID string,
	lintConfig bufconfig.LintConfig,
	breakingConfig bufconfig.BreakingConfig,
	pluginConfigs []bufconfig.PluginConfig,
	policyConfigs []bufconfig.PolicyConfig,
) *imageWithConfig {
	return &imageWithConfig{
		Image:          image,
		moduleFullName: moduleFullName,
		moduleOpaqueID: moduleOpaqueID,
		lintConfig:     lintConfig,
		breakingConfig: breakingConfig,
		pluginConfigs:  pluginConfigs,
		policyConfigs:  policyConfigs,
	}
}

func (i *imageWithConfig) ModuleFullName() bufparse.FullName {
	return i.moduleFullName
}

func (i *imageWithConfig) ModuleOpaqueID() string {
	return i.moduleOpaqueID
}

func (i *imageWithConfig) LintConfig() bufconfig.LintConfig {
	return i.lintConfig
}

func (i *imageWithConfig) BreakingConfig() bufconfig.BreakingConfig {
	return i.breakingConfig
}

func (i *imageWithConfig) PluginConfigs() []bufconfig.PluginConfig {
	return i.pluginConfigs
}

func (i *imageWithConfig) PolicyConfigs() []bufconfig.PolicyConfig {
	return i.policyConfigs
}

func (*imageWithConfig) isImageWithConfig() {}

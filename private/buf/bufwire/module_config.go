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

package bufwire

import (
	"github.com/bufbuild/buf/private/buf/bufconfig"
	"github.com/bufbuild/buf/private/buf/bufwork"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
)

type moduleConfig struct {
	module          bufmodule.Module
	config          *bufconfig.Config
	workspace       bufmodule.Workspace
	workspaceConfig *bufwork.Config
}

func newModuleConfig(
	module bufmodule.Module,
	config *bufconfig.Config,
	workspace bufmodule.Workspace,
	workspaceConfig *bufwork.Config,
) *moduleConfig {
	return &moduleConfig{
		module:          module,
		config:          config,
		workspace:       workspace,
		workspaceConfig: workspaceConfig,
	}
}

func (m *moduleConfig) Module() bufmodule.Module {
	return m.module
}

func (m *moduleConfig) Config() *bufconfig.Config {
	return m.config
}

func (m *moduleConfig) Workspace() bufmodule.Workspace {
	return m.workspace
}

func (m *moduleConfig) WorkspaceConfig() *bufwork.Config {
	return m.workspaceConfig
}

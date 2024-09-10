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

package bufcheck

import (
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"pluginrpc.com/pluginrpc"
)

type plugin struct {
	pluginrpc.Runner

	pluginConfig bufconfig.PluginConfig
}

func newPlugin(
	pluginConfig bufconfig.PluginConfig,
	runner pluginrpc.Runner,
) *plugin {
	return &plugin{
		Runner:       runner,
		pluginConfig: pluginConfig,
	}
}

func (p *plugin) Config() bufconfig.PluginConfig {
	return p.pluginConfig
}

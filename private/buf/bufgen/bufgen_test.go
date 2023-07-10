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

package bufgen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPluginConfig_GetRemoteHostname(t *testing.T) {
	t.Parallel()
	assertPluginConfigRemoteHostname := func(config *PluginConfig, expected string) {
		t.Helper()
		assert.Equal(t, config.GetRemoteHostname(), expected)
	}
	assertPluginConfigRemoteHostname(&PluginConfig{Plugin: "buf.build/protocolbuffers/go:v1.28.1"}, "buf.build")
	assertPluginConfigRemoteHostname(&PluginConfig{Plugin: "buf.build/protocolbuffers/go"}, "buf.build")
	assertPluginConfigRemoteHostname(&PluginConfig{Remote: "buf.build/protocolbuffers/plugins/go:v1.28.1-1"}, "buf.build")
	assertPluginConfigRemoteHostname(&PluginConfig{Remote: "buf.build/protocolbuffers/plugins/go"}, "buf.build")
}

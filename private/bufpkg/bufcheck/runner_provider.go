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
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/pkg/pluginrpcutil"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/wasm"
	"pluginrpc.com/pluginrpc"
)

type runnerProvider struct {
	wasmRuntime wasm.Runtime
	readBucket  storage.ReadBucket
}

func newRunnerProvider(
	wasmRuntime wasm.Runtime,
) *runnerProvider {
	return &runnerProvider{
		wasmRuntime: wasmRuntime,
	}
}

func (r *runnerProvider) NewRunner(pluginConfig bufconfig.PluginConfig) (pluginrpc.Runner, error) {
	switch pluginConfig.Type() {
	case bufconfig.PluginConfigTypeLocal:
		return pluginrpcutil.NewLocalRunner(
			pluginConfig.Name(),
			pluginConfig.Args()...,
		), nil
	case bufconfig.PluginConfigTypeLocalWasm:
		return pluginrpcutil.NewLocalWasmRunner(
			r.wasmRuntime,
			pluginConfig.Name(),
			pluginConfig.Args()...,
		), nil
	case bufconfig.PluginConfigTypeRemoteWasm:
		return nil, fmt.Errorf("remote plugins are not supported")
	default:
		return nil, syserror.Newf("unknown PluginConfigType: %v", pluginConfig.Type())
	}
}

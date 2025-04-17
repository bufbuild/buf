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

package bufcheck

import (
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	"github.com/bufbuild/buf/private/pkg/pluginrpcutil"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/wasm"
	"pluginrpc.com/pluginrpc"
)

type localRunnerProvider struct {
	wasmRuntime wasm.Runtime
}

func newLocalRunnerProvider(
	wasmRuntime wasm.Runtime,
) *localRunnerProvider {
	return &localRunnerProvider{
		wasmRuntime: wasmRuntime,
	}
}

func (r *localRunnerProvider) NewRunner(plugin bufplugin.Plugin) (pluginrpc.Runner, error) {
	switch isLocal, isWasm := plugin.IsLocal(), plugin.IsWasm(); {
	case isLocal && !isWasm:
		return pluginrpcutil.NewLocalRunner(
			plugin.Name(),
			plugin.Args()...,
		), nil
	case isWasm:
		return pluginrpcutil.NewWasmRunner(
			r.wasmRuntime,
			plugin.Data,
			plugin.Name(),
			plugin.Args()...,
		), nil
	default:
		return nil, syserror.Newf("unsupported Plugin runtime: %v", plugin.OpaqueID())
	}
}

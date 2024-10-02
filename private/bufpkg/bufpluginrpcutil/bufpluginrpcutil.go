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

package bufpluginrpcutil

import (
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/wasm"
	"pluginrpc.com/pluginrpc"
)

// NewRunner returns a new pluginrpc.Runner for the command.Runner and program name.
func NewRunner(delegate command.Runner, programName string, programArgs ...string) pluginrpc.Runner {
	return newRunner(delegate, programName, programArgs...)
}

// NewWasmRunner returns a new pluginrpc.Runner for the wasm.Runtime and program name.
//
// This runner is used for local Wasm plugins. The program name is the path to the Wasm file.
func NewWasmRunner(wasmRuntime wasm.Runtime, programName string, programArgs ...string) pluginrpc.Runner {
	return newWasmRunner(wasmRuntime, programName, programArgs...)
}

// RunnerProvider provides pluginrpc.Runners for the PluginConfig.
//
// RunnerProvider selects the correct runner based on the PluginConfig.Type.
type RunnerProvider interface {
	NewRunner(pluginConfig bufconfig.PluginConfig) (pluginrpc.Runner, error)
}

// NewRunnerProvider returns a new RunnerProvider for the command.Runner and wasm.Runtime.
//
// This implementation should only be used for local applications. It is safe to
// use concurrently.
//
// The RunnerProvider selects the correct runner based on the PluginConfig.Type.
// The supported types are:
//   - bufconfig.PluginConfigTypeLocal
//   - bufconfig.PluginConfigTypeLocalWasm
//
// If the PluginConfig.Type is not supported, an error is returned.
func NewRunnerProvider(delegate command.Runner, wasmRuntime wasm.Runtime) RunnerProvider {
	return newRunnerProvider(delegate, wasmRuntime)
}

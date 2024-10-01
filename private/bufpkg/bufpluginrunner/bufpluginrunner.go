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

package bufpluginrunner

import (
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufwasm"
	"github.com/bufbuild/buf/private/pkg/command"
	"pluginrpc.com/pluginrpc"
)

// RunnerProvider provides pluginrpc.Runners for a given plugin config.
type RunnerProvider interface {
	NewRunner(pluginConfig bufconfig.PluginConfig) (pluginrpc.Runner, error)
}

// NewRunnerProvider returns a new RunnerProvider for the command.Runner.
//
// This implementation should only be used for local applications.
func NewRunnerProvider(
	delegate command.Runner,
	options ...RunnerProviderOption,
) RunnerProvider {
	return newRunnerProvider(delegate, options...)
}

// RunnerProviderOption is an option for NewRunnerProvider.
type RunnerProviderOption func(*runnerProviderOptions)

// RunnerProviderWithWasmRuntime returns a new RunnerProviderOption that
// specifies a Wasm runtime. This is required for local Wasm plugins.
func RunnerProviderWithWasmRuntime(wasmRuntime bufwasm.Runtime) RunnerProviderOption {
	return func(runnerProviderOptions *runnerProviderOptions) {
		runnerProviderOptions.wasmRuntime = wasmRuntime
	}
}

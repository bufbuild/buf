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
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufwasm"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/pluginrpcutil"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"pluginrpc.com/pluginrpc"
)

type runnerProvider struct {
	delegate    command.Runner
	wasmRuntime bufwasm.Runtime
}

func newRunnerProvider(delegate command.Runner, options ...RunnerProviderOption) *runnerProvider {
	runnerProviderOptions := newRunnerProviderOptions()
	for _, option := range options {
		option(runnerProviderOptions)
	}
	return &runnerProvider{
		delegate:    delegate,
		wasmRuntime: runnerProviderOptions.wasmRuntime,
	}
}

func (r *runnerProvider) NewRunner(pluginConfig bufconfig.PluginConfig) (pluginrpc.Runner, error) {
	switch pluginConfig.Type() {
	case bufconfig.PluginConfigTypeLocal:
		path := pluginConfig.Path()
		return pluginrpcutil.NewRunner(
			r.delegate,
			// We know that Path is of at least length 1.
			path[0],
			path[1:]...,
		), nil
	case bufconfig.PluginConfigTypeLocalWasm:
		if r.wasmRuntime == nil {
			return nil, syserror.Newf("wasm runtime is required for local wasm plugins")
		}
		return newWasmRunner(
			r.wasmRuntime,
			pluginConfig.Name(),
		), nil
	default:
		return nil, syserror.Newf("unsupported plugin type: %v", pluginConfig.Type())
	}
}

type runnerProviderOptions struct {
	wasmRuntime bufwasm.Runtime
}

func newRunnerProviderOptions() *runnerProviderOptions {
	return &runnerProviderOptions{
		wasmRuntime: bufwasm.NewUnimplementedRuntime(),
	}
}

// wasmRunner is a runner that loads a Wasm plugin.
type wasmRunner struct {
	programName string
	wasmRuntime bufwasm.Runtime
	// Once protects plugin and pluginErr.
	once      sync.Once
	plugin    bufwasm.Plugin
	pluginErr error
}

// newWasmRunner returns a new pluginrpc.Runner for the Wasm binary on a
// bufwasm.Runtime and program name. This runner is only suitable for use with
// short-lived programs, compiled plugins lifetime is tied to the runtime.
//
// The program name should be the name of the program as it appears in the
// plugin config. The runner will call os.GetEnv("PATH") with os.Stat on each
// directory and file to find the program. This is similar to exec.LookPath
// but does not require the file to be executable. This is only safe for use
// in the CLI, as it is not safe to use in a server environment.
func newWasmRunner(
	runtime bufwasm.Runtime,
	programName string,
) *wasmRunner {
	return &wasmRunner{
		programName: programName,
		wasmRuntime: runtime,
	}
}

func (r *wasmRunner) Run(ctx context.Context, env pluginrpc.Env) (retErr error) {
	plugin, err := r.loadPluginOnce(ctx)
	if err != nil {
		return err
	}
	return plugin.Run(ctx, env)
}

func (r *wasmRunner) loadPluginOnce(ctx context.Context) (bufwasm.Plugin, error) {
	r.once.Do(func() {
		r.plugin, r.pluginErr = r.loadPlugin(ctx)
	})
	return r.plugin, r.pluginErr
}

func (r *wasmRunner) loadPlugin(ctx context.Context) (bufwasm.Plugin, error) {
	// Find the plugin path. We use the same logic as exec.LookPath, but we do
	// not require the file to be executable. So check the local directory
	// first before checking the PATH.
	var path string
	if fileInfo, err := os.Stat(r.programName); err == nil && !fileInfo.IsDir() {
		path = r.programName
	} else {
		var err error
		path, err = unsafeLookPath(r.programName)
		if err != nil {
			return nil, fmt.Errorf("could not find plugin %q in PATH: %v", r.programName, err)
		}
	}
	wasmModule, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Compile and run, releasing the plugin at the end.
	plugin, err := r.wasmRuntime.Compile(ctx, r.programName, wasmModule)
	if err != nil {
		return nil, err
	}
	// This plugin is never released, so subsequent calls to this function
	// will benefit from the cached plugin. This is only safe as the
	// runner is limited to the CLI.
	return plugin, nil
}

// unsafeLookPath is a wrapper around exec.LookPath that restores the original
// pre-Go 1.19 behavior of resolving queries that would use relative PATH
// entries. We consider it acceptable for the use case of locating plugins.
//
// https://pkg.go.dev/os/exec#hdr-Executables_in_the_current_directory
func unsafeLookPath(file string) (string, error) {
	path, err := exec.LookPath(file)
	if errors.Is(err, exec.ErrDot) {
		err = nil
	}
	return path, err
}

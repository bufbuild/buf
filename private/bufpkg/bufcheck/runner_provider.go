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
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	case bufconfig.PluginConfigTypeLocalWASM:
		if r.wasmRuntime == nil {
			return nil, syserror.Newf("wasm runtime is required for local wasm plugins")
		}
		return newWASMRunner(
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

// wasmRunner is a runner that loads a WASM plugin.
type wasmRunner struct {
	programName string
	wasmRuntime bufwasm.Runtime
	// Once protects plugin and pluginErr.
	once      sync.Once
	plugin    bufwasm.Plugin
	pluginErr error
}

// newWASMRunner returns a new pluginrpc.Runner for the WASM binary on a
// bufwasm.Runtime and program name. This runner is only suitable for use with
// short-lived programs, compiled plugins lifetime is tied to the runtime.
//
// The program name should be the name of the program as it appears in the
// plugin config. The runner will call os.GetEnv("PATH") with os.Stat on each
// directory and file to find the program. This is similar to exec.LookPath
// but does not require the file to be executable. This is only safe for use
// in the CLI, as it is not safe to use in a server environment.
func newWASMRunner(
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
	path, err := lookPath(r.programName)
	if err != nil {
		return nil, fmt.Errorf("could not find plugin %q in PATH: %v", r.programName, err)
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

// lookPath looks for a wasm file in the PATH, not only an executable. This
// doesn't use exec.LookPath to avoid requiring an executable bit.
func lookPath(file string) (string, error) {
	// First, check in the current directory.
	if ok, err := findFile(file); err != nil {
		return "", err
	} else if ok {
		return file, nil
	}
	// If the file has a path separator, fail early.
	if strings.Contains(file, string(os.PathSeparator)) {
		return "", os.ErrNotExist
	}
	path := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(path) {
		if dir == "" {
			// Unix shell semantics: path element "" means "."
			dir = "."
		}
		path := filepath.Join(dir, file)
		if ok, err := findFile(path); err != nil {
			return "", err
		} else if ok {
			return path, nil
		}
	}
	return "", os.ErrNotExist
}

func findFile(file string) (bool, error) {
	d, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return !d.Mode().IsDir(), nil
}

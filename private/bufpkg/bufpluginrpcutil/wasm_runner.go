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
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/bufbuild/buf/private/pkg/wasm"
	"pluginrpc.com/pluginrpc"
)

// wasmRunner is a runner that loads a Wasm plugin.
type wasmRunner struct {
	programName string
	wasmRuntime wasm.Runtime
	// Once protects compiledModule and compiledModuleErr.
	once              sync.Once
	compiledModule    wasm.CompiledModule
	compiledModuleErr error
}

// newWasmRunner returns a new pluginrpc.Runner for the Wasm binary on a
// wasm.Runtime and program name. This runner is only suitable for use with
// short-lived programs, compiled plugins lifetime is tied to the runtime.
//
// The program name should be the name of the program as it appears in the
// plugin config. The runner will check the current directory and fallback to
// exec.LookPath to find the program in the PATH. This is only safe for use in
// the CLI, as it is not safe to use in a server environment.
func newWasmRunner(
	runtime wasm.Runtime,
	programName string,
) *wasmRunner {
	return &wasmRunner{
		programName: programName,
		wasmRuntime: runtime,
	}
}

func (r *wasmRunner) Run(ctx context.Context, env pluginrpc.Env) (retErr error) {
	compiledModule, err := r.loadCompiledModuleOnce(ctx)
	if err != nil {
		return err
	}
	return compiledModule.Run(ctx, env)
}

func (r *wasmRunner) loadCompiledModuleOnce(ctx context.Context) (wasm.CompiledModule, error) {
	r.once.Do(func() {
		r.compiledModule, r.compiledModuleErr = r.loadCompiledModule(ctx)
	})
	return r.compiledModule, r.compiledModuleErr
}

func (r *wasmRunner) loadCompiledModule(ctx context.Context) (wasm.CompiledModule, error) {
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
	// Compile and run, releasing the compiledModule at the end.
	compiledModule, err := r.wasmRuntime.Compile(ctx, r.programName, wasmModule)
	if err != nil {
		return nil, err
	}
	// This plugin is never released, so subsequent calls to this function
	// will benefit from the cached plugin. This is only safe as the
	// runner is limited to the CLI.
	return compiledModule, nil
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

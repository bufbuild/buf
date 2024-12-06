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

package pluginrpcutil

import (
	"context"
	"slices"
	"sync"

	"github.com/bufbuild/buf/private/pkg/wasm"
	"pluginrpc.com/pluginrpc"
)

type wasmRunner struct {
	delegate    wasm.Runtime
	getData     func() ([]byte, error)
	programName string
	programArgs []string
	// lock protects compiledModule and compiledModuleErr. Store called as
	// a boolean to avoid nil comparison.
	lock              sync.RWMutex
	called            bool
	compiledModule    wasm.CompiledModule
	compiledModuleErr error
}

func newWasmRunner(
	delegate wasm.Runtime,
	getData func() ([]byte, error),
	programName string,
	programArgs ...string,
) *wasmRunner {
	return &wasmRunner{
		delegate:    delegate,
		getData:     getData,
		programName: programName,
		programArgs: programArgs,
	}
}

func (r *wasmRunner) Run(ctx context.Context, env pluginrpc.Env) (retErr error) {
	compiledModule, err := r.loadCompiledModuleOnce(ctx)
	if err != nil {
		return err
	}
	if len(r.programArgs) > 0 {
		env.Args = append(slices.Clone(r.programArgs), env.Args...)
	}
	return compiledModule.Run(ctx, env)
}

func (r *wasmRunner) loadCompiledModuleOnce(ctx context.Context) (wasm.CompiledModule, error) {
	r.lock.RLock()
	if r.called {
		r.lock.RUnlock()
		return r.compiledModule, r.compiledModuleErr
	}
	r.lock.RUnlock()
	r.lock.Lock()
	defer r.lock.Unlock()
	if !r.called {
		r.compiledModule, r.compiledModuleErr = r.loadCompiledModule(ctx)
		r.called = true
	}
	return r.compiledModule, r.compiledModuleErr
}

func (r *wasmRunner) loadCompiledModule(ctx context.Context) (wasm.CompiledModule, error) {
	moduleWasm, err := r.getData()
	if err != nil {
		return nil, err
	}
	// Compile the module. This CompiledModule is never released, so
	// subsequent calls to this function will benefit from the cached
	// module. This is only safe as the runner is limited to the CLI.
	compiledModule, err := r.delegate.Compile(ctx, r.programName, moduleWasm)
	if err != nil {
		return nil, err
	}
	return compiledModule, nil
}

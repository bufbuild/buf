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

package wasm

import (
	"context"
	"fmt"

	"github.com/tetratelabs/wazero"
	"pluginrpc.com/pluginrpc"
)

type compiledModule struct {
	moduleName     string
	runtime        wazero.Runtime
	compiledModule wazero.CompiledModule
}

func (p *compiledModule) Run(ctx context.Context, env pluginrpc.Env) error {
	// Create a new module wazeroModuleConfig with the given environment.
	wazeroModuleConfig := wazero.NewModuleConfig().
		WithStdin(env.Stdin).
		WithStdout(env.Stdout).
		WithStderr(env.Stderr).
		// Use an empty name to allow for multiple instances of the same module.
		// See wazero.ModuleConfig.WithName.
		WithName("").
		// Use the program name as the first argument to replicate the
		// behavior of os.Exec.
		// See wazero.ModuleConfig.WithArgs.
		WithArgs(append([]string{p.moduleName}, env.Args...)...)

	// Instantiate the Wasm module into the runtime. This effectively runs
	// the Wasm module. Only the effect of instantiating the module is used,
	// the module is closed immediately after running to free up resources.
	// See https://github.com/tetratelabs/wazero/issues/985.
	wazeroModule, err := p.runtime.InstantiateModule(
		ctx,
		p.compiledModule,
		wazeroModuleConfig,
	)
	if err != nil {
		return fmt.Errorf("failed to instantiate module: %w", err)
	}
	if err := wazeroModule.Close(ctx); err != nil {
		return fmt.Errorf("failed to close module: %w", err)
	}
	return nil
}

func (p *compiledModule) Close(ctx context.Context) error {
	return p.compiledModule.Close(ctx)
}

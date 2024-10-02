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

var _ CompiledModule = (*compiledModule)(nil)

func (p *compiledModule) Run(ctx context.Context, env pluginrpc.Env) error {
	// Create a new module config with the given environment.
	config := wazero.NewModuleConfig().
		WithStdin(env.Stdin).
		WithStdout(env.Stdout).
		WithStderr(env.Stderr)

	// Instantiate the guest wasm module into the same runtime.
	// See: https://github.com/tetratelabs/wazero/issues/985
	mod, err := p.runtime.InstantiateModule(
		ctx,
		p.compiledModule,
		// Use an empty name to allow for multiple instances of the same module.
		// See wazero.ModuleConfig.WithName.
		config.WithName("").WithArgs(
			append([]string{p.moduleName}, env.Args...)...,
		),
	)
	if err != nil {
		return fmt.Errorf("failed to instantiate module: %w", err)
	}
	if err := mod.Close(ctx); err != nil {
		return fmt.Errorf("failed to close module: %w", err)
	}
	return nil
}

func (p *compiledModule) Close(ctx context.Context) error {
	return p.compiledModule.Close(ctx)
}

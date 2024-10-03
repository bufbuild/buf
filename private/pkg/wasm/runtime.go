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
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"go.uber.org/multierr"
)

const (
	// defaultMaxMemoryBytes is the maximum memory size in bytes.
	defaultMaxMemoryBytes = 1 << 29 // 512 MiB
	// wasmPageSize is the page size in bytes.
	wasmPageSize = 1 << 16 // 64 KiB
)

type runtime struct {
	runtime          wazero.Runtime
	compilationCache wazero.CompilationCache
}

func newRuntime(ctx context.Context, options ...RuntimeOption) (*runtime, error) {
	runtimeOptions := newRuntimeOptions()
	for _, option := range options {
		option(runtimeOptions)
	}
	if runtimeOptions.maxMemoryBytes == 0 {
		return nil, fmt.Errorf("Wasm max memory bytes must be greater than 0")
	}
	// The maximum memory size is limited to 4 GiB. Sizes less than the page
	// size (64 KiB) are truncated. memoryLimitPages is guaranteed to be
	// below 2^16 as the maxium uint32 value is 2^32 - 1.
	// NOTE: The option represented as a uint32 restricts the max number of
	// pages to 2^16-1, one less the the actual max value of 2^16. But this
	// is a nicer API then specifying the max number of pages directly.
	memoryLimitPages := runtimeOptions.maxMemoryBytes / wasmPageSize
	if memoryLimitPages == 0 {
		return nil, fmt.Errorf("Wasm max memory bytes %d is too small", runtimeOptions.maxMemoryBytes)
	}

	// Create the wazero.RuntimeConfig with enforceable limits. Limits are
	// enforced through the Wasm sandbox. The following limits are set:
	//  - Memory limit: The maximum memory size in pages.
	//  - CPU limit: The runtime stops work on context done.
	//  - Access limit: All system interfaces are stubbed. No network,
	//    disk, clock, etc.
	// See wazero.NewRuntimeConfig for more details.
	wazeroRuntimeConfig := wazero.NewRuntimeConfig().
		WithCoreFeatures(api.CoreFeaturesV2).
		WithCloseOnContextDone(true).
		WithMemoryLimitPages(memoryLimitPages)
	var wazeroCompilationCache wazero.CompilationCache
	if runtimeOptions.cacheDir != "" {
		var err error
		wazeroCompilationCache, err = wazero.NewCompilationCacheWithDir(runtimeOptions.cacheDir)
		if err != nil {
			return nil, fmt.Errorf("failed to create compilation cache: %w", err)
		}
		wazeroRuntimeConfig = wazeroRuntimeConfig.WithCompilationCache(wazeroCompilationCache)
	}
	wazeroRuntime := wazero.NewRuntimeWithConfig(ctx, wazeroRuntimeConfig)

	// Init WASI preview1 APIs. This is required to support the pluginrpc
	// protocol. The returned closer method is discarded as the
	// instantiated module is never required to be unloaded.
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, wazeroRuntime); err != nil {
		return nil, fmt.Errorf("failed to instantiate WASI snapshot preview1: %w", err)
	}
	return &runtime{
		runtime:          wazeroRuntime,
		compilationCache: wazeroCompilationCache,
	}, nil
}

func (r *runtime) Compile(ctx context.Context, moduleName string, moduleWasm []byte) (CompiledModule, error) {
	if moduleName == "" {
		// The plugin is required to be named. We cannot use the name
		// from the Wasm binary as this is not guaranteed to be set and
		// may conflict with the provided name.
		return nil, fmt.Errorf("name is empty")
	}
	// Compile the WebAssembly. This operation is hashed on the module
	// bytes and the runtime configuration. The compiled module is
	// cached in memory and on disk if an optional cache directory is
	// provided.
	wazeroCompiledModule, err := r.runtime.CompileModule(ctx, moduleWasm)
	if err != nil {
		return nil, err
	}
	return &compiledModule{
		moduleName:     moduleName,
		runtime:        r.runtime,
		compiledModule: wazeroCompiledModule,
	}, nil
}

func (r *runtime) Close(ctx context.Context) error {
	err := r.runtime.Close(ctx)
	if r.compilationCache != nil {
		err = multierr.Append(err, r.compilationCache.Close(ctx))
	}
	return err
}

type runtimeOptions struct {
	maxMemoryBytes uint32
	cacheDir       string
}

func newRuntimeOptions() *runtimeOptions {
	return &runtimeOptions{
		maxMemoryBytes: defaultMaxMemoryBytes,
	}
}

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

	"github.com/bufbuild/buf/private/pkg/syserror"
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
	runtime wazero.Runtime
	cache   wazero.CompilationCache
}

var _ Runtime = (*runtime)(nil)

func newRuntime(ctx context.Context, options ...RuntimeOption) (*runtime, error) {
	var cfg runtimeConfig
	for _, opt := range options {
		opt.apply(&cfg)
	}
	// Create the runtime config with enforceable limits.
	runtimeConfig := wazero.NewRuntimeConfig().
		WithCoreFeatures(api.CoreFeaturesV2).
		WithCloseOnContextDone(true).
		WithMemoryLimitPages(cfg.getMaxMemoryBytes() / wasmPageSize)
	var cache wazero.CompilationCache
	if cfg.cacheDir != "" {
		var err error
		cache, err = wazero.NewCompilationCacheWithDir(cfg.cacheDir)
		if err != nil {
			return nil, fmt.Errorf("failed to create compilation cache: %w", err)
		}
		runtimeConfig = runtimeConfig.WithCompilationCache(cache)
	}
	r := wazero.NewRuntimeWithConfig(ctx, runtimeConfig)

	// Init wasi.
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	return &runtime{
		runtime: r,
		cache:   cache,
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
	compiledModulePlugin, err := r.runtime.CompileModule(ctx, moduleWasm)
	if err != nil {
		return nil, err
	}
	return &compiledModule{
		moduleName:     moduleName,
		runtime:        r.runtime,
		compiledModule: compiledModulePlugin,
	}, nil
}

func (r *runtime) Release(ctx context.Context) error {
	err := r.runtime.Close(ctx)
	if r.cache != nil {
		err = multierr.Append(err, r.cache.Close(ctx))
	}
	return err
}

type runtimeConfig struct {
	maxMemoryBytes uint32
	cacheDir       string
}

func (r *runtimeConfig) getMaxMemoryBytes() uint32 {
	if r.maxMemoryBytes == 0 {
		return defaultMaxMemoryBytes
	}
	return r.maxMemoryBytes
}

type runtimeOptionFunc func(*runtimeConfig)

func (f runtimeOptionFunc) apply(cfg *runtimeConfig) {
	f(cfg)
}

var _ RuntimeOption = runtimeOptionFunc(nil)

type unimplementedRuntime struct{}

var _ Runtime = unimplementedRuntime{}

func (unimplementedRuntime) Compile(ctx context.Context, name string, moduleBytes []byte) (CompiledModule, error) {
	return nil, syserror.Newf("not implemented")
}
func (unimplementedRuntime) Release(ctx context.Context) error {
	return nil
}

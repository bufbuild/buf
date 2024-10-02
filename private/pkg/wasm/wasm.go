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

// Package wasm provides a Wasm runtime for plugins.
package wasm

import (
	"context"

	"pluginrpc.com/pluginrpc"
)

// CompiledModule is a Wasm module ready to be run.
//
// It is safe to use this module concurrently. When done, call Release to free
// resources held by the CompiledModule. All CompiledModules created by the
// same Runtime will be invalidated when the Runtime is released.
//
// Memory is limited by the runtime. To restrict CPU usage, cancel the context.
type CompiledModule interface {
	pluginrpc.Runner
	// Release releases all resources held by the compiled module.
	Release(ctx context.Context) error
}

// Runtime is a Wasm runtime.
//
// It is safe to use the Runtime concurrently. Release must be called when done
// with the Runtime to free resources. All CompiledModules created by the same
// Runtime will be invalidated when the Runtime is released.
type Runtime interface {
	// Compile compiles the given Wasm module bytes into a CompiledModule.
	Compile(ctx context.Context, moduleName string, moduleWasm []byte) (CompiledModule, error)
	// Release releases all resources held by the Runtime.
	Release(ctx context.Context) error
}

// NewRuntime creates a new Wasm runtime.
func NewRuntime(ctx context.Context, options ...RuntimeOption) (Runtime, error) {
	return newRuntime(ctx, options...)
}

// RuntimeOption is an option for Runtime.
type RuntimeOption func(*runtimeOptions)

// WithMaxMemoryBytes sets the maximum memory size in bytes.
func WithMaxMemoryBytes(maxMemoryBytes uint32) RuntimeOption {
	return func(runtimeOptions *runtimeOptions) {
		runtimeOptions.maxMemoryBytes = maxMemoryBytes
	}
}

// WithLocalCacheDir sets the local cache directory.
//
// The cache directory is used to store compiled Wasm modules. This can be used
// to speed up subsequent runs of the same module. The internal cache structure
// and versioning is handled by the runtime.
//
// This option is only safe use in CLI environments.
func WithLocalCacheDir(cacheDir string) RuntimeOption {
	return func(runtimeOptions *runtimeOptions) {
		runtimeOptions.cacheDir = cacheDir
	}
}

// UnimplementedRuntime is an unimplemented Runtime.
var UnimplementedRuntime = unimplementedRuntime{}

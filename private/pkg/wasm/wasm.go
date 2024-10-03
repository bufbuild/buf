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

// UnimplementedRuntime is an unimplemented Runtime.
var UnimplementedRuntime = unimplementedRuntime{}

// CompiledModule is a Wasm module ready to be run.
//
// It is safe to use this module concurrently. When done, call Close to free
// resources held by the CompiledModule. All CompiledModules created by the
// same Runtime will be invalidated when the Runtime is closed.
//
// Memory is limited by the Runtime. To restrict CPU usage, cancel the context.
type CompiledModule interface {
	pluginrpc.Runner
	// Close releases all resources held by the compiled module.
	Close(ctx context.Context) error
}

// Runtime is a Wasm runtime.
//
// It is safe to use the Runtime concurrently. Close must be called when done
// with the Runtime to free resources. All CompiledModules created by the same
// Runtime will be invalidated when the Runtime is closed.
type Runtime interface {
	// Compile compiles the given Wasm module bytes into a CompiledModule.
	Compile(ctx context.Context, moduleName string, moduleWasm []byte) (CompiledModule, error)
	// Close releases all resources held by the Runtime.
	Close(ctx context.Context) error
}

// NewRuntime creates a new Wasm Runtime.
func NewRuntime(ctx context.Context, options ...RuntimeOption) (Runtime, error) {
	return newRuntime(ctx, options...)
}

// RuntimeOption is an option for Runtime.
type RuntimeOption func(*runtimeOptions)

// WithMaxMemoryBytes sets the maximum memory size in bytes.
//
// The maximuim memory size is limited to 4 GiB. The default is 512 MiB. Sizes
// less then the page size (64 KiB) are truncated.
func WithMaxMemoryBytes(maxMemoryBytes uint32) RuntimeOption {
	return func(runtimeOptions *runtimeOptions) {
		runtimeOptions.maxMemoryBytes = maxMemoryBytes
	}
}

// WithLocalCacheDir sets the local cache directory.
//
// The cache directory is used to store compiled Wasm modules. This can be used
// to speed up subsequent runs of the same module. The internal cache structure
// and versioning is handled by the Runtime.
//
// This option is only safe use in CLI environments.
func WithLocalCacheDir(cacheDir string) RuntimeOption {
	return func(runtimeOptions *runtimeOptions) {
		runtimeOptions.cacheDir = cacheDir
	}
}

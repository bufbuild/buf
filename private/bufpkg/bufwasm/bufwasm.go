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

// Package bufwasm provides a Wasm runtime for plugins.
package bufwasm

import (
	"context"

	"pluginrpc.com/pluginrpc"
)

// CompiledModule is a Wasm module ready to be run.
//
// It is safe to use this module concurrently. Ensure that you call [Release]
// to free resources associated with the module.
//
// Memory is limited by the runtime. To restrict CPU usage, cancel the context.
type CompiledModule interface {
	pluginrpc.Runner
	// PluginName returns the name of the plugin.
	PluginName() string
	// Release releases all resources held by the compiled module.
	Release(ctx context.Context) error
}

// Runtime is a Wasm runtime.
//
// It is safe to use this runtime concurrently. Ensure that you call [Release]
// when you are done with the runtime. All plugins created by this runtime will
// be invalidated when [Release] is called.
type Runtime interface {
	// Compile compiles the given Wasm module bytes into a [CompiledModule].
	Compile(ctx context.Context, pluginName string, pluginWasm []byte) (CompiledModule, error)
	// Release releases all resources held by the runtime.
	Release(ctx context.Context) error
}

// NewRuntime creates a new Wasm runtime.
func NewRuntime(ctx context.Context, options ...RuntimeOption) (Runtime, error) {
	return newRuntime(ctx, options...)
}

// RuntimeOption is an option for [NewRuntime].
type RuntimeOption interface {
	apply(*runtimeConfig)
}

// WithMaxMemoryBytes sets the maximum memory size in bytes.
func WithMaxMemoryBytes(maxMemoryBytes uint32) RuntimeOption {
	return runtimeOptionFunc(func(cfg *runtimeConfig) {
		cfg.maxMemoryBytes = maxMemoryBytes
	})
}

// WithLocalCacheDir sets the local cache directory.
//
// This option is only safe use in CLI environments.
func WithLocalCacheDir(cacheDir string) RuntimeOption {
	return runtimeOptionFunc(func(cfg *runtimeConfig) {
		cfg.cacheDir = cacheDir
	})
}

// UnimplementedRuntime returns a new unimplemented Runtime.
var UnimplementedRuntime = unimplementedRuntime{}

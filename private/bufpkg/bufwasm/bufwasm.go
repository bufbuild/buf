// Copyright 2020-2023 Buf Technologies, Inc.
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

package bufwasm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing/fstest"

	wasmpluginv1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/wasmplugin/v1"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental/gojs"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
	"go.uber.org/multierr"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

// CustomSectionName is the name of the custom wasm section we look into for buf
// extensions.
const CustomSectionName = ".bufplugin"

// https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#sections%E2%91%A0
const customSectionID = 0

// maxMemoryBytes wazero memory page limit
const maxMemoryBytes = 1 << 29 // 512MB

// CompiledPlugin is the compiled representation of loading wasm bytes.
type CompiledPlugin struct {
	Module wazero.CompiledModule

	// Metadata parsed from custom sections of the wasm file. May be nil if
	// no buf specific sections were found.
	ExecConfig *wasmpluginv1.ExecConfig
}

func (c *CompiledPlugin) ABI() wasmpluginv1.WasmABI {
	if c.ExecConfig.GetWasmAbi() != wasmpluginv1.WasmABI_WASM_ABI_UNSPECIFIED {
		return c.ExecConfig.GetWasmAbi()
	}
	exportedFuncs := c.Module.ExportedFunctions()
	if _, ok := exportedFuncs["_start"]; ok {
		return wasmpluginv1.WasmABI_WASM_ABI_WASI_SNAPSHOT_PREVIEW1
	} else if _, ok := exportedFuncs["run"]; ok {
		return wasmpluginv1.WasmABI_WASM_ABI_GOJS
	}
	return wasmpluginv1.WasmABI_WASM_ABI_UNSPECIFIED
}

// PluginExecutionError is a wrapper for WASM plugin execution errors.
type PluginExecutionError struct {
	Stderr   string
	Exitcode uint32
}

// NewPluginExecutionError constructs a new execution error.
func NewPluginExecutionError(stderr string, exitcode uint32) *PluginExecutionError {
	return &PluginExecutionError{Stderr: strings.ToValidUTF8(stderr, ""), Exitcode: exitcode}
}

func (e *PluginExecutionError) Error() string {
	return "plugin exited with code " + strconv.Itoa(int(e.Exitcode))
}

// EncodeBufSection encodes the ExecConfig message as a custom wasm section.
// The resulting bytes can be appended to any valid wasm file to add the new
// section to that file.
func EncodeBufSection(config *wasmpluginv1.ExecConfig) ([]byte, error) {
	metadataBinary, err := protoencoding.NewWireMarshaler().Marshal(config)
	if err != nil {
		return nil, err
	}
	// Abusing the protowire package because the wasm file format is similar.
	return protowire.AppendBytes(
		[]byte{customSectionID},
		append(
			protowire.AppendString(
				nil,
				CustomSectionName,
			),
			metadataBinary...,
		),
	), nil
}

// PluginExecutor wraps a wazero end exposes functions to compile and run wasm plugins.
type PluginExecutor interface {
	CompilePlugin(ctx context.Context, plugin []byte) (_ *CompiledPlugin, retErr error)
	Run(ctx context.Context, plugin *CompiledPlugin, stdin io.Reader, stdout io.Writer) (retErr error)
}

type WASMPluginExecutor struct {
	runtimeConfig wazero.RuntimeConfig
}

// NewPluginExecutor creates a new pluginExecutor with a compilation cache dir
// and other buf defaults.
func NewPluginExecutor(compilationCacheDir string, options ...PluginExecutorOption) (*WASMPluginExecutor, error) {
	var cache wazero.CompilationCache
	if compilationCacheDir == "" {
		cache = wazero.NewCompilationCache()
	} else {
		var err error
		cache, err = wazero.NewCompilationCacheWithDir(compilationCacheDir)
		if err != nil {
			return nil, err
		}
	}
	runtimeConfig := wazero.NewRuntimeConfig().
		WithCoreFeatures(api.CoreFeaturesV2).
		WithMemoryLimitPages(maxMemoryBytes >> 16). // a page is 2^16 bytes
		WithCompilationCache(cache).
		WithCustomSections(true).
		WithCloseOnContextDone(true)
	executor := &WASMPluginExecutor{
		runtimeConfig: runtimeConfig,
	}
	for _, opt := range options {
		opt(executor)
	}
	return executor, nil
}

// PluginExecutorOption configuration options for the PluginExecutor.
type PluginExecutorOption func(*WASMPluginExecutor)

// WithMemoryLimitPages provides a custom per memory limit for a plugin
// executor. The default is 8192 pages for 512MB.
func WithMemoryLimitPages(memoryLimitPages uint32) PluginExecutorOption {
	return func(pluginExecutor *WASMPluginExecutor) {
		pluginExecutor.runtimeConfig = pluginExecutor.runtimeConfig.WithMemoryLimitPages(memoryLimitPages)
	}
}

// CompilePlugin takes a byte slice with a valid wasm module, compiles it and
// optionally reads out buf plugin metadata.
func (e *WASMPluginExecutor) CompilePlugin(ctx context.Context, plugin []byte) (_ *CompiledPlugin, retErr error) {
	runtime := wazero.NewRuntimeWithConfig(ctx, e.runtimeConfig)
	defer func() {
		retErr = multierr.Append(retErr, runtime.Close(ctx))
	}()
	// Note: before we start accepting user plugins, we should do more
	// validation on the metadata here: file path cleaning etc.
	compiledModule, err := runtime.CompileModule(ctx, plugin)
	if err != nil {
		return nil, fmt.Errorf("error compiling wasm: %w", err)
	}
	var bufsectionBytes []byte
	for _, section := range compiledModule.CustomSections() {
		if section.Name() == CustomSectionName {
			bufsectionBytes = append(bufsectionBytes, section.Data()...)
		}
	}
	compiledPlugin := &CompiledPlugin{Module: compiledModule}
	if len(bufsectionBytes) > 0 {
		metadata := &wasmpluginv1.ExecConfig{}
		if err := proto.Unmarshal(bufsectionBytes, metadata); err != nil {
			return nil, err
		}
		compiledPlugin.ExecConfig = metadata
	}
	return compiledPlugin, nil
}

// Run executes a plugin. If the plugin exited with non-zero status, this
// returns a *PluginExecutionError.
func (e *WASMPluginExecutor) Run(
	ctx context.Context,
	plugin *CompiledPlugin,
	stdin io.Reader,
	stdout io.Writer,
) (retErr error) {
	name := plugin.Module.Name()
	if name == "" {
		// Some plugins will attempt to read argv[0], but don't have a
		// name in the wasm file. Fallback to this.
		name = "protoc-gen-wasm"
	}

	stderr := bytes.NewBuffer(nil)
	config := wazero.NewModuleConfig().
		WithName(plugin.Module.Name()).
		WithArgs(append([]string{name}, plugin.ExecConfig.GetArgs()...)...).
		WithStdin(stdin).
		WithStdout(stdout).
		WithStderr(stderr)
	if len(plugin.ExecConfig.GetFiles()) > 0 {
		mapFS := make(fstest.MapFS, len(plugin.ExecConfig.GetFiles()))
		for _, file := range plugin.ExecConfig.Files {
			mapFS[strings.TrimPrefix(file.Path, "/")] = &fstest.MapFile{
				Data: file.Contents,
			}
		}
		config = config.WithFS(mapFS)
	}

	runtime := wazero.NewRuntimeWithConfig(ctx, e.runtimeConfig)
	defer func() {
		retErr = multierr.Append(retErr, runtime.Close(ctx))
	}()

	var err error
	switch plugin.ABI() {
	case wasmpluginv1.WasmABI_WASM_ABI_GOJS:
		hostModuleBuilder := runtime.NewHostModuleBuilder("go")
		gojs.NewFunctionExporter().ExportFunctions(hostModuleBuilder)
		var module api.Module
		module, err = hostModuleBuilder.Instantiate(ctx)
		if err != nil {
			return fmt.Errorf("error instantiating gojs: %w", err)
		}
		defer func() {
			retErr = multierr.Append(retErr, module.Close(ctx))
		}()
		err = gojs.Run(ctx, runtime, plugin.Module, gojs.NewConfig(config))
	case wasmpluginv1.WasmABI_WASM_ABI_WASI_SNAPSHOT_PREVIEW1:
		var closer api.Closer
		closer, err = wasi_snapshot_preview1.NewBuilder(runtime).Instantiate(ctx)
		if err != nil {
			return fmt.Errorf("error instantiating wasi: %w", err)
		}
		defer func() {
			retErr = multierr.Append(retErr, closer.Close(ctx))
		}()
		var module api.Module
		module, err = runtime.InstantiateModule(ctx, plugin.Module, config)
		if err != nil {
			if exitErr := new(sys.ExitError); errors.As(err, &exitErr) {
				if exitErr.ExitCode() == 0 {
					return nil
				}
				return NewPluginExecutionError(
					fmt.Sprintf("error instantiating wasi module: %s", err.Error()), exitErr.ExitCode())
			}
			return err
		}
		defer func() {
			retErr = multierr.Append(retErr, module.Close(ctx))
		}()
	default:
		err = errors.New("unable to detect wasm abi")
	}
	if err != nil {
		if exitErr := new(sys.ExitError); errors.As(err, &exitErr) {
			if exitErr.ExitCode() == 0 {
				return nil
			}
			return NewPluginExecutionError(stderr.String(), exitErr.ExitCode())
		}
		return err
	}
	return nil
}

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
	_ "embed"
	"errors"
	"io"
	"sync"
	"testing"

	wasmpluginv1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/wasmplugin/v1"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
)

// echoWasm is a basic wasm file that echos the first 100 bytes from stdin
// before exitting with exit code 11. It serves as a valid wasm file to executed
// during test and manipulate for section reading.
//
// Regenerate it using "wat2wasm echo.wat -o echo.wasm"
//
// For more complex tests, we need to check in bulkier wasm files and add more
// complex toolchains to build them. Since we don't need to re-test wazero here,
// this should suffice for now.
//
//go:embed testdata/echo.wasm
var echoWasm []byte

func TestSectionEncodeDecode(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	executor, err := NewPluginExecutor(t.TempDir())
	require.NoError(t, err)
	plugin, err := executor.CompilePlugin(ctx, echoWasm)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, plugin.Close())
	}()
	assert.Nil(t, plugin.ExecConfig)

	metadataProto := &wasmpluginv1.ExecConfig{
		WasmAbi: wasmpluginv1.WasmABI_WASM_ABI_WASI_SNAPSHOT_PREVIEW1,
		Args:    []string{"some", "params"},
	}
	bufSectionBytes, err := EncodeBufSection(metadataProto)
	require.NoError(t, err)

	wasmFileWithBufSection := make([]byte, 0, len(echoWasm)+len(bufSectionBytes))
	wasmFileWithBufSection = append(wasmFileWithBufSection, echoWasm...)
	wasmFileWithBufSection = append(wasmFileWithBufSection, bufSectionBytes...)

	plugin, err = executor.CompilePlugin(ctx, wasmFileWithBufSection)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, plugin.Close())
	}()
	assert.Empty(t, cmp.Diff(plugin.ExecConfig, metadataProto, protocmp.Transform()))
}

func TestPluginExecutor(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	executor, err := NewPluginExecutor(t.TempDir())
	require.NoError(t, err)
	plugin, err := executor.CompilePlugin(ctx, echoWasm)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, plugin.Close())
	}()
	assert.Nil(t, plugin.ExecConfig)

	stdin := bytes.NewBufferString("foo")
	stdout := bytes.NewBuffer(nil)
	err = executor.Run(ctx, plugin, stdin, stdout)
	pluginErr := new(PluginExecutionError)
	require.ErrorAs(t, err, &pluginErr)
	assert.Equal(t, uint32(11), pluginErr.Exitcode)
	assert.Equal(t, "foo", stdout.String())
}

func TestParallelPlugins(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	executor, err := NewPluginExecutor(t.TempDir())
	require.NoError(t, err)
	plugin, err := executor.CompilePlugin(ctx, echoWasm)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, plugin.Close())
	}()
	assert.Nil(t, plugin.ExecConfig)

	n := 2
	errors := make([]error, n)
	stdOuts := make([]*bytes.Buffer, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			stdOut := bytes.NewBuffer(nil)
			stdOuts[i] = stdOut
			errors[i] = executor.Run(ctx, plugin, bytes.NewBufferString("foo"), stdOut)
		}()
	}
	wg.Wait()

	for i := 0; i < n; i++ {
		pluginErr := new(PluginExecutionError)
		require.ErrorAs(t, errors[i], &pluginErr)
		assert.Equal(t, uint32(11), pluginErr.Exitcode)
		assert.Equal(t, "foo", stdOuts[i].String())
	}
}

func BenchmarkPluginRun(b *testing.B) {
	ctx := context.Background()
	executor, err := NewPluginExecutor(b.TempDir())
	if err != nil {
		b.Fatal(err)
	}
	plugin, err := executor.CompilePlugin(ctx, echoWasm)
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		require.NoError(b, plugin.Close())
	}()
	if plugin.ExecConfig != nil {
		b.Fatal("expected no plugin.ExecConfig")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := executor.Run(ctx, plugin, bytes.NewBufferString("foo"), io.Discard)
		pluginErr := new(PluginExecutionError)
		if ok := errors.As(err, &pluginErr); !ok || pluginErr.Exitcode != 11 {
			b.Fatal(err)
		}
	}
}

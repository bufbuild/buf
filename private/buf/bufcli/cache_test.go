// Copyright 2020-2026 Buf Technologies, Inc.
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

package bufcli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// emptyWasmModule is the smallest valid Wasm module: the magic number and
// version.
var emptyWasmModule = []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}

func TestWasmRuntimeDoesNotCreateCacheDirUntilCompile(t *testing.T) {
	t.Parallel()
	baseCacheDirPath := t.TempDir()
	wasmRuntime := newTestWasmRuntime(baseCacheDirPath)
	require.NoError(t, wasmRuntime.Close(t.Context()))
	assert.NoDirExists(t, filepath.Join(baseCacheDirPath, normalpath.Unnormalize(v3CacheWasmRuntimeRelDirPath)))
}

func TestWasmRuntimeCompileCreatesCacheDir(t *testing.T) {
	t.Parallel()
	baseCacheDirPath := t.TempDir()
	wasmRuntime := newTestWasmRuntime(baseCacheDirPath)
	compiledModule, err := wasmRuntime.Compile(t.Context(), "test", emptyWasmModule)
	require.NoError(t, err)
	require.NoError(t, compiledModule.Close(t.Context()))
	require.NoError(t, wasmRuntime.Close(t.Context()))
	assert.DirExists(t, filepath.Join(baseCacheDirPath, normalpath.Unnormalize(v3CacheWasmRuntimeRelDirPath)))
}

func TestWasmRuntimeCompileWithReadOnlyCacheDirErrors(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions are not enforced on windows")
	}
	baseCacheDirPath := t.TempDir()
	require.NoError(t, os.Chmod(baseCacheDirPath, 0500))
	t.Cleanup(func() {
		// Allow t.TempDir to remove the directory.
		_ = os.Chmod(baseCacheDirPath, 0700)
	})
	if file, err := os.CreateTemp(baseCacheDirPath, ""); err == nil {
		_ = file.Close()
		t.Skip("directory is writable despite permissions, likely running as root")
	}
	wasmRuntime := newTestWasmRuntime(baseCacheDirPath)
	_, err := wasmRuntime.Compile(t.Context(), "test", emptyWasmModule)
	require.Error(t, err)
	require.NoError(t, wasmRuntime.Close(t.Context()))
}

func newTestWasmRuntime(baseCacheDirPath string) *lazyWasmRuntime {
	return &lazyWasmRuntime{
		baseCacheDirPath: baseCacheDirPath,
	}
}

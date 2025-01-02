// Copyright 2020-2025 Buf Technologies, Inc.
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

package pluginrpcutil

import (
	"fmt"
	"os"

	"github.com/bufbuild/buf/private/pkg/wasm"
	"pluginrpc.com/pluginrpc"
)

// NewLocalRunner returns a new pluginrpc.Runner for the local program.
//
// The programName is the path or name of the program. Any program args are passed to
// the program when it is run. The programArgs may be nil.
func NewLocalRunner(programName string, programArgs ...string) pluginrpc.Runner {
	return newRunner(programName, programArgs...)
}

// NewWasmRunner returns a new pluginrpc.Runner for the wasm.Runtime and Wasm data.
//
// This is used for Wasm plugins. getData returns the Wasm data.
// The program name is the name of the source file. Must be non-empty.
// The program args are the arguments to the program. May be empty.
func NewWasmRunner(delegate wasm.Runtime, getData func() ([]byte, error), programName string, programArgs ...string) pluginrpc.Runner {
	return newWasmRunner(delegate, getData, programName, programArgs...)
}

// NewLocalWasmRunner returns a new pluginrpc.Runner for the local wasm file.
//
// This is used for local Wasm plugins. The program name is the path to the Wasm file.
// The program args are the arguments to the program. May be empty.
// The Wasm file is read from the filesystem.
func NewLocalWasmRunner(delegate wasm.Runtime, programName string, programArgs ...string) pluginrpc.Runner {
	getData := func() ([]byte, error) {
		// Find the plugin path. We use the same logic as exec.LookPath, but we do
		// not require the file to be executable. So check the local directory
		// first before checking the PATH.
		var path string
		if fileInfo, err := os.Stat(programName); err == nil && !fileInfo.IsDir() {
			path = programName
		} else {
			var err error
			path, err = unsafeLookPath(programName)
			if err != nil {
				return nil, fmt.Errorf("could not find file %q in PATH: %v", programName, err)
			}
		}
		return os.ReadFile(path)
	}
	return NewWasmRunner(delegate, getData, programName, programArgs...)
}

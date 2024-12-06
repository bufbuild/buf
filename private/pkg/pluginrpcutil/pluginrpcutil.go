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

// NewWasmRunner returns a new pluginrpc.Runner for the wasm.Runtime.
//
// The getData function should return the Wasm module bytes for the program.
// The program name is the name of the program. Any program args are passed to
// the program when it is run. The programArgs may be nil.
func NewWasmRunner(delegate wasm.Runtime, getData func() ([]byte, error), programName string, programArgs ...string) pluginrpc.Runner {
	return newWasmRunner(delegate, getData, programName, programArgs...)
}

// NewLocalWasmRunner returns a new pluginrpc.Runner for the wasm.Runtime.
//
// The program name is the path to the Wasm file. Any program args are passed to
// the program when it is run. The programArgs may be nil.
func NewLocalWasmRunner(delegate wasm.Runtime, programName string, programArgs ...string) pluginrpc.Runner {
	getData := func() ([]byte, error) {
		// Find the plugin filePath. We use the same logic as exec.LookPath, but we do
		// not require the file to be executable. So check the local directory
		// first before checking the PATH.
		var filePath string
		if fileInfo, err := os.Stat(programName); err == nil && !fileInfo.IsDir() {
			filePath = programName
		} else {
			var err error
			filePath, err = unsafeLookPath(programName)
			if err != nil {
				return nil, fmt.Errorf("could not find plugin %q in PATH: %v", programName, err)
			}
		}
		moduleWasm, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("could not read plugin %q: %v", programName, err)
		}
		return moduleWasm, nil
	}
	return newWasmRunner(delegate, getData, programName, programArgs...)
}

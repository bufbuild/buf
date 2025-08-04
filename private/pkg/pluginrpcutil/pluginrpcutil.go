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
// the program when it is run. The programArgs may be nil. The environment is set to
// os.Environ().
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

// ReadWasmFileFromOS reads the Wasm data from the OS.
//
// This is used for local Wasm plugins. The lookup is similar to exec.LookPath.
// The file must be in the local directory or in the PATH. The file is not
// required to be executable.
func ReadWasmFileFromOS(name string) ([]byte, error) {
	// Find the plugin path. We use the same logic as exec.LookPath, but we do
	// not require the file to be executable. So check the local directory
	// first before checking the PATH.
	var path string
	if fileInfo, err := os.Stat(name); err == nil && !fileInfo.IsDir() {
		path = name
	} else {
		var err error
		path, err = unsafeLookPath(name)
		if err != nil {
			return nil, fmt.Errorf("could not find file %q in PATH: %v", name, err)
		}
	}
	return os.ReadFile(path)
}

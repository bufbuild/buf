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

// NewLocalRunner returns a new pluginrpc.Runner for the program name.
func NewLocalRunner(programName string, programArgs ...string) pluginrpc.Runner {
	return newRunner(programName, programArgs...)
}

// NewWasmRunner returns a new pluginrpc.Runner for the wasm.Runtime and program name.
func NewWasmRunner(delegate wasm.Runtime, getData func() ([]byte, error), programName string, programArgs ...string) pluginrpc.Runner {
	return newWasmRunner(delegate, getData, programName, programArgs...)
}

// NewLocalWasmRunner returns a new pluginrpc.Runner for the wasm.Runtime and program name.
//
// This runner is used for local Wasm plugins. The program name is the path to the Wasm file.
func NewLocalWasmRunner(delegate wasm.Runtime, programName string, programArgs ...string) pluginrpc.Runner {
	getData := func() ([]byte, error) {
		moduleWasm, err := os.ReadFile(programName)
		if err != nil {
			return nil, fmt.Errorf("could not read plugin %q: %v", programName, err)
		}
		return moduleWasm, nil
	}
	return newWasmRunner(delegate, getData, programName, programArgs...)
}

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

package bufcli

import "github.com/bufbuild/buf/private/pkg/app"

const (
	// WASMCompilationCacheDir compiled WASM plugin cache directory
	WASMCompilationCacheDir = "wasmplugin-bin"

	alphaEnableWASMEnvKey = "BUF_ALPHA_ENABLE_WASM"
)

// IsAlphaWASMEnabled returns an BUF_ALPHA_ENABLE_WASM is set to true.
func IsAlphaWASMEnabled(container app.EnvContainer) (bool, error) {
	return app.EnvBool(container, alphaEnableWASMEnvKey, false)
}

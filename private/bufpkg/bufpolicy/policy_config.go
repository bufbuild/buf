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

package bufpolicy

import (
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
)

// PolicyConfig is the configuration for a Policy.
type PolicyConfig interface {
	// LintConfig returns the LintConfig for the File.
	LintConfig() bufconfig.LintConfig
	// BreakingConfig returns the BreakingConfig for the File.
	BreakingConfig() bufconfig.BreakingConfig
	// PluginConfigs returns the PluginConfigs for the File.
	PluginConfigs() []bufconfig.PluginConfig
}

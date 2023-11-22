// GenerateInputConfig is an input configuration.
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

package bufconfig

import "github.com/bufbuild/buf/private/buf/buffetch"

// GenerateInputConfig is an input configuration for code generation.
type GenerateInputConfig interface {
	// Ref returns the input ref.
	Ref() buffetch.Ref
	// Types returns the types to generate. If GenerateConfig.GenerateTypeConfig()
	// returns a non-empty list of types.
	Types() []string
	// ExcludePaths returns paths not to generate for.
	ExcludePaths() []string
	// IncludePaths returns paths to generate for.
	IncludePaths() []string

	isGenerateInputConfig()
}

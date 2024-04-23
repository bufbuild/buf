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

package bufconfig

// GenerateTypeConfig is a type filter configuration.
type GenerateTypeConfig interface {
	// If IncludeTypes returns a non-empty list, it means that only those types are
	// generated. Otherwise all types are generated.
	IncludeTypes() []string

	isGenerateTypeConfig()
}

// NewGenerateTypeConfig returns a new GenerateTypeConfig.
func NewGenerateTypeConfig(includeTypes []string) GenerateTypeConfig {
	return newGenerateTypeConfig(includeTypes)
}

// *** PRIVATE ***

type generateTypeConfig struct {
	includeTypes []string
}

func newGenerateTypeConfig(includeTypes []string) GenerateTypeConfig {
	if len(includeTypes) == 0 {
		return nil
	}
	return &generateTypeConfig{
		includeTypes: includeTypes,
	}
}

func (g *generateTypeConfig) IncludeTypes() []string {
	return g.includeTypes
}

func (g *generateTypeConfig) isGenerateTypeConfig() {}

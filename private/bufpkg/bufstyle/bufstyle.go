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

// Package bufstyle defines lint analyzers that help enforce Buf's Go code standards.
package bufstyle

import "golang.org/x/tools/go/analysis"

// ExternalConfig is an external configuration for bufstyle.
type ExternalConfig struct {
	// Ignore is a map from analyzer name to a list of relative paths to ignore.
	Ignore map[string][]string `json:"ignore,omitempty" yaml:"ignore,omitempty"`
}

// AnalyzerProvider provides analyzers.
type AnalyzerProvider interface {
	Analyzers() []*analysis.Analyzer
}

// NewAnalyzerProvider returns a new AnalyzerProvider.
func NewAnalyzerProvider(rootDirPath string, options ...AnalyzerProviderOption) (AnalyzerProvider, error) {
	return newAnalyzerProvider(rootDirPath, options...)
}

// AnalyzerProviderOption is an option for a new AnalyzerProvider.
type AnalyzerProviderOption func(*analyzerProvider)

// WithIgnore will ignore diagnostics for the given file path and analyzer name.
//
// relFilePath should be relative to rootDirPath.
func WithIgnore(analyzerName string, relFilePath string) AnalyzerProviderOption {
	return func(analyzerProvider *analyzerProvider) {
		relFilePaths, ok := analyzerProvider.ignoreAnalyzerNameToRelFilePaths[analyzerName]
		if !ok {
			relFilePaths = make(map[string]struct{})
			analyzerProvider.ignoreAnalyzerNameToRelFilePaths[analyzerName] = relFilePaths
		}
		relFilePaths[relFilePath] = struct{}{}
	}
}

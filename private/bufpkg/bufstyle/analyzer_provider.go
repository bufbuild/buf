// Copyright 2020-2022 Buf Technologies, Inc.
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

package bufstyle

import (
	"golang.org/x/tools/go/analysis"
)

type analyzerProvider struct {
	filePathToIgnoreAnalyzerNames map[string]map[string]struct{}
}

func newAnalyzerProvider(options ...AnalyzerProviderOption) *analyzerProvider {
	analyzerProvider := &analyzerProvider{
		filePathToIgnoreAnalyzerNames: make(map[string]map[string]struct{}),
	}
	for _, option := range options {
		option(analyzerProvider)
	}
	return analyzerProvider
}

func (a *analyzerProvider) Analyzers() []*analysis.Analyzer {
	analyzers := newAnalyzers()
	for _, analyzer := range analyzers {
		a.modifyAnalyzer(analyzer)
		for _, requireAnalyzer := range analyzer.Requires {
			a.modifyAnalyzer(requireAnalyzer)
		}
	}
	return analyzers
}

func (a *analyzerProvider) modifyAnalyzer(analyzer *analysis.Analyzer) {
	if analyzer.Run == nil {
		return
	}
	oldRun := analyzer.Run
	analyzer.Run = func(pass *analysis.Pass) (interface{}, error) {
		oldReport := pass.Report
		pass.Report = func(diagnostic analysis.Diagnostic) {
			if pass.Fset == nil {
				oldReport(diagnostic)
				return
			}
			position := pass.Fset.Position(diagnostic.Pos)
			ignoreAnalyzerNames, ok := a.filePathToIgnoreAnalyzerNames[position.Filename]
			if !ok {
				oldReport(diagnostic)
				return
			}
			if _, ok := ignoreAnalyzerNames[analyzer.Name]; !ok {
				oldReport(diagnostic)
			}
		}
		return oldRun(pass)
	}
}

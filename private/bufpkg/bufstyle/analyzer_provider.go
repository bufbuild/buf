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

package bufstyle

import (
	"path/filepath"

	"go.uber.org/multierr"
	"golang.org/x/tools/go/analysis"
)

type analyzerProvider struct {
	absRootDirPath                   string
	ignoreAnalyzerNameToRelFilePaths map[string]map[string]struct{}
}

func newAnalyzerProvider(rootDirPath string, options ...AnalyzerProviderOption) (*analyzerProvider, error) {
	if rootDirPath == "" {
		rootDirPath = "."
	}
	absRootDirPath, err := filepath.Abs(rootDirPath)
	if err != nil {
		return nil, err
	}
	analyzerProvider := &analyzerProvider{
		absRootDirPath:                   absRootDirPath,
		ignoreAnalyzerNameToRelFilePaths: make(map[string]map[string]struct{}),
	}
	for _, option := range options {
		option(analyzerProvider)
	}
	return analyzerProvider, nil
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
		var reportErr error
		pass.Report = func(diagnostic analysis.Diagnostic) {
			if pass.Fset == nil {
				oldReport(diagnostic)
				return
			}
			relFilePaths, ok := a.ignoreAnalyzerNameToRelFilePaths[analyzer.Name]
			if !ok {
				oldReport(diagnostic)
				return
			}
			position := pass.Fset.Position(diagnostic.Pos)
			filePath := position.Filename
			if filePath == "" {
				oldReport(diagnostic)
				return
			}
			absFilePath, err := filepath.Abs(position.Filename)
			if err != nil {
				reportErr = multierr.Append(reportErr, err)
				return
			}
			relFilePath, err := filepath.Rel(a.absRootDirPath, absFilePath)
			if err != nil {
				reportErr = multierr.Append(reportErr, err)
				return
			}
			if _, ok := relFilePaths[relFilePath]; !ok {
				oldReport(diagnostic)
			}
		}
		if reportErr != nil {
			return nil, reportErr
		}
		return oldRun(pass)
	}
}

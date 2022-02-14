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

// Package bufstyle defines a golangi-lint plugin that enforces
// Buf's Go code standards.
package bufstyle

import (
	"github.com/bufbuild/buf/private/bufstyle/packagefilename"
	"golang.org/x/tools/go/analysis"
)

// AnalyzerPlugin implements the go/analysis API for Buf's Go code
// standards.
type AnalyzerPlugin struct{}

// GetAnalyzers returns all of the analysis checks to enforce
// Buf's code style.
func (*AnalyzerPlugin) GetAnalyzers() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		packagefilename.NewAnalyzer(),
	}
}

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

// Package bufstyle defines lint analyzers that help enforce Buf's Go code standards.
package bufstyle

import (
	"go/token"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var (
	// Analyzers are all the analyzers implemented to help enforce Buf's style guide.
	Analyzers = []*analysis.Analyzer{
		packagefilenameAnalyzer,
	}

	packagefilenameAnalyzer = &analysis.Analyzer{
		Name: "packagefilename",
		Doc:  "Verifies that every package has a file with the same name as the package.",
		Run:  packagefilenameRun,
	}
)

// packagefilenameRun is run once per package.
func packagefilenameRun(pass *analysis.Pass) (interface{}, error) {
	packageName := pass.Pkg.Name()
	if strings.HasSuffix(packageName, "_test") {
		// Ignore test packages.
		return nil, nil
	}
	var found bool
	pass.Fset.Iterate(
		func(file *token.File) bool {
			filename := filepath.Base(file.Name())
			if strings.TrimSuffix(filename, ".go") == packageName {
				found = true
				return false
			}
			return true
		},
	)
	if !found {
		var pos token.Pos
		// The package should be guaranteed to have at least one
		// file with a package declaration, so we report the failure there.
		// We do a check here to make sure we don't panic.
		if len(pass.Files) > 0 {
			pos = pass.Files[0].Package
		}
		pass.Reportf(pos, "Package %q does not have a %s.go", packageName, packageName)
	}
	return nil, nil
}

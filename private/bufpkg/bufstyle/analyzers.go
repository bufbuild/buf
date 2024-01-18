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

// Package bufstyle defines lint analyzers that help enforce Buf's Go code standards.
package bufstyle

import (
	"go/token"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// we don't store this as a global because we modify these in the analyzerProvider.
func newAnalyzers() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		{
			Name: "PACKAGE_FILENAME",
			Doc:  "Verifies that every package has a file with the same name as the package.",
			Run: func(pass *analysis.Pass) (interface{}, error) {
				if len(pass.Files) == 0 {
					// Nothing to do. We can't report the error anywhere because
					// this package doesn't have any files.
					return nil, nil
				}
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
					// The package is guaranteed to have at least one
					// file with a package declaration, so we report the failure there.
					// We checked that len(pass.Files) > 0 above.
					pass.Reportf(pass.Files[0].Package, "Package %q does not have a %s.go", packageName, packageName)
				}
				return nil, nil
			},
		},
		{
			Name: "NO_SYNC_POOL",
			Doc:  "Verifies that sync.Pool is not used.",
			Run: func(pass *analysis.Pass) (interface{}, error) {
				if typesInfo := pass.TypesInfo; typesInfo != nil {
					for expr, typeAndValue := range pass.TypesInfo.Types {
						if t := typeAndValue.Type; t != nil {
							if t.String() == "sync.Pool" {
								pass.Reportf(expr.Pos(), "sync.Pool cannot be used")
							}
						}
					}
				}
				return nil, nil
			},
		},
		{
			Name: "BEHAVIOUR",
			Doc:  "Verifies that the word \"behaviour\" is not used in any comment.",
			Run: func(pass *analysis.Pass) (interface{}, error) {
				for _, file := range pass.Files {
					for _, commentGroup := range file.Comments {
						for _, comment := range commentGroup.List {
							if strings.Contains(strings.ToLower(comment.Text), "behaviour") {
								pass.Reportf(comment.Slash, `It is spelled "behavior" not "behaviour"`)
							}
						}
					}
				}
				return nil, nil
			},
		},
		{
			Name: "FILEPATH_CASING",
			Doc:  "Verifies filePath or FilePath is used, not filepath or Filepath.",
			Run: func(pass *analysis.Pass) (interface{}, error) {
				if typesInfo := pass.TypesInfo; typesInfo != nil {
					for _, object := range pass.TypesInfo.Defs {
						if object != nil {
							if strings.Contains(object.Name(), "Filepath") {
								pass.Reportf(object.Pos(), `Use "FilePath" instead of "Filepath" in name %q`, object.Name())
							}
							if strings.Contains(object.Name(), "filepath") {
								pass.Reportf(object.Pos(), `Use "filePath" instead of "filepath" in name %q`, object.Name())
							}
						}
					}
				}
				return nil, nil
			},
		},
		//{
		//Name: "FILENAME_CASING",
		//Doc:  "Verifies fileName or FileName is used, not filename or Filename.",
		//Run: func(pass *analysis.Pass) (interface{}, error) {
		//if typesInfo := pass.TypesInfo; typesInfo != nil {
		//for _, object := range pass.TypesInfo.Defs {
		//if object != nil {
		//if strings.Contains(object.Name(), "Filename") {
		//pass.Reportf(object.Pos(), `Use "FileName" instead of "Filename" in name %q`, object.Name())
		//}
		//if strings.Contains(object.Name(), "filename") {
		//pass.Reportf(object.Pos(), `Use "fileName" instead of "filename" in name %q`, object.Name())
		//}
		//}
		//}
		//}
		//return nil, nil
		//},
		//},
		//{
		//Name: "DIRPATH_CASING",
		//Doc:  "Verifies dirPath or DirPath is used, not dirpath or Dirpath.",
		//Run: func(pass *analysis.Pass) (interface{}, error) {
		//if typesInfo := pass.TypesInfo; typesInfo != nil {
		//for _, object := range pass.TypesInfo.Defs {
		//if object != nil {
		//if strings.Contains(object.Name(), "Dirpath") {
		//pass.Reportf(object.Pos(), `Use "DirPath" instead of "Dirpath" in name %q`, object.Name())
		//}
		//if strings.Contains(object.Name(), "dirpath") {
		//pass.Reportf(object.Pos(), `Use "dirPath" instead of "dirpath" in name %q`, object.Name())
		//}
		//}
		//}
		//}
		//return nil, nil
		//},
		//},
	}
}

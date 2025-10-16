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

package casing

import (
	"go/types"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufstyle/internal/util"
	"golang.org/x/tools/go/analysis"
)

var badToGood = map[string]string{
	"dirname":  "dirName",
	"Dirname":  "DirName",
	"dirpath":  "dirPath",
	"Dirpath":  "DirPath",
	"filename": "fileName",
	"Filename": "FileName",
	"filepath": "filePath",
	"Filepath": "FilePath",
}

// New returns a new Analyzer.
func New() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "CASING",
		Doc:  "Verifies proper casing for specific words.",
		Run: func(pass *analysis.Pass) (any, error) {
			return nil, util.ForEachObject(
				pass,
				func(object types.Object) error {
					for bad, good := range badToGood {
						if strings.Contains(object.Name(), bad) {
							pass.Reportf(object.Pos(), `Use %q instead of %q`, good, bad)
						}
					}
					return nil
				},
			)
		},
	}
}

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
	"fmt"
	"go/token"
	"go/types"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufstyle/internal/util"
	"golang.org/x/tools/go/analysis"
)

// New returns a new set of Analyzers.
func New() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		newFor("DIRNAME_LOWER", "dirname", "dirName"),
		newFor("DIRNAME_UPPER", "Dirname", "DirName"),
		newFor("DIRPATH_LOWER", "dirpath", "dirPath"),
		newFor("DIRPATH_UPPER", "Dirpath", "DirPath"),
		//newFor("FILENAME_LOWER", "filename", "fileName"),
		//newFor("FILENAME_UPPER", "Filename", "FileName"),
		newFor("FILEPATH_LOWER", "filepath", "filePath"),
		newFor("FILEPATH_UPPER", "Filepath", "FilePath"),
	}
}

func newFor(name string, good string, bad string) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: name,
		Doc:  fmt.Sprintf("Verifies that %q is used instead of %q.", good, bad),
		Run: func(pass *analysis.Pass) (any, error) {
			return nil, util.ForEachObject(
				pass,
				func(object types.Object) error {
					check(pass, object.Pos(), object.Name(), good, bad)
					return nil
				},
			)
		},
	}
}

func check(pass *analysis.Pass, pos token.Pos, text string, good string, bad string) {
	if strings.Contains(strings.ToLower(text), bad) {
		pass.Reportf(pos, `Use  %q instead of %q`, good, bad)
	}
}

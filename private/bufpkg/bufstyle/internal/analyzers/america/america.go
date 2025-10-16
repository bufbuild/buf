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

package america

import (
	"errors"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufstyle/internal/util"
	"golang.org/x/tools/go/analysis"
)

var badToGood = map[string]string{
	"behaviour": "behavior",
	"favourite": "favorite",
}

// New returns a new Analyzer.
func New() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "AMERICA",
		Doc:  "Verifies that the UK English is not used in any comment.",
		Run: func(pass *analysis.Pass) (any, error) {
			return nil, errors.Join(
				util.ForEachComment(
					pass,
					func(comment *ast.Comment) error {
						check(pass, comment.Slash, comment.Text)
						return nil
					},
				),
				util.ForEachObject(
					pass,
					func(object types.Object) error {
						check(pass, object.Pos(), object.Name())
						return nil
					},
				),
			)
		},
	}
}

func check(pass *analysis.Pass, pos token.Pos, text string) {
	for bad, good := range badToGood {
		if strings.Contains(strings.ToLower(text), bad) {
			pass.Reportf(pos, `It is spelled %q not %q`, good, bad)
		}
	}
}

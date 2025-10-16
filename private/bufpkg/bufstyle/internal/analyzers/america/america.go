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
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufstyle/internal/util"
	"golang.org/x/tools/go/analysis"
)

// New returns a new set of Analyzers.
func New() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		newFor("BEHAVIOUR", "behavior", "behaviour"),
		newFor("FAVOURITE", "favorite", "favourite"),
	}
}

func newFor(name string, good string, bad string) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: name,
		Doc:  fmt.Sprintf("Verifies that %q is used instead of %q.", good, bad),
		Run: func(pass *analysis.Pass) (any, error) {
			return nil, errors.Join(
				util.ForEachComment(
					pass,
					func(comment *ast.Comment) error {
						check(pass, comment.Slash, comment.Text, good, bad)
						return nil
					},
				),
				util.ForEachObject(
					pass,
					func(object types.Object) error {
						check(pass, object.Pos(), object.Name(), good, bad)
						return nil
					},
				),
			)
		},
	}
}

func check(pass *analysis.Pass, pos token.Pos, text string, good string, bad string) {
	if index := strings.Index(text, bad); index != -1 {
		badPos := token.Pos(int(pos) + index)
		endPos := token.Pos(int(badPos) + len(bad))
		pass.Report(
			analysis.Diagnostic{
				Pos:     badPos,
				End:     endPos,
				Message: fmt.Sprintf(`It is spelled %q not %q`, good, bad),
				SuggestedFixes: []analysis.SuggestedFix{
					{
						Message: fmt.Sprintf("Replace %q with %q", bad, good),
						TextEdits: []analysis.TextEdit{
							{
								Pos:     badPos,
								End:     endPos,
								NewText: []byte(good),
							},
						},
					},
				},
			},
		)
	}
}

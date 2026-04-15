// Copyright 2020-2026 Buf Technologies, Inc.
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

package buflintvalidate

import (
	"testing"

	"github.com/google/cel-go/cel"
	"github.com/stretchr/testify/require"
)

func TestCelIssuesErrors(t *testing.T) {
	t.Parallel()
	celEnv, err := cel.NewEnv()
	require.NoError(t, err)
	t.Run("single_issue", func(t *testing.T) {
		t.Parallel()
		_, issues := celEnv.Compile("1 / 'a'")
		require.Error(t, issues.Err())
		errs := issues.Errors()
		require.Len(t, errs, 1)
		require.Equal(t, "found no matching overload for '_/_' applied to '(int, string)'", errs[0].Message)
	})
	t.Run("multiple_issues", func(t *testing.T) {
		t.Parallel()
		_, issues := celEnv.Compile("(1 / 'a') * (1 - 'a') * (1 * 'a')")
		require.Error(t, issues.Err())
		errs := issues.Errors()
		require.Len(t, errs, 3)
		require.Equal(t, "found no matching overload for '_/_' applied to '(int, string)'", errs[0].Message)
		require.Equal(t, "found no matching overload for '_-_' applied to '(int, string)'", errs[1].Message)
		require.Equal(t, "found no matching overload for '_*_' applied to '(int, string)'", errs[2].Message)
	})
	t.Run("invalid_escape_in_string_literal", func(t *testing.T) {
		t.Parallel()
		// Reproduces the CEL parse failure from using \. (invalid escape) inside a CEL string
		// literal. This arises when a proto field like:
		//   expression: "this.matches('(^|.*/)[^/]+\\.lock($|/.*)')"
		// is used — the proto string \\ becomes a single \ in the CEL expression, and CEL does
		// not recognize \. as a valid escape sequence.
		stringEnv, err := celEnv.Extend(cel.Variable("this", cel.StringType))
		require.NoError(t, err)
		_, issues := stringEnv.Compile("this == '' || !this.matches('(^|.*/)[^/]+\\.lock($|/.*)')")
		require.Error(t, issues.Err())
		errs := issues.Errors()
		require.Len(t, errs, 6)
		// These are the raw messages from cel-go — the "Syntax error: " prefix is stripped
		// by checkCEL before being reported to the user.
		require.Equal(t, "Syntax error: token recognition error at: ''(^|.*/)[^/]+\\.'", errs[0].Message)
		require.Equal(t, "Syntax error: token recognition error at: '$'", errs[1].Message)
		require.Equal(t, "Syntax error: token recognition error at: '|/'", errs[2].Message)
		require.Equal(t, "Syntax error: no viable alternative at input '.*'", errs[3].Message)
		require.Equal(t, "Syntax error: mismatched input ')' expecting {'[', '{', '(', '.', '-', '!', 'true', 'false', 'null', NUM_FLOAT, NUM_INT, NUM_UINT, STRING, BYTES, IDENTIFIER}", errs[4].Message)
		require.Equal(t, "Syntax error: token recognition error at: '')'", errs[5].Message)
	})
}

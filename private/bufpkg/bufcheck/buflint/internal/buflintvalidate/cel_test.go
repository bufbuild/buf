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

package buflintvalidate

import (
	"testing"

	"github.com/google/cel-go/cel"
	"github.com/stretchr/testify/require"
)

func TestCelIssuesErrorTextMatch(t *testing.T) {
	t.Parallel()
	celEnv, err := cel.NewEnv()
	require.NoError(t, err)
	t.Run("parse_single_issue", func(t *testing.T) {
		_, issues := celEnv.Compile("1 / 'a'")
		expectedErrorText := `ERROR: <input>:1:3: found no matching overload for '_/_' applied to '(int, string)'
 | 1 / 'a'
 | ..^`
		require.Equal(t, expectedErrorText, issues.Err().Error())
		expectedParsedTexts := []string{
			`found no matching overload for '_/_' applied to '(int, string)'
 | 1 / 'a'
 | ..^`,
		}
		require.Equal(t, expectedParsedTexts, parseCelIssuesText(issues.Err().Error()))
	})
	t.Run("parse_multiple_issues", func(t *testing.T) {
		_, issues := celEnv.Compile("(1 / 'a') * (1 - 'a') * (1 * 'a')")
		expectedErrorText := `ERROR: <input>:1:4: found no matching overload for '_/_' applied to '(int, string)'
 | (1 / 'a') * (1 - 'a') * (1 * 'a')
 | ...^
ERROR: <input>:1:16: found no matching overload for '_-_' applied to '(int, string)'
 | (1 / 'a') * (1 - 'a') * (1 * 'a')
 | ...............^
ERROR: <input>:1:28: found no matching overload for '_*_' applied to '(int, string)'
 | (1 / 'a') * (1 - 'a') * (1 * 'a')
 | ...........................^`
		require.Equal(t, expectedErrorText, issues.Err().Error())
		expectedParsedTexts := []string{
			`found no matching overload for '_/_' applied to '(int, string)'
 | (1 / 'a') * (1 - 'a') * (1 * 'a')
 | ...^`,
			`found no matching overload for '_-_' applied to '(int, string)'
 | (1 / 'a') * (1 - 'a') * (1 * 'a')
 | ...............^`,
			`found no matching overload for '_*_' applied to '(int, string)'
 | (1 / 'a') * (1 - 'a') * (1 * 'a')
 | ...........................^`,
		}
		require.Equal(t, expectedParsedTexts, parseCelIssuesText(issues.Err().Error()))
	})
}

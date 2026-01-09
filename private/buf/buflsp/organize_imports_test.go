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

package buflsp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufformat"
	"github.com/bufbuild/protocompile/parser"
	"github.com/bufbuild/protocompile/reporter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func TestOrganizeImports(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	tests := []struct {
		testDir         string
		expectNil       bool
		skipFormatCheck bool
	}{
		{
			testDir:   "no_changes",
			expectNil: true,
		},
		{
			testDir: "sort_imports",
		},
		{
			testDir: "remove_unused",
		},
		{
			testDir: "combined",
		},
		{
			testDir: "add_missing",
		},
		{
			testDir: "add_multiple",
		},
		{
			testDir:   "ambiguous",
			expectNil: true,
		},
		{
			testDir: "invalid_import",
		},
		{
			testDir: "no_existing_imports",
		},
		{
			testDir: "comprehensive_types",
		},
		{
			testDir:         "no_format_rest",
			skipFormatCheck: true,
		},
		{
			testDir: "fully_qualified",
		},
		{
			testDir: "public_import",
		},
		{
			testDir: "nested_messages",
		},
		{
			testDir: "extensions",
		},
		{
			testDir: "comments_in_imports",
		},
		{
			testDir: "duplicate_imports",
		},
	}

	for _, tt := range tests {
		t.Run(tt.testDir, func(t *testing.T) {
			t.Parallel()
			beforePath, err := filepath.Abs(filepath.Join("testdata/organize_imports", tt.testDir, "before.proto"))
			require.NoError(t, err)
			afterPath, err := filepath.Abs(filepath.Join("testdata/organize_imports", tt.testDir, "after.proto"))
			require.NoError(t, err)

			clientJSONConn, testURI := setupLSPServer(t, beforePath)

			var codeActions []protocol.CodeAction
			_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentCodeAction, protocol.CodeActionParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: testURI,
				},
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 0, Character: 0},
				},
				Context: protocol.CodeActionContext{
					Only: []protocol.CodeActionKind{protocol.SourceOrganizeImports},
				},
			}, &codeActions)
			require.NoError(t, err)

			if tt.expectNil {
				assert.Empty(t, codeActions)
				return
			}

			require.Len(t, codeActions, 1)
			assert.Equal(t, "Organize Imports", codeActions[0].Title)
			assert.Equal(t, protocol.SourceOrganizeImports, codeActions[0].Kind)
			require.NotNil(t, codeActions[0].Edit)
			require.NotNil(t, codeActions[0].Edit.Changes)

			changes := codeActions[0].Edit.Changes[testURI]
			require.NotEmpty(t, changes, "should return at least one text edit")

			// Apply the edits to get the result
			beforeContent, err := os.ReadFile(beforePath)
			require.NoError(t, err)
			result := applyTextEdits(string(beforeContent), changes)

			expectedContent, err := os.ReadFile(afterPath)
			require.NoError(t, err)
			assert.Equal(t, string(expectedContent), result)

			// Verify that running bufformat on the result doesn't change it
			// (skip this check if the test intentionally has unformatted content)
			if !tt.skipFormatCheck {
				formatted := formatProtoString(t, result, beforePath)
				assert.Equal(t, result, formatted, "bufformat should not change the organized imports result")
			}
		})
	}
}

func formatProtoString(t *testing.T, content string, filename string) string {
	t.Helper()
	// Create a handler that collects errors
	var parseErrors []reporter.ErrorWithPos
	handler := reporter.NewHandler(reporter.NewReporter(
		func(err reporter.ErrorWithPos) error {
			parseErrors = append(parseErrors, err)
			return nil
		},
		func(err reporter.ErrorWithPos) {
			// Ignore warnings
		},
	))

	// Parse the proto content
	parsed, err := parser.Parse(filename, strings.NewReader(content), handler)
	require.NoError(t, err)
	require.Empty(t, parseErrors, "parse errors: %v", parseErrors)

	// Format using bufformat
	var buf strings.Builder
	err = bufformat.FormatFileNode(&buf, parsed)
	require.NoError(t, err)

	return buf.String()
}

func applyTextEdits(text string, edits []protocol.TextEdit) string {
	lines := strings.Split(text, "\n")
	// Apply edits in reverse order to maintain line numbers
	for i := len(edits) - 1; i >= 0; i-- {
		edit := edits[i]
		startLine := int(edit.Range.Start.Line)
		startChar := int(edit.Range.Start.Character)
		endLine := int(edit.Range.End.Line)
		endChar := int(edit.Range.End.Character)

		// Get the lines before the edit
		before := lines[:startLine]

		// Get the start line with text before edit start
		var startLineText string
		if startLine < len(lines) {
			startLineText = lines[startLine][:startChar]
		}

		// Get the end line with text after edit end
		var endLineText string
		if endLine < len(lines) {
			endLineText = lines[endLine][endChar:]
		}

		// Split the new text into lines
		newLines := strings.Split(edit.NewText, "\n")

		// Combine: before lines + (start line prefix + new text + end line suffix) + after lines
		result := make([]string, 0, len(before)+len(newLines)+len(lines)-endLine-1)
		result = append(result, before...)
		if len(newLines) > 0 {
			// First line of new text gets the start line prefix
			result = append(result, startLineText+newLines[0])
			// Middle lines of new text
			if len(newLines) > 1 {
				result = append(result, newLines[1:]...)
			}
			// Last line of new text gets the end line suffix
			result[len(result)-1] += endLineText
		} else {
			result = append(result, startLineText+endLineText)
		}
		// Lines after the edit
		if endLine+1 < len(lines) {
			result = append(result, lines[endLine+1:]...)
		}

		lines = result
	}
	return strings.Join(lines, "\n")
}

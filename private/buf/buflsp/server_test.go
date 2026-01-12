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

package buflsp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func TestComputeTextEdits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		oldText       string
		newText       string
		expectedEdits []protocol.TextEdit
	}{
		{
			name:          "no_changes",
			oldText:       "line1\nline2\nline3",
			newText:       "line1\nline2\nline3",
			expectedEdits: nil,
		},
		{
			name:    "pure_insertion_at_start",
			oldText: "line2\nline3",
			newText: "line1\nline2\nline3",
			expectedEdits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 0, Character: 0}, End: protocol.Position{Line: 0, Character: 0}},
					NewText: "line1\n",
				},
			},
		},
		{
			name:    "pure_insertion_at_end",
			oldText: "line1\nline2",
			newText: "line1\nline2\nline3",
			expectedEdits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 2, Character: 0}, End: protocol.Position{Line: 2, Character: 0}},
					NewText: "line3\n",
				},
			},
		},
		{
			name:    "pure_insertion_in_middle",
			oldText: "line1\nline3",
			newText: "line1\nline2\nline3",
			expectedEdits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 1, Character: 0}, End: protocol.Position{Line: 1, Character: 0}},
					NewText: "line2\n",
				},
			},
		},
		{
			name:    "pure_deletion_at_start",
			oldText: "line1\nline2\nline3",
			newText: "line2\nline3",
			expectedEdits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 0, Character: 0}, End: protocol.Position{Line: 1, Character: 0}},
					NewText: "",
				},
			},
		},
		{
			name:    "pure_deletion_at_end",
			oldText: "line1\nline2\nline3",
			newText: "line1\nline2",
			expectedEdits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 2, Character: 0}, End: protocol.Position{Line: 3, Character: 0}},
					NewText: "",
				},
			},
		},
		{
			name:    "pure_deletion_in_middle",
			oldText: "line1\nline2\nline3",
			newText: "line1\nline3",
			expectedEdits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 1, Character: 0}, End: protocol.Position{Line: 2, Character: 0}},
					NewText: "",
				},
			},
		},
		{
			name:    "single_line_change",
			oldText: "line1\nline2\nline3",
			newText: "line1\nmodified\nline3",
			expectedEdits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 1, Character: 0}, End: protocol.Position{Line: 2, Character: 0}},
					NewText: "modified\n",
				},
			},
		},
		{
			name:    "multiple_line_changes",
			oldText: "line1\nline2\nline3\nline4",
			newText: "line1\nmodified2\nmodified3\nline4",
			expectedEdits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 1, Character: 0}, End: protocol.Position{Line: 3, Character: 0}},
					NewText: "modified2\nmodified3\n",
				},
			},
		},
		{
			name:    "delete_multiple_lines",
			oldText: "line1\nline2\nline3\nline4\nline5",
			newText: "line1\nline5",
			expectedEdits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 1, Character: 0}, End: protocol.Position{Line: 4, Character: 0}},
					NewText: "",
				},
			},
		},
		{
			name:    "insert_multiple_lines",
			oldText: "line1\nline5",
			newText: "line1\nline2\nline3\nline4\nline5",
			expectedEdits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 1, Character: 0}, End: protocol.Position{Line: 1, Character: 0}},
					NewText: "line2\nline3\nline4\n",
				},
			},
		},
		{
			name:    "alternating_changes",
			oldText: "keep1\nchange1\nkeep2\nchange2\nkeep3",
			newText: "keep1\nmodified1\nkeep2\nmodified2\nkeep3",
			expectedEdits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 1, Character: 0}, End: protocol.Position{Line: 2, Character: 0}},
					NewText: "modified1\n",
				},
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 3, Character: 0}, End: protocol.Position{Line: 4, Character: 0}},
					NewText: "modified2\n",
				},
			},
		},
		{
			name:    "empty_to_content",
			oldText: "",
			newText: "line1\nline2\nline3",
			expectedEdits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 0, Character: 0}, End: protocol.Position{Line: 0, Character: 0}},
					NewText: "line1\nline2\nline3\n",
				},
			},
		},
		{
			name:    "content_to_empty",
			oldText: "line1\nline2\nline3",
			newText: "",
			expectedEdits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 0, Character: 0}, End: protocol.Position{Line: 3, Character: 0}},
					NewText: "",
				},
			},
		},
		{
			name:    "replace_entire_file",
			oldText: "old1\nold2\nold3",
			newText: "new1\nnew2\nnew3",
			expectedEdits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 0, Character: 0}, End: protocol.Position{Line: 3, Character: 0}},
					NewText: "new1\nnew2\nnew3\n",
				},
			},
		},
		{
			name:    "mixed_insert_delete_replace",
			oldText: "line1\nline2\nline3\nline4",
			newText: "line1\ninserted\nmodified3\nline4\nline5",
			expectedEdits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 1, Character: 0}, End: protocol.Position{Line: 3, Character: 0}},
					NewText: "inserted\nmodified3\n",
				},
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 4, Character: 0}, End: protocol.Position{Line: 4, Character: 0}},
					NewText: "line5\n",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actualEdits := computeTextEdits(tt.oldText, tt.newText)

			require.Equal(t, len(tt.expectedEdits), len(actualEdits), "number of edits mismatch")

			for i, expected := range tt.expectedEdits {
				actual := actualEdits[i]
				assert.Equal(t, expected.Range, actual.Range, "edit %d: range mismatch", i)
				assert.Equal(t, expected.NewText, actual.NewText, "edit %d: newText mismatch", i)
			}

			// Verify edits actually transform oldText to newText
			result := applyTextEditsForTest(tt.oldText, actualEdits)
			assert.Equal(t, tt.newText, result, "applying edits should produce expected text")
		})
	}
}

func TestComputeTextEditsMinimal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		oldText string
		newText string
	}{
		{
			name:    "small_change_in_large_file",
			oldText: "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10",
			newText: "line1\nline2\nline3\nmodified\nline5\nline6\nline7\nline8\nline9\nline10",
		},
		{
			name: "multiple_small_changes",
			oldText: `syntax = "proto3";

package test;

message User {
  string name = 1;
  int32 age = 2;
  string email = 3;
}`,
			newText: `syntax = "proto3";

package test;

message User {
  string full_name = 1;
  int32 age = 2;
  string email_address = 3;
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actualEdits := computeTextEdits(tt.oldText, tt.newText)

			if tt.oldText == tt.newText {
				assert.Empty(t, actualEdits, "no edits should be generated for identical text")
				return
			}

			// Verify edits don't overlap
			for i := 0; i < len(actualEdits); i++ {
				for j := i + 1; j < len(actualEdits); j++ {
					assert.Falsef(t, rangesOverlap(actualEdits[i].Range, actualEdits[j].Range),
						"edits %d and %d overlap", i, j)
				}
			}

			// Verify total edit size is reasonable (shouldn't be much larger than full file)
			totalEditSize := 0
			for _, edit := range actualEdits {
				totalEditSize += len(edit.NewText)
			}
			assert.LessOrEqualf(t, totalEditSize, len(tt.newText)*2,
				"edits are not minimal: total edit size %d is much larger than full file size %d",
				totalEditSize, len(tt.newText))

			// Verify edits actually transform oldText to newText
			result := applyTextEditsForTest(tt.oldText, actualEdits)
			assert.Equal(t, tt.newText, result, "applying edits should produce expected text")
		})
	}
}

// applyTextEditsForTest is a simplified version for testing computeTextEdits
func applyTextEditsForTest(text string, edits []protocol.TextEdit) string {
	lines := append([]string{}, splitLines(text)...)

	// Apply edits in reverse order to maintain line positions
	for i := len(edits) - 1; i >= 0; i-- {
		edit := edits[i]
		startLine := int(edit.Range.Start.Line)
		endLine := int(edit.Range.End.Line)

		// Handle pure insertion
		if startLine == endLine && edit.Range.Start.Character == 0 && edit.Range.End.Character == 0 {
			insertLines := splitLines(edit.NewText)
			lines = append(lines[:startLine], append(insertLines, lines[startLine:]...)...)
			continue
		}

		// Handle deletion or replacement
		var newLines []string
		if edit.NewText != "" {
			newLines = splitLines(edit.NewText)
		}

		// Replace lines[startLine:endLine] with newLines
		lines = append(lines[:startLine], append(newLines, lines[endLine:]...)...)
	}

	result := ""
	for _, line := range lines {
		result += line
	}
	return result
}

func splitLines(text string) []string {
	if text == "" {
		return []string{}
	}
	var lines []string
	start := 0
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			lines = append(lines, text[start:i+1])
			start = i + 1
		}
	}
	if start < len(text) {
		lines = append(lines, text[start:]+"\n")
	}
	return lines
}

func rangesOverlap(r1, r2 protocol.Range) bool {
	return positionLess(r1.Start, r2.End) && positionLess(r2.Start, r1.End)
}

func positionLess(p1, p2 protocol.Position) bool {
	if p1.Line != p2.Line {
		return p1.Line < p2.Line
	}
	return p1.Character < p2.Character
}

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
	"go.lsp.dev/protocol"
)

func TestApplyTextEdits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		edits    []protocol.TextEdit
		expected string
	}{
		{
			name:     "no_edits",
			input:    "line1\nline2\nline3",
			edits:    []protocol.TextEdit{},
			expected: "line1\nline2\nline3",
		},
		{
			name:  "delete_blank_line",
			input: "line1\n\n\nline4",
			edits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 2, Character: 0}, End: protocol.Position{Line: 3, Character: 0}},
					NewText: "",
				},
			},
			expected: "line1\n\nline4",
		},
		{
			name:  "replace_lines",
			input: "line1\nline2\nline3\nline4",
			edits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 1, Character: 0}, End: protocol.Position{Line: 3, Character: 0}},
					NewText: "newline2\nnewline3\n",
				},
			},
			expected: "line1\nnewline2\nnewline3\nline4",
		},
		{
			name:  "multiple_edits",
			input: "line1\nline2\nline3\nline4\nline5",
			edits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 1, Character: 0}, End: protocol.Position{Line: 2, Character: 0}},
					NewText: "",
				},
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 3, Character: 0}, End: protocol.Position{Line: 4, Character: 0}},
					NewText: "",
				},
			},
			expected: "line1\nline3\nline5",
		},
		{
			name:  "swap_lines_with_delete_and_insert",
			input: "line1\nimport\nline3\npackage\nline5",
			edits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 1, Character: 0}, End: protocol.Position{Line: 3, Character: 0}},
					NewText: "package\n",
				},
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 3, Character: 0}, End: protocol.Position{Line: 4, Character: 0}},
					NewText: "import\n",
				},
			},
			expected: "line1\npackage\nimport\nline5",
		},
		{
			name:  "insert_at_start",
			input: "line2\nline3",
			edits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 0, Character: 0}, End: protocol.Position{Line: 0, Character: 0}},
					NewText: "line1\n",
				},
			},
			expected: "line1\nline2\nline3",
		},
		{
			name:  "insert_at_end",
			input: "line1\nline2",
			edits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 2, Character: 0}, End: protocol.Position{Line: 2, Character: 0}},
					NewText: "line3\n",
				},
			},
			expected: "line1\nline2\nline3\n",
		},
		{
			name:  "mid_line_edit",
			input: "hello world",
			edits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 0, Character: 6}, End: protocol.Position{Line: 0, Character: 11}},
					NewText: "there",
				},
			},
			expected: "hello there",
		},
		{
			name:  "delete_entire_file",
			input: "line1\nline2\nline3",
			edits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 0, Character: 0}, End: protocol.Position{Line: 3, Character: 0}},
					NewText: "",
				},
			},
			expected: "",
		},
		{
			name:  "insert_into_empty_file",
			input: "",
			edits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 0, Character: 0}, End: protocol.Position{Line: 0, Character: 0}},
					NewText: "line1\nline2\n",
				},
			},
			expected: "line1\nline2\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ApplyTextEdits(tt.input, tt.edits)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestApplyTextEditsEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		edits    []protocol.TextEdit
		expected string
	}{
		{
			name:  "empty_to_content",
			input: "",
			edits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 0, Character: 0}, End: protocol.Position{Line: 0, Character: 0}},
					NewText: "new content\n",
				},
			},
			expected: "new content\n",
		},
		{
			name:  "content_to_empty",
			input: "some content\nmore lines\n",
			edits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 0, Character: 0}, End: protocol.Position{Line: 2, Character: 0}},
					NewText: "",
				},
			},
			expected: "",
		},
		{
			name:  "single_character_change",
			input: "x",
			edits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 0, Character: 0}, End: protocol.Position{Line: 0, Character: 1}},
					NewText: "y",
				},
			},
			expected: "y",
		},
		{
			name:  "only_whitespace_changes",
			input: "hello  world",
			edits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 0, Character: 5}, End: protocol.Position{Line: 0, Character: 7}},
					NewText: " ",
				},
			},
			expected: "hello world",
		},
		{
			name:  "unicode_content",
			input: "Hello 世界",
			edits: []protocol.TextEdit{
				{
					// Note: This uses byte offsets, not UTF-16 code units
					// "Hello " = bytes 0-5, "世界" = bytes 6-11 (each Chinese char is 3 bytes in UTF-8)
					Range:   protocol.Range{Start: protocol.Position{Line: 0, Character: 6}, End: protocol.Position{Line: 0, Character: 12}},
					NewText: "World",
				},
			},
			expected: "Hello World",
		},
		{
			name:  "edit_at_file_start",
			input: "line1\nline2",
			edits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 0, Character: 0}, End: protocol.Position{Line: 0, Character: 0}},
					NewText: "prefix ",
				},
			},
			expected: "prefix line1\nline2",
		},
		{
			name:  "edit_at_file_end",
			input: "line1\nline2",
			edits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 1, Character: 5}, End: protocol.Position{Line: 1, Character: 5}},
					NewText: " suffix",
				},
			},
			expected: "line1\nline2 suffix",
		},
		{
			name:  "multiple_edits_same_line",
			input: "the quick brown fox",
			edits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 0, Character: 16}, End: protocol.Position{Line: 0, Character: 19}},
					NewText: "dog",
				},
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 0, Character: 10}, End: protocol.Position{Line: 0, Character: 15}},
					NewText: "red",
				},
			},
			expected: "the quick red dog",
		},
		{
			name:  "newline_only_content",
			input: "\n\n\n",
			edits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 1, Character: 0}, End: protocol.Position{Line: 2, Character: 0}},
					NewText: "",
				},
			},
			expected: "\n\n",
		},
		{
			name:  "preserve_trailing_newline",
			input: "line1\nline2\n",
			edits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 0, Character: 0}, End: protocol.Position{Line: 0, Character: 5}},
					NewText: "modified",
				},
			},
			expected: "modified\nline2\n",
		},
		{
			name:  "remove_trailing_newline",
			input: "line1\nline2\n",
			edits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 2, Character: 0}, End: protocol.Position{Line: 2, Character: 0}},
					NewText: "",
				},
			},
			expected: "line1\nline2\n",
		},
		{
			name:  "insert_multiple_newlines",
			input: "line1\nline2",
			edits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 1, Character: 0}, End: protocol.Position{Line: 1, Character: 0}},
					NewText: "\n\n",
				},
			},
			expected: "line1\n\n\nline2",
		},
		{
			name:  "large_file_small_change",
			input: "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\nline11\nline12\nline13\nline14\nline15",
			edits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 7, Character: 0}, End: protocol.Position{Line: 8, Character: 0}},
					NewText: "modified8\n",
				},
			},
			expected: "line1\nline2\nline3\nline4\nline5\nline6\nline7\nmodified8\nline9\nline10\nline11\nline12\nline13\nline14\nline15",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ApplyTextEdits(tt.input, tt.edits)
			assert.Equal(t, tt.expected, result)
		})
	}
}

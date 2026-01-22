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
	"context"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufformat"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func TestFormatting(t *testing.T) {
	t.Parallel()

	tests := []formatTest{
		{
			name:           "format_unformatted_file",
			protoFile:      "unformatted.proto",
			expectEdits:    true,
			expectNumEdits: 1,
		},
		{
			name:           "format_already_formatted_file",
			protoFile:      "formatted.proto",
			expectEdits:    false,
			expectNumEdits: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runFormatTest(t, tt)
		})
	}
}

func TestRangeFormatting(t *testing.T) {
	t.Parallel()

	tests := []formatTest{
		{
			name:      "format_range_in_unformatted_file",
			protoFile: "unformatted.proto",
			formatRange: &protocol.Range{
				Start: protocol.Position{Line: 5, Character: 0},  // Start of Product message
				End:   protocol.Position{Line: 10, Character: 1}, // End of Product message
			},
			expectEdits:    true,
			expectNumEdits: 1,
		},
		{
			name:      "format_range_in_formatted_file",
			protoFile: "formatted.proto",
			formatRange: &protocol.Range{
				Start: protocol.Position{Line: 3, Character: 0},
				End:   protocol.Position{Line: 7, Character: 1},
			},
			expectEdits:    false,
			expectNumEdits: 0,
		},
		{
			name:      "format_entire_file_via_range",
			protoFile: "unformatted.proto",
			formatRange: &protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 100, Character: 0}, // Beyond end of file
			},
			expectEdits:    true,
			expectNumEdits: 1,
		},
		{
			name:      "format_service_range",
			protoFile: "unformatted.proto",
			formatRange: &protocol.Range{
				Start: protocol.Position{Line: 18, Character: 0}, // Start of ProductService
				End:   protocol.Position{Line: 21, Character: 1}, // End of ProductService
			},
			expectEdits:    true,
			expectNumEdits: 1,
		},
		{
			name:      "format_enum_range",
			protoFile: "unformatted.proto",
			formatRange: &protocol.Range{
				Start: protocol.Position{Line: 12, Character: 0}, // Start of Status enum
				End:   protocol.Position{Line: 16, Character: 1}, // End of Status enum
			},
			expectEdits:    true,
			expectNumEdits: 1,
		},
		{
			name:      "format_multiple_declarations",
			protoFile: "unformatted.proto",
			formatRange: &protocol.Range{
				Start: protocol.Position{Line: 6, Character: 0},  // Start of Product message
				End:   protocol.Position{Line: 16, Character: 1}, // End of Status enum
			},
			expectEdits:    true,
			expectNumEdits: 1,
		},
		{
			name:      "format_single_line_in_message",
			protoFile: "unformatted.proto",
			formatRange: &protocol.Range{
				Start: protocol.Position{Line: 7, Character: 0}, // Just the "string name = 1;" line
				End:   protocol.Position{Line: 7, Character: 20},
			},
			expectEdits:    true,
			expectNumEdits: 1,
		},
		{
			name:      "format_extend_block",
			protoFile: "with_extensions.proto",
			formatRange: &protocol.Range{
				Start: protocol.Position{Line: 10, Character: 0}, // Start of extend block
				End:   protocol.Position{Line: 13, Character: 1}, // End of extend block
			},
			expectEdits:    true,
			expectNumEdits: 1,
		},
		{
			name:      "format_nested_message",
			protoFile: "nested.proto",
			formatRange: &protocol.Range{
				Start: protocol.Position{Line: 7, Character: 0},  // Start of Nested message
				End:   protocol.Position{Line: 10, Character: 3}, // End of Nested message
			},
			expectEdits:    true,
			expectNumEdits: 1,
		},
		{
			name:      "format_nested_enum",
			protoFile: "nested.proto",
			formatRange: &protocol.Range{
				Start: protocol.Position{Line: 14, Character: 0}, // Start of Status enum
				End:   protocol.Position{Line: 17, Character: 3}, // End of Status enum
			},
			expectEdits:    true,
			expectNumEdits: 1,
		},
		{
			name:      "format_invalid_file",
			protoFile: "invalid.proto",
			formatRange: &protocol.Range{
				Start: protocol.Position{Line: 4, Character: 0}, // Start of Broken message
				End:   protocol.Position{Line: 8, Character: 1}, // End of Broken message
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runFormatTest(t, tt)
		})
	}
}

// formatTest describes a formatting test case, supporting both full-file and range formatting.
type formatTest struct {
	name           string
	protoFile      string
	formatRange    *protocol.Range // nil for full file formatting
	expectEdits    bool
	expectNumEdits int
	expectError    bool
}

// runFormatTest executes a formatting test, supporting both full-file and range formatting.
func runFormatTest(t *testing.T, tt formatTest) {
	t.Helper()
	ctx := t.Context()

	testDir := "testdata/format"
	if tt.expectError {
		testDir = "testdata/format_invalid"
	}

	testProtoPath, err := filepath.Abs(filepath.Join(testDir, tt.protoFile))
	require.NoError(t, err)
	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)

	var textEdits []protocol.TextEdit
	var formatErr error

	if tt.formatRange != nil {
		// Range formatting
		_, formatErr = clientJSONConn.Call(ctx, protocol.MethodTextDocumentRangeFormatting,
			protocol.DocumentRangeFormattingParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: testURI},
				Range:        *tt.formatRange,
			}, &textEdits)
	} else {
		// Full file formatting
		_, formatErr = clientJSONConn.Call(ctx, protocol.MethodTextDocumentFormatting,
			protocol.DocumentFormattingParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: testURI},
			}, &textEdits)
	}

	if tt.expectError {
		require.Error(t, formatErr)
		return
	}

	require.NoError(t, formatErr)
	assert.Len(t, textEdits, tt.expectNumEdits)

	if tt.expectEdits {
		validateFormattedOutput(t, ctx, testProtoPath, textEdits, tt.formatRange)
	}
}

// validateFormattedOutput verifies that the formatted output matches expectations.
func validateFormattedOutput(t *testing.T, ctx context.Context, testProtoPath string,
	textEdits []protocol.TextEdit, formatRange *protocol.Range) {
	t.Helper()
	require.NotEmpty(t, textEdits)

	expectedFormatted := getExpectedFormattedContent(t, ctx, testProtoPath)
	edit := textEdits[0]

	if formatRange == nil {
		// Full file formatting - should match exactly and start at line 0
		assert.Equal(t, expectedFormatted, edit.NewText)
		assert.Equal(t, uint32(0), edit.Range.Start.Line)
		assert.Equal(t, uint32(0), edit.Range.Start.Character)
	} else {
		// Range formatting - should be a proper subset
		assert.NotEmpty(t, edit.NewText)
		assert.Contains(t, expectedFormatted, edit.NewText,
			"Range formatted text should match the corresponding section in the full formatted file")

		// For non-full-file ranges, verify we're not formatting the whole file
		if formatRange.Start.Line > 0 || formatRange.End.Line < 100 {
			assert.NotEqual(t, uint32(0), edit.Range.Start.Line,
				"Range formatting should not start at line 0 for non-full-file ranges")
			assert.Less(t, len(edit.NewText), len(expectedFormatted),
				"Range formatting should return less text than full file formatting")
		}

		// Additional validation: check that formatted text doesn't have trailing garbage
		if len(edit.NewText) > 0 {
			lastChar := edit.NewText[len(edit.NewText)-1]
			// For messages, enums, services, and extends, the last character should be '}'
			if lastChar == '}' && len(edit.NewText) > 1 {
				// Check if there's a duplicate closing brace
				assert.NotEqual(t, byte('}'), edit.NewText[len(edit.NewText)-2],
					"Found duplicate closing brace at end of formatted text: %q",
					edit.NewText[len(edit.NewText)-10:])
			}
		}
	}
}

func getExpectedFormattedContent(t *testing.T, ctx context.Context, protoPath string) string {
	t.Helper()
	dir := filepath.Dir(protoPath)
	bucket, err := storageos.NewProvider().NewReadWriteBucket(dir)
	require.NoError(t, err)
	formattedBucket, err := bufformat.FormatBucket(ctx, bucket)
	require.NoError(t, err)
	formatted, err := storage.ReadPath(ctx, formattedBucket, filepath.Base(protoPath))
	require.NoError(t, err)
	return string(formatted)
}

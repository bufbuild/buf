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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufformat"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

// Test file list for various test scenarios
var (
	allFormatTestFiles = []string{
		"unformatted.proto",
		"proto2_unformatted.proto",
		"with_comments.proto",
		"with_options.proto",
		"with_imports.proto",
		"nested_and_oneofs.proto",
		"maps_and_reserved.proto",
		"whitespace_edge_cases.proto",
	}
)

func TestFormatting(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	tests := []struct {
		name          string
		protoFile     string
		expectedEdits []protocol.TextEdit
		checkEdits    bool // if false, only verify formatted output, not specific edits
	}{
		{
			name:       "format_unformatted_file",
			protoFile:  "unformatted.proto",
			checkEdits: true,
			expectedEdits: []protocol.TextEdit{
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 4, Character: 0}, End: protocol.Position{Line: 5, Character: 0}},
					NewText: "",
				},
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 6, Character: 0}, End: protocol.Position{Line: 8, Character: 0}},
					NewText: "  string name = 1;\n  uint32 price = 2;\n",
				},
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 13, Character: 0}, End: protocol.Position{Line: 15, Character: 0}},
					NewText: "  STATUS_ACTIVE = 1;\n  STATUS_INACTIVE = 2;\n",
				},
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 18, Character: 0}, End: protocol.Position{Line: 20, Character: 0}},
					NewText: "  rpc GetProduct(GetProductRequest) returns (GetProductResponse);\n  rpc ListProducts(ListProductsRequest) returns (ListProductsResponse);\n",
				},
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 23, Character: 0}, End: protocol.Position{Line: 24, Character: 0}},
					NewText: "  string id = 1;\n",
				},
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 26, Character: 0}, End: protocol.Position{Line: 27, Character: 0}},
					NewText: "message GetProductResponse {\n",
				},
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 31, Character: 0}, End: protocol.Position{Line: 32, Character: 0}},
					NewText: "  uint32 page_size = 1;\n",
				},
				{
					Range:   protocol.Range{Start: protocol.Position{Line: 36, Character: 0}, End: protocol.Position{Line: 38, Character: 0}},
					NewText: "  repeated Product products = 1;\n  string next_page_token = 2;\n",
				},
			},
		},
		{
			name:          "format_already_formatted_file",
			protoFile:     "formatted.proto",
			checkEdits:    true,
			expectedEdits: nil,
		},
		{
			name:       "format_proto2_file",
			protoFile:  "proto2_unformatted.proto",
			checkEdits: false,
		},
		{
			name:       "format_file_with_comments",
			protoFile:  "with_comments.proto",
			checkEdits: false,
		},
		{
			name:       "format_file_with_options",
			protoFile:  "with_options.proto",
			checkEdits: false,
		},
		{
			name:       "format_file_with_imports",
			protoFile:  "with_imports.proto",
			checkEdits: false,
		},
		{
			name:       "format_nested_messages_and_oneofs",
			protoFile:  "nested_and_oneofs.proto",
			checkEdits: false,
		},
		{
			name:       "format_maps_and_reserved",
			protoFile:  "maps_and_reserved.proto",
			checkEdits: false,
		},
		{
			name:          "format_empty_file",
			protoFile:     "empty.proto",
			checkEdits:    true,
			expectedEdits: nil,
		},
		{
			name:       "format_whitespace_edge_cases",
			protoFile:  "whitespace_edge_cases.proto",
			checkEdits: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			testProtoPath := mustAbsPath(t, "testdata/format", tt.protoFile)
			textEdits := callFormatting(t, ctx, testProtoPath)

			if tt.checkEdits {
				assert.Equal(t, tt.expectedEdits, textEdits)
			}

			originalContent := mustReadFile(t, testProtoPath)
			result := applyTextEdits(originalContent, textEdits)
			expectedFormatted := getExpectedFormattedContent(t, ctx, testProtoPath)

			assert.Equal(t, expectedFormatted, result)
		})
	}
}

func TestFormattingIdempotency(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	for _, protoFile := range allFormatTestFiles {
		protoFile := protoFile
		t.Run(protoFile, func(t *testing.T) {
			t.Parallel()

			testProtoPath := mustAbsPath(t, "testdata/format", protoFile)

			// First format: get the formatted content
			firstEdits := callFormatting(t, ctx, testProtoPath)
			originalContent := mustReadFile(t, testProtoPath)
			formattedContent := applyTextEdits(originalContent, firstEdits)

			// Write formatted content to a temp directory with necessary dependencies
			tmpDir := setupTempFormatDir(t, testProtoPath, protoFile, formattedContent)

			// Second format: should produce no edits (idempotency check)
			tmpFile := filepath.Join(tmpDir, protoFile)
			secondEdits := callFormatting(t, ctx, tmpFile)

			assert.Empty(t, secondEdits, "formatting an already-formatted file should produce no edits")
		})
	}
}

func TestFormattingMinimalEdits(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	testFiles := []string{
		"unformatted.proto",
		"with_comments.proto",
		"with_options.proto",
		"whitespace_edge_cases.proto",
	}

	for _, protoFile := range testFiles {
		protoFile := protoFile
		t.Run(protoFile, func(t *testing.T) {
			t.Parallel()

			testProtoPath := mustAbsPath(t, "testdata/format", protoFile)
			textEdits := callFormatting(t, ctx, testProtoPath)
			originalText := mustReadFile(t, testProtoPath)
			expectedFormatted := getExpectedFormattedContent(t, ctx, testProtoPath)

			assertEditsAreMinimal(t, originalText, expectedFormatted, textEdits)
		})
	}
}

// assertEditsAreMinimal verifies that the provided edits are minimal.
func assertEditsAreMinimal(t *testing.T, original, expected string, edits []protocol.TextEdit) {
	t.Helper()

	if original == expected {
		assert.Empty(t, edits, "no edits should be generated for identical text")
		return
	}

	// Verify edits don't overlap
	assertEditsDoNotOverlap(t, edits)

	// Verify total edit size is reasonable
	totalEditSize := 0
	for _, edit := range edits {
		totalEditSize += len(edit.NewText)
	}

	fullReplaceSize := len(expected)
	assert.LessOrEqualf(t, totalEditSize, fullReplaceSize*2,
		"edits are not minimal: total edit size %d is much larger than full file size %d",
		totalEditSize, fullReplaceSize)

	// Verify edits actually transform original to expected
	result := applyTextEdits(original, edits)
	assert.Equal(t, expected, result, "applying edits should produce expected formatted text")

	// Verify no edit is entirely unchanged
	assertNoRedundantEdits(t, original, edits)
}

// assertEditsDoNotOverlap verifies that no two edits in the list overlap.
func assertEditsDoNotOverlap(t *testing.T, edits []protocol.TextEdit) {
	t.Helper()

	for i := 0; i < len(edits); i++ {
		for j := i + 1; j < len(edits); j++ {
			assert.Falsef(t, rangesOverlap(edits[i].Range, edits[j].Range),
				"edits %d and %d overlap: %s and %s",
				i, j,
				formatRange(edits[i].Range),
				formatRange(edits[j].Range))
		}
	}
}

// assertNoRedundantEdits verifies that no edit replaces text with identical text.
func assertNoRedundantEdits(t *testing.T, original string, edits []protocol.TextEdit) {
	t.Helper()

	originalLines := strings.Split(original, "\n")
	for i, edit := range edits {
		startLine := int(edit.Range.Start.Line)
		endLine := int(edit.Range.End.Line)

		if startLine >= len(originalLines) || endLine >= len(originalLines) {
			continue
		}

		// Extract the text being replaced
		var replacedLines []string
		for line := startLine; line <= endLine; line++ {
			replacedLines = append(replacedLines, originalLines[line])
		}
		replacedText := strings.Join(replacedLines, "\n")

		assert.NotEqualf(t, edit.NewText, replacedText,
			"edit %d replaces text with identical text (not minimal): range %s",
			i, formatRange(edit.Range))
	}
}

func applyTextEdits(text string, edits []protocol.TextEdit) string {
	lines := strings.Split(text, "\n")
	// Apply edits in reverse order to maintain line/character positions
	for i := len(edits) - 1; i >= 0; i-- {
		edit := edits[i]
		startLine := int(edit.Range.Start.Line)
		startChar := int(edit.Range.Start.Character)
		endLine := int(edit.Range.End.Line)
		endChar := int(edit.Range.End.Character)

		if startLine < 0 || startLine >= len(lines) {
			continue
		}
		if endLine < 0 || endLine > len(lines) {
			endLine = len(lines)
		}

		// LSP TextEdit semantics: Range.End is exclusive at character level
		// Get the prefix (part of start line before the edit)
		prefix := ""
		if startLine < len(lines) && startChar <= len(lines[startLine]) {
			prefix = lines[startLine][:startChar]
		}

		// Get the suffix (part of end line after the edit)
		// If endChar is 0, we're at the start of endLine, so no suffix from that line
		suffix := ""
		if endChar > 0 && endLine < len(lines) {
			if endChar <= len(lines[endLine]) {
				suffix = lines[endLine][endChar:]
			}
		} else if startLine == endLine && endChar == 0 {
			// Pure insertion at the beginning of a line - need the whole line as suffix
			if endLine < len(lines) {
				suffix = lines[endLine]
			}
		}

		// Determine which lines to delete
		// For a pure insertion (start == end), the line gets incorporated into replacement via suffix
		// So we need to delete it from the original lines array
		// Otherwise, we delete from startLine to endLine (inclusive or exclusive depending on endChar)
		deleteEndLine := endLine
		if startLine == endLine && startChar == endChar {
			// Pure insertion - the line is preserved in suffix, so delete it from lines array
			deleteEndLine = startLine
		} else if endChar == 0 && endLine > startLine {
			// Range ends at the beginning of endLine, so don't include endLine in deletion
			deleteEndLine = endLine - 1
		}

		// Split the new text into lines
		newText := edit.NewText
		var newLines []string
		endsWithNewline := false
		if newText != "" {
			newLines = strings.Split(newText, "\n")
			// If newText ends with \n, split gives us a trailing empty string.
			// This indicates that subsequent content should be on a new line.
			endsWithNewline = strings.HasSuffix(newText, "\n")
			if endsWithNewline && len(newLines) > 0 && newLines[len(newLines)-1] == "" {
				newLines = newLines[:len(newLines)-1]
			}
		}

		// Build the replacement
		var replacement []string
		if len(newLines) == 0 {
			// No new content
			combined := prefix + suffix
			// Only create a line if there's actually content or we're doing a mid-line edit
			if combined != "" || (startChar > 0 || endChar > 0) {
				replacement = []string{combined}
			} else {
				// Deleting full lines with no replacement
				replacement = []string{}
			}
		} else {
			// Add prefix to first line
			replacement = make([]string, len(newLines))
			copy(replacement, newLines)
			replacement[0] = prefix + replacement[0]
			// Add suffix: if newText ended with \n, suffix goes on a new line
			// Otherwise, append it to the last line
			if suffix != "" {
				if endsWithNewline {
					replacement = append(replacement, suffix)
				} else {
					replacement[len(replacement)-1] = replacement[len(replacement)-1] + suffix
				}
			} else if startLine == endLine && startChar == endChar && endsWithNewline {
				// Pure insertion at start of line where newText ends with newline
				// Even if the original line is blank (suffix == ""), we should preserve it
				replacement = append(replacement, "")
			}
		}

		// Replace lines[startLine:deleteEndLine+1] with replacement
		lines = append(lines[:startLine], append(replacement, lines[deleteEndLine+1:]...)...)
	}
	return strings.Join(lines, "\n")
}

func TestApplyTextEdits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		edits    []protocol.TextEdit
		expected string
	}{
		{
			name:  "delete blank line",
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
			name:  "replace lines",
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
			name:  "multiple edits",
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
			name:  "swap lines with delete and insert",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyTextEdits(tt.input, tt.edits)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestApplyTextEditsDebug(t *testing.T) {
	// This test helps debug the actual edits being generated
	// t.Skip("Debug test - enable when needed")

	ctx := t.Context()
	testProtoPath := mustAbsPath(t, "testdata/format", "whitespace_edge_cases.proto")

	edits := callFormatting(t, ctx, testProtoPath)
	originalContent := mustReadFile(t, testProtoPath)
	result := applyTextEdits(originalContent, edits)
	expected := getExpectedFormattedContent(t, ctx, testProtoPath)

	assert.Equal(t, expected, result, "Result doesn't match expected")
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

// Helper functions

// mustAbsPath returns the absolute path by joining the given path elements.
// It fails the test if the path cannot be resolved.
func mustAbsPath(t *testing.T, elem ...string) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join(elem...))
	require.NoError(t, err)
	return path
}

// mustReadFile reads the file at the given path and returns its content as a string.
// It fails the test if the file cannot be read.
func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(content)
}

// callFormatting calls the LSP formatting method on the given file and returns the edits.
// It fails the test if formatting fails.
func callFormatting(t *testing.T, ctx context.Context, filePath string) []protocol.TextEdit {
	t.Helper()
	clientJSONConn, testURI := setupLSPServer(t, filePath)
	var textEdits []protocol.TextEdit
	_, err := clientJSONConn.Call(ctx, protocol.MethodTextDocumentFormatting, protocol.DocumentFormattingParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: testURI,
		},
	}, &textEdits)
	require.NoError(t, err)
	return textEdits
}

// setupTempFormatDir creates a temporary directory with the formatted file and necessary
// dependencies (buf.yaml and imported files). Returns the temp directory path.
func setupTempFormatDir(t *testing.T, originalPath, protoFile, formattedContent string) string {
	t.Helper()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, protoFile)
	require.NoError(t, os.WriteFile(tmpFile, []byte(formattedContent), 0o644))

	// Copy buf.yaml
	bufYamlSrc := filepath.Join(filepath.Dir(originalPath), "buf.yaml")
	bufYamlDst := filepath.Join(tmpDir, "buf.yaml")
	bufYamlContent := mustReadFile(t, bufYamlSrc)
	require.NoError(t, os.WriteFile(bufYamlDst, []byte(bufYamlContent), 0o644))

	// Copy other.proto if needed for imports test
	if protoFile == "with_imports.proto" {
		otherSrc := filepath.Join(filepath.Dir(originalPath), "other.proto")
		otherDst := filepath.Join(tmpDir, "other.proto")
		otherContent := mustReadFile(t, otherSrc)
		require.NoError(t, os.WriteFile(otherDst, []byte(otherContent), 0o644))
	}

	return tmpDir
}

// rangesOverlap returns true if two LSP ranges overlap.
// Two ranges overlap if one starts before the other ends.
func rangesOverlap(r1, r2 protocol.Range) bool {
	// Compare positions: r1.Start < r2.End && r2.Start < r1.End
	return positionLess(r1.Start, r2.End) && positionLess(r2.Start, r1.End)
}

// positionLess returns true if p1 is before p2 in the document.
func positionLess(p1, p2 protocol.Position) bool {
	if p1.Line != p2.Line {
		return p1.Line < p2.Line
	}
	return p1.Character < p2.Character
}

// formatRange returns a human-readable string representation of an LSP range.
func formatRange(r protocol.Range) string {
	return fmt.Sprintf("[%d:%d-%d:%d]",
		r.Start.Line, r.Start.Character,
		r.End.Line, r.End.Character)
}

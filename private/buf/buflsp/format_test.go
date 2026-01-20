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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/buf/buflsp"
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
			result := buflsp.ApplyTextEdits(originalContent, textEdits)
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
			formattedContent := buflsp.ApplyTextEdits(originalContent, firstEdits)

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
	result := buflsp.ApplyTextEdits(original, edits)
	assert.Equal(t, expected, result, "applying edits should produce expected formatted text")

	// Verify no edit is entirely unchanged
	assertNoRedundantEdits(t, original, edits)
}

// assertEditsDoNotOverlap verifies that no two edits in the list overlap.
func assertEditsDoNotOverlap(t *testing.T, edits []protocol.TextEdit) {
	t.Helper()

	for i := 0; i < len(edits); i++ {
		for j := i + 1; j < len(edits); j++ {
			assert.Falsef(t, buflsp.RangesOverlap(edits[i].Range, edits[j].Range),
				"edits %d and %d overlap: %s and %s",
				i, j,
				buflsp.FormatRange(edits[i].Range),
				buflsp.FormatRange(edits[j].Range))
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
			i, buflsp.FormatRange(edit.Range))
	}
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
			result := buflsp.ApplyTextEdits(tt.input, tt.edits)
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
	result := buflsp.ApplyTextEdits(originalContent, edits)
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


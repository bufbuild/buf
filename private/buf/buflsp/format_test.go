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

	"github.com/bufbuild/buf/private/buf/bufformat"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func TestFormatting(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	tests := []struct {
		name          string
		protoFile     string
		expectedEdits []protocol.TextEdit
	}{
		{
			name:      "format_unformatted_file",
			protoFile: "unformatted.proto",
			expectedEdits: []protocol.TextEdit{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 4, Character: 0},
						End:   protocol.Position{Line: 5, Character: 0},
					},
					NewText: "",
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 6, Character: 0},
						End:   protocol.Position{Line: 8, Character: 0},
					},
					NewText: "  string name = 1;\n  uint32 price = 2;\n",
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 13, Character: 0},
						End:   protocol.Position{Line: 15, Character: 0},
					},
					NewText: "  STATUS_ACTIVE = 1;\n  STATUS_INACTIVE = 2;\n",
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 18, Character: 0},
						End:   protocol.Position{Line: 20, Character: 0},
					},
					NewText: "  rpc GetProduct(GetProductRequest) returns (GetProductResponse);\n  rpc ListProducts(ListProductsRequest) returns (ListProductsResponse);\n",
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 23, Character: 0},
						End:   protocol.Position{Line: 24, Character: 0},
					},
					NewText: "  string id = 1;\n",
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 26, Character: 0},
						End:   protocol.Position{Line: 27, Character: 0},
					},
					NewText: "message GetProductResponse {\n",
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 31, Character: 0},
						End:   protocol.Position{Line: 32, Character: 0},
					},
					NewText: "  uint32 page_size = 1;\n",
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 36, Character: 0},
						End:   protocol.Position{Line: 38, Character: 0},
					},
					NewText: "  repeated Product products = 1;\n  string next_page_token = 2;\n",
				},
			},
		},
		{
			name:          "format_already_formatted_file",
			protoFile:     "formatted.proto",
			expectedEdits: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testProtoPath, err := filepath.Abs(filepath.Join("testdata/format", tt.protoFile))
			require.NoError(t, err)
			clientJSONConn, testURI := setupLSPServer(t, testProtoPath)
			var textEdits []protocol.TextEdit
			_, formatErr := clientJSONConn.Call(ctx, protocol.MethodTextDocumentFormatting, protocol.DocumentFormattingParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: testURI,
				},
			}, &textEdits)
			require.NoError(t, formatErr)
			assert.Equal(t, tt.expectedEdits, textEdits)
			originalContent, err := os.ReadFile(testProtoPath)
			require.NoError(t, err)
			result := applyTextEdits(string(originalContent), textEdits)
			expectedFormatted := getExpectedFormattedContent(t, ctx, testProtoPath)
			assert.Equal(t, expectedFormatted, result)
		})
	}
}

func applyTextEdits(text string, edits []protocol.TextEdit) string {
	lines := strings.Split(text, "\n")
	for i := len(edits) - 1; i >= 0; i-- {
		edit := edits[i]
		startLine := int(edit.Range.Start.Line)
		endLine := int(edit.Range.End.Line)
		var newLines []string
		if edit.NewText != "" {
			newLines = strings.Split(edit.NewText, "\n")
			if len(newLines) > 0 && newLines[len(newLines)-1] == "" {
				newLines = newLines[:len(newLines)-1]
			}
		}
		lines = append(lines[:startLine], append(newLines, lines[endLine:]...)...)
	}
	return strings.Join(lines, "\n")
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

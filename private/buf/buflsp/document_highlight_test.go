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
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestDocumentHighlight(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	testProtoPath, err := filepath.Abs("testdata/document_highlight/highlight.proto")
	require.NoError(t, err)

	typesProtoPath, err := filepath.Abs("testdata/document_highlight/types.proto")
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)
	typesURI := uri.New(typesProtoPath)

	type highlightLocation struct {
		line           uint32
		startCharacter uint32
		endCharacter   uint32
		kind           protocol.DocumentHighlightKind
	}
	tests := []struct {
		name               string
		targetURI          protocol.URI
		line               uint32
		character          uint32
		expectedHighlights []highlightLocation
	}{
		{
			name:      "highlight_product_message",
			targetURI: testURI,
			line:      7,
			character: 8,
			expectedHighlights: []highlightLocation{
				{line: 7, startCharacter: 8, endCharacter: 15, kind: protocol.DocumentHighlightKindText},   // message Product (definition)
				{line: 11, startCharacter: 11, endCharacter: 18, kind: protocol.DocumentHighlightKindText}, // repeated Product related
				{line: 16, startCharacter: 11, endCharacter: 18, kind: protocol.DocumentHighlightKindText}, // repeated Product products in Catalog
				{line: 30, startCharacter: 2, endCharacter: 9, kind: protocol.DocumentHighlightKindText},   // Product product in GetProductResponse
				{line: 38, startCharacter: 11, endCharacter: 18, kind: protocol.DocumentHighlightKindText}, // repeated Product products in ListProductsResponse
			},
		},
		{
			name:      "highlight_from_reference",
			targetURI: testURI,
			line:      11,
			character: 11,
			expectedHighlights: []highlightLocation{
				{line: 7, startCharacter: 8, endCharacter: 15, kind: protocol.DocumentHighlightKindText},   // message Product (definition)
				{line: 11, startCharacter: 11, endCharacter: 18, kind: protocol.DocumentHighlightKindText}, // repeated Product related
				{line: 16, startCharacter: 11, endCharacter: 18, kind: protocol.DocumentHighlightKindText}, // repeated Product products in Catalog
				{line: 30, startCharacter: 2, endCharacter: 9, kind: protocol.DocumentHighlightKindText},   // Product product in GetProductResponse
				{line: 38, startCharacter: 11, endCharacter: 18, kind: protocol.DocumentHighlightKindText}, // repeated Product products in ListProductsResponse
			},
		},
		{
			name:      "highlight_catalog_message",
			targetURI: testURI,
			line:      14,
			character: 8,
			expectedHighlights: []highlightLocation{
				{line: 14, startCharacter: 8, endCharacter: 15, kind: protocol.DocumentHighlightKindText}, // message Catalog (definition)
			},
		},
		{
			name:      "highlight_category_enum_from_types",
			targetURI: typesURI,
			line:      4,
			character: 5,
			expectedHighlights: []highlightLocation{
				{line: 4, startCharacter: 5, endCharacter: 13, kind: protocol.DocumentHighlightKindText}, // enum Category (definition)
			},
		},
		{
			name:               "no_highlight_on_enum_value",
			targetURI:          typesURI,
			line:               6,
			character:          2,
			expectedHighlights: nil, // Enum values should not be highlighted
		},
		{
			name:               "no_highlight_on_service",
			targetURI:          testURI,
			line:               20,
			character:          8,
			expectedHighlights: nil, // Services should not be highlighted
		},
		{
			name:               "no_highlight_on_rpc_method",
			targetURI:          testURI,
			line:               21,
			character:          6,
			expectedHighlights: nil, // RPC methods should not be highlighted
		},
		{
			name:      "highlight_request_message",
			targetURI: testURI,
			line:      25,
			character: 8,
			expectedHighlights: []highlightLocation{
				{line: 25, startCharacter: 8, endCharacter: 25, kind: protocol.DocumentHighlightKindText},  // message GetProductRequest (definition)
				{line: 21, startCharacter: 17, endCharacter: 34, kind: protocol.DocumentHighlightKindText}, // GetProductRequest in rpc
			},
		},
		{
			name:               "no_highlight_on_field_name",
			targetURI:          testURI,
			line:               8,
			character:          9,
			expectedHighlights: nil, // Field names should not be highlighted
		},
		{
			name:      "highlight_nested_message",
			targetURI: testURI,
			line:      42,
			character: 10,
			expectedHighlights: []highlightLocation{
				{line: 42, startCharacter: 10, endCharacter: 14, kind: protocol.DocumentHighlightKindText}, // message Item (definition)
				{line: 46, startCharacter: 11, endCharacter: 15, kind: protocol.DocumentHighlightKindText}, // repeated Item items (reference)
			},
		},
		{
			name:      "highlight_nested_enum",
			targetURI: testURI,
			line:      47,
			character: 7,
			expectedHighlights: []highlightLocation{
				{line: 47, startCharacter: 7, endCharacter: 13, kind: protocol.DocumentHighlightKindText}, // enum Status (definition)
				{line: 52, startCharacter: 2, endCharacter: 8, kind: protocol.DocumentHighlightKindText},  // Status status (reference)
			},
		},
		{
			name:               "no_highlight_on_nested_enum_value",
			targetURI:          testURI,
			line:               49,
			character:          4,
			expectedHighlights: nil, // Enum values should not be highlighted
		},
		{
			name:               "no_highlight_on_oneof_field",
			targetURI:          testURI,
			line:               54,
			character:          11,
			expectedHighlights: nil, // Oneof field names should not be highlighted
		},
		{
			name:               "no_highlight_on_keyword",
			targetURI:          testURI,
			line:               7,
			character:          0,
			expectedHighlights: nil,
		},
		{
			name:               "no_highlight_on_primitive_type",
			targetURI:          testURI,
			line:               8,
			character:          2,
			expectedHighlights: nil,
		},
		{
			name:               "no_highlight_on_field_number",
			targetURI:          testURI,
			line:               8,
			character:          14,
			expectedHighlights: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var highlights []protocol.DocumentHighlight
			_, highlightErr := clientJSONConn.Call(ctx, protocol.MethodTextDocumentDocumentHighlight, protocol.DocumentHighlightParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{
						URI: tt.targetURI,
					},
					Position: protocol.Position{
						Line:      tt.line,
						Character: tt.character,
					},
				},
			}, &highlights)
			require.NoError(t, highlightErr)

			if tt.expectedHighlights == nil {
				require.Nil(t, highlights)
			} else {
				require.Len(t, highlights, len(tt.expectedHighlights))

				for _, expectedHighlight := range tt.expectedHighlights {
					idx := slices.IndexFunc(highlights, func(h protocol.DocumentHighlight) bool {
						return h.Range.Start.Line == expectedHighlight.line &&
							h.Range.Start.Character == expectedHighlight.startCharacter &&
							h.Range.End.Character == expectedHighlight.endCharacter &&
							h.Kind == expectedHighlight.kind
					})
					assert.NotEqual(t, -1, idx, "expected highlight at line %d:%d-%d with kind %v not found", expectedHighlight.line, expectedHighlight.startCharacter, expectedHighlight.endCharacter, expectedHighlight.kind)
				}
			}
		})
	}
}

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

func TestReferences(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	testProtoPath, err := filepath.Abs("testdata/references/references.proto")
	require.NoError(t, err)

	typesProtoPath, err := filepath.Abs("testdata/references/types.proto")
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)
	typesURI := uri.New(typesProtoPath)

	type refLocation struct {
		uri  protocol.URI
		line uint32
	}
	tests := []struct {
		name               string
		targetURI          protocol.URI
		line               uint32
		character          uint32
		includeDeclaration bool
		expectedReferences []refLocation
	}{
		{
			name:               "references_to_item_message",
			targetURI:          testURI,
			line:               6,
			character:          8,
			includeDeclaration: true,
			expectedReferences: []refLocation{
				{uri: testURI, line: 6},  // message Item
				{uri: testURI, line: 10}, // repeated Item related
				{uri: testURI, line: 15}, // repeated Item items in Container
				{uri: testURI, line: 17}, // map<string, Item> items_by_id in Container
				{uri: testURI, line: 30}, // Item item in GetItemResponse
				{uri: testURI, line: 38}, // repeated Item items in ListItemsResponse
			},
		},
		{
			name:               "references_to_item_message_no_declaration",
			targetURI:          testURI,
			line:               6,
			character:          8,
			includeDeclaration: false,
			expectedReferences: []refLocation{
				{uri: testURI, line: 10}, // repeated Item related
				{uri: testURI, line: 15}, // repeated Item items in Container
				{uri: testURI, line: 17}, // map<string, Item> items_by_id in Container
				{uri: testURI, line: 30}, // Item item in GetItemResponse
				{uri: testURI, line: 38}, // repeated Item items in ListItemsResponse
			},
		},
		{
			name:               "references_to_color_enum_imported",
			targetURI:          typesURI,
			line:               4,
			character:          5,
			includeDeclaration: true,
			expectedReferences: []refLocation{
				{uri: typesURI, line: 4}, // enum Color
				{uri: testURI, line: 8},  // Color color in Item
				{uri: testURI, line: 16}, // Color default_color in Container
			},
		},
		{
			name:               "references_to_container_message",
			targetURI:          testURI,
			line:               13,
			character:          8,
			includeDeclaration: true,
			expectedReferences: []refLocation{
				{uri: testURI, line: 13}, // message Container
			},
		},
		{
			name:               "references_to_label_imported_type",
			targetURI:          typesURI,
			line:               10,
			character:          8,
			includeDeclaration: true,
			expectedReferences: []refLocation{
				{uri: typesURI, line: 10}, // message Label
				{uri: testURI, line: 9},   // Label label in Item
			},
		},
		{
			name:               "references_to_get_item_request",
			targetURI:          testURI,
			line:               25,
			character:          8,
			includeDeclaration: true,
			expectedReferences: []refLocation{
				{uri: testURI, line: 25}, // message GetItemRequest
				{uri: testURI, line: 21}, // rpc GetItem(GetItemRequest)
			},
		},
		{
			name:               "references_to_service",
			targetURI:          testURI,
			line:               20,
			character:          8,
			includeDeclaration: true,
			expectedReferences: []refLocation{
				{uri: testURI, line: 20}, // service ItemService
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var locations []protocol.Location
			_, refErr := clientJSONConn.Call(ctx, protocol.MethodTextDocumentReferences, protocol.ReferenceParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{
						URI: tt.targetURI,
					},
					Position: protocol.Position{
						Line:      tt.line,
						Character: tt.character,
					},
				},
				Context: protocol.ReferenceContext{
					IncludeDeclaration: tt.includeDeclaration,
				},
			}, &locations)
			require.NoError(t, refErr)

			require.Len(t, locations, len(tt.expectedReferences))

			for _, expectedRef := range tt.expectedReferences {
				idx := slices.IndexFunc(locations, func(loc protocol.Location) bool {
					return loc.URI == expectedRef.uri && loc.Range.Start.Line == expectedRef.line
				})
				assert.NotEqual(t, -1, idx, "expected reference at %s:%d not found", expectedRef.uri, expectedRef.line)
			}
		})
	}
}

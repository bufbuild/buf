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

func TestRename(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	testProtoPath, err := filepath.Abs("testdata/rename/rename.proto")
	require.NoError(t, err)
	typesProtoPath, err := filepath.Abs("testdata/rename/types.proto")
	require.NoError(t, err)
	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)
	typesURI := uri.New(typesProtoPath)

	type editLocation struct {
		uri            protocol.URI
		line           uint32
		startCharacter uint32
		endCharacter   uint32
	}
	tests := []struct {
		name          string
		targetURI     protocol.URI
		line          uint32
		character     uint32
		newName       string
		expectError   bool
		expectedEdits []editLocation
	}{
		{
			name:      "rename_product_message",
			targetURI: testURI,
			line:      6,
			character: 8,
			newName:   "Item",
			expectedEdits: []editLocation{
				{uri: testURI, line: 6, startCharacter: 8, endCharacter: 15},
				{uri: testURI, line: 10, startCharacter: 11, endCharacter: 18},
				{uri: testURI, line: 15, startCharacter: 11, endCharacter: 18},
				{uri: testURI, line: 29, startCharacter: 2, endCharacter: 9},
				{uri: testURI, line: 37, startCharacter: 11, endCharacter: 18},
			},
		},
		{
			name:      "rename_category_enum_imported",
			targetURI: typesURI,
			line:      4,
			character: 5,
			newName:   "Type",
			expectedEdits: []editLocation{
				{uri: typesURI, line: 4, startCharacter: 5, endCharacter: 13},
				{uri: testURI, line: 9, startCharacter: 2, endCharacter: 10},
				{uri: testURI, line: 16, startCharacter: 2, endCharacter: 10},
			},
		},
		{
			name:      "rename_category_electronics_enum_value",
			targetURI: typesURI,
			line:      6,
			character: 2,
			newName:   "TYPE_GADGETS",
			expectedEdits: []editLocation{
				{uri: typesURI, line: 6, startCharacter: 2, endCharacter: 22},
			},
		},
		{
			name:      "rename_name_field",
			targetURI: testURI,
			line:      8,
			character: 9,
			newName:   "title",
			expectedEdits: []editLocation{
				{uri: testURI, line: 8, startCharacter: 9, endCharacter: 13},
			},
		},
		{
			name:      "rename_service",
			targetURI: testURI,
			line:      19,
			character: 8,
			newName:   "ItemService",
			expectedEdits: []editLocation{
				{uri: testURI, line: 19, startCharacter: 8, endCharacter: 22},
			},
		},
		{
			name:      "rename_rpc_method",
			targetURI: testURI,
			line:      20,
			character: 6,
			newName:   "FetchProduct",
			expectedEdits: []editLocation{
				{uri: testURI, line: 20, startCharacter: 6, endCharacter: 16},
			},
		},
		{
			name:      "rename_request_message",
			targetURI: testURI,
			line:      24,
			character: 8,
			newName:   "FetchProductRequest",
			expectedEdits: []editLocation{
				{uri: testURI, line: 24, startCharacter: 8, endCharacter: 25},
				{uri: testURI, line: 20, startCharacter: 17, endCharacter: 34},
			},
		},
		{
			name:      "rename_metadata_imported_message",
			targetURI: typesURI,
			line:      10,
			character: 8,
			newName:   "Tag",
			expectedEdits: []editLocation{
				{uri: typesURI, line: 10, startCharacter: 8, endCharacter: 16},
			},
		},
		{
			name:      "rename_nested_message",
			targetURI: testURI,
			line:      40,
			character: 10,
			newName:   "OrderItem",
			expectedEdits: []editLocation{
				{uri: testURI, line: 40, startCharacter: 10, endCharacter: 14},
				{uri: testURI, line: 44, startCharacter: 11, endCharacter: 15},
			},
		},
		{
			name:      "rename_nested_enum",
			targetURI: testURI,
			line:      45,
			character: 7,
			newName:   "State",
			expectedEdits: []editLocation{
				{uri: testURI, line: 45, startCharacter: 7, endCharacter: 13},
				{uri: testURI, line: 50, startCharacter: 2, endCharacter: 8},
			},
		},
		{
			name:      "rename_nested_enum_value",
			targetURI: testURI,
			line:      47,
			character: 4,
			newName:   "STATE_PENDING",
			expectedEdits: []editLocation{
				{uri: testURI, line: 47, startCharacter: 4, endCharacter: 18},
			},
		},
		{
			name:      "rename_oneof_field",
			targetURI: testURI,
			line:      52,
			character: 11,
			newName:   "card_number",
			expectedEdits: []editLocation{
				{uri: testURI, line: 52, startCharacter: 11, endCharacter: 22},
			},
		},
		{
			name:      "rename_deeply_nested_message",
			targetURI: testURI,
			line:      57,
			character: 10,
			newName:   "InnerData",
			expectedEdits: []editLocation{
				{uri: testURI, line: 57, startCharacter: 10, endCharacter: 15},
				{uri: testURI, line: 61, startCharacter: 4, endCharacter: 9},
			},
		},
		{
			name:      "rename_nested_message_used_in_sibling",
			targetURI: testURI,
			line:      60,
			character: 10,
			newName:   "Wrapper",
			expectedEdits: []editLocation{
				{uri: testURI, line: 60, startCharacter: 10, endCharacter: 19},
				{uri: testURI, line: 63, startCharacter: 2, endCharacter: 11},
			},
		},
		{
			name:        "rename_to_existing_message",
			targetURI:   testURI,
			line:        6,
			character:   8,
			newName:     "Catalog",
			expectError: true,
		},
		{
			name:        "rename_to_existing_service",
			targetURI:   testURI,
			line:        13,
			character:   10,
			newName:     "ProductService",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var workspaceEdit protocol.WorkspaceEdit
			_, renameErr := clientJSONConn.Call(ctx, protocol.MethodTextDocumentRename, protocol.RenameParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{
						URI: tt.targetURI,
					},
					Position: protocol.Position{
						Line:      tt.line,
						Character: tt.character,
					},
				},
				NewName: tt.newName,
			}, &workspaceEdit)
			if tt.expectError {
				require.Error(t, renameErr)
			} else {
				require.NoError(t, renameErr)
				require.NotNil(t, workspaceEdit.Changes)
				var allEdits []editLocation
				for uri, edits := range workspaceEdit.Changes {
					for _, edit := range edits {
						allEdits = append(allEdits, editLocation{
							uri:            uri,
							line:           edit.Range.Start.Line,
							startCharacter: edit.Range.Start.Character,
							endCharacter:   edit.Range.End.Character,
						})
					}
				}
				require.Len(t, allEdits, len(tt.expectedEdits))
				for _, expectedEdit := range tt.expectedEdits {
					idx := slices.IndexFunc(allEdits, func(e editLocation) bool {
						return e.uri == expectedEdit.uri && e.line == expectedEdit.line && e.startCharacter == expectedEdit.startCharacter
					})
					require.NotEqual(t, -1, idx, "expected edit at %s:%d:%d not found", expectedEdit.uri, expectedEdit.line, expectedEdit.startCharacter)
					assert.Equal(t, expectedEdit.startCharacter, allEdits[idx].startCharacter)
					assert.Equal(t, expectedEdit.endCharacter, allEdits[idx].endCharacter)
				}
			}
		})
	}
}

func TestPrepareRename(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	testProtoPath, err := filepath.Abs("testdata/rename/rename.proto")
	require.NoError(t, err)
	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)

	tests := []struct {
		name          string
		line          uint32
		character     uint32
		expectRange   bool
		expectedStart protocol.Position
		expectedEnd   protocol.Position
	}{
		{
			name:        "prepare_rename_product_message",
			line:        6,
			character:   8,
			expectRange: true,
			expectedStart: protocol.Position{
				Line:      6,
				Character: 8,
			},
			expectedEnd: protocol.Position{
				Line:      6,
				Character: 15,
			},
		},
		{
			name:        "prepare_rename_field",
			line:        8,
			character:   9,
			expectRange: true,
			expectedStart: protocol.Position{
				Line:      8,
				Character: 9,
			},
			expectedEnd: protocol.Position{
				Line:      8,
				Character: 13,
			},
		},
		{
			name:        "prepare_rename_on_keyword",
			line:        6,
			character:   0,
			expectRange: false,
		},
		{
			name:        "prepare_rename_on_primitive_type",
			line:        7,
			character:   2,
			expectRange: false,
		},
		{
			name:        "prepare_rename_on_field_number",
			line:        7,
			character:   14,
			expectRange: false,
		},
		{
			name:        "prepare_rename_on_whitespace",
			line:        7,
			character:   0,
			expectRange: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var rnge *protocol.Range
			_, prepareErr := clientJSONConn.Call(ctx, protocol.MethodTextDocumentPrepareRename, protocol.PrepareRenameParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{
						URI: testURI,
					},
					Position: protocol.Position{
						Line:      tt.line,
						Character: tt.character,
					},
				},
			}, &rnge)
			require.NoError(t, prepareErr)
			if tt.expectRange {
				require.NotNil(t, rnge)
				assert.Equal(t, tt.expectedStart, rnge.Start)
				assert.Equal(t, tt.expectedEnd, rnge.End)
			} else {
				require.Nil(t, rnge)
			}
		})
	}
}

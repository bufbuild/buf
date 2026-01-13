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
	extensionsProtoPath, err := filepath.Abs("testdata/rename/extensions.proto")
	require.NoError(t, err)
	subpkgOptionsProtoPath, err := filepath.Abs("testdata/rename/subpkg/options.proto")
	require.NoError(t, err)
	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)
	typesURI := uri.New(typesProtoPath)
	extensionsURI := uri.New(extensionsProtoPath)
	subpkgOptionsURI := uri.New(subpkgOptionsProtoPath)

	type editLocation struct {
		uri            protocol.URI
		line           uint32
		startCharacter uint32
		endCharacter   uint32
		newText        string // Optional: if set, verify the NewText matches
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
			line:      8,
			character: 8,
			newName:   "Item",
			expectedEdits: []editLocation{
				{uri: testURI, line: 8, startCharacter: 8, endCharacter: 15},
				{uri: testURI, line: 12, startCharacter: 11, endCharacter: 18},
				{uri: testURI, line: 17, startCharacter: 11, endCharacter: 18},
				{uri: testURI, line: 31, startCharacter: 2, endCharacter: 9},
				{uri: testURI, line: 39, startCharacter: 11, endCharacter: 18},
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
				{uri: testURI, line: 11, startCharacter: 2, endCharacter: 10},
				{uri: testURI, line: 18, startCharacter: 2, endCharacter: 10},
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
			line:      10,
			character: 9,
			newName:   "title",
			expectedEdits: []editLocation{
				{uri: testURI, line: 10, startCharacter: 9, endCharacter: 13},
			},
		},
		{
			name:      "rename_service",
			targetURI: testURI,
			line:      21,
			character: 8,
			newName:   "ItemService",
			expectedEdits: []editLocation{
				{uri: testURI, line: 21, startCharacter: 8, endCharacter: 22},
			},
		},
		{
			name:      "rename_rpc_method",
			targetURI: testURI,
			line:      22,
			character: 6,
			newName:   "FetchProduct",
			expectedEdits: []editLocation{
				{uri: testURI, line: 22, startCharacter: 6, endCharacter: 16},
			},
		},
		{
			name:      "rename_request_message",
			targetURI: testURI,
			line:      26,
			character: 8,
			newName:   "FetchProductRequest",
			expectedEdits: []editLocation{
				{uri: testURI, line: 26, startCharacter: 8, endCharacter: 25},
				{uri: testURI, line: 22, startCharacter: 17, endCharacter: 34},
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
			line:      42,
			character: 10,
			newName:   "OrderItem",
			expectedEdits: []editLocation{
				{uri: testURI, line: 42, startCharacter: 10, endCharacter: 14},
				{uri: testURI, line: 46, startCharacter: 11, endCharacter: 15},
			},
		},
		{
			name:      "rename_nested_enum",
			targetURI: testURI,
			line:      47,
			character: 7,
			newName:   "State",
			expectedEdits: []editLocation{
				{uri: testURI, line: 47, startCharacter: 7, endCharacter: 13},
				{uri: testURI, line: 52, startCharacter: 2, endCharacter: 8},
			},
		},
		{
			name:      "rename_nested_enum_value",
			targetURI: testURI,
			line:      49,
			character: 4,
			newName:   "STATE_PENDING",
			expectedEdits: []editLocation{
				{uri: testURI, line: 49, startCharacter: 4, endCharacter: 18},
			},
		},
		{
			name:      "rename_oneof_field",
			targetURI: testURI,
			line:      54,
			character: 11,
			newName:   "card_number",
			expectedEdits: []editLocation{
				{uri: testURI, line: 54, startCharacter: 11, endCharacter: 22},
			},
		},
		{
			name:      "rename_deeply_nested_message",
			targetURI: testURI,
			line:      59,
			character: 10,
			newName:   "InnerData",
			expectedEdits: []editLocation{
				{uri: testURI, line: 59, startCharacter: 10, endCharacter: 15},
				{uri: testURI, line: 63, startCharacter: 4, endCharacter: 9},
			},
		},
		{
			name:      "rename_nested_message_used_in_sibling",
			targetURI: testURI,
			line:      62,
			character: 10,
			newName:   "Wrapper",
			expectedEdits: []editLocation{
				{uri: testURI, line: 62, startCharacter: 10, endCharacter: 19},
				{uri: testURI, line: 65, startCharacter: 2, endCharacter: 11},
			},
		},
		{
			name:        "rename_to_existing_message",
			targetURI:   testURI,
			line:        8,
			character:   8,
			newName:     "Catalog",
			expectError: true,
		},
		{
			name:        "rename_to_existing_service",
			targetURI:   testURI,
			line:        15,
			character:   10,
			newName:     "ProductService",
			expectError: true,
		},
		{
			name:      "rename_custom_extension",
			targetURI: extensionsURI,
			line:      7,
			character: 10,
			newName:   "validated",
			expectedEdits: []editLocation{
				{uri: extensionsURI, line: 7, startCharacter: 10, endCharacter: 17, newText: "validated"},
				{uri: testURI, line: 69, startCharacter: 9, endCharacter: 18, newText: "(rename.v1.validated)"},
				{uri: testURI, line: 74, startCharacter: 9, endCharacter: 18, newText: "(rename.v1.validated)"},
				{uri: testURI, line: 75, startCharacter: 9, endCharacter: 18, newText: "(rename.v1.validated)"},
				{uri: testURI, line: 75, startCharacter: 9, endCharacter: 18, newText: "(rename.v1.validated)"}, // appears twice on this line: (testing).test and (testing).nested.value
			},
		},
		{
			name:      "rename_extension_field",
			targetURI: extensionsURI,
			line:      11,
			character: 7,
			newName:   "blah",
			expectedEdits: []editLocation{
				{uri: extensionsURI, line: 11, startCharacter: 7, endCharacter: 11},
				{uri: testURI, line: 69, startCharacter: 19, endCharacter: 23},
				{uri: testURI, line: 74, startCharacter: 19, endCharacter: 23},
			},
		},
		{
			name:      "rename_nested_extension_field",
			targetURI: extensionsURI,
			line:      15,
			character: 9,
			newName:   "val",
			expectedEdits: []editLocation{
				{uri: extensionsURI, line: 15, startCharacter: 9, endCharacter: 14},
				{uri: testURI, line: 75, startCharacter: 26, endCharacter: 31},
			},
		},
		{
			name:      "rename_subpackage_extension",
			targetURI: subpkgOptionsURI,
			line:      7,
			character: 10,
			newName:   "validated",
			expectedEdits: []editLocation{
				{uri: subpkgOptionsURI, line: 7, startCharacter: 10, endCharacter: 17, newText: "validated"},
				{uri: testURI, line: 80, startCharacter: 9, endCharacter: 25, newText: "(rename.v1.subpkg.validated)"},
			},
		},
		{
			name:      "rename_subpackage_extension_field",
			targetURI: subpkgOptionsURI,
			line:      11,
			character: 7,
			newName:   "blah",
			expectedEdits: []editLocation{
				{uri: subpkgOptionsURI, line: 11, startCharacter: 7, endCharacter: 11},
				{uri: testURI, line: 80, startCharacter: 26, endCharacter: 30},
			},
		},
		{
			name:      "rename_extension_message_type",
			targetURI: extensionsURI,
			line:      10,
			character: 8,
			newName:   "Test",
			expectedEdits: []editLocation{
				{uri: extensionsURI, line: 10, startCharacter: 8, endCharacter: 15},
				{uri: extensionsURI, line: 7, startCharacter: 2, endCharacter: 9},
			},
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
							newText:        edit.NewText,
						})
					}
				}
				if len(allEdits) != len(tt.expectedEdits) {
					t.Logf("Expected %d edits, got %d", len(tt.expectedEdits), len(allEdits))
					for _, e := range allEdits {
						t.Logf("  actual: %s:%d:%d-%d (newText: %q)", e.uri, e.line, e.startCharacter, e.endCharacter, e.newText)
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
					if expectedEdit.newText != "" {
						assert.Equal(t, expectedEdit.newText, allEdits[idx].newText, "NewText mismatch at %s:%d:%d", expectedEdit.uri, expectedEdit.line, expectedEdit.startCharacter)
					}
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
			line:        8,
			character:   8,
			expectRange: true,
			expectedStart: protocol.Position{
				Line:      8,
				Character: 8,
			},
			expectedEnd: protocol.Position{
				Line:      8,
				Character: 15,
			},
		},
		{
			name:        "prepare_rename_field",
			line:        10,
			character:   9,
			expectRange: true,
			expectedStart: protocol.Position{
				Line:      10,
				Character: 9,
			},
			expectedEnd: protocol.Position{
				Line:      10,
				Character: 13,
			},
		},
		{
			name:        "prepare_rename_on_keyword",
			line:        8,
			character:   0,
			expectRange: false,
		},
		{
			name:        "prepare_rename_on_primitive_type",
			line:        9,
			character:   2,
			expectRange: false,
		},
		{
			name:        "prepare_rename_on_field_number",
			line:        9,
			character:   14,
			expectRange: false,
		},
		{
			name:        "prepare_rename_on_whitespace",
			line:        9,
			character:   0,
			expectRange: false,
		},
		{
			name:        "prepare_rename_on_extension_usage",
			line:        69,
			character:   11,
			expectRange: false,
		},
		{
			name:        "prepare_rename_on_extension_field_usage",
			line:        69,
			character:   20,
			expectRange: false,
		},
		{
			name:        "prepare_rename_on_qualified_extension_usage",
			line:        80,
			character:   18,
			expectRange: false,
		},
		{
			name:        "prepare_rename_on_qualified_extension_field_usage",
			line:        80,
			character:   27,
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

// Copyright 2020-2026 Buf Technologies, Inc.
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
	"slices"
	"testing"

	"github.com/bufbuild/buf/private/buf/buflsp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

func TestReferences(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	testProtoPath, err := filepath.Abs("testdata/references/references.proto")
	require.NoError(t, err)

	typesProtoPath, err := filepath.Abs("testdata/references/types.proto")
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)
	typesURI := buflsp.FilePathToURI(typesProtoPath)

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

			locations := testRequestReferences(ctx, t, clientJSONConn, tt.targetURI, tt.line, tt.character, tt.includeDeclaration)

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

func TestReferencesToDependency(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	firstProtoPath, err := filepath.Abs("testdata/references_dependency/first.proto")
	require.NoError(t, err)

	secondProtoPath, err := filepath.Abs("testdata/references_dependency/second.proto")
	require.NoError(t, err)

	sharedProtoPath, err := filepath.Abs("testdata/references_dependency/shared.proto")
	require.NoError(t, err)

	// Only first.proto is opened. second.proto and shared.proto are indexed as part of the
	// workspace and must still contribute and receive references.
	clientJSONConn, firstURI := setupLSPServer(t, firstProtoPath)
	secondURI := buflsp.FilePathToURI(secondProtoPath)
	sharedURI := buflsp.FilePathToURI(sharedProtoPath)

	type refLocation struct {
		uri  protocol.URI
		line uint32
	}
	tests := []struct {
		name               string
		line               uint32
		character          uint32
		includeDeclaration bool
		expectedReferences []refLocation
	}{
		{
			name:               "local_dependency_across_files",
			line:               8, // Shared shared = 1;
			character:          3,
			includeDeclaration: false,
			expectedReferences: []refLocation{
				{uri: firstURI, line: 8},  // Shared shared
				{uri: firstURI, line: 10}, // repeated Shared others
				{uri: secondURI, line: 8}, // Shared shared, in the unopened file
			},
		},
		{
			name:               "local_dependency_with_declaration",
			line:               8,
			character:          3,
			includeDeclaration: true,
			expectedReferences: []refLocation{
				{uri: firstURI, line: 8},
				{uri: firstURI, line: 10},
				{uri: secondURI, line: 8},
				{uri: sharedURI, line: 4}, // message Shared
			},
		},
		{
			name:               "wellknown_type_across_files",
			line:               9, // google.protobuf.Timestamp created_at = 2;
			character:          20,
			includeDeclaration: false,
			expectedReferences: []refLocation{
				{uri: firstURI, line: 9},
				{uri: secondURI, line: 9},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			locations := testRequestReferences(ctx, t, clientJSONConn, firstURI, tt.line, tt.character, tt.includeDeclaration)

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

func TestReferencesStableAcrossReindex(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	firstProtoPath, err := filepath.Abs("testdata/references_dependency/first.proto")
	require.NoError(t, err)

	clientJSONConn, firstURI := setupLSPServer(t, firstProtoPath)

	firstProtoContent, err := os.ReadFile(firstProtoPath)
	require.NoError(t, err)

	// Position of the google.protobuf.Timestamp reference in first.proto.
	const timestampLine, timestampCharacter = 9, 20

	before := testRequestReferences(ctx, t, clientJSONConn, firstURI, timestampLine, timestampCharacter, true)
	require.NotEmpty(t, before)

	// Re-send the file unchanged. This forces a full re-index without moving any position, so
	// the reference set must come back identical.
	require.NoError(t, clientJSONConn.Notify(ctx, protocol.MethodTextDocumentDidChange, &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: firstURI},
			Version:                2,
		},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{{Text: string(firstProtoContent)}},
	}))

	after := testRequestReferences(ctx, t, clientJSONConn, firstURI, timestampLine, timestampCharacter, true)

	assert.ElementsMatch(t, before, after, "reference set changed after re-indexing")
	assert.Len(t, slices.Compact(slices.Clone(after)), len(after), "re-indexing introduced duplicate references")
}

// testRequestReferences sends a textDocument/references request and returns the locations.
func testRequestReferences(
	ctx context.Context,
	t *testing.T,
	clientJSONConn jsonrpc2.Conn,
	uri protocol.URI,
	line uint32,
	character uint32,
	includeDeclaration bool,
) []protocol.Location {
	t.Helper()

	var locations []protocol.Location
	_, err := clientJSONConn.Call(ctx, protocol.MethodTextDocumentReferences, protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: line, Character: character},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: includeDeclaration},
	}, &locations)
	require.NoError(t, err)
	return locations
}

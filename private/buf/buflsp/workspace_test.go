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
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/buf/buflsp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

func TestWorkspaceSymbol(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	testProtoPath, err := filepath.Abs("testdata/workspace_symbols/workspace_symbols.proto")
	require.NoError(t, err)

	typesProtoPath, err := filepath.Abs("testdata/workspace_symbols/types.proto")
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)
	typesURI := buflsp.FilePathToURI(typesProtoPath)

	type symbolInfo struct {
		name       string
		kind       protocol.SymbolKind
		line       uint32
		deprecated bool
		uri        protocol.URI
	}

	tests := []struct {
		name            string
		query           string
		expectedSymbols []symbolInfo // Symbols that should be found with their details
		minResults      int          // Minimum number of results expected
	}{
		{
			name:  "search_for_item",
			query: "Item",
			expectedSymbols: []symbolInfo{
				{name: "workspace_symbols.v1.Item", kind: protocol.SymbolKindClass, line: 6, uri: testURI},
				{name: "workspace_symbols.v1.GetItemRequest", kind: protocol.SymbolKindClass, line: 24, uri: testURI},
				{name: "workspace_symbols.v1.GetItemResponse", kind: protocol.SymbolKindClass, line: 28, uri: testURI},
				{name: "workspace_symbols.v1.ListItemsRequest", kind: protocol.SymbolKindClass, line: 32, uri: testURI},
				{name: "workspace_symbols.v1.ListItemsResponse", kind: protocol.SymbolKindClass, line: 36, uri: testURI},
				{name: "workspace_symbols.v1.ItemService", kind: protocol.SymbolKindInterface, line: 19, uri: testURI},
			},
			minResults: 6,
		},
		{
			name:  "search_for_color",
			query: "Color",
			expectedSymbols: []symbolInfo{
				{name: "workspace_symbols.v1.Color", kind: protocol.SymbolKindEnum, line: 4, uri: typesURI},
				{name: "workspace_symbols.v1.COLOR_UNSPECIFIED", kind: protocol.SymbolKindEnumMember, line: 5, uri: typesURI},
				{name: "workspace_symbols.v1.COLOR_RED", kind: protocol.SymbolKindEnumMember, line: 6, uri: typesURI},
				{name: "workspace_symbols.v1.COLOR_BLUE", kind: protocol.SymbolKindEnumMember, line: 7, uri: typesURI},
			},
			minResults: 4,
		},
		{
			name:  "search_for_label",
			query: "Label",
			expectedSymbols: []symbolInfo{
				{name: "workspace_symbols.v1.Label", kind: protocol.SymbolKindClass, line: 10, uri: typesURI},
			},
			minResults: 1,
		},
		{
			name:  "search_for_container",
			query: "Container",
			expectedSymbols: []symbolInfo{
				{name: "workspace_symbols.v1.Container", kind: protocol.SymbolKindClass, line: 13, uri: testURI},
			},
			minResults: 1,
		},
		{
			name:  "search_for_deprecated",
			query: "Legacy",
			expectedSymbols: []symbolInfo{
				{name: "workspace_symbols.v1.LegacyItem", kind: protocol.SymbolKindClass, line: 40, deprecated: true, uri: testURI},
			},
			minResults: 1,
		},
		{
			name:       "empty_query_returns_all_symbols",
			query:      "",
			minResults: 20, // Should return many symbols from both files
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var symbols []protocol.SymbolInformation
			_, symErr := clientJSONConn.Call(ctx, protocol.MethodWorkspaceSymbol, protocol.WorkspaceSymbolParams{
				Query: tt.query,
			}, &symbols)
			require.NoError(t, symErr)

			assert.GreaterOrEqual(t, len(symbols), tt.minResults)

			for _, expectedSymbol := range tt.expectedSymbols {
				idx := slices.IndexFunc(symbols, func(s protocol.SymbolInformation) bool {
					return s.Name == expectedSymbol.name
				})
				require.NotEqual(t, -1, idx, "expected to find symbol %s", expectedSymbol.name)
				found := symbols[idx]
				assert.Equal(t, expectedSymbol.kind, found.Kind, "symbol %s has wrong kind", expectedSymbol.name)
				assert.Equal(t, expectedSymbol.uri, found.Location.URI, "symbol %s has wrong URI", expectedSymbol.name)
				assert.Equal(t, expectedSymbol.line, found.Location.Range.Start.Line, "symbol %s has wrong line number", expectedSymbol.name)
				assert.Equal(t, expectedSymbol.deprecated, found.Deprecated, "symbol %s has wrong deprecated status", expectedSymbol.name)
			}
		})
	}
}

func TestWorkspaceDependencyFile(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	protoPath, err := filepath.Abs("testdata/hover_dependency/main.proto")
	require.NoError(t, err)
	clientJSONConn, testURI := setupLSPServer(t, protoPath)

	dependencyURI := resolveDependencyURI(ctx, t, clientJSONConn, testURI)
	dependencyPosition := dependencyTypePosition(t, dependencyURI)

	// Completion inside the dependency file works before it is even opened.
	items := requestCompletion(ctx, t, clientJSONConn, dependencyURI, dependencyPosition)
	assert.NotEmpty(t, items, "expected completions inside the dependency file")

	// Opening the dependency file in the editor keeps it working.
	openFileFromDisk(ctx, t, clientJSONConn, dependencyURI)
	items = requestCompletion(ctx, t, clientJSONConn, dependencyURI, dependencyPosition)
	assert.NotEmpty(t, items, "expected completions in the opened dependency file")
}

func TestWorkspaceReleasedOnClose(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	protoPath, err := filepath.Abs("testdata/hover_dependency/main.proto")
	require.NoError(t, err)
	clientJSONConn, testURI := setupLSPServer(t, protoPath)

	// Position of the Api type reference in main.proto.
	position := protocol.Position{Line: 9, Character: 20}
	hover := requestHover(ctx, t, clientJSONConn, testURI, position)
	require.NotNil(t, hover, "expected hover while the file is open")

	// Close and reopen the only open document twice. The workspace must be
	// dropped and recreated each time.
	for range 2 {
		closeFile(ctx, t, clientJSONConn, testURI)

		// The file is evicted with its workspace, so hover has nothing to
		// answer from.
		hover = requestHover(ctx, t, clientJSONConn, testURI, position)
		assert.Nil(t, hover, "expected no hover after the last open document closed")

		openFileFromDisk(ctx, t, clientJSONConn, testURI)
		hover = requestHover(ctx, t, clientJSONConn, testURI, position)
		assert.NotNil(t, hover, "expected hover after reopening")
	}
}

func TestWorkspaceSurvivesCloseOrder(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	protoPath, err := filepath.Abs("testdata/hover_dependency/main.proto")
	require.NoError(t, err)
	clientJSONConn, testURI := setupLSPServer(t, protoPath)

	dependencyURI := resolveDependencyURI(ctx, t, clientJSONConn, testURI)
	dependencyPosition := dependencyTypePosition(t, dependencyURI)
	openFileFromDisk(ctx, t, clientJSONConn, dependencyURI)

	// Close the main file first. The workspace must survive on the dependency
	// file's lease.
	closeFile(ctx, t, clientJSONConn, testURI)
	items := requestCompletion(ctx, t, clientJSONConn, dependencyURI, dependencyPosition)
	assert.NotEmpty(t, items, "expected completions while the dependency file is still open")

	// Closing the dependency file drops the last lease and evicts everything.
	closeFile(ctx, t, clientJSONConn, dependencyURI)
	items = requestCompletion(ctx, t, clientJSONConn, dependencyURI, dependencyPosition)
	assert.Empty(t, items, "expected no completions after the last open document closed")
}

// resolveDependencyURI follows the import in main.proto to the well-known
// type, as an editor would before opening it.
func resolveDependencyURI(
	ctx context.Context,
	t *testing.T,
	clientJSONConn jsonrpc2.Conn,
	testURI protocol.URI,
) protocol.URI {
	t.Helper()

	var locations []protocol.Location
	_, err := clientJSONConn.Call(ctx, protocol.MethodTextDocumentDefinition, protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: testURI},
			Position:     protocol.Position{Line: 6, Character: 12}, // import "google/protobuf/api.proto";
		},
	}, &locations)
	require.NoError(t, err)
	require.Len(t, locations, 1)
	return locations[0].URI
}

// dependencyTypePosition finds a position on a type reference in the
// dependency file, so completion there suggests types. The line is found
// rather than hardcoded so the test survives a well-known type update.
func dependencyTypePosition(t *testing.T, dependencyURI protocol.URI) protocol.Position {
	t.Helper()

	text, err := os.ReadFile(dependencyURI.Filename())
	require.NoError(t, err)
	var line uint32
	for fileLine := range strings.SplitSeq(string(text), "\n") {
		if strings.HasPrefix(strings.TrimSpace(fileLine), "SourceContext source_context") {
			// Position within the SourceContext type name, past the indentation.
			var indent uint32
			for _, r := range fileLine {
				if r != ' ' && r != '\t' {
					break
				}
				indent++
			}
			return protocol.Position{Line: line, Character: indent + 5}
		}
		line++
	}
	t.Fatalf("no type reference found in %s", dependencyURI.Filename())
	return protocol.Position{}
}

// openFileFromDisk opens the file in the editor with its on-disk contents.
func openFileFromDisk(
	ctx context.Context,
	t *testing.T,
	clientJSONConn jsonrpc2.Conn,
	uri protocol.URI,
) {
	t.Helper()

	text, err := os.ReadFile(uri.Filename())
	require.NoError(t, err)
	require.NoError(t, clientJSONConn.Notify(ctx, protocol.MethodTextDocumentDidOpen, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "protobuf",
			Version:    1,
			Text:       string(text),
		},
	}))
}

// closeFile closes the file in the editor.
func closeFile(
	ctx context.Context,
	t *testing.T,
	clientJSONConn jsonrpc2.Conn,
	uri protocol.URI,
) {
	t.Helper()

	require.NoError(t, clientJSONConn.Notify(ctx, protocol.MethodTextDocumentDidClose, &protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	}))
}

// requestCompletion requests completion items at the given position.
func requestCompletion(
	ctx context.Context,
	t *testing.T,
	clientJSONConn jsonrpc2.Conn,
	uri protocol.URI,
	position protocol.Position,
) []protocol.CompletionItem {
	t.Helper()

	var completionList *protocol.CompletionList
	_, err := clientJSONConn.Call(ctx, protocol.MethodTextDocumentCompletion, protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position,
		},
	}, &completionList)
	require.NoError(t, err)
	if completionList == nil {
		return nil
	}
	return completionList.Items
}

// requestHover requests hover contents at the given position.
func requestHover(
	ctx context.Context,
	t *testing.T,
	clientJSONConn jsonrpc2.Conn,
	uri protocol.URI,
	position protocol.Position,
) *protocol.Hover {
	t.Helper()

	var hover *protocol.Hover
	_, err := clientJSONConn.Call(ctx, protocol.MethodTextDocumentHover, protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position,
		},
	}, &hover)
	require.NoError(t, err)
	return hover
}

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

	"github.com/bufbuild/buf/private/buf/buflsp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

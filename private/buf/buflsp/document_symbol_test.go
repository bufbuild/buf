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
)

func TestDocumentSymbol(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	testProtoPath, err := filepath.Abs("testdata/symbols/symbols.proto")
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)

	type symbolInfo struct {
		name       string
		kind       protocol.SymbolKind
		line       uint32
		deprecated bool
	}
	tests := []struct {
		name            string
		expectedSymbols []symbolInfo
	}{
		{
			name: "all_document_symbols",
			expectedSymbols: []symbolInfo{
				{name: "symbols.v1.Document", kind: protocol.SymbolKindClass, line: 4},                          // message Document
				{name: "symbols.v1.Document.id", kind: protocol.SymbolKindField, line: 5},                       // string id
				{name: "symbols.v1.Document.title", kind: protocol.SymbolKindField, line: 6},                    // string title
				{name: "symbols.v1.Document.status", kind: protocol.SymbolKindField, line: 7},                   // Status status
				{name: "symbols.v1.Document.metadata", kind: protocol.SymbolKindField, line: 8},                 // Metadata metadata
				{name: "symbols.v1.Document.Metadata", kind: protocol.SymbolKindClass, line: 9},                 // message Metadata (nested)
				{name: "symbols.v1.Document.Metadata.author", kind: protocol.SymbolKindField, line: 10},         // string author
				{name: "symbols.v1.Document.Metadata.created_at", kind: protocol.SymbolKindField, line: 11},     // int64 created_at
				{name: "symbols.v1.Status", kind: protocol.SymbolKindEnum, line: 15},                            // enum Status
				{name: "symbols.v1.STATUS_UNSPECIFIED", kind: protocol.SymbolKindEnumMember, line: 16},          // STATUS_UNSPECIFIED = 0
				{name: "symbols.v1.STATUS_DRAFT", kind: protocol.SymbolKindEnumMember, line: 17},                // STATUS_DRAFT = 1
				{name: "symbols.v1.STATUS_PUBLISHED", kind: protocol.SymbolKindEnumMember, line: 18},            // STATUS_PUBLISHED = 2
				{name: "symbols.v1.DocumentService", kind: protocol.SymbolKindInterface, line: 21},              // service DocumentService
				{name: "symbols.v1.DocumentService.GetDocument", kind: protocol.SymbolKindMethod, line: 22},     // rpc GetDocument
				{name: "symbols.v1.DocumentService.CreateDocument", kind: protocol.SymbolKindMethod, line: 23},  // rpc CreateDocument
				{name: "symbols.v1.GetDocumentRequest", kind: protocol.SymbolKindClass, line: 26},               // message GetDocumentRequest
				{name: "symbols.v1.GetDocumentRequest.document_id", kind: protocol.SymbolKindField, line: 27},   // string document_id
				{name: "symbols.v1.GetDocumentResponse", kind: protocol.SymbolKindClass, line: 30},              // message GetDocumentResponse
				{name: "symbols.v1.GetDocumentResponse.document", kind: protocol.SymbolKindField, line: 31},     // Document document
				{name: "symbols.v1.CreateDocumentRequest", kind: protocol.SymbolKindClass, line: 34},            // message CreateDocumentRequest
				{name: "symbols.v1.CreateDocumentRequest.document", kind: protocol.SymbolKindField, line: 35},   // Document document
				{name: "symbols.v1.CreateDocumentResponse", kind: protocol.SymbolKindClass, line: 38},           // message CreateDocumentResponse
				{name: "symbols.v1.CreateDocumentResponse.document", kind: protocol.SymbolKindField, line: 39},  // Document document
				{name: "symbols.v1.LegacyDocument", kind: protocol.SymbolKindClass, line: 42, deprecated: true}, // message LegacyDocument (deprecated)
				{name: "symbols.v1.LegacyDocument.id", kind: protocol.SymbolKindField, line: 44},                // string id
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var symbols []protocol.SymbolInformation
			_, symErr := clientJSONConn.Call(ctx, protocol.MethodTextDocumentDocumentSymbol, protocol.DocumentSymbolParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: testURI,
				},
			}, &symbols)
			require.NoError(t, symErr)

			require.Len(t, symbols, len(tt.expectedSymbols))

			for _, expectedSymbol := range tt.expectedSymbols {
				idx := slices.IndexFunc(symbols, func(s protocol.SymbolInformation) bool {
					return s.Name == expectedSymbol.name
				})
				require.NotEqual(t, -1, idx, "expected to find symbol %s", expectedSymbol.name)
				found := symbols[idx]
				assert.Equal(t, expectedSymbol.kind, found.Kind, "symbol %s has wrong kind", expectedSymbol.name)
				assert.Equal(t, testURI, found.Location.URI, "symbol %s has wrong URI", expectedSymbol.name)
				assert.Equal(t, expectedSymbol.line, found.Location.Range.Start.Line, "symbol %s has wrong line number", expectedSymbol.name)
				assert.Equal(t, expectedSymbol.deprecated, found.Deprecated, "symbol %s has wrong deprecated status", expectedSymbol.name)
			}
		})
	}
}

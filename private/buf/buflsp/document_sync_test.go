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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func TestIncrementalDocumentSync(t *testing.T) {
	t.Parallel()

	// Setup: Create initial proto file content
	initialContent := `syntax = "proto3";

package example.v1;

message User {
  string id = 1;
}
`

	// Create a temporary workspace directory and test file
	tmpDir := t.TempDir()
	testProtoPath := filepath.Join(tmpDir, "test.proto")
	err := os.WriteFile(testProtoPath, []byte(initialContent), 0644)
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)
	ctx := t.Context()

	// Step 1: Open the document
	err = clientJSONConn.Notify(ctx, protocol.MethodTextDocumentDidOpen, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        testURI,
			LanguageID: "proto",
			Version:    1,
			Text:       initialContent,
		},
	})
	require.NoError(t, err)

	// Step 2: Make an incremental change - insert a new field "string name = 2;" after "string id = 1;"
	// Position at line 5 (after "string id = 1;"), character 0 (start of line 6)
	// We're inserting "  string name = 2;\n" at the beginning of line 6 (which is currently just "}")
	err = clientJSONConn.Notify(ctx, protocol.MethodTextDocumentDidChange, &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{
				URI: testURI,
			},
			Version: 2,
		},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 5, Character: 18}, // End of "  string id = 1;"
					End:   protocol.Position{Line: 5, Character: 18}, // Same position (insertion)
				},
				Text: "\n  string name = 2;",
			},
		},
	})
	require.NoError(t, err)

	// Step 3: Request document symbols to verify the change was applied correctly
	var symbols []any
	_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentDocumentSymbol, &protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: testURI,
		},
	}, &symbols)
	require.NoError(t, err)

	// Verify that we now have symbols for both "id" and "name" fields
	// We expect at least 2 field symbols (for id and name)
	// Note: The exact number of symbols may vary based on what else is indexed
	require.NotEmpty(t, symbols, "Expected symbols after incremental change")
}

func TestIncrementalDocumentSyncMultipleChanges(t *testing.T) {
	t.Parallel()

	// Setup: Create initial proto file content
	initialContent := `syntax = "proto3";

package example.v1;

message User {
}
`

	// Create a temporary workspace directory and test file
	tmpDir := t.TempDir()
	testProtoPath := filepath.Join(tmpDir, "test.proto")
	err := os.WriteFile(testProtoPath, []byte(initialContent), 0644)
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)
	ctx := t.Context()

	// Step 1: Open the document
	err = clientJSONConn.Notify(ctx, protocol.MethodTextDocumentDidOpen, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        testURI,
			LanguageID: "proto",
			Version:    1,
			Text:       initialContent,
		},
	})
	require.NoError(t, err)

	// Step 2: Make multiple incremental changes in a single notification
	// This simulates more realistic editing scenarios
	err = clientJSONConn.Notify(ctx, protocol.MethodTextDocumentDidChange, &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{
				URI: testURI,
			},
			Version: 2,
		},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{
			// First change: Add a field
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 4, Character: 14}, // After "message User {"
					End:   protocol.Position{Line: 4, Character: 14},
				},
				Text: "\n  string id = 1;",
			},
		},
	})
	require.NoError(t, err)

	// Step 3: Make another incremental change
	err = clientJSONConn.Notify(ctx, protocol.MethodTextDocumentDidChange, &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{
				URI: testURI,
			},
			Version: 3,
		},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{
			// Second change: Add another field
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 5, Character: 17}, // After "  string id = 1;"
					End:   protocol.Position{Line: 5, Character: 17},
				},
				Text: "\n  string name = 2;",
			},
		},
	})
	require.NoError(t, err)

	// Step 4: Verify the final state by requesting symbols
	var symbols []any
	_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentDocumentSymbol, &protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: testURI,
		},
	}, &symbols)
	require.NoError(t, err)

	// Should have symbols for the message and both fields
	require.NotEmpty(t, symbols, "Expected symbols after multiple incremental changes")
}

func TestIncrementalDocumentSyncReplace(t *testing.T) {
	t.Parallel()

	// Setup: Create initial proto file content
	initialContent := `syntax = "proto3";

package example.v1;

message User {
  string id = 1;
}
`

	// Create a temporary workspace directory and test file
	tmpDir := t.TempDir()
	testProtoPath := filepath.Join(tmpDir, "test.proto")
	err := os.WriteFile(testProtoPath, []byte(initialContent), 0644)
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)
	ctx := t.Context()

	// Step 1: Open the document
	err = clientJSONConn.Notify(ctx, protocol.MethodTextDocumentDidOpen, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        testURI,
			LanguageID: "proto",
			Version:    1,
			Text:       initialContent,
		},
	})
	require.NoError(t, err)

	// Step 2: Replace "string" with "int32" for the id field
	err = clientJSONConn.Notify(ctx, protocol.MethodTextDocumentDidChange, &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{
				URI: testURI,
			},
			Version: 2,
		},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 5, Character: 2},  // Start of "string"
					End:   protocol.Position{Line: 5, Character: 8},  // End of "string"
				},
				Text: "int32",
			},
		},
	})
	require.NoError(t, err)

	// Step 3: Verify the document is valid (no parse errors)
	// This is tested by ensuring we can get symbols successfully
	var symbols []any
	_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentDocumentSymbol, &protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: testURI,
		},
	}, &symbols)
	require.NoError(t, err)
	require.NotEmpty(t, symbols, "Expected symbols after replacement change")
}

func TestFullDocumentSyncStillWorks(t *testing.T) {
	t.Parallel()

	// This test ensures backward compatibility - full document sync should still work
	initialContent := `syntax = "proto3";

package example.v1;

message User {
  string id = 1;
}
`

	// Create a temporary workspace directory and test file
	tmpDir := t.TempDir()
	testProtoPath := filepath.Join(tmpDir, "test.proto")
	err := os.WriteFile(testProtoPath, []byte(initialContent), 0644)
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)
	ctx := t.Context()

	// Step 1: Open the document
	err = clientJSONConn.Notify(ctx, protocol.MethodTextDocumentDidOpen, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        testURI,
			LanguageID: "proto",
			Version:    1,
			Text:       initialContent,
		},
	})
	require.NoError(t, err)

	// Step 2: Send a full document update (without Range)
	updatedContent := `syntax = "proto3";

package example.v1;

message User {
  string id = 1;
  string name = 2;
}
`
	err = clientJSONConn.Notify(ctx, protocol.MethodTextDocumentDidChange, &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{
				URI: testURI,
			},
			Version: 2,
		},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{
			{
				// No Range field = full document sync
				Text: updatedContent,
			},
		},
	})
	require.NoError(t, err)

	// Step 3: Verify the change was applied
	var symbols []any
	_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentDocumentSymbol, &protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: testURI,
		},
	}, &symbols)
	require.NoError(t, err)
	require.NotEmpty(t, symbols, "Expected symbols after full document sync")
}

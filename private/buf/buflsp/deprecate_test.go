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
	"testing"

	"github.com/bufbuild/buf/private/buf/buflsp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func TestCodeAction_Deprecate_Message(t *testing.T) {
	t.Parallel()

	testProtoPath, err := filepath.Abs("testdata/deprecate/message_test.proto")
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)

	// Request code actions at the message declaration (line 5: "message MyMessage {")
	// Position is 0-indexed, so line 5 in editor = line 4 here
	var codeActions []protocol.CodeAction
	_, err = clientJSONConn.Call(t.Context(), protocol.MethodTextDocumentCodeAction, protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: testURI,
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8}, // On "MyMessage"
			End:   protocol.Position{Line: 4, Character: 8},
		},
		Context: protocol.CodeActionContext{
			Only: []protocol.CodeActionKind{
				buflsp.CodeActionKindSourceDeprecate,
			},
		},
	}, &codeActions)
	require.NoError(t, err)

	// Find the deprecate code action
	var deprecateAction *protocol.CodeAction
	for _, codeAction := range codeActions {
		if codeAction.Kind == buflsp.CodeActionKindSourceDeprecate {
			deprecateAction = &codeAction
			break
		}
	}
	require.NotNil(t, deprecateAction, "expected to find deprecate code action, got actions: %v", codeActions)

	// Verify the code action properties
	assert.Equal(t, buflsp.CodeActionKindSourceDeprecate, deprecateAction.Kind, "code action should be source.deprecate")
	assert.NotNil(t, deprecateAction.Edit, "code action should have a workspace edit")
	assert.Contains(t, deprecateAction.Title, "Deprecate", "title should mention 'Deprecate'")

	// Verify the edit contains changes for the current file
	require.NotNil(t, deprecateAction.Edit.Changes, "workspace edit should have changes")
	actualEdits, ok := deprecateAction.Edit.Changes[testURI]
	require.True(t, ok, "workspace edit should have changes for the current file")
	require.NotEmpty(t, actualEdits, "should have text edits")

	// Verify that at least one edit inserts "deprecated = true"
	hasDeprecatedEdit := false
	for _, edit := range actualEdits {
		if edit.NewText != "" && (contains(edit.NewText, "deprecated = true") || contains(edit.NewText, "option deprecated = true")) {
			hasDeprecatedEdit = true
			break
		}
	}
	assert.True(t, hasDeprecatedEdit, "should have an edit that adds deprecation option")
}

func TestCodeAction_Deprecate_Enum(t *testing.T) {
	t.Parallel()

	testProtoPath, err := filepath.Abs("testdata/deprecate/message_test.proto")
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)

	// Request code actions at the enum declaration (line 10: "enum MyEnum {")
	var codeActions []protocol.CodeAction
	_, err = clientJSONConn.Call(t.Context(), protocol.MethodTextDocumentCodeAction, protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: testURI,
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 9, Character: 5}, // On "MyEnum"
			End:   protocol.Position{Line: 9, Character: 5},
		},
		Context: protocol.CodeActionContext{
			Only: []protocol.CodeActionKind{
				buflsp.CodeActionKindSourceDeprecate,
			},
		},
	}, &codeActions)
	require.NoError(t, err)

	// Find the deprecate code action
	var deprecateAction *protocol.CodeAction
	for _, codeAction := range codeActions {
		if codeAction.Kind == buflsp.CodeActionKindSourceDeprecate {
			deprecateAction = &codeAction
			break
		}
	}
	require.NotNil(t, deprecateAction, "expected to find deprecate code action for enum")
	assert.Contains(t, deprecateAction.Title, "MyEnum", "title should mention the enum name")
}

func TestCodeAction_Deprecate_Service(t *testing.T) {
	t.Parallel()

	testProtoPath, err := filepath.Abs("testdata/deprecate/message_test.proto")
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)

	// Request code actions at the service declaration (line 15: "service MyService {")
	var codeActions []protocol.CodeAction
	_, err = clientJSONConn.Call(t.Context(), protocol.MethodTextDocumentCodeAction, protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: testURI,
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 14, Character: 8}, // On "MyService"
			End:   protocol.Position{Line: 14, Character: 8},
		},
		Context: protocol.CodeActionContext{
			Only: []protocol.CodeActionKind{
				buflsp.CodeActionKindSourceDeprecate,
			},
		},
	}, &codeActions)
	require.NoError(t, err)

	// Find the deprecate code action
	var deprecateAction *protocol.CodeAction
	for _, codeAction := range codeActions {
		if codeAction.Kind == buflsp.CodeActionKindSourceDeprecate {
			deprecateAction = &codeAction
			break
		}
	}
	require.NotNil(t, deprecateAction, "expected to find deprecate code action for service")
	assert.Contains(t, deprecateAction.Title, "MyService", "title should mention the service name")
}

func TestCodeAction_Deprecate_NoAction_OnWhitespace(t *testing.T) {
	t.Parallel()

	testProtoPath, err := filepath.Abs("testdata/deprecate/message_test.proto")
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)

	// Request code actions on an empty line (line 4: empty)
	var codeActions []protocol.CodeAction
	_, err = clientJSONConn.Call(t.Context(), protocol.MethodTextDocumentCodeAction, protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: testURI,
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 3, Character: 0}, // Empty line after package
			End:   protocol.Position{Line: 3, Character: 0},
		},
		Context: protocol.CodeActionContext{
			Only: []protocol.CodeActionKind{
				buflsp.CodeActionKindSourceDeprecate,
			},
		},
	}, &codeActions)
	require.NoError(t, err)

	// Should not find a deprecate action on whitespace
	var deprecateAction *protocol.CodeAction
	for _, codeAction := range codeActions {
		if codeAction.Kind == buflsp.CodeActionKindSourceDeprecate {
			deprecateAction = &codeAction
			break
		}
	}
	assert.Nil(t, deprecateAction, "should not find deprecate code action on whitespace")
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

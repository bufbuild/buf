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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func TestCodeAction_OrganizeImports(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	testProtoPath, err := filepath.Abs("testdata/code_action/import_test.proto")
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)

	// Request code actions for the file
	// The specific position doesn't matter since we're checking for file-level actions
	var codeActions []protocol.CodeAction
	_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentCodeAction, protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: testURI,
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 8, Character: 2}, // Position on "User user = 1;"
			End:   protocol.Position{Line: 8, Character: 2},
		},
		Context: protocol.CodeActionContext{
			Only: []protocol.CodeActionKind{
				protocol.SourceOrganizeImports,
			},
		},
	}, &codeActions)
	require.NoError(t, err)

	// Find the "Organize imports" code action
	var organizeImportsAction *protocol.CodeAction
	for _, codeAction := range codeActions {
		if codeAction.Title == "Organize imports" {
			organizeImportsAction = &codeAction
			break
		}
	}
	require.NotNil(t, organizeImportsAction, "expected to find 'Organize imports' code action, got actions: %v", codeActions)

	// Verify the code action properties
	assert.Equal(t, protocol.QuickFix, organizeImportsAction.Kind, "code action should be a quick fix")
	assert.NotNil(t, organizeImportsAction.Edit, "code action should have a workspace edit")

	// Verify the edit adds the imports
	require.NotNil(t, organizeImportsAction.Edit.Changes, "workspace edit should have changes")
	changes, ok := organizeImportsAction.Edit.Changes[testURI]
	require.True(t, ok, "workspace edit should have changes for the current file")
	require.Len(t, changes, 1, "should have exactly one text edit (all imports together)")

	importEdit := changes[0]

	// Verify the edit is at the expected location (after existing imports, line 6)
	// Note: Line 5 has "import existing_field.proto", so new imports go after that
	assert.Equal(t, uint32(6), importEdit.Range.Start.Line, "imports should be added at line 6 (after existing imports)")

	// Verify the edit contains all required import statements
	// The imports should be sorted alphabetically
	newText := importEdit.NewText

	// Check that all imports are present - one for each scenario:
	// - TopLevelField (toplevel_field.proto)
	// - NestedField (nested_field.proto)
	// - MethodInputType (method_input.proto)
	// - MethodOutputType (method_output.proto)
	// - google.protobuf.Empty (google/protobuf/empty.proto)
	// - google.protobuf.Timestamp (google/protobuf/timestamp.proto)
	// - custom_option (custom_option.proto)
	assert.Contains(t, newText, `import "custom_option.proto";`, "edit should contain custom_option.proto import (for custom option)")
	assert.Contains(t, newText, `import "google/protobuf/empty.proto";`, "edit should contain google/protobuf/empty.proto import (for RPC input)")
	assert.Contains(t, newText, `import "google/protobuf/timestamp.proto";`, "edit should contain google/protobuf/timestamp.proto import (for RPC output)")
	assert.Contains(t, newText, `import "method_input.proto";`, "edit should contain method_input.proto import (for RPC input)")
	assert.Contains(t, newText, `import "method_output.proto";`, "edit should contain method_output.proto import (for RPC output)")
	assert.Contains(t, newText, `import "nested_field.proto";`, "edit should contain nested_field.proto import (for nested message field)")
	assert.Contains(t, newText, `import "toplevel_field.proto";`, "edit should contain toplevel_field.proto import (for top-level field)")

	// Verify imports are sorted alphabetically
	customOptionIndex := strings.Index(newText, `import "custom_option.proto";`)
	googleEmptyIndex := strings.Index(newText, `import "google/protobuf/empty.proto";`)
	googleTimestampIndex := strings.Index(newText, `import "google/protobuf/timestamp.proto";`)
	methodInputIndex := strings.Index(newText, `import "method_input.proto";`)
	methodOutputIndex := strings.Index(newText, `import "method_output.proto";`)
	nestedFieldIndex := strings.Index(newText, `import "nested_field.proto";`)
	topLevelFieldIndex := strings.Index(newText, `import "toplevel_field.proto";`)

	assert.Less(t, customOptionIndex, googleEmptyIndex, "custom_option should come before google/protobuf/empty")
	assert.Less(t, googleEmptyIndex, googleTimestampIndex, "google/protobuf/empty should come before google/protobuf/timestamp")
	assert.Less(t, googleTimestampIndex, methodInputIndex, "google/protobuf/timestamp should come before method_input")
	assert.Less(t, methodInputIndex, methodOutputIndex, "method_input should come before method_output")
	assert.Less(t, methodOutputIndex, nestedFieldIndex, "method_output should come before nested_field")
	assert.Less(t, nestedFieldIndex, topLevelFieldIndex, "nested_field should come before toplevel_field")
}

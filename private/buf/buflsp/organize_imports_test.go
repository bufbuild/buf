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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func TestCodeAction_OrganizeImports(t *testing.T) {
	t.Parallel()

	testCodeActionOrganizeImports(
		t,
		"testdata/organize_imports/import_test.proto",
		[]protocol.TextEdit{
			// Delete #1: Remove unused import (file line 5)
			// Deletes: `import "google/protobuf/cpp_features.proto"; // unused import\n`
			// This import is marked as unused by IR diagnostics and should be removed
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 4, Character: 0},
					End:   protocol.Position{Line: 5, Character: 0},
				},
				NewText: "",
			},
			// Delete #2: Remove existing import that will be re-added in sorted order (file lines 6-7)
			// Deletes: `import "types/existing_field.proto";\n\n`
			// This import is kept but will be re-inserted in alphabetically sorted position
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 5, Character: 0},
					End:   protocol.Position{Line: 7, Character: 0},
				},
				NewText: "",
			},
			// Insert organized imports after package declaration (file line 4, after line 3)
			// Adds 7 new imports (empty, timestamp, custom_option, method_input, method_output, nested, toplevel)
			// Plus re-adds existing_field in sorted position
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 3, Character: 0},
					End:   protocol.Position{Line: 3, Character: 0},
				},
				NewText: `
import "google/protobuf/empty.proto";
import "google/protobuf/timestamp.proto";
import "options/custom_option.proto";
import "types/existing_field.proto";
import "types/method_input.proto";
import "types/method_output.proto";
import "types/nested_field.proto";
import "types/toplevel_field.proto";
`,
			},
		},
	)

	testCodeActionOrganizeImports(
		t,
		"testdata/organize_imports/comments_test.proto",
		[]protocol.TextEdit{
			// Delete #1: Remove first import with comments (file lines 5-6)
			// Deletes: `// A comment about this\nimport "types/existing_field.proto"; // trailing comment\n`
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 4, Character: 0},
					End:   protocol.Position{Line: 6, Character: 0},
				},
				NewText: "",
			},
			// Delete #2: Remove second import with multi-line comment (file lines 7-11)
			// Deletes: `/* MultiComment\n * \n */\nimport "google/protobuf/timestamp.proto";\n\n`
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 6, Character: 0},
					End:   protocol.Position{Line: 11, Character: 0},
				},
				NewText: "",
			},
			// Delete #3: Remove third import not at the top of the file (file lines 19-20)
			// Deletes: `// Import is not at top of file\nimport "types/notattop_import.proto"; // another trailing comment`
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 18, Character: 0},
					End:   protocol.Position{Line: 20, Character: 0},
				},
				NewText: "",
			},
			// Insert organized imports with all comments preserved in sorted order
			// Adds: notattop_import (moved from line 21), toplevel_field (new)
			// Re-adds: timestamp, existing_field with their comments attached
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 3, Character: 0},
					End:   protocol.Position{Line: 3, Character: 0},
				},
				NewText: `
/* MultiComment
 * 
 */
import "google/protobuf/timestamp.proto";
// A comment about this
import "types/existing_field.proto"; // trailing comment
// Import is not at top of file
import "types/notattop_import.proto"; // another trailing comment
import "types/toplevel_field.proto";
`,
			},
		},
	)

	testCodeActionOrganizeImports(
		t,
		"testdata/organize_imports/modifier_test.proto",
		[]protocol.TextEdit{
			// Delete #1: Remove public import with comment (file lines 8-9)
			// Deletes: `// Comment on public import\nimport public "types/modifier_public.proto";`
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 7, Character: 0},
					End:   protocol.Position{Line: 9, Character: 0},
				},
				NewText: "",
			},
			// Delete #2: Remove unused weak import (file line 5)
			// Deletes: `import weak "google/protobuf/cpp_features.proto"; // unused import\n`
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 4, Character: 0},
					End:   protocol.Position{Line: 5, Character: 0},
				},
				NewText: "",
			},
			// Delete #3: Remove weak import with comment (file lines 6-7)
			// Deletes: `// Comment on weak import\nimport weak "types/modifier_weak.proto";\n`
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 5, Character: 0},
					End:   protocol.Position{Line: 7, Character: 0},
				},
				NewText: "",
			},
			// Delete #4: Remove existing_field import (file lines 10-11)
			// Deletes: `import "types/existing_field.proto";\n\n`
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 9, Character: 0},
					End:   protocol.Position{Line: 11, Character: 0},
				},
				NewText: "",
			},
			// Insert organized imports in sorted order
			// Re-adds: existing_field, modifer_public (from line 9, note typo in filename),
			//          modifier_weak with comment and weak modifier
			// Adds: toplevel_field (new)
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 3, Character: 0},
					End:   protocol.Position{Line: 3, Character: 0},
				},
				NewText: `
import "types/existing_field.proto";
// Comment on public import
import public "types/modifier_public.proto";
// Comment on weak import
import weak "types/modifier_weak.proto";
import "types/toplevel_field.proto";
`,
			},
		},
	)

	testCodeActionOrganizeImports(
		t,
		"testdata/organize_imports/noimports_test.proto",
		[]protocol.TextEdit{
			// Insert imports when file has no existing imports
			// File currently has no imports, so nothing to delete
			// Inserts after package declaration (line 3)
			// Adds: existing_field (for ExistingField type), toplevel_field (for TopLevelField type)
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 3, Character: 0},
					End:   protocol.Position{Line: 3, Character: 0},
				},
				NewText: `
import "types/existing_field.proto";
import "types/toplevel_field.proto";
`,
			},
		},
	)

	testCodeActionOrganizeImports(
		t,
		"testdata/organize_imports/nopackage_test.proto",
		[]protocol.TextEdit{
			// Insert imports when file has no package declaration
			// File has no package, so imports are inserted after syntax (line 1)
			// This tests the fallback behavior when package declaration is missing
			// Adds: existing_field (for ExistingField type), toplevel_field (for TopLevelField type)
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 1, Character: 0},
					End:   protocol.Position{Line: 1, Character: 0},
				},
				NewText: `
import "types/existing_field.proto";
import "types/toplevel_field.proto";
`,
			},
		},
	)

	testCodeActionOrganizeImports(
		t,
		"testdata/organize_imports/nochanges_test.proto",
		nil, // No changes are expected.
	)
}

func testCodeActionOrganizeImports(t *testing.T, filename string, expectedEdits []protocol.TextEdit) {
	t.Run(filename, func(t *testing.T) {
		t.Parallel()
		testProtoPath, err := filepath.Abs(filename)
		require.NoError(t, err)

		clientJSONConn, testURI := setupLSPServer(t, testProtoPath)

		// Request code actions for the file
		// The specific position doesn't matter since we're checking for file-level actions
		var codeActions []protocol.CodeAction
		_, err = clientJSONConn.Call(t.Context(), protocol.MethodTextDocumentCodeAction, protocol.CodeActionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: testURI,
			},
			Range: protocol.Range{
				Start: protocol.Position{Line: 8, Character: 2},
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
		if expectedEdits == nil {
			require.Nil(t, organizeImportsAction, "expected no changes to 'Organize imports' code action")
			return
		}
		require.NotNil(t, organizeImportsAction, "expected to find 'Organize imports' code action, got actions: %v", codeActions)

		// Verify the code action properties
		assert.Equal(t, protocol.SourceOrganizeImports, organizeImportsAction.Kind, "code action should be SourceOrganizeImports")
		assert.NotNil(t, organizeImportsAction.Edit, "code action should have a workspace edit")

		// Verify the edit replaces the imports section
		require.NotNil(t, organizeImportsAction.Edit.Changes, "workspace edit should have changes")
		actualEdits, ok := organizeImportsAction.Edit.Changes[testURI]
		require.True(t, ok, "workspace edit should have changes for the current file")
		require.NotEmpty(t, actualEdits, "should have text edits")

		// Validate that actual edits match expected edits
		require.Len(t, actualEdits, len(expectedEdits), "should have %d edits", len(expectedEdits))

		// Compare each edit
		for i, expected := range expectedEdits {
			actual := actualEdits[i]
			assert.Equal(t, expected.Range, actual.Range, "edit %d: range mismatch", i)
			assert.Equal(t, expected.NewText, actual.NewText, "edit %d: newText mismatch", i)
		}
	})
}

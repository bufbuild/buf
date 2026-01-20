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
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/buf/buflsp"
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
			// Deletes: `import weak "google/protobuf/cpp_features.proto"; // unused import\n`
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 4, Character: 0},
					End:   protocol.Position{Line: 5, Character: 0},
				},
				NewText: "",
			},
			// Deletes: `// Comment on weak import\nimport weak "types/modifier_weak.proto";\n`
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 5, Character: 0},
					End:   protocol.Position{Line: 7, Character: 0},
				},
				NewText: "",
			},
			// Deletes: `// Comment on public import\nimport public "types/modifier_public.proto";`
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 7, Character: 0},
					End:   protocol.Position{Line: 9, Character: 0},
				},
				NewText: "",
			},
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

	testCodeActionOrganizeImports(
		t,
		"testdata/organize_imports/unknown_test.proto",
		[]protocol.TextEdit{
			// Delete unknown import (file lines 3-4)
			// Deletes: `import "types/unknown.proto";\n\n`
			// File doesn't exist, not resolved by IR, so it's removed
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 2, Character: 0},
					End:   protocol.Position{Line: 4, Character: 0},
				},
				NewText: "",
			},
		},
	)

	testCodeActionOrganizeImports(
		t,
		"testdata/organize_imports/duplicate_test.proto",
		[]protocol.TextEdit{
			// Delete duplicate #1 (file line 5)
			// Deletes: `import "types/existing_field.proto";\n`
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 4, Character: 0},
					End:   protocol.Position{Line: 5, Character: 0},
				},
				NewText: "",
			},
			// Delete duplicate #2 (file lines 6-7)
			// Deletes: `// A duplicate comment\nimport "types/existing_field.proto";\n`
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 5, Character: 0},
					End:   protocol.Position{Line: 7, Character: 0},
				},
				NewText: "",
			},
			// Delete duplicate #3 (file line 8)
			// Deletes: `import "types/existing_field.proto"; // this also has a trailing comment\n`
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 7, Character: 0},
					End:   protocol.Position{Line: 8, Character: 0},
				},
				NewText: "",
			},
			// Delete duplicate #4 (file line 9)
			// Deletes: `import "types/existing_field.proto";\n`
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 8, Character: 0},
					End:   protocol.Position{Line: 9, Character: 0},
				},
				NewText: "",
			},
			// Delete duplicate #5 (file lines 10-12)
			// Deletes: `// A duplicate comment\nimport "types/existing_field.proto";\n\n`
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 9, Character: 0},
					End:   protocol.Position{Line: 12, Character: 0},
				},
				NewText: "",
			},
			// Insert deduplicated imports (keeps unique versions by text content)
			// After deduplication: plain import, import with trailing comment, import with leading comment
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 3, Character: 0},
					End:   protocol.Position{Line: 3, Character: 0},
				},
				NewText: `
import "types/existing_field.proto"; // this also has a trailing comment
// A duplicate comment
import "types/existing_field.proto";
import "types/existing_field.proto";
`,
			},
		},
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

func TestOrganizeImportsMinimalEdits(t *testing.T) {
	t.Parallel()

	testFiles := []string{
		"testdata/organize_imports/import_test.proto",
		"testdata/organize_imports/comments_test.proto",
		"testdata/organize_imports/modifier_test.proto",
		"testdata/organize_imports/duplicate_test.proto",
	}

	for _, testFile := range testFiles {
		testFile := testFile
		t.Run(testFile, func(t *testing.T) {
			t.Parallel()

			testProtoPath, err := filepath.Abs(testFile)
			require.NoError(t, err)

			clientJSONConn, testURI := setupLSPServer(t, testProtoPath)

			// Get original content
			originalContent, err := os.ReadFile(testProtoPath)
			require.NoError(t, err)

			// Request organize imports code action
			var codeActions []protocol.CodeAction
			_, err = clientJSONConn.Call(t.Context(), protocol.MethodTextDocumentCodeAction, protocol.CodeActionParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: testURI,
				},
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 0, Character: 0},
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

			// Some files may not need any changes
			if organizeImportsAction == nil {
				return
			}

			require.NotNil(t, organizeImportsAction.Edit, "code action should have a workspace edit")
			require.NotNil(t, organizeImportsAction.Edit.Changes, "workspace edit should have changes")

			actualEdits, ok := organizeImportsAction.Edit.Changes[testURI]
			require.True(t, ok, "workspace edit should have changes for the current file")

			if len(actualEdits) == 0 {
				return
			}

			// Verify edits are minimal
			assertOrganizeImportsEditsAreMinimal(t, string(originalContent), actualEdits)
		})
	}
}

// assertOrganizeImportsEditsAreMinimal verifies that organize imports edits are minimal
func assertOrganizeImportsEditsAreMinimal(t *testing.T, original string, edits []protocol.TextEdit) {
	t.Helper()

	// Verify edits don't overlap
	for i := 0; i < len(edits); i++ {
		for j := i + 1; j < len(edits); j++ {
			assert.Falsef(t, buflsp.RangesOverlap(edits[i].Range, edits[j].Range),
				"edits %d and %d overlap: %s and %s",
				i, j,
				buflsp.FormatRange(edits[i].Range),
				buflsp.FormatRange(edits[j].Range))
		}
	}

	// Verify total edit size is reasonable
	totalEditSize := 0
	for _, edit := range edits {
		totalEditSize += len(edit.NewText)
	}

	// For organize imports, we expect the edit size to be related to the import section size
	// not the full file, so this is a sanity check rather than a strict requirement
	assert.LessOrEqualf(t, totalEditSize, len(original),
		"edits size %d should not exceed original file size %d",
		totalEditSize, len(original))

	// Verify no edit is entirely unchanged (replacing text with identical text)
	assertNoRedundantEditsOrganizeImports(t, original, edits)
}

// assertNoRedundantEditsOrganizeImports verifies no edit replaces text with identical text
func assertNoRedundantEditsOrganizeImports(t *testing.T, original string, edits []protocol.TextEdit) {
	t.Helper()

	lines := strings.Split(original, "\n")
	for i, edit := range edits {
		startLine := int(edit.Range.Start.Line)
		startChar := int(edit.Range.Start.Character)
		endLine := int(edit.Range.End.Line)
		endChar := int(edit.Range.End.Character)

		if startLine >= len(lines) || endLine > len(lines) {
			continue
		}

		// Extract the text being replaced
		var replacedText string
		if startLine == endLine {
			// Single line edit
			if startLine < len(lines) && endChar <= len(lines[startLine]) {
				replacedText = lines[startLine][startChar:endChar]
			}
		} else {
			// Multi-line edit
			var parts []string
			for line := startLine; line < endLine; line++ {
				if line >= len(lines) {
					break
				}
				if line == startLine {
					parts = append(parts, lines[line][startChar:])
				} else if line < endLine-1 || (line == endLine-1 && endChar == 0) {
					parts = append(parts, lines[line])
				} else {
					parts = append(parts, lines[line][:endChar])
				}
			}
			replacedText = strings.Join(parts, "\n")
			if endLine > startLine && endChar == 0 {
				replacedText += "\n"
			}
		}

		assert.NotEqualf(t, edit.NewText, replacedText,
			"edit %d replaces text with identical text (not minimal): range %s",
			i, buflsp.FormatRange(edit.Range))
	}
}

// RangesOverlap returns true if two LSP ranges overlap.

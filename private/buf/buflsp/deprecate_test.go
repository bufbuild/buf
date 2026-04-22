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
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/bufbuild/protocompile/experimental/source"
	"github.com/bufbuild/protocompile/experimental/source/length"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func TestCodeAction_Deprecate(t *testing.T) {
	t.Parallel()

	testCodeActionDeprecate(
		t,
		"testdata/deprecate/message_test.proto",
		4, 8, // cursor on "MyMessage"
		"Deprecate message test.deprecate.MyMessage",
		"testdata/deprecate/golden/message.MyMessage.golden.proto",
	)

	testCodeActionDeprecate(
		t,
		"testdata/deprecate/message_test.proto",
		9, 5, // cursor on "MyEnum"
		"Deprecate enum test.deprecate.MyEnum",
		"testdata/deprecate/golden/message.MyEnum.golden.proto",
	)

	testCodeActionDeprecate(
		t,
		"testdata/deprecate/message_test.proto",
		14, 8, // cursor on "MyService"
		"Deprecate service test.deprecate.MyService",
		"testdata/deprecate/golden/message.MyService.golden.proto",
	)

	// No action on whitespace (empty line after package).
	testCodeActionDeprecate(
		t,
		"testdata/deprecate/message_test.proto",
		3, 0,
		"",
		"",
	)

	testCodeActionDeprecate(
		t,
		"testdata/deprecate/nested_test.proto",
		7, 10, // cursor on "Inner"
		"Deprecate message test.deprecate.nested.Outer.Inner",
		"testdata/deprecate/golden/nested.Inner.golden.proto",
	)

	testCodeActionDeprecate(
		t,
		"testdata/deprecate/nested_test.proto",
		10, 9, // cursor on "NestedEnum"
		"Deprecate enum test.deprecate.nested.Outer.Inner.NestedEnum",
		"testdata/deprecate/golden/nested.NestedEnum.golden.proto",
	)

	testCodeActionDeprecate(
		t,
		"testdata/deprecate/nested_test.proto",
		4, 8, // cursor on "Outer"
		"Deprecate message test.deprecate.nested.Outer",
		"testdata/deprecate/golden/nested.Outer.golden.proto",
	)

	testCodeActionDeprecate(
		t,
		"testdata/deprecate/nested_test.proto",
		2, 8, // cursor on package name
		"Deprecate package test.deprecate.nested",
		"testdata/deprecate/golden/nested.package.golden.proto",
	)

	testCodeActionDeprecate(
		t,
		"testdata/deprecate/edge_cases_test.proto",
		5, 8, // cursor on "EmptyMessage"
		"Deprecate message test.deprecate.edge.EmptyMessage",
		"testdata/deprecate/golden/edge.EmptyMessage.golden.proto",
	)

	testCodeActionDeprecate(
		t,
		"testdata/deprecate/edge_cases_test.proto",
		8, 8, // cursor on "EmptyWithSpace"
		"Deprecate message test.deprecate.edge.EmptyWithSpace",
		"testdata/deprecate/golden/edge.EmptyWithSpace.golden.proto",
	)

	// Already deprecated: no new deprecation edit should be produced.
	testCodeActionDeprecateNoEdit(
		t,
		"testdata/deprecate/edge_cases_test.proto",
		11, 8, // cursor on "AlreadyDeprecated"
	)

	// Explicit deprecated=false: do not overwrite with true.
	testCodeActionDeprecateNoEdit(
		t,
		"testdata/deprecate/edge_cases_test.proto",
		16, 8, // cursor on "DeprecatedFalse"
	)

	testCodeActionDeprecate(
		t,
		"testdata/deprecate/edge_cases_test.proto",
		26, 8, // cursor on "OuterWithEmpty"
		"Deprecate message test.deprecate.edge.OuterWithEmpty",
		"testdata/deprecate/golden/edge.OuterWithEmpty.golden.proto",
	)

	// Enum value with preexisting compact options: add deprecated entry.
	testCodeActionDeprecate(
		t,
		"testdata/deprecate/edge_cases_test.proto",
		38, 2, // cursor on "MULTI_LINE_OPTIONS_ONE"
		"Deprecate enum value test.deprecate.edge.MULTI_LINE_OPTIONS_ONE",
		"testdata/deprecate/golden/edge.MULTI_LINE_OPTIONS_ONE.golden.proto",
	)

	// Field with preexisting multi-entry compact options: add deprecated entry.
	testCodeActionDeprecate(
		t,
		"testdata/deprecate/edge_cases_test.proto",
		43, 9, // cursor on "name" field
		"Deprecate field test.deprecate.edge.FieldWithMultiLineOptions.name",
		"testdata/deprecate/golden/edge.FieldWithMultiLineOptions.name.golden.proto",
	)

	// Enum value with single preexisting compact option: add deprecated entry.
	testCodeActionDeprecate(
		t,
		"testdata/deprecate/edge_cases_test.proto",
		52, 2, // cursor on "SINGLE_LINE_OPTIONS_ONE"
		"Deprecate enum value test.deprecate.edge.SINGLE_LINE_OPTIONS_ONE",
		"testdata/deprecate/golden/edge.SINGLE_LINE_OPTIONS_ONE.golden.proto",
	)
}

// testCodeActionDeprecate tests that a deprecation code action at the given position
// produces an edit whose applied result matches the content of goldenPath. Use
// expectedTitle="" and goldenPath="" to test that no action is offered.
//
// Comparing the final applied content keeps these tests robust to printer formatting
// details (edit count, exact ranges, line-based vs character-based edits).
//
// Set UPDATE_GOLDEN=1 when running the test to overwrite the golden file with the
// current output.
func testCodeActionDeprecate(
	t *testing.T,
	filename string,
	cursorLine, cursorChar uint32,
	expectedTitle string,
	goldenPath string,
) {
	t.Helper()
	name := filename
	if expectedTitle != "" {
		name = expectedTitle
	}
	t.Run(name, func(t *testing.T) {
		t.Parallel()
		testProtoPath, err := filepath.Abs(filename)
		require.NoError(t, err)

		clientJSONConn, testURI := setupLSPServer(t, testProtoPath)

		var codeActions []protocol.CodeAction
		_, err = clientJSONConn.Call(t.Context(), protocol.MethodTextDocumentCodeAction, protocol.CodeActionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: testURI,
			},
			Range: protocol.Range{
				Start: protocol.Position{Line: cursorLine, Character: cursorChar},
				End:   protocol.Position{Line: cursorLine, Character: cursorChar},
			},
			Context: protocol.CodeActionContext{
				Only: []protocol.CodeActionKind{
					protocol.RefactorRewrite,
				},
			},
		}, &codeActions)
		require.NoError(t, err)

		// Find the deprecate code action
		var deprecateAction *protocol.CodeAction
		for _, codeAction := range codeActions {
			if codeAction.Kind == protocol.RefactorRewrite {
				deprecateAction = &codeAction
				break
			}
		}

		// If no action expected, verify none found
		if expectedTitle == "" && goldenPath == "" {
			assert.Nil(t, deprecateAction, "expected no deprecate code action")
			return
		}

		require.NotNil(t, deprecateAction, "expected deprecate code action, got none")
		assert.Equal(t, expectedTitle, deprecateAction.Title)
		assert.Equal(t, protocol.RefactorRewrite, deprecateAction.Kind)

		require.NotNil(t, deprecateAction.Edit, "code action should have workspace edit")
		require.NotNil(t, deprecateAction.Edit.Changes, "workspace edit should have changes")

		actualEdits, ok := deprecateAction.Edit.Changes[testURI]
		require.True(t, ok, "workspace edit should have changes for current file")

		// Read original file content
		originalContent, err := os.ReadFile(testProtoPath)
		require.NoError(t, err)

		// Create source file for position conversion
		srcFile := source.NewFile(testProtoPath, string(originalContent))

		// Apply actual edits and compare with the golden file.
		actualResult := applyTextEdits(srcFile, string(originalContent), actualEdits)
		goldenAbs, err := filepath.Abs(goldenPath)
		require.NoError(t, err)
		if os.Getenv("UPDATE_GOLDEN") == "1" {
			require.NoError(t, os.MkdirAll(filepath.Dir(goldenAbs), 0o755))
			require.NoError(t, os.WriteFile(goldenAbs, []byte(actualResult), 0o644))
			return
		}
		expected, err := os.ReadFile(goldenAbs)
		require.NoError(t, err, "golden file not found (set UPDATE_GOLDEN=1 to create)")
		assert.Equal(t, string(expected), actualResult, "applied edits should produce the golden content")
	})
}

// applyTextEdits applies LSP text edits to a string and returns the result.
// Edits are applied in reverse order to preserve positions.
func applyTextEdits(srcFile *source.File, content string, edits []protocol.TextEdit) string {
	// Sort edits by position in reverse order (later positions first)
	sortedEdits := make([]protocol.TextEdit, len(edits))
	copy(sortedEdits, edits)
	sort.Slice(sortedEdits, func(i, j int) bool {
		if sortedEdits[i].Range.Start.Line != sortedEdits[j].Range.Start.Line {
			return sortedEdits[i].Range.Start.Line > sortedEdits[j].Range.Start.Line
		}
		return sortedEdits[i].Range.Start.Character > sortedEdits[j].Range.Start.Character
	})

	// Apply edits in reverse order
	for _, edit := range sortedEdits {
		// Convert LSP positions (0-based) to protocompile positions (1-based)
		startLoc := srcFile.InverseLocation(
			int(edit.Range.Start.Line)+1,
			int(edit.Range.Start.Character)+1,
			length.UTF16,
		)
		endLoc := srcFile.InverseLocation(
			int(edit.Range.End.Line)+1,
			int(edit.Range.End.Character)+1,
			length.UTF16,
		)
		content = content[:startLoc.Offset] + edit.NewText + content[endLoc.Offset:]
	}
	return content
}

// testCodeActionDeprecateNoEdit tests that a deprecation code action is either not offered
// or produces no edits for the target (e.g., already deprecated or deprecated=false).
func testCodeActionDeprecateNoEdit(
	t *testing.T,
	filename string,
	cursorLine, cursorChar uint32,
) {
	t.Helper()
	t.Run(filename, func(t *testing.T) {
		t.Parallel()
		testProtoPath, err := filepath.Abs(filename)
		require.NoError(t, err)

		clientJSONConn, testURI := setupLSPServer(t, testProtoPath)

		var codeActions []protocol.CodeAction
		_, err = clientJSONConn.Call(t.Context(), protocol.MethodTextDocumentCodeAction, protocol.CodeActionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: testURI,
			},
			Range: protocol.Range{
				Start: protocol.Position{Line: cursorLine, Character: cursorChar},
				End:   protocol.Position{Line: cursorLine, Character: cursorChar},
			},
			Context: protocol.CodeActionContext{
				Only: []protocol.CodeActionKind{
					protocol.RefactorRewrite,
				},
			},
		}, &codeActions)
		require.NoError(t, err)

		// Find the deprecate code action
		var deprecateAction *protocol.CodeAction
		for _, codeAction := range codeActions {
			if codeAction.Kind == protocol.RefactorRewrite {
				deprecateAction = &codeAction
				break
			}
		}

		// Either no action, or action with no edits for the target
		if deprecateAction == nil {
			return // No action offered is acceptable
		}

		// If action offered, verify no edits that add deprecation
		if deprecateAction.Edit != nil && deprecateAction.Edit.Changes != nil {
			edits := deprecateAction.Edit.Changes[testURI]
			for _, edit := range edits {
				assert.NotContains(t, edit.NewText, "option deprecated = true;",
					"should not add deprecation to already deprecated or deprecated=false type")
			}
		}
	})
}

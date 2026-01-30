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

	// Test: Deprecate message at cursor on "MyMessage" (line 5)
	testCodeActionDeprecate(
		t,
		"testdata/deprecate/message_test.proto",
		4, 8, // Line 5, on "MyMessage" (0-indexed: line 4)
		"Deprecate message test.deprecate.MyMessage",
		[]protocol.TextEdit{
			// Insert "option deprecated = true;" after opening brace on line 5
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 4, Character: 19},
					End:   protocol.Position{Line: 4, Character: 19},
				},
				NewText: "\n  option deprecated = true;",
			},
		},
	)

	// Test: Deprecate enum at cursor on "MyEnum" (line 10)
	testCodeActionDeprecate(
		t,
		"testdata/deprecate/message_test.proto",
		9, 5, // Line 10, on "MyEnum" (0-indexed: line 9)
		"Deprecate enum test.deprecate.MyEnum",
		[]protocol.TextEdit{
			// Insert "option deprecated = true;" after opening brace on line 10
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 9, Character: 13},
					End:   protocol.Position{Line: 9, Character: 13},
				},
				NewText: "\n  option deprecated = true;",
			},
		},
	)

	// Test: Deprecate service at cursor on "MyService" (line 15)
	// Service deprecation also deprecates the method.
	// Line-based diff produces single edit replacing both changed lines.
	testCodeActionDeprecate(
		t,
		"testdata/deprecate/message_test.proto",
		14, 8, // Line 15, on "MyService" (0-indexed: line 14)
		"Deprecate service test.deprecate.MyService",
		[]protocol.TextEdit{
			// Lines 15-16 both change, so diff replaces them together
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 14, Character: 0},
					End:   protocol.Position{Line: 16, Character: 0},
				},
				NewText: "service MyService {\n  option deprecated = true;\n  rpc GetMessage(MyMessage) returns (MyMessage) {\n    option deprecated = true;\n  }\n",
			},
		},
	)

	// Test: No action on whitespace (empty line after package)
	testCodeActionDeprecate(
		t,
		"testdata/deprecate/message_test.proto",
		3, 0, // Line 4, empty line (0-indexed: line 3)
		"", // No action expected
		nil,
	)

	// Test: Deprecate nested message "Inner" - verifies 4-space indentation
	testCodeActionDeprecate(
		t,
		"testdata/deprecate/nested_test.proto",
		7, 10, // Line 8, on "Inner" (0-indexed: line 7)
		"Deprecate message test.deprecate.nested.Outer.Inner",
		[]protocol.TextEdit{
			// Inner message - Line 8: "  message Inner {" - insert at 17
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 7, Character: 17},
					End:   protocol.Position{Line: 7, Character: 17},
				},
				NewText: "\n    option deprecated = true;",
			},
			// NestedEnum - Line 11: "    enum NestedEnum {" - insert at 21
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 10, Character: 21},
					End:   protocol.Position{Line: 10, Character: 21},
				},
				NewText: "\n      option deprecated = true;",
			},
		},
	)

	// Test: Deprecate doubly-nested enum "NestedEnum" - verifies 6-space indentation
	testCodeActionDeprecate(
		t,
		"testdata/deprecate/nested_test.proto",
		10, 9, // Line 11, on "NestedEnum" (0-indexed: line 10)
		"Deprecate enum test.deprecate.nested.Outer.Inner.NestedEnum",
		[]protocol.TextEdit{
			// NestedEnum - Line 11: "    enum NestedEnum {" - insert at 21
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 10, Character: 21},
					End:   protocol.Position{Line: 10, Character: 21},
				},
				NewText: "\n      option deprecated = true;",
			},
		},
	)

	// Test: Deprecate parent "Outer" - deprecates all nested types but NOT fields/enum values
	testCodeActionDeprecate(
		t,
		"testdata/deprecate/nested_test.proto",
		4, 8, // Line 5, on "Outer" (0-indexed: line 4)
		"Deprecate message test.deprecate.nested.Outer",
		[]protocol.TextEdit{
			// Outer message - Line 5: "message Outer {" - insert at 15
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 4, Character: 15},
					End:   protocol.Position{Line: 4, Character: 15},
				},
				NewText: "\n  option deprecated = true;",
			},
			// Inner message - Line 8: "  message Inner {" - insert at 17
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 7, Character: 17},
					End:   protocol.Position{Line: 7, Character: 17},
				},
				NewText: "\n    option deprecated = true;",
			},
			// NestedEnum - Line 11: "    enum NestedEnum {" - insert at 21
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 10, Character: 21},
					End:   protocol.Position{Line: 10, Character: 21},
				},
				NewText: "\n      option deprecated = true;",
			},
			// OuterEnum - Line 16: "  enum OuterEnum {" - insert at 18
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 15, Character: 18},
					End:   protocol.Position{Line: 15, Character: 18},
				},
				NewText: "\n    option deprecated = true;",
			},
		},
	)

	// Test: Deprecate package - deprecates file + all types
	testCodeActionDeprecate(
		t,
		"testdata/deprecate/nested_test.proto",
		2, 8, // Line 3, on package name (0-indexed: line 2)
		"Deprecate package test.deprecate.nested",
		[]protocol.TextEdit{
			// File-level deprecated option after package declaration.
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 2, Character: 30},
					End:   protocol.Position{Line: 2, Character: 30},
				},
				NewText: "\noption deprecated = true;",
			},
			// Outer message - Line 5: "message Outer {" - insert at 15
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 4, Character: 15},
					End:   protocol.Position{Line: 4, Character: 15},
				},
				NewText: "\n  option deprecated = true;",
			},
			// Inner message - Line 8: "  message Inner {" - insert at 17
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 7, Character: 17},
					End:   protocol.Position{Line: 7, Character: 17},
				},
				NewText: "\n    option deprecated = true;",
			},
			// NestedEnum - Line 11: "    enum NestedEnum {" - insert at 21
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 10, Character: 21},
					End:   protocol.Position{Line: 10, Character: 21},
				},
				NewText: "\n      option deprecated = true;",
			},
			// OuterEnum - Line 16: "  enum OuterEnum {" - insert at 18
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 15, Character: 18},
					End:   protocol.Position{Line: 15, Character: 18},
				},
				NewText: "\n    option deprecated = true;",
			},
		},
	)

	// Test: Empty body message `{}` - must insert with proper newlines
	testCodeActionDeprecate(
		t,
		"testdata/deprecate/edge_cases_test.proto",
		5, 8, // Line 6, on "EmptyMessage" (0-indexed: line 5)
		"Deprecate message test.deprecate.edge.EmptyMessage",
		[]protocol.TextEdit{
			// Insert after `{`, add newlines to format properly
			// Result: message EmptyMessage {\n  option deprecated = true;\n}
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 5, Character: 22},
					End:   protocol.Position{Line: 5, Character: 22},
				},
				NewText: "\n  option deprecated = true;\n",
			},
		},
	)

	// Test: Empty body with space `{ }`.
	testCodeActionDeprecate(
		t,
		"testdata/deprecate/edge_cases_test.proto",
		8, 8, // Line 9, on "EmptyWithSpace" (0-indexed: line 8)
		"Deprecate message test.deprecate.edge.EmptyWithSpace",
		[]protocol.TextEdit{
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 8, Character: 24},
					End:   protocol.Position{Line: 8, Character: 26},
				},
				NewText: "\n  option deprecated = true;\n}",
			},
		},
	)

	// Test: Already deprecated - no edit for this message
	testCodeActionDeprecateNoEdit(
		t,
		"testdata/deprecate/edge_cases_test.proto",
		11, 8, // Line 12, on "AlreadyDeprecated" (0-indexed: line 11)
	)

	// Test: Deprecated = false - no edit (don't conflict with explicit false)
	testCodeActionDeprecateNoEdit(
		t,
		"testdata/deprecate/edge_cases_test.proto",
		16, 8, // Line 17, on "DeprecatedFalse" (0-indexed: line 16)
	)

	// Test: Nested empty body message.
	// Line-based diff replaces both lines 27-28 together.
	testCodeActionDeprecate(
		t,
		"testdata/deprecate/edge_cases_test.proto",
		26, 8, // Line 27, on "OuterWithEmpty" (0-indexed: line 26)
		"Deprecate message test.deprecate.edge.OuterWithEmpty",
		[]protocol.TextEdit{
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 26, Character: 0},
					End:   protocol.Position{Line: 28, Character: 0},
				},
				NewText: "message OuterWithEmpty {\n  option deprecated = true;\n  message NestedEmpty {\n    option deprecated = true;\n  }\n",
			},
		},
	)

	// Test: Multi-line options on enum value - appends on same line.
	testCodeActionDeprecate(
		t,
		"testdata/deprecate/edge_cases_test.proto",
		38, 2, // Line 39, on "MULTI_LINE_OPTIONS_ONE" (0-indexed: line 38)
		"Deprecate enum value test.deprecate.edge.MULTI_LINE_OPTIONS_ONE",
		[]protocol.TextEdit{
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 39, Character: 23},
					End:   protocol.Position{Line: 39, Character: 23},
				},
				NewText: ", deprecated = true",
			},
		},
	)

	// Test: Multi-line options on field - appends on same line.
	testCodeActionDeprecate(
		t,
		"testdata/deprecate/edge_cases_test.proto",
		45, 9, // Line 46, on "name" field (0-indexed: line 45)
		"Deprecate field test.deprecate.edge.FieldWithMultiLineOptions.name",
		[]protocol.TextEdit{
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 47, Character: 23},
					End:   protocol.Position{Line: 47, Character: 23},
				},
				NewText: ", deprecated = true",
			},
		},
	)

	// Test: Single-line options on enum value
	// Should insert ", deprecated = true" before closing bracket
	testCodeActionDeprecate(
		t,
		"testdata/deprecate/edge_cases_test.proto",
		54, 2, // Line 55, on "SINGLE_LINE_OPTIONS_ONE" (0-indexed: line 54)
		"Deprecate enum value test.deprecate.edge.SINGLE_LINE_OPTIONS_ONE",
		[]protocol.TextEdit{
			// Insert before "]" on line 55
			// Line 55: "  SINGLE_LINE_OPTIONS_ONE = 1 [debug_redact = true];"
			// The "]" is at char 50
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 54, Character: 50},
					End:   protocol.Position{Line: 54, Character: 50},
				},
				NewText: ", deprecated = true",
			},
		},
	)
}

// testCodeActionDeprecate tests that a deprecation code action at the given position
// produces the expected edits. Use expectedTitle="" and expectedEdits=nil to test
// that no action is offered.
//
// Instead of checking exact edit positions, this verifies that applying the actual
// edits produces the same result as applying the expected edits. This makes the
// tests robust to different edit formats (character-based vs line-based).
func testCodeActionDeprecate(
	t *testing.T,
	filename string,
	cursorLine, cursorChar uint32,
	expectedTitle string,
	expectedEdits []protocol.TextEdit,
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
		if expectedTitle == "" && expectedEdits == nil {
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
		require.Len(t, actualEdits, len(expectedEdits), "edit count mismatch")

		// Read original file content
		originalContent, err := os.ReadFile(testProtoPath)
		require.NoError(t, err)

		// Create source file for position conversion
		srcFile := source.NewFile(testProtoPath, string(originalContent))

		// Apply expected edits to get expected result
		expectedResult := applyTextEdits(srcFile, string(originalContent), expectedEdits)

		// Apply actual edits to get actual result
		actualResult := applyTextEdits(srcFile, string(originalContent), actualEdits)

		// Compare results (both approaches should produce the same final content)
		assert.Equal(t, expectedResult, actualResult, "applied edits should produce the same result")
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

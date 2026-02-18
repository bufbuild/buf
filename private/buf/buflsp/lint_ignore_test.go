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
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func TestCodeAction_LintIgnore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		filename          string
		cursorLine        uint32
		cursorChar        uint32
		expectedRuleIDs   []string // If multiple, tests multiple actions on same line
		expectedEdits     []protocol.TextEdit
		expectNoAction    bool
		expectIsPreferred bool
	}{
		{
			name:              "basic_field_lint_error",
			filename:          "testdata/lint_ignore/field_test.proto",
			cursorLine:        5,
			cursorChar:        10,
			expectedRuleIDs:   []string{"FIELD_LOWER_SNAKE_CASE"},
			expectIsPreferred: true,
			expectedEdits: []protocol.TextEdit{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 5, Character: 0},
						End:   protocol.Position{Line: 5, Character: 0},
					},
					NewText: "  // buf:lint:ignore FIELD_LOWER_SNAKE_CASE\n",
				},
			},
		},
		{
			name:              "nested_message_with_indentation",
			filename:          "testdata/lint_ignore/nested_test.proto",
			cursorLine:        6,
			cursorChar:        10,
			expectedRuleIDs:   []string{"FIELD_LOWER_SNAKE_CASE"},
			expectIsPreferred: true,
			expectedEdits: []protocol.TextEdit{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 6, Character: 0},
						End:   protocol.Position{Line: 6, Character: 0},
					},
					NewText: "    // buf:lint:ignore FIELD_LOWER_SNAKE_CASE\n",
				},
			},
		},
		{
			name:            "package_line_diagnostic",
			filename:        "testdata/lint_ignore/syntax_test.proto",
			cursorLine:      2,
			cursorChar:      0,
			expectedRuleIDs: []string{"PACKAGE_DIRECTORY_MATCH", "PACKAGE_VERSION_SUFFIX"},
			// Multiple actions, so IsPreferred should not be set
		},
		{
			name:           "no_action_on_line_without_lints",
			filename:       "testdata/lint_ignore/field_test.proto",
			cursorLine:     1,
			cursorChar:     0,
			expectNoAction: true,
		},
		{
			name:            "multiple_lints_on_same_line",
			filename:        "testdata/lint_ignore/multiple_test.proto",
			cursorLine:      2,
			cursorChar:      0,
			expectedRuleIDs: []string{"PACKAGE_DIRECTORY_MATCH", "PACKAGE_VERSION_SUFFIX"},
		},
		{
			name:           "file_wide_lints_are_skipped",
			filename:       "testdata/lint_ignore/filewide_test.proto",
			cursorLine:     0,
			cursorChar:     0,
			expectNoAction: true,
		},
		{
			name:           "already_has_ignore_for_same_rule",
			filename:       "testdata/lint_ignore/already_ignored_test.proto",
			cursorLine:     6,
			cursorChar:     10,
			expectNoAction: true,
		},
		{
			name:              "already_has_ignore_for_different_rule",
			filename:          "testdata/lint_ignore/different_rule_test.proto",
			cursorLine:        6,
			cursorChar:        10,
			expectedRuleIDs:   []string{"FIELD_LOWER_SNAKE_CASE"},
			expectIsPreferred: true,
			expectedEdits: []protocol.TextEdit{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 6, Character: 0},
						End:   protocol.Position{Line: 6, Character: 0},
					},
					NewText: "  // buf:lint:ignore FIELD_LOWER_SNAKE_CASE\n",
				},
			},
		},
		{
			name:              "tab_indentation_is_preserved",
			filename:          "testdata/lint_ignore/tabs_test.proto",
			cursorLine:        5,
			cursorChar:        10,
			expectedRuleIDs:   []string{"FIELD_LOWER_SNAKE_CASE"},
			expectIsPreferred: true,
			expectedEdits: []protocol.TextEdit{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 5, Character: 0},
						End:   protocol.Position{Line: 5, Character: 0},
					},
					NewText: "\t// buf:lint:ignore FIELD_LOWER_SNAKE_CASE\n",
				},
			},
		},
		{
			name:            "trailing_comment_does_not_suppress_package_lint",
			filename:        "testdata/lint_ignore/trailing_comment_test.proto",
			cursorLine:      2,
			cursorChar:      10,
			expectedRuleIDs: []string{"PACKAGE_DIRECTORY_MATCH", "PACKAGE_VERSION_SUFFIX"},
			// Multiple actions, so IsPreferred should not be set
		},
		{
			name:              "trailing_comment_does_not_suppress_field_lint",
			filename:          "testdata/lint_ignore/trailing_comment_test.proto",
			cursorLine:        5,
			cursorChar:        10,
			expectedRuleIDs:   []string{"FIELD_LOWER_SNAKE_CASE"},
			expectIsPreferred: true,
			expectedEdits: []protocol.TextEdit{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 5, Character: 0},
						End:   protocol.Position{Line: 5, Character: 0},
					},
					NewText: "  // buf:lint:ignore FIELD_LOWER_SNAKE_CASE\n",
				},
			},
		},
		{
			name:              "trailing_comment_on_previous_line_does_not_suppress_field_lint",
			filename:          "testdata/lint_ignore/trailing_comment_test.proto",
			cursorLine:        6,
			cursorChar:        10,
			expectedRuleIDs:   []string{"FIELD_LOWER_SNAKE_CASE"},
			expectIsPreferred: true,
			expectedEdits: []protocol.TextEdit{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 6, Character: 0},
						End:   protocol.Position{Line: 6, Character: 0},
					},
					NewText: "  // buf:lint:ignore FIELD_LOWER_SNAKE_CASE\n",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				testProtoPath, err := filepath.Abs(tt.filename)
				require.NoError(t, err)

				clientJSONConn, testURI, capture := setupLSPServerWithDiagnostics(t, testProtoPath)

				// Wait for lint diagnostics if we expect actions
				if !tt.expectNoAction && len(tt.expectedRuleIDs) > 0 {
					diagnostics := capture.wait(t, testURI, 10*time.Second, func(p *protocol.PublishDiagnosticsParams) bool {
						count := 0
						for _, d := range p.Diagnostics {
							if d.Source == "buf lint" {
								if code, ok := d.Code.(string); ok {
									for _, ruleID := range tt.expectedRuleIDs {
										if code == ruleID {
											count++
											break
										}
									}
								}
							}
						}
						return count >= len(tt.expectedRuleIDs)
					})
					require.NotNil(t, diagnostics, "expected lint diagnostics to be published")
				}

				// Request code actions at the specified position
				var codeActions []protocol.CodeAction
				_, err = clientJSONConn.Call(t.Context(), protocol.MethodTextDocumentCodeAction, protocol.CodeActionParams{
					TextDocument: protocol.TextDocumentIdentifier{
						URI: testURI,
					},
					Range: protocol.Range{
						Start: protocol.Position{Line: tt.cursorLine, Character: tt.cursorChar},
						End:   protocol.Position{Line: tt.cursorLine, Character: tt.cursorChar},
					},
					Context: protocol.CodeActionContext{
						Only: []protocol.CodeActionKind{
							protocol.QuickFix,
						},
					},
				}, &codeActions)
				require.NoError(t, err)

				// Filter to lint ignore actions only
				var lintIgnoreActions []protocol.CodeAction
				for _, action := range codeActions {
					if action.Kind == protocol.QuickFix && strings.Contains(action.Title, "buf:lint:ignore") {
						lintIgnoreActions = append(lintIgnoreActions, action)
					}
				}

				if tt.expectNoAction {
					assert.Empty(t, lintIgnoreActions, "expected no lint ignore actions")
					return
				}

				// Verify we have the expected number of actions
				require.Len(t, lintIgnoreActions, len(tt.expectedRuleIDs),
					"expected %d lint ignore action(s)", len(tt.expectedRuleIDs))

				// Verify each expected rule has an action
				for _, ruleID := range tt.expectedRuleIDs {
					expectedTitle := "Suppress " + ruleID + " with buf:lint:ignore"
					var foundAction *protocol.CodeAction
					for i := range lintIgnoreActions {
						if lintIgnoreActions[i].Title == expectedTitle {
							foundAction = &lintIgnoreActions[i]
							break
						}
					}
					require.NotNil(t, foundAction, "expected to find action for rule %s", ruleID)
					assert.Equal(t, protocol.QuickFix, foundAction.Kind)

					// Check IsPreferred flag
					if tt.expectIsPreferred {
						assert.True(t, foundAction.IsPreferred, "expected action to be marked as preferred")
					}
				}

				// Verify edits if provided (only for single-action tests)
				if len(tt.expectedEdits) > 0 && len(tt.expectedRuleIDs) == 1 {
					action := lintIgnoreActions[0]
					require.NotNil(t, action.Edit)
					require.NotNil(t, action.Edit.Changes)

					changes, ok := action.Edit.Changes[testURI]
					require.True(t, ok, "expected changes for test file")
					require.Equal(t, len(tt.expectedEdits), len(changes),
						"expected %d edit(s), got %d", len(tt.expectedEdits), len(changes))

					for i, expectedEdit := range tt.expectedEdits {
						assert.Equal(t, expectedEdit.Range, changes[i].Range, "edit %d: range mismatch", i)
						assert.Equal(t, expectedEdit.NewText, changes[i].NewText, "edit %d: new text mismatch", i)
					}
				}
			})
		})
	}
}

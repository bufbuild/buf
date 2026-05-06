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

package buflsp

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func TestGetBufYAMLCompletionItems(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		text             string
		pos              protocol.Position
		wantLabels       []string
		wantAbsentLabels []string
		wantKind         protocol.CompletionItemKind
		wantNilResult    bool
	}{
		// ── Top-level key completions ──────────────────────────────────────
		{
			name:       "top_level_empty",
			text:       "\n",
			pos:        protocol.Position{Line: 0, Character: 0},
			wantLabels: []string{"version", "name", "modules", "deps", "lint", "breaking", "plugins", "policies"},
			wantKind:   protocol.CompletionItemKindField,
		},
		{
			name:       "top_level_partial",
			text:       "br\n",
			pos:        protocol.Position{Line: 0, Character: 2},
			wantLabels: []string{"breaking"},
			wantKind:   protocol.CompletionItemKindField,
		},
		{
			name:             "top_level_after_existing",
			text:             "version: v2\n",
			pos:              protocol.Position{Line: 1, Character: 0},
			wantLabels:       []string{"modules", "lint", "breaking"},
			wantAbsentLabels: []string{"version"},
			wantKind:         protocol.CompletionItemKindField,
		},

		// ── lint key completions ───────────────────────────────────────────
		{
			name:       "lint_keys",
			text:       "lint:\n  \n",
			pos:        protocol.Position{Line: 1, Character: 2},
			wantLabels: []string{"use", "except", "ignore", "ignore_only", "disallow_comment_ignores", "disable_builtin"},
			wantKind:   protocol.CompletionItemKindField,
		},
		{
			name:       "lint_keys_partial",
			text:       "lint:\n  ig\n",
			pos:        protocol.Position{Line: 1, Character: 4},
			wantLabels: []string{"ignore", "ignore_only"},
			wantKind:   protocol.CompletionItemKindField,
		},

		// ── breaking key completions ───────────────────────────────────────
		{
			name:       "breaking_keys",
			text:       "breaking:\n  \n",
			pos:        protocol.Position{Line: 1, Character: 2},
			wantLabels: []string{"use", "except", "ignore", "ignore_only", "ignore_unstable_packages", "disable_builtin"},
			wantKind:   protocol.CompletionItemKindField,
		},

		// ── modules item key completions ───────────────────────────────────
		{
			name: "modules_item_keys",
			text: "modules:\n  - \n",
			pos:  protocol.Position{Line: 1, Character: 4},
			wantLabels: []string{
				"path", "name", "includes", "excludes", "lint", "breaking",
			},
			wantKind: protocol.CompletionItemKindField,
		},
		{
			name: "modules_item_continuation",
			text: "modules:\n  - path: .\n    \n",
			pos:  protocol.Position{Line: 2, Character: 4},
			wantLabels: []string{
				"name", "includes", "excludes", "lint", "breaking",
			},
			wantAbsentLabels: []string{"path"},
			wantKind:         protocol.CompletionItemKindField,
		},

		// ── module-level lint keys ─────────────────────────────────────────
		{
			name:       "module_lint_keys",
			text:       "modules:\n  - path: .\n    lint:\n      \n",
			pos:        protocol.Position{Line: 3, Character: 6},
			wantLabels: []string{"use", "except", "ignore", "ignore_only", "disable_builtin"},
			wantKind:   protocol.CompletionItemKindField,
		},

		// ── plugins item key completions ───────────────────────────────────
		{
			name:       "plugins_item_keys",
			text:       "plugins:\n  - \n",
			pos:        protocol.Position{Line: 1, Character: 4},
			wantLabels: []string{"plugin", "options"},
			wantKind:   protocol.CompletionItemKindField,
		},

		// ── policies item key completions ──────────────────────────────────
		{
			name:       "policies_item_keys",
			text:       "policies:\n  - \n",
			pos:        protocol.Position{Line: 1, Character: 4},
			wantLabels: []string{"policy", "ignore", "ignore_only"},
			wantKind:   protocol.CompletionItemKindField,
		},

		// ── Value completions ──────────────────────────────────────────────
		{
			name:       "version_value",
			text:       "version: \n",
			pos:        protocol.Position{Line: 0, Character: 9},
			wantLabels: []string{"v2", "v1", "v1beta1"},
			wantKind:   protocol.CompletionItemKindValue,
		},
		{
			name:       "disable_builtin_value",
			text:       "lint:\n  disable_builtin: \n",
			pos:        protocol.Position{Line: 1, Character: 18},
			wantLabels: []string{"true", "false"},
			wantKind:   protocol.CompletionItemKindValue,
		},
		{
			name:       "ignore_unstable_packages_value",
			text:       "breaking:\n  ignore_unstable_packages: \n",
			pos:        protocol.Position{Line: 1, Character: 27},
			wantLabels: []string{"true", "false"},
			wantKind:   protocol.CompletionItemKindValue,
		},

		// ── Sequence value completions (lint.use / lint.except) ────────────
		{
			name:       "lint_use_values",
			text:       "lint:\n  use:\n    - \n",
			pos:        protocol.Position{Line: 2, Character: 6},
			wantLabels: []string{"MINIMAL", "BASIC", "STANDARD", "COMMENTS", "UNARY_RPC"},
			wantKind:   protocol.CompletionItemKindValue,
		},
		{
			name:       "lint_use_partial",
			text:       "lint:\n  use:\n    - STAN\n",
			pos:        protocol.Position{Line: 2, Character: 10},
			wantLabels: []string{"STANDARD"},
			wantKind:   protocol.CompletionItemKindValue,
		},
		{
			name:       "lint_except_values",
			text:       "lint:\n  except:\n    - \n",
			pos:        protocol.Position{Line: 2, Character: 6},
			wantLabels: []string{"STANDARD", "FIELD_LOWER_SNAKE_CASE"},
			wantKind:   protocol.CompletionItemKindValue,
		},

		// ── Sequence value completions (breaking.use / breaking.except) ────
		{
			name:       "breaking_use_values",
			text:       "breaking:\n  use:\n    - \n",
			pos:        protocol.Position{Line: 2, Character: 6},
			wantLabels: []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"},
			wantKind:   protocol.CompletionItemKindValue,
		},
		{
			name:       "breaking_except_values",
			text:       "breaking:\n  except:\n    - \n",
			pos:        protocol.Position{Line: 2, Character: 6},
			wantLabels: []string{"FILE", "FIELD_NO_DELETE"},
			wantKind:   protocol.CompletionItemKindValue,
		},

		// ── ignore_only key completions ────────────────────────────────────
		{
			name:       "lint_ignore_only_keys",
			text:       "lint:\n  ignore_only:\n    \n",
			pos:        protocol.Position{Line: 2, Character: 4},
			wantLabels: []string{"STANDARD", "FIELD_LOWER_SNAKE_CASE", "MINIMAL"},
			wantKind:   protocol.CompletionItemKindField,
		},
		{
			name:       "breaking_ignore_only_keys",
			text:       "breaking:\n  ignore_only:\n    \n",
			pos:        protocol.Position{Line: 2, Character: 4},
			wantLabels: []string{"FILE", "FIELD_NO_DELETE"},
			wantKind:   protocol.CompletionItemKindField,
		},
		{
			name:       "module_lint_ignore_only_keys",
			text:       "modules:\n  - path: .\n    lint:\n      ignore_only:\n        \n",
			pos:        protocol.Position{Line: 4, Character: 8},
			wantLabels: []string{"STANDARD", "FIELD_LOWER_SNAKE_CASE"},
			wantKind:   protocol.CompletionItemKindField,
		},

		// ── Sequence after existing items ─────────────────────────────────
		{
			name:       "lint_use_after_existing_item",
			text:       "lint:\n  use:\n    - STANDARD\n    - \n",
			pos:        protocol.Position{Line: 3, Character: 6},
			wantLabels: []string{"MINIMAL", "BASIC", "STANDARD", "COMMENTS"},
			wantKind:   protocol.CompletionItemKindValue,
		},
		{
			name:       "breaking_except_after_existing_item",
			text:       "breaking:\n  except:\n    - FILE\n    - \n",
			pos:        protocol.Position{Line: 3, Character: 6},
			wantLabels: []string{"FILE", "PACKAGE", "WIRE_JSON", "FIELD_NO_DELETE"},
			wantKind:   protocol.CompletionItemKindValue,
		},

		// ── Module-level lint and breaking sequence values ─────────────────
		{
			name:       "module_breaking_use_values",
			text:       "modules:\n  - path: .\n    breaking:\n      use:\n        - \n",
			pos:        protocol.Position{Line: 4, Character: 10},
			wantLabels: []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"},
			wantKind:   protocol.CompletionItemKindValue,
		},
		{
			name:       "module_lint_use_values",
			text:       "modules:\n  - path: .\n    lint:\n      use:\n        - \n",
			pos:        protocol.Position{Line: 4, Character: 10},
			wantLabels: []string{"MINIMAL", "BASIC", "STANDARD"},
			wantKind:   protocol.CompletionItemKindValue,
		},

		// ── Sibling key (except alongside use) ────────────────────────────
		{
			name:       "lint_except_after_use_section",
			text:       "lint:\n  use:\n    - STANDARD\n  except:\n    - \n",
			pos:        protocol.Position{Line: 4, Character: 6},
			wantLabels: []string{"STANDARD", "FIELD_LOWER_SNAKE_CASE"},
			wantKind:   protocol.CompletionItemKindValue,
		},

		// ── Blank lines between parent and child ──────────────────────────
		{
			name:       "blank_line_between_lint_and_use",
			text:       "lint:\n\n  use:\n    - \n",
			pos:        protocol.Position{Line: 3, Character: 6},
			wantLabels: []string{"MINIMAL", "BASIC", "STANDARD"},
			wantKind:   protocol.CompletionItemKindValue,
		},

		// ── Partial key in lint block ──────────────────────────────────────
		{
			name:       "lint_partial_rpc_key",
			text:       "lint:\n  rpc_a\n",
			pos:        protocol.Position{Line: 1, Character: 7},
			wantLabels: []string{"rpc_allow_same_request_response", "rpc_allow_google_protobuf_empty_requests", "rpc_allow_google_protobuf_empty_responses"},
			wantKind:   protocol.CompletionItemKindField,
		},

		// ── Second module item keys ────────────────────────────────────────
		{
			name:       "second_module_item_keys",
			text:       "modules:\n  - path: a\n  - \n",
			pos:        protocol.Position{Line: 2, Character: 4},
			wantLabels: []string{"path", "name", "includes", "excludes", "lint", "breaking"},
			wantKind:   protocol.CompletionItemKindField,
		},
		{
			// Completing inside an existing module item should filter keys already present.
			name:             "module_item_existing_key_filtered",
			text:             "modules:\n  - path: .\n    \n",
			pos:              protocol.Position{Line: 2, Character: 4},
			wantLabels:       []string{"name", "lint", "breaking"},
			wantAbsentLabels: []string{"path"},
			wantKind:         protocol.CompletionItemKindField,
		},

		// ── policies[].ignore sequence (no completions — file paths) ───────
		{
			name:          "policy_ignore_no_completions",
			text:          "policies:\n  - policy: buf.build/foo/bar\n    ignore:\n      - \n",
			pos:           protocol.Position{Line: 3, Character: 8},
			wantNilResult: true,
		},

		// ── Bare-parent-key heuristic ─────────────────────────────────────
		{
			// Cursor at col 0 after bare "breaking:" should offer breaking's
			// sub-keys (not top-level keys like "deps:").
			name:             "bare_parent_breaking",
			text:             "breaking:\n",
			pos:              protocol.Position{Line: 1, Character: 0},
			wantLabels:       []string{"use", "except", "ignore", "ignore_only", "ignore_unstable_packages", "disable_builtin"},
			wantAbsentLabels: []string{"deps", "lint", "modules", "version"},
			wantKind:         protocol.CompletionItemKindField,
		},
		{
			name:             "bare_parent_lint",
			text:             "lint:\n",
			pos:              protocol.Position{Line: 1, Character: 0},
			wantLabels:       []string{"use", "except", "ignore", "ignore_only", "disallow_comment_ignores", "disable_builtin"},
			wantAbsentLabels: []string{"deps", "breaking", "modules", "version"},
			wantKind:         protocol.CompletionItemKindField,
		},
		{
			// Cursor at col 0 after breaking's children are defined falls back
			// to top-level completions (the children create indented content
			// that causes bufYAMLBareParentKey to return "").
			name:             "after_breaking_with_children",
			text:             "breaking:\n  use:\n    - FILE\n",
			pos:              protocol.Position{Line: 3, Character: 0},
			wantLabels:       []string{"lint", "modules", "deps"},
			wantAbsentLabels: []string{"breaking"},
			wantKind:         protocol.CompletionItemKindField,
		},
		{
			// Sequence-valued keys (plugins, modules, deps) must NOT trigger the
			// bare-parent heuristic — their items start with "- " list markers.
			name:             "bare_plugins_no_child_keys",
			text:             "plugins:\n",
			pos:              protocol.Position{Line: 1, Character: 0},
			wantLabels:       []string{"version", "lint", "breaking"},
			wantAbsentLabels: []string{"plugin", "options"},
			wantKind:         protocol.CompletionItemKindField,
		},

		// ── Existing-key filtering ────────────────────────────────────────
		{
			// Uses text where lint has children, so the cursor is clearly past
			// the lint block and at top level.
			name:             "top_level_existing_keys_filtered",
			text:             "version: v2\nlint:\n  use:\n    - STANDARD\n\n",
			pos:              protocol.Position{Line: 4, Character: 0},
			wantLabels:       []string{"breaking", "modules"},
			wantAbsentLabels: []string{"version", "lint"},
			wantKind:         protocol.CompletionItemKindField,
		},
		{
			name:             "lint_section_existing_key_filtered",
			text:             "lint:\n  use:\n    - STANDARD\n  \n",
			pos:              protocol.Position{Line: 3, Character: 2},
			wantLabels:       []string{"except", "ignore"},
			wantAbsentLabels: []string{"use"},
			wantKind:         protocol.CompletionItemKindField,
		},
		{
			name:             "ignore_only_existing_rule_filtered",
			text:             "lint:\n  ignore_only:\n    STANDARD:\n    \n",
			pos:              protocol.Position{Line: 3, Character: 4},
			wantAbsentLabels: []string{"STANDARD"},
			wantKind:         protocol.CompletionItemKindField,
		},

		// ── No completions ─────────────────────────────────────────────────
		{
			name:          "list_marker_only",
			text:          "modules:\n  -\n",
			pos:           protocol.Position{Line: 1, Character: 3},
			wantNilResult: true,
		},
		{
			name:          "out_of_bounds",
			text:          "version: v2\n",
			pos:           protocol.Position{Line: 99, Character: 0},
			wantNilResult: true,
		},
		{
			name:          "name_no_value_completions",
			text:          "name: \n",
			pos:           protocol.Position{Line: 0, Character: 6},
			wantNilResult: true,
		},
		{
			// "enum_zero_value_suffix" is 22 chars; "  " + key + ": " = 26 chars total.
			// Cursor past the ": " is in value position; no predefined values exist for this key.
			name:          "enum_zero_value_suffix_no_completions",
			text:          "lint:\n  enum_zero_value_suffix: \n",
			pos:           protocol.Position{Line: 1, Character: 26},
			wantNilResult: true,
		},

		// ── CRLF line endings ─────────────────────────────────────────────
		{
			name:             "crlf_top_level_existing_key_filtered",
			text:             "version: v2\r\n",
			pos:              protocol.Position{Line: 1, Character: 0},
			wantLabels:       []string{"modules", "lint", "breaking"},
			wantAbsentLabels: []string{"version"},
			wantKind:         protocol.CompletionItemKindField,
		},
		{
			// CRLF bare parent: "breaking:\r\n" should trigger bare-parent heuristic.
			name:             "crlf_bare_parent_breaking",
			text:             "breaking:\r\n",
			pos:              protocol.Position{Line: 1, Character: 0},
			wantLabels:       []string{"use", "except", "ignore_unstable_packages"},
			wantAbsentLabels: []string{"version", "modules"},
			wantKind:         protocol.CompletionItemKindField,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			items := getBufYAMLCompletionItems(parseYAMLDoc(testCase.text), testCase.text, testCase.pos)

			if testCase.wantNilResult {
				assert.Nil(t, items)
				return
			}

			require.NotNil(t, items)
			labels := completionLabels(items)
			for _, wantLabel := range testCase.wantLabels {
				assert.True(t, slices.Contains(labels, wantLabel),
					"expected label %q in completion items %v", wantLabel, labels)
			}
			for _, absentLabel := range testCase.wantAbsentLabels {
				assert.False(t, slices.Contains(labels, absentLabel),
					"unexpected label %q in completion items %v", absentLabel, labels)
			}
			for _, item := range items {
				assert.Equal(t, testCase.wantKind, item.Kind,
					"item %q has unexpected kind", item.Label)
			}
		})
	}
}

func TestGetBufYAMLCompletionItemsTextEdit(t *testing.T) {
	t.Parallel()

	t.Run("key_textEdit_includes_colon_space", func(t *testing.T) {
		t.Parallel()
		items := getBufYAMLCompletionItems(parseYAMLDoc("lint:\n  \n"), "lint:\n  \n", protocol.Position{Line: 1, Character: 2})
		require.NotNil(t, items)
		for _, item := range items {
			require.NotNil(t, item.TextEdit, "item %q has no TextEdit", item.Label)
			assert.Equal(t, item.Label+": ", item.TextEdit.NewText,
				"item %q TextEdit.NewText should be label + \": \"", item.Label)
		}
	})

	t.Run("sequence_value_textEdit_replaces_partial_token", func(t *testing.T) {
		t.Parallel()
		// "    - STAN" — cursor at character 10, token "STAN" starts at character 6.
		text := "lint:\n  use:\n    - STAN\n"
		items := getBufYAMLCompletionItems(
			parseYAMLDoc(text),
			text,
			protocol.Position{Line: 2, Character: 10},
		)
		require.NotNil(t, items)
		for _, item := range items {
			require.NotNil(t, item.TextEdit, "item %q has no TextEdit", item.Label)
			assert.Equal(t, uint32(6), item.TextEdit.Range.Start.Character,
				"item %q TextEdit range should start at token start (col 6)", item.Label)
			assert.Equal(t, uint32(10), item.TextEdit.Range.End.Character,
				"item %q TextEdit range should end at cursor (col 10)", item.Label)
		}
	})

	t.Run("ignore_only_key_textEdit_includes_colon_space", func(t *testing.T) {
		t.Parallel()
		text := "lint:\n  ignore_only:\n    \n"
		items := getBufYAMLCompletionItems(
			parseYAMLDoc(text),
			text,
			protocol.Position{Line: 2, Character: 4},
		)
		require.NotNil(t, items)
		for _, item := range items {
			require.NotNil(t, item.TextEdit, "item %q has no TextEdit", item.Label)
			assert.Equal(t, item.Label+": ", item.TextEdit.NewText,
				"item %q TextEdit.NewText should be label + \": \"", item.Label)
		}
	})

	t.Run("bare_parent_key_items_indented", func(t *testing.T) {
		t.Parallel()
		// After bare "breaking:", completion items should have "  " indent in NewText.
		text := "breaking:\n"
		items := getBufYAMLCompletionItems(parseYAMLDoc(text), text, protocol.Position{Line: 1, Character: 0})
		require.NotNil(t, items)
		for _, item := range items {
			require.NotNil(t, item.TextEdit, "item %q has no TextEdit", item.Label)
			assert.Equal(t, "  "+item.Label+": ", item.TextEdit.NewText,
				"item %q TextEdit.NewText should have 2-space indent prefix", item.Label)
		}
	})

	t.Run("mid_token_range_extends_to_token_end", func(t *testing.T) {
		t.Parallel()
		// "  breaking" inside lint — cursor at col 7 (mid-token after "break").
		// editRange.End should extend to col 10 (end of "breaking"), not stop at cursor.
		text := "lint:\n  breaking\n"
		items := getBufYAMLCompletionItems(
			parseYAMLDoc(text),
			text,
			protocol.Position{Line: 1, Character: 7},
		)
		require.NotNil(t, items)
		for _, item := range items {
			require.NotNil(t, item.TextEdit, "item %q has no TextEdit", item.Label)
			assert.Equal(t, uint32(2), item.TextEdit.Range.Start.Character,
				"item %q TextEdit range should start at token start (col 2)", item.Label)
			assert.Equal(t, uint32(10), item.TextEdit.Range.End.Character,
				"item %q TextEdit range should end at token end (col 10), not cursor (col 7)", item.Label)
		}
	})

	t.Run("text_fallback_filters_inline_seq_first_key", func(t *testing.T) {
		t.Parallel()
		// With nil docNode (simulating a parse failure), the text-fallback should
		// still recognize "  - path: ." as having "path" at effective indent 4
		// and filter it from the suggestions.
		text := "modules:\n  - path: .\n    \n"
		items := getBufYAMLCompletionItems(nil, text, protocol.Position{Line: 2, Character: 4})
		require.NotNil(t, items)
		labels := completionLabels(items)
		assert.False(t, slices.Contains(labels, "path"),
			"text fallback should filter already-present inline seq key %q", "path")
		assert.True(t, slices.Contains(labels, "lint"),
			"text fallback should still offer sibling keys like %q", "lint")
	})
}

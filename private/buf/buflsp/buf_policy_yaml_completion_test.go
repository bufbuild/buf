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

func TestGetBufPolicyYAMLCompletionItems(t *testing.T) {
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
			wantLabels: []string{"version", "name", "lint", "breaking", "plugins"},
			wantKind:   protocol.CompletionItemKindField,
		},
		{
			name:       "top_level_partial",
			text:       "br\n",
			pos:        protocol.Position{Line: 0, Character: 2},
			wantLabels: []string{"breaking"},
			wantKind:   protocol.CompletionItemKindField,
		},

		// ── lint key completions ───────────────────────────────────────────
		{
			name:       "lint_keys",
			text:       "lint:\n  \n",
			pos:        protocol.Position{Line: 1, Character: 2},
			wantLabels: []string{"use", "except", "disable_builtin"},
			// buf.policy.yaml lint does not support ignore or ignore_only.
			wantAbsentLabels: []string{"ignore", "ignore_only", "disallow_comment_ignores"},
			wantKind:         protocol.CompletionItemKindField,
		},

		// ── breaking key completions ───────────────────────────────────────
		{
			name:       "breaking_keys",
			text:       "breaking:\n  \n",
			pos:        protocol.Position{Line: 1, Character: 2},
			wantLabels: []string{"use", "except", "ignore_unstable_packages", "disable_builtin"},
			// buf.policy.yaml breaking does not support ignore or ignore_only.
			wantAbsentLabels: []string{"ignore", "ignore_only"},
			wantKind:         protocol.CompletionItemKindField,
		},

		// ── plugins item key completions ───────────────────────────────────
		{
			name:       "plugins_item_keys",
			text:       "plugins:\n  - \n",
			pos:        protocol.Position{Line: 1, Character: 4},
			wantLabels: []string{"plugin", "options"},
			wantKind:   protocol.CompletionItemKindField,
		},

		// ── Value completions ──────────────────────────────────────────────
		{
			name:       "version_value",
			text:       "version: \n",
			pos:        protocol.Position{Line: 0, Character: 9},
			wantLabels: []string{"v2"},
			wantKind:   protocol.CompletionItemKindValue,
		},
		{
			name:       "disable_builtin_value",
			text:       "lint:\n  disable_builtin: \n",
			pos:        protocol.Position{Line: 1, Character: 18},
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

		// ── Sequence after existing items ─────────────────────────────────
		{
			name:       "lint_use_after_existing_item",
			text:       "lint:\n  use:\n    - STANDARD\n    - \n",
			pos:        protocol.Position{Line: 3, Character: 6},
			wantLabels: []string{"MINIMAL", "BASIC", "STANDARD", "COMMENTS"},
			wantKind:   protocol.CompletionItemKindValue,
		},

		// ── Blank lines between parent and child ──────────────────────────
		{
			name:       "blank_line_between_breaking_and_use",
			text:       "breaking:\n\n  use:\n    - \n",
			pos:        protocol.Position{Line: 3, Character: 6},
			wantLabels: []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"},
			wantKind:   protocol.CompletionItemKindValue,
		},

		// ── Partial key in breaking block ─────────────────────────────────
		{
			name:       "breaking_partial_key",
			text:       "breaking:\n  ign\n",
			pos:        protocol.Position{Line: 1, Character: 5},
			wantLabels: []string{"ignore_unstable_packages"},
			wantKind:   protocol.CompletionItemKindField,
		},

		// ── Plugin item continuation ───────────────────────────────────────
		{
			name:             "plugin_item_continuation",
			text:             "plugins:\n  - plugin: buf.build/foo/bar\n    \n",
			pos:              protocol.Position{Line: 2, Character: 4},
			wantLabels:       []string{"options"},
			wantAbsentLabels: []string{"plugin"},
			wantKind:         protocol.CompletionItemKindField,
		},

		// ── Bare-parent-key heuristic ─────────────────────────────────────
		{
			name:             "bare_parent_breaking",
			text:             "breaking:\n",
			pos:              protocol.Position{Line: 1, Character: 0},
			wantLabels:       []string{"use", "except", "ignore_unstable_packages", "disable_builtin"},
			wantAbsentLabels: []string{"version", "lint", "plugins", "ignore", "ignore_only"},
			wantKind:         protocol.CompletionItemKindField,
		},
		{
			name:             "bare_parent_lint",
			text:             "lint:\n",
			pos:              protocol.Position{Line: 1, Character: 0},
			wantLabels:       []string{"use", "except", "disable_builtin"},
			wantAbsentLabels: []string{"version", "breaking", "plugins"},
			wantKind:         protocol.CompletionItemKindField,
		},

		// ── No completions ─────────────────────────────────────────────────
		{
			name:          "list_marker_only",
			text:          "plugins:\n  -\n",
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
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			items := getBufPolicyYAMLCompletionItems(parseYAMLDoc(testCase.text), testCase.text, testCase.pos)

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

func TestGetBufPolicyYAMLCompletionItemsTextEdit(t *testing.T) {
	t.Parallel()

	t.Run("key_textEdit_includes_colon_space", func(t *testing.T) {
		t.Parallel()
		text := "lint:\n  \n"
		items := getBufPolicyYAMLCompletionItems(parseYAMLDoc(text), text, protocol.Position{Line: 1, Character: 2})
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
		items := getBufPolicyYAMLCompletionItems(
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
}

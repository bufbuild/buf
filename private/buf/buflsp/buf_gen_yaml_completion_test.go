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

// completionLabels extracts the labels from a slice of CompletionItems.
func completionLabels(items []protocol.CompletionItem) []string {
	labels := make([]string, len(items))
	for idx, item := range items {
		labels[idx] = item.Label
	}
	return labels
}

func TestGetBufGenYAMLCompletionItems(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		text             string
		pos              protocol.Position
		wantLabels       []string // all of these must appear in the result
		wantAbsentLabels []string // none of these may appear in the result
		wantKind         protocol.CompletionItemKind
		wantNilResult    bool
	}{
		// ── Top-level key completions ───────────────────────────────────────
		{
			name:       "top_level_empty_line",
			text:       "\n",
			pos:        protocol.Position{Line: 0, Character: 0},
			wantLabels: []string{"version", "clean", "managed", "plugins", "inputs"},
			wantKind:   protocol.CompletionItemKindField,
		},
		{
			name:       "top_level_partial_key",
			text:       "pl\n",
			pos:        protocol.Position{Line: 0, Character: 2},
			wantLabels: []string{"plugins"},
			wantKind:   protocol.CompletionItemKindField,
		},
		{
			name:             "top_level_after_existing",
			text:             "version: v2\n",
			pos:              protocol.Position{Line: 1, Character: 0},
			wantLabels:       []string{"clean", "managed", "plugins", "inputs"},
			wantAbsentLabels: []string{"version"},
			wantKind:         protocol.CompletionItemKindField,
		},

		// ── Plugin key completions ──────────────────────────────────────────
		{
			name: "plugin_item_first_key",
			text: "plugins:\n  - \n",
			pos:  protocol.Position{Line: 1, Character: 4},
			wantLabels: []string{
				"remote", "local", "protoc_builtin", "protoc_path", "out", "opt",
				"revision", "include_imports", "include_wkt", "strategy",
				"types", "exclude_types",
			},
			wantKind: protocol.CompletionItemKindField,
		},
		{
			// remote present: revision is available; protoc_path and strategy are absent
			// (protoc_path is protoc_builtin-only; strategy is local/protoc_builtin-only).
			name: "plugin_item_continuation_key",
			text: "plugins:\n  - remote: buf.build/foo/bar\n    \n",
			pos:  protocol.Position{Line: 2, Character: 4},
			wantLabels: []string{
				"out", "revision",
			},
			wantAbsentLabels: []string{"remote", "local", "protoc_builtin", "protoc_path", "strategy"},
			wantKind:         protocol.CompletionItemKindField,
		},
		{
			name:       "plugin_item_partial_key",
			text:       "plugins:\n  - prot\n",
			pos:        protocol.Position{Line: 1, Character: 7},
			wantLabels: []string{"protoc_builtin", "protoc_path"},
			wantKind:   protocol.CompletionItemKindField,
		},
		{
			// local present: strategy is available; revision (remote-only) and
			// protoc_path (protoc_builtin-only) must be absent.
			name:             "plugin_item_local_excludes_others",
			text:             "plugins:\n  - local: protoc-gen-go\n    \n",
			pos:              protocol.Position{Line: 2, Character: 4},
			wantLabels:       []string{"out", "opt", "strategy"},
			wantAbsentLabels: []string{"local", "remote", "protoc_builtin", "revision", "protoc_path"},
			wantKind:         protocol.CompletionItemKindField,
		},
		{
			// protoc_builtin present: protoc_path and strategy are available;
			// revision (remote-only) must be absent.
			name:             "plugin_item_protoc_builtin_excludes_others",
			text:             "plugins:\n  - protoc_builtin: java\n    \n",
			pos:              protocol.Position{Line: 2, Character: 4},
			wantLabels:       []string{"out", "opt", "protoc_path", "strategy"},
			wantAbsentLabels: []string{"protoc_builtin", "remote", "local", "revision"},
			wantKind:         protocol.CompletionItemKindField,
		},

		// ── Input key completions ────────────────────────────────────────────
		{
			name: "input_item_first_key",
			text: "inputs:\n  - \n",
			pos:  protocol.Position{Line: 1, Character: 4},
			wantLabels: []string{
				"directory", "module", "proto_file", "git_repo", "types", "paths",
			},
			wantKind: protocol.CompletionItemKindField,
		},
		{
			// directory present: all source type keys and all git/archive/proto_file-specific
			// keys must be absent; only the universal input options remain.
			name:             "input_item_source_excludes_others",
			text:             "inputs:\n  - directory: proto\n    \n",
			pos:              protocol.Position{Line: 2, Character: 4},
			wantLabels:       []string{"types", "paths", "exclude_paths", "exclude_types"},
			wantAbsentLabels: []string{"directory", "module", "proto_file", "git_repo", "tarball", "zip_archive", "binary_image", "json_image", "text_image", "yaml_image", "branch", "tag", "commit", "ref", "depth", "recurse_submodules", "include_package_files", "subdir", "strip_components", "compression"},
			wantKind:         protocol.CompletionItemKindField,
		},
		{
			// git_repo present: all other source type keys absent; git-specific keys available;
			// proto_file-only and archive-only keys absent.
			name:             "input_item_git_repo_source_excludes_others",
			text:             "inputs:\n  - git_repo: github.com/acme/protos\n    \n",
			pos:              protocol.Position{Line: 2, Character: 4},
			wantLabels:       []string{"branch", "tag", "commit", "subdir", "depth", "ref", "recurse_submodules"},
			wantAbsentLabels: []string{"git_repo", "directory", "module", "proto_file", "include_package_files", "strip_components", "compression"},
			wantKind:         protocol.CompletionItemKindField,
		},
		{
			// proto_file present: include_package_files is available; git and archive keys absent.
			name:             "input_item_proto_file_shows_package_files",
			text:             "inputs:\n  - proto_file: foo.proto\n    \n",
			pos:              protocol.Position{Line: 2, Character: 4},
			wantLabels:       []string{"include_package_files", "types", "paths"},
			wantAbsentLabels: []string{"branch", "tag", "commit", "subdir", "strip_components", "compression"},
			wantKind:         protocol.CompletionItemKindField,
		},
		{
			// tarball present: subdir, strip_components, and compression are available;
			// git-only and proto_file-only keys absent.
			name:             "input_item_tarball_shows_archive_keys",
			text:             "inputs:\n  - tarball: archive.tar.gz\n    \n",
			pos:              protocol.Position{Line: 2, Character: 4},
			wantLabels:       []string{"subdir", "strip_components", "compression", "types", "paths"},
			wantAbsentLabels: []string{"branch", "tag", "commit", "include_package_files"},
			wantKind:         protocol.CompletionItemKindField,
		},
		{
			// tag present: commit must be absent.
			name:             "input_item_tag_excludes_commit",
			text:             "inputs:\n  - git_repo: github.com/acme/protos\n    tag: v1.0.0\n    \n",
			pos:              protocol.Position{Line: 3, Character: 4},
			wantLabels:       []string{"branch", "subdir", "depth"},
			wantAbsentLabels: []string{"tag", "commit"},
			wantKind:         protocol.CompletionItemKindField,
		},
		{
			// commit present: tag must be absent.
			name:             "input_item_commit_excludes_tag",
			text:             "inputs:\n  - git_repo: github.com/acme/protos\n    commit: abc123\n    \n",
			pos:              protocol.Position{Line: 3, Character: 4},
			wantLabels:       []string{"branch", "subdir", "depth"},
			wantAbsentLabels: []string{"commit", "tag"},
			wantKind:         protocol.CompletionItemKindField,
		},

		// ── Managed key completions ─────────────────────────────────────────
		{
			name:       "managed_keys",
			text:       "managed:\n  \n",
			pos:        protocol.Position{Line: 1, Character: 2},
			wantLabels: []string{"enabled", "disable", "override"},
			wantKind:   protocol.CompletionItemKindField,
		},
		{
			name:             "managed_disable_rule_keys",
			text:             "managed:\n  disable:\n    - \n",
			pos:              protocol.Position{Line: 2, Character: 6},
			wantLabels:       []string{"file_option", "field_option", "module", "path", "field"},
			wantAbsentLabels: []string{"value"},
			wantKind:         protocol.CompletionItemKindField,
		},
		{
			name:       "managed_override_rule_keys",
			text:       "managed:\n  override:\n    - \n",
			pos:        protocol.Position{Line: 2, Character: 6},
			wantLabels: []string{"file_option", "field_option", "module", "path", "field", "value"},
			wantKind:   protocol.CompletionItemKindField,
		},
		{
			// file_option present: field_option must be absent; value is never valid in disable.
			name:             "managed_disable_file_option_excludes_field_option",
			text:             "managed:\n  disable:\n    - file_option: go_package\n      \n",
			pos:              protocol.Position{Line: 3, Character: 6},
			wantLabels:       []string{"module", "path"},
			wantAbsentLabels: []string{"file_option", "field_option", "value"},
			wantKind:         protocol.CompletionItemKindField,
		},
		{
			// field_option present: file_option must be absent.
			name:             "managed_override_field_option_excludes_file_option",
			text:             "managed:\n  override:\n    - field_option: jstype\n      \n",
			pos:              protocol.Position{Line: 3, Character: 6},
			wantLabels:       []string{"module", "path", "field", "value"},
			wantAbsentLabels: []string{"field_option", "file_option"},
			wantKind:         protocol.CompletionItemKindField,
		},

		// ── Value completions ────────────────────────────────────────────────
		{
			name:       "version_value",
			text:       "version: \n",
			pos:        protocol.Position{Line: 0, Character: 9},
			wantLabels: []string{"v2", "v1", "v1beta1"},
			wantKind:   protocol.CompletionItemKindValue,
		},
		{
			name:       "version_value_partial",
			text:       "version: v\n",
			pos:        protocol.Position{Line: 0, Character: 10},
			wantLabels: []string{"v2", "v1", "v1beta1"},
			wantKind:   protocol.CompletionItemKindValue,
		},
		{
			name:       "protoc_builtin_value",
			text:       "plugins:\n  - protoc_builtin: \n",
			pos:        protocol.Position{Line: 1, Character: 21},
			wantLabels: bufGenYAMLBuiltinPlugins,
			wantKind:   protocol.CompletionItemKindValue,
		},
		{
			name:       "protoc_builtin_value_partial",
			text:       "plugins:\n  - protoc_builtin: ja\n",
			pos:        protocol.Position{Line: 1, Character: 23},
			wantLabels: []string{"java"},
			wantKind:   protocol.CompletionItemKindValue,
		},
		{
			name:       "strategy_value",
			text:       "plugins:\n  - local: foo\n    strategy: \n",
			pos:        protocol.Position{Line: 2, Character: 14},
			wantLabels: []string{"directory", "all"},
			wantKind:   protocol.CompletionItemKindValue,
		},
		{
			name:       "clean_boolean_value",
			text:       "clean: \n",
			pos:        protocol.Position{Line: 0, Character: 7},
			wantLabels: []string{"true", "false"},
			wantKind:   protocol.CompletionItemKindValue,
		},
		{
			name:       "include_imports_boolean_value",
			text:       "plugins:\n  - remote: foo\n    include_imports: \n",
			pos:        protocol.Position{Line: 2, Character: 21},
			wantLabels: []string{"true", "false"},
			wantKind:   protocol.CompletionItemKindValue,
		},

		// ── Bare-parent-key heuristic ─────────────────────────────────────
		{
			name:             "bare_parent_managed",
			text:             "managed:\n",
			pos:              protocol.Position{Line: 1, Character: 0},
			wantLabels:       []string{"enabled", "disable", "override"},
			wantAbsentLabels: []string{"version", "plugins", "inputs", "clean"},
			wantKind:         protocol.CompletionItemKindField,
		},
		{
			// Sequence-valued keys (plugins, inputs) must not trigger the heuristic.
			name:             "bare_plugins_no_child_keys",
			text:             "plugins:\n",
			pos:              protocol.Position{Line: 1, Character: 0},
			wantLabels:       []string{"version", "managed", "inputs"},
			wantAbsentLabels: []string{"remote", "local", "out"},
			wantKind:         protocol.CompletionItemKindField,
		},

		// ── No completions ───────────────────────────────────────────────────
		{
			name:          "list_marker_only",
			text:          "plugins:\n  -\n",
			pos:           protocol.Position{Line: 1, Character: 3},
			wantNilResult: true,
		},
		{
			name:          "out_of_bounds_line",
			text:          "version: v2\n",
			pos:           protocol.Position{Line: 99, Character: 0},
			wantNilResult: true,
		},
		{
			name:          "no_value_completions_for_remote",
			text:          "plugins:\n  - remote: \n",
			pos:           protocol.Position{Line: 1, Character: 14},
			wantNilResult: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			items := getBufGenYAMLCompletionItems(parseYAMLDoc(testCase.text), testCase.text, testCase.pos)

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

func TestGetBufGenYAMLCompletionItemsTextEdit(t *testing.T) {
	t.Parallel()

	// Verify that key completions include ": " in the TextEdit and that value
	// completions replace only the partial token at the cursor.
	t.Run("key_textEdit_includes_colon_space", func(t *testing.T) {
		t.Parallel()
		text := "plugins:\n  - \n"
		items := getBufGenYAMLCompletionItems(parseYAMLDoc(text), text, protocol.Position{Line: 1, Character: 4})
		require.NotNil(t, items)
		for _, item := range items {
			require.NotNil(t, item.TextEdit, "item %q has no TextEdit", item.Label)
			assert.Equal(t, item.Label+": ", item.TextEdit.NewText,
				"item %q TextEdit.NewText should be label + \": \"", item.Label)
		}
	})

	t.Run("value_textEdit_replaces_partial_token", func(t *testing.T) {
		t.Parallel()
		// "version: v" — cursor at character 10, token "v" starts at character 9.
		text := "version: v\n"
		items := getBufGenYAMLCompletionItems(parseYAMLDoc(text), text, protocol.Position{Line: 0, Character: 10})
		require.NotNil(t, items)
		for _, item := range items {
			require.NotNil(t, item.TextEdit, "item %q has no TextEdit", item.Label)
			assert.Equal(t, uint32(9), item.TextEdit.Range.Start.Character,
				"item %q TextEdit range should start at token start (col 9)", item.Label)
			assert.Equal(t, uint32(10), item.TextEdit.Range.End.Character,
				"item %q TextEdit range should end at cursor (col 10)", item.Label)
		}
	})
}

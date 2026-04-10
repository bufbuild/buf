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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

// TestBufGenYAMLHoverMalformedYAML verifies that hovering over a buf.gen.yaml
// with invalid YAML syntax returns no hover and does not crash the server.
func TestBufGenYAMLHoverMalformedYAML(t *testing.T) {
	t.Parallel()

	absPath, err := filepath.Abs("testdata/buf_gen_yaml/invalid/buf.gen.yaml")
	require.NoError(t, err)

	clientJSONConn, bufGenYAMLURI, _ := setupLSPServerForBufYAML(t, absPath, nil)
	ctx := t.Context()

	var hover *protocol.Hover
	_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentHover, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: bufGenYAMLURI},
			Position:     protocol.Position{Line: 0, Character: 0},
		},
	}, &hover)
	require.NoError(t, err)
	assert.Nil(t, hover, "malformed YAML must not crash the server or return hover")
}

// TestBufGenYAMLHoverDidChange verifies that after a didChange notification,
// hover reflects the updated file content.
func TestBufGenYAMLHoverDidChange(t *testing.T) {
	t.Parallel()

	absPath, err := filepath.Abs("testdata/buf_gen_yaml/hover/buf.gen.yaml")
	require.NoError(t, err)

	clientJSONConn, bufGenYAMLURI, _ := setupLSPServerForBufYAML(t, absPath, nil)
	ctx := t.Context()

	// Replace the entire file with minimal content (version key on line 0).
	err = clientJSONConn.Notify(ctx, protocol.MethodTextDocumentDidChange, &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: bufGenYAMLURI},
			Version:                2,
		},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{
			{Text: "version: v2\n"},
		},
	})
	require.NoError(t, err)

	// Hover on version should still work after the update.
	var hover *protocol.Hover
	_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentHover, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: bufGenYAMLURI},
			Position:     protocol.Position{Line: 0, Character: 0},
		},
	}, &hover)
	require.NoError(t, err)
	require.NotNil(t, hover, "expected hover for version key after DidChange")
	assert.Contains(t, hover.Contents.Value, "version")
}

// TestBufGenYAMLHover verifies that hovering over buf.gen.yaml keys returns
// the correct markdown documentation, and that unrecognised positions return
// no hover.
func TestBufGenYAMLHover(t *testing.T) {
	t.Parallel()

	fixture := "testdata/buf_gen_yaml/hover/buf.gen.yaml"

	// The fixture layout (0-indexed lines):
	//  0: version: v2
	//  1: clean: false
	//  2: managed:
	//  3:   enabled: true
	//  4:   disable:
	//  5:     - file_option: go_package_prefix
	//  6:       module: buf.build/acme/petapis
	//  7:   override:
	//  8:     - file_option: java_package_prefix
	//  9:       value: com
	// 10: plugins:
	// 11:   - remote: buf.build/protocolbuffers/go
	// 12:     out: gen/go
	// 13:     opt: paths=source_relative
	// 14:     include_imports: true
	// 15:     include_wkt: false
	// 16:     revision: 1
	// 17:   - local: protoc-gen-grpc-gateway
	// 18:     out: gen/grpc
	// 19:     strategy: all
	// 20:     exclude_types:
	// 21:       - acme.v1.Internal
	// 22:   - protoc_builtin: java
	// 23:     protoc_path: /usr/local/bin/protoc
	// 24: inputs:
	// 25:   - directory: proto
	// 26:     types:
	// 27:       - acme.v1.FooService
	// 28:     paths:
	// 29:       - acme/v1/foo.proto
	// 30:     exclude_paths:
	// 31:       - acme/v1/internal.proto
	// 32:   - module: buf.build/acme/petapis
	// 33:     exclude_types:
	// 34:       - acme.v1.Internal
	// 35:   - proto_file: acme/v1/bar.proto
	// 36:     include_package_files: true
	// 37:   - git_repo: github.com/acme/protos
	// 38:     branch: main
	// 39:     subdir: proto
	// 40:     depth: 1

	tests := []struct {
		name      string
		line      uint32
		character uint32
		// expectedContains lists substrings that must appear in the hover markdown.
		// Leave nil (with expectNoHover true) to assert no hover is returned.
		expectedContains []string
		expectNoHover    bool
	}{
		// ── Top-level keys ──────────────────────────────────────────────────────
		{
			name: "version_key",
			line: 0, character: 0,
			expectedContains: []string{"version", "configuration format version", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#version"},
		},
		{
			name: "clean_key",
			line: 1, character: 0,
			expectedContains: []string{"clean", "output directories", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#clean"},
		},
		{
			name: "managed_key",
			line: 2, character: 0,
			expectedContains: []string{"managed", "managed mode", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#managed"},
		},
		{
			name: "plugins_key",
			line: 10, character: 0,
			expectedContains: []string{"plugins", "code generation plugins", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#plugins"},
		},
		{
			name: "inputs_key",
			line: 24, character: 0,
			expectedContains: []string{"inputs", "Protobuf sources", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#inputs"},
		},

		// ── managed sub-keys ────────────────────────────────────────────────────
		{
			name: "managed_enabled_key",
			line: 3, character: 2,
			expectedContains: []string{"managed.enabled", "managed mode globally", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#enabled"},
		},
		{
			name: "managed_disable_key",
			line: 4, character: 2,
			expectedContains: []string{"managed.disable", "exclude specific", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#disable"},
		},
		{
			name: "managed_override_key",
			line: 7, character: 2,
			expectedContains: []string{"managed.override", "override", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#override"},
		},

		// ── managed.disable rule keys ───────────────────────────────────────────
		{
			name: "managed_disable_file_option_key",
			line: 5, character: 6,
			expectedContains: []string{"file_option", "File-level Protobuf option", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#file_option"},
		},
		{
			name: "managed_disable_module_key",
			line: 6, character: 6,
			expectedContains: []string{"module", "BSR module", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#module"},
		},

		// ── managed.override rule keys ──────────────────────────────────────────
		{
			name: "managed_override_file_option_key",
			line: 8, character: 6,
			expectedContains: []string{"file_option", "File-level Protobuf option", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#file_option"},
		},
		{
			name: "managed_override_value_key",
			line: 9, character: 6,
			expectedContains: []string{"value", "value to set for the option", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#value"},
		},

		// ── plugin entry keys ───────────────────────────────────────────────────
		{
			name: "plugin_remote_key",
			line: 11, character: 4,
			expectedContains: []string{"remote", "Remote BSR plugin", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#remote"},
		},
		{
			name: "plugin_out_key",
			line: 12, character: 4,
			expectedContains: []string{"out", "Output directory", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#out"},
		},
		{
			name: "plugin_opt_key",
			line: 13, character: 4,
			expectedContains: []string{"opt", "Plugin options", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#opt"},
		},
		{
			name: "plugin_include_imports_key",
			line: 14, character: 4,
			expectedContains: []string{"include_imports", "generates code for all files imported", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#include_imports"},
		},
		{
			name: "plugin_include_wkt_key",
			line: 15, character: 4,
			expectedContains: []string{"include_wkt", "Well-Known Types", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#include_wkt"},
		},
		{
			name: "plugin_revision_key",
			line: 16, character: 4,
			expectedContains: []string{"revision", "revision", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#revision"},
		},
		{
			name: "plugin_local_key",
			line: 17, character: 4,
			expectedContains: []string{"local", "local plugin binary", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#local"},
		},
		{
			name: "plugin_strategy_key",
			line: 19, character: 4,
			expectedContains: []string{"strategy", "invocation strategy", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#strategy"},
		},
		{
			name: "plugin_exclude_types_key",
			line: 20, character: 4,
			expectedContains: []string{"exclude_types", "Exclude", "type names", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#exclude_types"},
		},
		{
			name: "plugin_protoc_builtin_key",
			line: 22, character: 4,
			expectedContains: []string{"protoc_builtin", "protoc", "generator name", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#protoc_builtin"},
		},
		{
			name: "plugin_protoc_path_key",
			line: 23, character: 4,
			expectedContains: []string{"protoc_path", "protoc", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#protoc_path"},
		},

		// ── input entry keys ────────────────────────────────────────────────────
		{
			name: "input_directory_key",
			line: 25, character: 4,
			expectedContains: []string{"directory", "Local directory", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#directory"},
		},
		{
			name: "input_types_key",
			line: 26, character: 4,
			expectedContains: []string{"types", "fully-qualified type names", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#types"},
		},
		{
			name: "input_paths_key",
			line: 28, character: 4,
			expectedContains: []string{"paths", "relative paths", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#paths"},
		},
		{
			name: "input_exclude_paths_key",
			line: 30, character: 4,
			expectedContains: []string{"exclude_paths", "Exclude", "relative paths", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#exclude_paths"},
		},
		{
			name: "input_module_key",
			line: 32, character: 4,
			expectedContains: []string{"module", "Remote BSR module", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#module"},
		},
		{
			name: "input_exclude_types_key",
			line: 33, character: 4,
			expectedContains: []string{"exclude_types", "Exclude", "type names", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#exclude_types"},
		},
		{
			name: "input_proto_file_key",
			line: 35, character: 4,
			expectedContains: []string{"proto_file", "single `.proto` file", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#proto_file"},
		},
		{
			name: "input_include_package_files_key",
			line: 36, character: 4,
			expectedContains: []string{"include_package_files", "same package", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#include_package_files"},
		},
		{
			name: "input_git_repo_key",
			line: 37, character: 4,
			expectedContains: []string{"git_repo", "Git repository", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#git_repo"},
		},
		{
			name: "input_branch_key",
			line: 38, character: 4,
			expectedContains: []string{"branch", "Git branch", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#branch"},
		},
		{
			name: "input_subdir_key",
			line: 39, character: 4,
			expectedContains: []string{"subdir", "Subdirectory", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#subdir"},
		},
		{
			name: "input_depth_key",
			line: 40, character: 4,
			expectedContains: []string{"depth", "clone depth", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#depth"},
		},

		// ── Positions that should return no hover ────────────────────────────────
		{
			// Scalar value in a sequence (not a key).
			name: "sequence_value_no_hover",
			line: 21, character: 6,
			expectNoHover: true,
		},
		{
			// Off the end of the file entirely.
			name: "off_file_no_hover",
			line: 999, character: 0,
			expectNoHover: true,
		},
		{
			// Mid-line whitespace before a key.
			name: "whitespace_no_hover",
			line: 3, character: 0,
			expectNoHover: true,
		},
	}

	absPath, err := filepath.Abs(fixture)
	require.NoError(t, err)

	clientJSONConn, bufGenYAMLURI, _ := setupLSPServerForBufYAML(t, absPath, nil)
	ctx := t.Context()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var hover *protocol.Hover
			_, err := clientJSONConn.Call(ctx, protocol.MethodTextDocumentHover, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: bufGenYAMLURI},
					Position:     protocol.Position{Line: tc.line, Character: tc.character},
				},
			}, &hover)
			require.NoError(t, err)

			if tc.expectNoHover {
				assert.Nil(t, hover, "expected no hover at (%d, %d)", tc.line, tc.character)
				return
			}
			require.NotNil(t, hover, "expected hover at (%d, %d)", tc.line, tc.character)
			assert.Equal(t, protocol.Markdown, hover.Contents.Kind)
			for _, want := range tc.expectedContains {
				assert.Contains(t, hover.Contents.Value, want,
					"hover at (%d, %d) should contain %q", tc.line, tc.character, want)
			}
		})
	}
}

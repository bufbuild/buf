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

// TestBufGenYAMLCompletion verifies that completion items are returned for buf.gen.yaml
// files at various cursor positions, and that already-present keys are excluded.
func TestBufGenYAMLCompletion(t *testing.T) {
	t.Parallel()

	fixture := "testdata/buf_gen_yaml/completion/buf.gen.yaml"

	// The fixture layout (0-indexed lines):
	//  0: version: v2
	//  1: clean: false
	//  2: managed:
	//  3:   enabled: true
	//  4: plugins:
	//  5:   - remote: buf.build/protocolbuffers/go
	//  6:     out: gen/go
	//  7: inputs:
	//  8:   - directory: proto

	tests := []struct {
		name         string
		line         uint32
		character    uint32
		wantContains []string
		wantAbsent   []string
	}{
		{
			// Cursor after "version: " — value position for the version key.
			name: "version_value",
			line: 0, character: 9,
			wantContains: []string{"v2", "v1", "v1beta1"},
		},
		{
			// Cursor after "clean: " — value position for the clean key.
			name: "clean_value",
			line: 1, character: 7,
			wantContains: []string{"true", "false"},
		},
		{
			// Cursor at indent 2 on "  enabled: true" — key position inside managed.
			// "enabled" already exists so it must be absent.
			name: "managed_keys_no_enabled",
			line: 3, character: 2,
			wantContains: []string{"disable", "override"},
			wantAbsent:   []string{"enabled"},
		},
		{
			// Cursor at indent 4 on "    out: gen/go" — key position inside a plugins item.
			// "remote" and "out" already exist so they must be absent.
			// "local" and "protoc_builtin" are mutually exclusive with "remote" and must
			// also be absent.
			name: "plugin_keys_no_remote_out",
			line: 6, character: 4,
			wantContains: []string{"opt", "revision", "include_imports"},
			wantAbsent:   []string{"remote", "out", "local", "protoc_builtin"},
		},
		{
			// Cursor at "  - directory: proto" char 4 — key position inside an inputs item.
			// "directory" and all other source type keys must be absent (mutually exclusive).
			name: "input_keys_no_source_types",
			line: 8, character: 4,
			wantContains: []string{"types", "paths", "exclude_paths"},
			wantAbsent:   []string{"directory", "module", "git_repo", "proto_file"},
		},
	}

	absPath, err := filepath.Abs(fixture)
	require.NoError(t, err)

	clientJSONConn, bufGenYAMLURI, _ := setupLSPServerForBufYAML(t, absPath, nil, nil)
	ctx := t.Context()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var list *protocol.CompletionList
			_, err := clientJSONConn.Call(ctx, protocol.MethodTextDocumentCompletion, protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: bufGenYAMLURI},
					Position:     protocol.Position{Line: tc.line, Character: tc.character},
				},
			}, &list)
			require.NoError(t, err)

			require.NotNil(t, list, "expected completions at (%d, %d)", tc.line, tc.character)
			labels := make([]string, len(list.Items))
			for i, item := range list.Items {
				labels[i] = item.Label
			}
			for _, want := range tc.wantContains {
				assert.Contains(t, labels, want,
					"completions at (%d, %d) should contain %q", tc.line, tc.character, want)
			}
			for _, absent := range tc.wantAbsent {
				assert.NotContains(t, labels, absent,
					"completions at (%d, %d) should not contain %q", tc.line, tc.character, absent)
			}
		})
	}
}

// TestBufGenYAMLDocumentLinks verifies that document links are returned for
// remote plugin and input module BSR references in buf.gen.yaml files.
func TestBufGenYAMLDocumentLinks(t *testing.T) {
	t.Parallel()

	// Fixture layout (0-indexed lines):
	//  0: version: v2
	//  1: plugins:
	//  2:   - remote: buf.build/protocolbuffers/go
	//  3:     out: gen/go
	//  4:   - remote: buf.build/bufbuild/es:v2.2.2
	//  5:     out: gen/es
	//  6:   - local: protoc-gen-custom
	//  7:     out: gen/custom
	//  8: inputs:
	//  9:   - module: buf.build/acme/petapis
	// 10:   - directory: proto

	tests := []struct {
		name      string
		fixture   string
		wantLinks []protocol.DocumentLink
	}{
		{
			name:    "no_plugins_or_inputs",
			fixture: "testdata/buf_gen_yaml/invalid/buf.gen.yaml",
			// Malformed YAML must not crash; returns no links.
		},
		{
			name:    "with_remote_plugins_and_input_modules",
			fixture: "testdata/buf_gen_yaml/document_link/buf.gen.yaml",
			wantLinks: []protocol.DocumentLink{
				{
					// plugins[0].remote: buf.build/protocolbuffers/go
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 12},
						End:   protocol.Position{Line: 2, Character: 40},
					},
					Target: "https://buf.build/protocolbuffers/go",
				},
				{
					// plugins[1].remote: buf.build/bufbuild/es:v2.2.2
					Range: protocol.Range{
						Start: protocol.Position{Line: 4, Character: 12},
						End:   protocol.Position{Line: 4, Character: 40},
					},
					Target: "https://buf.build/bufbuild/es/docs/v2.2.2",
				},
				{
					// inputs[0].module: buf.build/acme/petapis
					Range: protocol.Range{
						Start: protocol.Position{Line: 9, Character: 12},
						End:   protocol.Position{Line: 9, Character: 34},
					},
					Target: "https://buf.build/acme/petapis",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			absPath, err := filepath.Abs(tc.fixture)
			require.NoError(t, err)

			clientJSONConn, bufGenYAMLURI, _ := setupLSPServerForBufYAML(t, absPath, nil, nil)
			ctx := t.Context()

			var links []protocol.DocumentLink
			_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentDocumentLink, &protocol.DocumentLinkParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: bufGenYAMLURI},
			}, &links)
			require.NoError(t, err)
			require.Len(t, links, len(tc.wantLinks))
			for i, want := range tc.wantLinks {
				assert.Equal(t, want.Range, links[i].Range, "link %d range", i)
				assert.Equal(t, want.Target, links[i].Target, "link %d target", i)
			}
		})
	}
}

// TestBufGenYAMLHoverMalformedYAML verifies that hovering over a buf.gen.yaml
// with invalid YAML syntax returns no hover and does not crash the server.
func TestBufGenYAMLHoverMalformedYAML(t *testing.T) {
	t.Parallel()

	absPath, err := filepath.Abs("testdata/buf_gen_yaml/invalid/buf.gen.yaml")
	require.NoError(t, err)

	clientJSONConn, bufGenYAMLURI, _ := setupLSPServerForBufYAML(t, absPath, nil, nil)
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

	clientJSONConn, bufGenYAMLURI, _ := setupLSPServerForBufYAML(t, absPath, nil, nil)
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
			expectedContains: []string{"file_option", "File-level Protobuf option", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#disable"},
		},
		{
			name: "managed_disable_module_key",
			line: 6, character: 6,
			expectedContains: []string{"module", "BSR module", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#disable"},
		},

		// ── managed.override rule keys ──────────────────────────────────────────
		{
			name: "managed_override_file_option_key",
			line: 8, character: 6,
			expectedContains: []string{"file_option", "File-level Protobuf option", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#disable"},
		},
		{
			name: "managed_override_value_key",
			line: 9, character: 6,
			expectedContains: []string{"value", "value to set for the option", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#override"},
		},

		// ── plugin entry keys ───────────────────────────────────────────────────
		{
			name: "plugin_remote_key",
			line: 11, character: 4,
			expectedContains: []string{"remote", "Remote BSR plugin", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#plugins"},
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
			expectedContains: []string{"revision", "revision", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#plugins"},
		},
		{
			name: "plugin_local_key",
			line: 17, character: 4,
			expectedContains: []string{"local", "local plugin binary", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#plugins"},
		},
		{
			name: "plugin_strategy_key",
			line: 19, character: 4,
			expectedContains: []string{"strategy", "invocation strategy", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#strategy"},
		},
		{
			name: "plugin_exclude_types_key",
			line: 20, character: 4,
			expectedContains: []string{"exclude_types", "Exclude", "type names", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#exclude-types"},
		},
		{
			name: "plugin_protoc_builtin_key",
			line: 22, character: 4,
			expectedContains: []string{"protoc_builtin", "protoc", "generator name", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#plugins"},
		},
		{
			name: "plugin_protoc_path_key",
			line: 23, character: 4,
			expectedContains: []string{"protoc_path", "protoc", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#plugins"},
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
			expectedContains: []string{"paths", "relative paths", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#inputs"},
		},
		{
			name: "input_exclude_paths_key",
			line: 30, character: 4,
			expectedContains: []string{"exclude_paths", "Exclude", "relative paths", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#inputs"},
		},
		{
			name: "input_module_key",
			line: 32, character: 4,
			expectedContains: []string{"module", "Remote BSR module", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#module"},
		},
		{
			name: "input_exclude_types_key",
			line: 33, character: 4,
			expectedContains: []string{"exclude_types", "Exclude", "type names", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#exclude-types"},
		},
		{
			name: "input_proto_file_key",
			line: 35, character: 4,
			expectedContains: []string{"proto_file", "single `.proto` file", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#proto_file"},
		},
		{
			name: "input_include_package_files_key",
			line: 36, character: 4,
			expectedContains: []string{"include_package_files", "same package", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#proto_file"},
		},
		{
			name: "input_git_repo_key",
			line: 37, character: 4,
			expectedContains: []string{"git_repo", "Git repository", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#git_repo"},
		},
		{
			name: "input_branch_key",
			line: 38, character: 4,
			expectedContains: []string{"branch", "Git branch", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#git_repo"},
		},
		{
			name: "input_subdir_key",
			line: 39, character: 4,
			expectedContains: []string{"subdir", "Subdirectory", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#subdir"},
		},
		{
			name: "input_depth_key",
			line: 40, character: 4,
			expectedContains: []string{"depth", "clone depth", "https://buf.build/docs/configuration/v2/buf-gen-yaml/#git_repo"},
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

	clientJSONConn, bufGenYAMLURI, _ := setupLSPServerForBufYAML(t, absPath, nil, nil)
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

// TestBufGenYAMLCodeLens verifies the code lenses returned for buf.gen.yaml files.
// "Run buf generate" is always returned at line 0.
func TestBufGenYAMLCodeLens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		fixture string
	}{
		{
			name:    "no_plugins",
			fixture: "testdata/buf_gen_yaml/invalid/buf.gen.yaml",
		},
		{
			name:    "with_versioned_remote_plugin",
			fixture: "testdata/buf_gen_yaml/document_link/buf.gen.yaml",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			absPath, err := filepath.Abs(tc.fixture)
			require.NoError(t, err)

			clientJSONConn, bufGenYAMLURI, _ := setupLSPServerForBufYAML(t, absPath, nil, nil)
			ctx := t.Context()

			var lenses []protocol.CodeLens
			_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentCodeLens, &protocol.CodeLensParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: bufGenYAMLURI},
			}, &lenses)
			require.NoError(t, err)
			require.Len(t, lenses, 1)
			require.NotNil(t, lenses[0].Command)
			assert.Equal(t, "Run buf generate", lenses[0].Command.Title)
			assert.Equal(t, uint32(0), lenses[0].Range.Start.Line, "Run buf generate lens should be at line 0")
		})
	}
}

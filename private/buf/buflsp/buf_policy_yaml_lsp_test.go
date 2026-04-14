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

// TestBufPolicyYAMLDocumentLinks verifies that document links are returned for
// BSR references in buf.policy.yaml files: the top-level name field and any
// plugins[*].plugin values that are BSR references.
func TestBufPolicyYAMLDocumentLinks(t *testing.T) {
	t.Parallel()

	// Fixture layout (0-indexed lines):
	//  0: version: v2
	//  1: name: buf.build/acme/my-policy
	//  2: lint:
	//  3:   use:
	//  4:     - STANDARD
	//  5: breaking:
	//  6:   use:
	//  7:     - FILE
	//  8: plugins:
	//  9:   - plugin: buf.build/acme/my-lint-plugin
	// 10:     options:
	// 11:       key: value
	// 12:   - plugin: local-linter-binary
	// 13:     options:
	// 14:       key: value

	tests := []struct {
		name      string
		fixture   string
		wantLinks []protocol.DocumentLink
	}{
		{
			name:    "invalid",
			fixture: "testdata/buf_policy_yaml/invalid/buf.policy.yaml",
			// Malformed YAML must not crash; returns no links.
		},
		{
			name:    "with_bsr_name_and_plugins",
			fixture: "testdata/buf_policy_yaml/document_link/buf.policy.yaml",
			wantLinks: []protocol.DocumentLink{
				{
					// name: buf.build/acme/my-policy
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 6},
						End:   protocol.Position{Line: 1, Character: 30},
					},
					Target: "https://buf.build/acme/my-policy",
				},
				{
					// plugins[0].plugin: buf.build/acme/my-lint-plugin
					Range: protocol.Range{
						Start: protocol.Position{Line: 9, Character: 12},
						End:   protocol.Position{Line: 9, Character: 41},
					},
					Target: "https://buf.build/acme/my-lint-plugin",
				},
				// plugins[1].plugin: local-linter-binary is skipped (not a BSR ref).
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			absPath, err := filepath.Abs(tc.fixture)
			require.NoError(t, err)

			clientJSONConn, bufPolicyYAMLURI, _ := setupLSPServerForBufYAML(t, absPath, nil)
			ctx := t.Context()

			var links []protocol.DocumentLink
			_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentDocumentLink, &protocol.DocumentLinkParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: bufPolicyYAMLURI},
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

// TestBufPolicyYAMLHoverMalformedYAML verifies that hovering over a buf.policy.yaml
// with invalid YAML syntax returns no hover and does not crash the server.
func TestBufPolicyYAMLHoverMalformedYAML(t *testing.T) {
	t.Parallel()

	absPath, err := filepath.Abs("testdata/buf_policy_yaml/invalid/buf.policy.yaml")
	require.NoError(t, err)

	clientJSONConn, bufPolicyYAMLURI, _ := setupLSPServerForBufYAML(t, absPath, nil)
	ctx := t.Context()

	var hover *protocol.Hover
	_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentHover, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: bufPolicyYAMLURI},
			Position:     protocol.Position{Line: 0, Character: 0},
		},
	}, &hover)
	require.NoError(t, err)
	assert.Nil(t, hover, "malformed YAML must not crash the server or return hover")
}

// TestBufPolicyYAMLHover verifies that hovering over buf.policy.yaml keys and
// rule names returns the correct markdown documentation.
func TestBufPolicyYAMLHover(t *testing.T) {
	t.Parallel()

	fixture := "testdata/buf_policy_yaml/hover/buf.policy.yaml"

	// The fixture layout (0-indexed lines):
	//  0: version: v2
	//  1: name: buf.build/acme/my-policy
	//  2: lint:
	//  3:   use:
	//  4:     - STANDARD
	//  5:     - COMMENTS
	//  6:   except:
	//  7:     - IMPORT_USED
	//  8:   enum_zero_value_suffix: _UNSPECIFIED
	//  9:   service_suffix: Service
	// 10:   rpc_allow_same_request_response: false
	// 11:   rpc_allow_google_protobuf_empty_requests: false
	// 12:   rpc_allow_google_protobuf_empty_responses: false
	// 13:   disable_builtin: false
	// 14: breaking:
	// 15:   use:
	// 16:     - FILE
	// 17:   except:
	// 18:     - ENUM_NO_DELETE
	// 19:   ignore_unstable_packages: false
	// 20:   disable_builtin: false
	// 21: plugins:
	// 22:   - plugin: buf.build/acme/my-plugin
	// 23:     options:

	tests := []struct {
		name             string
		line             uint32
		character        uint32
		expectedContains []string
		expectNoHover    bool
	}{
		// ── Top-level keys ──────────────────────────────────────────────────────
		{
			name: "version_key",
			line: 0, character: 0,
			expectedContains: []string{"version", "configuration format version", "https://buf.build/docs/configuration/v2/buf-policy-yaml/#version"},
		},
		{
			name: "name_key",
			line: 1, character: 0,
			expectedContains: []string{"name", "Buf Schema Registry path", "https://buf.build/docs/configuration/v2/buf-policy-yaml/#name"},
		},
		{
			name: "lint_key",
			line: 2, character: 0,
			expectedContains: []string{"lint", "lint rules", "https://buf.build/docs/configuration/v2/buf-policy-yaml/#lint"},
		},
		{
			name: "breaking_key",
			line: 14, character: 0,
			expectedContains: []string{"breaking", "breaking change", "https://buf.build/docs/configuration/v2/buf-policy-yaml/#breaking"},
		},
		{
			name: "plugins_key",
			line: 21, character: 0,
			expectedContains: []string{"plugins", "lint and breaking change plugins", "https://buf.build/docs/configuration/v2/buf-policy-yaml/#plugins"},
		},

		// ── lint sub-keys ───────────────────────────────────────────────────────
		{
			name: "lint_use_key",
			line: 3, character: 2,
			expectedContains: []string{"lint.use", "rule categories", "https://buf.build/docs/lint/rules/"},
		},
		{
			name: "lint_except_key",
			line: 6, character: 2,
			expectedContains: []string{"lint.except", "Removes specific rules", "https://buf.build/docs/configuration/v2/buf-policy-yaml/#lint"},
		},
		{
			name: "lint_enum_zero_value_suffix_key",
			line: 8, character: 2,
			expectedContains: []string{"lint.enum_zero_value_suffix", "ENUM_ZERO_VALUE_SUFFIX", "https://buf.build/docs/configuration/v2/buf-policy-yaml/#lint"},
		},
		{
			name: "lint_service_suffix_key",
			line: 9, character: 2,
			expectedContains: []string{"lint.service_suffix", "SERVICE_SUFFIX", "https://buf.build/docs/configuration/v2/buf-policy-yaml/#lint"},
		},
		{
			name: "lint_rpc_allow_same_request_response_key",
			line: 10, character: 2,
			expectedContains: []string{"lint.rpc_allow_same_request_response", "same message type", "https://buf.build/docs/configuration/v2/buf-policy-yaml/#lint"},
		},
		{
			name: "lint_disable_builtin_key",
			line: 13, character: 2,
			expectedContains: []string{"lint.disable_builtin", "built-in lint rules", "https://buf.build/docs/configuration/v2/buf-policy-yaml/#lint"},
		},

		// ── breaking sub-keys ───────────────────────────────────────────────────
		{
			name: "breaking_use_key",
			line: 15, character: 2,
			expectedContains: []string{"breaking.use", "rule categories", "https://buf.build/docs/breaking/rules/"},
		},
		{
			name: "breaking_ignore_unstable_packages_key",
			line: 19, character: 2,
			expectedContains: []string{"breaking.ignore_unstable_packages", "unstable", "https://buf.build/docs/configuration/v2/buf-policy-yaml/#breaking"},
		},
		{
			name: "breaking_disable_builtin_key",
			line: 20, character: 2,
			expectedContains: []string{"breaking.disable_builtin", "built-in breaking", "https://buf.build/docs/configuration/v2/buf-policy-yaml/#breaking"},
		},

		// ── Rule names as values in use/except ──────────────────────────────────
		{
			name: "lint_use_value_STANDARD",
			line: 4, character: 6,
			expectedContains: []string{"STANDARD", "default lint rule set", "https://buf.build/docs/lint/rules/"},
		},
		{
			name: "lint_use_value_COMMENTS",
			line: 5, character: 6,
			expectedContains: []string{"COMMENTS", "leading comments", "https://buf.build/docs/lint/rules/"},
		},
		{
			name: "lint_except_value_IMPORT_USED",
			line: 7, character: 6,
			expectedContains: []string{"IMPORT_USED", "imported files", "https://buf.build/docs/lint/rules/"},
		},
		{
			name: "breaking_use_value_FILE",
			line: 16, character: 6,
			expectedContains: []string{"FILE", "generated code", "https://buf.build/docs/breaking/rules/"},
		},
		{
			name: "breaking_except_value_ENUM_NO_DELETE",
			line: 18, character: 6,
			expectedContains: []string{"ENUM_NO_DELETE", "enum type is deleted", "https://buf.build/docs/breaking/rules/"},
		},

		// ── plugin entry keys ───────────────────────────────────────────────────
		{
			name: "plugin_plugin_key",
			line: 22, character: 4,
			expectedContains: []string{"plugin", "Plugin location", "https://buf.build/docs/configuration/v2/buf-policy-yaml/#plugins"},
		},
		{
			name: "plugin_options_key",
			line: 23, character: 4,
			expectedContains: []string{"options", "Key-value pairs", "https://buf.build/docs/configuration/v2/buf-policy-yaml/#plugins"},
		},

		// ── Positions that should return no hover ────────────────────────────────
		{
			name: "off_file_no_hover",
			line: 999, character: 0,
			expectNoHover: true,
		},
		{
			name: "whitespace_no_hover",
			line: 3, character: 0,
			expectNoHover: true,
		},
	}

	absPath, err := filepath.Abs(fixture)
	require.NoError(t, err)

	clientJSONConn, bufPolicyYAMLURI, _ := setupLSPServerForBufYAML(t, absPath, nil)
	ctx := t.Context()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var hover *protocol.Hover
			_, err := clientJSONConn.Call(ctx, protocol.MethodTextDocumentHover, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: bufPolicyYAMLURI},
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

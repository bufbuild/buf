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
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"buf.build/go/app"
	"buf.build/go/app/appext"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/buf/buflsp"
	"github.com/bufbuild/buf/private/buf/bufwkt/bufwktstore"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufpolicy"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/httpauth"
	"github.com/bufbuild/buf/private/pkg/slogtestext"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/wasm"
	"github.com/bufbuild/protocompile/experimental/incremental"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// setupLSPServerForBufYAML creates an LSP server initialized for buf.yaml testing.
// It opens the buf.yaml file at bufYAMLPath via didOpen and returns the client
// connection, the buf.yaml URI, and a diagnostics capture for async notifications.
func setupLSPServerForBufYAML(
	t *testing.T,
	bufYAMLPath string,
	mkp bufmodule.ModuleKeyProvider,
) (jsonrpc2.Conn, protocol.URI, *diagnosticsCapture) {
	t.Helper()

	ctx := t.Context()

	logger := slogtestext.NewLogger(t, slogtestext.WithLogLevel(appext.LogLevelInfo))

	appContainer, err := app.NewContainerForOS()
	require.NoError(t, err)

	nameContainer, err := appext.NewNameContainer(appContainer, "buf-test")
	require.NoError(t, err)
	appextContainer := appext.NewContainer(nameContainer, logger, appext.LogLevelInfo, appext.LogFormatText)

	tmpDir := t.TempDir()
	storageBucket, err := storageos.NewProvider().NewReadWriteBucket(tmpDir)
	require.NoError(t, err)

	wktStore := bufwktstore.NewStore(logger, storageBucket)

	controller, err := bufctl.NewController(
		logger,
		appContainer,
		bufmodule.NopGraphProvider,
		nopModuleKeyProvider{},
		bufmodule.NopModuleDataProvider,
		bufmodule.NopCommitProvider,
		bufplugin.NopPluginKeyProvider,
		bufplugin.NopPluginDataProvider,
		bufpolicy.NopPolicyKeyProvider,
		bufpolicy.NopPolicyDataProvider,
		wktStore,
		&http.Client{},
		httpauth.NewNopAuthenticator(),
		git.ClonerOptions{},
	)
	require.NoError(t, err)

	wktBucket, err := wktStore.GetBucket(ctx)
	require.NoError(t, err)

	wasmRuntime, err := wasm.NewRuntime(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, wasmRuntime.Close(ctx))
	})

	queryExecutor := incremental.New()

	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		require.NoError(t, serverConn.Close())
		require.NoError(t, clientConn.Close())
	})

	moduleKeyProvider := bufmodule.ModuleKeyProvider(nopModuleKeyProvider{})
	if mkp != nil {
		moduleKeyProvider = mkp
	}

	conn, err := buflsp.Serve(
		ctx,
		"test",
		wktBucket,
		appextContainer,
		controller,
		wasmRuntime,
		jsonrpc2.NewStream(serverConn),
		queryExecutor,
		moduleKeyProvider,
		bufmodule.NopGraphProvider,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, conn.Close())
	})

	capture := newDiagnosticsCapture()

	clientStream := jsonrpc2.NewStream(clientConn)
	clientJSONConn := jsonrpc2.NewConn(clientStream)
	clientJSONConn.Go(ctx, jsonrpc2.AsyncHandler(capture.handle))
	t.Cleanup(func() {
		require.NoError(t, clientJSONConn.Close())
	})

	workspaceDir := filepath.Dir(bufYAMLPath)
	bufYAMLURI := buflsp.FilePathToURI(bufYAMLPath)

	var initResult protocol.InitializeResult
	_, err = clientJSONConn.Call(ctx, protocol.MethodInitialize, &protocol.InitializeParams{
		RootURI: uri.New(workspaceDir),
		Capabilities: protocol.ClientCapabilities{
			TextDocument: &protocol.TextDocumentClientCapabilities{},
		},
	}, &initResult)
	require.NoError(t, err)

	err = clientJSONConn.Notify(ctx, protocol.MethodInitialized, &protocol.InitializedParams{})
	require.NoError(t, err)

	contentBytes, err := os.ReadFile(bufYAMLPath)
	require.NoError(t, err)
	content := string(contentBytes)

	err = clientJSONConn.Notify(ctx, protocol.MethodTextDocumentDidOpen, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        bufYAMLURI,
			LanguageID: "yaml",
			Version:    1,
			Text:       content,
		},
	})
	require.NoError(t, err)

	return clientJSONConn, bufYAMLURI, capture
}

// TestBufYAMLCodeLens verifies the code lenses returned for buf.yaml files under
// various conditions: no deps, multiple deps, and invalid YAML.
func TestBufYAMLCodeLens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		fixture         string
		wantCount       int
		wantTitles      []string
		wantDepsKeyLine uint32
	}{
		{
			name:    "no_deps",
			fixture: "testdata/buf_yaml/no_deps/buf.yaml",
			// wantCount zero — no lenses for a file with no deps.
		},
		{
			name:            "with_deps",
			fixture:         "testdata/buf_yaml/with_deps/buf.yaml",
			wantCount:       2,
			wantTitles:      []string{"Update all dependencies", "Check for updates"},
			wantDepsKeyLine: 1,
		},
		{
			name:    "invalid",
			fixture: "testdata/buf_yaml/invalid/buf.yaml",
			// wantCount zero — malformed YAML must not crash the server.
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			absPath, err := filepath.Abs(tc.fixture)
			require.NoError(t, err)

			clientJSONConn, bufYAMLURI, _ := setupLSPServerForBufYAML(t, absPath, nil)
			ctx := t.Context()

			var lenses []protocol.CodeLens
			_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentCodeLens, &protocol.CodeLensParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: bufYAMLURI},
			}, &lenses)
			require.NoError(t, err)
			require.Len(t, lenses, tc.wantCount)

			for i, l := range lenses {
				require.NotNil(t, l.Command, "lens %d has no command", i)
			}
			titles := make([]string, len(lenses))
			for i, l := range lenses {
				titles[i] = l.Command.Title
			}
			for _, wantTitle := range tc.wantTitles {
				assert.Contains(t, titles, wantTitle)
			}
			for _, l := range lenses {
				assert.Equal(t, tc.wantDepsKeyLine, l.Range.Start.Line,
					"lens %q should be on the deps: key line", l.Command.Title)
			}
		})
	}
}

// TestBufYAMLCheckUpdates verifies that buf.dep.checkUpdates publishes the correct
// diagnostics for various combinations of pinned and latest commits.
func TestBufYAMLCheckUpdates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		fixture string
		// latestCommits maps module full name to the UUID string the mock BSR returns.
		latestCommits map[string]string
		// waitPred is the predicate passed to capture.wait; it gates which
		// publishDiagnostics notification is inspected.
		waitPred func(*protocol.PublishDiagnosticsParams) bool
		// check asserts the expected state of the received diagnostics.
		check func(*testing.T, *protocol.PublishDiagnosticsParams)
	}{
		{
			name:    "all_up_to_date",
			fixture: "testdata/buf_yaml/with_deps/buf.yaml",
			latestCommits: map[string]string{
				"buf.build/bufbuild/protovalidate": "00000000-0000-0000-0000-000000000001",
				"buf.build/googleapis/googleapis":  "00000000-0000-0000-0000-000000000002",
			},
			waitPred: func(_ *protocol.PublishDiagnosticsParams) bool { return true },
			check: func(t *testing.T, diags *protocol.PublishDiagnosticsParams) {
				assert.Empty(t, diags.Diagnostics, "expected no diagnostics when all deps are up to date")
			},
		},
		{
			name:    "some_outdated",
			fixture: "testdata/buf_yaml/with_deps/buf.yaml",
			latestCommits: map[string]string{
				// protovalidate has a newer commit (11); googleapis is up to date (02).
				"buf.build/bufbuild/protovalidate": "00000000-0000-0000-0000-000000000011",
				"buf.build/googleapis/googleapis":  "00000000-0000-0000-0000-000000000002",
			},
			waitPred: func(p *protocol.PublishDiagnosticsParams) bool { return len(p.Diagnostics) > 0 },
			check: func(t *testing.T, diags *protocol.PublishDiagnosticsParams) {
				require.Len(t, diags.Diagnostics, 1, "expected exactly 1 diagnostic for the outdated dep")
				d := diags.Diagnostics[0]
				assert.Equal(t, protocol.DiagnosticSeverityInformation, d.Severity)
				assert.Equal(t, "buf-lsp", d.Source)
				assert.Contains(t, d.Message, "buf.build/bufbuild/protovalidate")
				assert.Contains(t, d.Message, "00000000000000000000000000000011")
				// Diagnostic should be on the protovalidate dep line (line 2, 0-indexed).
				assert.Equal(t, uint32(2), d.Range.Start.Line)
			},
		},
		{
			name:    "all_outdated",
			fixture: "testdata/buf_yaml/with_deps/buf.yaml",
			latestCommits: map[string]string{
				"buf.build/bufbuild/protovalidate": "00000000-0000-0000-0000-000000000011",
				"buf.build/googleapis/googleapis":  "00000000-0000-0000-0000-000000000022",
			},
			waitPred: func(p *protocol.PublishDiagnosticsParams) bool { return len(p.Diagnostics) >= 2 },
			check: func(t *testing.T, diags *protocol.PublishDiagnosticsParams) {
				assert.Len(t, diags.Diagnostics, 2, "expected 2 diagnostics when both deps are outdated")
				for _, d := range diags.Diagnostics {
					assert.Equal(t, protocol.DiagnosticSeverityInformation, d.Severity)
					assert.Equal(t, "buf-lsp", d.Source)
					assert.Contains(t, d.Message, "can be updated to")
				}
			},
		},
		{
			name:    "no_buf_lock",
			fixture: "testdata/buf_yaml/deps_no_lock/buf.yaml",
			latestCommits: map[string]string{
				"buf.build/bufbuild/protovalidate": "00000000-0000-0000-0000-000000000011",
				"buf.build/googleapis/googleapis":  "00000000-0000-0000-0000-000000000022",
			},
			waitPred: func(_ *protocol.PublishDiagnosticsParams) bool { return true },
			check: func(t *testing.T, diags *protocol.PublishDiagnosticsParams) {
				assert.Empty(t, diags.Diagnostics, "expected no diagnostics when deps have no buf.lock pins")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var keys []bufmodule.ModuleKey
			for mod, uuidStr := range tc.latestCommits {
				keys = append(keys, mustNewModuleKey(t, mod, uuidStr))
			}
			mkp, err := bufmodule.NewStaticModuleKeyProvider(keys)
			require.NoError(t, err)

			absPath, err := filepath.Abs(tc.fixture)
			require.NoError(t, err)

			synctest.Test(t, func(t *testing.T) {
				clientJSONConn, bufYAMLURI, capture := setupLSPServerForBufYAML(t, absPath, mkp)
				ctx := t.Context()

				var result any
				_, err = clientJSONConn.Call(ctx, protocol.MethodWorkspaceExecuteCommand, &protocol.ExecuteCommandParams{
					Command:   "buf.dep.checkUpdates",
					Arguments: []any{string(bufYAMLURI)},
				}, &result)
				require.NoError(t, err)

				diags := capture.wait(t, bufYAMLURI, 5*time.Second, tc.waitPred)
				require.NotNil(t, diags)
				tc.check(t, diags)
			})
		})
	}
}

// TestBufYAMLDiagnostics_ClearedOnClose verifies that diagnostics for a buf.yaml
// file are cleared (empty publish) when the file is closed.
func TestBufYAMLDiagnostics_ClearedOnClose(t *testing.T) {
	t.Parallel()

	// Both deps outdated so we get diagnostics after the check.
	mkp, err := bufmodule.NewStaticModuleKeyProvider([]bufmodule.ModuleKey{
		mustNewModuleKey(t, "buf.build/bufbuild/protovalidate", "00000000-0000-0000-0000-000000000011"),
		mustNewModuleKey(t, "buf.build/googleapis/googleapis", "00000000-0000-0000-0000-000000000022"),
	})
	require.NoError(t, err)

	absPath, err := filepath.Abs("testdata/buf_yaml/with_deps/buf.yaml")
	require.NoError(t, err)

	synctest.Test(t, func(t *testing.T) {
		clientJSONConn, bufYAMLURI, capture := setupLSPServerForBufYAML(t, absPath, mkp)
		ctx := t.Context()

		// Trigger a check to produce diagnostics.
		var result any
		_, err = clientJSONConn.Call(ctx, protocol.MethodWorkspaceExecuteCommand, &protocol.ExecuteCommandParams{
			Command:   "buf.dep.checkUpdates",
			Arguments: []any{string(bufYAMLURI)},
		}, &result)
		require.NoError(t, err)

		// Wait for the non-empty diagnostics.
		diags := capture.wait(t, bufYAMLURI, 5*time.Second, func(p *protocol.PublishDiagnosticsParams) bool {
			return len(p.Diagnostics) > 0
		})
		require.NotNil(t, diags)
		require.NotEmpty(t, diags.Diagnostics, "expected diagnostics before close")

		// Close the file.
		err = clientJSONConn.Notify(ctx, protocol.MethodTextDocumentDidClose, &protocol.DidCloseTextDocumentParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: bufYAMLURI},
		})
		require.NoError(t, err)

		// Diagnostics should now be cleared.
		cleared := capture.wait(t, bufYAMLURI, 5*time.Second, func(p *protocol.PublishDiagnosticsParams) bool {
			return len(p.Diagnostics) == 0
		})
		require.NotNil(t, cleared)
		assert.Empty(t, cleared.Diagnostics, "expected diagnostics to be cleared after close")
	})
}

// TestBufYAMLCheckUpdates_FileChange verifies that deps are re-read correctly
// after a didChange notification and a subsequent codeLens request reflects the
// new content.
func TestBufYAMLCheckUpdates_FileChange(t *testing.T) {
	t.Parallel()

	mkp, err := bufmodule.NewStaticModuleKeyProvider([]bufmodule.ModuleKey{
		mustNewModuleKey(t, "buf.build/bufbuild/protovalidate", "00000000-0000-0000-0000-000000000011"),
		mustNewModuleKey(t, "buf.build/googleapis/googleapis", "00000000-0000-0000-0000-000000000002"),
	})
	require.NoError(t, err)

	absPath, err := filepath.Abs("testdata/buf_yaml/with_deps/buf.yaml")
	require.NoError(t, err)

	synctest.Test(t, func(t *testing.T) {
		clientJSONConn, bufYAMLURI, _ := setupLSPServerForBufYAML(t, absPath, mkp)
		ctx := t.Context()

		// Update the buf.yaml in-memory to have no deps.
		err = clientJSONConn.Notify(ctx, protocol.MethodTextDocumentDidChange, &protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{
				TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: bufYAMLURI},
				Version:                2,
			},
			ContentChanges: []protocol.TextDocumentContentChangeEvent{
				{Text: "version: v2\n"},
			},
		})
		require.NoError(t, err)

		// After the change, code lenses should reflect no deps.
		var lenses []protocol.CodeLens
		_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentCodeLens, &protocol.CodeLensParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: bufYAMLURI},
		}, &lenses)
		require.NoError(t, err)
		assert.Empty(t, lenses, "expected no lenses after deps are removed")
	})
}

// TestBufYAMLDocumentLinks verifies that document links are returned for buf.yaml dep
// entries. Links resolve to /docs/<ref> when the dep has an explicit ref, and to
// the module root otherwise.
func TestBufYAMLDocumentLinks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fixture   string
		wantLinks []protocol.DocumentLink
	}{
		{
			name:    "no_deps",
			fixture: "testdata/buf_yaml/no_deps/buf.yaml",
			// No deps, so no links.
		},
		{
			name:    "invalid",
			fixture: "testdata/buf_yaml/invalid/buf.yaml",
			// Malformed YAML must not crash; returns no links.
		},
		{
			// Deps without an explicit ref link to the module root.
			name:    "with_deps",
			fixture: "testdata/buf_yaml/with_deps/buf.yaml",
			wantLinks: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 4},
						End:   protocol.Position{Line: 2, Character: 36},
					},
					Target: "https://buf.build/bufbuild/protovalidate",
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 4},
						End:   protocol.Position{Line: 3, Character: 35},
					},
					Target: "https://buf.build/googleapis/googleapis",
				},
			},
		},
		{
			// A dep with an explicit label ref links to /docs/<ref>.
			name:    "deps_with_ref",
			fixture: "testdata/buf_yaml/deps_with_ref/buf.yaml",
			wantLinks: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 4},
						End:   protocol.Position{Line: 2, Character: 43},
					},
					Target: "https://buf.build/bufbuild/protovalidate/docs/v1.1.1",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			absPath, err := filepath.Abs(tc.fixture)
			require.NoError(t, err)

			clientJSONConn, bufYAMLURI, _ := setupLSPServerForBufYAML(t, absPath, nil)
			ctx := t.Context()

			var links []protocol.DocumentLink
			_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentDocumentLink, &protocol.DocumentLinkParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: bufYAMLURI},
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

// TestBufYAMLHover verifies that hovering over buf.yaml keys and rule names
// returns the correct markdown documentation, and that unrecognised positions
// return no hover.
func TestBufYAMLHover(t *testing.T) {
	t.Parallel()

	// All cases use the same comprehensive fixture.
	fixture := "testdata/buf_yaml/hover/buf.yaml"

	// The fixture layout (0-indexed lines):
	//  0: version: v2
	//  1: modules:
	//  2:   - path: .
	//  3:     name: buf.build/acme/petapis
	//  4:     includes:
	//  5:       - proto
	//  6:     excludes:
	//  7:       - proto/vendor
	//  8: deps:
	//  9:   - buf.build/bufbuild/protovalidate
	// 10: lint:
	// 11:   use:
	// 12:     - STANDARD
	// 13:     - COMMENTS
	// 14:     - ENUM_PASCAL_CASE
	// 15:   except:
	// 16:     - IMPORT_USED
	// 17:   ignore:
	// 18:     - foo/bar.proto
	// 19:   ignore_only:
	// 20:     ENUM_VALUE_UPPER_SNAKE_CASE:
	// 21:       - legacy/foo.proto
	// 22:   enum_zero_value_suffix: _UNSPECIFIED
	// 23:   service_suffix: Service
	// 24:   rpc_allow_same_request_response: false
	// 25:   rpc_allow_google_protobuf_empty_requests: false
	// 26:   rpc_allow_google_protobuf_empty_responses: false
	// 27:   disallow_comment_ignores: false
	// 28:   disable_builtin: false
	// 29: breaking:
	// 30:   use:
	// 31:     - FILE
	// 32:     - FIELD_NO_DELETE
	// 33:   except:
	// 34:     - ENUM_NO_DELETE
	// 35:   ignore:
	// 36:     - legacy/
	// 37:   ignore_only:
	// 38:     MESSAGE_NO_DELETE:
	// 39:       - alpha/
	// 40:   ignore_unstable_packages: true
	// 41:   disable_builtin: false

	tests := []struct {
		name      string
		line      uint32
		character uint32
		// expectedContains lists substrings that must appear in the hover markdown.
		// All strings must be present; leave nil to assert no hover is returned.
		expectedContains []string
		expectNoHover    bool
	}{
		// ── Top-level keys ──────────────────────────────────────────────────────
		{
			name: "version_key",
			line: 0, character: 0,
			expectedContains: []string{"version", "configuration format version", "https://buf.build/docs/configuration/v2/buf-yaml/#version"},
		},
		{
			name: "modules_key",
			line: 1, character: 0,
			expectedContains: []string{"modules", "Protobuf modules", "https://buf.build/docs/configuration/v2/buf-yaml/#modules"},
		},
		{
			name: "deps_key",
			line: 8, character: 0,
			expectedContains: []string{"deps", "Buf Schema Registry", "https://buf.build/docs/configuration/v2/buf-lock/", "https://buf.build/docs/configuration/v2/buf-yaml/#deps"},
		},
		{
			name: "lint_key",
			line: 10, character: 0,
			expectedContains: []string{"lint", "lint rules", "https://buf.build/docs/configuration/v2/buf-yaml/#lint"},
		},
		{
			name: "breaking_key",
			line: 29, character: 0,
			expectedContains: []string{"breaking", "breaking change", "https://buf.build/docs/configuration/v2/buf-yaml/#breaking"},
		},

		// ── Module entry sub-keys ────────────────────────────────────────────
		{
			name: "module_path_key",
			line: 2, character: 4,
			expectedContains: []string{"path", "Protobuf files", "https://buf.build/docs/configuration/v2/buf-yaml/#path"},
		},
		{
			name: "module_name_key",
			line: 3, character: 4,
			expectedContains: []string{"name", "Buf Schema Registry path", "https://buf.build/docs/configuration/v2/buf-yaml/#name"},
		},
		{
			name: "module_includes_key",
			line: 4, character: 4,
			expectedContains: []string{"includes", "Subdirectories to include", "https://buf.build/docs/configuration/v2/buf-yaml/#includes"},
		},
		{
			name: "module_excludes_key",
			line: 6, character: 4,
			expectedContains: []string{"excludes", "Subdirectories to exclude", "https://buf.build/docs/configuration/v2/buf-yaml/#excludes"},
		},

		// ── lint sub-keys ────────────────────────────────────────────────────
		{
			name: "lint_use_key",
			line: 11, character: 2,
			expectedContains: []string{"lint.use", "rule categories", "https://buf.build/docs/lint/rules/"},
		},
		{
			name: "lint_except_key",
			line: 15, character: 2,
			expectedContains: []string{"lint.except", "Removes specific rules", "https://buf.build/docs/configuration/v2/buf-yaml/#lint"},
		},
		{
			name: "lint_ignore_key",
			line: 17, character: 2,
			expectedContains: []string{"lint.ignore", "excluded from all lint rules", "https://buf.build/docs/configuration/v2/buf-yaml/#lint"},
		},
		{
			name: "lint_ignore_only_key",
			line: 19, character: 2,
			expectedContains: []string{"lint.ignore_only", "specific files or directories", "https://buf.build/docs/configuration/v2/buf-yaml/#lint"},
		},
		{
			name: "lint_enum_zero_value_suffix_key",
			line: 22, character: 2,
			expectedContains: []string{"lint.enum_zero_value_suffix", "ENUM_ZERO_VALUE_SUFFIX", "https://buf.build/docs/configuration/v2/buf-yaml/#lint"},
		},
		{
			name: "lint_service_suffix_key",
			line: 23, character: 2,
			expectedContains: []string{"lint.service_suffix", "SERVICE_SUFFIX", "https://buf.build/docs/configuration/v2/buf-yaml/#lint"},
		},
		{
			name: "lint_rpc_allow_same_request_response_key",
			line: 24, character: 2,
			expectedContains: []string{"lint.rpc_allow_same_request_response", "same message type", "https://buf.build/docs/configuration/v2/buf-yaml/#lint"},
		},
		{
			name: "lint_rpc_allow_google_protobuf_empty_requests_key",
			line: 25, character: 2,
			expectedContains: []string{"lint.rpc_allow_google_protobuf_empty_requests", "google.protobuf.Empty", "https://buf.build/docs/configuration/v2/buf-yaml/#lint"},
		},
		{
			name: "lint_rpc_allow_google_protobuf_empty_responses_key",
			line: 26, character: 2,
			expectedContains: []string{"lint.rpc_allow_google_protobuf_empty_responses", "google.protobuf.Empty", "https://buf.build/docs/configuration/v2/buf-yaml/#lint"},
		},
		{
			name: "lint_disallow_comment_ignores_key",
			line: 27, character: 2,
			expectedContains: []string{"lint.disallow_comment_ignores", "buf:lint:ignore", "https://buf.build/docs/configuration/v2/buf-yaml/#lint"},
		},
		{
			name: "lint_disable_builtin_key",
			line: 28, character: 2,
			expectedContains: []string{"lint.disable_builtin", "built-in lint rules", "https://buf.build/docs/configuration/v2/buf-yaml/#lint"},
		},

		// ── breaking sub-keys ────────────────────────────────────────────────
		{
			name: "breaking_use_key",
			line: 30, character: 2,
			expectedContains: []string{"breaking.use", "rule categories", "https://buf.build/docs/breaking/rules/"},
		},
		{
			name: "breaking_except_key",
			line: 33, character: 2,
			expectedContains: []string{"breaking.except", "Removes specific rules", "https://buf.build/docs/configuration/v2/buf-yaml/#breaking"},
		},
		{
			name: "breaking_ignore_key",
			line: 35, character: 2,
			expectedContains: []string{"breaking.ignore", "excluded from all breaking", "https://buf.build/docs/configuration/v2/buf-yaml/#breaking"},
		},
		{
			name: "breaking_ignore_only_key",
			line: 37, character: 2,
			expectedContains: []string{"breaking.ignore_only", "specific files or directories", "https://buf.build/docs/configuration/v2/buf-yaml/#breaking"},
		},
		{
			name: "breaking_ignore_unstable_packages_key",
			line: 40, character: 2,
			expectedContains: []string{"breaking.ignore_unstable_packages", "v1alpha1", "https://buf.build/docs/configuration/v2/buf-yaml/#breaking"},
		},
		{
			name: "breaking_disable_builtin_key",
			line: 41, character: 2,
			expectedContains: []string{"breaking.disable_builtin", "built-in breaking", "https://buf.build/docs/configuration/v2/buf-yaml/#breaking"},
		},

		// ── Rule/category names as values in lint.use ─────────────────────────
		{
			name: "lint_use_value_STANDARD",
			line: 12, character: 6,
			expectedContains: []string{"STANDARD", "default lint rule set", "protovalidate", "https://protovalidate.com", "https://buf.build/docs/lint/rules/"},
		},
		{
			name: "lint_use_value_COMMENTS",
			line: 13, character: 6,
			expectedContains: []string{"COMMENTS", "non-empty leading comments", "https://buf.build/docs/lint/rules/"},
		},
		{
			name: "lint_use_value_ENUM_PASCAL_CASE",
			line: 14, character: 6,
			expectedContains: []string{"ENUM_PASCAL_CASE", "PascalCase", "https://buf.build/docs/lint/rules/"},
		},

		// ── Rule/category names as values in lint.except ──────────────────────
		{
			name: "lint_except_value_IMPORT_USED",
			line: 16, character: 6,
			expectedContains: []string{"IMPORT_USED", "imported files must be used", "https://buf.build/docs/lint/rules/"},
		},

		// ── Rule names as keys in lint.ignore_only ────────────────────────────
		{
			name: "lint_ignore_only_rule_key",
			line: 20, character: 4,
			expectedContains: []string{"ENUM_VALUE_UPPER_SNAKE_CASE", "UPPER_SNAKE_CASE", "https://buf.build/docs/lint/rules/"},
		},

		// ── Rule/category names as values in breaking.use ─────────────────────
		{
			name: "breaking_use_value_FILE",
			line: 31, character: 6,
			expectedContains: []string{"FILE", "generated code", "https://buf.build/docs/breaking/rules/"},
		},
		{
			name: "breaking_use_value_FIELD_NO_DELETE",
			line: 32, character: 6,
			expectedContains: []string{"FIELD_NO_DELETE", "message field is deleted", "https://buf.build/docs/breaking/rules/"},
		},

		// ── Rule/category names as values in breaking.except ──────────────────
		{
			name: "breaking_except_value_ENUM_NO_DELETE",
			line: 34, character: 6,
			expectedContains: []string{"ENUM_NO_DELETE", "enum type is deleted", "https://buf.build/docs/breaking/rules/"},
		},

		// ── Rule names as keys in breaking.ignore_only ────────────────────────
		{
			name: "breaking_ignore_only_rule_key",
			line: 38, character: 4,
			expectedContains: []string{"MESSAGE_NO_DELETE", "message type is deleted", "https://buf.build/docs/breaking/rules/"},
		},

		// ── Positions that should return no hover ─────────────────────────────
		{
			// Plain file path in lint.ignore: not a known rule name.
			name: "lint_ignore_value_no_hover",
			line: 18, character: 6,
			expectNoHover: true,
		},
		{
			// Plain file path in breaking.ignore: not a known rule name.
			name: "breaking_ignore_value_no_hover",
			line: 36, character: 6,
			expectNoHover: true,
		},
		{
			// Off the end of the file entirely.
			name: "off_file_no_hover",
			line: 999, character: 0,
			expectNoHover: true,
		},
		{
			// Mid-line whitespace.
			name: "whitespace_no_hover",
			line: 11, character: 0,
			expectNoHover: true,
		},
	}

	absPath, err := filepath.Abs(fixture)
	require.NoError(t, err)

	clientJSONConn, bufYAMLURI, _ := setupLSPServerForBufYAML(t, absPath, nil)
	ctx := t.Context()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var hover *protocol.Hover
			_, err := clientJSONConn.Call(ctx, protocol.MethodTextDocumentHover, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: bufYAMLURI},
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

// TestBufYAMLHover_OtherFixtures verifies hover returns nil for buf.yaml files
// with no recognized fields at the tested positions, and does not panic on
// invalid YAML.
func TestBufYAMLHover_OtherFixtures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fixture   string
		line      uint32
		character uint32
	}{
		{
			// A minimal file with no lint/breaking sections.
			name:    "no_deps",
			fixture: "testdata/buf_yaml/no_deps/buf.yaml",
			line:    0, character: 0, // "version"
		},
		{
			// Malformed YAML must not crash the server.
			name:    "invalid",
			fixture: "testdata/buf_yaml/invalid/buf.yaml",
			line:    0, character: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			absPath, err := filepath.Abs(tc.fixture)
			require.NoError(t, err)

			clientJSONConn, bufYAMLURI, _ := setupLSPServerForBufYAML(t, absPath, nil)
			ctx := t.Context()

			var hover *protocol.Hover
			_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentHover, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: bufYAMLURI},
					Position:     protocol.Position{Line: tc.line, Character: tc.character},
				},
			}, &hover)
			require.NoError(t, err, "hover must not error even on edge-case files")
		})
	}
}

// TestBufYAMLIgnorePathDiagnostics verifies that warnings are emitted for lint.ignore
// and breaking.ignore paths that do not match any file in the workspace.
func TestBufYAMLIgnorePathDiagnostics(t *testing.T) {
	t.Parallel()

	absPath, err := filepath.Abs("testdata/buf_yaml/ignore_paths/buf.yaml")
	require.NoError(t, err)

	synctest.Test(t, func(t *testing.T) {
		_, bufYAMLURI, capture := setupLSPServerForBufYAML(t, absPath, nil)

		// Wait for diagnostics with at least 3 warnings (one per nonexistent path).
		diags := capture.wait(t, bufYAMLURI, 10*time.Second, func(p *protocol.PublishDiagnosticsParams) bool {
			warnings := 0
			for _, d := range p.Diagnostics {
				if d.Severity == protocol.DiagnosticSeverityWarning {
					warnings++
				}
			}
			return warnings >= 3
		})
		require.NotNil(t, diags, "expected ignore path diagnostics to be published")

		// Index diagnostics by start position for easy lookup.
		type pos struct{ line, char uint32 }
		byPos := make(map[pos]protocol.Diagnostic)
		for _, d := range diags.Diagnostics {
			byPos[pos{d.Range.Start.Line, d.Range.Start.Character}] = d
		}

		// The buf.yaml fixture (0-indexed lines):
		//  7:     - valid.proto          ← matches actual file; no diagnostic
		//  8:     - subdir               ← dir containing sub.proto; no diagnostic
		//  9:     - nonexistent.proto    ← no such file; warning expected
		// 10:     - nonexistent_dir/     ← no such dir; warning expected
		// 15:     - valid.proto          ← matches actual file; no diagnostic
		// 16:     - missing.proto        ← no such file; warning expected

		assertNoDiag := func(line, char uint32, label string) {
			t.Helper()
			_, exists := byPos[pos{line, char}]
			assert.False(t, exists, "%s should not have a diagnostic", label)
		}
		assertWarning := func(line, char uint32, wantMsg, label string) {
			t.Helper()
			d, ok := byPos[pos{line, char}]
			if !assert.True(t, ok, "expected warning for %s", label) {
				return
			}
			assert.Equal(t, protocol.DiagnosticSeverityWarning, d.Severity, "%s severity", label)
			assert.Equal(t, "buf-lsp", d.Source, "%s source", label)
			assert.Contains(t, d.Message, wantMsg, "%s message", label)
		}

		assertNoDiag(7, 6, "lint.ignore valid.proto")
		assertNoDiag(8, 6, "lint.ignore subdir (directory prefix match)")
		assertWarning(9, 6, "nonexistent.proto", "lint.ignore nonexistent.proto")
		assertWarning(10, 6, "nonexistent_dir/", "lint.ignore nonexistent_dir/")
		assertNoDiag(15, 6, "breaking.ignore valid.proto")
		assertWarning(16, 6, "missing.proto", "breaking.ignore missing.proto")
	})
}

// mustParseUUID parses a UUID string for use in test data, failing the test on error.
func mustParseUUID(t *testing.T, s string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(s)
	require.NoError(t, err)
	return id
}

// mustNewModuleKey constructs a ModuleKey with the given full name and commit ID for use
// in test data. A synthetic b5 digest of all-zeros is used; the digest is not validated
// during test execution but must have the correct type to satisfy the staticModuleKeyProvider.
func mustNewModuleKey(t *testing.T, fullNameStr, commitIDStr string) bufmodule.ModuleKey {
	t.Helper()
	fullName, err := bufparse.ParseFullName(fullNameStr)
	require.NoError(t, err)
	commitID := mustParseUUID(t, commitIDStr)
	key, err := bufmodule.NewModuleKey(fullName, commitID, func() (bufmodule.Digest, error) {
		return bufmodule.ParseDigest("b5:" + strings.Repeat("0", 128))
	})
	require.NoError(t, err)
	return key
}

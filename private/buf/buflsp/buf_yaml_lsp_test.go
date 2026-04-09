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
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
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

// mockModuleKeyProvider is a ModuleKeyProvider that returns fixed latest commits for testing.
type mockModuleKeyProvider struct {
	// latestByFullName maps module full name string ("registry/owner/name") to the
	// commit UUID that represents the "latest" version on the BSR.
	latestByFullName map[string]uuid.UUID
}

func (m *mockModuleKeyProvider) GetModuleKeysForModuleRefs(
	_ context.Context,
	refs []bufparse.Ref,
	_ bufmodule.DigestType,
) ([]bufmodule.ModuleKey, error) {
	keys := make([]bufmodule.ModuleKey, 0, len(refs))
	for _, ref := range refs {
		commitID, ok := m.latestByFullName[ref.FullName().String()]
		if !ok {
			return nil, fmt.Errorf("mockModuleKeyProvider: no mock commit for %s", ref.FullName())
		}
		key, err := bufmodule.NewModuleKey(
			ref.FullName(),
			commitID,
			func() (bufmodule.Digest, error) {
				return nil, fmt.Errorf("digest not available in mock")
			},
		)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

// setupLSPServerForBufYAML creates an LSP server initialized for buf.yaml testing.
// It opens the buf.yaml file at bufYAMLPath via didOpen and returns the client
// connection, the buf.yaml URI, and a diagnostics capture for async notifications.
//
// If mkp is non-nil it is injected as the ModuleKeyProvider for checkUpdates.
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
	appextContainer := appext.NewContainer(nameContainer, logger)

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

	var serveOptions []buflsp.ServeOption
	if mkp != nil {
		serveOptions = append(serveOptions, buflsp.WithModuleKeyProvider(mkp))
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
		serveOptions...,
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

			mkp := &mockModuleKeyProvider{latestByFullName: make(map[string]uuid.UUID, len(tc.latestCommits))}
			for mod, uuidStr := range tc.latestCommits {
				mkp.latestByFullName[mod] = mustParseUUID(t, uuidStr)
			}

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
	mkp := &mockModuleKeyProvider{
		latestByFullName: map[string]uuid.UUID{
			"buf.build/bufbuild/protovalidate": mustParseUUID(t, "00000000-0000-0000-0000-000000000011"),
			"buf.build/googleapis/googleapis":  mustParseUUID(t, "00000000-0000-0000-0000-000000000022"),
		},
	}

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

	mkp := &mockModuleKeyProvider{
		latestByFullName: map[string]uuid.UUID{
			"buf.build/bufbuild/protovalidate": mustParseUUID(t, "00000000-0000-0000-0000-000000000011"),
			"buf.build/googleapis/googleapis":  mustParseUUID(t, "00000000-0000-0000-0000-000000000002"),
		},
	}

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

// mustParseUUID parses a UUID string for use in test data, failing the test on error.
func mustParseUUID(t *testing.T, s string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(s)
	require.NoError(t, err)
	return id
}

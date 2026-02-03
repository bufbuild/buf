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

//go:build go1.25

package buflsp_test

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"buf.build/go/app"
	"buf.build/go/app/appext"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/buf/buflsp"
	"github.com/bufbuild/buf/private/buf/bufwkt/bufwktstore"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufpolicy"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/httpauth"
	"github.com/bufbuild/buf/private/pkg/slogtestext"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/wasm"
	"github.com/bufbuild/protocompile/experimental/incremental"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// setupLSPServerWithDiagnostics creates and initializes an LSP server for testing with diagnostic capture.
func setupLSPServerWithDiagnostics(
	t *testing.T,
	testProtoPath string,
) (jsonrpc2.Conn, protocol.URI, *diagnosticsCapture) {
	t.Helper()

	ctx := t.Context()

	logger := slogtestext.NewLogger(t, slogtestext.WithLogLevel(appext.LogLevelInfo))

	appContainer, err := app.NewContainerForOS()
	require.NoError(t, err)

	nameContainer, err := appext.NewNameContainer(appContainer, "buf-test")
	require.NoError(t, err)
	appextContainer := appext.NewContainer(nameContainer, logger)

	graphProvider := bufmodule.NopGraphProvider
	moduleDataProvider := bufmodule.NopModuleDataProvider
	commitProvider := bufmodule.NopCommitProvider
	pluginKeyProvider := bufplugin.NopPluginKeyProvider
	pluginDataProvider := bufplugin.NopPluginDataProvider
	policyKeyProvider := bufpolicy.NopPolicyKeyProvider
	policyDataProvider := bufpolicy.NopPolicyDataProvider

	tmpDir := t.TempDir()
	storageBucket, err := storageos.NewProvider().NewReadWriteBucket(tmpDir)
	require.NoError(t, err)

	wktStore := bufwktstore.NewStore(logger, storageBucket)

	controller, err := bufctl.NewController(
		logger,
		appContainer,
		graphProvider,
		nopModuleKeyProvider{},
		moduleDataProvider,
		commitProvider,
		pluginKeyProvider,
		pluginDataProvider,
		policyKeyProvider,
		policyDataProvider,
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

	stream := jsonrpc2.NewStream(serverConn)

	conn, err := buflsp.Serve(
		ctx,
		"test",
		wktBucket,
		appextContainer,
		controller,
		wasmRuntime,
		stream,
		queryExecutor,
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

	testWorkspaceDir := filepath.Dir(testProtoPath)
	testURI := uri.New(testProtoPath)
	var initResult protocol.InitializeResult
	_, initErr := clientJSONConn.Call(ctx, protocol.MethodInitialize, &protocol.InitializeParams{
		RootURI: uri.New(testWorkspaceDir),
		Capabilities: protocol.ClientCapabilities{
			TextDocument: &protocol.TextDocumentClientCapabilities{},
		},
	}, &initResult)
	require.NoError(t, initErr)

	err = clientJSONConn.Notify(ctx, protocol.MethodInitialized, &protocol.InitializedParams{})
	require.NoError(t, err)

	testProtoContent, err := os.ReadFile(testProtoPath)
	require.NoError(t, err)

	err = clientJSONConn.Notify(ctx, protocol.MethodTextDocumentDidOpen, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        testURI,
			LanguageID: "protobuf",
			Version:    1,
			Text:       string(testProtoContent),
		},
	})
	require.NoError(t, err)

	return clientJSONConn, testURI, capture
}

// TestDiagnostics tests various diagnostic scenarios published by the LSP server.
// Each subtest uses synctest to provide deterministic timing for async diagnostics.
func TestDiagnostics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		protoFile           string
		expectedDiagnostics []protocol.Diagnostic
	}{
		{
			name:                "valid_proto_no_diagnostics",
			protoFile:           "testdata/diagnostics/valid.proto",
			expectedDiagnostics: []protocol.Diagnostic{},
		},
		{
			name:      "syntax_error_diagnostic",
			protoFile: "testdata/diagnostics/syntax_error.proto",
			expectedDiagnostics: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 8, Character: 0},
						End:   protocol.Position{Line: 8, Character: 0},
					},
					Severity: protocol.DiagnosticSeverityError,
					Source:   "buf-lsp",
					Message:  "syntax error: expecting ';'",
				},
			},
		},
		{
			name:      "unused_import_diagnostic_with_tag",
			protoFile: "testdata/diagnostics/unused_import.proto",
			expectedDiagnostics: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 4, Character: 0},
						End:   protocol.Position{Line: 4, Character: 41},
					},
					Severity: protocol.DiagnosticSeverityWarning,
					Code:     "IMPORT_USED",
					CodeDescription: &protocol.CodeDescription{
						Href: "https://buf.build/docs/lint/rules/#import_used",
					},
					Source:  "buf lint",
					Message: `Import "google/protobuf/timestamp.proto" is unused.`,
					Tags:    []protocol.DiagnosticTag{protocol.DiagnosticTagUnnecessary},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				protoPath, err := filepath.Abs(tt.protoFile)
				require.NoError(t, err)

				_, testURI, capture := setupLSPServerWithDiagnostics(t, protoPath)

				// Wait for diagnostics to be published
				timeout := 5 * time.Second
				if len(tt.expectedDiagnostics) > 0 {
					timeout = 10 * time.Second // Lint checks take longer
				}
				diagnostics := capture.wait(t, testURI, timeout, func(p *protocol.PublishDiagnosticsParams) bool {
					return len(p.Diagnostics) >= len(tt.expectedDiagnostics)
				})

				require.NotNil(t, diagnostics, "expected diagnostics to be published")
				assert.Equal(t, testURI, diagnostics.URI)

				// Check that we have the expected number of diagnostics
				require.Len(t, diagnostics.Diagnostics, len(tt.expectedDiagnostics),
					"expected %d diagnostic(s), got %d", len(tt.expectedDiagnostics), len(diagnostics.Diagnostics))

				// Compare each diagnostic directly
				for i, expected := range tt.expectedDiagnostics {
					actual := diagnostics.Diagnostics[i]
					assert.Equal(t, expected, actual, "diagnostic %d mismatch", i)
				}
			})
		})
	}
}

// TestDiagnosticsUpdate tests that diagnostics are updated when file content changes.
// Uses synctest to provide deterministic timing for the async diagnostic updates.
func TestDiagnosticsUpdate(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		protoPath, err := filepath.Abs("testdata/diagnostics/valid.proto")
		require.NoError(t, err)

		clientJSONConn, testURI, capture := setupLSPServerWithDiagnostics(t, protoPath)

		ctx := t.Context()

		// Wait for initial diagnostics (should be empty for valid file)
		initialDiagnostics := capture.wait(t, testURI, 5*time.Second, func(p *protocol.PublishDiagnosticsParams) bool {
			return true // Accept any diagnostics
		})
		require.NotNil(t, initialDiagnostics)
		assert.Empty(t, initialDiagnostics.Diagnostics, "expected no initial diagnostics for valid file")

		// Update the file with invalid content (missing semicolon)
		invalidContent := `syntax = "proto3";

package diagnostics.v1;

message TestMessage {
  string name = 1
  // Missing semicolon above
}
`

		err = clientJSONConn.Notify(ctx, protocol.MethodTextDocumentDidChange, &protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{
				TextDocumentIdentifier: protocol.TextDocumentIdentifier{
					URI: testURI,
				},
				Version: 2,
			},
			ContentChanges: []protocol.TextDocumentContentChangeEvent{
				{
					Text: invalidContent,
				},
			},
		})
		require.NoError(t, err)

		// Wait for updated diagnostics with version 2 and at least one error
		updatedDiagnostics := capture.wait(t, testURI, 5*time.Second, func(p *protocol.PublishDiagnosticsParams) bool {
			return p.Version == 2 && len(p.Diagnostics) > 0
		})
		require.NotNil(t, updatedDiagnostics)
		assert.Equal(t, uint32(2), updatedDiagnostics.Version, "expected diagnostics version to match file version")
		assert.NotEmpty(t, updatedDiagnostics.Diagnostics, "expected diagnostics after introducing syntax error")

		if len(updatedDiagnostics.Diagnostics) > 0 {
			assert.Equal(t, protocol.DiagnosticSeverityError, updatedDiagnostics.Diagnostics[0].Severity)
		}
	})
}

// diagnosticsCapture captures publishDiagnostics notifications from the LSP server.
type diagnosticsCapture struct {
	mu          sync.Mutex
	diagnostics map[protocol.URI]*protocol.PublishDiagnosticsParams
}

func newDiagnosticsCapture() *diagnosticsCapture {
	return &diagnosticsCapture{
		diagnostics: make(map[protocol.URI]*protocol.PublishDiagnosticsParams),
	}
}

func (dc *diagnosticsCapture) handle(_ context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	if req.Method() == protocol.MethodTextDocumentPublishDiagnostics {
		var params protocol.PublishDiagnosticsParams
		if err := json.Unmarshal(req.Params(), &params); err == nil {
			dc.mu.Lock()
			dc.diagnostics[params.URI] = &params
			dc.mu.Unlock()
		}
	}
	return reply(context.Background(), nil, nil)
}

// wait polls for diagnostics matching the predicate.
func (dc *diagnosticsCapture) wait(t *testing.T, uri protocol.URI, timeout time.Duration, pred func(*protocol.PublishDiagnosticsParams) bool) *protocol.PublishDiagnosticsParams {
	t.Helper()

	require.Eventually(t, func() bool {
		dc.mu.Lock()
		params := dc.diagnostics[uri]
		dc.mu.Unlock()
		return params != nil && pred(params)
	}, timeout, 50*time.Millisecond, "timeout waiting for diagnostics matching predicate")

	dc.mu.Lock()
	defer dc.mu.Unlock()
	return dc.diagnostics[uri]
}

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
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// nopModuleKeyProvider is a no-op implementation of ModuleKeyProvider for testing
type nopModuleKeyProvider struct{}

func (nopModuleKeyProvider) GetModuleKeysForModuleRefs(context.Context, []bufparse.Ref, bufmodule.DigestType) ([]bufmodule.ModuleKey, error) {
	return nil, os.ErrNotExist
}

// setupLSPServer creates and initializes an LSP server for testing.
// Returns the client JSON-RPC connection and the test file URI.
func setupLSPServer(
	t *testing.T,
	testProtoPath string,
) (jsonrpc2.Conn, protocol.URI) {
	t.Helper()

	ctx := t.Context()

	logger := slogtestext.NewLogger(t)

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

	go func() {
		conn, err := buflsp.Serve(
			ctx,
			wktBucket,
			appextContainer,
			controller,
			wasmRuntime,
			stream,
			queryExecutor,
		)
		if err != nil {
			t.Errorf("Failed to start server: %v", err)
			return
		}
		t.Cleanup(func() {
			require.NoError(t, conn.Close())
		})
		<-ctx.Done()
	}()

	clientStream := jsonrpc2.NewStream(clientConn)
	clientJSONConn := jsonrpc2.NewConn(clientStream)
	clientJSONConn.Go(ctx, jsonrpc2.AsyncHandler(func(_ context.Context, reply jsonrpc2.Replier, _ jsonrpc2.Request) error {
		return reply(ctx, nil, nil)
	}))
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
	assert.True(t, initResult.Capabilities.HoverProvider != nil)

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

	return clientJSONConn, testURI
}

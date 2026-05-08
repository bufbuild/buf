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
	"encoding/json"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

const (
	methodTextDocumentInlayHint     = "textDocument/inlayHint"
	methodWorkspaceInlayHintRefresh = "workspace/inlayHint/refresh"
)

// inlayHintWire mirrors the on-the-wire shape used by the server. Defined here
// because the protocol library predates LSP 3.17.
type inlayHintWire struct {
	Position protocol.Position `json:"position"`
	Label    string            `json:"label"`
}

// inlayHintParamsWire mirrors the request parameter shape on the wire.
type inlayHintParamsWire struct {
	TextDocument protocol.TextDocumentIdentifier `json:"textDocument"`
	Range        protocol.Range                  `json:"range"`
}

// fakePluginVersionProvider returns predetermined latest versions for plugins
// and optionally fails with err. Calls() returns the number of times the
// provider was invoked.
type fakePluginVersionProvider struct {
	versionsByFullName map[string]string
	err                error
	calls              atomic.Int32
}

func (f *fakePluginVersionProvider) GetLatestVersion(_ context.Context, registry, owner, plugin string) (string, error) {
	f.calls.Add(1)
	if f.err != nil {
		return "", f.err
	}
	return f.versionsByFullName[registry+"/"+owner+"/"+plugin], nil
}

func (f *fakePluginVersionProvider) Calls() int32 { return f.calls.Load() }

// fakeModuleKeyProvider returns predetermined commits for module refs and
// optionally fails with err. Calls() returns the number of times the provider
// was invoked.
type fakeModuleKeyProvider struct {
	commitsByFullName map[string]uuid.UUID
	err               error
	calls             atomic.Int32
}

func (f *fakeModuleKeyProvider) GetModuleKeysForModuleRefs(
	_ context.Context,
	refs []bufparse.Ref,
	_ bufmodule.DigestType,
) ([]bufmodule.ModuleKey, error) {
	f.calls.Add(1)
	if f.err != nil {
		return nil, f.err
	}
	keys := make([]bufmodule.ModuleKey, 0, len(refs))
	for _, ref := range refs {
		commit, ok := f.commitsByFullName[ref.FullName().String()]
		if !ok {
			continue
		}
		key, err := bufmodule.NewModuleKey(ref.FullName(), commit, func() (bufmodule.Digest, error) {
			return nil, errors.New("digest not needed for tests")
		})
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

func (f *fakeModuleKeyProvider) Calls() int32 { return f.calls.Load() }

// requestInlayHints sends a textDocument/inlayHint request and returns the
// hints. The server uses our LSP 3.17 backport types; we round-trip via JSON
// because the lsp library doesn't know the method.
func requestInlayHints(
	ctx context.Context,
	t *testing.T,
	conn jsonrpc2.Conn,
	uri protocol.URI,
) []inlayHintWire {
	t.Helper()
	params := inlayHintParamsWire{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	}
	var raw json.RawMessage
	_, err := conn.Call(ctx, methodTextDocumentInlayHint, &params, &raw)
	require.NoError(t, err)
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var hints []inlayHintWire
	require.NoError(t, json.Unmarshal(raw, &hints))
	return hints
}

// TestBufLockInlayHints_BehindHint verifies that when the BSR reports a newer
// commit than the buf.lock pin, an inlay hint is emitted next to the commit
// value once the cache is populated.
func TestBufLockInlayHints_BehindHint(t *testing.T) {
	t.Parallel()

	// Pinned commits in the fixture buf.lock:
	//   buf.build/bufbuild/protovalidate -> 00000000000000000000000000000001
	//   buf.build/googleapis/googleapis  -> 00000000000000000000000000000002
	// The fake provider returns *different* commits, so both should produce hints.
	provider := &fakeModuleKeyProvider{
		commitsByFullName: map[string]uuid.UUID{
			"buf.build/bufbuild/protovalidate": uuid.MustParse("00000000-0000-0000-0000-0000000000aa"),
			"buf.build/googleapis/googleapis":  uuid.MustParse("00000000-0000-0000-0000-0000000000bb"),
		},
	}

	conn, hints := openLockAndAwaitHints(t, "testdata/buf_lock/inlay_hints/buf.lock", provider, 5*time.Second)
	defer conn.Close() // closed by setup cleanup; double-close is fine

	require.Len(t, hints, 2, "expected one hint per pinned dep when both are behind")
	assert.Equal(t, " → 000000000000000000000000000000aa", hints[0].Label)
	assert.Equal(t, " → 000000000000000000000000000000bb", hints[1].Label)
}

// TestBufLockInlayHints_UpToDate verifies that no hints are emitted when the
// BSR reports the same commits as the buf.lock pins.
func TestBufLockInlayHints_UpToDate(t *testing.T) {
	t.Parallel()

	provider := &fakeModuleKeyProvider{
		commitsByFullName: map[string]uuid.UUID{
			"buf.build/bufbuild/protovalidate": uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			"buf.build/googleapis/googleapis":  uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		},
	}

	_, hints := openLockAndAwaitHints(t, "testdata/buf_lock/inlay_hints/buf.lock", provider, 5*time.Second)
	assert.Empty(t, hints, "no hints expected when everything is up to date")
}

// TestBufLockInlayHints_NetworkError verifies graceful degradation when the
// BSR is unreachable: no hints are emitted, no diagnostics are published, and
// no inlay hint refresh notification is sent. The user must not see errors —
// the feature simply produces nothing.
func TestBufLockInlayHints_NetworkError(t *testing.T) {
	t.Parallel()

	provider := &fakeModuleKeyProvider{err: errors.New("dial tcp: connection refused")}

	conn, uri, capture := setupLockTest(t, "testdata/buf_lock/inlay_hints/buf.lock", provider)

	hints := requestInlayHints(t.Context(), t, conn, uri)
	assert.Empty(t, hints, "first request: cache miss, returns nothing")

	// The async fetch happens after the request returns. Wait for the provider
	// to be invoked, then confirm no refresh was sent.
	require.Eventually(t, func() bool {
		return provider.Calls() > 0
	}, 5*time.Second, 50*time.Millisecond, "provider should have been called at least once")
	time.Sleep(200 * time.Millisecond) // let any pending notifications drain

	assert.Equal(t, 0, capture.methodCount(methodWorkspaceInlayHintRefresh),
		"workspace/inlayHint/refresh must not fire when the BSR fetch fails")

	// A second request, post-error, must still be safe and silent.
	hints = requestInlayHints(t.Context(), t, conn, uri)
	assert.Empty(t, hints)

	// And no diagnostics — inlay hint failures must not surface to the user.
	capture.mu.Lock()
	defer capture.mu.Unlock()
	if got := capture.diagnostics[uri]; got != nil {
		assert.Empty(t, got.Diagnostics, "no diagnostics should be published for inlay hint failures")
	}
}

// TestBufLockInlayHints_AuthError verifies graceful degradation specifically
// for auth/permission failures (private BSR with bad netrc). Behavior is
// identical to the network error case: silent, no hints, no diagnostics.
func TestBufLockInlayHints_AuthError(t *testing.T) {
	t.Parallel()

	provider := &fakeModuleKeyProvider{err: errors.New("unauthenticated: invalid credentials")}

	conn, uri, capture := setupLockTest(t, "testdata/buf_lock/inlay_hints/buf.lock", provider)

	hints := requestInlayHints(t.Context(), t, conn, uri)
	assert.Empty(t, hints)

	require.Eventually(t, func() bool {
		return provider.Calls() > 0
	}, 5*time.Second, 50*time.Millisecond)
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, 0, capture.methodCount(methodWorkspaceInlayHintRefresh),
		"workspace/inlayHint/refresh must not fire on auth failure")

	capture.mu.Lock()
	defer capture.mu.Unlock()
	if got := capture.diagnostics[uri]; got != nil {
		assert.Empty(t, got.Diagnostics, "no diagnostics should be published for auth failures")
	}
}

// setupLockTest opens the buf.lock at fixture and returns the connection,
// URI, and diagnostics capture. The capture also tracks method-level
// notification counters used to assert on workspace/inlayHint/refresh.
func setupLockTest(
	t *testing.T,
	fixture string,
	provider bufmodule.ModuleKeyProvider,
) (jsonrpc2.Conn, protocol.URI, *diagnosticsCapture) {
	t.Helper()

	absPath, err := filepath.Abs(fixture)
	require.NoError(t, err)

	conn, uri, capture := setupLSPServerForBufYAML(t, absPath, provider, nil)
	return conn, uri, capture
}

// openLockAndAwaitHints opens a buf.lock, sends the first inlay hint request
// (which triggers the async fetch), waits for the workspace/inlayHint/refresh
// notification, then sends a second request and returns the hints. Returns the
// connection so callers can perform additional assertions.
func openLockAndAwaitHints(
	t *testing.T,
	fixture string,
	provider *fakeModuleKeyProvider,
	timeout time.Duration,
) (jsonrpc2.Conn, []inlayHintWire) {
	t.Helper()

	conn, uri, capture := setupLockTest(t, fixture, provider)

	// First request: cache miss, kicks off async fetch.
	first := requestInlayHints(t.Context(), t, conn, uri)
	assert.Empty(t, first, "first request should return no hints (cache miss)")

	// Wait for the refresh notification, which signals the cache populated.
	require.Eventually(t, func() bool {
		return capture.methodCount(methodWorkspaceInlayHintRefresh) > 0
	}, timeout, 50*time.Millisecond, "expected workspace/inlayHint/refresh after async fetch")

	// Second request: cache hit, returns hints (or nothing if up-to-date).
	return conn, requestInlayHints(t.Context(), t, conn, uri)
}

// TestBufYAMLInlayHints_BehindHint verifies that buf.yaml emits inlay hints
// next to deps whose buf.lock-pinned commit is behind the BSR's latest commit.
// The hint sits at the end of the dep line, not the lock entry — so users see
// "drift" right where the dep is declared.
func TestBufYAMLInlayHints_BehindHint(t *testing.T) {
	t.Parallel()

	// Fixture buf.lock pins:
	//   buf.build/bufbuild/protovalidate -> 00000000000000000000000000000001
	//   buf.build/googleapis/googleapis  -> 00000000000000000000000000000002
	provider := &fakeModuleKeyProvider{
		commitsByFullName: map[string]uuid.UUID{
			"buf.build/bufbuild/protovalidate": uuid.MustParse("00000000-0000-0000-0000-0000000000aa"),
			"buf.build/googleapis/googleapis":  uuid.MustParse("00000000-0000-0000-0000-000000000002"), // up-to-date
		},
	}

	absPath, err := filepath.Abs("testdata/buf_lock/inlay_hints/buf.yaml")
	require.NoError(t, err)
	conn, uri, capture := setupLSPServerForBufYAML(t, absPath, provider, nil)
	hints := awaitHints(t, conn, uri, capture, 5*time.Second)

	require.Len(t, hints, 1, "expected one hint for the single behind dep")
	assert.Equal(t, " → 000000000000000000000000000000aa", hints[0].Label)
}

// TestBufGenYAMLInlayHints_BehindHint verifies that buf.gen.yaml emits inlay
// hints next to versioned remote plugins whose pinned version is behind the
// latest BSR-published version.
func TestBufGenYAMLInlayHints_BehindHint(t *testing.T) {
	t.Parallel()

	// Fixture pins buf.build/bufbuild/es:v2.10.0 — fake provider returns v2.11.0.
	provider := &fakePluginVersionProvider{
		versionsByFullName: map[string]string{
			"buf.build/bufbuild/es": "v2.11.0",
		},
	}

	absPath, err := filepath.Abs("testdata/buf_gen_yaml/with_versioned_plugins/buf.gen.yaml")
	require.NoError(t, err)
	conn, uri, capture := setupLSPServerForBufYAML(t, absPath, nil, provider)
	hints := awaitHints(t, conn, uri, capture, 5*time.Second)

	require.Len(t, hints, 1, "expected exactly one hint for the versioned plugin")
	assert.Equal(t, " → v2.11.0", hints[0].Label)
}

// TestBufGenYAMLInlayHints_NetworkError verifies graceful degradation: when
// the plugin version provider fails, no hints are emitted, no refresh is
// triggered, and no diagnostics are published.
func TestBufGenYAMLInlayHints_NetworkError(t *testing.T) {
	t.Parallel()

	provider := &fakePluginVersionProvider{err: errors.New("dial tcp: connection refused")}

	absPath, err := filepath.Abs("testdata/buf_gen_yaml/with_versioned_plugins/buf.gen.yaml")
	require.NoError(t, err)
	conn, uri, capture := setupLSPServerForBufYAML(t, absPath, nil, provider)

	hints := requestInlayHints(t.Context(), t, conn, uri)
	assert.Empty(t, hints)

	require.Eventually(t, func() bool { return provider.Calls() > 0 },
		5*time.Second, 50*time.Millisecond)
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, 0, capture.methodCount(methodWorkspaceInlayHintRefresh),
		"workspace/inlayHint/refresh must not fire on plugin fetch failure")
	capture.mu.Lock()
	defer capture.mu.Unlock()
	if got := capture.diagnostics[uri]; got != nil {
		assert.Empty(t, got.Diagnostics, "no diagnostics for plugin fetch failures")
	}
}

// TestBufPolicyYAMLInlayHints_BehindHint verifies that buf.policy.yaml emits
// inlay hints next to versioned plugin entries whose pinned version is behind.
func TestBufPolicyYAMLInlayHints_BehindHint(t *testing.T) {
	t.Parallel()

	// Fixture pins buf.build/acme/my-lint-plugin:v1.2.0; provider returns v1.3.0.
	provider := &fakePluginVersionProvider{
		versionsByFullName: map[string]string{
			"buf.build/acme/my-lint-plugin": "v1.3.0",
		},
	}

	absPath, err := filepath.Abs("testdata/buf_policy_yaml/inlay_hints/buf.policy.yaml")
	require.NoError(t, err)
	conn, uri, capture := setupLSPServerForBufYAML(t, absPath, nil, provider)
	hints := awaitHints(t, conn, uri, capture, 5*time.Second)

	require.Len(t, hints, 1)
	assert.Equal(t, " → v1.3.0", hints[0].Label)
}

// awaitHints sends the first inlay hint request to trigger an async fetch,
// waits for the resulting workspace/inlayHint/refresh, and returns the hints
// from a follow-up request. Generic over file type — works for any URI the
// server responds to inlay hint requests for.
func awaitHints(
	t *testing.T,
	conn jsonrpc2.Conn,
	uri protocol.URI,
	capture *diagnosticsCapture,
	timeout time.Duration,
) []inlayHintWire {
	t.Helper()
	first := requestInlayHints(t.Context(), t, conn, uri)
	assert.Empty(t, first, "first request should return no hints (cache miss)")
	require.Eventually(t, func() bool {
		return capture.methodCount(methodWorkspaceInlayHintRefresh) > 0
	}, timeout, 50*time.Millisecond, "expected workspace/inlayHint/refresh after async fetch")
	return requestInlayHints(t.Context(), t, conn, uri)
}

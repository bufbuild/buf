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

// This file backports a minimal slice of the LSP 3.17 inlay hint surface so the
// server can emit virtual text alongside dep/plugin versions. The protocol
// library at go.lsp.dev/protocol@v0.12.0 predates LSP 3.17 and does not include
// these types or methods.

package buflsp

import (
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginref"
	"go.lsp.dev/protocol"
)

// methodTextDocumentInlayHint is the LSP 3.17 request method for textDocument/inlayHint.
const methodTextDocumentInlayHint = "textDocument/inlayHint"

// methodWorkspaceInlayHintRefresh is the LSP 3.17 server-to-client notification
// asking the client to invalidate any cached inlay hints. Sent after the
// version cache populates so the editor re-requests hints with fresh data.
const methodWorkspaceInlayHintRefresh = "workspace/inlayHint/refresh"

// inlayHintParams is the request parameter shape for textDocument/inlayHint.
type inlayHintParams struct {
	TextDocument protocol.TextDocumentIdentifier `json:"textDocument"`
	Range        protocol.Range                  `json:"range"`
}

// inlayHint is a single hint rendered at a position by the editor.
type inlayHint struct {
	Position     protocol.Position `json:"position"`
	Label        string            `json:"label"`
	PaddingLeft  bool              `json:"paddingLeft,omitempty"`
	PaddingRight bool              `json:"paddingRight,omitempty"`
}

// inlayHintOptions advertises inlay hint support in the server capabilities.
// Empty for now; resolveProvider/workDoneProgress are not used.
type inlayHintOptions struct{}

// extendedServerCapabilities embeds protocol.ServerCapabilities and adds the
// fields the protocol library is missing. JSON marshaling flattens the
// embedded struct, so the shape on the wire is identical to the spec.
type extendedServerCapabilities struct {
	protocol.ServerCapabilities

	InlayHintProvider *inlayHintOptions `json:"inlayHintProvider,omitempty"`
}

// extendedInitializeResult mirrors protocol.InitializeResult but uses the
// extended capabilities type.
type extendedInitializeResult struct {
	Capabilities extendedServerCapabilities `json:"capabilities"`
	ServerInfo   *protocol.ServerInfo       `json:"serverInfo,omitempty"`
}

// pluginInlayHintsForRefs builds inlay hints for plugin refs whose pinned
// version is behind the cached latest version. Refs whose latest version is
// not yet cached are returned in missingRefs so the caller can schedule a
// background fetch.
//
// Each ref's value is expected to be of the form "registry/owner/plugin:version".
// Refs that fail to parse, or that lack a pinned version, are silently skipped.
func pluginInlayHintsForRefs(refs []bsrRef, cache *versionCache) (hints []inlayHint, missingRefs []bsrRef) {
	for _, entry := range refs {
		identity, pinnedVersion, err := bufremotepluginref.ParsePluginIdentityOptionalVersion(entry.ref)
		if err != nil || pinnedVersion == "" {
			continue
		}
		fullName := identity.IdentityString()
		latest, cached := cache.GetPluginVersion(fullName)
		if !cached {
			missingRefs = append(missingRefs, entry)
			continue
		}
		if latest == pinnedVersion {
			continue
		}
		hints = append(hints, inlayHint{
			Position:    entry.refRange.End,
			Label:       fmt.Sprintf(" → %s", latest),
			PaddingLeft: true,
		})
	}
	return hints, missingRefs
}

// fetchPluginVersionsAndRefresh resolves latest versions for the given plugin
// refs via the curated plugin version provider and, if at least one new entry
// was cached, asks the client to refresh inlay hints. Each ref is fetched
// independently; failures are logged and otherwise ignored so a transient
// error for one plugin does not block the rest. Runs on the connection
// context so it does not outlive the LSP session.
func fetchPluginVersionsAndRefresh(l *lsp, refs []bsrRef) {
	populated := false
	for _, entry := range refs {
		identity, _, err := bufremotepluginref.ParsePluginIdentityOptionalVersion(entry.ref)
		if err != nil {
			continue
		}
		if l.versionCache.FetchPluginVersion(
			l.connCtx, l.curatedPluginVersionProvider,
			identity.Remote(), identity.Owner(), identity.Plugin(),
		) {
			populated = true
		}
	}
	if !populated {
		return
	}
	_ = l.conn.Notify(l.connCtx, methodWorkspaceInlayHintRefresh, nil)
}

// fetchModuleCommitsAndRefresh resolves latest commits for refs via the
// module key provider and, on a populated success, asks the client to refresh
// inlay hints. Provider errors are logged in the cache and otherwise ignored.
// Runs on the connection context so it does not outlive the LSP session.
func fetchModuleCommitsAndRefresh(l *lsp, refs []bufparse.Ref, digestType bufmodule.DigestType) {
	populated := l.versionCache.FetchModuleCommits(
		l.connCtx, l.moduleKeyProvider, refs, digestType,
	)
	if !populated {
		return
	}
	_ = l.conn.Notify(l.connCtx, methodWorkspaceInlayHintRefresh, nil)
}

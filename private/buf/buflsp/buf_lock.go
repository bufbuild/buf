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
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"go.lsp.dev/protocol"
	"gopkg.in/yaml.v3"
)

// isBufLockURI reports whether uri refers to a buf.lock file.
func isBufLockURI(uri protocol.URI) bool {
	return filepath.Base(uri.Filename()) == bufconfig.DefaultBufLockFileName
}

// bufLockManager tracks open buf.lock files in the LSP session.
type bufLockManager struct {
	lsp       *lsp
	mu        sync.Mutex
	uriToFile map[protocol.URI]*bufLockFile
}

func newBufLockManager(lsp *lsp) *bufLockManager {
	return &bufLockManager{
		lsp:       lsp,
		uriToFile: make(map[protocol.URI]*bufLockFile),
	}
}

// bufLockFile holds the parsed state of an open buf.lock file.
type bufLockFile struct {
	docNode *yaml.Node   // parsed YAML document node, nil if parse failed
	deps    []bsrRef     // deps[*].name BSR module references
	commits []bufLockPin // deps[*].commit values, paired with the matching dep's full name
}

// bufLockPin captures a deps[*].commit scalar with its source position and the
// full name of the module it pins. Used by inlay hints to compare the pinned
// commit against the latest commit returned by the BSR.
type bufLockPin struct {
	fullName    string         // module full name, e.g. "buf.build/foo/bar"
	commit      string         // dashless commit string as written in the file
	commitRange protocol.Range // position of the commit scalar
}

// Track opens or refreshes a buf.lock file.
func (m *bufLockManager) Track(uri protocol.URI, text string) {
	normalized := normalizeURI(uri)
	docNode := parseYAMLDoc(text)
	deps, commits := parseBufLockDepsAndCommits(docNode)
	f := &bufLockFile{
		docNode: docNode,
		deps:    deps,
		commits: commits,
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.uriToFile[normalized] = f
}

// Close stops tracking a buf.lock file.
func (m *bufLockManager) Close(uri protocol.URI) {
	m.mu.Lock()
	delete(m.uriToFile, normalizeURI(uri))
	m.mu.Unlock()
}

// GetHover returns hover documentation for the buf.lock field at the given
// position, or nil if the position does not correspond to a known element.
func (m *bufLockManager) GetHover(uri protocol.URI, pos protocol.Position) *protocol.Hover {
	m.mu.Lock()
	f, ok := m.uriToFile[normalizeURI(uri)]
	m.mu.Unlock()
	if !ok || f.docNode == nil {
		return nil
	}
	return bufLockHover(f.docNode, pos)
}

// GetDocumentLinks returns document links for all deps[*].name module
// references in the buf.lock file. Each link points to the BSR page for the
// referenced module.
func (m *bufLockManager) GetDocumentLinks(uri protocol.URI) []protocol.DocumentLink {
	m.mu.Lock()
	f, ok := m.uriToFile[normalizeURI(uri)]
	m.mu.Unlock()
	if !ok {
		return nil
	}
	links := make([]protocol.DocumentLink, 0, len(f.deps))
	for _, dep := range f.deps {
		ref, err := bufparse.ParseRef(dep.ref)
		if err != nil {
			continue
		}
		links = append(links, protocol.DocumentLink{
			Range:  dep.refRange,
			Target: protocol.DocumentURI(bsrRefDocURL(ref)),
		})
	}
	return links
}

// InlayHints returns inlay hints for the given buf.lock URI, rendering the
// latest commit as virtual text next to each dep's commit pin when the BSR
// reports a newer commit.
//
// Cache misses are resolved asynchronously: the call kicks off a background
// fetch and returns whatever is currently cached. After the fetch completes
// successfully, the server sends workspace/inlayHint/refresh so the client
// re-requests with populated cache data. Provider errors (network, auth) are
// logged and otherwise ignored — inlay hints are an enhancement and must
// degrade silently when the BSR is unreachable.
func (m *bufLockManager) InlayHints(ctx context.Context, uri protocol.URI) []inlayHint {
	m.mu.Lock()
	f, ok := m.uriToFile[normalizeURI(uri)]
	m.mu.Unlock()
	if !ok || len(f.commits) == 0 {
		return nil
	}

	hints := buildBufLockInlayHints(f.commits, m.lsp.versionCache)

	// Resolve uncached entries in the background and refresh inlay hints when done.
	refsToFetch := make([]bufparse.Ref, 0, len(f.commits))
	for _, pin := range f.commits {
		if _, cached := m.lsp.versionCache.GetModuleCommit(pin.fullName); cached {
			continue
		}
		ref, err := bufparse.ParseRef(pin.fullName)
		if err != nil {
			continue
		}
		refsToFetch = append(refsToFetch, ref)
	}
	if len(refsToFetch) > 0 {
		go m.fetchAndRefresh(uri, refsToFetch)
	}
	return hints
}

// fetchAndRefresh resolves latest commits for refs in the workspace
// containing uri. Looks up the buf.lock digest type via the workspace dep
// manager (cheap local read), then delegates to the shared fetch helper.
func (m *bufLockManager) fetchAndRefresh(uri protocol.URI, refs []bufparse.Ref) {
	dirPath := filepath.Dir(uri.Filename())
	wdm, err := m.lsp.controller.GetWorkspaceDepManager(m.lsp.connCtx, dirPath)
	if err != nil {
		// Workspace not constructible (rare; usually means buf.yaml errors). Skip.
		return
	}
	fetchModuleCommitsAndRefresh(m.lsp, refs, wdm.BufLockFileDigestType())
}

// buildBufLockInlayHints renders one hint per pinned dep whose latest commit
// (from the cache) differs from the file's pinned commit.
func buildBufLockInlayHints(commits []bufLockPin, cache *versionCache) []inlayHint {
	var hints []inlayHint
	for _, pin := range commits {
		latest, cached := cache.GetModuleCommit(pin.fullName)
		if !cached {
			continue
		}
		latestStr := uuidutil.ToDashless(latest)
		if latestStr == pin.commit {
			continue
		}
		hints = append(hints, inlayHint{
			Position:    pin.commitRange.End,
			Label:       fmt.Sprintf(" → %s", latestStr),
			PaddingLeft: true,
		})
	}
	return hints
}

// parseBufLockDepsAndCommits walks the parsed buf.lock document and collects
// each entry's name (as a bsrRef) and commit pin (as a bufLockPin). Names
// without a matching commit are still returned in the deps slice; commits
// without a name are skipped (we cannot tie them to a module).
func parseBufLockDepsAndCommits(doc *yaml.Node) ([]bsrRef, []bufLockPin) {
	if doc == nil || doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, nil
	}
	mapping := doc.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return nil, nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		keyNode := mapping.Content[i]
		valNode := mapping.Content[i+1]
		if keyNode.Value != "deps" || valNode.Kind != yaml.SequenceNode {
			continue
		}
		var (
			deps    []bsrRef
			commits []bufLockPin
		)
		for _, item := range valNode.Content {
			if item.Kind != yaml.MappingNode {
				continue
			}
			var (
				name        string
				nameRange   protocol.Range
				commit      string
				commitRange protocol.Range
			)
			for j := 0; j+1 < len(item.Content); j += 2 {
				k, v := item.Content[j], item.Content[j+1]
				if v.Kind != yaml.ScalarNode || v.Value == "" {
					continue
				}
				switch k.Value {
				case "name":
					name = v.Value
					nameRange = yamlNodeRange(v)
				case "commit":
					commit = v.Value
					commitRange = yamlNodeRange(v)
				}
			}
			if name != "" {
				deps = append(deps, bsrRef{ref: name, refRange: nameRange})
				if commit != "" {
					commits = append(commits, bufLockPin{
						fullName:    name,
						commit:      commit,
						commitRange: commitRange,
					})
				}
			}
		}
		return deps, commits
	}
	return nil, nil
}

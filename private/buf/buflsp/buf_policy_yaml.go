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
	"path/filepath"
	"strings"
	"sync"

	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufpolicy/bufpolicyconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginref"
	"go.lsp.dev/protocol"
	"gopkg.in/yaml.v3"
)

// isBufPolicyYAMLURI reports whether uri refers to a buf.policy.yaml file.
func isBufPolicyYAMLURI(uri protocol.URI) bool {
	return filepath.Base(uri.Filename()) == bufpolicyconfig.DefaultBufPolicyYAMLFileName
}

// bufPolicyYAMLManager tracks open buf.policy.yaml files in the LSP session.
type bufPolicyYAMLManager struct {
	lsp       *lsp
	mu        sync.Mutex
	uriToFile map[protocol.URI]*bufPolicyYAMLFile
}

func newBufPolicyYAMLManager(lsp *lsp) *bufPolicyYAMLManager {
	return &bufPolicyYAMLManager{
		lsp:       lsp,
		uriToFile: make(map[protocol.URI]*bufPolicyYAMLFile),
	}
}

// bufPolicyYAMLFile holds the parsed state of an open buf.policy.yaml file.
type bufPolicyYAMLFile struct {
	text                string     // raw file content, used for completion
	docNode             *yaml.Node // parsed YAML document node, nil if parse failed
	refs                []bsrRef   // name: and plugins[*].plugin BSR references, in document order
	versionedPluginRefs []bsrRef   // plugins[*].plugin entries with an explicit version
}

// Track opens or refreshes a buf.policy.yaml file.
func (m *bufPolicyYAMLManager) Track(uri protocol.URI, text string) {
	normalized := normalizeURI(uri)
	docNode := parseYAMLDoc(text)
	refs, versionedPluginRefs := parseBufPolicyYAMLRefs(docNode)
	f := &bufPolicyYAMLFile{
		text:                text,
		docNode:             docNode,
		refs:                refs,
		versionedPluginRefs: versionedPluginRefs,
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.uriToFile[normalized] = f
}

// Close stops tracking a buf.policy.yaml file.
func (m *bufPolicyYAMLManager) Close(uri protocol.URI) {
	m.mu.Lock()
	delete(m.uriToFile, normalizeURI(uri))
	m.mu.Unlock()
}

// GetCompletion returns completion items for the buf.policy.yaml field or value
// at the given cursor position, or nil if no completions apply.
func (m *bufPolicyYAMLManager) GetCompletion(uri protocol.URI, pos protocol.Position) []protocol.CompletionItem {
	m.mu.Lock()
	f, ok := m.uriToFile[normalizeURI(uri)]
	m.mu.Unlock()
	if !ok {
		return nil
	}
	return getBufPolicyYAMLCompletionItems(f.docNode, f.text, pos)
}

// GetHover returns hover documentation for the buf.policy.yaml field at the
// given position, or nil if the position does not correspond to a known element.
func (m *bufPolicyYAMLManager) GetHover(uri protocol.URI, pos protocol.Position) *protocol.Hover {
	m.mu.Lock()
	f, ok := m.uriToFile[normalizeURI(uri)]
	m.mu.Unlock()
	if !ok || f.docNode == nil {
		return nil
	}
	return bufPolicyYAMLHover(f.docNode, pos)
}

// GetDocumentLinks returns document links for BSR references in the
// buf.policy.yaml file.
//
// Links are created for:
//   - The top-level name: value (always a BSR policy reference).
//   - plugins[*].plugin values that parse as BSR references (registry/owner/name
//     format). Local binary names and file paths are skipped.
func (m *bufPolicyYAMLManager) GetDocumentLinks(uri protocol.URI) []protocol.DocumentLink {
	m.mu.Lock()
	f, ok := m.uriToFile[normalizeURI(uri)]
	m.mu.Unlock()
	if !ok {
		return nil
	}
	links := make([]protocol.DocumentLink, 0, len(f.refs))
	for _, entry := range f.refs {
		ref, err := bufparse.ParseRef(entry.ref)
		if err != nil {
			continue
		}
		links = append(links, protocol.DocumentLink{
			Range:  entry.refRange,
			Target: protocol.DocumentURI(bsrRefDocURL(ref)),
		})
	}
	return links
}

// InlayHints returns inlay hints for the given buf.policy.yaml URI, rendering
// the latest version as virtual text next to each versioned plugin entry whose
// pinned version is behind the latest published on the BSR.
func (m *bufPolicyYAMLManager) InlayHints(_ context.Context, uri protocol.URI) []inlayHint {
	m.mu.Lock()
	f, ok := m.uriToFile[normalizeURI(uri)]
	m.mu.Unlock()
	if !ok || len(f.versionedPluginRefs) == 0 {
		return nil
	}
	hints, missing := pluginInlayHintsForRefs(f.versionedPluginRefs, m.lsp.versionCache)
	if len(missing) > 0 {
		go fetchPluginVersionsAndRefresh(m.lsp, missing)
	}
	return hints
}

// parseBufPolicyYAMLRefs walks the parsed buf.policy.yaml document and
// collects BSR references in document order: the top-level name value followed
// by any plugins[*].plugin values that look like BSR references
// (registry/owner/name format). versionedPluginRefs is the subset of
// plugins[*].plugin entries that include an explicit version, used by inlay
// hints to compare against the latest BSR version.
//
// Local plugin names (no slashes) and absolute paths are not linked. Values
// where the registry component does not contain a dot (e.g. "./bin/tool"
// parses as registry ".") are skipped.
//
// Returns nil for both slices if doc is nil or not a valid YAML document.
func parseBufPolicyYAMLRefs(doc *yaml.Node) (refs []bsrRef, versionedPluginRefs []bsrRef) {
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
		switch keyNode.Value {
		case "name":
			if valNode.Kind == yaml.ScalarNode && valNode.Value != "" {
				refs = append(refs, bsrRef{ref: valNode.Value, refRange: yamlNodeRange(valNode)})
			}
		case "plugins":
			if valNode.Kind != yaml.SequenceNode {
				continue
			}
			for _, item := range valNode.Content {
				if item.Kind != yaml.MappingNode {
					continue
				}
				for j := 0; j+1 < len(item.Content); j += 2 {
					k, v := item.Content[j], item.Content[j+1]
					if k.Value != "plugin" || v.Kind != yaml.ScalarNode || v.Value == "" {
						continue
					}
					// Skip entries that can't be BSR references: a valid BSR
					// reference must parse as registry/owner/name where the
					// registry looks like a hostname (contains a dot, does not
					// start with one). This filters out local binary names and
					// file paths like "./bin/tool".
					if !looksLikeBSRRef(v.Value) {
						continue
					}
					entry := bsrRef{ref: v.Value, refRange: yamlNodeRange(v)}
					refs = append(refs, entry)
					if _, version, err := bufremotepluginref.ParsePluginIdentityOptionalVersion(v.Value); err == nil && version != "" {
						versionedPluginRefs = append(versionedPluginRefs, entry)
					}
				}
			}
		}
	}
	return refs, versionedPluginRefs
}

// looksLikeBSRRef reports whether s could be a BSR reference by checking
// whether the portion before the first "/" contains a dot and does not start
// with one. This distinguishes "buf.build/owner/name" from local paths like
// "./bin/tool" (registry ".") or bare binary names like "protoc-gen-go" (no
// slash).
func looksLikeBSRRef(s string) bool {
	registry, _, ok := strings.Cut(s, "/")
	if !ok {
		return false
	}
	return strings.Contains(registry, ".") && !strings.HasPrefix(registry, ".")
}

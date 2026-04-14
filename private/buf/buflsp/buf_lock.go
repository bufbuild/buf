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
	"path/filepath"
	"sync"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"go.lsp.dev/protocol"
	"gopkg.in/yaml.v3"
)

// isBufLockURI reports whether uri refers to a buf.lock file.
func isBufLockURI(uri protocol.URI) bool {
	return filepath.Base(uri.Filename()) == bufconfig.DefaultBufLockFileName
}

// bufLockManager tracks open buf.lock files in the LSP session.
type bufLockManager struct {
	mu        sync.Mutex
	uriToFile map[protocol.URI]*bufLockFile
}

func newBufLockManager() *bufLockManager {
	return &bufLockManager{
		uriToFile: make(map[protocol.URI]*bufLockFile),
	}
}

// bufLockFile holds the parsed state of an open buf.lock file.
type bufLockFile struct {
	docNode *yaml.Node // parsed YAML document node, nil if parse failed
	deps    []bsrRef   // deps[*].name BSR module references
}

// Track opens or refreshes a buf.lock file.
func (m *bufLockManager) Track(uri protocol.URI, text string) {
	normalized := normalizeURI(uri)
	docNode := parseYAMLDoc(text)
	f := &bufLockFile{
		docNode: docNode,
		deps:    parseBufLockDeps(docNode),
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

// parseBufLockDeps walks the parsed buf.lock document and collects all
// deps[*].name scalar values with their source positions.
//
// Returns nil if doc is nil or not a valid YAML document.
func parseBufLockDeps(doc *yaml.Node) []bsrRef {
	if doc == nil || doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil
	}
	mapping := doc.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		keyNode := mapping.Content[i]
		valNode := mapping.Content[i+1]
		if keyNode.Value != "deps" || valNode.Kind != yaml.SequenceNode {
			continue
		}
		var deps []bsrRef
		for _, item := range valNode.Content {
			if item.Kind != yaml.MappingNode {
				continue
			}
			for j := 0; j+1 < len(item.Content); j += 2 {
				k, v := item.Content[j], item.Content[j+1]
				if k.Value == "name" && v.Kind == yaml.ScalarNode && v.Value != "" {
					deps = append(deps, bsrRef{
						ref:      v.Value,
						refRange: yamlNodeRange(v),
					})
				}
			}
		}
		return deps
	}
	return nil
}

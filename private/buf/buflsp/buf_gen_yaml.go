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

// isBufGenYAMLURI reports whether uri refers to a buf.gen.yaml file.
func isBufGenYAMLURI(uri protocol.URI) bool {
	return filepath.Base(uri.Filename()) == bufconfig.DefaultBufGenYAMLFileName
}

// bufGenYAMLManager tracks open buf.gen.yaml files in the LSP session.
type bufGenYAMLManager struct {
	mu        sync.Mutex
	uriToFile map[protocol.URI]*bufGenYAMLFile
}

func newBufGenYAMLManager() *bufGenYAMLManager {
	return &bufGenYAMLManager{
		uriToFile: make(map[protocol.URI]*bufGenYAMLFile),
	}
}

// bufGenYAMLFile holds the parsed state of an open buf.gen.yaml file.
type bufGenYAMLFile struct {
	docNode *yaml.Node // parsed YAML document node, nil if parse failed
	refs    []bsrRef   // plugins[*].remote and inputs[*].module BSR references
}

// Track opens or refreshes a buf.gen.yaml file.
func (m *bufGenYAMLManager) Track(uri protocol.URI, text string) {
	normalized := normalizeURI(uri)
	docNode := parseYAMLDoc(text)
	f := &bufGenYAMLFile{
		docNode: docNode,
		refs:    parseBufGenYAMLRefs(docNode),
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.uriToFile[normalized] = f
}

// Close stops tracking a buf.gen.yaml file.
func (m *bufGenYAMLManager) Close(uri protocol.URI) {
	m.mu.Lock()
	delete(m.uriToFile, normalizeURI(uri))
	m.mu.Unlock()
}

// GetHover returns hover documentation for the buf.gen.yaml field at the given
// position, or nil if the position does not correspond to a known element.
func (m *bufGenYAMLManager) GetHover(uri protocol.URI, pos protocol.Position) *protocol.Hover {
	m.mu.Lock()
	f, ok := m.uriToFile[normalizeURI(uri)]
	m.mu.Unlock()
	if !ok || f.docNode == nil {
		return nil
	}
	return bufGenYAMLHover(f.docNode, pos)
}

// GetDocumentLinks returns document links for all remote plugin and input
// module BSR references in the buf.gen.yaml file.
//
// Links are created for plugins[*].remote and inputs[*].module values that
// parse as valid BSR references. Each link points to the BSR page for the
// referenced plugin or module, including a /docs/<ref> path when an explicit
// version or label is present.
func (m *bufGenYAMLManager) GetDocumentLinks(uri protocol.URI) []protocol.DocumentLink {
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

// parseBufGenYAMLRefs walks the parsed buf.gen.yaml document and collects all
// BSR references: plugins[*].remote and inputs[*].module scalar values with
// their source positions, in document order.
//
// Returns nil if doc is nil or not a valid document.
func parseBufGenYAMLRefs(doc *yaml.Node) []bsrRef {
	if doc == nil || doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil
	}
	mapping := doc.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return nil
	}
	var refs []bsrRef
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		keyNode := mapping.Content[i]
		valNode := mapping.Content[i+1]
		switch keyNode.Value {
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
					if k.Value == "remote" && v.Kind == yaml.ScalarNode && v.Value != "" {
						refs = append(refs, bsrRef{ref: v.Value, refRange: yamlNodeRange(v)})
					}
				}
			}
		case "inputs":
			if valNode.Kind != yaml.SequenceNode {
				continue
			}
			for _, item := range valNode.Content {
				if item.Kind != yaml.MappingNode {
					continue
				}
				for j := 0; j+1 < len(item.Content); j += 2 {
					k, v := item.Content[j], item.Content[j+1]
					if k.Value == "module" && v.Kind == yaml.ScalarNode && v.Value != "" {
						refs = append(refs, bsrRef{ref: v.Value, refRange: yamlNodeRange(v)})
					}
				}
			}
		}
	}
	return refs
}

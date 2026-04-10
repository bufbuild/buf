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
}

// Track opens or refreshes a buf.gen.yaml file.
func (m *bufGenYAMLManager) Track(uri protocol.URI, text string) {
	normalized := normalizeURI(uri)
	f := &bufGenYAMLFile{docNode: parseYAMLDoc(text)}
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

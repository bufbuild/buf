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
}

// Track opens or refreshes a buf.lock file.
func (m *bufLockManager) Track(uri protocol.URI, text string) {
	normalized := normalizeURI(uri)
	f := &bufLockFile{docNode: parseYAMLDoc(text)}
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

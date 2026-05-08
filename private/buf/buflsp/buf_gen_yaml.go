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
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"buf.build/go/standard/xos/xexec"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginref"
	"go.lsp.dev/protocol"
	"gopkg.in/yaml.v3"
)

// CommandRunGenerate is the LSP workspace command to run buf generate for a buf.gen.yaml file.
const CommandRunGenerate = "buf.generate.run"

// isBufGenYAMLURI reports whether uri refers to a buf.gen.yaml file.
func isBufGenYAMLURI(uri protocol.URI) bool {
	return filepath.Base(uri.Filename()) == bufconfig.DefaultBufGenYAMLFileName
}

// bufGenYAMLManager tracks open buf.gen.yaml files in the LSP session.
type bufGenYAMLManager struct {
	lsp       *lsp
	mu        sync.Mutex
	uriToFile map[protocol.URI]*bufGenYAMLFile
}

func newBufGenYAMLManager(lsp *lsp) *bufGenYAMLManager {
	return &bufGenYAMLManager{
		lsp:       lsp,
		uriToFile: make(map[protocol.URI]*bufGenYAMLFile),
	}
}

// bufGenYAMLFile holds the parsed state of an open buf.gen.yaml file.
type bufGenYAMLFile struct {
	text                string     // raw file content, used for completion
	docNode             *yaml.Node // parsed YAML document node, nil if parse failed
	refs                []bsrRef   // plugins[*].remote and inputs[*].module BSR references
	versionedPluginRefs []bsrRef   // plugins[*].remote with an explicit version (for update checks)
	pluginsKeyLine      uint32     // 0-indexed line of the "plugins:" key
}

// Track opens or refreshes a buf.gen.yaml file.
func (m *bufGenYAMLManager) Track(uri protocol.URI, text string) {
	normalized := normalizeURI(uri)
	docNode := parseYAMLDoc(text)
	allRefs, versionedPluginRefs, pluginsKeyLine := parseBufGenYAMLRefs(docNode)
	f := &bufGenYAMLFile{
		text:                text,
		docNode:             docNode,
		refs:                allRefs,
		versionedPluginRefs: versionedPluginRefs,
		pluginsKeyLine:      pluginsKeyLine,
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.uriToFile[normalized] = f
}

// Close stops tracking a buf.gen.yaml file and clears any diagnostics it published.
func (m *bufGenYAMLManager) Close(ctx context.Context, uri protocol.URI) {
	normalized := normalizeURI(uri)
	m.mu.Lock()
	delete(m.uriToFile, normalized)
	m.mu.Unlock()
	publishDiagnostics(ctx, m.lsp.client, normalized, nil)
}

// GetCompletion returns completion items for the buf.gen.yaml field or value at
// the given cursor position, or nil if no completions apply.
func (m *bufGenYAMLManager) GetCompletion(uri protocol.URI, pos protocol.Position) []protocol.CompletionItem {
	m.mu.Lock()
	f, ok := m.uriToFile[normalizeURI(uri)]
	m.mu.Unlock()
	if !ok {
		return nil
	}
	return getBufGenYAMLCompletionItems(f.docNode, f.text, pos)
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

// GetCodeLenses returns code lenses for the given buf.gen.yaml URI.
func (m *bufGenYAMLManager) GetCodeLenses(uri protocol.URI) []protocol.CodeLens {
	m.mu.Lock()
	_, ok := m.uriToFile[normalizeURI(uri)]
	m.mu.Unlock()
	if !ok {
		return nil
	}
	return []protocol.CodeLens{
		{
			Range: protocol.Range{},
			Command: &protocol.Command{
				Title:     "Run buf generate",
				Command:   CommandRunGenerate,
				Arguments: []any{string(uri)},
			},
		},
	}
}

// ExecuteRunGenerate runs buf generate in the directory containing the given
// buf.gen.yaml URI. Results are reported to the user via ShowMessage.
func (m *bufGenYAMLManager) ExecuteRunGenerate(ctx context.Context, uri protocol.URI) error {
	dirPath := filepath.Dir(uri.Filename())
	executable, err := os.Executable()
	if err != nil {
		executable = "buf"
	}
	msgType := protocol.MessageTypeInfo
	msg := "buf generate completed successfully"
	var outBuf bytes.Buffer
	if err := xexec.Run(ctx, executable,
		xexec.WithArgs("generate"),
		xexec.WithDir(dirPath),
		xexec.WithStdout(&outBuf),
		xexec.WithStderr(&outBuf),
	); err != nil {
		msgType = protocol.MessageTypeError
		msg = fmt.Sprintf("buf generate failed:\n%s", outBuf.String())
	}
	_ = m.lsp.client.ShowMessage(ctx, &protocol.ShowMessageParams{
		Type:    msgType,
		Message: msg,
	})
	return nil
}

// InlayHints returns inlay hints for the given buf.gen.yaml URI, rendering the
// latest version as virtual text next to each versioned plugin entry whose
// pinned version is behind the latest published on the BSR.
//
// Cache misses trigger a background fetch; once the cache populates, the
// server sends workspace/inlayHint/refresh and the client re-requests.
// Provider errors are logged at debug level and otherwise ignored.
func (m *bufGenYAMLManager) InlayHints(_ context.Context, uri protocol.URI) []inlayHint {
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

// parseBufGenYAMLRefs walks the parsed buf.gen.yaml document and collects BSR
// references in document order: plugins[*].remote and inputs[*].module scalar
// values with their source positions.
func parseBufGenYAMLRefs(doc *yaml.Node) ([]bsrRef, []bsrRef, uint32) {
	if doc == nil || doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, nil, 0
	}
	mapping := doc.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return nil, nil, 0
	}
	var refs, versionedPluginRefs []bsrRef
	var pluginsKeyLine uint32
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		keyNode := mapping.Content[i]
		valNode := mapping.Content[i+1]
		switch keyNode.Value {
		case "plugins":
			if valNode.Kind != yaml.SequenceNode {
				continue
			}
			pluginsKeyLine = uint32(keyNode.Line - 1) // yaml.Node.Line is 1-indexed and always ≥ 1
			for _, item := range valNode.Content {
				if item.Kind != yaml.MappingNode {
					continue
				}
				for j := 0; j+1 < len(item.Content); j += 2 {
					k, v := item.Content[j], item.Content[j+1]
					if k.Value == "remote" && v.Kind == yaml.ScalarNode && v.Value != "" {
						entry := bsrRef{ref: v.Value, refRange: yamlNodeRange(v)}
						refs = append(refs, entry)
						if _, version, err := bufremotepluginref.ParsePluginIdentityOptionalVersion(v.Value); err == nil && version != "" {
							versionedPluginRefs = append(versionedPluginRefs, entry)
						}
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
	return refs, versionedPluginRefs, pluginsKeyLine
}

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

// CommandCheckPluginUpdates is the LSP workspace command to check for newer versions of remote
// plugins in a buf.gen.yaml file and publish informational diagnostics for any that are outdated.
const CommandCheckPluginUpdates = "buf.generate.checkPluginUpdates"

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
	f, ok := m.uriToFile[normalizeURI(uri)]
	m.mu.Unlock()
	if !ok {
		return nil
	}
	lenses := []protocol.CodeLens{
		{
			Range: protocol.Range{},
			Command: &protocol.Command{
				Title:     "Run buf generate",
				Command:   CommandRunGenerate,
				Arguments: []any{string(uri)},
			},
		},
	}
	if len(f.versionedPluginRefs) > 0 {
		pluginsRange := protocol.Range{
			Start: protocol.Position{Line: f.pluginsKeyLine},
			End:   protocol.Position{Line: f.pluginsKeyLine},
		}
		lenses = append(lenses, protocol.CodeLens{
			Range: pluginsRange,
			Command: &protocol.Command{
				Title:     "Check for plugin updates",
				Command:   CommandCheckPluginUpdates,
				Arguments: []any{string(uri)},
			},
		})
	}
	return lenses
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

// ExecuteCheckPluginUpdates queries the BSR for the latest version of each
// versioned remote plugin in the buf.gen.yaml file and publishes an
// informational diagnostic on any plugin line where a newer version is
// available. It does not modify any files.
func (m *bufGenYAMLManager) ExecuteCheckPluginUpdates(ctx context.Context, uri protocol.URI) error {
	normalized := normalizeURI(uri)
	m.mu.Lock()
	f, ok := m.uriToFile[normalized]
	m.mu.Unlock()
	if !ok || len(f.versionedPluginRefs) == 0 {
		publishDiagnostics(ctx, m.lsp.client, normalized, nil)
		return nil
	}

	var diagnostics []protocol.Diagnostic
	for _, entry := range f.versionedPluginRefs {
		identity, pinnedVersion, err := bufremotepluginref.ParsePluginIdentityOptionalVersion(entry.ref)
		if err != nil || pinnedVersion == "" {
			continue
		}
		latestVersion, err := m.lsp.curatedPluginVersionProvider.GetLatestVersion(
			ctx, identity.Remote(), identity.Owner(), identity.Plugin(),
		)
		if err != nil {
			return fmt.Errorf("resolving latest version for %s: %w", identity.IdentityString(), err)
		}
		if latestVersion == "" || latestVersion == pinnedVersion {
			continue
		}
		diagnostics = append(diagnostics, protocol.Diagnostic{
			Range:    entry.refRange,
			Severity: protocol.DiagnosticSeverityInformation,
			Source:   serverName,
			Message: fmt.Sprintf(
				"%s can be updated (latest: %s)",
				identity.IdentityString(),
				latestVersion,
			),
		})
	}
	publishDiagnostics(ctx, m.lsp.client, normalized, diagnostics)
	return nil
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

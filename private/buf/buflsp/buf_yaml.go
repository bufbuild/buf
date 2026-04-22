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

// This file implements tracking and code lens support for buf.yaml files.

package buflsp

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bufbuild/buf/private/buf/bufworkspace"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"go.lsp.dev/protocol"
	"gopkg.in/yaml.v3"
)

const (
	// commandUpdateAllDeps is the LSP workspace command to update all dependencies
	// in the buf.yaml file to their latest versions.
	commandUpdateAllDeps = "buf.dep.updateAll"
	// commandCheckUpdates is the LSP workspace command to check whether newer
	// versions of dependencies are available and publish informational diagnostics
	// for any that are outdated. It does not modify any files.
	commandCheckUpdates = "buf.dep.checkUpdates"
	// "deps" is the deps: key in buf.yaml.
	//
	// Ref: https://buf.build/docs/configuration/v2/buf-yaml/#deps
	bufYAMLDepsKey = "deps"
)

// bufYAMLManager tracks open buf.yaml files in the LSP session.
type bufYAMLManager struct {
	lsp       *lsp
	mu        sync.Mutex
	uriToFile map[protocol.URI]*bufYAMLFile
}

func newBufYAMLManager(lsp *lsp) *bufYAMLManager {
	return &bufYAMLManager{
		lsp:       lsp,
		uriToFile: make(map[protocol.URI]*bufYAMLFile),
	}
}

// bufYAMLFile holds the parsed state of an open buf.yaml file.
type bufYAMLFile struct {
	depsKeyLine          uint32 // 0-indexed line of the "deps:" key
	deps                 []bufYAMLDep
	docNode              *yaml.Node // parsed YAML document node, nil if parse failed
	updateDiagnostics    []protocol.Diagnostic
	unusedDepDiagnostics []protocol.Diagnostic
	cancelUnusedDep      context.CancelFunc // cancels the in-flight checkUnusedDeps goroutine
}

// bufYAMLDep is a single entry in the deps sequence with its source position.
type bufYAMLDep struct {
	// ref is the dep string, e.g. "buf.build/googleapis/googleapis".
	ref string
	// depRange is the range spanning the dep string value in the file.
	depRange protocol.Range
}

// isBufYAMLURI reports whether uri refers to a buf.yaml file.
func isBufYAMLURI(uri protocol.URI) bool {
	return filepath.Base(uri.Filename()) == bufconfig.DefaultBufYAMLFileName
}

// Track opens or refreshes a buf.yaml file.
func (m *bufYAMLManager) Track(uri protocol.URI, text string) {
	normalized := normalizeURI(uri)
	f := &bufYAMLFile{}
	f.depsKeyLine, f.deps, f.docNode, _ = parseBufYAMLDeps([]byte(text))

	ctx, cancel := context.WithCancel(m.lsp.connCtx)
	f.cancelUnusedDep = cancel

	m.mu.Lock()
	if old, ok := m.uriToFile[normalized]; ok && old.cancelUnusedDep != nil {
		old.cancelUnusedDep()
	}
	m.uriToFile[normalized] = f
	m.mu.Unlock()

	go m.checkUnusedDeps(ctx, normalized)
}

// Close stops tracking a buf.yaml file and clears any diagnostics it published.
func (m *bufYAMLManager) Close(ctx context.Context, uri protocol.URI) {
	normalized := normalizeURI(uri)
	m.mu.Lock()
	if f, ok := m.uriToFile[normalized]; ok && f.cancelUnusedDep != nil {
		f.cancelUnusedDep()
	}
	delete(m.uriToFile, normalized)
	m.mu.Unlock()
	m.publishDiagnostics(ctx, normalized, nil)
}

// GetCodeLenses returns code lenses for the given buf.yaml URI.
// Returns nil if no deps are declared.
//
// Two whole-file lenses are shown at the deps: key line:
//   - "Update all dependencies" triggers buf.dep.updateAll
//   - "Check for updates" triggers buf.dep.checkUpdates
func (m *bufYAMLManager) GetCodeLenses(uri protocol.URI) []protocol.CodeLens {
	m.mu.Lock()
	f, ok := m.uriToFile[normalizeURI(uri)]
	m.mu.Unlock()
	if !ok || len(f.deps) == 0 {
		return nil
	}
	keyLine := f.depsKeyLine
	keyRange := protocol.Range{
		Start: protocol.Position{Line: keyLine, Character: 0},
		End:   protocol.Position{Line: keyLine, Character: 0},
	}
	return []protocol.CodeLens{
		{
			Range: keyRange,
			Command: &protocol.Command{
				Title:     "Update all dependencies",
				Command:   commandUpdateAllDeps,
				Arguments: []any{string(uri)},
			},
		},
		{
			Range: keyRange,
			Command: &protocol.Command{
				Title:     "Check for updates",
				Command:   commandCheckUpdates,
				Arguments: []any{string(uri)},
			},
		},
	}
}

// ExecuteUpdateAll runs buf dep update for all configured deps in the
// workspace containing the given buf.yaml URI.
func (m *bufYAMLManager) ExecuteUpdateAll(ctx context.Context, uri protocol.URI) error {
	dirPath := filepath.Dir(uri.Filename())
	return updateDeps(ctx, m.lsp, dirPath)
}

// ExecuteCheckUpdates queries the BSR for the latest commit of each configured
// dependency and publishes an informational diagnostic on any dep line where a
// newer version is available. It does not modify any files.
func (m *bufYAMLManager) ExecuteCheckUpdates(ctx context.Context, uri protocol.URI) error {
	normalized := normalizeURI(uri)
	m.mu.Lock()
	f, ok := m.uriToFile[normalized]
	m.mu.Unlock()
	if !ok {
		return nil
	}

	dirPath := filepath.Dir(uri.Filename())
	workspaceDepManager, err := m.lsp.controller.GetWorkspaceDepManager(ctx, dirPath)
	if err != nil {
		return fmt.Errorf("getting workspace dep manager: %w", err)
	}

	configuredRefs, err := workspaceDepManager.ConfiguredDepModuleRefs(ctx)
	if err != nil {
		return fmt.Errorf("getting configured dep module refs: %w", err)
	}
	if len(configuredRefs) == 0 {
		m.mu.Lock()
		if f, ok = m.uriToFile[normalized]; ok {
			f.updateDiagnostics = nil
		}
		m.mu.Unlock()
		m.publishAllDiagnostics(ctx, normalized)
		return nil
	}

	// Build a map from full name → current pinned commit (from buf.lock).
	currentKeys, err := workspaceDepManager.ExistingBufLockFileDepModuleKeys(ctx)
	if err != nil {
		return fmt.Errorf("getting existing buf.lock deps: %w", err)
	}
	currentByFullName := make(map[string]bufmodule.ModuleKey, len(currentKeys))
	for _, key := range currentKeys {
		currentByFullName[key.FullName().String()] = key
	}

	// Build a map from full name → YAML position for each dep entry.
	depPosByFullName := make(map[string]protocol.Range, len(f.deps))
	for _, dep := range f.deps {
		ref, err := bufparse.ParseRef(dep.ref)
		if err != nil {
			continue
		}
		depPosByFullName[ref.FullName().String()] = dep.depRange
	}

	latestKeys, err := m.lsp.moduleKeyProvider.GetModuleKeysForModuleRefs(
		ctx,
		configuredRefs,
		workspaceDepManager.BufLockFileDigestType(),
	)
	if err != nil {
		return fmt.Errorf("resolving latest module versions: %w", err)
	}

	// Emit an informational diagnostic for every dep whose latest commit differs
	// from the currently pinned commit.
	var diagnostics []protocol.Diagnostic
	for _, latestKey := range latestKeys {
		fullName := latestKey.FullName().String()
		currentKey, pinned := currentByFullName[fullName]
		if !pinned {
			// Not yet pinned in buf.lock; skip.
			continue
		}
		if latestKey.CommitID() == currentKey.CommitID() {
			continue
		}
		depRange, ok := depPosByFullName[fullName]
		if !ok {
			continue
		}
		diagnostics = append(diagnostics, protocol.Diagnostic{
			Range:    depRange,
			Severity: protocol.DiagnosticSeverityInformation,
			Source:   serverName,
			Message: fmt.Sprintf(
				"%s can be updated to %s",
				fullName,
				uuidutil.ToDashless(latestKey.CommitID()),
			),
		})
	}
	m.mu.Lock()
	if f, ok = m.uriToFile[normalized]; ok {
		f.updateDiagnostics = diagnostics
	}
	m.mu.Unlock()
	m.publishAllDiagnostics(ctx, normalized)
	return nil
}

// publishDiagnostics clears existing diagnostics when passed nil.
func (m *bufYAMLManager) publishDiagnostics(ctx context.Context, uri protocol.URI, diagnostics []protocol.Diagnostic) {
	if diagnostics == nil {
		diagnostics = []protocol.Diagnostic{}
	}
	_ = m.lsp.client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	})
}

func (m *bufYAMLManager) publishAllDiagnostics(ctx context.Context, uri protocol.URI) {
	m.mu.Lock()
	f, ok := m.uriToFile[uri]
	var all []protocol.Diagnostic
	if ok {
		all = append(all, f.updateDiagnostics...)
		all = append(all, f.unusedDepDiagnostics...)
	}
	m.mu.Unlock()
	m.publishDiagnostics(ctx, uri, all)
}

func (m *bufYAMLManager) checkUnusedDeps(ctx context.Context, uri protocol.URI) {
	m.mu.Lock()
	f, ok := m.uriToFile[uri]
	m.mu.Unlock()
	if !ok || len(f.deps) == 0 {
		return
	}

	dirPath := filepath.Dir(uri.Filename())
	workspace, err := m.lsp.controller.GetWorkspace(ctx, dirPath)
	if err != nil {
		return
	}

	malformedDeps, err := bufworkspace.MalformedDepsForWorkspace(workspace)
	if err != nil {
		return
	}

	unusedFullNames := make(map[string]struct{}, len(malformedDeps))
	for _, d := range malformedDeps {
		if d.Type() == bufworkspace.MalformedDepTypeUnused {
			unusedFullNames[d.ModuleRef().FullName().String()] = struct{}{}
		}
	}

	var diags []protocol.Diagnostic
	for _, dep := range f.deps {
		ref, err := bufparse.ParseRef(dep.ref)
		if err != nil {
			continue
		}
		if _, unused := unusedFullNames[ref.FullName().String()]; !unused {
			continue
		}
		diags = append(diags, protocol.Diagnostic{
			Range:    dep.depRange,
			Severity: protocol.DiagnosticSeverityWarning,
			Source:   serverName,
			Message:  fmt.Sprintf("%s is declared as a dependency but is unused", ref.FullName()),
			Tags:     []protocol.DiagnosticTag{protocol.DiagnosticTagUnnecessary},
		})
	}

	m.mu.Lock()
	if f, ok = m.uriToFile[uri]; ok {
		f.unusedDepDiagnostics = diags
	}
	m.mu.Unlock()
	m.publishAllDiagnostics(ctx, uri)
}

// GetCodeActions returns QuickFix code actions for unused-dep diagnostics overlapping params.Range.
func (m *bufYAMLManager) GetCodeActions(uri protocol.URI, params *protocol.CodeActionParams) []protocol.CodeAction {
	normalized := normalizeURI(uri)
	m.mu.Lock()
	f, ok := m.uriToFile[normalized]
	var unusedDiags []protocol.Diagnostic
	var deps []bufYAMLDep
	if ok {
		unusedDiags = f.unusedDepDiagnostics
		deps = f.deps
	}
	m.mu.Unlock()
	if !ok {
		return nil
	}

	// Read buf.lock adjacent to buf.yaml (best-effort; absent lock is fine).
	lockFilePath := filepath.Join(filepath.Dir(uri.Filename()), bufconfig.DefaultBufLockFileName)
	lockURI := FilePathToURI(lockFilePath)
	var lockDepRanges map[string]protocol.Range
	if content, err := os.ReadFile(lockFilePath); err == nil {
		lockDepRanges = parseBufLockDepRanges(content)
	}

	var actions []protocol.CodeAction
	for _, diag := range unusedDiags {
		if !rangesOverlap(diag.Range, params.Range) {
			continue
		}
		for _, dep := range deps {
			if dep.depRange != diag.Range {
				continue
			}
			line := dep.depRange.Start.Line
			changes := map[protocol.DocumentURI][]protocol.TextEdit{
				uri: {{
					Range: protocol.Range{
						Start: protocol.Position{Line: line, Character: 0},
						End:   protocol.Position{Line: line + 1, Character: 0},
					},
					NewText: "",
				}},
			}
			if ref, err := bufparse.ParseRef(dep.ref); err == nil {
				if lockRange, ok := lockDepRanges[ref.FullName().String()]; ok {
					changes[lockURI] = []protocol.TextEdit{{Range: lockRange, NewText: ""}}
				}
			}
			actions = append(actions, protocol.CodeAction{
				Title:       fmt.Sprintf("Remove unused dependency %s", dep.ref),
				Kind:        protocol.QuickFix,
				IsPreferred: true,
				Diagnostics: []protocol.Diagnostic{diag},
				Edit:        &protocol.WorkspaceEdit{Changes: changes},
			})
		}
	}
	return actions
}

// parseBufLockDepRanges returns a map from module full name to the line range
// spanning its entry in buf.lock. The range runs from the `  - name:` line up to
// (but not including) the start of the next entry, or the end of the non-empty
// content for the last entry.
func parseBufLockDepRanges(content []byte) map[string]protocol.Range {
	lines := strings.Split(string(content), "\n")
	result := make(map[string]protocol.Range)

	type pending struct {
		name      string
		startLine uint32
	}
	var cur *pending

	for i, line := range lines {
		name, ok := strings.CutPrefix(line, "  - name: ")
		if !ok {
			continue
		}
		if cur != nil {
			result[cur.name] = protocol.Range{
				Start: protocol.Position{Line: cur.startLine},
				End:   protocol.Position{Line: uint32(i)},
			}
		}
		cur = &pending{name: name, startLine: uint32(i)}
	}
	if cur != nil {
		end := uint32(len(lines))
		for end > cur.startLine+1 && lines[end-1] == "" {
			end--
		}
		result[cur.name] = protocol.Range{
			Start: protocol.Position{Line: cur.startLine},
			End:   protocol.Position{Line: end},
		}
	}
	return result
}

// updateDeps resolves all configured deps in buf.yaml to their latest commits
// (including transitive dependencies) and writes the updated buf.lock file.
func updateDeps(ctx context.Context, l *lsp, dirPath string) error {
	workspaceDepManager, err := l.controller.GetWorkspaceDepManager(ctx, dirPath)
	if err != nil {
		return fmt.Errorf("getting workspace dep manager: %w", err)
	}
	refs, err := workspaceDepManager.ConfiguredDepModuleRefs(ctx)
	if err != nil {
		return fmt.Errorf("getting configured dep module refs: %w", err)
	}
	if len(refs) == 0 {
		return nil
	}
	moduleKeys, err := l.moduleKeyProvider.GetModuleKeysForModuleRefs(
		ctx,
		refs,
		workspaceDepManager.BufLockFileDigestType(),
	)
	if err != nil {
		return fmt.Errorf("resolving module refs: %w", err)
	}
	allModuleKeys, err := moduleKeysWithTransitiveDeps(ctx, l, moduleKeys)
	if err != nil {
		return fmt.Errorf("resolving transitive deps: %w", err)
	}
	existingPluginKeys, err := workspaceDepManager.ExistingBufLockFileRemotePluginKeys(ctx)
	if err != nil {
		return err
	}
	existingPolicyKeys, err := workspaceDepManager.ExistingBufLockFileRemotePolicyKeys(ctx)
	if err != nil {
		return err
	}
	existingPolicyPluginKeys, err := workspaceDepManager.ExistingBufLockFilePolicyNameToRemotePluginKeys(ctx)
	if err != nil {
		return err
	}
	return workspaceDepManager.UpdateBufLockFile(
		ctx,
		allModuleKeys,
		existingPluginKeys,
		existingPolicyKeys,
		existingPolicyPluginKeys,
	)
}

// moduleKeysWithTransitiveDeps returns the given module keys plus all their
// transitive dependencies, using the BSR graph API.
func moduleKeysWithTransitiveDeps(
	ctx context.Context,
	l *lsp,
	moduleKeys []bufmodule.ModuleKey,
) ([]bufmodule.ModuleKey, error) {
	graph, err := l.graphProvider.GetGraphForModuleKeys(ctx, moduleKeys)
	if err != nil {
		return nil, err
	}
	var all []bufmodule.ModuleKey
	if err := graph.WalkNodes(
		func(key bufmodule.ModuleKey, _ []bufmodule.ModuleKey, _ []bufmodule.ModuleKey) error {
			all = append(all, key)
			return nil
		},
	); err != nil {
		return nil, err
	}
	return all, nil
}

// GetDocumentLinks returns document links for all dep entries in the buf.yaml
// file. If the dep has an explicit ref (e.g. "buf.build/acme/mod:v1.2.3"),
// the link points to that specific label or commit on BSR. Otherwise it
// points to the module root.
func (m *bufYAMLManager) GetDocumentLinks(uri protocol.URI) []protocol.DocumentLink {
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
			Range:  dep.depRange,
			Target: protocol.DocumentURI(bsrRefDocURL(ref)),
		})
	}
	return links
}

// GetHover returns hover documentation for the buf.yaml field or rule at the
// given position, or nil if the position does not correspond to a known element.
func (m *bufYAMLManager) GetHover(uri protocol.URI, pos protocol.Position) *protocol.Hover {
	m.mu.Lock()
	f, ok := m.uriToFile[normalizeURI(uri)]
	m.mu.Unlock()
	if !ok || f.docNode == nil {
		return nil
	}
	return bufYAMLHover(f.docNode, pos)
}

// parseBufYAMLDeps parses the top-level deps sequence from buf.yaml content.
// It returns the 0-indexed line of the "deps:" key, the dep entries with their
// source positions, the parsed YAML document node for further traversal, and
// any parse error.
//
// Both v1/v1beta1 and v2 buf.yaml formats are supported, as both have a
// top-level deps key containing a sequence of module reference strings.
func parseBufYAMLDeps(content []byte) (depsKeyLine uint32, deps []bufYAMLDep, docNode *yaml.Node, _ error) {
	var doc yaml.Node
	if err := yaml.NewDecoder(bytes.NewReader(content)).Decode(&doc); err != nil {
		return 0, nil, nil, err
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return 0, nil, &doc, nil
	}
	mapping := doc.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return 0, nil, &doc, nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		keyNode := mapping.Content[i]
		if keyNode.Value != bufYAMLDepsKey {
			continue
		}
		// yaml.v3 uses 1-indexed line/column; LSP uses 0-indexed.
		depsKeyLine = uint32(keyNode.Line - 1)
		seqNode := mapping.Content[i+1]
		if seqNode.Kind != yaml.SequenceNode {
			return depsKeyLine, nil, &doc, nil
		}
		deps = make([]bufYAMLDep, 0, len(seqNode.Content))
		for _, node := range seqNode.Content {
			if node.Kind != yaml.ScalarNode {
				continue
			}
			deps = append(deps, bufYAMLDep{
				ref:      node.Value,
				depRange: yamlNodeRange(node),
			})
		}
		return depsKeyLine, deps, &doc, nil
	}
	return 0, nil, &doc, nil
}

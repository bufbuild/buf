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
	"maps"
	"slices"
	"strings"

	"github.com/bufbuild/protocompile/experimental/ir"
	"go.lsp.dev/protocol"
	"gopkg.in/yaml.v3"
)

// bufGenYAMLPathValueKeys maps buf.gen.yaml value keys whose values are
// filesystem paths to whether only directory entries should be offered.
// out and directory are always directories.
var bufGenYAMLPathValueKeys = map[string]bool{
	"out":       true,
	"directory": true,
}

// bufGenYAMLPathSequenceSections maps buf.gen.yaml section names whose
// sequence items are filesystem paths to whether only directory entries
// should be offered. paths and exclude_paths accept files or directories.
var bufGenYAMLPathSequenceSections = map[string]bool{
	"paths":         false,
	"exclude_paths": false,
}

// bufGenYAMLTypeSequenceSections is the set of buf.gen.yaml section names
// whose sequence items are fully-qualified proto type names.
var bufGenYAMLTypeSequenceSections = map[string]bool{
	"types": true, "exclude_types": true,
}

// getBufGenYAMLCompletionItems returns completion items for a buf.gen.yaml file
// at the given cursor position. dirPath is the directory containing the
// buf.gen.yaml file, used to resolve filesystem path completions; pass "" to
// disable path completions (e.g. in tests). ws is the leased workspace for
// proto type name completions; pass nil to disable type completions.
func getBufGenYAMLCompletionItems(docNode *yaml.Node, text string, pos protocol.Position, dirPath string, ws *workspace) []protocol.CompletionItem {
	lines, prefix, ok := bufYAMLParseCursor(text, pos)
	if !ok {
		return nil
	}
	cursorLine := int(pos.Line)
	tokenStart := bufYAMLTokenStart(prefix)
	editRange := bufYAMLEditRange(pos, tokenStart, lines[cursorLine])
	if valueKey := bufYAMLValueKey(prefix); valueKey != "" {
		if dirsOnly, ok := bufGenYAMLPathValueKeys[valueKey]; ok {
			return bufYAMLPathItems(dirPath, prefix[tokenStart:], editRange, dirsOnly)
		}
		return bufGenYAMLValueItems(valueKey, editRange)
	}
	if tokenStart < len(prefix) && prefix[tokenStart] == '-' {
		return nil
	}
	currentIndent := bufYAMLLeadingSpaces(prefix)
	section, parentSection := bufYAMLCursorPath(lines, cursorLine, currentIndent)
	if section == "" && currentIndent == 0 {
		if bareKey := bufYAMLBareParentKey(lines, cursorLine); bufGenYAMLBareParentMappingKeys[bareKey] {
			return bufYAMLPrependIndent(bufGenYAMLKeyItems(bareKey, editRange, nil))
		}
	}
	if dirsOnly, ok := bufGenYAMLPathSequenceSections[section]; ok {
		return bufYAMLPathItems(dirPath, prefix[tokenStart:], editRange, dirsOnly)
	}
	if bufGenYAMLTypeSequenceSections[section] {
		return bufGenYAMLTypeItems(ws, prefix[tokenStart:], editRange)
	}
	existingKeys := bufYAMLASTExistingKeys(docNode, section, parentSection, cursorLine)
	if existingKeys == nil {
		existingKeys = bufYAMLExistingKeys(lines, cursorLine, currentIndent)
	}
	return bufGenYAMLKeyItems(section, editRange, existingKeys)
}

// bufGenYAMLValueItems returns completion items for the value of a known buf.gen.yaml key.
func bufGenYAMLValueItems(key string, editRange protocol.Range) []protocol.CompletionItem {
	var values []string
	switch key {
	case "version":
		values = []string{"v2", "v1", "v1beta1"}
	case "protoc_builtin":
		values = bufGenYAMLBuiltinPlugins
	case "strategy":
		values = []string{"directory", "all"}
	case "clean", "include_imports", "include_wkt", "enabled",
		"include_package_files", "recurse_submodules":
		values = []string{"true", "false"}
	case "compression":
		values = []string{"gzip", "bzip2", "zstd"}
	default:
		return nil
	}
	return makeCompletionValueItems(values, nil, editRange)
}

// bufGenYAMLBareParentMappingKeys is the set of top-level buf.gen.yaml keys
// whose value is a mapping. The bare-parent-key heuristic applies only to
// these: "managed:" is the only top-level mapping key; plugins and inputs are
// sequences.
var bufGenYAMLBareParentMappingKeys = map[string]bool{
	"managed": true,
}

// bufGenYAMLPluginTypeKeys are the mutually exclusive plugin type specifiers: exactly
// one of these can appear in a plugins item. If any is present, the others are excluded.
var bufGenYAMLPluginTypeKeys = []string{"remote", "local", "protoc_builtin"}

// bufGenYAMLInputSourceKeys are the mutually exclusive input source type keys: exactly
// one must appear in each inputs item. If any is present, the others are excluded.
var bufGenYAMLInputSourceKeys = []string{
	"directory", "module", "proto_file", "git_repo", "tarball", "zip_archive",
	"binary_image", "json_image", "text_image", "yaml_image",
}

// bufGenYAMLGitRepoOnlyKeys are the keys only valid with git_repo inputs.
var bufGenYAMLGitRepoOnlyKeys = []string{
	"branch", "tag", "commit", "ref", "depth", "recurse_submodules",
}

// bufGenYAMLSubdirSources is the set of input source types for which subdir is valid.
var bufGenYAMLSubdirSources = map[string]bool{
	"git_repo": true, "tarball": true, "zip_archive": true,
}

// bufGenYAMLStripComponentsSources is the set of input source types for which
// strip_components is valid.
var bufGenYAMLStripComponentsSources = map[string]bool{
	"tarball": true, "zip_archive": true,
}

// bufGenYAMLCompressionSources is the set of input source types for which
// compression is valid.
var bufGenYAMLCompressionSources = map[string]bool{
	"tarball": true, "binary_image": true, "json_image": true, "text_image": true, "yaml_image": true,
}

// bufGenYAMLKeyItems returns completion items for map keys in the given section.
func bufGenYAMLKeyItems(section string, editRange protocol.Range, existingKeys map[string]bool) []protocol.CompletionItem {
	var keys []string
	var docs map[string]bufYAMLDoc
	switch section {
	case "plugins":
		keys, docs = bufGenYAMLPluginKeys, bufGenYAMLPluginDocs
		var pluginType string
		for _, k := range bufGenYAMLPluginTypeKeys {
			if existingKeys[k] {
				pluginType = k
				break
			}
		}
		existingKeys = bufGenYAMLExcludeMutuallyExclusive(existingKeys, bufGenYAMLPluginTypeKeys)
		if pluginType != "" {
			// existingKeys was cloned by bufGenYAMLExcludeMutuallyExclusive above.
			// Hide keys not applicable to the chosen plugin type.
			if pluginType != "protoc_builtin" {
				existingKeys["protoc_path"] = true
			}
			if pluginType != "remote" {
				existingKeys["revision"] = true
			}
			if pluginType == "remote" {
				existingKeys["strategy"] = true
			}
		}
	case "inputs":
		keys, docs = bufGenYAMLInputKeys, bufGenYAMLInputDocs
		var sourceType string
		for _, k := range bufGenYAMLInputSourceKeys {
			if existingKeys[k] {
				sourceType = k
				break
			}
		}
		existingKeys = bufGenYAMLExcludeMutuallyExclusive(existingKeys, bufGenYAMLInputSourceKeys)
		if sourceType != "" {
			// existingKeys was cloned by bufGenYAMLExcludeMutuallyExclusive above.
			// Hide keys not applicable to the chosen source type.
			if sourceType != "git_repo" {
				for _, k := range bufGenYAMLGitRepoOnlyKeys {
					existingKeys[k] = true
				}
			}
			if sourceType != "proto_file" {
				existingKeys["include_package_files"] = true
			}
			if !bufGenYAMLSubdirSources[sourceType] {
				existingKeys["subdir"] = true
			}
			if !bufGenYAMLStripComponentsSources[sourceType] {
				existingKeys["strip_components"] = true
			}
			if !bufGenYAMLCompressionSources[sourceType] {
				existingKeys["compression"] = true
			}
		}
		// tag and commit are mutually exclusive git revision specifiers.
		if existingKeys["tag"] {
			existingKeys = maps.Clone(existingKeys)
			existingKeys["commit"] = true
		} else if existingKeys["commit"] {
			existingKeys = maps.Clone(existingKeys)
			existingKeys["tag"] = true
		}
	case "managed":
		keys, docs = bufGenYAMLManagedKeys, bufGenYAMLManagedDocs
	case "disable", "override":
		keys, docs = bufGenYAMLManagedRuleKeys, bufGenYAMLManagedRuleDocs
		if section == "disable" {
			// value is only valid in override rules, not disable rules.
			existingKeys = maps.Clone(existingKeys)
			existingKeys["value"] = true
		}
		// file_option and field_option are mutually exclusive option target types.
		if existingKeys["file_option"] {
			existingKeys = maps.Clone(existingKeys)
			existingKeys["field_option"] = true
		} else if existingKeys["field_option"] {
			existingKeys = maps.Clone(existingKeys)
			existingKeys["file_option"] = true
		}
	default:
		keys, docs = bufGenYAMLTopLevelKeys, bufGenYAMLTopLevelDocs
	}
	return makeCompletionKeyItems(keys, docs, nil, editRange, existingKeys)
}

// bufGenYAMLExcludeMutuallyExclusive returns existingKeys with all keys in the group
// added when any one of them is already present, so makeCompletionKeyItems omits the
// whole group. Returns existingKeys unchanged when none of the group keys are set.
func bufGenYAMLExcludeMutuallyExclusive(existingKeys map[string]bool, group []string) map[string]bool {
	for _, k := range group {
		if existingKeys[k] {
			cloned := maps.Clone(existingKeys)
			for _, g := range group {
				cloned[g] = true
			}
			return cloned
		}
	}
	return existingKeys
}

// bufGenYAMLBuiltinPlugins lists the valid values for the protoc_builtin key,
// matching the set in bufconfig.ProtocProxyPluginNames.
var bufGenYAMLBuiltinPlugins = []string{
	"cpp", "csharp", "java", "js", "kotlin", "objc",
	"php", "python", "pyi", "rbs", "ruby", "rust",
}

// bufGenYAMLTopLevelKeys lists the top-level buf.gen.yaml keys in definition order.
var bufGenYAMLTopLevelKeys = []string{"version", "clean", "managed", "plugins", "inputs"}

// bufGenYAMLPluginKeys lists keys valid within a plugins[] item.
var bufGenYAMLPluginKeys = []string{
	"remote", "local", "protoc_builtin", "protoc_path", "out", "opt",
	"revision", "include_imports", "include_wkt", "strategy", "types", "exclude_types",
}

// bufGenYAMLInputKeys lists keys valid within an inputs[] item. The source type
// keys (the first group in bufGenYAMLInputSourceKeys) are listed first.
var bufGenYAMLInputKeys = append(
	bufGenYAMLInputSourceKeys,
	"types", "exclude_types", "paths", "exclude_paths", "include_package_files",
	"branch", "tag", "commit", "ref", "depth", "recurse_submodules",
	"subdir", "strip_components", "compression",
)

// bufGenYAMLManagedKeys lists keys valid within the managed: block.
var bufGenYAMLManagedKeys = []string{"enabled", "disable", "override"}

// bufGenYAMLManagedRuleKeys lists keys valid within managed.disable[] and managed.override[] items.
var bufGenYAMLManagedRuleKeys = []string{
	"file_option", "field_option", "module", "path", "field", "value",
}

// bufGenYAMLTypeItems returns completion items for fully-qualified proto type names
// (messages, enums, services) from the workspace. partialName is the text already
// typed; items whose name does not have partialName as a prefix are excluded.
// Returns nil when ws is nil or no symbols are indexed.
//
// Iterates each file's IR directly (rather than referenceableSymbols, which only
// holds messages and enums) so services are included.
func bufGenYAMLTypeItems(ws *workspace, partialName string, editRange protocol.Range) []protocol.CompletionItem {
	if ws == nil {
		return nil
	}
	seen := make(map[string]struct{})
	var items []protocol.CompletionItem
	for _, wsFile := range ws.PathToFile() {
		if wsFile.ir == nil {
			continue
		}
		symbols := wsFile.ir.Symbols()
		for i := range symbols.Len() {
			sym := symbols.At(i)
			switch sym.Kind() {
			case ir.SymbolKindMessage:
				if sym.AsType().IsMapEntry() {
					continue
				}
			case ir.SymbolKindEnum, ir.SymbolKindService:
			default:
				continue
			}
			name := string(sym.FullName())
			if partialName != "" && !strings.HasPrefix(name, partialName) {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			items = append(items, protocol.CompletionItem{
				Label: name,
				Kind:  protocol.CompletionItemKindValue,
				TextEdit: &protocol.TextEdit{
					Range:   editRange,
					NewText: name,
				},
			})
		}
	}
	slices.SortFunc(items, func(a, b protocol.CompletionItem) int {
		return strings.Compare(a.Label, b.Label)
	})
	return items
}

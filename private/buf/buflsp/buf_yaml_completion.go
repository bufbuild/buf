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
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"go.lsp.dev/protocol"
	"gopkg.in/yaml.v3"
)

// getBufYAMLCompletionItems returns completion items for a buf.yaml file at
// the given cursor position. dirPath is the directory containing the buf.yaml
// file, used to resolve filesystem path completions; pass "" to disable path
// completions (e.g. in tests).
func getBufYAMLCompletionItems(docNode *yaml.Node, text string, pos protocol.Position, dirPath string) []protocol.CompletionItem {
	lines, prefix, ok := bufYAMLParseCursor(text, pos)
	if !ok {
		return nil
	}
	cursorLine := int(pos.Line)
	tokenStart := bufYAMLTokenStart(prefix)
	editRange := bufYAMLEditRange(pos, tokenStart, lines[cursorLine])
	if valueKey := bufYAMLValueKey(prefix); valueKey != "" {
		if dirsOnly, ok := bufYAMLPathValueKeys[valueKey]; ok {
			return bufYAMLPathItems(dirPath, prefix[tokenStart:], editRange, dirsOnly)
		}
		return bufYAMLValueItems(valueKey, editRange)
	}
	if tokenStart < len(prefix) && prefix[tokenStart] == '-' {
		return nil
	}
	currentIndent := bufYAMLLeadingSpaces(prefix)
	section, parentSection := bufYAMLCursorPath(lines, cursorLine, currentIndent)
	if section == "" && currentIndent == 0 {
		if bareKey := bufYAMLBareParentKey(lines, cursorLine); bufYAMLBareParentMappingKeys[bareKey] {
			return bufYAMLPrependIndent(bufYAMLKeyItems(bareKey, editRange, nil))
		}
	}
	if section == "use" || section == "except" {
		return bufYAMLSequenceItems(parentSection, editRange)
	}
	if dirsOnly, ok := bufYAMLPathSequenceSections[section]; ok {
		return bufYAMLPathItems(dirPath, prefix[tokenStart:], editRange, dirsOnly)
	}
	// ignore_only maps each rule ID to a list of file/directory paths.
	// When the cursor is inside such a list, section is the rule ID itself
	// (e.g., "STANDARD", "FILE_NO_DELETE"), and the immediate parent is "ignore_only".
	if parentSection == "ignore_only" {
		return bufYAMLPathItems(dirPath, prefix[tokenStart:], editRange, false)
	}
	existingKeys := bufYAMLASTExistingKeys(docNode, section, parentSection, cursorLine)
	if existingKeys == nil {
		existingKeys = bufYAMLExistingKeys(lines, cursorLine, currentIndent)
	}
	if section == "ignore_only" {
		return bufYAMLIgnoreOnlyKeyItems(parentSection, editRange, existingKeys)
	}
	return bufYAMLKeyItems(section, editRange, existingKeys)
}

// bufYAMLPathValueKeys maps buf.yaml value keys whose values are filesystem
// paths to whether only directory entries should be offered.
var bufYAMLPathValueKeys = map[string]bool{
	"path": true,
}

// bufYAMLPathSequenceSections maps buf.yaml section names whose sequence items
// are filesystem paths to whether only directory entries should be offered.
// includes/excludes target subdirectories of a module; ignore accepts files
// or directories.
var bufYAMLPathSequenceSections = map[string]bool{
	"ignore":   false,
	"includes": true,
	"excludes": true,
}

// bufYAMLPathItems returns completion items for filesystem paths relative to
// dirPath. partialPath is the text the user has typed so far (may be empty).
// When dirsOnly is true, file entries are excluded.
// Returns nil when dirPath is empty or the search directory cannot be read.
func bufYAMLPathItems(dirPath, partialPath string, editRange protocol.Range, dirsOnly bool) []protocol.CompletionItem {
	if dirPath == "" {
		return nil
	}
	dir, base := path.Split(partialPath)
	searchDir := filepath.Join(dirPath, filepath.FromSlash(dir))
	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return nil
	}
	var items []protocol.CompletionItem
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if base != "" && !strings.HasPrefix(name, base) {
			continue
		}
		var text string
		var kind protocol.CompletionItemKind
		if entry.IsDir() {
			text = dir + name + "/"
			kind = protocol.CompletionItemKindFolder
		} else {
			if dirsOnly {
				continue
			}
			text = dir + name
			kind = protocol.CompletionItemKindFile
		}
		items = append(items, protocol.CompletionItem{
			Label: text,
			Kind:  kind,
			TextEdit: &protocol.TextEdit{
				Range:   editRange,
				NewText: text,
			},
		})
	}
	return items
}

// bufYAMLTokenStart returns the byte index in prefix where the current token begins.
func bufYAMLTokenStart(prefix string) int {
	tokenStart := len(prefix)
	for tokenStart > 0 && prefix[tokenStart-1] != ' ' && prefix[tokenStart-1] != '\t' {
		tokenStart--
	}
	return tokenStart
}

// bufYAMLLeadingSpaces returns the count of leading space/tab characters in s.
func bufYAMLLeadingSpaces(s string) int {
	leadingSpaces := 0
	for leadingSpaces < len(s) && (s[leadingSpaces] == ' ' || s[leadingSpaces] == '\t') {
		leadingSpaces++
	}
	return leadingSpaces
}

// bufYAMLExtractKey extracts the key from a trimmed "key:" or "key: value"
// line. Keys with embedded whitespace are invalid and return "".
func bufYAMLExtractKey(trimmed string) string {
	colonIdx := strings.IndexByte(trimmed, ':')
	if colonIdx <= 0 {
		return ""
	}
	key := trimmed[:colonIdx]
	if strings.ContainsAny(key, " \t") {
		return ""
	}
	return key
}

// bufYAMLValueKey returns the map key when the cursor is in a value position
// (prefix ends with "key:" or "key: <partial>"), or "" for key positions.
func bufYAMLValueKey(prefix string) string {
	stripped := strings.TrimPrefix(strings.TrimLeft(prefix, " \t"), "- ")
	return bufYAMLExtractKey(stripped)
}

// bufYAMLBareParentMappingKeys is the set of top-level buf.yaml keys whose
// value is a mapping (not a sequence or scalar). The bare-parent-key heuristic
// applies only to these keys: when cursor is at indent 0 right after "lint:"
// or "breaking:" with no children yet, we offer the key's children.
var bufYAMLBareParentMappingKeys = map[string]bool{
	"lint": true, "breaking": true,
}

// bufYAMLBareParentKey returns the name of the nearest preceding non-empty
// top-level bare mapping key (e.g. "breaking:") when no indented content
// exists between that line and cursorLine. Returns "" otherwise.
//
// This detects the pattern where the cursor is at indent 0 right after a
// bare parent key that has no children defined yet — a case where
// bufYAMLCursorPath cannot resolve a section because indent 0 < 0 is never
// true.
func bufYAMLBareParentKey(lines []string, cursorLine int) string {
	for i := cursorLine - 1; i >= 0; i-- {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			continue
		}
		// Any indented content between the cursor and the candidate parent means
		// the parent already has children; the cursor follows those children.
		if bufYAMLLeadingSpaces(line) > 0 {
			return ""
		}
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "-" || strings.HasPrefix(trimmed, "- ") {
			return ""
		}
		colonIdx := strings.IndexByte(trimmed, ':')
		if colonIdx <= 0 {
			return ""
		}
		// A key with an inline value (e.g. "version: v2") is not a bare parent.
		if rest := strings.TrimSpace(trimmed[colonIdx+1:]); rest != "" {
			return ""
		}
		return trimmed[:colonIdx]
	}
	return ""
}

// bufYAMLPrependIndent returns a copy of items with "  " prepended to each
// TextEdit.NewText, so child-key completions offered for a bare parent key
// (e.g. after "breaking:\n") are correctly indented when inserted.
func bufYAMLPrependIndent(items []protocol.CompletionItem) []protocol.CompletionItem {
	result := make([]protocol.CompletionItem, len(items))
	for i, item := range items {
		result[i] = item
		if item.TextEdit != nil {
			editCopy := *item.TextEdit
			editCopy.NewText = "  " + item.TextEdit.NewText
			result[i].TextEdit = &editCopy
		}
	}
	return result
}

// bufYAMLCursorPath determines the (section, parentSection) for the cursor
// position by making a single backward pass through lines from cursorLine.
// Returns ("", "") for a top-level position.
//
// List-item lines (trimmed content starts with "- " or equals "-") are skipped.
// targetIndent starts at currentIndent; each time a qualifying ancestor is found
// its indent becomes the new target, so the pass naturally finds both levels.
func bufYAMLCursorPath(lines []string, cursorLine, currentIndent int) (section, parentSection string) {
	targetIndent := currentIndent
	for lineIndex := cursorLine - 1; lineIndex >= 0; lineIndex-- {
		line := lines[lineIndex]
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := bufYAMLLeadingSpaces(line)
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "-" || strings.HasPrefix(trimmed, "- ") {
			continue
		}
		if indent < targetIndent {
			if section == "" {
				section = bufYAMLExtractKey(trimmed)
				targetIndent = indent
			} else {
				parentSection = bufYAMLExtractKey(trimmed)
				break
			}
		}
	}
	return section, parentSection
}

// bufYAMLSequenceItems returns completion items for lint.use, lint.except,
// breaking.use, or breaking.except sequences.
func bufYAMLSequenceItems(parentSection string, editRange protocol.Range) []protocol.CompletionItem {
	switch parentSection {
	case "lint":
		return makeCompletionValueItems(bufYAMLLintRuleValues, bufYAMLLintRuleDocs, editRange)
	case "breaking":
		return makeCompletionValueItems(bufYAMLBreakingRuleValues, bufYAMLBreakingRuleDocs, editRange)
	default:
		return nil
	}
}

// bufYAMLIgnoreOnlyKeyItems returns completion items for rule IDs used as
// mapping keys under ignore_only blocks. policies[*].ignore_only can suppress
// either lint or breaking rules, so both sets are offered there.
func bufYAMLIgnoreOnlyKeyItems(parentSection string, editRange protocol.Range, existingKeys map[string]bool) []protocol.CompletionItem {
	switch parentSection {
	case "lint":
		return makeCompletionKeyItems(bufYAMLLintRuleValues, bufYAMLLintRuleDocs, nil, editRange, existingKeys)
	case "breaking":
		return makeCompletionKeyItems(bufYAMLBreakingRuleValues, bufYAMLBreakingRuleDocs, nil, editRange, existingKeys)
	case "policies":
		// Lint and breaking rule names are disjoint, so the primary/fallback
		// lookup in makeCompletionKeyItems unambiguously routes each rule to
		// the right docs.
		return makeCompletionKeyItems(bufYAMLAllRuleValues, bufYAMLLintRuleDocs, bufYAMLBreakingRuleDocs, editRange, existingKeys)
	default:
		return nil
	}
}

// bufYAMLValueItems returns completion items for scalar values of known buf.yaml keys.
func bufYAMLValueItems(key string, editRange protocol.Range) []protocol.CompletionItem {
	var values []string
	switch key {
	case "version":
		values = []string{"v2", "v1", "v1beta1"}
	case "disallow_comment_ignores", "ignore_unstable_packages",
		"rpc_allow_same_request_response", "rpc_allow_google_protobuf_empty_requests",
		"rpc_allow_google_protobuf_empty_responses", "disable_builtin":
		values = []string{"true", "false"}
	default:
		return nil
	}
	return makeCompletionValueItems(values, nil, editRange)
}

// bufYAMLKeyItems returns completion items for mapping keys in the given section.
func bufYAMLKeyItems(section string, editRange protocol.Range, existingKeys map[string]bool) []protocol.CompletionItem {
	switch section {
	case "":
		return makeCompletionKeyItems(bufYAMLTopLevelKeysList, bufYAMLTopLevelDocs, nil, editRange, existingKeys)
	case "lint":
		return makeCompletionKeyItems(bufYAMLLintKeysList, bufYAMLLintDocs, nil, editRange, existingKeys)
	case "breaking":
		return makeCompletionKeyItems(bufYAMLBreakingKeysList, bufYAMLBreakingDocs, nil, editRange, existingKeys)
	case "modules":
		// bufYAMLModuleDocs covers path/name/includes/excludes; lint and breaking
		// fall back to top-level docs that contain their summaries.
		return makeCompletionKeyItems(bufYAMLModuleItemKeysList, bufYAMLModuleDocs, bufYAMLTopLevelDocs, editRange, existingKeys)
	case "plugins":
		return makeCompletionKeyItems(bufYAMLPluginItemKeysList, bufPolicyYAMLPluginDocs, nil, editRange, existingKeys)
	case "policies":
		return makeCompletionKeyItems(bufYAMLPolicyItemKeysList, bufYAMLPolicyItemDocs, nil, editRange, existingKeys)
	default:
		return nil
	}
}

var bufYAMLTopLevelKeysList = []string{
	"version", "name", "modules", "deps", "lint", "breaking", "plugins", "policies",
}

var bufYAMLModuleItemKeysList = []string{
	"path", "name", "includes", "excludes", "lint", "breaking",
}

var bufYAMLLintKeysList = []string{
	"use", "except", "ignore", "ignore_only",
	"disallow_comment_ignores", "enum_zero_value_suffix",
	"rpc_allow_same_request_response",
	"rpc_allow_google_protobuf_empty_requests",
	"rpc_allow_google_protobuf_empty_responses",
	"service_suffix", "disable_builtin",
}

var bufYAMLBreakingKeysList = []string{
	"use", "except", "ignore", "ignore_only", "ignore_unstable_packages", "disable_builtin",
}

var bufYAMLPluginItemKeysList = []string{"plugin", "options"}

var bufYAMLPolicyItemKeysList = []string{"policy", "ignore", "ignore_only"}

var bufYAMLPolicyItemDocs = map[string]bufYAMLDoc{
	"policy": {
		summary:   "A local path to a `buf.policy.yaml` file or a remote BSR policy reference (e.g. `buf.build/acme/my-policy`). The policy's lint and breaking rules are applied to the workspace.",
		valueType: "string",
		url:       bufYAMLDocsURL + "#policies",
	},
	"ignore": {
		summary:   "Files and directories excluded from this policy's rules. Paths are relative to `buf.yaml`.",
		valueType: "[]string",
		url:       bufYAMLDocsURL + "#policies",
	},
	"ignore_only": {
		summary:   "Excludes specific files or directories from particular rules in this policy. Maps each rule ID or category name to a list of file/directory paths.",
		valueType: "object",
		url:       bufYAMLDocsURL + "#policies",
	},
}

var bufYAMLLintRuleValues = bufYAMLSortedDocKeys(bufYAMLLintRuleDocs)

var bufYAMLBreakingRuleValues = bufYAMLSortedDocKeys(bufYAMLBreakingRuleDocs)

// bufYAMLAllRuleValues is the union of lint and breaking rule values, used for
// policies[*].ignore_only completions where either kind of rule can be suppressed.
var bufYAMLAllRuleValues = func() []string {
	all := slices.Concat(bufYAMLLintRuleValues, bufYAMLBreakingRuleValues)
	slices.Sort(all)
	return all
}()

func bufYAMLSortedDocKeys(docs map[string]bufYAMLDoc) []string {
	keys := make([]string, 0, len(docs))
	for k := range docs {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// bufYAMLASTExistingKeys navigates the parsed YAML AST using section and
// parentSection (from bufYAMLCursorPath) to find the mapping that contains
// the cursor, then returns the set of keys already present in that mapping.
// Returns nil when the cursor location cannot be resolved in the AST.
func bufYAMLASTExistingKeys(docNode *yaml.Node, section, parentSection string, cursorLine int) map[string]bool {
	if docNode == nil || docNode.Kind != yaml.DocumentNode || len(docNode.Content) == 0 {
		return nil
	}
	root := docNode.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil
	}
	var targetMapping *yaml.Node
	switch {
	case section == "":
		targetMapping = root
	case parentSection == "":
		val := yamlMappingFindValue(root, section)
		if val == nil {
			return nil
		}
		if val.Kind == yaml.SequenceNode {
			targetMapping = yamlSequenceFindItemAt(val, cursorLine)
		} else {
			targetMapping = val
		}
	default:
		parent := yamlMappingFindValue(root, parentSection)
		if parent == nil {
			return nil
		}
		var sectionOwner *yaml.Node
		if parent.Kind == yaml.SequenceNode {
			sectionOwner = yamlSequenceFindItemAt(parent, cursorLine)
			if sectionOwner == nil {
				return nil
			}
		} else {
			sectionOwner = parent
		}
		targetMapping = yamlMappingFindValue(sectionOwner, section)
		// If the section value is itself a sequence (e.g. managed.disable or
		// managed.override), locate the specific item containing the cursor.
		if targetMapping != nil && targetMapping.Kind == yaml.SequenceNode {
			targetMapping = yamlSequenceFindItemAt(targetMapping, cursorLine)
		}
	}
	if targetMapping == nil || targetMapping.Kind != yaml.MappingNode {
		return nil
	}
	existing := make(map[string]bool)
	for i := 0; i+1 < len(targetMapping.Content); i += 2 {
		if targetMapping.Content[i].Kind == yaml.ScalarNode {
			existing[targetMapping.Content[i].Value] = true
		}
	}
	return existing
}

// yamlMappingFindValue returns the value node for key in a YAML mapping node,
// or nil if the key is not present.
func yamlMappingFindValue(mapping *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Kind == yaml.ScalarNode && mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

// yamlSequenceFindItemAt returns the sequence item whose start line is closest
// to but not past cursorLine (0-indexed). yaml.Node.Line is 1-indexed.
func yamlSequenceFindItemAt(seq *yaml.Node, cursorLine int) *yaml.Node {
	yamlLine := cursorLine + 1
	var result *yaml.Node
	for _, item := range seq.Content {
		if item.Line <= yamlLine {
			result = item
		} else {
			break
		}
	}
	return result
}

// bufYAMLEditRange returns a protocol.Range spanning the full token at the
// cursor: from tokenStart to the end of the token (whitespace or ':').
func bufYAMLEditRange(pos protocol.Position, tokenStart int, line string) protocol.Range {
	end := int(pos.Character)
	for end < len(line) && line[end] != ' ' && line[end] != '\t' && line[end] != ':' {
		end++
	}
	return protocol.Range{
		Start: protocol.Position{Line: pos.Line, Character: uint32(tokenStart)},
		End:   protocol.Position{Line: pos.Line, Character: uint32(end)},
	}
}

// bufYAMLExistingKeys returns the set of mapping keys already present at
// currentIndent within the same section block as cursorLine. For top-level
// completions (currentIndent == 0) it scans the entire document.
func bufYAMLExistingKeys(lines []string, cursorLine, currentIndent int) map[string]bool {
	parentLine := -1
	parentIndent := -1
	if currentIndent > 0 {
		for i := cursorLine - 1; i >= 0; i-- {
			line := lines[i]
			if strings.TrimSpace(line) == "" {
				continue
			}
			trimmed := strings.TrimLeft(line, " \t")
			if trimmed == "-" || strings.HasPrefix(trimmed, "- ") {
				continue
			}
			if bufYAMLLeadingSpaces(line) < currentIndent {
				parentLine = i
				parentIndent = bufYAMLLeadingSpaces(line)
				break
			}
		}
		if parentLine == -1 {
			return nil
		}
	}
	existing := map[string]bool{}
	for i := parentLine + 1; i < len(lines); i++ {
		if i == cursorLine {
			continue
		}
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := bufYAMLLeadingSpaces(line)
		if parentIndent >= 0 && indent <= parentIndent {
			break
		}
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "-" {
			continue
		}
		if rest, ok := strings.CutPrefix(trimmed, "- "); ok {
			// An inline sequence item "  - key: val" has its key at effective indent
			// indent+2. When that matches currentIndent, include the key so it is
			// treated as already-present.
			if indent+2 == currentIndent {
				if key := bufYAMLExtractKey(rest); key != "" {
					existing[key] = true
				}
			}
			continue
		}
		if indent != currentIndent {
			continue
		}
		if key := bufYAMLExtractKey(trimmed); key != "" {
			existing[key] = true
		}
	}
	return existing
}

// bufYAMLParseCursor splits text into lines and returns the prefix up to the
// cursor position. Returns ok=false if pos.Line is out of range.
func bufYAMLParseCursor(text string, pos protocol.Position) (lines []string, prefix string, ok bool) {
	lines = strings.Split(text, "\n")
	lineIdx := int(pos.Line)
	if lineIdx >= len(lines) {
		return nil, "", false
	}
	charIdx := int(pos.Character)
	currentLine := lines[lineIdx]
	if charIdx > len(currentLine) {
		charIdx = len(currentLine)
	}
	return lines, currentLine[:charIdx], true
}

// makeCompletionValueItems builds CompletionItemKindValue items for the given values.
func makeCompletionValueItems(values []string, docs map[string]bufYAMLDoc, editRange protocol.Range) []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, len(values))
	for idx, value := range values {
		item := protocol.CompletionItem{
			Label: value,
			Kind:  protocol.CompletionItemKindValue,
			TextEdit: &protocol.TextEdit{
				Range:   editRange,
				NewText: value,
			},
		}
		if docs != nil {
			if doc, ok := docs[value]; ok {
				item.Documentation = bufYAMLDocMarkup(doc)
			}
		}
		items[idx] = item
	}
	return items
}

// makeCompletionKeyItems builds CompletionItemKindField items for the given keys,
// with ": " appended to each TextEdit. Documentation comes from primaryDocs,
// falling back to fallbackDocs when no entry is found there.
func makeCompletionKeyItems(keys []string, primaryDocs, fallbackDocs map[string]bufYAMLDoc, editRange protocol.Range, existingKeys map[string]bool) []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0, len(keys))
	for _, key := range keys {
		if existingKeys[key] {
			continue
		}
		item := protocol.CompletionItem{
			Label: key,
			Kind:  protocol.CompletionItemKindField,
			TextEdit: &protocol.TextEdit{
				Range:   editRange,
				NewText: key + ": ",
			},
		}
		var doc bufYAMLDoc
		var ok bool
		if doc, ok = primaryDocs[key]; !ok && fallbackDocs != nil {
			doc, ok = fallbackDocs[key]
		}
		if ok {
			item.Detail = doc.valueType
			item.Documentation = bufYAMLDocMarkup(doc)
		}
		items = append(items, item)
	}
	return items
}

// bufYAMLDocMarkup returns a MarkupContent for the given doc.
func bufYAMLDocMarkup(doc bufYAMLDoc) protocol.MarkupContent {
	value := doc.summary
	if doc.url != "" {
		value += "\n\n[Documentation](" + doc.url + ")"
	}
	return protocol.MarkupContent{
		Kind:  protocol.Markdown,
		Value: value,
	}
}

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
	"go.lsp.dev/protocol"
	"gopkg.in/yaml.v3"
)

// getBufPolicyYAMLCompletionItems returns completion items for a buf.policy.yaml
// file at the given cursor position.
func getBufPolicyYAMLCompletionItems(docNode *yaml.Node, text string, pos protocol.Position) []protocol.CompletionItem {
	lines, prefix, ok := bufYAMLParseCursor(text, pos)
	if !ok {
		return nil
	}
	cursorLine := int(pos.Line)
	tokenStart := bufYAMLTokenStart(prefix)
	editRange := bufYAMLEditRange(pos, tokenStart, lines[cursorLine])
	if valueKey := bufYAMLValueKey(prefix); valueKey != "" {
		return bufPolicyYAMLValueItems(valueKey, editRange)
	}
	if tokenStart < len(prefix) && prefix[tokenStart] == '-' {
		return nil
	}
	currentIndent := bufYAMLLeadingSpaces(prefix)
	section, parentSection := bufYAMLCursorPath(lines, cursorLine, currentIndent)
	if section == "" && currentIndent == 0 {
		if bareKey := bufYAMLBareParentKey(lines, cursorLine); bufPolicyYAMLBareParentMappingKeys[bareKey] {
			return bufYAMLPrependIndent(bufPolicyYAMLKeyItems(bareKey, editRange, nil))
		}
	}
	if section == "use" || section == "except" {
		return bufYAMLSequenceItems(parentSection, editRange)
	}
	existingKeys := bufYAMLASTExistingKeys(docNode, section, parentSection, cursorLine)
	if existingKeys == nil {
		existingKeys = bufYAMLExistingKeys(lines, cursorLine, currentIndent)
	}
	return bufPolicyYAMLKeyItems(section, editRange, existingKeys)
}

// bufPolicyYAMLValueItems returns completion items for scalar values of known
// buf.policy.yaml keys.
func bufPolicyYAMLValueItems(key string, editRange protocol.Range) []protocol.CompletionItem {
	var values []string
	switch key {
	case "version":
		values = []string{"v2"}
	case "ignore_unstable_packages",
		"rpc_allow_same_request_response", "rpc_allow_google_protobuf_empty_requests",
		"rpc_allow_google_protobuf_empty_responses", "disable_builtin":
		values = []string{"true", "false"}
	default:
		return nil
	}
	return makeCompletionValueItems(values, nil, editRange)
}

// bufPolicyYAMLKeyItems returns completion items for mapping keys in the given section.
func bufPolicyYAMLKeyItems(section string, editRange protocol.Range, existingKeys map[string]bool) []protocol.CompletionItem {
	var keys []string
	var docs map[string]bufYAMLDoc
	switch section {
	case "":
		keys, docs = bufPolicyYAMLTopLevelKeysList, bufPolicyYAMLTopLevelDocs
	case "lint":
		keys, docs = bufPolicyYAMLLintKeysList, bufPolicyYAMLLintDocs
	case "breaking":
		keys, docs = bufPolicyYAMLBreakingKeysList, bufPolicyYAMLBreakingDocs
	case "plugins":
		keys, docs = bufYAMLPluginItemKeysList, bufPolicyYAMLPluginDocs
	default:
		return nil
	}
	return makeCompletionKeyItems(keys, docs, nil, editRange, existingKeys)
}

// bufPolicyYAMLBareParentMappingKeys is the set of top-level buf.policy.yaml
// keys whose value is a mapping. The bare-parent-key heuristic applies only to
// these: lint and breaking are mappings; plugins is a sequence.
var bufPolicyYAMLBareParentMappingKeys = map[string]bool{
	"lint": true, "breaking": true,
}

// bufPolicyYAMLTopLevelKeysList lists the top-level buf.policy.yaml keys.
var bufPolicyYAMLTopLevelKeysList = []string{"version", "name", "lint", "breaking", "plugins"}

// bufPolicyYAMLLintKeysList lists keys valid within the lint: block of buf.policy.yaml.
// This is a subset of buf.yaml lint keys (no ignore, ignore_only, or disallow_comment_ignores).
var bufPolicyYAMLLintKeysList = []string{
	"use", "except",
	"enum_zero_value_suffix",
	"rpc_allow_same_request_response",
	"rpc_allow_google_protobuf_empty_requests",
	"rpc_allow_google_protobuf_empty_responses",
	"service_suffix", "disable_builtin",
}

// bufPolicyYAMLBreakingKeysList lists keys valid within the breaking: block of buf.policy.yaml.
// This is a subset of buf.yaml breaking keys (no ignore or ignore_only).
var bufPolicyYAMLBreakingKeysList = []string{
	"use", "except", "ignore_unstable_packages", "disable_builtin",
}

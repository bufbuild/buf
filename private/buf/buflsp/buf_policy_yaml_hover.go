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

const bufPolicyYAMLDocsURL = "https://buf.build/docs/configuration/v2/buf-policy-yaml/"

// bufPolicyYAMLTopLevelDocs maps top-level buf.policy.yaml keys to their documentation.
var bufPolicyYAMLTopLevelDocs = map[string]bufYAMLDoc{
	"version":  {summary: "Defines the configuration format version. Must be `v2`.", url: bufPolicyYAMLDocsURL + "#version"},
	"name":     {summary: "A Buf Schema Registry path (e.g. `buf.build/acme/my-policy`) that uniquely identifies this policy. Setting a name associates the file with a BSR repository for publishing commits and label history.", url: bufPolicyYAMLDocsURL + "#name"},
	"lint":     {summary: "Configures lint rules for this policy. These settings are applied to workspaces that reference this policy. If unspecified, the `STANDARD` rule category is used.", url: bufPolicyYAMLDocsURL + "#lint"},
	"breaking": {summary: "Configures breaking change detection rules for this policy. These settings are applied to workspaces that reference this policy. If unspecified, the `FILE` rule category is used.", url: bufPolicyYAMLDocsURL + "#breaking"},
	"plugins":  {summary: "Lists custom lint and breaking change plugins that provide additional rules for this policy. Each entry specifies a local binary, a path, or a remote BSR plugin.", url: bufPolicyYAMLDocsURL + "#plugins"},
}

// bufPolicyYAMLLintDocs maps lint sub-keys supported in buf.policy.yaml.
// This is a subset of bufYAMLLintDocs (no ignore, ignore_only, or disallow_comment_ignores).
var bufPolicyYAMLLintDocs = map[string]bufYAMLDoc{
	"use":                             {summary: "Lists lint rule categories and/or specific rule IDs to enable. Category names (e.g. `MINIMAL`, `BASIC`, `STANDARD`) select a predefined set of rules.", url: bufYAMLLintRulesURL},
	"except":                          {summary: "Removes specific rules or categories from the active lint rule set. Rules listed here are excluded even if they are part of a category in `use`.", url: bufPolicyYAMLDocsURL + "#lint"},
	"enum_zero_value_suffix":          {summary: "Sets the required suffix for zero-value enum entries, enforced by the `ENUM_ZERO_VALUE_SUFFIX` rule. Defaults to `_UNSPECIFIED`.", url: bufPolicyYAMLDocsURL + "#lint"},
	"service_suffix":                  {summary: "Sets the required suffix for service names, enforced by the `SERVICE_SUFFIX` rule. Defaults to `Service`.", url: bufPolicyYAMLDocsURL + "#lint"},
	"rpc_allow_same_request_response": {summary: "When `true`, permits using the same message type for both the request and response of an RPC. Defaults to `false`.", url: bufPolicyYAMLDocsURL + "#lint"},
	"rpc_allow_google_protobuf_empty_requests":  {summary: "When `true`, allows RPC methods to use `google.protobuf.Empty` as the request type. Defaults to `false`.", url: bufPolicyYAMLDocsURL + "#lint"},
	"rpc_allow_google_protobuf_empty_responses": {summary: "When `true`, allows RPC methods to use `google.protobuf.Empty` as the response type. Defaults to `false`.", url: bufPolicyYAMLDocsURL + "#lint"},
	"disable_builtin":                           {summary: "When `true`, disables all built-in lint rules. Use this when relying entirely on custom plugin-provided rules. Defaults to `false`.", url: bufPolicyYAMLDocsURL + "#lint"},
}

// bufPolicyYAMLBreakingDocs maps breaking sub-keys supported in buf.policy.yaml.
// This is a subset of bufYAMLBreakingDocs (no ignore or ignore_only).
var bufPolicyYAMLBreakingDocs = map[string]bufYAMLDoc{
	"use":                      {summary: "Lists breaking change rule categories and/or specific rule IDs to enable. Category names (`FILE`, `PACKAGE`, `WIRE_JSON`, `WIRE`) select a predefined set of rules.", url: bufYAMLBreakingRulesURL},
	"except":                   {summary: "Removes specific rules or categories from the active breaking change rule set. Using `except` is generally discouraged.", url: bufPolicyYAMLDocsURL + "#breaking"},
	"ignore_unstable_packages": {summary: "When `true`, ignores packages matching unstable version patterns such as `v1alpha1`, `v1beta1`, or `v1test`. Defaults to `false`.", url: bufPolicyYAMLDocsURL + "#breaking"},
	"disable_builtin":          {summary: "When `true`, disables all built-in breaking change rules. Use this when relying entirely on custom plugin-provided rules. Defaults to `false`.", url: bufPolicyYAMLDocsURL + "#breaking"},
}

// bufPolicyYAMLPluginDocs maps plugin entry sub-keys for buf.policy.yaml.
var bufPolicyYAMLPluginDocs = map[string]bufYAMLDoc{
	"plugin":  {summary: "Plugin location: a binary name on `$PATH`, a local file path, or a remote BSR plugin reference (e.g. `buf.build/acme/my-plugin`).", url: bufPolicyYAMLDocsURL + "#plugins"},
	"options": {summary: "Key-value pairs passed to the plugin to customize its behavior.", url: bufPolicyYAMLDocsURL + "#plugins"},
}

// bufPolicyYAMLHover searches the parsed buf.policy.yaml document for hover
// information at the given position and returns a Hover response, or nil.
func bufPolicyYAMLHover(docNode *yaml.Node, pos protocol.Position) *protocol.Hover {
	if docNode.Kind != yaml.DocumentNode || len(docNode.Content) == 0 {
		return nil
	}
	mapping := docNode.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return nil
	}
	return searchBufPolicyYAMLMappingForHover(mapping, pos, nil)
}

func searchBufPolicyYAMLMappingForHover(node *yaml.Node, pos protocol.Position, parentPath []string) *protocol.Hover {
	for i := 0; i+1 < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]

		currentPath := append(parentPath[:len(parentPath):len(parentPath)], keyNode.Value)

		if yamlNodeContainsPosition(keyNode, pos) {
			return bufPolicyYAMLHoverForKeyPath(currentPath, yamlNodeRange(keyNode))
		}

		switch valNode.Kind {
		case yaml.MappingNode:
			if h := searchBufPolicyYAMLMappingForHover(valNode, pos, currentPath); h != nil {
				return h
			}
		case yaml.SequenceNode:
			if h := searchBufPolicyYAMLSequenceForHover(valNode, pos, currentPath); h != nil {
				return h
			}
		}
	}
	return nil
}

func searchBufPolicyYAMLSequenceForHover(node *yaml.Node, pos protocol.Position, parentPath []string) *protocol.Hover {
	for _, item := range node.Content {
		switch item.Kind {
		case yaml.ScalarNode:
			// Rule/category names in lint.use and lint.except.
			if yamlNodeContainsPosition(item, pos) {
				return bufPolicyYAMLHoverForSequenceValue(item.Value, parentPath, yamlNodeRange(item))
			}
		case yaml.MappingNode:
			if h := searchBufPolicyYAMLMappingForHover(item, pos, parentPath); h != nil {
				return h
			}
		}
	}
	return nil
}

// bufPolicyYAMLHoverForSequenceValue returns hover for rule/category names in
// lint.use and lint.except sequences.
func bufPolicyYAMLHoverForSequenceValue(value string, parentPath []string, nodeRange protocol.Range) *protocol.Hover {
	if len(parentPath) < 2 {
		return nil
	}
	field := parentPath[len(parentPath)-1]
	if field != "use" && field != "except" {
		return nil
	}
	section := parentPath[len(parentPath)-2]
	switch section {
	case "lint":
		if doc, ok := bufYAMLLintRuleDocs[value]; ok {
			return makeBufYAMLHover(value, doc, nodeRange)
		}
	case "breaking":
		if doc, ok := bufYAMLBreakingRuleDocs[value]; ok {
			return makeBufYAMLHover(value, doc, nodeRange)
		}
	}
	return nil
}

func bufPolicyYAMLHoverForKeyPath(path []string, nodeRange protocol.Range) *protocol.Hover {
	switch len(path) {
	case 1:
		if doc, ok := bufPolicyYAMLTopLevelDocs[path[0]]; ok {
			return makeBufYAMLHover(path[0], doc, nodeRange)
		}
	case 2:
		switch path[0] {
		case "lint":
			if doc, ok := bufPolicyYAMLLintDocs[path[1]]; ok {
				return makeBufYAMLHover("lint."+path[1], doc, nodeRange)
			}
		case "breaking":
			if doc, ok := bufPolicyYAMLBreakingDocs[path[1]]; ok {
				return makeBufYAMLHover("breaking."+path[1], doc, nodeRange)
			}
		case "plugins":
			if doc, ok := bufPolicyYAMLPluginDocs[path[1]]; ok {
				return makeBufYAMLHover(path[1], doc, nodeRange)
			}
		}
	}
	return nil
}

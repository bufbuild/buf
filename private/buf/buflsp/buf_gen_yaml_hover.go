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

const bufGenYAMLDocsURL = "https://buf.build/docs/configuration/v2/buf-gen-yaml/"

// bufGenYAMLTopLevelDocs maps top-level buf.gen.yaml keys to their documentation.
var bufGenYAMLTopLevelDocs = map[string]bufYAMLDoc{
	"version": {summary: "Defines the configuration format version. Must be `v2`, `v1` or `v1beta1`.", valueType: "string", url: bufGenYAMLDocsURL + "#version"},
	"clean":   {summary: "When `true`, removes all output directories, zip files, and jar files specified in `plugins[].out` before running generation. Defaults to `false`.", valueType: "bool", url: bufGenYAMLDocsURL + "#clean"},
	"managed": {summary: "Configures managed mode, which automatically sets Protobuf file and field options to sensible defaults for each target language.", valueType: "object", url: bufGenYAMLDocsURL + "#managed"},
	"plugins": {summary: "Defines the code generation plugins to run. At least one plugin must be specified. Each entry must specify exactly one of `remote`, `local`, or `protoc_builtin`.", valueType: "[]object", url: bufGenYAMLDocsURL + "#plugins"},
	"inputs":  {summary: "Specifies the Protobuf sources to generate code from. Each entry defines one input and optional type/path filters. If omitted, `buf generate` uses the current directory.", valueType: "[]object", url: bufGenYAMLDocsURL + "#inputs"},
}

// bufGenYAMLPluginDocs maps plugin entry sub-keys to their documentation.
var bufGenYAMLPluginDocs = map[string]bufYAMLDoc{
	"remote":          {summary: "Remote BSR plugin reference in the format `buf.build/<owner>/<plugin>` or `buf.build/<owner>/<plugin>:<version>`. Mutually exclusive with `local` and `protoc_builtin`.", valueType: "string", url: bufGenYAMLDocsURL + "#plugins"},
	"local":           {summary: "Path to a local plugin binary, or a list of `[binary, arg, ...]` for a plugin with fixed arguments. Mutually exclusive with `remote` and `protoc_builtin`.", valueType: "string", url: bufGenYAMLDocsURL + "#plugins"},
	"protoc_builtin":  {summary: "Built-in `protoc` generator name without the `protoc-gen-` prefix (e.g. `java`, `python`, `cpp`). Mutually exclusive with `remote` and `local`.", valueType: "string", url: bufGenYAMLDocsURL + "#plugins"},
	"protoc_path":     {summary: "Path to the `protoc` binary, or a list of `[path, arg, ...]`. Only valid with `protoc_builtin`.", valueType: "string", url: bufGenYAMLDocsURL + "#plugins"},
	"out":             {summary: "Output directory for generated files. The directory is created if it does not exist.", valueType: "string", url: bufGenYAMLDocsURL + "#out"},
	"opt":             {summary: "Plugin options passed as `--<plugin>_opt` flags. Can be a single string or a list of strings.", valueType: "string", url: bufGenYAMLDocsURL + "#opt"},
	"revision":        {summary: "Plugin revision number. Only valid with `remote`.", valueType: "integer", url: bufGenYAMLDocsURL + "#plugins"},
	"include_imports": {summary: "When `true`, generates code for all files imported by the input, excluding Well-Known Types. Defaults to `false`.", valueType: "bool", url: bufGenYAMLDocsURL + "#include_imports"},
	"include_wkt":     {summary: "When `true`, also generates code for Well-Known Types. Requires `include_imports: true`. Defaults to `false`.", valueType: "bool", url: bufGenYAMLDocsURL + "#include_wkt"},
	"strategy":        {summary: "Plugin invocation strategy: `directory` (invoke once per directory, default for most plugins) or `all` (invoke once with all files). Only valid with `local` and `protoc_builtin`.", valueType: "string", url: bufGenYAMLDocsURL + "#strategy"},
	"types":           {summary: "Generate code only for the listed fully-qualified type names (messages, enums, services). An empty list means all types.", valueType: "[]string", url: bufGenYAMLDocsURL + "#types"},
	"exclude_types":   {summary: "Exclude the listed fully-qualified type names from code generation.", valueType: "[]string", url: bufGenYAMLDocsURL + "#exclude-types"},
}

// bufGenYAMLManagedDocs maps managed-mode sub-keys to their documentation.
var bufGenYAMLManagedDocs = map[string]bufYAMLDoc{
	"enabled":  {summary: "Enables managed mode globally. Must be `true` for other `managed` settings to take effect. Defaults to `false`.", valueType: "bool", url: bufGenYAMLDocsURL + "#enabled"},
	"disable":  {summary: "Rules that exclude specific file or field options from managed mode handling. Each entry may target a `file_option` or `field_option`, optionally restricted to a `module`, `path`, or `field`.", valueType: "[]object", url: bufGenYAMLDocsURL + "#disable"},
	"override": {summary: "Rules that set specific file or field option values, overriding managed mode defaults. Each entry must specify exactly one of `file_option` or `field_option`, plus a `value`.", valueType: "[]object", url: bufGenYAMLDocsURL + "#override"},
}

// bufGenYAMLManagedRuleDocs maps keys shared by managed.disable and managed.override entries.
var bufGenYAMLManagedRuleDocs = map[string]bufYAMLDoc{
	"file_option":  {summary: "File-level Protobuf option to target (e.g. `java_package`, `go_package_prefix`, `csharp_namespace`). Mutually exclusive with `field_option`.", valueType: "string", url: bufGenYAMLDocsURL + "#disable"},
	"field_option": {summary: "Field-level Protobuf option to target (e.g. `jstype`). Mutually exclusive with `file_option`.", valueType: "string", url: bufGenYAMLDocsURL + "#disable"},
	"module":       {summary: "Restrict this rule to a specific BSR module (e.g. `buf.build/acme/petapis`).", valueType: "string", url: bufGenYAMLDocsURL + "#disable"},
	"path":         {summary: "Restrict this rule to a specific `.proto` file path, relative to the module root.", valueType: "string", url: bufGenYAMLDocsURL + "#disable"},
	"field":        {summary: "Restrict this rule to a specific fully-qualified field name (e.g. `acme.v1.Foo.bar`). Only valid with `field_option`.", valueType: "string", url: bufGenYAMLDocsURL + "#disable"},
	"value":        {summary: "The value to set for the option. Type depends on the option: string, boolean, or enum value name (e.g. `SPEED` for `optimize_for`). Only valid in `override` rules.", valueType: "string", url: bufGenYAMLDocsURL + "#override"},
}

// bufGenYAMLInputDocs maps input entry keys to their documentation.
// This covers both the input type keys and the common filtering/option keys.
var bufGenYAMLInputDocs = map[string]bufYAMLDoc{
	// Input type keys — exactly one must be set per input entry.
	"directory":    {summary: "Local directory path containing `.proto` files.", valueType: "string", url: bufGenYAMLDocsURL + "#directory"},
	"module":       {summary: "Remote BSR module reference (e.g. `buf.build/acme/petapis` or `buf.build/acme/petapis:v1.0.0`).", valueType: "string", url: bufGenYAMLDocsURL + "#module"},
	"proto_file":   {summary: "Path to a single `.proto` file.", valueType: "string", url: bufGenYAMLDocsURL + "#proto_file"},
	"git_repo":     {summary: "Git repository URL or local path. Use `branch`, `tag`, `commit`, or `ref` to select a specific revision.", valueType: "string", url: bufGenYAMLDocsURL + "#git_repo"},
	"tarball":      {summary: "Path or URL to a `.tar`, `.tar.gz`, or similar archive.", valueType: "string", url: bufGenYAMLDocsURL + "#tarball"},
	"zip_archive":  {summary: "Path or URL to a `.zip` archive.", valueType: "string", url: bufGenYAMLDocsURL + "#zip_archive"},
	"binary_image": {summary: "Path to a Buf binary image file (a protobuf-encoded `FileDescriptorSet`).", valueType: "string", url: bufGenYAMLDocsURL + "#inputs"},
	"json_image":   {summary: "Path to a Buf image file in JSON format.", valueType: "string", url: bufGenYAMLDocsURL + "#inputs"},
	"text_image":   {summary: "Path to a Buf image file in protobuf text format.", valueType: "string", url: bufGenYAMLDocsURL + "#inputs"},
	"yaml_image":   {summary: "Path to a Buf image file in YAML format.", valueType: "string", url: bufGenYAMLDocsURL + "#inputs"},
	// Common input options.
	"types":         {summary: "Include only the listed fully-qualified type names (messages, enums, services) in code generation.", valueType: "[]string", url: bufGenYAMLDocsURL + "#types"},
	"exclude_types": {summary: "Exclude the listed fully-qualified type names from code generation.", valueType: "[]string", url: bufGenYAMLDocsURL + "#exclude-types"},
	"paths":         {summary: "Include only `.proto` files at the listed relative paths.", valueType: "[]string", url: bufGenYAMLDocsURL + "#inputs"},
	"exclude_paths": {summary: "Exclude `.proto` files at the listed relative paths.", valueType: "[]string", url: bufGenYAMLDocsURL + "#inputs"},
	// Input-specific options.
	"include_package_files": {summary: "When `true`, includes all `.proto` files in the same package as the specified file. Only valid with `proto_file`.", valueType: "bool", url: bufGenYAMLDocsURL + "#proto_file"},
	"branch":                {summary: "Git branch to check out. Only valid with `git_repo`.", valueType: "string", url: bufGenYAMLDocsURL + "#git_repo"},
	"tag":                   {summary: "Git tag to check out. Only valid with `git_repo`. Mutually exclusive with `commit`.", valueType: "string", url: bufGenYAMLDocsURL + "#git_repo"},
	"commit":                {summary: "Full Git commit hash to check out. Only valid with `git_repo`. Mutually exclusive with `tag`.", valueType: "string", url: bufGenYAMLDocsURL + "#git_repo"},
	"ref":                   {summary: "Git ref to check out. Only valid with `git_repo`.", valueType: "string", url: bufGenYAMLDocsURL + "#git_repo"},
	"depth":                 {summary: "Shallow clone depth for the Git repository. Only valid with `git_repo`.", valueType: "integer", url: bufGenYAMLDocsURL + "#git_repo"},
	"recurse_submodules":    {summary: "When `true`, recursively clones submodules. Only valid with `git_repo`.", valueType: "bool", url: bufGenYAMLDocsURL + "#git_repo"},
	"subdir":                {summary: "Subdirectory within the source to use as the root for `.proto` file discovery. Valid with `git_repo`, `tarball`, and `zip_archive`.", valueType: "string", url: bufGenYAMLDocsURL + "#subdir"},
	"strip_components":      {summary: "Number of leading directory path components to strip from archive entries. Valid with `tarball` and `zip_archive`.", valueType: "integer", url: bufGenYAMLDocsURL + "#strip_components"},
	"compression":           {summary: "Compression format of the archive (e.g. `gzip`, `bzip2`). Valid with `tarball`, `binary_image`, `json_image`, `text_image`, and `yaml_image`.", valueType: "string", url: bufGenYAMLDocsURL + "#compression"},
}

// bufGenYAMLHover searches the parsed buf.gen.yaml document for hover
// information at the given position and returns a Hover response, or nil if
// the position does not correspond to a known field.
func bufGenYAMLHover(docNode *yaml.Node, pos protocol.Position) *protocol.Hover {
	if docNode.Kind != yaml.DocumentNode || len(docNode.Content) == 0 {
		return nil
	}
	mapping := docNode.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return nil
	}
	return searchBufGenYAMLMappingForHover(mapping, pos, nil)
}

// searchBufGenYAMLMappingForHover recursively searches a YAML mapping node
// for the cursor position and returns hover info when a known key is found.
func searchBufGenYAMLMappingForHover(node *yaml.Node, pos protocol.Position, parentPath []string) *protocol.Hover {
	for i := 0; i+1 < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]

		currentPath := append(parentPath[:len(parentPath):len(parentPath)], keyNode.Value)

		if yamlNodeContainsPosition(keyNode, pos) {
			return bufGenYAMLHoverForKeyPath(currentPath, yamlNodeRange(keyNode))
		}

		switch valNode.Kind {
		case yaml.MappingNode:
			if h := searchBufGenYAMLMappingForHover(valNode, pos, currentPath); h != nil {
				return h
			}
		case yaml.SequenceNode:
			if h := searchBufGenYAMLSequenceForHover(valNode, pos, currentPath); h != nil {
				return h
			}
		}
	}
	return nil
}

// searchBufGenYAMLSequenceForHover searches a YAML sequence node for the
// cursor position. Scalar items have no hover; mapping items are recursed into.
func searchBufGenYAMLSequenceForHover(node *yaml.Node, pos protocol.Position, parentPath []string) *protocol.Hover {
	for _, item := range node.Content {
		if item.Kind == yaml.MappingNode {
			if h := searchBufGenYAMLMappingForHover(item, pos, parentPath); h != nil {
				return h
			}
		}
	}
	return nil
}

// bufGenYAMLHoverForKeyPath returns hover documentation for a buf.gen.yaml
// key identified by its path (e.g. ["plugins", "out"]).
func bufGenYAMLHoverForKeyPath(path []string, nodeRange protocol.Range) *protocol.Hover {
	switch len(path) {
	case 1:
		if doc, ok := bufGenYAMLTopLevelDocs[path[0]]; ok {
			return makeBufYAMLHover(path[0], doc, nodeRange)
		}
	case 2:
		switch path[0] {
		case "plugins":
			if doc, ok := bufGenYAMLPluginDocs[path[1]]; ok {
				return makeBufYAMLHover(path[1], doc, nodeRange)
			}
		case "inputs":
			if doc, ok := bufGenYAMLInputDocs[path[1]]; ok {
				return makeBufYAMLHover(path[1], doc, nodeRange)
			}
		case "managed":
			if doc, ok := bufGenYAMLManagedDocs[path[1]]; ok {
				return makeBufYAMLHover("managed."+path[1], doc, nodeRange)
			}
		}
	case 3:
		// Keys inside managed.disable or managed.override rule entries.
		if path[0] == "managed" && (path[1] == "disable" || path[1] == "override") {
			if doc, ok := bufGenYAMLManagedRuleDocs[path[2]]; ok {
				return makeBufYAMLHover(path[2], doc, nodeRange)
			}
		}
	}
	return nil
}

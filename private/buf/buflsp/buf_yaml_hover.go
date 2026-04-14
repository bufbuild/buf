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
	"fmt"
	"strings"

	"go.lsp.dev/protocol"
	"gopkg.in/yaml.v3"
)

const (
	bufYAMLDocsURL          = "https://buf.build/docs/configuration/v2/buf-yaml/"
	bufYAMLLintRulesURL     = "https://buf.build/docs/lint/rules/"
	bufYAMLBreakingRulesURL = "https://buf.build/docs/breaking/rules/"
)

// bufYAMLDoc holds hover documentation for a single buf.yaml field or rule.
type bufYAMLDoc struct {
	// summary is the markdown body text describing the field or rule.
	summary string
	// url is the documentation page URL, shown as a [Documentation](url) link.
	url string
}

// bufYAMLTopLevelDocs maps top-level buf.yaml keys to their documentation.
var bufYAMLTopLevelDocs = map[string]bufYAMLDoc{
	"version":  {summary: "Defines the configuration format version. Must be `v2`, `v1` or `v1beta1`.", url: bufYAMLDocsURL + "#version"},
	"modules":  {summary: "Defines the Protobuf modules in the workspace. Each entry specifies a directory of Protobuf files with optional per-module lint and breaking settings.", url: bufYAMLDocsURL + "#modules"},
	"deps":     {summary: "Declares module dependencies hosted on the Buf Schema Registry. Dependencies can pin a specific commit or label using the format `buf.build/owner/module:reference`. Pinned versions are stored in [`buf.lock`](https://buf.build/docs/configuration/v2/buf-lock/).", url: bufYAMLDocsURL + "#deps"},
	"lint":     {summary: "Configures lint rules applied to all modules in the workspace. Module-specific lint settings override these defaults entirely. If unspecified, the `STANDARD` rule category is used.", url: bufYAMLDocsURL + "#lint"},
	"breaking": {summary: "Configures breaking change detection rules applied to all modules in the workspace. Module-specific breaking settings override these defaults entirely. If unspecified, the `FILE` rule category is used.", url: bufYAMLDocsURL + "#breaking"},
	"plugins":  {summary: "Specifies custom Buf plugins that provide additional lint or breaking change rules. Each entry references a plugin binary on `$PATH`, a local path, or a remote BSR plugin.", url: bufYAMLDocsURL + "#plugins"},
	"policies": {summary: "Lists policies that apply shared lint and breaking change rule sets to the workspace. Policies can be local files or remote BSR policies.", url: bufYAMLDocsURL + "#policies"},
}

// bufYAMLModuleDocs maps module-entry sub-keys to their documentation.
var bufYAMLModuleDocs = map[string]bufYAMLDoc{
	"path":     {summary: "Directory containing Protobuf files, relative to the workspace root. All `.proto` files in the directory and its subdirectories are included unless further restricted by `includes` or `excludes`.", url: bufYAMLDocsURL + "#path"},
	"name":     {summary: "A Buf Schema Registry path (e.g. `buf.build/acme/petapis`) that uniquely identifies this module. Setting a name associates the directory with a BSR repository for publishing commits and generated artifacts.", url: bufYAMLDocsURL + "#name"},
	"includes": {summary: "Subdirectories to include when discovering Protobuf files. When set, only files within the listed directories are processed. When omitted, all subdirectories are included.", url: bufYAMLDocsURL + "#includes"},
	"excludes": {summary: "Subdirectories to exclude from Protobuf file discovery. When used together with `includes`, each excluded directory must be within an included one.", url: bufYAMLDocsURL + "#excludes"},
}

// bufYAMLLintDocs maps lint sub-keys to their documentation.
var bufYAMLLintDocs = map[string]bufYAMLDoc{
	"use":                             {summary: "Lists lint rule categories and/or specific rule IDs to enable. Category names (e.g. `MINIMAL`, `BASIC`, `STANDARD`) select a predefined set of rules.", url: bufYAMLLintRulesURL},
	"except":                          {summary: "Removes specific rules or categories from the active lint rule set. Rules listed here are excluded even if they are part of a category in `use`. Prefer `ignore_only` to suppress rules for specific files.", url: bufYAMLDocsURL + "#lint"},
	"ignore":                          {summary: "Files and directories excluded from all lint rules. Paths are relative to `buf.yaml`. All files within a listed directory are also excluded.", url: bufYAMLDocsURL + "#lint"},
	"ignore_only":                     {summary: "Excludes specific files or directories from particular lint rules or categories. Maps each rule ID or category name to a list of file/directory paths (relative to `buf.yaml`) where that rule is suppressed.", url: bufYAMLDocsURL + "#lint"},
	"disallow_comment_ignores":        {summary: "When `true`, disables `// buf:lint:ignore RULE_ID` comment directives in `.proto` files. Defaults to `false`, which permits per-location rule suppression using comments.", url: bufYAMLDocsURL + "#lint"},
	"enum_zero_value_suffix":          {summary: "Sets the required suffix for zero-value enum entries, enforced by the `ENUM_ZERO_VALUE_SUFFIX` rule. Defaults to `_UNSPECIFIED`. For example, setting this to `_NONE` allows `FOO_NONE = 0`.", url: bufYAMLDocsURL + "#lint"},
	"rpc_allow_same_request_response": {summary: "When `true`, permits using the same message type for both the request and response of an RPC. Defaults to `false`. Buf discourages this pattern because it prevents independent evolution of request and response types.", url: bufYAMLDocsURL + "#lint"},
	"rpc_allow_google_protobuf_empty_requests":  {summary: "When `true`, allows RPC methods to use `google.protobuf.Empty` as the request type. Defaults to `false`. Prefer a dedicated request message to allow adding fields without breaking changes.", url: bufYAMLDocsURL + "#lint"},
	"rpc_allow_google_protobuf_empty_responses": {summary: "When `true`, allows RPC methods to use `google.protobuf.Empty` as the response type. Defaults to `false`. Prefer a dedicated response message to allow adding fields without breaking changes.", url: bufYAMLDocsURL + "#lint"},
	"service_suffix":  {summary: "Sets the required suffix for service names, enforced by the `SERVICE_SUFFIX` rule. Defaults to `Service`. For example, setting this to `API` allows service names like `FooAPI`.", url: bufYAMLDocsURL + "#lint"},
	"disable_builtin": {summary: "When `true`, disables all built-in lint rules. Use this when relying entirely on custom plugin-provided rules. Defaults to `false`.", url: bufYAMLDocsURL + "#lint"},
}

// bufYAMLBreakingDocs maps breaking sub-keys to their documentation.
var bufYAMLBreakingDocs = map[string]bufYAMLDoc{
	"use":                      {summary: "Lists breaking change rule categories and/or specific rule IDs to enable. Category names (`FILE`, `PACKAGE`, `WIRE_JSON`, `WIRE`) select a predefined set of rules.", url: bufYAMLBreakingRulesURL},
	"except":                   {summary: "Removes specific rules or categories from the active breaking change rule set. Using `except` is generally discouraged.", url: bufYAMLDocsURL + "#breaking"},
	"ignore":                   {summary: "Files and directories excluded from all breaking change rules. Paths are relative to `buf.yaml`. Useful for alpha or unstable packages that are expected to change.", url: bufYAMLDocsURL + "#breaking"},
	"ignore_only":              {summary: "Excludes specific files or directories from particular breaking change rules or categories. Maps each rule ID or category name to a list of file/directory paths (relative to `buf.yaml`) where that rule is suppressed.", url: bufYAMLDocsURL + "#breaking"},
	"ignore_unstable_packages": {summary: "When `true`, ignores packages matching unstable version patterns such as `v1alpha1`, `v1beta1`, or `v1test`. Defaults to `false`.", url: bufYAMLDocsURL + "#breaking"},
	"disable_builtin":          {summary: "When `true`, disables all built-in breaking change rules. Use this when relying entirely on custom plugin-provided rules. Defaults to `false`.", url: bufYAMLDocsURL + "#breaking"},
}

// bufYAMLLintRuleDocs maps lint rule IDs and category names to their documentation.
var bufYAMLLintRuleDocs = map[string]bufYAMLDoc{
	// Categories
	"MINIMAL":   {summary: "Fundamental rules for correct Protobuf file structure and package organization. Covers directory/package matching, package declarations, and import cycle detection.", url: bufYAMLLintRulesURL},
	"BASIC":     {summary: "All `MINIMAL` rules plus widely accepted Protobuf style standards: naming conventions (PascalCase, snake_case, UPPER_SNAKE_CASE) and import discipline.", url: bufYAMLLintRulesURL},
	"STANDARD":  {summary: "The default lint rule set. All `BASIC` rules plus versioned package suffixes, unique request/response messages, [protovalidate](https://protovalidate.com) constraint validation, and RPC naming conventions.", url: bufYAMLLintRulesURL},
	"COMMENTS":  {summary: "Optional category requiring non-empty leading comments on all schema elements: enums, enum values, fields, messages, oneofs, RPC methods, and services.", url: bufYAMLLintRulesURL},
	"UNARY_RPC": {summary: "Optional category prohibiting streaming RPCs. Disallows both client streaming (`RPC_NO_CLIENT_STREAMING`) and server streaming (`RPC_NO_SERVER_STREAMING`).", url: bufYAMLLintRulesURL},
	// MINIMAL rules
	"DIRECTORY_SAME_PACKAGE":  {summary: "All `.proto` files in a directory must have the same package declaration.", url: bufYAMLLintRulesURL},
	"PACKAGE_DEFINED":         {summary: "All `.proto` files must have a package declaration.", url: bufYAMLLintRulesURL},
	"PACKAGE_DIRECTORY_MATCH": {summary: "The package name must match the directory structure relative to the module root.", url: bufYAMLLintRulesURL},
	"PACKAGE_NO_IMPORT_CYCLE": {summary: "Detects import cycles at the package level.", url: bufYAMLLintRulesURL},
	"PACKAGE_SAME_DIRECTORY":  {summary: "All `.proto` files with the same package must be in the same directory.", url: bufYAMLLintRulesURL},
	// BASIC rules (additional)
	"ENUM_FIRST_VALUE_ZERO":            {summary: "The first enum value must have a numeric value of zero.", url: bufYAMLLintRulesURL},
	"ENUM_NO_ALLOW_ALIAS":              {summary: "Enums must not use the `allow_alias = true` option.", url: bufYAMLLintRulesURL},
	"ENUM_PASCAL_CASE":                 {summary: "Enum names must use PascalCase (e.g. `MyEnum`, not `my_enum`).", url: bufYAMLLintRulesURL},
	"ENUM_VALUE_UPPER_SNAKE_CASE":      {summary: "Enum value names must use UPPER_SNAKE_CASE (e.g. `MY_VALUE`, not `myValue`).", url: bufYAMLLintRulesURL},
	"FIELD_LOWER_SNAKE_CASE":           {summary: "Field names must use lower_snake_case (e.g. `my_field`, not `myField`).", url: bufYAMLLintRulesURL},
	"IMPORT_NO_PUBLIC":                 {summary: "Public imports (`import public`) are not allowed.", url: bufYAMLLintRulesURL},
	"IMPORT_NO_WEAK":                   {summary: "Weak imports (`import weak`) are not allowed.", url: bufYAMLLintRulesURL},
	"IMPORT_USED":                      {summary: "All imported files must be used within the `.proto` file.", url: bufYAMLLintRulesURL},
	"MESSAGE_PASCAL_CASE":              {summary: "Message names must use PascalCase (e.g. `MyMessage`, not `my_message`).", url: bufYAMLLintRulesURL},
	"ONEOF_LOWER_SNAKE_CASE":           {summary: "Oneof names must use lower_snake_case.", url: bufYAMLLintRulesURL},
	"PACKAGE_LOWER_SNAKE_CASE":         {summary: "Package names must use lower_snake_case.", url: bufYAMLLintRulesURL},
	"PACKAGE_SAME_CSHARP_NAMESPACE":    {summary: "All `.proto` files in the same package must declare the same `csharp_namespace` option.", url: bufYAMLLintRulesURL},
	"PACKAGE_SAME_GO_PACKAGE":          {summary: "All `.proto` files in the same package must declare the same `go_package` option.", url: bufYAMLLintRulesURL},
	"PACKAGE_SAME_JAVA_MULTIPLE_FILES": {summary: "All `.proto` files in the same package must declare the same `java_multiple_files` option.", url: bufYAMLLintRulesURL},
	"PACKAGE_SAME_JAVA_PACKAGE":        {summary: "All `.proto` files in the same package must declare the same `java_package` option.", url: bufYAMLLintRulesURL},
	"PACKAGE_SAME_PHP_NAMESPACE":       {summary: "All `.proto` files in the same package must declare the same `php_namespace` option.", url: bufYAMLLintRulesURL},
	"PACKAGE_SAME_RUBY_PACKAGE":        {summary: "All `.proto` files in the same package must declare the same `ruby_package` option.", url: bufYAMLLintRulesURL},
	"PACKAGE_SAME_SWIFT_PREFIX":        {summary: "All `.proto` files in the same package must declare the same `swift_prefix` option.", url: bufYAMLLintRulesURL},
	"RPC_PASCAL_CASE":                  {summary: "RPC method names must use PascalCase.", url: bufYAMLLintRulesURL},
	"SERVICE_PASCAL_CASE":              {summary: "Service names must use PascalCase.", url: bufYAMLLintRulesURL},
	"SYNTAX_SPECIFIED":                 {summary: "All `.proto` files must declare a syntax (e.g. `syntax = \"proto3\";`).", url: bufYAMLLintRulesURL},
	// STANDARD rules (additional)
	"ENUM_VALUE_PREFIX":           {summary: "Enum value names must be prefixed with the enum name in UPPER_SNAKE_CASE (e.g. `MY_ENUM_VALUE` for enum `MyEnum`).", url: bufYAMLLintRulesURL},
	"ENUM_ZERO_VALUE_SUFFIX":      {summary: "The zero-value enum entry must end with the configured suffix (default: `_UNSPECIFIED`). Configurable via `enum_zero_value_suffix`.", url: bufYAMLLintRulesURL},
	"FILE_LOWER_SNAKE_CASE":       {summary: "Proto filenames must use lower_snake_case (e.g. `my_service.proto`, not `MyService.proto`).", url: bufYAMLLintRulesURL},
	"PACKAGE_VERSION_SUFFIX":      {summary: "The last component of every package name must be a version (e.g. `v1`, `v2`, `v1alpha1`, `v1beta1`).", url: bufYAMLLintRulesURL},
	"PROTOVALIDATE":               {summary: "Validates that all [protovalidate](https://protovalidate.com) constraint annotations are syntactically correct and semantically compatible with their field types.", url: bufYAMLLintRulesURL},
	"RPC_REQUEST_STANDARD_NAME":   {summary: "RPC request messages must be named `MethodNameRequest` or `ServiceNameMethodNameRequest`.", url: bufYAMLLintRulesURL},
	"RPC_RESPONSE_STANDARD_NAME":  {summary: "RPC response messages must be named `MethodNameResponse` or `ServiceNameMethodNameResponse`.", url: bufYAMLLintRulesURL},
	"RPC_REQUEST_RESPONSE_UNIQUE": {summary: "Each RPC method must use a unique request message and a unique response message not shared with any other RPC.", url: bufYAMLLintRulesURL},
	"SERVICE_SUFFIX":              {summary: "Service names must end with the configured suffix (default: `Service`). Configurable via `service_suffix`.", url: bufYAMLLintRulesURL},
	// COMMENTS rules
	"COMMENT_ENUM":       {summary: "Enum types must have a non-empty leading comment.", url: bufYAMLLintRulesURL},
	"COMMENT_ENUM_VALUE": {summary: "Enum values must have a non-empty leading comment.", url: bufYAMLLintRulesURL},
	"COMMENT_FIELD":      {summary: "Fields must have a non-empty leading comment.", url: bufYAMLLintRulesURL},
	"COMMENT_MESSAGE":    {summary: "Messages must have a non-empty leading comment.", url: bufYAMLLintRulesURL},
	"COMMENT_ONEOF":      {summary: "Oneof groups must have a non-empty leading comment.", url: bufYAMLLintRulesURL},
	"COMMENT_RPC":        {summary: "RPC methods must have a non-empty leading comment.", url: bufYAMLLintRulesURL},
	"COMMENT_SERVICE":    {summary: "Services must have a non-empty leading comment.", url: bufYAMLLintRulesURL},
	// UNARY_RPC rules
	"RPC_NO_CLIENT_STREAMING": {summary: "RPC methods must not use client streaming.", url: bufYAMLLintRulesURL},
	"RPC_NO_SERVER_STREAMING": {summary: "RPC methods must not use server streaming.", url: bufYAMLLintRulesURL},
	// Uncategorized rules
	"FIELD_NOT_REQUIRED":                {summary: "Fields must not use the `required` label. Available in v2 configuration only.", url: bufYAMLLintRulesURL},
	"STABLE_PACKAGE_NO_IMPORT_UNSTABLE": {summary: "Stable versioned packages (e.g. `v1`) must not import unstable packages (e.g. `v1alpha1`, `v1beta1`).", url: bufYAMLLintRulesURL},
}

// bufYAMLBreakingRuleDocs maps breaking change rule IDs and category names to their documentation.
var bufYAMLBreakingRuleDocs = map[string]bufYAMLDoc{
	// Categories
	"FILE":      {summary: "Detects changes that would break generated code on a per-file basis. This is the default breaking change category.", url: bufYAMLBreakingRulesURL},
	"PACKAGE":   {summary: "Detects changes that would break generated code at the package level, allowing moves between files within the same package.", url: bufYAMLBreakingRulesURL},
	"WIRE_JSON": {summary: "Detects changes that would break binary wire encoding or JSON encoding/decoding, ignoring generated code compatibility.", url: bufYAMLBreakingRulesURL},
	"WIRE":      {summary: "Detects only changes that would break binary wire encoding. The most permissive breaking change category.", url: bufYAMLBreakingRulesURL},
	// Enum rules
	"ENUM_NO_DELETE":                              {summary: "Checks that no enum type is deleted.", url: bufYAMLBreakingRulesURL},
	"ENUM_SAME_JSON_FORMAT":                       {summary: "Checks that the enum's JSON format support does not change.", url: bufYAMLBreakingRulesURL},
	"ENUM_SAME_TYPE":                              {summary: "Checks that the enum's open/closed status does not change.", url: bufYAMLBreakingRulesURL},
	"ENUM_VALUE_NO_DELETE":                        {summary: "Checks that no enum value is deleted.", url: bufYAMLBreakingRulesURL},
	"ENUM_VALUE_NO_DELETE_UNLESS_NAME_RESERVED":   {summary: "Checks that deleted enum values have their name reserved before deletion.", url: bufYAMLBreakingRulesURL},
	"ENUM_VALUE_NO_DELETE_UNLESS_NUMBER_RESERVED": {summary: "Checks that deleted enum values have their number reserved before deletion.", url: bufYAMLBreakingRulesURL},
	"ENUM_VALUE_SAME_NAME":                        {summary: "Checks that enum value names do not change for a given numeric value.", url: bufYAMLBreakingRulesURL},
	// Extension rules
	"EXTENSION_MESSAGE_NO_DELETE": {summary: "Checks that no extension range is deleted from a message.", url: bufYAMLBreakingRulesURL},
	"EXTENSION_NO_DELETE":         {summary: "Checks that no extension is deleted. Available in v2 configuration only.", url: bufYAMLBreakingRulesURL},
	"PACKAGE_EXTENSION_NO_DELETE": {summary: "Checks that no extension is deleted at the package level. Available in v2 configuration only.", url: bufYAMLBreakingRulesURL},
	// Field rules
	"FIELD_NO_DELETE":                        {summary: "Checks that no message field is deleted.", url: bufYAMLBreakingRulesURL},
	"FIELD_NO_DELETE_UNLESS_NAME_RESERVED":   {summary: "Checks that deleted fields have their name reserved before deletion.", url: bufYAMLBreakingRulesURL},
	"FIELD_NO_DELETE_UNLESS_NUMBER_RESERVED": {summary: "Checks that deleted fields have their number reserved before deletion.", url: bufYAMLBreakingRulesURL},
	"FIELD_SAME_CARDINALITY":                 {summary: "Checks that field cardinality (optional/repeated) does not change.", url: bufYAMLBreakingRulesURL},
	"FIELD_SAME_CPP_STRING_TYPE":             {summary: "Checks that the C++ string type for string/bytes fields does not change.", url: bufYAMLBreakingRulesURL},
	"FIELD_SAME_DEFAULT":                     {summary: "Checks that field default values do not change.", url: bufYAMLBreakingRulesURL},
	"FIELD_SAME_JAVA_UTF8_VALIDATION":        {summary: "Checks that the Java UTF-8 validation mode for string fields does not change.", url: bufYAMLBreakingRulesURL},
	"FIELD_SAME_JSON_NAME":                   {summary: "Checks that the JSON name of a field does not change.", url: bufYAMLBreakingRulesURL},
	"FIELD_SAME_JSTYPE":                      {summary: "Checks that the `jstype` option for a field does not change.", url: bufYAMLBreakingRulesURL},
	"FIELD_SAME_NAME":                        {summary: "Checks that field names do not change for a given field number.", url: bufYAMLBreakingRulesURL},
	"FIELD_SAME_ONEOF":                       {summary: "Checks that fields are not moved into or out of a oneof.", url: bufYAMLBreakingRulesURL},
	"FIELD_SAME_TYPE":                        {summary: "Checks that field types do not change.", url: bufYAMLBreakingRulesURL},
	"FIELD_SAME_UTF8_VALIDATION":             {summary: "Checks that the runtime UTF-8 validation mode for string fields does not change.", url: bufYAMLBreakingRulesURL},
	"FIELD_WIRE_COMPATIBLE_CARDINALITY":      {summary: "Checks that cardinality changes remain wire-compatible.", url: bufYAMLBreakingRulesURL},
	"FIELD_WIRE_COMPATIBLE_TYPE":             {summary: "Checks that scalar type changes remain wire-compatible.", url: bufYAMLBreakingRulesURL},
	"FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY": {summary: "Checks that cardinality changes remain wire and JSON compatible.", url: bufYAMLBreakingRulesURL},
	"FIELD_WIRE_JSON_COMPATIBLE_TYPE":        {summary: "Checks that type changes remain wire and JSON compatible.", url: bufYAMLBreakingRulesURL},
	// File rules
	"FILE_NO_DELETE":                   {summary: "Checks that no `.proto` file is deleted.", url: bufYAMLBreakingRulesURL},
	"FILE_SAME_CC_ENABLE_ARENAS":       {summary: "Checks that the `cc_enable_arenas` file option does not change.", url: bufYAMLBreakingRulesURL},
	"FILE_SAME_CC_GENERIC_SERVICES":    {summary: "Checks that the `cc_generic_services` file option does not change.", url: bufYAMLBreakingRulesURL},
	"FILE_SAME_CSHARP_NAMESPACE":       {summary: "Checks that the `csharp_namespace` file option does not change.", url: bufYAMLBreakingRulesURL},
	"FILE_SAME_GO_PACKAGE":             {summary: "Checks that the `go_package` file option does not change.", url: bufYAMLBreakingRulesURL},
	"FILE_SAME_JAVA_GENERIC_SERVICES":  {summary: "Checks that the `java_generic_services` file option does not change.", url: bufYAMLBreakingRulesURL},
	"FILE_SAME_JAVA_MULTIPLE_FILES":    {summary: "Checks that the `java_multiple_files` file option does not change.", url: bufYAMLBreakingRulesURL},
	"FILE_SAME_JAVA_OUTER_CLASSNAME":   {summary: "Checks that the `java_outer_classname` file option does not change.", url: bufYAMLBreakingRulesURL},
	"FILE_SAME_JAVA_PACKAGE":           {summary: "Checks that the `java_package` file option does not change.", url: bufYAMLBreakingRulesURL},
	"FILE_SAME_OBJC_CLASS_PREFIX":      {summary: "Checks that the `objc_class_prefix` file option does not change.", url: bufYAMLBreakingRulesURL},
	"FILE_SAME_OPTIMIZE_FOR":           {summary: "Checks that the `optimize_for` file option does not change.", url: bufYAMLBreakingRulesURL},
	"FILE_SAME_PACKAGE":                {summary: "Checks that the file's package declaration does not change.", url: bufYAMLBreakingRulesURL},
	"FILE_SAME_PHP_CLASS_PREFIX":       {summary: "Checks that the `php_class_prefix` file option does not change.", url: bufYAMLBreakingRulesURL},
	"FILE_SAME_PHP_METADATA_NAMESPACE": {summary: "Checks that the `php_metadata_namespace` file option does not change.", url: bufYAMLBreakingRulesURL},
	"FILE_SAME_PHP_NAMESPACE":          {summary: "Checks that the `php_namespace` file option does not change.", url: bufYAMLBreakingRulesURL},
	"FILE_SAME_PY_GENERIC_SERVICES":    {summary: "Checks that the `py_generic_services` file option does not change.", url: bufYAMLBreakingRulesURL},
	"FILE_SAME_RUBY_PACKAGE":           {summary: "Checks that the `ruby_package` file option does not change.", url: bufYAMLBreakingRulesURL},
	"FILE_SAME_SWIFT_PREFIX":           {summary: "Checks that the `swift_prefix` file option does not change.", url: bufYAMLBreakingRulesURL},
	"FILE_SAME_SYNTAX":                 {summary: "Checks that the file's syntax version (`proto2`/`proto3`) does not change.", url: bufYAMLBreakingRulesURL},
	// Message rules
	"MESSAGE_NO_DELETE": {summary: "Checks that no message type is deleted.", url: bufYAMLBreakingRulesURL},
	"MESSAGE_NO_REMOVE_STANDARD_DESCRIPTOR_ACCESSOR": {summary: "Checks that the standard descriptor accessor is not removed from a message.", url: bufYAMLBreakingRulesURL},
	"MESSAGE_SAME_JSON_FORMAT":                       {summary: "Checks that the message's JSON format support does not change.", url: bufYAMLBreakingRulesURL},
	"MESSAGE_SAME_REQUIRED_FIELDS":                   {summary: "Checks that required fields are not added or deleted.", url: bufYAMLBreakingRulesURL},
	// Oneof rules
	"ONEOF_NO_DELETE": {summary: "Checks that no oneof is deleted from a message.", url: bufYAMLBreakingRulesURL},
	// Package rules
	"PACKAGE_ENUM_NO_DELETE":    {summary: "Checks that no enum type is deleted from a package.", url: bufYAMLBreakingRulesURL},
	"PACKAGE_MESSAGE_NO_DELETE": {summary: "Checks that no message type is deleted from a package.", url: bufYAMLBreakingRulesURL},
	"PACKAGE_NO_DELETE":         {summary: "Checks that no package is deleted.", url: bufYAMLBreakingRulesURL},
	"PACKAGE_SERVICE_NO_DELETE": {summary: "Checks that no service is deleted from a package.", url: bufYAMLBreakingRulesURL},
	// Reserved rules
	"RESERVED_ENUM_NO_DELETE":    {summary: "Checks that reserved enum field ranges and names are not deleted.", url: bufYAMLBreakingRulesURL},
	"RESERVED_MESSAGE_NO_DELETE": {summary: "Checks that reserved message field ranges and names are not deleted.", url: bufYAMLBreakingRulesURL},
	// RPC and service rules
	"RPC_NO_DELETE":              {summary: "Checks that no RPC method is deleted from a service.", url: bufYAMLBreakingRulesURL},
	"RPC_SAME_CLIENT_STREAMING":  {summary: "Checks that the client streaming mode of an RPC does not change.", url: bufYAMLBreakingRulesURL},
	"RPC_SAME_IDEMPOTENCY_LEVEL": {summary: "Checks that the idempotency level of an RPC does not change.", url: bufYAMLBreakingRulesURL},
	"RPC_SAME_REQUEST_TYPE":      {summary: "Checks that the request message type of an RPC does not change.", url: bufYAMLBreakingRulesURL},
	"RPC_SAME_RESPONSE_TYPE":     {summary: "Checks that the response message type of an RPC does not change.", url: bufYAMLBreakingRulesURL},
	"RPC_SAME_SERVER_STREAMING":  {summary: "Checks that the server streaming mode of an RPC does not change.", url: bufYAMLBreakingRulesURL},
	"SERVICE_NO_DELETE":          {summary: "Checks that no service is deleted.", url: bufYAMLBreakingRulesURL},
}

// bufYAMLHover searches the parsed buf.yaml document for hover information at
// the given position and returns a Hover response, or nil if the position does
// not correspond to a known field or rule.
func bufYAMLHover(docNode *yaml.Node, pos protocol.Position) *protocol.Hover {
	if docNode.Kind != yaml.DocumentNode || len(docNode.Content) == 0 {
		return nil
	}
	mapping := docNode.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return nil
	}
	return searchMappingForHover(mapping, pos, nil)
}

// searchMappingForHover recursively searches a YAML mapping node for the
// cursor position and returns hover info when a known key or value is found.
// parentPath is the sequence of key names leading to this mapping.
func searchMappingForHover(node *yaml.Node, pos protocol.Position, parentPath []string) *protocol.Hover {
	for i := 0; i+1 < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]

		// Build the path for this key, ensuring a new backing array each iteration
		// to avoid mutating parentPath across loop iterations.
		currentPath := append(parentPath[:len(parentPath):len(parentPath)], keyNode.Value)

		if yamlNodeContainsPosition(keyNode, pos) {
			return hoverForKeyPath(currentPath, yamlNodeRange(keyNode))
		}

		switch valNode.Kind {
		case yaml.MappingNode:
			if h := searchMappingForHover(valNode, pos, currentPath); h != nil {
				return h
			}
		case yaml.SequenceNode:
			if h := searchSequenceForHover(valNode, pos, currentPath); h != nil {
				return h
			}
		}
	}
	return nil
}

// searchSequenceForHover searches a YAML sequence node for the cursor position.
// parentPath is the path of keys leading to this sequence (e.g. ["lint", "use"]).
func searchSequenceForHover(node *yaml.Node, pos protocol.Position, parentPath []string) *protocol.Hover {
	for _, item := range node.Content {
		switch item.Kind {
		case yaml.ScalarNode:
			if yamlNodeContainsPosition(item, pos) {
				return hoverForSequenceValue(item.Value, parentPath, yamlNodeRange(item))
			}
		case yaml.MappingNode:
			// Sequence items that are mappings (e.g. module entries under "modules").
			if h := searchMappingForHover(item, pos, parentPath); h != nil {
				return h
			}
		}
	}
	return nil
}

// hoverForKeyPath returns hover documentation for a buf.yaml key identified by
// its dot-path (e.g. ["lint", "use"]).
func hoverForKeyPath(path []string, nodeRange protocol.Range) *protocol.Hover {
	if len(path) == 0 {
		return nil
	}

	// Top-level keys (version, modules, deps, lint, breaking, plugins, policies).
	if len(path) == 1 {
		if doc, ok := bufYAMLTopLevelDocs[path[0]]; ok {
			return makeBufYAMLHover(path[0], doc, nodeRange)
		}
		return nil
	}

	// For keys inside a module sequence entry, strip the leading "modules" prefix
	// so that module-level lint/breaking sub-fields resolve the same as top-level.
	effective := path
	if path[0] == "modules" {
		effective = path[1:]
		// Module-specific sub-fields (path, name, includes, excludes).
		if len(effective) == 1 {
			if doc, ok := bufYAMLModuleDocs[effective[0]]; ok {
				return makeBufYAMLHover(effective[0], doc, nodeRange)
			}
			// Module-level lint/breaking keys share docs with their top-level counterparts.
			if doc, ok := bufYAMLTopLevelDocs[effective[0]]; ok {
				return makeBufYAMLHover(effective[0], doc, nodeRange)
			}
			return nil
		}
	}

	switch len(effective) {
	case 2:
		switch effective[0] {
		case "lint":
			if doc, ok := bufYAMLLintDocs[effective[1]]; ok {
				return makeBufYAMLHover("lint."+effective[1], doc, nodeRange)
			}
		case "breaking":
			if doc, ok := bufYAMLBreakingDocs[effective[1]]; ok {
				return makeBufYAMLHover("breaking."+effective[1], doc, nodeRange)
			}
		}
	case 3:
		// lint.ignore_only.<RULE> and breaking.ignore_only.<RULE>: the rule name
		// is used as a mapping key; show docs for that rule.
		if effective[1] == "ignore_only" {
			ruleName := effective[2]
			switch effective[0] {
			case "lint":
				if doc, ok := bufYAMLLintRuleDocs[ruleName]; ok {
					return makeBufYAMLHover(ruleName, doc, nodeRange)
				}
			case "breaking":
				if doc, ok := bufYAMLBreakingRuleDocs[ruleName]; ok {
					return makeBufYAMLHover(ruleName, doc, nodeRange)
				}
			}
		}
	}
	return nil
}

// hoverForSequenceValue returns hover documentation for a scalar value inside a
// YAML sequence whose parentPath ends with "use" or "except" under a known
// lint or breaking section.
func hoverForSequenceValue(value string, parentPath []string, nodeRange protocol.Range) *protocol.Hover {
	// parentPath for lint.use items is ["lint", "use"] (or ["modules", "lint", "use"]).
	// The last element is "use" or "except"; the enclosing section is the one before it.
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

// makeBufYAMLHover formats a Hover response with a markdown heading, summary,
// and documentation link for the given buf.yaml field or rule.
func makeBufYAMLHover(displayName string, doc bufYAMLDoc, nodeRange protocol.Range) *protocol.Hover {
	body := fmt.Sprintf("**`%s`**\n\n%s", displayName, doc.summary)
	if doc.url != "" {
		body += fmt.Sprintf("\n\n[Documentation](%s)", doc.url)
	}
	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: body,
		},
		Range: &nodeRange,
	}
}

// yamlNodeContainsPosition reports whether the given scalar YAML node's text
// span contains the LSP cursor position. yaml.v3 uses 1-indexed line/column;
// LSP uses 0-indexed.
func yamlNodeContainsPosition(node *yaml.Node, pos protocol.Position) bool {
	nodeLine := uint32(node.Line - 1)
	if pos.Line != nodeLine {
		return false
	}
	nodeCol := uint32(node.Column - 1)
	return pos.Character >= nodeCol && pos.Character < nodeCol+uint32(len(node.Value))
}

// bsrRef holds a BSR reference string and its source position in a YAML file.
// It is used by the buf.yaml, buf.gen.yaml, buf.policy.yaml, and buf.lock
// document-link implementations to track which YAML scalar values map to BSR
// pages.
type bsrRef struct {
	// ref is the reference string, e.g. "buf.build/bufbuild/es:v2.2.2".
	ref string
	// refRange is the range spanning this value in the file.
	refRange protocol.Range
}

// yamlNodeRange returns the LSP protocol Range for a scalar YAML node.
func yamlNodeRange(node *yaml.Node) protocol.Range {
	line := uint32(node.Line - 1)
	col := uint32(node.Column - 1)
	return protocol.Range{
		Start: protocol.Position{Line: line, Character: col},
		End:   protocol.Position{Line: line, Character: col + uint32(len(node.Value))},
	}
}

// parseYAMLDoc decodes text as a YAML document and returns the document node,
// or nil if parsing fails.
func parseYAMLDoc(text string) *yaml.Node {
	var doc yaml.Node
	if err := yaml.NewDecoder(strings.NewReader(text)).Decode(&doc); err != nil {
		return nil
	}
	return &doc
}

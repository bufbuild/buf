// Copyright 2020-2023 Buf Technologies, Inc.
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

package bufconfig

// FileOption is a file option.
type FileOption int

const (
	// FileOptionUnspecified is an unspecified file option.
	FileOptionUnspecified FileOption = iota
	// FileOptionJavaPackage is the file option java_package.
	FileOptionJavaPackage
	// FileOptionJavaPackagePrefix is the file option java_package_prefix.
	FileOptionJavaPackagePrefix
	// FileOptionJavaPackageSuffix is the file option java_package_suffix.
	FileOptionJavaPackageSuffix
	// FileOptionJavaOuterClassname is the file option java_outer_classname.
	FileOptionJavaOuterClassname
	// FileOptionJavaMultipleFiles is the file option java_multiple_files.
	FileOptionJavaMultipleFiles
	// FileOptionJavaStringCheckUtf8 is the file option java_string_check_utf8.
	FileOptionJavaStringCheckUtf8
	// FileOptionOptimizeFor is the file option optimize_for.
	FileOptionOptimizeFor
	// FileOptionGoPackage is the file option go_package.
	FileOptionGoPackage
	// FileOptionGoPackagePrefix is the file option go_package_prefix.
	FileOptionGoPackagePrefix
	// FileOptionCcEnableArenas is the file option cc_enable_arenas.
	FileOptionCcEnableArenas
	// FileOptionObjcClassPrefix is the file option objc_class_prefix.
	FileOptionObjcClassPrefix
	// FileOptionCsharpNamespace is the file option csharp_namespace.
	FileOptionCsharpNamespace
	// FileOptionCsharpNamespacePrefix is the file option csharp_namespace_prefix.
	FileOptionCsharpNamespacePrefix
	// FileOptionPhpNamespace is the file option php_namespace.
	FileOptionPhpNamespace
	// FileOptionPhpMetadataNamespace is the file option php_metadata_namespace.
	FileOptionPhpMetadataNamespace
	// FileOptionPhpMetadataNamespaceSuffix is the file option php_metadata_namespace_suffix.
	FileOptionPhpMetadataNamespaceSuffix
	// FileOptionRubyPackage is the file option ruby_package.
	FileOptionRubyPackage
	// FileOptionRubyPackageSuffix is the file option ruby_package_suffix.
	FileOptionRubyPackageSuffix
)

// FieldOption is a field option.
type FieldOption int

const (
	// FieldOptionUnspecified is an unspecified field option.
	FieldOptionUnspecified FieldOption = iota
	// FieldOptionJSType is the field option js_type.
	FieldOptionJSType
)

// GenerateManagedConfig is a managed mode configuration.
type GenerateManagedConfig interface {
	// Disables returns the disable rules in the configuration.
	Disables() []ManagedDisableRule
	// Overrides returns the override rules in the configuration.
	Overrides() []ManagedOverrideRule

	isGenerateManagedConfig()
}

// ManagedDisableRule is a disable rule. A disable rule describes:
//
//   - The options to not modify. If not specified, it means all options (both
//     file options and field options) are not modified.
//   - The files/fields for which these options are not modified. If not specified,
//     it means for all files/fields the specified options are not modified.
//
// A ManagedDisableRule is guaranteed to specify at least one of the two aspects.
// i.e. At least one of Path, ModuleFullName, FieldName, FileOption and
// FieldOption is not empty. A rule can disable all options for certain files/fields,
// disable certains options for all files/fields, or disable certain options for
// certain files/fields. To disable all options for all files/fields, turn off managed mode.
type ManagedDisableRule interface {
	// Path returns the file path, relative to its module, to disable managed mode for.
	Path() string
	// ModuleFullName returns the full name string of the module to disable
	// managed mode for.
	ModuleFullName() string
	// FieldName returns the fully qualified name for the field to disable managed
	// mode for. This is guaranteed to be empty if FileOption is not empty.
	FieldName() string
	// FileOption returns the file option to disable managed mode for. This is
	// guaranteed to be empty if FieldName is not empty.
	FileOption() FileOption
	// FieldOption returns the field option to disalbe managed mode for.
	FieldOption() FieldOption

	isManagedDisableRule()
}

// ManagedOverrideRule is an override rule. An override describes:
//
//   - The options to modify. Exactly one of FileOption and FieldOption is not empty.
//   - The value to modify these options with.
//   - The files/fields for which the options are modified. If all of Path, ModuleFullName
//   - or FieldName are empty, all files/fields are modified. Otherwise, only
//     file/fields that match the specified Path, ModuleFullName and FieldName
//     is modified.
type ManagedOverrideRule interface {
	// Path is the file path, relative to its module, to disable managed mode for.
	Path() string
	// ModuleFullName is the full name string of the module to disable
	// managed mode for.
	ModuleFullName() string
	// FieldName is the fully qualified name for the field to disable managed
	// mode for. This is guranteed to be empty is FileOption is not empty.
	FieldName() string
	// FileOption returns the file option to disable managed mode for. This is
	// guaranteed to be empty (FileOptionUnspecified) if FieldName is empty.
	FileOption() FileOption
	// FieldOption returns the field option to disable managed mode for.
	FieldOption() FieldOption
	// Value returns the override value.
	Value() interface{}

	isManagedOverrideRule()
}

type generateManagedConfig struct {
	disables  []ManagedDisableRule
	overrides []ManagedOverrideRule
}

func (g *generateManagedConfig) Disables() []ManagedDisableRule {
	return g.disables
}

func (g *generateManagedConfig) Overrides() []ManagedOverrideRule {
	return g.overrides
}

type managedDisableRule struct {
	path           string
	moduleFullName string
	fieldName      string
	fileOption     FileOption
	fieldOption    FieldOption
}

func (m *managedDisableRule) Path() string {
	return m.path
}

func (m *managedDisableRule) ModuleFullName() string {
	return m.moduleFullName
}

func (m *managedDisableRule) FieldName() string {
	return m.fieldName
}

func (m *managedDisableRule) FileOption() FileOption {
	return m.fileOption
}

func (m *managedDisableRule) FieldOption() FieldOption {
	return m.fieldOption
}

func (m *managedDisableRule) isManagedDisableRule() {}

type managedOverrideRule struct {
	path           string
	moduleFullName string
	fieldName      string
	fileOption     FileOption
	fieldOption    FieldOption
	value          interface{}
	prefix         string
	suffix         string
}

// TODO: decide where to validate path and module full name
func newManagedOverrideRule(
	path string,
	moduleFullName string,
	fileOption FileOption,
	value interface{},
) (*managedOverrideRule, error) {
	return &managedOverrideRule{
		path:           path,
		moduleFullName: moduleFullName,
		fileOption:     fileOption,
		value:          value,
	}, nil
}

func (m *managedOverrideRule) Path() string {
	return m.path
}

func (m *managedOverrideRule) ModuleFullName() string {
	return m.moduleFullName
}

func (m *managedOverrideRule) FieldName() string {
	return m.fieldName
}

func (m *managedOverrideRule) FileOption() FileOption {
	return m.fileOption
}

func (m *managedOverrideRule) FieldOption() FieldOption {
	return m.fieldOption
}

func (m *managedOverrideRule) Value() interface{} {
	return m.value
}

func (m *managedOverrideRule) Prefix() string {
	return m.prefix
}

func (m *managedOverrideRule) Suffix() string {
	return m.suffix
}

func (m *managedOverrideRule) isManagedOverrideRule() {}

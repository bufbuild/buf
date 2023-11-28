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

import "encoding/json"

// TODO: this is a temporary file to avoid crowing other files. We can choose to move stuff from this file over.
// TODO: this is also completely copied over from bufgen.go, the only change made to it so far is unexporting the type.
// TODO: update struct type names to externalXYZFileV1/2/1Beta1
// TODO: update GODOCs to the style of '// externalBufLockFileV2 represents the v2 buf.lock file.'

// externalBufGenYAMLFileV1 is a v1 external generate configuration.
type externalBufGenYAMLFileV1 struct {
	Version string                           `json:"version,omitempty" yaml:"version,omitempty"`
	Plugins []externalGeneratePluginConfigV1 `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	Managed externalGenerateManagedConfigV1  `json:"managed,omitempty" yaml:"managed,omitempty"`
	Types   externalTypesConfigV1            `json:"types,omitempty" yaml:"types,omitempty"`
}

// externalGeneratePluginConfigV1 is an external plugin configuration.
type externalGeneratePluginConfigV1 struct {
	Plugin     string      `json:"plugin,omitempty" yaml:"plugin,omitempty"`
	Revision   int         `json:"revision,omitempty" yaml:"revision,omitempty"`
	Name       string      `json:"name,omitempty" yaml:"name,omitempty"`
	Remote     string      `json:"remote,omitempty" yaml:"remote,omitempty"`
	Out        string      `json:"out,omitempty" yaml:"out,omitempty"`
	Opt        interface{} `json:"opt,omitempty" yaml:"opt,omitempty"`
	Path       interface{} `json:"path,omitempty" yaml:"path,omitempty"`
	ProtocPath string      `json:"protoc_path,omitempty" yaml:"protoc_path,omitempty"`
	Strategy   string      `json:"strategy,omitempty" yaml:"strategy,omitempty"`
}

// externalGenerateManagedConfigV1 is an external managed mode configuration.
//
// Only use outside of this package for testing.
type externalGenerateManagedConfigV1 struct {
	Enabled             bool                              `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	CcEnableArenas      *bool                             `json:"cc_enable_arenas,omitempty" yaml:"cc_enable_arenas,omitempty"`
	JavaMultipleFiles   *bool                             `json:"java_multiple_files,omitempty" yaml:"java_multiple_files,omitempty"`
	JavaStringCheckUtf8 *bool                             `json:"java_string_check_utf8,omitempty" yaml:"java_string_check_utf8,omitempty"`
	JavaPackagePrefix   externalJavaPackagePrefixConfigV1 `json:"java_package_prefix,omitempty" yaml:"java_package_prefix,omitempty"`
	CsharpNamespace     externalCsharpNamespaceConfigV1   `json:"csharp_namespace,omitempty" yaml:"csharp_namespace,omitempty"`
	OptimizeFor         externalOptimizeForConfigV1       `json:"optimize_for,omitempty" yaml:"optimize_for,omitempty"`
	GoPackagePrefix     externalGoPackagePrefixConfigV1   `json:"go_package_prefix,omitempty" yaml:"go_package_prefix,omitempty"`
	ObjcClassPrefix     externalObjcClassPrefixConfigV1   `json:"objc_class_prefix,omitempty" yaml:"objc_class_prefix,omitempty"`
	RubyPackage         externalRubyPackageConfigV1       `json:"ruby_package,omitempty" yaml:"ruby_package,omitempty"`
	Override            map[string]map[string]string      `json:"override,omitempty" yaml:"override,omitempty"`
}

// isEmpty returns true if the config is empty, excluding the 'Enabled' setting.
func (e externalGenerateManagedConfigV1) isEmpty() bool {
	return e.CcEnableArenas == nil &&
		e.JavaMultipleFiles == nil &&
		e.JavaStringCheckUtf8 == nil &&
		e.JavaPackagePrefix.isEmpty() &&
		e.CsharpNamespace.isEmpty() &&
		e.CsharpNamespace.isEmpty() &&
		e.OptimizeFor.isEmpty() &&
		e.GoPackagePrefix.isEmpty() &&
		e.ObjcClassPrefix.isEmpty() &&
		e.RubyPackage.isEmpty() &&
		len(e.Override) == 0
}

// externalJavaPackagePrefixConfigV1 is the external java_package prefix configuration.
type externalJavaPackagePrefixConfigV1 struct {
	Default  string            `json:"default,omitempty" yaml:"default,omitempty"`
	Except   []string          `json:"except,omitempty" yaml:"except,omitempty"`
	Override map[string]string `json:"override,omitempty" yaml:"override,omitempty"`
}

// isEmpty returns true if the config is empty.
func (e externalJavaPackagePrefixConfigV1) isEmpty() bool {
	return e.Default == "" &&
		len(e.Except) == 0 &&
		len(e.Override) == 0
}

// UnmarshalYAML satisfies the yaml.Unmarshaler interface. This is done to maintain backward compatibility
// of accepting a plain string value for java_package_prefix.
func (e *externalJavaPackagePrefixConfigV1) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return e.unmarshalWith(unmarshal)
}

// UnmarshalJSON satisfies the json.Unmarshaler interface. This is done to maintain backward compatibility
// of accepting a plain string value for java_package_prefix.
func (e *externalJavaPackagePrefixConfigV1) UnmarshalJSON(data []byte) error {
	unmarshal := func(v interface{}) error {
		return json.Unmarshal(data, v)
	}

	return e.unmarshalWith(unmarshal)
}

// unmarshalWith is used to unmarshal into json/yaml. See https://abhinavg.net/posts/flexible-yaml for details.
func (e *externalJavaPackagePrefixConfigV1) unmarshalWith(unmarshal func(interface{}) error) error {
	var prefix string
	if err := unmarshal(&prefix); err == nil {
		e.Default = prefix
		return nil
	}

	type rawExternalJavaPackagePrefixConfigV1 externalJavaPackagePrefixConfigV1
	if err := unmarshal((*rawExternalJavaPackagePrefixConfigV1)(e)); err != nil {
		return err
	}

	return nil
}

// externalOptimizeForConfigV1 is the external optimize_for configuration.
type externalOptimizeForConfigV1 struct {
	Default  string            `json:"default,omitempty" yaml:"default,omitempty"`
	Except   []string          `json:"except,omitempty" yaml:"except,omitempty"`
	Override map[string]string `json:"override,omitempty" yaml:"override,omitempty"`
}

// isEmpty returns true if the config is empty
func (e externalOptimizeForConfigV1) isEmpty() bool { // TODO: does it need to be public?
	return e.Default == "" &&
		len(e.Except) == 0 &&
		len(e.Override) == 0
}

// UnmarshalYAML satisfies the yaml.Unmarshaler interface. This is done to maintain backward compatibility
// of accepting a plain string value for optimize_for.
func (e *externalOptimizeForConfigV1) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return e.unmarshalWith(unmarshal)
}

// UnmarshalJSON satisfies the json.Unmarshaler interface. This is done to maintain backward compatibility
// of accepting a plain string value for optimize_for.
func (e *externalOptimizeForConfigV1) UnmarshalJSON(data []byte) error {
	unmarshal := func(v interface{}) error {
		return json.Unmarshal(data, v)
	}

	return e.unmarshalWith(unmarshal)
}

// unmarshalWith is used to unmarshal into json/yaml. See https://abhinavg.net/posts/flexible-yaml for details.
func (e *externalOptimizeForConfigV1) unmarshalWith(unmarshal func(interface{}) error) error {
	var optimizeFor string
	if err := unmarshal(&optimizeFor); err == nil {
		e.Default = optimizeFor
		return nil
	}

	type rawExternalOptimizeForConfigV1 externalOptimizeForConfigV1
	if err := unmarshal((*rawExternalOptimizeForConfigV1)(e)); err != nil {
		return err
	}

	return nil
}

// externalGoPackagePrefixConfigV1 is the external go_package prefix configuration.
type externalGoPackagePrefixConfigV1 struct {
	Default  string            `json:"default,omitempty" yaml:"default,omitempty"`
	Except   []string          `json:"except,omitempty" yaml:"except,omitempty"`
	Override map[string]string `json:"override,omitempty" yaml:"override,omitempty"`
}

// isEmpty returns true if the config is empty.
func (e externalGoPackagePrefixConfigV1) isEmpty() bool {
	return e.Default == "" &&
		len(e.Except) == 0 &&
		len(e.Override) == 0
}

// externalCsharpNamespaceConfigV1 is the external csharp_namespace configuration.
type externalCsharpNamespaceConfigV1 struct {
	Except   []string          `json:"except,omitempty" yaml:"except,omitempty"`
	Override map[string]string `json:"override,omitempty" yaml:"override,omitempty"`
}

// isEmpty returns true if the config is empty.
func (e externalCsharpNamespaceConfigV1) isEmpty() bool {
	return len(e.Except) == 0 &&
		len(e.Override) == 0
}

// externalRubyPackageConfigV1 is the external ruby_package configuration
type externalRubyPackageConfigV1 struct {
	Except   []string          `json:"except,omitempty" yaml:"except,omitempty"`
	Override map[string]string `json:"override,omitempty" yaml:"override,omitempty"`
}

// isEmpty returns true is the config is empty
func (e externalRubyPackageConfigV1) isEmpty() bool { // TODO: does this need to be public? same with other IsEmpty()
	return len(e.Except) == 0 && len(e.Override) == 0
}

// externalObjcClassPrefixConfigV1 is the external objc_class_prefix configuration.
type externalObjcClassPrefixConfigV1 struct {
	Default  string            `json:"default,omitempty" yaml:"default,omitempty"`
	Except   []string          `json:"except,omitempty" yaml:"except,omitempty"`
	Override map[string]string `json:"override,omitempty" yaml:"override,omitempty"`
}

func (e externalObjcClassPrefixConfigV1) isEmpty() bool {
	return e.Default == "" &&
		len(e.Except) == 0 &&
		len(e.Override) == 0
}

// externalBufGenYAMLV1Beta1 is a v1 external generate configuration.
type externalBufGenYAMLV1Beta1 struct {
	Version string                                `json:"version,omitempty" yaml:"version,omitempty"`
	Managed bool                                  `json:"managed,omitempty" yaml:"managed,omitempty"`
	Plugins []externalGeneratePluginConfigV1Beta1 `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	Options externalOptionsConfigV1Beta1          `json:"options,omitempty" yaml:"options,omitempty"`
}

// externalGeneratePluginConfigV1Beta1 is an external plugin configuration.
type externalGeneratePluginConfigV1Beta1 struct {
	Name     string      `json:"name,omitempty" yaml:"name,omitempty"`
	Out      string      `json:"out,omitempty" yaml:"out,omitempty"`
	Opt      interface{} `json:"opt,omitempty" yaml:"opt,omitempty"`
	Path     string      `json:"path,omitempty" yaml:"path,omitempty"`
	Strategy string      `json:"strategy,omitempty" yaml:"strategy,omitempty"`
}

// externalOptionsConfigV1Beta1 is an external options configuration.
type externalOptionsConfigV1Beta1 struct {
	CcEnableArenas    *bool  `json:"cc_enable_arenas,omitempty" yaml:"cc_enable_arenas,omitempty"`
	JavaMultipleFiles *bool  `json:"java_multiple_files,omitempty" yaml:"java_multiple_files,omitempty"`
	OptimizeFor       string `json:"optimize_for,omitempty" yaml:"optimize_for,omitempty"`
}

// externalTypesConfigV1 is an external types configuration.
type externalTypesConfigV1 struct {
	Include []string `json:"include,omitempty" yaml:"include"`
}

// isEmpty returns true if e is empty.
func (e externalTypesConfigV1) isEmpty() bool {
	return len(e.Include) == 0
}

// externalBufGenYAMLFileV2 is an external configuration.
type externalBufGenYAMLFileV2 struct {
	Version string                           `json:"version,omitempty" yaml:"version,omitempty"`
	Managed externalGenerateManagedConfigV2  `json:"managed,omitempty" yaml:"managed,omitempty"`
	Plugins []externalGeneratePluginConfigV2 `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	Inputs  []externalInputConfigV2          `json:"inputs,omitempty" yaml:"inputs,omitempty"`
}

// externalGenerateManagedConfigV2 is an external managed mode configuration.
type externalGenerateManagedConfigV2 struct {
	Enabled  bool                              `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Disable  []externalManagedDisableConfigV2  `json:"disable,omitempty" yaml:"disable,omitempty"`
	Override []externalManagedOverrideConfigV2 `json:"override,omitempty" yaml:"override,omitempty"`
}

// isEmpty returns true if the config is empty.
func (m externalGenerateManagedConfigV2) isEmpty() bool {
	return !m.Enabled && len(m.Disable) == 0 && len(m.Override) == 0
}

// externalManagedDisableConfigV2 is an external configuration that disables file options in
// managed mode.
type externalManagedDisableConfigV2 struct {
	// At most one of FileOption and FieldOption can be set
	// Must be validated to be a valid FileOption
	FileOption string `json:"file_option,omitempty" yaml:"file_option,omitempty"`
	// Must be validated to be a valid FieldOption
	FieldOption string `json:"field_option,omitempty" yaml:"field_option,omitempty"`
	// Must be validated to be a valid module path
	Module string `json:"module,omitempty" yaml:"module,omitempty"`
	// Must be normalized and validated
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
	// Must be validated to be a valid to be a valid field name.
	Field string `json:"field,omitempty" yaml:"field,omitempty"`
}

// externalManagedOverrideConfigV2 is an external configuration that overrides file options in
// managed mode.
type externalManagedOverrideConfigV2 struct {
	// Must set exactly one of FileOption and FieldOption
	// Must be validated to be a valid FileOption
	FileOption string `json:"file_option,omitempty" yaml:"file_option,omitempty"`
	// Must be validated to be a valid FieldOption
	FieldOption string `json:"field_option,omitempty" yaml:"field_option,omitempty"`
	// Must be validated to be a valid module path
	Module string `json:"module,omitempty" yaml:"module,omitempty"`
	// Must be normalized and validated
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
	// Must be validated to be a valid field name
	Field string `json:"field,omitempty" yaml:"field,omitempty"`
	// Required
	Value interface{} `json:"value,omitempty" yaml:"value,omitempty"`
}

// externalGeneratePluginConfigV2 is an external plugin configuration.
type externalGeneratePluginConfigV2 struct {
	// Only one of Remote, Binary, Wasm, ProtocBuiltin can be set
	Remote *string `json:"remote,omitempty" yaml:"remote,omitempty"`
	// Can be multiple arguments
	// All arguments must be strings
	Binary        interface{} `json:"binary,omitempty" yaml:"binary,omitempty"`
	ProtocBuiltin *string     `json:"protoc_builtin,omitempty" yaml:"protoc_builtin,omitempty"`
	// Only valid with Remote
	Revision *int `json:"revision,omitempty" yaml:"revision,omitempty"`
	// Only valid with ProtocBuiltin
	ProtocPath *string `json:"protoc_path,omitempty" yaml:"protoc_path,omitempty"`
	// Required
	Out string `json:"out,omitempty" yaml:"out,omitempty"`
	// Can be one string or multiple strings
	Opt            interface{} `json:"opt,omitempty" yaml:"opt,omitempty"`
	IncludeImports bool        `json:"include_imports,omitempty" yaml:"include_imports,omitempty"`
	IncludeWKT     bool        `json:"include_wkt,omitempty" yaml:"include_wkt,omitempty"`
	// Must be a valid Strategy, only valid with ProtoBuiltin and Binary
	Strategy *string `json:"strategy,omitempty" yaml:"strategy,omitempty"`
}

// externalInputConfigV2 is an external input configuration.
type externalInputConfigV2 struct {
	// One and only one of Module, Directory, ProtoFile, Tarball, ZipArchive, BinaryImage,
	// JSONImage and GitRepo must be specified as the format.
	Module      *string `json:"module,omitempty" yaml:"module,omitempty"`
	Directory   *string `json:"directory,omitempty" yaml:"directory,omitempty"`
	ProtoFile   *string `json:"proto_file,omitempty" yaml:"proto_file,omitempty"`
	Tarball     *string `json:"tarball,omitempty" yaml:"tarball,omitempty"`
	ZipArchive  *string `json:"zip_archive,omitempty" yaml:"zip_archive,omitempty"`
	BinaryImage *string `json:"binary_image,omitempty" yaml:"binary_image,omitempty"`
	JSONImage   *string `json:"json_image,omitempty" yaml:"json_image,omitempty"`
	TextImage   *string `json:"text_image,omitempty" yaml:"text_image,omitempty"`
	GitRepo     *string `json:"git_repo,omitempty" yaml:"git_repo,omitempty"`
	// Compression, StripComponents, Subdir, Branch, Tag, Ref, Depth, RecurseSubmodules
	// and IncludePackageFils are available for only some formats.
	Compression         *string `json:"compression,omitempty" yaml:"compression,omitempty"`
	StripComponents     *uint32 `json:"strip_components,omitempty" yaml:"strip_components,omitempty"`
	Subdir              *string `json:"subdir,omitempty" yaml:"subdir,omitempty"`
	Branch              *string `json:"branch,omitempty" yaml:"branch,omitempty"`
	Tag                 *string `json:"tag,omitempty" yaml:"tag,omitempty"`
	Ref                 *string `json:"ref,omitempty" yaml:"ref,omitempty"`
	Depth               *uint32 `json:"depth,omitempty" yaml:"depth,omitempty"`
	RecurseSubmodules   *bool   `json:"recurse_submodules,omitempty" yaml:"recurse_submodules,omitempty"`
	IncludePackageFiles *bool   `json:"include_package_files,omitempty" yaml:"include_package_files,omitempty"`
	// Types, IncludePaths and ExcludePaths are available for all formats.
	Types        []string `json:"types,omitempty" yaml:"types,omitempty"`
	IncludePaths []string `json:"include_paths,omitempty" yaml:"include_paths,omitempty"`
	ExcludePaths []string `json:"exclude_paths,omitempty" yaml:"exclude_paths,omitempty"`
}

// Copyright 2020-2024 Buf Technologies, Inc.
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

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

const (
	defaultBufGenYAMLFileName    = "buf.gen.yaml"
	defaultBufGenYAMLFileVersion = FileVersionV1Beta1
)

var (
	// ordered
	bufGenYAMLFileNames                       = []string{defaultBufGenYAMLFileName}
	bufGenYAMLFileNameToSupportedFileVersions = map[string]map[FileVersion]struct{}{
		defaultBufGenYAMLFileName: {
			FileVersionV1Beta1: struct{}{},
			FileVersionV1:      struct{}{},
			FileVersionV2:      struct{}{},
		},
	}
)

// BufGenYAMLFile represents a buf.gen.yaml file.
//
// For v2, generation configuration has been merged into BufYAMLFiles.
type BufGenYAMLFile interface {
	File

	// GenerateConfig returns the generate config.
	GenerateConfig() GenerateConfig
	// InputConfigs returns the input configs, which can be empty.
	InputConfigs() []InputConfig

	isBufGenYAMLFile()
}

// NewBufGenYAMLFile returns a new BufGenYAMLFile. It is validated given each
// parameter is validated.
func NewBufGenYAMLFile(
	fileVersion FileVersion,
	generateConfig GenerateConfig,
	inputConfigs []InputConfig,
) BufGenYAMLFile {
	return newBufGenYAMLFile(
		fileVersion,
		nil,
		generateConfig,
		inputConfigs,
	)
}

// GetBufGenYAMLFileForPrefix gets the buf.gen.yaml file at the given bucket prefix.
//
// The buf.gen.yaml file will be attempted to be read at prefix/buf.gen.yaml.
func GetBufGenYAMLFileForPrefix(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
) (BufGenYAMLFile, error) {
	return getFileForPrefix(ctx, bucket, prefix, bufGenYAMLFileNames, bufGenYAMLFileNameToSupportedFileVersions, readBufGenYAMLFile)
}

// GetBufGenYAMLFileForPrefix gets the buf.gen.yaml file version at the given bucket prefix.
//
// The buf.gen.yaml file will be attempted to be read at prefix/buf.gen.yaml.
func GetBufGenYAMLFileVersionForPrefix(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
) (FileVersion, error) {
	return getFileVersionForPrefix(ctx, bucket, prefix, bufGenYAMLFileNames, bufGenYAMLFileNameToSupportedFileVersions, true, FileVersionV2, defaultBufGenYAMLFileVersion)
}

// PutBufGenYAMLFileForPrefix puts the buf.gen.yaml file at the given bucket prefix.
//
// The buf.gen.yaml file will be attempted to be written to prefix/buf.gen.yaml.
// The buf.gen.yaml file will be written atomically.
func PutBufGenYAMLFileForPrefix(
	ctx context.Context,
	bucket storage.WriteBucket,
	prefix string,
	bufYAMLFile BufGenYAMLFile,
) error {
	return putFileForPrefix(ctx, bucket, prefix, bufYAMLFile, defaultBufGenYAMLFileName, bufGenYAMLFileNameToSupportedFileVersions, writeBufGenYAMLFile)
}

// ReadBufGenYAMLFile reads the BufGenYAMLFile from the io.Reader.
func ReadBufGenYAMLFile(reader io.Reader) (BufGenYAMLFile, error) {
	return readFile(reader, "", readBufGenYAMLFile)
}

// WriteBufGenYAMLFile writes the BufGenYAMLFile to the io.Writer.
func WriteBufGenYAMLFile(writer io.Writer, bufGenYAMLFile BufGenYAMLFile) error {
	return writeFile(writer, bufGenYAMLFile, writeBufGenYAMLFile)
}

// *** PRIVATE ***

type bufGenYAMLFile struct {
	generateConfig GenerateConfig
	inputConfigs   []InputConfig

	fileVersion FileVersion
	objectData  ObjectData
}

func newBufGenYAMLFile(
	fileVersion FileVersion,
	objectData ObjectData,
	generateConfig GenerateConfig,
	inputConfigs []InputConfig,
) *bufGenYAMLFile {
	return &bufGenYAMLFile{
		fileVersion:    fileVersion,
		objectData:     objectData,
		generateConfig: generateConfig,
		inputConfigs:   inputConfigs,
	}
}

func (g *bufGenYAMLFile) FileVersion() FileVersion {
	return g.fileVersion
}

func (*bufGenYAMLFile) FileType() FileType {
	return FileTypeBufGenYAML
}

func (g *bufGenYAMLFile) ObjectData() ObjectData {
	return g.objectData
}

func (g *bufGenYAMLFile) GenerateConfig() GenerateConfig {
	return g.generateConfig
}

func (g *bufGenYAMLFile) InputConfigs() []InputConfig {
	return g.inputConfigs
}

func (*bufGenYAMLFile) isBufGenYAMLFile() {}
func (*bufGenYAMLFile) isFile()           {}
func (*bufGenYAMLFile) isFileInfo()       {}

func readBufGenYAMLFile(
	data []byte,
	objectData ObjectData,
	allowJSON bool,
) (BufGenYAMLFile, error) {
	// We have always enforced that buf.gen.yamls have file versions.
	fileVersion, err := getFileVersionForData(data, allowJSON, true, bufGenYAMLFileNameToSupportedFileVersions, FileVersionV2, defaultBufGenYAMLFileVersion)
	if err != nil {
		return nil, err
	}
	switch fileVersion {
	case FileVersionV1Beta1:
		var externalGenYAMLFile externalBufGenYAMLFileV1Beta1
		if err := getUnmarshalStrict(allowJSON)(data, &externalGenYAMLFile); err != nil {
			return nil, fmt.Errorf("invalid as version %v: %w", fileVersion, err)
		}
		generateConfig, err := newGenerateConfigFromExternalFileV1Beta1(externalGenYAMLFile)
		if err != nil {
			return nil, err
		}
		return newBufGenYAMLFile(
			fileVersion,
			objectData,
			generateConfig,
			nil,
		), nil
	case FileVersionV1:
		var externalGenYAMLFile externalBufGenYAMLFileV1
		if err := getUnmarshalStrict(allowJSON)(data, &externalGenYAMLFile); err != nil {
			return nil, fmt.Errorf("invalid as version %v: %w", fileVersion, err)
		}
		generateConfig, err := newGenerateConfigFromExternalFileV1(externalGenYAMLFile)
		if err != nil {
			return nil, err
		}
		return newBufGenYAMLFile(
			fileVersion,
			objectData,
			generateConfig,
			nil,
		), nil
	case FileVersionV2:
		var externalGenYAMLFile externalBufGenYAMLFileV2
		if err := getUnmarshalStrict(allowJSON)(data, &externalGenYAMLFile); err != nil {
			return nil, fmt.Errorf("invalid as version %v: %w", fileVersion, err)
		}
		generateConfig, err := newGenerateConfigFromExternalFileV2(externalGenYAMLFile)
		if err != nil {
			return nil, err
		}
		inputConfigs, err := slicesext.MapError(
			externalGenYAMLFile.Inputs,
			newInputConfigFromExternalV2,
		)
		if err != nil {
			return nil, err
		}
		return newBufGenYAMLFile(
			fileVersion,
			objectData,
			generateConfig,
			inputConfigs,
		), nil
	default:
		// This is a system error since we've already parsed.
		return nil, syserror.Newf("unknown FileVersion: %v", fileVersion)
	}
}

func writeBufGenYAMLFile(writer io.Writer, bufGenYAMLFile BufGenYAMLFile) error {
	// Regardless of version, we write the file as v2:
	externalPluginConfigsV2, err := slicesext.MapError(
		bufGenYAMLFile.GenerateConfig().GeneratePluginConfigs(),
		newExternalGeneratePluginConfigV2FromPluginConfig,
	)
	if err != nil {
		return err
	}
	externalManagedConfigV2, err := newExternalManagedConfigV2FromGenerateManagedConfig(
		bufGenYAMLFile.GenerateConfig().GenerateManagedConfig(),
	)
	if err != nil {
		return err
	}
	externalInputConfigsV2, err := slicesext.MapError(
		bufGenYAMLFile.InputConfigs(),
		newExternalInputConfigV2FromInputConfig,
	)
	if err != nil {
		return err
	}
	externalBufGenYAMLFileV2 := externalBufGenYAMLFileV2{
		Version: FileVersionV2.String(),
		Plugins: externalPluginConfigsV2,
		Managed: externalManagedConfigV2,
		Inputs:  externalInputConfigsV2,
	}
	data, err := encoding.MarshalYAML(&externalBufGenYAMLFileV2)
	if err != nil {
		return err
	}
	_, err = writer.Write(data)
	return err
}

// externalBufGenYAMLFileV1Beta1 represents the v1beta buf.gen.yaml file.
type externalBufGenYAMLFileV1Beta1 struct {
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
	// Managed is whether managed mode is enabled.
	Managed bool                                  `json:"managed,omitempty" yaml:"managed,omitempty"`
	Plugins []externalGeneratePluginConfigV1Beta1 `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	Options externalGenerateManagedConfigV1Beta1  `json:"options,omitempty" yaml:"options,omitempty"`
}

// externalGeneratePluginConfigV1Beta1 represents a single plugin conifg in a v1beta1 buf.gen.yaml file.
type externalGeneratePluginConfigV1Beta1 struct {
	Name     string      `json:"name,omitempty" yaml:"name,omitempty"`
	Out      string      `json:"out,omitempty" yaml:"out,omitempty"`
	Opt      interface{} `json:"opt,omitempty" yaml:"opt,omitempty"`
	Path     string      `json:"path,omitempty" yaml:"path,omitempty"`
	Strategy string      `json:"strategy,omitempty" yaml:"strategy,omitempty"`
}

// externalGenerateManagedConfigV1Beta1 represents the options (for managed mode) config in a v1beta1 buf.gen.yaml file.
type externalGenerateManagedConfigV1Beta1 struct {
	CcEnableArenas    *bool  `json:"cc_enable_arenas,omitempty" yaml:"cc_enable_arenas,omitempty"`
	JavaMultipleFiles *bool  `json:"java_multiple_files,omitempty" yaml:"java_multiple_files,omitempty"`
	OptimizeFor       string `json:"optimize_for,omitempty" yaml:"optimize_for,omitempty"`
}

// externalBufGenYAMLFileV1 represents the v1 buf.gen.yaml file.
type externalBufGenYAMLFileV1 struct {
	Version string                           `json:"version,omitempty" yaml:"version,omitempty"`
	Plugins []externalGeneratePluginConfigV1 `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	Managed externalGenerateManagedConfigV1  `json:"managed,omitempty" yaml:"managed,omitempty"`
	Types   externalTypesConfigV1            `json:"types,omitempty" yaml:"types,omitempty"`
}

// externalGeneratePluginConfigV1 represents a single plugin config in a v1 buf.gen.yaml file.
type externalGeneratePluginConfigV1 struct {
	// Exactly one of Plugin and Name is required.
	// Plugin is the key for a local or remote plugin.
	Plugin string `json:"plugin,omitempty" yaml:"plugin,omitempty"`
	// Name is the key for a local plugin.
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	// Out is required.
	Out      string `json:"out,omitempty" yaml:"out,omitempty"`
	Revision int    `json:"revision,omitempty" yaml:"revision,omitempty"`
	// Opt can be one string or multiple strings.
	Opt interface{} `json:"opt,omitempty" yaml:"opt,omitempty"`
	// Path can be one string or multiple strings.
	Path       any    `json:"path,omitempty" yaml:"path,omitempty"`
	ProtocPath any    `json:"protoc_path,omitempty" yaml:"protoc_path,omitempty"`
	Strategy   string `json:"strategy,omitempty" yaml:"strategy,omitempty"`
}

// externalGenerateManagedConfigV1 represents the managed mode config in a v1 buf.gen.yaml file.
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
	// Override maps from a file option to a file path then to the value.
	Override map[string]map[string]string `json:"override,omitempty" yaml:"override,omitempty"`
}

// externalJavaPackagePrefixConfigV1 represents the java_package_prefix config in a v1 buf.gen.yaml file.
type externalJavaPackagePrefixConfigV1 struct {
	Default  string            `json:"default,omitempty" yaml:"default,omitempty"`
	Except   []string          `json:"except,omitempty" yaml:"except,omitempty"`
	Override map[string]string `json:"override,omitempty" yaml:"override,omitempty"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface. This is done to maintain backward compatibility
// of accepting a plain string value for java_package_prefix.
func (e *externalJavaPackagePrefixConfigV1) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return e.unmarshalWith(unmarshal)
}

// UnmarshalJSON implements the json.Unmarshaler interface. This is done to maintain backward compatibility
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

// isEmpty returns true if the config is empty.
func (e externalJavaPackagePrefixConfigV1) isEmpty() bool {
	return e.Default == "" &&
		len(e.Except) == 0 &&
		len(e.Override) == 0
}

// externalOptimizeForConfigV1 represents the optimize_for config in a v1 buf.gen.yaml file.
type externalOptimizeForConfigV1 struct {
	Default  string            `json:"default,omitempty" yaml:"default,omitempty"`
	Except   []string          `json:"except,omitempty" yaml:"except,omitempty"`
	Override map[string]string `json:"override,omitempty" yaml:"override,omitempty"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface. This is done to maintain backward compatibility
// of accepting a plain string value for optimize_for.
func (e *externalOptimizeForConfigV1) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return e.unmarshalWith(unmarshal)
}

// UnmarshalJSON implements the json.Unmarshaler interface. This is done to maintain backward compatibility
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

// isEmpty returns true if the config is empty
func (e externalOptimizeForConfigV1) isEmpty() bool {
	return e.Default == "" &&
		len(e.Except) == 0 &&
		len(e.Override) == 0
}

// externalGoPackagePrefixConfigV1 represents the go_package_prefix config in a v1 buf.gen.yaml file.
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

// externalCsharpNamespaceConfigV1 represents the external csharp_namespace config in a v1 buf.gen.yaml file.
type externalCsharpNamespaceConfigV1 struct {
	Except   []string          `json:"except,omitempty" yaml:"except,omitempty"`
	Override map[string]string `json:"override,omitempty" yaml:"override,omitempty"`
}

// isEmpty returns true if the config is empty.
func (e externalCsharpNamespaceConfigV1) isEmpty() bool {
	return len(e.Except) == 0 &&
		len(e.Override) == 0
}

// externalRubyPackageConfigV1 represents the ruby_package config in a v1 buf.gen.yaml file.
type externalRubyPackageConfigV1 struct {
	Except   []string          `json:"except,omitempty" yaml:"except,omitempty"`
	Override map[string]string `json:"override,omitempty" yaml:"override,omitempty"`
}

// isEmpty returns true is the config is empty.
func (e externalRubyPackageConfigV1) isEmpty() bool {
	return len(e.Except) == 0 && len(e.Override) == 0
}

// externalObjcClassPrefixConfigV1 represents the objc_class_prefix config in a v1 buf.gen.yaml file.
type externalObjcClassPrefixConfigV1 struct {
	Default  string            `json:"default,omitempty" yaml:"default,omitempty"`
	Except   []string          `json:"except,omitempty" yaml:"except,omitempty"`
	Override map[string]string `json:"override,omitempty" yaml:"override,omitempty"`
}

// isEmpty returns true is the config is empty.
func (e externalObjcClassPrefixConfigV1) isEmpty() bool {
	return e.Default == "" &&
		len(e.Except) == 0 &&
		len(e.Override) == 0
}

// externalTypesConfigV1 represents the types config in a v1 buf.gen.yaml file.
type externalTypesConfigV1 struct {
	Include []string `json:"include,omitempty" yaml:"include"`
}

// externalBufGenYAMLFileV2 represents the v2 buf.gen.yaml file.
type externalBufGenYAMLFileV2 struct {
	Version string                           `json:"version,omitempty" yaml:"version,omitempty"`
	Managed externalGenerateManagedConfigV2  `json:"managed,omitempty" yaml:"managed,omitempty"`
	Plugins []externalGeneratePluginConfigV2 `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	Inputs  []externalInputConfigV2          `json:"inputs,omitempty" yaml:"inputs,omitempty"`
}

// externalGeneratePluginConfigV2 represents a single plugin config in a v2 buf.gen.yaml file.
type externalGeneratePluginConfigV2 struct {
	// Exactly one of Remote, Local and ProtocBuiltin is required.
	Remote *string `json:"remote,omitempty" yaml:"remote,omitempty"`
	// Revision is only valid with Remote set.
	Revision *int `json:"revision,omitempty" yaml:"revision,omitempty"`
	// Local is the local path (either relative or absolute) to a binary or other runnable program which
	// implements the protoc plugin interface. This can be one string (the program) or multiple (remaining
	// strings are arguments to the program).
	Local any `json:"local,omitempty" yaml:"local,omitempty"`
	// ProtocBuiltin is the protoc built-in plugin name, in the form of 'java' instead of 'protoc-gen-java'.
	ProtocBuiltin *string `json:"protoc_builtin,omitempty" yaml:"protoc_builtin,omitempty"`
	// ProtocPath is only valid with ProtocBuiltin. This can be one string (the path to protoc) or multiple
	// (remaining strings are extra args to pass to protoc).
	ProtocPath any `json:"protoc_path,omitempty" yaml:"protoc_path,omitempty"`
	// Out is required.
	Out string `json:"out,omitempty" yaml:"out,omitempty"`
	// Opt can be one string or multiple strings.
	Opt            any  `json:"opt,omitempty" yaml:"opt,omitempty"`
	IncludeImports bool `json:"include_imports,omitempty" yaml:"include_imports,omitempty"`
	IncludeWKT     bool `json:"include_wkt,omitempty" yaml:"include_wkt,omitempty"`
	// Strategy is only valid with ProtoBuiltin and Local.
	Strategy *string `json:"strategy,omitempty" yaml:"strategy,omitempty"`
}

// externalGenerateManagedConfigV2 represents the managed mode config in a v2 buf.gen.yaml file.
type externalGenerateManagedConfigV2 struct {
	Enabled  bool                              `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Disable  []externalManagedDisableConfigV2  `json:"disable,omitempty" yaml:"disable,omitempty"`
	Override []externalManagedOverrideConfigV2 `json:"override,omitempty" yaml:"override,omitempty"`
}

// externalManagedDisableConfigV2 represents a disable rule in managed mode in a v2 buf.gen.yaml file.
type externalManagedDisableConfigV2 struct {
	// At least one field must be set.
	// At most one of FileOption and FieldOption can be set
	FileOption  string `json:"file_option,omitempty" yaml:"file_option,omitempty"`
	FieldOption string `json:"field_option,omitempty" yaml:"field_option,omitempty"`
	Module      string `json:"module,omitempty" yaml:"module,omitempty"`
	// Path must be normalized.
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
	// Field must not be set if FileOption is set.
	Field string `json:"field,omitempty" yaml:"field,omitempty"`
}

// externalManagedOverrideConfigV2 represents an override rule in managed mode in a v2 buf.gen.yaml file.
type externalManagedOverrideConfigV2 struct {
	// Exactly one of FileOpion and FieldOption must be set.
	FileOption  string `json:"file_option,omitempty" yaml:"file_option,omitempty"`
	FieldOption string `json:"field_option,omitempty" yaml:"field_option,omitempty"`
	Module      string `json:"module,omitempty" yaml:"module,omitempty"`
	// Path must be normalized.
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
	// Field must not be set if FileOption is set.
	Field string `json:"field,omitempty" yaml:"field,omitempty"`
	// Value is required
	Value interface{} `json:"value,omitempty" yaml:"value,omitempty"`
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
	YAMLImage   *string `json:"yaml_image,omitempty" yaml:"yaml_image,omitempty"`
	GitRepo     *string `json:"git_repo,omitempty" yaml:"git_repo,omitempty"`
	// Types, TargetPaths and ExcludePaths are available for all formats.
	Types        []string `json:"types,omitempty" yaml:"types,omitempty"`
	TargetPaths  []string `json:"paths,omitempty" yaml:"paths,omitempty"`
	ExcludePaths []string `json:"exclude_paths,omitempty" yaml:"exclude_paths,omitempty"`
	// The following options are available depending on input format.
	Compression         *string `json:"compression,omitempty" yaml:"compression,omitempty"`
	StripComponents     *uint32 `json:"strip_components,omitempty" yaml:"strip_components,omitempty"`
	Subdir              *string `json:"subdir,omitempty" yaml:"subdir,omitempty"`
	Branch              *string `json:"branch,omitempty" yaml:"branch,omitempty"`
	Commit              *string `json:"commit,omitempty" yaml:"commit,omitempty"`
	Tag                 *string `json:"tag,omitempty" yaml:"tag,omitempty"`
	Ref                 *string `json:"ref,omitempty" yaml:"ref,omitempty"`
	Depth               *uint32 `json:"depth,omitempty" yaml:"depth,omitempty"`
	RecurseSubmodules   *bool   `json:"recurse_submodules,omitempty" yaml:"recurse_submodules,omitempty"`
	IncludePackageFiles *bool   `json:"include_package_files,omitempty" yaml:"include_package_files,omitempty"`
}

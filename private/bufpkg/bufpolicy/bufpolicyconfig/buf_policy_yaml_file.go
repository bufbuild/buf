// Copyright 2020-2025 Buf Technologies, Inc.
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

package bufpolicyconfig

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

var (
	// defaultLintConfigV2 is the default lint config for v2.
	defaultLintConfigV2 bufconfig.LintConfig = bufconfig.NewLintConfig(
		bufconfig.NewEnabledCheckConfigForUseIDsAndCategories(
			bufconfig.FileVersionV2,
			nil,
			true, // Disable builtin is true by default.
		),
		"",
		false,
		false,
		false,
		"",
		false, // Policy configs do not allow comment ignores.
	)

	defaultBreakingConfigV2 bufconfig.BreakingConfig = bufconfig.NewBreakingConfig(
		bufconfig.NewEnabledCheckConfigForUseIDsAndCategories(
			bufconfig.FileVersionV2,
			nil,
			true, // Disable builtin is true by default.
		),
		false,
	)
)

// BufPolicyYAMLFile represents a Policy config file.
type BufPolicyYAMLFile interface {
	File

	// Name returns the name for the File.
	Name() string
	// LintConfig returns the LintConfig for the File.
	LintConfig() bufconfig.LintConfig
	// BreakingConfig returns the BreakingConfig for the File.
	BreakingConfig() bufconfig.BreakingConfig
	// PluginConfigs returns the PluginConfigs for the File.
	PluginConfigs() []bufconfig.PluginConfig

	isBufPolicyYAMLFile()
}

// NewBufPolicyYAMLFile returns a new validated BufPolicyYAMLFile.
func NewBufPolicyYAMLFile(
	name string,
	lintConfig bufconfig.LintConfig,
	breakingConfig bufconfig.BreakingConfig,
	pluginConfigs []bufconfig.PluginConfig,
) (BufPolicyYAMLFile, error) {
	return newBufPolicyYAMLFile(
		nil,
		name,
		lintConfig,
		breakingConfig,
		pluginConfigs,
	)
}

// GetBufPolicyYAMLFile gets the PolicyYAMLFile at the given bucket path.
func GetBufPolicyYAMLFile(
	ctx context.Context,
	bucket storage.ReadBucket,
	path string,
) (BufPolicyYAMLFile, error) {
	return getFile(ctx, bucket, path, readBufPolicyYAMLFile)
}

// PutBufPolicyYAMLFile puts the PolicyYAMLFile at the given bucket path.
//
// The PolicyYAMLFile file will be attempted to be written to filePath.
// The PolicyYAMLFile file will be written atomically.
func PutBufPolicyYAMLFile(
	ctx context.Context,
	bucket storage.WriteBucket,
	path string,
	bufYAMLFile BufPolicyYAMLFile,
) error {
	return putFile(ctx, bucket, path, bufYAMLFile, writeBufPolicyYAMLFile)
}

// ReadBufPolicyYAMLFile reads the BufPolicyYAMLFile from the io.Reader.
func ReadBufPolicyYAMLFile(reader io.Reader, fileName string) (BufPolicyYAMLFile, error) {
	return readFile(reader, fileName, readBufPolicyYAMLFile)
}

// WriteBufPolicyYAMLFile writes the BufPolicyYAMLFile to the io.Writer.
func WriteBufPolicyYAMLFile(writer io.Writer, bufPolicyYAMLFile BufPolicyYAMLFile) error {
	return writeFile(writer, bufPolicyYAMLFile, writeBufPolicyYAMLFile)
}

// *** PRIVATE ***

type bufPolicyYAMLFile struct {
	fileVersion    bufconfig.FileVersion
	objectData     bufconfig.ObjectData
	name           string
	lintConfig     bufconfig.LintConfig
	breakingConfig bufconfig.BreakingConfig
	pluginConfigs  []bufconfig.PluginConfig
}

func newBufPolicyYAMLFile(
	objectData bufconfig.ObjectData,
	name string,
	lintConfig bufconfig.LintConfig,
	breakingConfig bufconfig.BreakingConfig,
	pluginConfigs []bufconfig.PluginConfig,
) (*bufPolicyYAMLFile, error) {
	var validationErr error
	if lintConfig != nil {
		if lintConfig.FileVersion() != bufconfig.FileVersionV2 {
			validationErr = errors.Join(validationErr, fmt.Errorf("lintConfig.FileVersion() must be %s", bufconfig.FileVersionV2))
		}
		if len(lintConfig.IgnorePaths()) > 0 {
			validationErr = errors.Join(validationErr, fmt.Errorf("lintConfig.IgnorePaths() must be empty"))
		}
		if len(lintConfig.IgnoreIDOrCategoryToPaths()) > 0 {
			validationErr = errors.Join(validationErr, fmt.Errorf("lintConfig.IgnoreIDOrCategoryToPaths() must be empty"))
		}
		if lintConfig.DisableBuiltin() {
			validationErr = errors.Join(validationErr, fmt.Errorf("lintConfig.DisableBuiltin() must be false"))
		}
		if lintConfig.AllowCommentIgnores() {
			validationErr = errors.Join(validationErr, fmt.Errorf("lintConfig.AllowCommentIgnores() must be false"))
		}
	}
	if breakingConfig != nil {
		if breakingConfig.FileVersion() != bufconfig.FileVersionV2 {
			validationErr = errors.Join(validationErr, fmt.Errorf("breakingConfig.FileVersion() must be %s", bufconfig.FileVersionV2))
		}
		if len(breakingConfig.IgnorePaths()) > 0 {
			validationErr = errors.Join(validationErr, fmt.Errorf("breakingConfig.IgnorePaths() must be empty"))
		}
		if len(breakingConfig.IgnoreIDOrCategoryToPaths()) > 0 {
			validationErr = errors.Join(validationErr, fmt.Errorf("breakingConfig.IgnoreIDOrCategoryToPaths() must be empty"))
		}
		if breakingConfig.DisableBuiltin() {
			validationErr = errors.Join(validationErr, fmt.Errorf("breakingConfig.DisableBuiltin() must be false"))
		}
	}
	if validationErr != nil {
		return nil, validationErr
	}
	return &bufPolicyYAMLFile{
		fileVersion:    bufconfig.FileVersionV2,
		name:           name,
		lintConfig:     lintConfig,
		breakingConfig: breakingConfig,
		pluginConfigs:  pluginConfigs,
	}, nil
}

func (p *bufPolicyYAMLFile) FileVersion() bufconfig.FileVersion {
	return p.fileVersion
}

func (p *bufPolicyYAMLFile) ObjectData() bufconfig.ObjectData {
	return p.objectData
}

func (p *bufPolicyYAMLFile) Name() string {
	return p.name
}

func (p *bufPolicyYAMLFile) LintConfig() bufconfig.LintConfig {
	if p.lintConfig == nil {
		return defaultLintConfigV2
	}
	return p.lintConfig
}

func (p *bufPolicyYAMLFile) BreakingConfig() bufconfig.BreakingConfig {
	if p.breakingConfig == nil {
		return defaultBreakingConfigV2
	}
	return p.breakingConfig
}

func (p *bufPolicyYAMLFile) PluginConfigs() []bufconfig.PluginConfig {
	return slices.Clone(p.pluginConfigs)
}

func (*bufPolicyYAMLFile) isBufPolicyYAMLFile() {}
func (*bufPolicyYAMLFile) isFile()              {}

// externalBufPolicyYAMLFileV2 represents the v2 buf.policy.yaml file.
type externalBufPolicyYAMLFileV2 struct {
	Version  string                              `json:"version,omitempty" yaml:"version,omitempty"`
	Name     string                              `json:"name,omitempty" yaml:"name,omitempty"`
	Lint     externalBufPolicyYAMLFileLintV2     `json:"lint,omitempty" yaml:"lint,omitempty"`
	Breaking externalBufPolicyYAMLFileBreakingV2 `json:"breaking,omitempty" yaml:"breaking,omitempty"`
	Plugins  []externalBufPolicyYAMLFilePluginV2 `json:"plugins,omitempty" yaml:"plugins,omitempty"`
}

func readBufPolicyYAMLFile(
	data []byte,
	objectData bufconfig.ObjectData,
	allowJSON bool,
) (BufPolicyYAMLFile, error) {
	fileVersion, err := getFileVersionForData(data, allowJSON)
	if err != nil {
		return nil, err
	}
	if objectData != nil {
		if err := validateSupportedFileVersion(objectData.Name(), fileVersion); err != nil {
			return nil, err
		}
	}
	var externalBufPolicyYAMLFile externalBufPolicyYAMLFileV2
	if err := getUnmarshalStrict(allowJSON)(data, &externalBufPolicyYAMLFile); err != nil {
		return nil, fmt.Errorf("invalid as version %v: %w", fileVersion, err)
	}

	var lintConfig bufconfig.LintConfig
	if !externalBufPolicyYAMLFile.Lint.isEmpty() {
		lintConfig, err = getLintConfigForExternalLintV2(
			externalBufPolicyYAMLFile.Lint,
		)
		if err != nil {
			return nil, err
		}
	}
	var breakingConfig bufconfig.BreakingConfig
	if !externalBufPolicyYAMLFile.Breaking.isEmpty() {
		breakingConfig, err = getBreakingConfigForExternalBreaking(
			externalBufPolicyYAMLFile.Breaking,
		)
		if err != nil {
			return nil, err
		}
	}
	var pluginConfigs []bufconfig.PluginConfig
	for _, externalPluginConfig := range externalBufPolicyYAMLFile.Plugins {
		pluginConfig, err := newPluginConfigForExternalPluginV2(externalPluginConfig)
		if err != nil {
			return nil, err
		}
		pluginConfigs = append(pluginConfigs, pluginConfig)
	}
	return newBufPolicyYAMLFile(
		objectData,
		externalBufPolicyYAMLFile.Name,
		lintConfig,
		breakingConfig,
		pluginConfigs,
	)
}

func writeBufPolicyYAMLFile(writer io.Writer, bufPolicyYAMLFile BufPolicyYAMLFile) error {
	fileVersion := bufPolicyYAMLFile.FileVersion()
	if fileVersion != bufconfig.FileVersionV2 {
		// This is effectively a system error.
		return syserror.Wrap(newUnsupportedFileVersionError("", fileVersion))
	}
	var externalLint externalBufPolicyYAMLFileLintV2
	if lintConfig := bufPolicyYAMLFile.LintConfig(); lintConfig != nil {
		externalLint = getExternalLintForLintConfig(lintConfig)
	}
	var externalBreaking externalBufPolicyYAMLFileBreakingV2
	if breakingConfig := bufPolicyYAMLFile.BreakingConfig(); breakingConfig != nil {
		externalBreaking = getExternalBreakingForBreakingConfig(breakingConfig)
	}
	var externalPlugins []externalBufPolicyYAMLFilePluginV2
	for _, pluginConfig := range bufPolicyYAMLFile.PluginConfigs() {
		externalPlugin, err := newExternalPluginV2ForPluginConfig(pluginConfig)
		if err != nil {
			return syserror.Wrap(err)
		}
		externalPlugins = append(externalPlugins, externalPlugin)
	}
	externalBufPolicyYAMLFile := externalBufPolicyYAMLFileV2{
		Version:  fileVersion.String(),
		Name:     bufPolicyYAMLFile.Name(),
		Lint:     externalLint,
		Breaking: externalBreaking,
		Plugins:  externalPlugins,
	}
	data, err := encoding.MarshalYAML(&externalBufPolicyYAMLFile)
	if err != nil {
		return err
	}
	_, err = writer.Write(data)
	return err
}

// externalBufPolicyYAMLFileLintV2 represents lint configuration within a v2 buf.policy.yaml file.
//
// It is a subset of the v2 buf.yaml lint configuration.
type externalBufPolicyYAMLFileLintV2 struct {
	Use    []string `json:"use,omitempty" yaml:"use,omitempty"`
	Except []string `json:"except,omitempty" yaml:"except,omitempty"`
	// Ignore are the paths to ignore.
	EnumZeroValueSuffix                  string `json:"enum_zero_value_suffix,omitempty" yaml:"enum_zero_value_suffix,omitempty"`
	RPCAllowSameRequestResponse          bool   `json:"rpc_allow_same_request_response,omitempty" yaml:"rpc_allow_same_request_response,omitempty"`
	RPCAllowGoogleProtobufEmptyRequests  bool   `json:"rpc_allow_google_protobuf_empty_requests,omitempty" yaml:"rpc_allow_google_protobuf_empty_requests,omitempty"`
	RPCAllowGoogleProtobufEmptyResponses bool   `json:"rpc_allow_google_protobuf_empty_responses,omitempty" yaml:"rpc_allow_google_protobuf_empty_responses,omitempty"`
	ServiceSuffix                        string `json:"service_suffix,omitempty" yaml:"service_suffix,omitempty"`
}

func (el externalBufPolicyYAMLFileLintV2) isEmpty() bool {
	return len(el.Use) == 0 &&
		len(el.Except) == 0 &&
		el.EnumZeroValueSuffix == "" &&
		!el.RPCAllowSameRequestResponse &&
		!el.RPCAllowGoogleProtobufEmptyRequests &&
		!el.RPCAllowGoogleProtobufEmptyResponses &&
		el.ServiceSuffix == ""
}

// externalBufPolicyYAMLFileBreakingV2 represents breaking configuration within a v2 buf.policy.yaml file.
//
// It is a subset of the v2 buf.yaml breaking configuration.
type externalBufPolicyYAMLFileBreakingV2 struct {
	Use                    []string `json:"use,omitempty" yaml:"use,omitempty"`
	Except                 []string `json:"except,omitempty" yaml:"except,omitempty"`
	IgnoreUnstablePackages bool     `json:"ignore_unstable_packages,omitempty" yaml:"ignore_unstable_packages,omitempty"`
}

func (eb externalBufPolicyYAMLFileBreakingV2) isEmpty() bool {
	return len(eb.Use) == 0 &&
		len(eb.Except) == 0 &&
		!eb.IgnoreUnstablePackages
}

// externalBufPolicyYAMLFilePluginV2 represents a single plugin config in a v2 buf.yaml file.
type externalBufPolicyYAMLFilePluginV2 struct {
	Plugin  any            `json:"plugin,omitempty" yaml:"plugin,omitempty"`
	Options map[string]any `json:"options,omitempty" yaml:"options,omitempty"`
}

func getLintConfigForExternalLintV2(externalLint externalBufPolicyYAMLFileLintV2) (bufconfig.LintConfig, error) {
	checkConfig, err := bufconfig.NewEnabledCheckConfig(
		bufconfig.FileVersionV2,
		externalLint.Use,
		externalLint.Except,
		nil,
		nil,
		false,
	)
	if err != nil {
		return nil, err
	}
	return bufconfig.NewLintConfig(
		checkConfig,
		externalLint.EnumZeroValueSuffix,
		externalLint.RPCAllowSameRequestResponse,
		externalLint.RPCAllowGoogleProtobufEmptyRequests,
		externalLint.RPCAllowGoogleProtobufEmptyResponses,
		externalLint.ServiceSuffix,
		false, // Comment ignores are not allowed in Policy files.
	), nil
}

func getExternalLintForLintConfig(lintConfig bufconfig.LintConfig) externalBufPolicyYAMLFileLintV2 {
	return externalBufPolicyYAMLFileLintV2{
		// Use and Except are already sorted.
		Use:                                  lintConfig.UseIDsAndCategories(),
		Except:                               lintConfig.ExceptIDsAndCategories(),
		EnumZeroValueSuffix:                  lintConfig.EnumZeroValueSuffix(),
		RPCAllowSameRequestResponse:          lintConfig.RPCAllowSameRequestResponse(),
		RPCAllowGoogleProtobufEmptyRequests:  lintConfig.RPCAllowGoogleProtobufEmptyRequests(),
		RPCAllowGoogleProtobufEmptyResponses: lintConfig.RPCAllowGoogleProtobufEmptyResponses(),
		ServiceSuffix:                        lintConfig.ServiceSuffix(),
	}
}

func getBreakingConfigForExternalBreaking(externalBreaking externalBufPolicyYAMLFileBreakingV2) (bufconfig.BreakingConfig, error) {
	checkConfig, err := bufconfig.NewEnabledCheckConfig(
		bufconfig.FileVersionV2,
		externalBreaking.Use,
		externalBreaking.Except,
		nil,
		nil,
		false,
	)
	if err != nil {
		return nil, err
	}
	return bufconfig.NewBreakingConfig(
		checkConfig,
		externalBreaking.IgnoreUnstablePackages,
	), nil
}

func getExternalBreakingForBreakingConfig(breakingConfig bufconfig.BreakingConfig) externalBufPolicyYAMLFileBreakingV2 {
	return externalBufPolicyYAMLFileBreakingV2{
		// Use and Except are already sorted.
		Use:                    breakingConfig.UseIDsAndCategories(),
		Except:                 breakingConfig.ExceptIDsAndCategories(),
		IgnoreUnstablePackages: breakingConfig.IgnoreUnstablePackages(),
	}
}

func newPluginConfigForExternalPluginV2(externalConfig externalBufPolicyYAMLFilePluginV2) (bufconfig.PluginConfig, error) {
	options := make(map[string]any)
	for key, value := range externalConfig.Options {
		if len(key) == 0 {
			return nil, errors.New("must specify option key")
		}
		// TODO: Validation here, how to expose from bufplugin?
		if value == nil {
			return nil, errors.New("must specify option value")
		}
		options[key] = value
	}
	// Plugins are specified as a path, remote reference, or Wasm file.
	path, err := encoding.InterfaceSliceOrStringToStringSlice(externalConfig.Plugin)
	if err != nil {
		return nil, err
	}
	if len(path) == 0 {
		return nil, errors.New("must specify a path to the plugin")
	}
	name, args := path[0], path[1:]
	// Remote plugins are specified as plugin references.
	if pluginRef, err := bufparse.ParseRef(path[0]); err == nil {
		// Check if the local filepath exists, if it does presume its
		// not a remote reference. Okay to use os.Stat instead of
		// os.Lstat.
		if _, err := os.Stat(path[0]); os.IsNotExist(err) {
			return bufconfig.NewRemoteWasmPluginConfig(
				pluginRef,
				options,
				args,
			)
		}
	}
	// Wasm plugins are suffixed with .wasm. Otherwise, it's a binary.
	if filepath.Ext(path[0]) == ".wasm" {
		return bufconfig.NewLocalWasmPluginConfig(
			name,
			options,
			args,
		)
	}
	return bufconfig.NewLocalPluginConfig(
		name,
		options,
		args,
	)
}

func newExternalPluginV2ForPluginConfig(
	config bufconfig.PluginConfig,
) (externalBufPolicyYAMLFilePluginV2, error) {
	externalBufYAMLFilePluginV2 := externalBufPolicyYAMLFilePluginV2{
		Options: config.Options(),
	}
	if args := config.Args(); len(args) > 0 {
		externalBufYAMLFilePluginV2.Plugin = append([]string{config.Name()}, args...)
	} else {
		externalBufYAMLFilePluginV2.Plugin = config.Name()
	}
	return externalBufYAMLFilePluginV2, nil
}

func validateSupportedFileVersion(fileName string, fileVersion bufconfig.FileVersion) error {
	switch fileVersion {
	case bufconfig.FileVersionV2:
		return nil
	default:
		return newUnsupportedFileVersionError(fileName, fileVersion)
	}
}

func newUnsupportedFileVersionError(name string, fileVersion bufconfig.FileVersion) error {
	if name == "" {
		return fmt.Errorf("%s is not supported", fileVersion)
	}
	return fmt.Errorf("%s is not supported for %s files", fileVersion, name)
}

type externalFileVersion struct {
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}

func getFileVersionForData(
	data []byte,
	allowJSON bool,
) (bufconfig.FileVersion, error) {
	var externalFileVersion externalFileVersion
	if err := getUnmarshalNonStrict(allowJSON)(data, &externalFileVersion); err != nil {
		return 0, err
	}
	switch externalFileVersion.Version {
	case bufconfig.FileVersionV1Beta1.String():
		return bufconfig.FileVersionV1Beta1, nil
	case bufconfig.FileVersionV1.String():
		return bufconfig.FileVersionV1, nil
	case bufconfig.FileVersionV2.String():
		return bufconfig.FileVersionV2, nil
	default:
		return 0, fmt.Errorf("unknown file version: %q", externalFileVersion.Version)
	}
}

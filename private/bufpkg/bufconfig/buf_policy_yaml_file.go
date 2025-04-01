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

package bufconfig

import (
	"context"
	"fmt"
	"io"
	"slices"

	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

const (
	defaultBufPolicyYAMLFileName    = "buf.policy.yaml"
	defaultBufPolicyYAMLFileVersion = FileVersionV2
)

var (
	bufPolicyYAMLFileNames                       = []string{defaultBufPolicyYAMLFileName}
	bufPolicyYAMLFileNameToSupportedFileVersions = map[string]map[FileVersion]struct{}{
		defaultBufPolicyYAMLFileName: {
			FileVersionV2: struct{}{},
		},
	}
)

// BufPolicyYAMLFile represents a buf.policy.yaml file.
type BufPolicyYAMLFile interface {
	File

	// Name returns the name for the File.
	Name() string
	// LintConfig returns the LintConfig for the File.
	LintConfig() LintConfig
	// BreakingConfig returns the BreakingConfig for the File.
	BreakingConfig() BreakingConfig
	// PluginConfigs returns the PluginConfigs for the File.
	PluginConfigs() []PluginConfig

	isBufPolicyYAMLFile()
}

// NewBufPolicyYAMLFile returns a new validated BufPolicyYAMLFile.
func NewBufPolicyYAMLFile(
	fileVersion FileVersion,
	name string,
	lintConfig LintConfig,
	breakingConfig BreakingConfig,
	pluginConfigs []PluginConfig,
) (BufPolicyYAMLFile, error) {
	return newBufPolicyYAMLFile(
		fileVersion,
		nil,
		name,
		lintConfig,
		breakingConfig,
		pluginConfigs,
	)
}

// GetBufPolicyYAMLFileForPrefix gets the buf.gen.yaml file at the given bucket prefix.
//
// The buf.gen.yaml file will be attempted to be read at prefix/buf.gen.yaml.
func GetBufPolicyYAMLFileForPrefix(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
) (BufPolicyYAMLFile, error) {
	return getFileForPrefix(ctx, bucket, prefix, bufPolicyYAMLFileNames, bufPolicyYAMLFileNameToSupportedFileVersions, readBufPolicyYAMLFile)
}

// GetBufPolicyYAMLFileVersionForPrefix gets the buf.gen.yaml file version at the given bucket prefix.
//
// The buf.gen.yaml file will be attempted to be read at prefix/buf.gen.yaml.
func GetBufPolicyYAMLFileVersionForPrefix(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
) (FileVersion, error) {
	return getFileVersionForPrefix(ctx, bucket, prefix, bufPolicyYAMLFileNames, bufPolicyYAMLFileNameToSupportedFileVersions, true, FileVersionV2, defaultBufPolicyYAMLFileVersion)
}

// PutBufPolicyYAMLFileForPrefix puts the buf.gen.yaml file at the given bucket prefix.
//
// The buf.gen.yaml file will be attempted to be written to prefix/buf.gen.yaml.
// The buf.gen.yaml file will be written atomically.
func PutBufPolicyYAMLFileForPrefix(
	ctx context.Context,
	bucket storage.WriteBucket,
	prefix string,
	bufYAMLFile BufPolicyYAMLFile,
) error {
	return putFileForPrefix(ctx, bucket, prefix, bufYAMLFile, defaultBufPolicyYAMLFileName, bufPolicyYAMLFileNameToSupportedFileVersions, writeBufPolicyYAMLFile)
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
	fileVersion    FileVersion
	objectData     ObjectData
	name           string
	lintConfig     LintConfig
	breakingConfig BreakingConfig
	pluginConfigs  []PluginConfig
}

func newBufPolicyYAMLFile(
	fileVersion FileVersion,
	objectData ObjectData,
	name string,
	lintConfig LintConfig,
	breakingConfig BreakingConfig,
	pluginConfigs []PluginConfig,
) (*bufPolicyYAMLFile, error) {
	return &bufPolicyYAMLFile{
		fileVersion:    fileVersion,
		objectData:     objectData,
		name:           name,
		lintConfig:     lintConfig,
		breakingConfig: breakingConfig,
		pluginConfigs:  pluginConfigs,
	}, nil
}

func (p *bufPolicyYAMLFile) FileVersion() FileVersion {
	return p.fileVersion
}

func (*bufPolicyYAMLFile) FileType() FileType {
	return FileTypeBufYAML
}

func (p *bufPolicyYAMLFile) ObjectData() ObjectData {
	return p.objectData
}

func (p *bufPolicyYAMLFile) Name() string {
	return p.name
}

func (p *bufPolicyYAMLFile) LintConfig() LintConfig {
	return p.lintConfig
}

func (p *bufPolicyYAMLFile) BreakingConfig() BreakingConfig {
	return p.breakingConfig
}

func (p *bufPolicyYAMLFile) PluginConfigs() []PluginConfig {
	return slices.Clone(p.pluginConfigs)
}
func (*bufPolicyYAMLFile) isBufPolicyYAMLFile() {}
func (*bufPolicyYAMLFile) isFile()              {}
func (*bufPolicyYAMLFile) isFileInfo()          {}

// externalBufPolicyYAMLFileV2 represents the v2 buf.policy.yaml file.
type externalBufPolicyYAMLFileV2 struct {
	Version  string                                 `json:"version,omitempty" yaml:"version,omitempty"`
	Name     string                                 `json:"name,omitempty" yaml:"name,omitempty"`
	Lint     externalBufYAMLFileLintV2              `json:"lint,omitempty" yaml:"lint,omitempty"`
	Breaking externalBufYAMLFileBreakingV1Beta1V1V2 `json:"breaking,omitempty" yaml:"breaking,omitempty"`
	Plugins  []externalBufYAMLFilePluginV2          `json:"plugins,omitempty" yaml:"plugins,omitempty"`
}

func readBufPolicyYAMLFile(
	data []byte,
	objectData ObjectData,
	allowJSON bool,
) (BufPolicyYAMLFile, error) {
	fileVersion, err := getFileVersionForData(data, allowJSON, true, bufPolicyYAMLFileNameToSupportedFileVersions, FileVersionV2, defaultBufPolicyYAMLFileVersion)
	if err != nil {
		return nil, err
	}
	if fileVersion != FileVersionV2 {
		fmt.Println("objectData", objectData)
		return nil, newUnsupportedFileVersionError(objectData.Name(), fileVersion)
	}
	var externalBufPolicyYAMLFile externalBufPolicyYAMLFileV2
	if err := getUnmarshalStrict(allowJSON)(data, &externalBufPolicyYAMLFile); err != nil {
		return nil, fmt.Errorf("invalid as version %v: %w", fileVersion, err)
	}

	var lintConfig LintConfig
	if !externalBufPolicyYAMLFile.Lint.isEmpty() {
		lintConfig, err = getLintConfigForExternalLintV2(
			fileVersion,
			externalBufPolicyYAMLFile.Lint,
			".",   // Top-level directory always has the root ".".
			false, // Not module-specific configuration.
		)
		if err != nil {
			return nil, err
		}
	}
	var breakingConfig BreakingConfig
	if !externalBufPolicyYAMLFile.Breaking.isEmpty() {
		breakingConfig, err = getBreakingConfigForExternalBreaking(
			fileVersion,
			externalBufPolicyYAMLFile.Breaking,
			".",   // Top-level directory always has the root ".".
			false, // Not module-specific configuration.
		)
		if err != nil {
			return nil, err
		}
	}
	var pluginConfigs []PluginConfig
	for _, externalPluginConfig := range externalBufPolicyYAMLFile.Plugins {
		pluginConfig, err := newPluginConfigForExternalV2(externalPluginConfig)
		if err != nil {
			return nil, err
		}
		pluginConfigs = append(pluginConfigs, pluginConfig)
	}
	return newBufPolicyYAMLFile(
		fileVersion,
		objectData,
		externalBufPolicyYAMLFile.Name,
		lintConfig,
		breakingConfig,
		pluginConfigs,
	)
}

func writeBufPolicyYAMLFile(writer io.Writer, bufPolicyYAMLFile BufPolicyYAMLFile) error {
	fileVersion := bufPolicyYAMLFile.FileVersion()
	if fileVersion != FileVersionV2 {
		// This is effectively a system error.
		return syserror.Wrap(newUnsupportedFileVersionError("", fileVersion))
	}
	var externalLint externalBufYAMLFileLintV2
	if lintConfig := bufPolicyYAMLFile.LintConfig(); lintConfig != nil {
		externalLint = getExternalLintV2ForLintConfig(lintConfig, ".")
	}
	var externalBreaking externalBufYAMLFileBreakingV1Beta1V1V2
	if breakingConfig := bufPolicyYAMLFile.BreakingConfig(); breakingConfig != nil {
		externalBreaking = getExternalBreakingForBreakingConfig(breakingConfig, ".")
	}
	var externalPlugins []externalBufYAMLFilePluginV2
	for _, pluginConfig := range bufPolicyYAMLFile.PluginConfigs() {
		externalPlugin, err := newExternalV2ForPluginConfig(pluginConfig)
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

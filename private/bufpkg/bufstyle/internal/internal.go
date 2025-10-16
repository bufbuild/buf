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

// Package internal defines lint analyzers that help enforce Buf's Go code standards.
package internal

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/analysis"
	"gopkg.in/yaml.v3"
)

const (
	// ExternalConfigPath is the external configuration path.
	ExternalConfigPath = ".bufstyle.yaml"

	v1 = "v1"
)

// AnalyzerProvider provides analyzers.
type AnalyzerProvider interface {
	Analyzers() []*analysis.Analyzer
}

// NewAnalyzerProvider returns a new AnalyzerProvider.
func NewAnalyzerProvider(rootDirPath string, options ...AnalyzerProviderOption) (AnalyzerProvider, error) {
	return newAnalyzerProvider(rootDirPath, options...)
}

// AnalyzerProviderOption is an option for a new AnalyzerProvider.
type AnalyzerProviderOption func(*analyzerProvider)

// WithDisable will disable the given analyzer name.
func WithDisable(analyzerName string) AnalyzerProviderOption {
	return func(analyzerProvider *analyzerProvider) {
		analyzerProvider.disableAnalyzerNames[analyzerName] = struct{}{}
	}
}

// WithIgnore will ignore diagnostics for the given file path and analyzer name.
//
// relFilePath should be relative to rootDirPath.
func WithIgnore(analyzerName string, relFilePath string) AnalyzerProviderOption {
	return func(analyzerProvider *analyzerProvider) {
		relFilePaths, ok := analyzerProvider.ignoreAnalyzerNameToRelFilePaths[analyzerName]
		if !ok {
			relFilePaths = make(map[string]struct{})
			analyzerProvider.ignoreAnalyzerNameToRelFilePaths[analyzerName] = relFilePaths
		}
		relFilePaths[relFilePath] = struct{}{}
	}
}

// ExternalConfig is an external configuration for bufstyle.
type ExternalConfig struct {
	// Version must be "v1".
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
	// Disable is a list of analyzer names to disable.
	Disable []string `json:"disable,omitempty" yaml:"disable,omitempty"`
	// Ignore is a map from analyzer name to a list of relative paths to ignore.
	Ignore map[string][]string `json:"ignore,omitempty" yaml:"ignore,omitempty"`
}

// ReadExternalConfig reads the ExternalConfig.
func ReadExternalConfig(dirPath string) (ExternalConfig, error) {
	filePath := filepath.Join(dirPath, ExternalConfigPath)
	var externalConfig ExternalConfig
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return ExternalConfig{Version: v1}, nil
		}
		return externalConfig, err
	}
	yamlDecoder := yaml.NewDecoder(bytes.NewReader(data))
	yamlDecoder.KnownFields(true)
	if err := yamlDecoder.Decode(&externalConfig); err != nil {
		return externalConfig, fmt.Errorf("could not unmarshal as YAML: %v", err)
	}
	switch externalConfig.Version {
	case v1:
		return externalConfig, nil
	default:
		return externalConfig, fmt.Errorf("%s: unknown version: %s", filePath, externalConfig.Version)
	}
}

// AnalyzerProviderOptionsForExternalConfig returns a new slice of AnalyzerProviderOptions for
// the given ExternalConfig.
func AnalyzerProviderOptionsForExternalConfig(externalConfig ExternalConfig) ([]AnalyzerProviderOption, error) {
	ignoreAnalyzerNameToRelFilePaths, err := getIgnoreAnalyzerNameToRelFilePaths(externalConfig)
	if err != nil {
		return nil, err
	}
	var analyzerProviderOptions []AnalyzerProviderOption
	for _, analyzerName := range externalConfig.Disable {
		analyzerProviderOptions = append(
			analyzerProviderOptions,
			WithDisable(analyzerName),
		)
	}
	for analyzerName, relFilePaths := range ignoreAnalyzerNameToRelFilePaths {
		for relFilePath := range relFilePaths {
			analyzerProviderOptions = append(
				analyzerProviderOptions,
				WithIgnore(analyzerName, relFilePath),
			)
		}
	}
	return analyzerProviderOptions, nil
}

// *** PRIVATE ***

func getIgnoreAnalyzerNameToRelFilePaths(externalConfig ExternalConfig) (map[string]map[string]struct{}, error) {
	ignoreAnalyzerNameToRelFilePaths := make(map[string]map[string]struct{})
	for analyzerName, relFilePaths := range externalConfig.Ignore {
		if len(relFilePaths) == 0 {
			return nil, fmt.Errorf("empty ignore file paths")
		}
		relFilePathMap := make(map[string]struct{})
		for _, relFilePath := range relFilePaths {
			if _, ok := relFilePathMap[relFilePath]; ok {
				return nil, fmt.Errorf("duplicate ignore file path %q for analyzer %q", relFilePath, analyzerName)
			}
			relFilePathMap[relFilePath] = struct{}{}
		}
		if _, ok := ignoreAnalyzerNameToRelFilePaths[analyzerName]; ok {
			return nil, fmt.Errorf("duplicate ignore analyzer name: %q", analyzerName)
		}
		ignoreAnalyzerNameToRelFilePaths[analyzerName] = relFilePathMap
	}
	return ignoreAnalyzerNameToRelFilePaths, nil
}

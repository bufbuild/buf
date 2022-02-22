// Copyright 2020-2022 Buf Technologies, Inc.
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

package main

import (
	"fmt"
	"os"

	"github.com/bufbuild/buf/private/bufpkg/bufstyle"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"golang.org/x/tools/go/analysis/multichecker"
)

var externalConfigPath = ".bufstyle.yaml"

func main() {
	analyzerProvider, err := newAnalyzerProvider()
	if err != nil {
		if errString := err.Error(); errString != "" {
			fmt.Fprintln(os.Stderr, errString)
		}
		os.Exit(1)
	}
	multichecker.Main(analyzerProvider.Analyzers()...)
}

func newAnalyzerProvider() (bufstyle.AnalyzerProvider, error) {
	externalConfig, err := readExternalConfig()
	if err != nil {
		return nil, err
	}
	ignoreAnalyzerNameToRelFilePaths, err := getIgnoreAnalyzerNameToRelFilePaths(externalConfig)
	if err != nil {
		return nil, err
	}
	var analyzerProviderOptions []bufstyle.AnalyzerProviderOption
	for analyzerName, relFilePaths := range ignoreAnalyzerNameToRelFilePaths {
		for relFilePath := range relFilePaths {
			analyzerProviderOptions = append(
				analyzerProviderOptions,
				bufstyle.WithIgnore(analyzerName, relFilePath),
			)
		}
	}
	rootDirPath, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return bufstyle.NewAnalyzerProvider(rootDirPath, analyzerProviderOptions...)
}

func readExternalConfig() (bufstyle.ExternalConfig, error) {
	var externalConfig bufstyle.ExternalConfig
	data, err := os.ReadFile(externalConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return externalConfig, nil
		}
		return externalConfig, err
	}
	if err := encoding.UnmarshalYAMLStrict(data, &externalConfig); err != nil {
		return externalConfig, err
	}
	return externalConfig, nil
}

func getIgnoreAnalyzerNameToRelFilePaths(externalConfig bufstyle.ExternalConfig) (map[string]map[string]struct{}, error) {
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

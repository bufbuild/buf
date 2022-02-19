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
	"sync"

	"github.com/bufbuild/buf/private/bufpkg/bufstyle"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"golang.org/x/tools/go/analysis/multichecker"
)

var externalConfigPath = ".bufstyle.yaml"

var _ sync.Pool

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
	ignoreRelFilePathToAnalyzerNames, err := getIgnoreRelFilePathToAnalyzerNames(externalConfig)
	if err != nil {
		return nil, err
	}
	var analyzerProviderOptions []bufstyle.AnalyzerProviderOption
	for filePath, analyzerNames := range ignoreRelFilePathToAnalyzerNames {
		for analyzerName := range analyzerNames {
			analyzerProviderOptions = append(
				analyzerProviderOptions,
				bufstyle.WithIgnore(filePath, analyzerName),
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

func getIgnoreRelFilePathToAnalyzerNames(externalConfig bufstyle.ExternalConfig) (map[string]map[string]struct{}, error) {
	ignoreRelFilePathToAnalyzerNames := make(map[string]map[string]struct{})
	for _, ignore := range externalConfig.Ignore {
		if ignore.Path == "" {
			return nil, fmt.Errorf("empty ignore.path")
		}
		if len(ignore.Analyzers) == 0 {
			return nil, fmt.Errorf("empty ignore.analyzers")
		}
		analyzerNames := make(map[string]struct{})
		for _, analyzer := range ignore.Analyzers {
			if _, ok := analyzerNames[analyzer]; ok {
				return nil, fmt.Errorf("duplicate ignore.analyzer %q for path: %q", analyzer, ignore.Path)
			}
			analyzerNames[analyzer] = struct{}{}
		}
		if _, ok := ignoreRelFilePathToAnalyzerNames[ignore.Path]; ok {
			return nil, fmt.Errorf("duplicate ignore.path: %q", ignore.Path)
		}
		ignoreRelFilePathToAnalyzerNames[ignore.Path] = analyzerNames
	}
	return ignoreRelFilePathToAnalyzerNames, nil
}

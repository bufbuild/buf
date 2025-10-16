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

package main

import (
	"fmt"
	"os"

	"github.com/bufbuild/buf/private/bufpkg/bufstyle/internal"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	analyzerProvider, err := newAnalyzerProvider(".")
	if err != nil {
		if errString := err.Error(); errString != "" {
			fmt.Fprintln(os.Stderr, errString)
		}
		os.Exit(1)
	}
	multichecker.Main(analyzerProvider.Analyzers()...)
}

func newAnalyzerProvider(dirPath string) (internal.AnalyzerProvider, error) {
	externalConfig, err := internal.ReadExternalConfig(dirPath)
	if err != nil {
		return nil, err
	}
	analyzerProviderOptions, err := internal.AnalyzerProviderOptionsForExternalConfig(externalConfig)
	if err != nil {
		return nil, err
	}
	return internal.NewAnalyzerProvider(dirPath, analyzerProviderOptions...)
}

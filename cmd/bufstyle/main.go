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

import "github.com/bufbuild/buf/private/bufstyle"

// AnalyzerPlugin implements the go/analysis API. This variable
// must be exported so that it's compatible with golangci-lint.
//
// We don't actually need to define the traditional main function
// here. We can compile this program as a plugin with the following
// command:
//
//  $ go build -buildmode=plugin cmd/bufstyle/main.go
//
// For more, see https://golangci-lint.run/contributing/new-linters.
var AnalyzerPlugin bufstyle.AnalyzerPlugin

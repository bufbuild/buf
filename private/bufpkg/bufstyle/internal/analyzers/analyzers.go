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

package analyzers

import (
	"github.com/bufbuild/buf/private/bufpkg/bufstyle/internal/analyzers/america"
	"github.com/bufbuild/buf/private/bufpkg/bufstyle/internal/analyzers/casing"
	"github.com/bufbuild/buf/private/bufpkg/bufstyle/internal/analyzers/packagefilename"
	"github.com/bufbuild/buf/private/bufpkg/bufstyle/internal/analyzers/typeban"
	"golang.org/x/tools/go/analysis"
)

// New returns all Analyzers.
//
// We don't store this as a global because we modify these.
func New() []*analysis.Analyzer {
	return append(
		america.New(),
		casing.New(),
		packagefilename.New(),
		typeban.New(),
	)
}

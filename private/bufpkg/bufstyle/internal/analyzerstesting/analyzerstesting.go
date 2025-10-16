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

package analyzerstesting

import (
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/pkg/osext"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

// Run runs the tests.
//
// It expects tests to be in "testdata/src/p" relative to the Go package the tests are being run in.
func Run(t *testing.T, analyzers []*analysis.Analyzer) {
	t.Parallel()
	pwd, err := osext.Getwd()
	require.NoError(t, err)
	all := &analysis.Analyzer{
		Name:     "all",
		Requires: analyzers,
	}
	analysistest.Run(t, filepath.Join(pwd, "testdata"), all, "p")
}

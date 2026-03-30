// Copyright 2020-2026 Buf Technologies, Inc.
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

package depgraph

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appcmd/appcmdtesting"
	"buf.build/go/app/appext"
	"github.com/bufbuild/buf/cmd/buf/internal/internaltesting"
	"github.com/stretchr/testify/require"
)

// TestJSONFormatSharedDep is a regression test for a bug where addDeps had
// a "return nil" instead of "continue" when a dependency was already in the
// seen map, causing all subsequent dependencies to be silently dropped.
//
// The workspace has three local modules: alpha -> common, zeta -> {alpha, common}.
// Because modules are walked alphabetically, both of zeta's deps are already
// in the seen map when zeta is processed, which triggered the early return.
func TestJSONFormatSharedDep(t *testing.T) {
	t.Parallel()
	var stdout bytes.Buffer
	appcmdtesting.Run(
		t,
		func(name string) *appcmd.Command {
			return NewCommand(name, appext.NewBuilder(name))
		},
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithEnv(internaltesting.NewEnvFunc(t)),
		appcmdtesting.WithStdout(&stdout),
		appcmdtesting.WithArgs(
			"--format", "json",
			filepath.Join("testdata", "shared_dep_workspace"),
		),
	)
	var got []externalModule
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &got))
	clearDigests(got)
	expected := []externalModule{
		{
			Name:  "alpha",
			Local: true,
			Deps: []externalModule{
				{Name: "common", Local: true},
			},
		},
		{
			Name:  "common",
			Local: true,
		},
		{
			Name:  "zeta",
			Local: true,
			Deps: []externalModule{
				{Name: "alpha", Local: true, Deps: []externalModule{
					{Name: "common", Local: true},
				}},
				{Name: "common", Local: true},
			},
		},
	}
	require.Equal(t, expected, got)
}

// clearDigests zeros out Digest fields so we can compare structures without
// depending on content hashes.
func clearDigests(modules []externalModule) {
	for i := range modules {
		modules[i].Digest = ""
		clearDigests(modules[i].Deps)
	}
}
